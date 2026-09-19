package icon

import (
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateTerminalBadgeDefaults(t *testing.T) {
	img, err := GenerateBadge(DefaultBadgeOptions())
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 256 || img.Bounds().Dy() != 256 {
		t.Fatalf("unexpected bounds: %v", img.Bounds())
	}
	if got := img.NRGBAAt(0, 0); got.A != 0 {
		t.Fatalf("outer canvas should remain transparent: %#v", got)
	}
	if got := img.NRGBAAt(128, 128); got != (color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}) {
		t.Fatalf("terminal chevron center = %#v", got)
	}
	if got := img.NRGBAAt(200, 200); got != (color.NRGBA{R: 0x24, G: 0x63, B: 0xeb, A: 0xff}) {
		t.Fatalf("badge background = %#v", got)
	}
}

func TestGenerateBadgeExplicitSmallSizeStillWorks(t *testing.T) {
	opts := DefaultBadgeOptions()
	opts.Size = 32
	opts.Inset = 2
	opts.Radius = 7
	img, err := GenerateBadge(opts)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 32 || img.Bounds().Dy() != 32 {
		t.Fatalf("explicit small icon bounds = %v, want 32x32", img.Bounds())
	}
}

func TestGenerateMonogramBadge(t *testing.T) {
	opts := DefaultBadgeOptions()
	opts.Symbol = "monogram"
	opts.Text = "f"

	img, err := GenerateBadge(opts)
	if err != nil {
		t.Fatal(err)
	}
	if got := img.NRGBAAt(88, 64); got.A == 0 {
		t.Fatalf("expected visible monogram pixel, got %#v", got)
	}
}

func TestGenerateBadgeFileRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "appicon.png")
	if err := GenerateBadgeFile(path, DefaultBadgeOptions()); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 256 || img.Bounds().Dy() != 256 {
		t.Fatalf("unexpected decoded bounds: %v", img.Bounds())
	}
}

func TestGenerateBadgeHonorsTransparentBackground(t *testing.T) {
	opts := DefaultBadgeOptions()
	opts.Background = color.NRGBA{}
	img, err := GenerateBadge(opts)
	if err != nil {
		t.Fatal(err)
	}
	if got := img.NRGBAAt(128, 128); got.A != 0 && got != opts.Foreground {
		t.Fatalf("unexpected transparent-background pixel: %#v", got)
	}
	if got := img.NRGBAAt(200, 200); got.A != 0 {
		t.Fatalf("explicit transparent background was replaced by a default: %#v", got)
	}
}

func TestParseHexColor(t *testing.T) {
	got, err := ParseHexColor("#2463EB")
	if err != nil {
		t.Fatal(err)
	}
	want := color.NRGBA{R: 0x24, G: 0x63, B: 0xeb, A: 0xff}
	if got != want {
		t.Fatalf("got %#v want %#v", got, want)
	}

	got, err = ParseHexColor("#11223344")
	if err != nil {
		t.Fatal(err)
	}
	if got != (color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 0x44}) {
		t.Fatalf("unexpected rgba: %#v", got)
	}
}

func TestGenerateBadgeRejectsInvalidMonogram(t *testing.T) {
	opts := DefaultBadgeOptions()
	opts.Symbol = "monogram"
	opts.Text = "SSH"
	if _, err := GenerateBadge(opts); err == nil {
		t.Fatal("expected invalid monogram error")
	}
}
