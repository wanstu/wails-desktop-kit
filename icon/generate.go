// Package icon provides cross-platform application icon normalization and generation.
package icon

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// BadgeOptions describes a deterministic product-family application icon.
// It deliberately avoids system fonts so output is identical on every platform.
type BadgeOptions struct {
	Size       int
	Inset      int
	Radius     int
	Background color.NRGBA
	Foreground color.NRGBA
	Symbol     string
	Text       string
}

// DefaultBadgeOptions returns the shared Kit family-icon defaults.
func DefaultBadgeOptions() BadgeOptions {
	return BadgeOptions{
		Size:       256,
		Inset:      16,
		Radius:     56,
		Background: color.NRGBA{R: 0x24, G: 0x63, B: 0xeb, A: 0xff},
		Foreground: color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		Symbol:     "terminal",
	}
}

// GenerateBadge returns a deterministic transparent PNG-ready image.
// Supported symbols are "terminal" and "monogram".
func GenerateBadge(opts BadgeOptions) (*image.NRGBA, error) {
	opts, err := normalizeBadgeOptions(opts)
	if err != nil {
		return nil, err
	}

	dst := image.NewNRGBA(image.Rect(0, 0, opts.Size, opts.Size))
	drawRoundedSquare(dst, opts.Inset, opts.Radius, opts.Background)

	switch opts.Symbol {
	case "terminal":
		drawTerminalSymbol(dst, opts.Foreground)
	case "monogram":
		if err := drawMonogram(dst, opts.Text, opts.Foreground); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("icon: unsupported badge symbol %q", opts.Symbol)
	}
	return dst, nil
}

// GenerateBadgeFile writes a generated badge icon as PNG. The output is
// replaced atomically so a failed generation never corrupts an existing icon.
func GenerateBadgeFile(outputPath string, opts BadgeOptions) error {
	img, err := GenerateBadge(opts)
	if err != nil {
		return err
	}
	return writePNGAtomic(outputPath, img)
}

// ParseHexColor parses #RRGGBB or #RRGGBBAA.
func ParseHexColor(value string) (color.NRGBA, error) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "#") {
		return color.NRGBA{}, fmt.Errorf("icon: color must use #RRGGBB or #RRGGBBAA")
	}
	raw := value[1:]
	if len(raw) != 6 && len(raw) != 8 {
		return color.NRGBA{}, fmt.Errorf("icon: color must use #RRGGBB or #RRGGBBAA")
	}
	var rgba uint64
	if _, err := fmt.Sscanf(raw, "%x", &rgba); err != nil {
		return color.NRGBA{}, fmt.Errorf("icon: parse color %q: %w", value, err)
	}
	if len(raw) == 6 {
		return color.NRGBA{
			R: uint8(rgba >> 16),
			G: uint8(rgba >> 8),
			B: uint8(rgba),
			A: 0xff,
		}, nil
	}
	return color.NRGBA{
		R: uint8(rgba >> 24),
		G: uint8(rgba >> 16),
		B: uint8(rgba >> 8),
		A: uint8(rgba),
	}, nil
}

func normalizeBadgeOptions(opts BadgeOptions) (BadgeOptions, error) {
	defaults := DefaultBadgeOptions()
	if opts == (BadgeOptions{}) {
		opts = defaults
	}
	if opts.Size == 0 {
		opts.Size = defaults.Size
	}
	if opts.Size < 16 || opts.Size > 1024 {
		return BadgeOptions{}, fmt.Errorf("icon: badge size must be between 16 and 1024")
	}
	if opts.Inset == 0 {
		opts.Inset = maxInt(1, int(float64(opts.Size)*float64(defaults.Inset)/float64(defaults.Size)+0.5))
	}
	if opts.Radius == 0 {
		opts.Radius = maxInt(2, int(float64(opts.Size)*float64(defaults.Radius)/float64(defaults.Size)+0.5))
	}
	if opts.Inset < 0 || opts.Inset*2 >= opts.Size {
		return BadgeOptions{}, fmt.Errorf("icon: badge inset is invalid for size %d", opts.Size)
	}
	maxRadius := (opts.Size - opts.Inset*2) / 2
	if opts.Radius < 0 || opts.Radius > maxRadius {
		return BadgeOptions{}, fmt.Errorf("icon: badge radius must be between 0 and %d", maxRadius)
	}
	opts.Symbol = strings.ToLower(strings.TrimSpace(opts.Symbol))
	if opts.Symbol == "" {
		opts.Symbol = defaults.Symbol
	}
	if opts.Symbol == "monogram" {
		text := strings.TrimSpace(opts.Text)
		runes := []rune(text)
		if len(runes) != 1 {
			return BadgeOptions{}, fmt.Errorf("icon: monogram text must contain exactly one ASCII letter or digit")
		}
		r := unicode.ToUpper(runes[0])
		if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return BadgeOptions{}, fmt.Errorf("icon: monogram text must contain exactly one ASCII letter or digit")
		}
		opts.Text = string(r)
	}
	return opts, nil
}

func drawRoundedSquare(dst *image.NRGBA, inset, radius int, fill color.NRGBA) {
	bounds := dst.Bounds()
	left := inset
	top := inset
	right := bounds.Dx() - inset
	bottom := bounds.Dy() - inset
	r2 := radius * radius

	for y := top; y < bottom; y++ {
		for x := left; x < right; x++ {
			inside := true
			if radius > 0 {
				cx := x
				cy := y
				switch {
				case x < left+radius && y < top+radius:
					cx, cy = left+radius-1, top+radius-1
				case x >= right-radius && y < top+radius:
					cx, cy = right-radius, top+radius-1
				case x < left+radius && y >= bottom-radius:
					cx, cy = left+radius-1, bottom-radius
				case x >= right-radius && y >= bottom-radius:
					cx, cy = right-radius, bottom-radius
				default:
					dst.SetNRGBA(x, y, fill)
					continue
				}
				dx, dy := x-cx, y-cy
				inside = dx*dx+dy*dy <= r2
			}
			if inside {
				dst.SetNRGBA(x, y, fill)
			}
		}
	}
}

func drawTerminalSymbol(dst *image.NRGBA, fg color.NRGBA) {
	size := dst.Bounds().Dx()
	scale := float64(size) / 32.0
	stroke := maxInt(1, int(3*scale+0.5))
	x0 := int(9*scale + 0.5)
	yTop := int(9*scale + 0.5)
	yMid := int(16*scale + 0.5)
	yBottom := int(23*scale + 0.5)
	span := int(7*scale + 0.5)

	for step := 0; step <= span; step++ {
		x := x0 + step
		y1 := yTop + step
		y2 := yBottom - step
		fillRect(dst, x, y1, stroke, stroke, fg)
		fillRect(dst, x, y2, stroke, stroke, fg)
	}
	underscoreX := int(18*scale + 0.5)
	underscoreY := int(21*scale + 0.5)
	underscoreW := maxInt(stroke, int(7*scale+0.5))
	fillRect(dst, underscoreX, underscoreY, underscoreW, stroke, fg)

	_ = yMid // kept as an explicit optical center for future symbols.
}

func drawMonogram(dst *image.NRGBA, text string, fg color.NRGBA) error {
	r := []rune(strings.ToUpper(strings.TrimSpace(text)))[0]
	rows, ok := pixelFont5x7[r]
	if !ok {
		return fmt.Errorf("icon: monogram %q is not supported", text)
	}

	size := dst.Bounds().Dx()
	cell := maxInt(1, int(float64(size)/10.5))
	width := 5 * cell
	height := 7 * cell
	startX := (size - width) / 2
	startY := (size-height)/2 + maxInt(0, cell/3)

	for row, pattern := range rows {
		for col, bit := range pattern {
			if bit == '1' {
				fillRect(dst, startX+col*cell, startY+row*cell, cell, cell, fg)
			}
		}
	}
	return nil
}

func fillRect(dst *image.NRGBA, x, y, width, height int, fill color.NRGBA) {
	for py := y; py < y+height; py++ {
		for px := x; px < x+width; px++ {
			if image.Pt(px, py).In(dst.Bounds()) {
				dst.SetNRGBA(px, py, fill)
			}
		}
	}
}

func writePNGAtomic(outputPath string, img image.Image) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("icon: create output directory: %w", err)
	}
	f, err := os.CreateTemp(dir, ".desktopkit-icon-*.png")
	if err != nil {
		return fmt.Errorf("icon: create output: %w", err)
	}
	name := f.Name()
	defer os.Remove(name)
	if err := f.Chmod(0o644); err != nil {
		_ = f.Close()
		return err
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return fmt.Errorf("icon: encode png: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("icon: flush output: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("icon: close output: %w", err)
	}
	if err := os.Rename(name, outputPath); err != nil {
		return fmt.Errorf("icon: replace output: %w", err)
	}
	return nil
}

var pixelFont5x7 = map[rune][7]string{
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'B': {"11110", "10001", "10001", "11110", "10001", "10001", "11110"},
	'C': {"01111", "10000", "10000", "10000", "10000", "10000", "01111"},
	'D': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'G': {"01111", "10000", "10000", "10111", "10001", "10001", "01111"},
	'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'I': {"11111", "00100", "00100", "00100", "00100", "00100", "11111"},
	'J': {"00111", "00010", "00010", "00010", "10010", "10010", "01100"},
	'K': {"10001", "10010", "10100", "11000", "10100", "10010", "10001"},
	'L': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'N': {"10001", "11001", "10101", "10011", "10001", "10001", "10001"},
	'O': {"01110", "10001", "10001", "10001", "10001", "10001", "01110"},
	'P': {"11110", "10001", "10001", "11110", "10000", "10000", "10000"},
	'Q': {"01110", "10001", "10001", "10001", "10101", "10010", "01101"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'S': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
	'V': {"10001", "10001", "10001", "10001", "10001", "01010", "00100"},
	'W': {"10001", "10001", "10001", "10101", "10101", "10101", "01010"},
	'X': {"10001", "10001", "01010", "00100", "01010", "10001", "10001"},
	'Y': {"10001", "10001", "01010", "00100", "00100", "00100", "00100"},
	'Z': {"11111", "00001", "00010", "00100", "01000", "10000", "11111"},
	'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "10000", "11110", "00001", "00001", "11110"},
	'6': {"01110", "10000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00001", "01110"},
}
