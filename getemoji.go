package getemoji

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	stdDraw "image/draw"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/kyokomi/emoji/v2"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/text/unicode/norm"
)

type Config struct {
	Size        int
	Emoji       string
	Output      string
	Outline     string
	OutlineSize int
}

var unicodeReg = regexp.MustCompile(`^[0-9a-fA-F]+(-[0-9a-fA-F])*$`)
var viewBoxReg = regexp.MustCompile(`(?i)\bviewBox\s*=\s*"([^"]+)"`)

const outlinedPNGScale = 4
const outlinedPNGMargin = 2

type outlineConfig struct {
	rgba color.RGBA
	hex  string
}

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
	if cfg.OutlineSize < 0 {
		return errors.New("outline size must be greater than or equal to 0")
	}
	outline, err := outlineColor(cfg.Outline)
	if err != nil {
		return err
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
		if outline != nil {
			svgContent = addSVGOutline(svgContent, outline.hex, svgOutlineRadius(cfg.OutlineSize))
		}
		return os.WriteFile(output, svgContent, 0644)
	case ".png":
		size := cfg.Size
		// Convert SVG to PNG
		icon, err := oksvg.ReadIconStream(strings.NewReader(string(svgContent)))
		if err != nil {
			return fmt.Errorf("couldn't read SVG: %v", err)
		}

		renderSize := size
		outlineRadius := 0
		padding := 0
		if outline != nil {
			renderSize = size * outlinedPNGScale
			outlineRadius = outlineWidth(size, cfg.OutlineSize) * outlinedPNGScale
			padding = outlinePadding(renderSize, outlineRadius)
		}

		icon.SetTarget(float64(padding), float64(padding), float64(renderSize-padding*2), float64(renderSize-padding*2))
		rgba := image.NewRGBA(image.Rect(0, 0, renderSize, renderSize))
		icon.Draw(rasterx.NewDasher(renderSize, renderSize, rasterx.NewScannerGV(renderSize, renderSize, rgba, rgba.Bounds())), 1)
		if outline != nil {
			if outlineRadius > 0 {
				rgba = addPNGOutline(rgba, outline.rgba, outlineRadius)
			}
			rgba = resizePNG(rgba, size)
		}

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

func outlineColor(input string) (*outlineConfig, error) {
	value := strings.ToLower(strings.TrimSpace(input))
	switch value {
	case "":
		return nil, nil
	case "white":
		return &outlineConfig{rgba: color.RGBA{R: 255, G: 255, B: 255, A: 255}, hex: "#fff"}, nil
	case "black":
		return &outlineConfig{rgba: color.RGBA{A: 255}, hex: "#000"}, nil
	}

	value = strings.TrimPrefix(value, "#")
	r, g, b, ok := parseHexColor(value)
	if !ok {
		return nil, fmt.Errorf("unsupported outline color: %s", input)
	}
	return &outlineConfig{
		rgba: color.RGBA{R: r, G: g, B: b, A: 255},
		hex:  fmt.Sprintf("#%02x%02x%02x", r, g, b),
	}, nil
}

func parseHexColor(value string) (uint8, uint8, uint8, bool) {
	switch len(value) {
	case 3:
		r, ok := parseHexByte(strings.Repeat(value[0:1], 2))
		if !ok {
			return 0, 0, 0, false
		}
		g, ok := parseHexByte(strings.Repeat(value[1:2], 2))
		if !ok {
			return 0, 0, 0, false
		}
		b, ok := parseHexByte(strings.Repeat(value[2:3], 2))
		if !ok {
			return 0, 0, 0, false
		}
		return r, g, b, true
	case 6:
		r, ok := parseHexByte(value[0:2])
		if !ok {
			return 0, 0, 0, false
		}
		g, ok := parseHexByte(value[2:4])
		if !ok {
			return 0, 0, 0, false
		}
		b, ok := parseHexByte(value[4:6])
		if !ok {
			return 0, 0, 0, false
		}
		return r, g, b, true
	default:
		return 0, 0, 0, false
	}
}

func parseHexByte(value string) (uint8, bool) {
	n, err := strconv.ParseUint(value, 16, 8)
	if err != nil {
		return 0, false
	}
	return uint8(n), true
}

func outlineWidth(size, configured int) int {
	if configured > 0 {
		return configured
	}
	return automaticOutlineWidth(size)
}

func automaticOutlineWidth(size int) int {
	width := size / 18
	if width < 1 {
		width = 1
	}
	if size <= width*2 {
		width = (size - 1) / 2
	}
	return width
}

func svgOutlineRadius(configured int) float64 {
	if configured > 0 {
		return float64(configured) / 4
	}
	return 1.6
}

func outlinePadding(size, radius int) int {
	padding := radius*2 + outlinedPNGMargin*outlinedPNGScale
	maxPadding := (size - 1) / 2
	if padding > maxPadding {
		padding = maxPadding
	}
	return padding
}

func resizePNG(src *image.RGBA, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

func addPNGOutline(src *image.RGBA, c color.RGBA, radius int) *image.RGBA {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	out := image.NewRGBA(bounds)
	distances := alphaDistanceSquared(src)
	radiusSquared := float64(radius * radius)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if distances[y*width+x] > radiusSquared {
				continue
			}
			offset := out.PixOffset(bounds.Min.X+x, bounds.Min.Y+y)
			out.Pix[offset+0] = c.R
			out.Pix[offset+1] = c.G
			out.Pix[offset+2] = c.B
			out.Pix[offset+3] = c.A
		}
	}

	stdDraw.Draw(out, bounds, src, bounds.Min, stdDraw.Over)
	return out
}

func alphaDistanceSquared(src *image.RGBA) []float64 {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	const inf = 1 << 30
	distances := make([]float64, width*height)

	for y := 0; y < height; y++ {
		row := make([]float64, width)
		for x := 0; x < width; x++ {
			row[x] = inf
			if src.Pix[src.PixOffset(bounds.Min.X+x, bounds.Min.Y+y)+3] > 0 {
				row[x] = 0
			}
		}
		copy(distances[y*width:(y+1)*width], distanceTransform1D(row))
	}

	for x := 0; x < width; x++ {
		column := make([]float64, height)
		for y := 0; y < height; y++ {
			column[y] = distances[y*width+x]
		}
		column = distanceTransform1D(column)
		for y := 0; y < height; y++ {
			distances[y*width+x] = column[y]
		}
	}

	return distances
}

func distanceTransform1D(values []float64) []float64 {
	n := len(values)
	result := make([]float64, n)
	v := make([]int, n)
	z := make([]float64, n+1)
	k := 0
	v[0] = 0
	z[0] = math.Inf(-1)
	z[1] = math.Inf(1)

	for q := 1; q < n; q++ {
		s := intersection(values, q, v[k])
		for s <= z[k] {
			k--
			s = intersection(values, q, v[k])
		}
		k++
		v[k] = q
		z[k] = s
		z[k+1] = math.Inf(1)
	}

	k = 0
	for q := 0; q < n; q++ {
		for z[k+1] < float64(q) {
			k++
		}
		d := float64(q - v[k])
		result[q] = d*d + values[v[k]]
	}
	return result
}

func intersection(values []float64, q, p int) float64 {
	return ((values[q] + float64(q*q)) - (values[p] + float64(p*p))) / float64(2*q-2*p)
}

func addSVGOutline(svgContent []byte, colorHex string, radius float64) []byte {
	svg := string(svgContent)
	start := strings.Index(svg, "<svg")
	if start == -1 {
		return svgContent
	}
	openEnd := strings.Index(svg[start:], ">")
	if openEnd == -1 {
		return svgContent
	}
	openEnd += start
	closeStart := strings.LastIndex(svg, "</svg>")
	if closeStart == -1 || closeStart <= openEnd {
		return svgContent
	}

	openTag := svg[start : openEnd+1]
	padding := math.Max(0.5, radius+0.5)
	filter := fmt.Sprintf(`<defs><filter id="getemoji-outline" x="-25%%" y="-25%%" width="150%%" height="150%%"><feMorphology in="SourceAlpha" operator="dilate" radius="%.2f" result="expanded"/><feFlood flood-color="%s" result="color"/><feComposite in="color" in2="expanded" operator="in" result="outline"/><feMerge><feMergeNode in="outline"/><feMergeNode in="SourceGraphic"/></feMerge></filter></defs>`, radius, colorHex)

	if minX, minY, width, height, ok := parseSVGViewBox(openTag); ok {
		filter = fmt.Sprintf(`<defs><filter id="getemoji-outline" filterUnits="userSpaceOnUse" x="%s" y="%s" width="%s" height="%s"><feMorphology in="SourceAlpha" operator="dilate" radius="%s" result="expanded"/><feFlood flood-color="%s" result="color"/><feComposite in="color" in2="expanded" operator="in" result="outline"/><feMerge><feMergeNode in="outline"/><feMergeNode in="SourceGraphic"/></feMerge></filter></defs>`,
			formatSVGFloat(minX-padding),
			formatSVGFloat(minY-padding),
			formatSVGFloat(width+padding*2),
			formatSVGFloat(height+padding*2),
			formatSVGFloat(radius),
			colorHex,
		)
		openTag = replaceSVGViewBox(openTag, minX-padding, minY-padding, width+padding*2, height+padding*2)
	}

	var builder strings.Builder
	builder.Grow(len(svg) + len(filter) + len(`<g filter="url(#getemoji-outline)"></g>`))
	builder.WriteString(svg[:start])
	builder.WriteString(openTag)
	builder.WriteString(filter)
	builder.WriteString(`<g filter="url(#getemoji-outline)">`)
	builder.WriteString(svg[openEnd+1 : closeStart])
	builder.WriteString(`</g>`)
	builder.WriteString(svg[closeStart:])
	return []byte(builder.String())
}

func parseSVGViewBox(openTag string) (float64, float64, float64, float64, bool) {
	match := viewBoxReg.FindStringSubmatch(openTag)
	if len(match) < 2 {
		return 0, 0, 0, 0, false
	}
	fields := strings.Fields(match[1])
	if len(fields) != 4 {
		return 0, 0, 0, 0, false
	}
	minX, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, 0, 0, 0, false
	}
	minY, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return 0, 0, 0, 0, false
	}
	width, err := strconv.ParseFloat(fields[2], 64)
	if err != nil || width <= 0 {
		return 0, 0, 0, 0, false
	}
	height, err := strconv.ParseFloat(fields[3], 64)
	if err != nil || height <= 0 {
		return 0, 0, 0, 0, false
	}
	return minX, minY, width, height, true
}

func replaceSVGViewBox(openTag string, minX, minY, width, height float64) string {
	replacement := fmt.Sprintf(`viewBox="%s %s %s %s"`,
		formatSVGFloat(minX),
		formatSVGFloat(minY),
		formatSVGFloat(width),
		formatSVGFloat(height),
	)
	return viewBoxReg.ReplaceAllString(openTag, replacement)
}

func formatSVGFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
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
