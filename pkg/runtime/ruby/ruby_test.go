package ruby

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatch(t *testing.T) {
	r := New()
	assert.True(t, r.Match("ruby3.2"))
	assert.True(t, r.Match("ruby3.3"))
	assert.False(t, r.Match("python3.11"))
	assert.False(t, r.Match("nodejs18.x"))
}

func TestShouldRebuild(t *testing.T) {
	r := New()
	assert.True(t, r.ShouldRebuild("func", "handler.rb"))
	assert.True(t, r.ShouldRebuild("func", "Gemfile"))
	assert.True(t, r.ShouldRebuild("func", "Gemfile.lock"))
	assert.False(t, r.ShouldRebuild("func", "handler.py"))
}
