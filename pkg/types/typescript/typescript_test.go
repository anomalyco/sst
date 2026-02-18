package typescript

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInfer(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		result := infer(map[string]interface{}{})
		assert.Equal(t, "{\n}", result)
	})

	t.Run("string value", func(t *testing.T) {
		result := infer(map[string]interface{}{"name": "hello"})
		assert.Equal(t, "{\n  \"name\": string\n}", result)
	})

	t.Run("number types", func(t *testing.T) {
		result := infer(map[string]interface{}{"a": 42})
		assert.Contains(t, result, "\"a\": number")

		result = infer(map[string]interface{}{"b": 3.14})
		assert.Contains(t, result, "\"b\": number")

		result = infer(map[string]interface{}{"c": float32(1.0)})
		assert.Contains(t, result, "\"c\": number")
	})

	t.Run("bool value", func(t *testing.T) {
		result := infer(map[string]interface{}{"flag": true})
		assert.Contains(t, result, "\"flag\": boolean")
	})

	t.Run("nested map", func(t *testing.T) {
		input := map[string]interface{}{
			"db": map[string]interface{}{
				"host": "localhost",
			},
		}
		result := infer(input)
		assert.Contains(t, result, "\"db\": {")
		assert.Contains(t, result, "\"host\": string")
	})

	t.Run("type key at top level quoted", func(t *testing.T) {
		input := map[string]interface{}{"type": "MyType"}
		result := infer(input, "")
		assert.Contains(t, result, "\"type\": \"MyType\"")
	})

	t.Run("type key nested still quoted", func(t *testing.T) {
		input := map[string]interface{}{
			"inner": map[string]interface{}{
				"type": "nested",
			},
		}
		// recursive calls also pass 1 indent arg, so type is always quoted
		result := infer(input, "")
		assert.Contains(t, result, "\"type\": \"nested\"")
	})

	t.Run("literal passthrough", func(t *testing.T) {
		input := map[string]interface{}{
			"binding": literal{value: "cloudflare.R2Bucket"},
		}
		result := infer(input)
		assert.Contains(t, result, "\"binding\": cloudflare.R2Bucket")
	})

	t.Run("nil value", func(t *testing.T) {
		result := infer(map[string]interface{}{"x": nil})
		assert.Contains(t, result, "\"x\": any")
	})

	t.Run("sorted keys", func(t *testing.T) {
		input := map[string]interface{}{
			"z": "a",
			"a": "b",
			"m": "c",
		}
		result := infer(input)
		aIdx := indexOf(result, "\"a\"")
		mIdx := indexOf(result, "\"m\"")
		zIdx := indexOf(result, "\"z\"")
		assert.Less(t, aIdx, mIdx)
		assert.Less(t, mIdx, zIdx)
	})
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
