package getemoji

import (
	"strings"
	"testing"
)

func TestParseSVGViewBox(t *testing.T) {
	minX, minY, width, height, ok := parseSVGViewBox(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 36 36">`)
	if !ok {
		t.Fatalf("expected viewBox to be parsed")
	}
	if minX != 0 || minY != 0 || width != 36 || height != 36 {
		t.Fatalf("unexpected values: %v %v %v %v", minX, minY, width, height)
	}
}

func TestAddSVGOutlineExpandsViewBoxAndFilterBounds(t *testing.T) {
	input := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 36 36"><path d="M0 0h36v36H0z"/></svg>`)
	got := string(addSVGOutline(input, "#fff", 1))

	if !strings.Contains(got, `viewBox="-1.5 -1.5 39 39"`) {
		t.Fatalf("expected expanded viewBox, got: %s", got)
	}
	if !strings.Contains(got, `filterUnits="userSpaceOnUse"`) {
		t.Fatalf("expected user-space filter bounds, got: %s", got)
	}
	if !strings.Contains(got, `x="-1.5" y="-1.5" width="39" height="39"`) {
		t.Fatalf("expected expanded filter bounds, got: %s", got)
	}
}
