package updater

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type semanticVersion struct {
	major      uint64
	minor      uint64
	patch      uint64
	prerelease []string
}

func CompareVersions(left, right string) (int, error) {
	a, err := parseVersion(left)
	if err != nil {
		return 0, fmt.Errorf("parse left version %q: %w", left, err)
	}
	b, err := parseVersion(right)
	if err != nil {
		return 0, fmt.Errorf("parse right version %q: %w", right, err)
	}
	if a.major != b.major {
		return compareUint(a.major, b.major), nil
	}
	if a.minor != b.minor {
		return compareUint(a.minor, b.minor), nil
	}
	if a.patch != b.patch {
		return compareUint(a.patch, b.patch), nil
	}
	return comparePrerelease(a.prerelease, b.prerelease), nil
}

func parseVersion(raw string) (semanticVersion, error) {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "v")
	value = strings.TrimPrefix(value, "V")
	if value == "" {
		return semanticVersion{}, fmt.Errorf("empty version")
	}
	if index := strings.IndexByte(value, '+'); index >= 0 {
		value = value[:index]
	}

	core := value
	var prerelease []string
	if index := strings.IndexByte(value, '-'); index >= 0 {
		core = value[:index]
		pre := value[index+1:]
		if pre == "" {
			return semanticVersion{}, fmt.Errorf("empty prerelease")
		}
		prerelease = strings.Split(pre, ".")
		for _, identifier := range prerelease {
			if !validPrereleaseIdentifier(identifier) {
				return semanticVersion{}, fmt.Errorf("invalid prerelease identifier %q", identifier)
			}
		}
	}

	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return semanticVersion{}, fmt.Errorf("expected major.minor.patch")
	}
	numbers := make([]uint64, 3)
	for i, part := range parts {
		if part == "" {
			return semanticVersion{}, fmt.Errorf("empty numeric component")
		}
		number, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return semanticVersion{}, fmt.Errorf("invalid numeric component %q", part)
		}
		numbers[i] = number
	}
	return semanticVersion{
		major:      numbers[0],
		minor:      numbers[1],
		patch:      numbers[2],
		prerelease: prerelease,
	}, nil
}

func validPrereleaseIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-') {
			return false
		}
	}
	return true
}

func compareUint(left, right uint64) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

func comparePrerelease(left, right []string) int {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}
	if len(left) == 0 {
		return 1
	}
	if len(right) == 0 {
		return -1
	}

	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		lNumeric, lValue := numericIdentifier(left[i])
		rNumeric, rValue := numericIdentifier(right[i])
		switch {
		case lNumeric && rNumeric:
			if lValue != rValue {
				return compareUint(lValue, rValue)
			}
		case lNumeric && !rNumeric:
			return -1
		case !lNumeric && rNumeric:
			return 1
		default:
			if left[i] < right[i] {
				return -1
			}
			if left[i] > right[i] {
				return 1
			}
		}
	}
	switch {
	case len(left) < len(right):
		return -1
	case len(left) > len(right):
		return 1
	default:
		return 0
	}
}

func numericIdentifier(value string) (bool, uint64) {
	if value == "" {
		return false, 0
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false, 0
		}
	}
	number, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return false, 0
	}
	return true, number
}
