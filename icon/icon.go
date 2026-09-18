// Package icon provides cross-platform application icon normalization.
package icon

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// Options controls icon normalization.
type Options struct {
	CanvasSize     int
	Fill           float64
	TrimAlpha      bool
	AlphaThreshold uint8
}

// DefaultOptions returns the shared desktop-kit icon defaults.
func DefaultOptions() Options {
	return Options{
		CanvasSize:     1024,
		Fill:           0.94,
		TrimAlpha:      true,
		AlphaThreshold: 8,
	}
}

// NormalizeFile reads a source PNG/JPEG/GIF image, normalizes visible artwork
// into a square transparent PNG canvas, and writes outputPath.
func NormalizeFile(inputPath, outputPath string, opts Options) error {
	in, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("icon: open input: %w", err)
	}
	defer in.Close()

	src, _, err := image.Decode(in)
	if err != nil {
		return fmt.Errorf("icon: decode input: %w", err)
	}

	out, err := Normalize(src, opts)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("icon: create output directory: %w", err)
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("icon: create output: %w", err)
	}
	defer f.Close()
	if err := png.Encode(f, out); err != nil {
		return fmt.Errorf("icon: encode png: %w", err)
	}
	return nil
}

// Normalize returns a square transparent icon image.
func Normalize(src image.Image, opts Options) (*image.NRGBA, error) {
	opts, err := normalizeOptions(opts)
	if err != nil {
		return nil, err
	}
	if src == nil {
		return nil, fmt.Errorf("icon: source image is nil")
	}

	sourceBounds := src.Bounds()
	if sourceBounds.Empty() {
		return nil, fmt.Errorf("icon: source image is empty")
	}
	content := sourceBounds
	if opts.TrimAlpha {
		content, err = visibleBounds(src, opts.AlphaThreshold)
		if err != nil {
			return nil, err
		}
	}

	targetExtent := float64(opts.CanvasSize) * opts.Fill
	scale := math.Min(targetExtent/float64(content.Dx()), targetExtent/float64(content.Dy()))
	drawWidth := maxInt(1, int(math.Round(float64(content.Dx())*scale)))
	drawHeight := maxInt(1, int(math.Round(float64(content.Dy())*scale)))
	drawX := (opts.CanvasSize - drawWidth) / 2
	drawY := (opts.CanvasSize - drawHeight) / 2

	dst := image.NewNRGBA(image.Rect(0, 0, opts.CanvasSize, opts.CanvasSize))
	for y := 0; y < drawHeight; y++ {
		sy := float64(content.Min.Y) + (float64(y)+0.5)*float64(content.Dy())/float64(drawHeight) - 0.5
		for x := 0; x < drawWidth; x++ {
			sx := float64(content.Min.X) + (float64(x)+0.5)*float64(content.Dx())/float64(drawWidth) - 0.5
			dst.SetNRGBA(drawX+x, drawY+y, sampleBilinear(src, content, sx, sy))
		}
	}
	return dst, nil
}

func normalizeOptions(opts Options) (Options, error) {
	defaults := DefaultOptions()
	if opts.CanvasSize == 0 {
		opts.CanvasSize = defaults.CanvasSize
	}
	if opts.Fill == 0 {
		opts.Fill = defaults.Fill
	}
	if opts.CanvasSize <= 0 {
		return Options{}, fmt.Errorf("icon: canvas size must be positive")
	}
	if opts.Fill <= 0 || opts.Fill > 1 {
		return Options{}, fmt.Errorf("icon: fill must be greater than 0 and at most 1")
	}
	return opts, nil
}

func visibleBounds(src image.Image, threshold uint8) (image.Rectangle, error) {
	bounds := src.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X-1, bounds.Min.Y-1

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			if uint8(a>>8) <= threshold {
				continue
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if maxX < minX || maxY < minY {
		return image.Rectangle{}, fmt.Errorf("icon: source image has no visible pixels above alpha threshold")
	}
	return image.Rect(minX, minY, maxX+1, maxY+1), nil
}

func sampleBilinear(src image.Image, bounds image.Rectangle, x, y float64) color.NRGBA {
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1 := x0 + 1
	y1 := y0 + 1
	tx := x - float64(x0)
	ty := y - float64(y0)

	x0 = clampInt(x0, bounds.Min.X, bounds.Max.X-1)
	x1 = clampInt(x1, bounds.Min.X, bounds.Max.X-1)
	y0 = clampInt(y0, bounds.Min.Y, bounds.Max.Y-1)
	y1 = clampInt(y1, bounds.Min.Y, bounds.Max.Y-1)

	c00 := premultiplied(src.At(x0, y0))
	c10 := premultiplied(src.At(x1, y0))
	c01 := premultiplied(src.At(x0, y1))
	c11 := premultiplied(src.At(x1, y1))

	top := lerpRGBA(c00, c10, tx)
	bottom := lerpRGBA(c01, c11, tx)
	return unpremultiply(lerpRGBA(top, bottom, ty))
}

type rgbaFloat struct {
	r float64
	g float64
	b float64
	a float64
}

func premultiplied(c color.Color) rgbaFloat {
	r, g, b, a := c.RGBA()
	const denom = 65535.0
	return rgbaFloat{
		r: float64(r) / denom,
		g: float64(g) / denom,
		b: float64(b) / denom,
		a: float64(a) / denom,
	}
}

func lerpRGBA(a, b rgbaFloat, t float64) rgbaFloat {
	return rgbaFloat{
		r: a.r + (b.r-a.r)*t,
		g: a.g + (b.g-a.g)*t,
		b: a.b + (b.b-a.b)*t,
		a: a.a + (b.a-a.a)*t,
	}
}

func unpremultiply(c rgbaFloat) color.NRGBA {
	alpha := clampFloat(c.a, 0, 1)
	if alpha <= 0 {
		return color.NRGBA{}
	}
	return color.NRGBA{
		R: uint8(math.Round(clampFloat(c.r/alpha, 0, 1) * 255)),
		G: uint8(math.Round(clampFloat(c.g/alpha, 0, 1) * 255)),
		B: uint8(math.Round(clampFloat(c.b/alpha, 0, 1) * 255)),
		A: uint8(math.Round(alpha * 255)),
	}
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clampFloat(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
