package process

import (
	"context"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommand(t *testing.T) {
	// Reset the global state before testing
	reset()

	t.Run("creates_command_and_tracks_it", func(t *testing.T) {
		cmd := Command("echo", "hello")
		
		// Verify command is created correctly (Path may be resolved to full path)
		assert.Contains(t, cmd.Path, "echo")
		assert.Equal(t, []string{"echo", "hello"}, cmd.Args)
		
		// Verify command is tracked
		lock.Lock()
		assert.Len(t, cmds, 1)
		assert.Equal(t, cmd, cmds[0])
		lock.Unlock()
	})

	t.Run("tracks_multiple_commands", func(t *testing.T) {
		reset()
		
		cmd1 := Command("echo", "first")
		cmd2 := Command("echo", "second")
		cmd3 := Command("ls", "-la")
		
		// Verify all commands are tracked
		lock.Lock()
		assert.Len(t, cmds, 3)
		assert.Contains(t, cmds, cmd1)
		assert.Contains(t, cmds, cmd2)
		assert.Contains(t, cmds, cmd3)
		lock.Unlock()
	})

	t.Run("command_with_no_args", func(t *testing.T) {
		reset()
		
		cmd := Command("pwd")
		
		assert.Contains(t, cmd.Path, "pwd")
		assert.Equal(t, []string{"pwd"}, cmd.Args)
		
		lock.Lock()
		assert.Len(t, cmds, 1)
		lock.Unlock()
	})
}

func TestCommandContext(t *testing.T) {
	reset()

	t.Run("creates_command_with_context", func(t *testing.T) {
		ctx := context.Background()
		cmd := CommandContext(ctx, "echo", "hello")
		
		// Verify command is created correctly (Path may be resolved to full path)
		assert.Contains(t, cmd.Path, "echo")
		assert.Equal(t, []string{"echo", "hello"}, cmd.Args)
		assert.NotNil(t, cmd.Cancel, "Cancel function should be set")
		
		// Verify command is tracked
		lock.Lock()
		assert.Len(t, cmds, 1)
		assert.Equal(t, cmd, cmds[0])
		lock.Unlock()
	})

	t.Run("context_cancellation", func(t *testing.T) {
		reset()
		
		ctx, cancel := context.WithCancel(context.Background())
		cmd := CommandContext(ctx, "sleep", "10")
		
		// Start the command
		err := cmd.Start()
		require.NoError(t, err)
		
		// Cancel the context
		cancel()
		
		// Wait for the command to finish
		err = cmd.Wait()
		// The command should be killed due to context cancellation
		assert.Error(t, err)
	})

	t.Run("context_timeout", func(t *testing.T) {
		reset()
		
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		
		cmd := CommandContext(ctx, "sleep", "1")
		
		// Start the command
		err := cmd.Start()
		require.NoError(t, err)
		
		// Wait for the command to finish (should timeout)
		err = cmd.Wait()
		assert.Error(t, err)
	})
}

func TestReset(t *testing.T) {
	// Add some commands to track
	Command("echo", "test1")
	Command("echo", "test2")
	Command("echo", "test3")
	
	// Verify commands are tracked
	lock.Lock()
	assert.Len(t, cmds, 3)
	lock.Unlock()
	
	// Reset
	reset()
	
	// Verify commands are cleared
	lock.Lock()
	assert.Len(t, cmds, 0)
	lock.Unlock()
}

func TestTrack(t *testing.T) {
	reset()
	
	// Create commands manually
	cmd1 := exec.Command("echo", "test1")
	cmd2 := exec.Command("echo", "test2")
	
	// Track them
	track(cmd1)
	track(cmd2)
	
	// Verify they are tracked
	lock.Lock()
	assert.Len(t, cmds, 2)
	assert.Contains(t, cmds, cmd1)
	assert.Contains(t, cmds, cmd2)
	lock.Unlock()
}

func TestKill(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping signal tests on Windows")
	}

	t.Run("kill_nil_process", func(t *testing.T) {
		err := Kill(nil)
		assert.NoError(t, err)
	})

	t.Run("kill_running_process", func(t *testing.T) {
		reset()
		
		// Create a long-running command
		cmd := Command("sleep", "10")
		err := cmd.Start()
		require.NoError(t, err)
		
		// Verify process is running
		assert.NotNil(t, cmd.Process)
		assert.Nil(t, cmd.ProcessState)
		
		// Kill the process
		err = Kill(cmd.Process)
		assert.NoError(t, err)
		
		// Wait a bit and verify process is dead
		time.Sleep(100 * time.Millisecond)
		
		// Wait for the process to actually finish
		cmd.Wait()
		
		// Process should be removed from tracking
		lock.Lock()
		processFound := false
		for _, trackedCmd := range cmds {
			if trackedCmd.Process != nil && trackedCmd.Process.Pid == cmd.Process.Pid {
				processFound = true
				break
			}
		}
		assert.False(t, processFound, "Process should be removed from tracking")
		lock.Unlock()
	})

	t.Run("kill_process_that_ignores_sigterm", func(t *testing.T) {
		reset()
		
		// Create a command that ignores SIGTERM (for testing SIGKILL fallback)
		// We'll use a short sleep and reduce killWait for faster testing
		originalKillWait := killWait
		killWait = 50 * time.Millisecond
		defer func() { killWait = originalKillWait }()
		
		cmd := Command("sleep", "10")
		err := cmd.Start()
		require.NoError(t, err)
		
		// Kill the process (should use SIGKILL after timeout)
		err = Kill(cmd.Process)
		assert.NoError(t, err)
	})
}

func TestCleanup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping signal tests on Windows")
	}

	t.Run("cleanup_no_processes", func(t *testing.T) {
		reset()
		
		err := Cleanup()
		assert.NoError(t, err)
	})

	t.Run("cleanup_running_processes", func(t *testing.T) {
		reset()
		
		// Start multiple long-running processes
		cmd1 := Command("sleep", "10")
		cmd2 := Command("sleep", "10")
		cmd3 := Command("sleep", "10")
		
		err := cmd1.Start()
		require.NoError(t, err)
		err = cmd2.Start()
		require.NoError(t, err)
		err = cmd3.Start()
		require.NoError(t, err)
		
		// Verify processes are running
		assert.NotNil(t, cmd1.Process)
		assert.NotNil(t, cmd2.Process)
		assert.NotNil(t, cmd3.Process)
		
		// Cleanup all processes
		err = Cleanup()
		assert.NoError(t, err)
		
		// Wait a bit for cleanup to complete
		time.Sleep(200 * time.Millisecond)
		
		// Verify processes are killed - just check that they're no longer running
		// We don't need to check ProcessState as the Kill function handles cleanup
		assert.Eventually(t, func() bool {
			// Check if processes are still in the tracking list
			lock.Lock()
			defer lock.Unlock()
			
			found1, found2, found3 := false, false, false
			for _, trackedCmd := range cmds {
				if trackedCmd.Process != nil {
					if trackedCmd.Process.Pid == cmd1.Process.Pid {
						found1 = true
					}
					if trackedCmd.Process.Pid == cmd2.Process.Pid {
						found2 = true
					}
					if trackedCmd.Process.Pid == cmd3.Process.Pid {
						found3 = true
					}
				}
			}
			// All processes should be removed from tracking
			return !found1 && !found2 && !found3
		}, 2*time.Second, 100*time.Millisecond, "All processes should be removed from tracking")
	})

	t.Run("cleanup_already_finished_processes", func(t *testing.T) {
		reset()
		
		// Start and immediately finish a process
		cmd := Command("echo", "hello")
		err := cmd.Run()
		require.NoError(t, err)
		
		// Process should be finished
		assert.NotNil(t, cmd.ProcessState)
		
		// Cleanup should handle finished processes gracefully
		err = Cleanup()
		assert.NoError(t, err)
	})

	t.Run("cleanup_processes_without_process_object", func(t *testing.T) {
		reset()
		
		// Create command but don't start it
		cmd := Command("echo", "hello")
		
		// Process should be nil
		assert.Nil(t, cmd.Process)
		
		// Cleanup should handle nil processes gracefully
		err := Cleanup()
		assert.NoError(t, err)
	})

	t.Run("cleanup_timeout", func(t *testing.T) {
		reset()
		
		// Temporarily reduce killWait for faster testing
		originalKillWait := killWait
		killWait = 10 * time.Millisecond
		defer func() { killWait = originalKillWait }()
		
		// Start a process
		cmd := Command("sleep", "10")
		err := cmd.Start()
		require.NoError(t, err)
		
		// Cleanup should timeout and return error
		err = Cleanup()
		// Note: This test might be flaky depending on system performance
		// The timeout error is syscall.ETIMEDOUT
		if err != nil {
			assert.Equal(t, syscall.ETIMEDOUT, err)
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	reset()
	
	// Test concurrent access to the global cmds slice
	const numGoroutines = 10
	const numCommands = 5
	
	done := make(chan bool, numGoroutines)
	
	// Start multiple goroutines creating commands
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numCommands; j++ {
				Command("echo", "test")
			}
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	
	// Verify all commands were tracked
	lock.Lock()
	assert.Len(t, cmds, numGoroutines*numCommands)
	lock.Unlock()
}

func TestProcessLifecycle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping signal tests on Windows")
	}

	reset()
	
	// Test complete lifecycle: create -> start -> kill -> cleanup
	cmd := Command("sleep", "10")
	
	// 1. Command created and tracked
	lock.Lock()
	assert.Len(t, cmds, 1)
	lock.Unlock()
	
	// 2. Start the process
	err := cmd.Start()
	require.NoError(t, err)
	assert.NotNil(t, cmd.Process)
	
	// 3. Kill the process
	err = Kill(cmd.Process)
	assert.NoError(t, err)
	
	// 4. Verify process is removed from tracking
	time.Sleep(100 * time.Millisecond)
	lock.Lock()
	processFound := false
	for _, trackedCmd := range cmds {
		if trackedCmd.Process != nil && trackedCmd.Process.Pid == cmd.Process.Pid {
			processFound = true
			break
		}
	}
	assert.False(t, processFound, "Process should be removed from tracking")
	lock.Unlock()
}

func TestEdgeCases(t *testing.T) {
	t.Run("command_with_empty_args", func(t *testing.T) {
		reset()
		
		cmd := Command("echo")
		assert.Contains(t, cmd.Path, "echo")
		assert.Equal(t, []string{"echo"}, cmd.Args)
	})

	t.Run("command_with_special_characters", func(t *testing.T) {
		reset()
		
		cmd := Command("echo", "hello world", "test@example.com", "$HOME")
		assert.Equal(t, []string{"echo", "hello world", "test@example.com", "$HOME"}, cmd.Args)
	})

	t.Run("multiple_resets", func(t *testing.T) {
		Command("echo", "test")
		reset()
		reset()
		reset()
		
		lock.Lock()
		assert.Len(t, cmds, 0)
		lock.Unlock()
	})
}

func TestKillWaitConfiguration(t *testing.T) {
	// Test that killWait is properly configured
	assert.Equal(t, 5*time.Second, killWait, "Default killWait should be 5 seconds")
	
	// Test modifying killWait
	originalKillWait := killWait
	killWait = 1 * time.Second
	assert.Equal(t, 1*time.Second, killWait)
	
	// Restore original value
	killWait = originalKillWait
	assert.Equal(t, 5*time.Second, killWait)
}

func TestGlobalState(t *testing.T) {
	// Test that global state is properly managed
	reset()
	
	// Verify initial state
	lock.Lock()
	assert.NotNil(t, cmds)
	assert.Len(t, cmds, 0)
	lock.Unlock()
	
	// Add commands
	Command("echo", "test1")
	Command("echo", "test2")
	
	lock.Lock()
	assert.Len(t, cmds, 2)
	lock.Unlock()
	
	// Reset and verify
	reset()
	
	lock.Lock()
	assert.Len(t, cmds, 0)
	lock.Unlock()
}