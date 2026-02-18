package python

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInferType(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"string", "hello", "str"},
		{"int", 42, "int"},
		{"float64", 3.14, "float"},
		{"float32", float32(1.0), "float"},
		{"bool", true, "bool"},
		{"map", map[string]interface{}{}, "dict"},
		{"nil", nil, "Any"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, inferType(tt.value))
		})
	}
}

func TestInfer(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		assert.Equal(t, "", infer(map[string]interface{}{}))
	})

	t.Run("flat map sorted", func(t *testing.T) {
		input := map[string]interface{}{
			"name":  "hello",
			"age":   42,
			"alive": true,
		}
		expected := "age: int\nalive: bool\nname: str\n"
		assert.Equal(t, expected, infer(input))
	})

	t.Run("nested map", func(t *testing.T) {
		input := map[string]interface{}{
			"User": map[string]interface{}{
				"name": "hello",
			},
		}
		expected := "class User:\n    name: str\n"
		assert.Equal(t, expected, infer(input))
	})

	t.Run("custom indent", func(t *testing.T) {
		input := map[string]interface{}{
			"x": "hello",
		}
		expected := "  x: str\n"
		assert.Equal(t, expected, infer(input, "  "))
	})
}
