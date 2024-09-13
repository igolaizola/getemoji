package getemoji

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kyokomi/emoji/v2"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"golang.org/x/text/unicode/norm"
)

type Config struct {
	Size   int
	Emoji  string
	Output string
}

var unicodeReg = regexp.MustCompile(`^[0-9a-fA-F]+(-[0-9a-fA-F])*$`)

// Run runs the getemoji process.
func Run(ctx context.Context, cfg *Config) error {
	// Check output file
	output := cfg.Output
	if output == "" {
		output = "icon.svg"
		if cfg.Size > 0 {
			output = fmt.Sprintf("icon%d.png", cfg.Size)
		}
	}

	// Check extension
	ext := strings.ToLower(filepath.Ext(output))
	switch ext {
	case ".svg":
	case ".png":
		if cfg.Size <= 0 {
			return errors.New("size must be greater than 0")
		}
	default:
		return fmt.Errorf("unsupported output format: %s", ext)
	}

	// Check emoji
	if cfg.Emoji == "" {
		return errors.New("emoji must not be empty")
	}

	// Convert emoji shortcode to emoji if needed
	emj := toEmoji(cfg.Emoji)

	// Convert emoji to unicode code points
	normalized := norm.NFC.String(emj)
	var codePoints []string
	for _, r := range normalized {
		codePoints = append(codePoints, fmt.Sprintf("%x", r))
	}
	unicodeHex := strings.Join(codePoints, "-")

	// Validate Unicode code points
	if !unicodeReg.MatchString(unicodeHex) {
		return fmt.Errorf("invalid unicode code points: %q for emoji %q", unicodeHex, emj)
	}

	// Construct URL
	u := fmt.Sprintf("https://cdn.jsdelivr.net/gh/jdecked/twemoji@15.0.2/assets/svg/%s.svg", unicodeHex)

	// Download SVG
	resp, err := http.Get(u)
	if err != nil {
		return fmt.Errorf("couldn't download SVG from %q: %v", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("couldn't download SVG from %q: %s", u, resp.Status)
	}

	// Read SVG content
	svgContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("couldn't read SVG content: %v", err)
	}

	// Process based on output file extension
	switch ext {
	case ".svg":
		// For SVG, we don't actually resize, we just save the original
		return os.WriteFile(output, svgContent, 0644)
	case ".png":
		size := cfg.Size
		// Convert SVG to PNG
		icon, err := oksvg.ReadIconStream(strings.NewReader(string(svgContent)))
		if err != nil {
			return fmt.Errorf("couldn't read SVG: %v", err)
		}

		icon.SetTarget(0, 0, float64(size), float64(size))
		rgba := image.NewRGBA(image.Rect(0, 0, size, size))
		icon.Draw(rasterx.NewDasher(size, size, rasterx.NewScannerGV(size, size, rgba, rgba.Bounds())), 1)

		outputFile, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("couldn't create output file: %v", err)
		}
		defer outputFile.Close()

		if err := png.Encode(outputFile, rgba); err != nil {
			return fmt.Errorf("couldn't encode PNG: %v", err)
		}

		return nil
	default:
		return fmt.Errorf("unsupported output format: %s", ext)
	}
}

func toEmoji(input string) string {
	emj := strings.TrimSpace(input)
	emjNorm := normalize(emj)
	for k, v := range emoji.CodeMap() {
		if normalize(k) == emjNorm {
			return v
		}
	}
	emj = strings.Trim(emj, ":")
	emj = emoji.Emojize(fmt.Sprintf(":%s:", emj))
	emj = strings.TrimSpace(emj)
	emj = strings.Trim(emj, ":")
	return emj
}

func normalize(input string) string {
	t := strings.ToLower(input)
	t = strings.Trim(t, ":")
	t = strings.ReplaceAll(t, "_", "")
	t = strings.ReplaceAll(t, "-", "")
	return t
}
