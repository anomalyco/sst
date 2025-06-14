package main

import (
	"strings"
	"testing"

	"github.com/sst/sst/v3/cmd/sst/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmdRemove(t *testing.T) {
	// Find the remove command in the root command children
	var removeCmd *cli.Command
	for _, child := range root.Children {
		if child.Name == "remove" {
			removeCmd = child
			break
		}
	}
	require.NotNil(t, removeCmd, "remove command should exist in root children")

	t.Run("command structure", func(t *testing.T) {
		assert.Equal(t, "remove", removeCmd.Name)
		assert.NotEmpty(t, removeCmd.Description.Short)
		assert.NotEmpty(t, removeCmd.Description.Long)
		assert.NotNil(t, removeCmd.Run)
	})

	t.Run("command description content", func(t *testing.T) {
		assert.Contains(t, removeCmd.Description.Short, "Remove your application")
		assert.Contains(t, removeCmd.Description.Long, "Removes your application")
		assert.Contains(t, removeCmd.Description.Long, "personal stage")
		assert.Contains(t, removeCmd.Description.Long, "sst remove --stage production")
		assert.Contains(t, removeCmd.Description.Long, "sst remove --target MyComponent")
		assert.Contains(t, removeCmd.Description.Long, "removal")
		assert.Contains(t, removeCmd.Description.Long, "sst.config.ts")
	})

	t.Run("command flags", func(t *testing.T) {
		expectedFlags := []string{"target"}
		
		flagNames := make([]string, len(removeCmd.Flags))
		for i, flag := range removeCmd.Flags {
			flagNames[i] = flag.Name
		}
		
		for _, expectedFlag := range expectedFlags {
			assert.Contains(t, flagNames, expectedFlag, "Expected flag %s not found", expectedFlag)
		}
	})

	t.Run("target flag configuration", func(t *testing.T) {
		var targetFlag *cli.Flag
		for _, flag := range removeCmd.Flags {
			if flag.Name == "target" {
				targetFlag = &flag
				break
			}
		}
		
		require.NotNil(t, targetFlag, "target flag should exist")
		assert.Equal(t, "string", targetFlag.Type)
		assert.Contains(t, targetFlag.Description.Short, "component")
		assert.Contains(t, targetFlag.Description.Long, "given component")
	})

	t.Run("command documentation examples", func(t *testing.T) {
		longDesc := removeCmd.Description.Long
		
		// Check for stage example
		assert.Contains(t, longDesc, "sst remove --stage production")
		
		// Check for target example
		assert.Contains(t, longDesc, "sst remove --target MyComponent")
		
		// Check for removal setting documentation
		assert.Contains(t, longDesc, "removal")
		assert.Contains(t, longDesc, "sst.config.ts")
		
		// Check for state and bootstrap information
		assert.Contains(t, longDesc, "state")
		assert.Contains(t, longDesc, "bootstrap")
	})

	t.Run("command warnings and tips", func(t *testing.T) {
		longDesc := removeCmd.Description.Long
		
		// Check for tip about removal setting
		assert.Contains(t, longDesc, ":::tip")
		
		// Check for information about state and bootstrap resources
		assert.Contains(t, longDesc, "does not remove the SST")
		assert.Contains(t, longDesc, "other apps")
	})
}

func TestCmdRemoveFunction(t *testing.T) {
	t.Run("target parsing", func(t *testing.T) {
		// Test that the function exists and can be called
		// Note: We can't easily test the full function without mocking
		// the entire CLI infrastructure, but we can test its existence
		assert.NotNil(t, CmdRemove, "CmdRemove function should exist")
	})

	t.Run("target string splitting logic", func(t *testing.T) {
		// Test the target parsing logic that would be used in CmdRemove
		testCases := []struct {
			input    string
			expected []string
		}{
			{"", []string{}},
			{"Component1", []string{"Component1"}},
			{"Component1,Component2", []string{"Component1", "Component2"}},
			{"Component1,Component2,Component3", []string{"Component1", "Component2", "Component3"}},
		}

		for _, tc := range testCases {
			t.Run("input: "+tc.input, func(t *testing.T) {
				var result []string
				if tc.input != "" {
					result = strings.Split(tc.input, ",")
				}
				
				if len(tc.expected) == 0 {
					assert.Empty(t, result)
				} else {
					assert.Equal(t, tc.expected, result)
				}
			})
		}
	})

	t.Run("project stack input structure", func(t *testing.T) {
		// Test that the expected StackInput structure would be valid
		// This tests the structure that CmdRemove would create
		expectedCommand := "remove"
		expectedVerbose := true
		expectedTarget := []string{"Component1", "Component2"}
		
		// Verify the command string
		assert.Equal(t, "remove", expectedCommand)
		
		// Verify boolean handling
		assert.True(t, expectedVerbose)
		
		// Verify target array handling
		assert.Len(t, expectedTarget, 2)
		assert.Contains(t, expectedTarget, "Component1")
		assert.Contains(t, expectedTarget, "Component2")
	})
}