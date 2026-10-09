#!/usr/bin/env bash
# Acceptance test: install an actual Kit-generated Debian package on the
# disposable GitHub Actions Ubuntu VM (systemd is PID 1 there), exercise
# service lifecycle through dpkg, and verify user data survives remove/purge.
set -euo pipefail

if [[ ! -d /run/systemd/system ]]; then
  echo 'ERROR: systemd is not running; this is a required end-to-end test, not an optional smoke check.' >&2
  exit 1
fi

package=desktopkit-systemd-e2e
data_dir="/var/lib/${package}"
root="$(mktemp -d)"
cleanup() {
  # Cleanup after assertions: this ephemeral CI test owns this package/data.
  sudo dpkg --purge "$package" >/dev/null 2>&1 || true
  sudo systemctl reset-failed "$package.service" >/dev/null 2>&1 || true
  sudo rm -rf -- "$data_dir"
  rm -rf -- "$root"
}
trap cleanup EXIT

build_deb() {
  local version="$1"
  local out="$root/$version"
  mkdir -p "$out"
  go run ./cmd/desktopkit package linux \
    --input /bin/sleep \
    --dist "$out" \
    --app-name "$package" \
    --package-name "$package" \
    --asset-base "$package" \
    --package-version "$version" \
    --description "Kit real systemd lifecycle test" \
    --maintainer "Kit CI" \
    --formats deb \
    --systemd \
    --service-arg 300 \
    --service-env "TEST_PORT=8610" \
    --service-environment-file "/etc/default/$package"
  test -f "$out/$package-linux-amd64.deb"
}

# Two distinct versions are needed to exercise Debian upgrade maintainer
# scripts, rather than accidentally repeating a fresh install.
build_deb 0.0.1
build_deb 0.0.2
build_deb 0.0.3
deb1="$root/0.0.1/$package-linux-amd64.deb"
deb2="$root/0.0.2/$package-linux-amd64.deb"
deb3="$root/0.0.3/$package-linux-amd64.deb"

echo '=== Install and start systemd service ==='
sudo dpkg --install "$deb1"
systemctl is-enabled --quiet "$package.service"
systemctl is-active --quiet "$package.service"
test "$(dpkg-query -W -f='${Version}' "$package")" = 0.0.1
first_pid="$(systemctl show -p MainPID --value "$package.service")"
test "$first_pid" -gt 0

test -d "$data_dir"
test "$(stat -c '%a' "$data_dir")" = 700
test "$(stat -c '%U:%G' "$data_dir")" = "$package:$package"
printf 'persist across upgrade, remove and purge\n' | sudo tee "$data_dir/preserved.txt" >/dev/null

echo '=== Upgrade and verify service restart and data ==='
sudo dpkg --install "$deb2"
test "$(dpkg-query -W -f='${Version}' "$package")" = 0.0.2
systemctl is-enabled --quiet "$package.service"
systemctl is-active --quiet "$package.service"
second_pid="$(systemctl show -p MainPID --value "$package.service")"
test "$second_pid" -gt 0
test "$second_pid" != "$first_pid"
sudo grep -Fx 'persist across upgrade, remove and purge' "$data_dir/preserved.txt" >/dev/null

echo '=== Respect explicit service disable on an in-place upgrade ==='
sudo systemctl disable --now "$package.service"
sudo dpkg --install "$deb3"
test "$(dpkg-query -W -f='${Version}' "$package")" = 0.0.3
if systemctl is-active --quiet "$package.service" || systemctl is-enabled --quiet "$package.service"; then
  echo 'ERROR: upgrading a manually disabled service must not re-enable it' >&2
  exit 1
fi
sudo grep -Fx 'persist across upgrade, remove and purge' "$data_dir/preserved.txt" >/dev/null
sudo systemctl enable --now "$package.service"
systemctl is-active --quiet "$package.service"

echo '=== Remove and verify service stopped, data retained ==='
sudo dpkg --remove "$package"
if systemctl is-active --quiet "$package.service"; then
  echo 'ERROR: service remained active after package removal' >&2
  exit 1
fi
if systemctl is-enabled --quiet "$package.service"; then
  echo 'ERROR: service remained enabled after package removal' >&2
  exit 1
fi
test ! -f "/lib/systemd/system/$package.service"
sudo grep -Fx 'persist across upgrade, remove and purge' "$data_dir/preserved.txt" >/dev/null

echo '=== Reinstall, purge and verify data retained ==='
sudo dpkg --install "$deb2"
systemctl is-active --quiet "$package.service"
sudo grep -Fx 'persist across upgrade, remove and purge' "$data_dir/preserved.txt" >/dev/null
sudo dpkg --purge "$package"
if systemctl is-active --quiet "$package.service"; then
  echo 'ERROR: service remained active after purge' >&2
  exit 1
fi
test ! -f "/lib/systemd/system/$package.service"
sudo grep -Fx 'persist across upgrade, remove and purge' "$data_dir/preserved.txt" >/dev/null
echo 'PASS: true systemd install, upgrade, restart, disable, remove, purge and data retention'
