package task_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/sst/sst/v3/pkg/task"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Run("succeeds first try", func(t *testing.T) {
		val, err := task.Run(context.Background(), func() (string, error) {
			return "ok", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "ok", val)
	})

	t.Run("fails twice then succeeds", func(t *testing.T) {
		var count atomic.Int32
		val, err := task.Run(context.Background(), func() (string, error) {
			n := count.Add(1)
			if n < 3 {
				return "", fmt.Errorf("fail %d", n)
			}
			return "ok", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "ok", val)
	})

	t.Run("fails 3 times returns last error", func(t *testing.T) {
		var count atomic.Int32
		_, err := task.Run(context.Background(), func() (string, error) {
			n := count.Add(1)
			return "", fmt.Errorf("fail %d", n)
		})
		require.Error(t, err)
		assert.Equal(t, "fail 3", err.Error())
	})

	t.Run("context cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := task.Run(ctx, func() (string, error) {
			select {} // block forever
		})
		assert.ErrorIs(t, err, context.Canceled)
	})
}
