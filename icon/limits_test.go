package icon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeRejectsUnsafeDimensionsAndNonfiniteFill(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	for _, opts := range []Options{
		{CanvasSize: MaxCanvasSize + 1, Fill: 1},
		{CanvasSize: 16, Fill: math.NaN()},
		{CanvasSize: 16, Fill: math.Inf(1)},
	} {
		if _, err := Normalize(src, opts); err == nil {
			t.Fatalf("accepted %+v", opts)
		}
	}
	if err := validateDimensions(1<<30, 1<<30); err == nil {
		t.Fatal("accepted huge input")
	}
}
func TestBilinearPreservesColorAtTransparentEdge(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	src.SetNRGBA(1, 0, color.NRGBA{B: 255, A: 0})
	out, err := Normalize(src, Options{CanvasSize: 3, Fill: 1})
	if err != nil {
		t.Fatal(err)
	}
	p := out.NRGBAAt(1, 1)
	if p.R != 255 || p.B != 0 || p.A < 120 || p.A > 135 {
		t.Fatalf("incorrect alpha interpolation: %+v", p)
	}
}
func TestNormalizeFilePreservesExistingOutputOnFailure(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "in.png")
	output := filepath.Join(dir, "out.png")
	if err := os.WriteFile(input, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	old := []byte("original")
	if err := os.WriteFile(output, old, 0600); err != nil {
		t.Fatal(err)
	}
	if err := NormalizeFile(input, output, DefaultOptions()); err == nil {
		t.Fatal("invalid input accepted")
	}
	got, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(got, old) {
		t.Fatal("overwrote output on failure")
	}
}
func TestNormalizeFileSupportsReplacingSource(t *testing.T) {
	file := filepath.Join(t.TempDir(), "icon.png")
	var data bytes.Buffer
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	if err := png.Encode(&data, src); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	if err := NormalizeFile(file, file, Options{CanvasSize: 32, Fill: 1}); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil || cfg.Width != 32 {
		t.Fatalf("bad replacement: %+v %v", cfg, err)
	}
}
