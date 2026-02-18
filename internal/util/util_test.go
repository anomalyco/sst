package util_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/sst/sst/v3/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomString(t *testing.T) {
	t.Run("correct length", func(t *testing.T) {
		assert.Len(t, util.RandomString(10), 10)
		assert.Len(t, util.RandomString(0), 0)
		assert.Len(t, util.RandomString(50), 50)
	})

	t.Run("valid charset", func(t *testing.T) {
		const charset = "abcdefhkmnorstuvwxz"
		s := util.RandomString(100)
		for _, c := range s {
			assert.Contains(t, charset, string(c))
		}
	})

	t.Run("different outputs", func(t *testing.T) {
		a := util.RandomString(20)
		b := util.RandomString(20)
		assert.NotEqual(t, a, b)
	})
}

func TestReadableError(t *testing.T) {
	inner := errors.New("inner")

	t.Run("NewReadableError", func(t *testing.T) {
		err := util.NewReadableError(inner, "readable msg")
		assert.Equal(t, "readable msg", err.Error())
		assert.Equal(t, inner, err.Unwrap())
		assert.False(t, err.IsHinted())
	})

	t.Run("NewHintedError", func(t *testing.T) {
		err := util.NewHintedError(inner, "hint msg")
		assert.Equal(t, "hint msg", err.Error())
		assert.Equal(t, inner, err.Unwrap())
		assert.True(t, err.IsHinted())
	})
}

func TestKeyLock(t *testing.T) {
	t.Run("lock unlock basic", func(t *testing.T) {
		kl := util.NewKeyLock()
		kl.Lock("a")
		kl.Unlock("a")
		// should not deadlock
		kl.Lock("a")
		kl.Unlock("a")
	})

	t.Run("concurrent lock blocks", func(t *testing.T) {
		kl := util.NewKeyLock()
		kl.Lock("key")

		started := make(chan struct{})
		done := make(chan struct{})
		go func() {
			close(started)
			kl.Lock("key")
			close(done)
			kl.Unlock("key")
		}()

		<-started
		select {
		case <-done:
			t.Fatal("second lock should block")
		default:
		}

		kl.Unlock("key")
		<-done // now it should complete
	})

	t.Run("different keys independent", func(t *testing.T) {
		kl := util.NewKeyLock()
		kl.Lock("a")
		kl.Lock("b") // should not block
		kl.Unlock("a")
		kl.Unlock("b")
	})
}

func TestSyncMap(t *testing.T) {
	t.Run("store and load", func(t *testing.T) {
		var m util.SyncMap[string, int]
		m.Store("x", 42)
		v, ok := m.Load("x")
		require.True(t, ok)
		assert.Equal(t, 42, v)
	})

	t.Run("load missing", func(t *testing.T) {
		var m util.SyncMap[string, int]
		_, ok := m.Load("missing")
		assert.False(t, ok)
	})

	t.Run("delete", func(t *testing.T) {
		var m util.SyncMap[string, int]
		m.Store("x", 1)
		m.Delete("x")
		_, ok := m.Load("x")
		assert.False(t, ok)
	})

	t.Run("load or store", func(t *testing.T) {
		var m util.SyncMap[string, int]
		m.Store("x", 1)
		v, loaded := m.LoadOrStore("x", 99)
		assert.True(t, loaded)
		assert.Equal(t, 1, v)

		v, loaded = m.LoadOrStore("y", 99)
		assert.False(t, loaded)
		assert.Equal(t, 99, v)
	})

	t.Run("load and delete", func(t *testing.T) {
		var m util.SyncMap[string, int]
		m.Store("x", 5)
		v, loaded := m.LoadAndDelete("x")
		assert.True(t, loaded)
		assert.Equal(t, 5, v)

		_, ok := m.Load("x")
		assert.False(t, ok)
	})

	t.Run("range", func(t *testing.T) {
		var m util.SyncMap[string, int]
		m.Store("a", 1)
		m.Store("b", 2)

		collected := map[string]int{}
		var mu sync.Mutex
		m.Range(func(k string, v int) bool {
			mu.Lock()
			collected[k] = v
			mu.Unlock()
			return true
		})
		assert.Equal(t, map[string]int{"a": 1, "b": 2}, collected)
	})
}
