package icon

import (
	"image"
	"image/color"
	"testing"
)

func TestNormalizeTrimsAndCentersVisibleContent(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for y := 3; y < 7; y++ {
		for x := 2; x < 8; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: 30, G: 140, B: 220, A: 255})
		}
	}

	out, err := Normalize(src, Options{CanvasSize: 100, Fill: 0.8, TrimAlpha: true})
	if err != nil {
		t.Fatal(err)
	}
	if out.Bounds().Dx() != 100 || out.Bounds().Dy() != 100 {
		t.Fatalf("bounds = %v", out.Bounds())
	}

	bounds, err := visibleBounds(out, 8)
	if err != nil {
		t.Fatal(err)
	}
	if bounds.Dx() != 80 {
		t.Fatalf("visible width = %d, want 80", bounds.Dx())
	}
	if bounds.Min.X != 10 || bounds.Max.X != 90 {
		t.Fatalf("visible horizontal bounds = %v, want x=[10,90)", bounds)
	}
}

func TestNormalizeRejectsInvalidFill(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	if _, err := Normalize(src, Options{CanvasSize: 32, Fill: 1.1}); err == nil {
		t.Fatal("expected invalid fill error")
	}
}

func TestVisibleBoundsRejectsTransparentImage(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	if _, err := visibleBounds(src, 8); err == nil {
		t.Fatal("expected transparent image error")
	}
}
