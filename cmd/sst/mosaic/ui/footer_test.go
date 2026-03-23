package ui

import (
	"io"
	"os"
	"testing"
)

func TestFooterRenderSkipsIdenticalFrame(t *testing.T) {
	m := NewFooter()
	if out := captureFooterOutput(t, func() { m.Render(80, "deploying") }); out == "" {
		t.Fatal("expected initial render output")
	}
	if out := captureFooterOutput(t, func() { m.Render(80, "deploying") }); out != "" {
		t.Fatalf("expected identical frame to be skipped, got %q", out)
	}
}

func TestFooterRenderRedrawsOnResize(t *testing.T) {
	m := NewFooter()
	captureFooterOutput(t, func() { m.Render(80, "deploying") })
	if out := captureFooterOutput(t, func() { m.Render(10, "deploying") }); out == "" {
		t.Fatal("expected resize to trigger redraw")
	}
}

func captureFooterOutput(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}
