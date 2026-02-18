package js_test

import (
	"testing"

	esbuild "github.com/evanw/esbuild/pkg/api"
	"github.com/sst/sst/v3/pkg/js"
	"github.com/stretchr/testify/assert"
)

func TestFormatError(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		assert.Equal(t, "", js.FormatError([]esbuild.Message{}))
	})

	t.Run("no location", func(t *testing.T) {
		msgs := []esbuild.Message{{Text: "something broke"}}
		assert.Equal(t, "something broke", js.FormatError(msgs))
	})

	t.Run("with location", func(t *testing.T) {
		msgs := []esbuild.Message{{
			Text:     "unexpected token",
			Location: &esbuild.Location{File: "app.ts", Line: 10, Column: 5},
		}}
		assert.Equal(t, "app.ts:10:5: unexpected token", js.FormatError(msgs))
	})

	t.Run("multiple errors", func(t *testing.T) {
		msgs := []esbuild.Message{
			{Text: "err1"},
			{Text: "err2", Location: &esbuild.Location{File: "b.ts", Line: 2, Column: 3}},
		}
		assert.Equal(t, "err1\nb.ts:2:3: err2", js.FormatError(msgs))
	})
}
