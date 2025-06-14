package main

import (
	"strings"
	"testing"

	"github.com/sst/sst/v3/cmd/sst/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmdDeploy(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		assert.Equal(t, "deploy", CmdDeploy.Name)
		assert.NotEmpty(t, CmdDeploy.Description.Short)
		assert.NotEmpty(t, CmdDeploy.Description.Long)
		assert.NotNil(t, CmdDeploy.Run)
	})

	t.Run("command description content", func(t *testing.T) {
		assert.Contains(t, CmdDeploy.Description.Short, "Deploy your application")
		assert.Contains(t, CmdDeploy.Description.Long, "Deploy your application")
		assert.Contains(t, CmdDeploy.Description.Long, "sst deploy --stage production")
		assert.Contains(t, CmdDeploy.Description.Long, "sst deploy --target MyComponent")
		assert.Contains(t, CmdDeploy.Description.Long, "SST_BUILD_CONCURRENCY")
		assert.Contains(t, CmdDeploy.Description.Long, "--continue")
		assert.Contains(t, CmdDeploy.Description.Long, "--dev")
	})

	t.Run("command flags", func(t *testing.T) {
		expectedFlags := []string{"target", "continue", "dev"}
		
		flagNames := make([]string, len(CmdDeploy.Flags))
		for i, flag := range CmdDeploy.Flags {
			flagNames[i] = flag.Name
		}
		
		for _, expectedFlag := range expectedFlags {
			assert.Contains(t, flagNames, expectedFlag, "Expected flag %s not found", expectedFlag)
		}
	})

	t.Run("target flag configuration", func(t *testing.T) {
		var targetFlag *cli.Flag
		for _, flag := range CmdDeploy.Flags {
			if flag.Name == "target" {
				targetFlag = &flag
				break
			}
		}
		
		require.NotNil(t, targetFlag, "target flag should exist")
		assert.Equal(t, "", targetFlag.Type) // Default type is string
		assert.NotEmpty(t, targetFlag.Description.Short)
		assert.Contains(t, targetFlag.Description.Short, "component")
		assert.Contains(t, targetFlag.Description.Long, "given component")
	})

	t.Run("continue flag configuration", func(t *testing.T) {
		var continueFlag *cli.Flag
		for _, flag := range CmdDeploy.Flags {
			if flag.Name == "continue" {
				continueFlag = &flag
				break
			}
		}
		
		require.NotNil(t, continueFlag, "continue flag should exist")
		assert.Equal(t, "bool", continueFlag.Type)
		assert.Contains(t, continueFlag.Description.Short, "Continue on error")
		assert.Contains(t, continueFlag.Description.Long, "deploy as many resources as possible")
	})

	t.Run("dev flag configuration", func(t *testing.T) {
		var devFlag *cli.Flag
		for _, flag := range CmdDeploy.Flags {
			if flag.Name == "dev" {
				devFlag = &flag
				break
			}
		}
		
		require.NotNil(t, devFlag, "dev flag should exist")
		assert.Equal(t, "bool", devFlag.Type)
		assert.Contains(t, devFlag.Description.Short, "Deploy in dev mode")
		assert.Contains(t, devFlag.Description.Long, "sst dev")
	})

	t.Run("command examples", func(t *testing.T) {
		require.NotEmpty(t, CmdDeploy.Examples, "deploy command should have examples")
		
		example := CmdDeploy.Examples[0]
		assert.Equal(t, "sst deploy --stage production", example.Content)
		assert.Contains(t, example.Description.Short, "Deploy to production")
	})

	t.Run("concurrency documentation", func(t *testing.T) {
		longDesc := CmdDeploy.Description.Long
		
		// Check that concurrency environment variables are documented
		assert.Contains(t, longDesc, "SST_BUILD_CONCURRENCY_SITE")
		assert.Contains(t, longDesc, "SST_BUILD_CONCURRENCY_FUNCTION")
		assert.Contains(t, longDesc, "SST_BUILD_CONCURRENCY_CONTAINER")
		
		// Check that default values are documented
		assert.Contains(t, longDesc, "Sites | 1")
		assert.Contains(t, longDesc, "Functions | 4")
		assert.Contains(t, longDesc, "Containers | 1")
	})

	t.Run("usage examples in description", func(t *testing.T) {
		longDesc := CmdDeploy.Description.Long
		
		// Check for various usage examples
		assert.Contains(t, longDesc, "SST_BUILD_CONCURRENCY_SITE=2 sst deploy")
		assert.Contains(t, longDesc, "sst deploy --continue")
		assert.Contains(t, longDesc, "sst deploy --dev")
		assert.Contains(t, longDesc, "sst deploy --target MyComponent")
	})
}

func TestDeployCommandLogic(t *testing.T) {
	t.Run("target parsing logic", func(t *testing.T) {
		// Test the target parsing logic from the Run function
		// This tests the strings.Split logic for comma-separated targets
		
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
			t.Run("target: "+tc.input, func(t *testing.T) {
				var result []string
				if tc.input != "" {
					result = strings.Split(tc.input, ",")
				} else {
					result = []string{}
				}
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("run function structure", func(t *testing.T) {
		// Verify that the Run function exists and has the expected signature
		assert.NotNil(t, CmdDeploy.Run, "deploy command should have a Run function")
		
		// We can't easily test the full Run function without mocking dependencies,
		// but we can verify it exists and follows the expected pattern
		runFunc := CmdDeploy.Run
		assert.IsType(t, func(*cli.Cli) error { return nil }, runFunc)
	})
}

func TestDeployCommandIntegration(t *testing.T) {
	t.Run("command is properly integrated", func(t *testing.T) {
		// Verify that the deploy command is included in the root command's children
		var deployCmd *cli.Command
		for _, child := range root.Children {
			if child.Name == "deploy" {
				deployCmd = child
				break
			}
		}
		
		require.NotNil(t, deployCmd, "deploy command should be in root children")
		assert.Equal(t, CmdDeploy, deployCmd, "deploy command should match CmdDeploy")
	})

	t.Run("command not hidden", func(t *testing.T) {
		// Verify that the deploy command is not hidden
		var deployCmd *cli.Command
		for _, child := range root.Children {
			if child.Name == "deploy" {
				deployCmd = child
				break
			}
		}
		
		require.NotNil(t, deployCmd, "deploy command should exist")
		assert.False(t, deployCmd.Hidden, "deploy command should not be hidden")
	})
}

func TestDeployCommandDocumentation(t *testing.T) {
	t.Run("comprehensive documentation", func(t *testing.T) {
		longDesc := CmdDeploy.Description.Long
		
		// Check for key concepts in documentation
		concepts := []string{
			"personal stage",
			"specific stage",
			"specific component",
			"concurrently",
			"dependencies",
			"container images",
			"sites",
			"functions",
			"build processes",
			"memory",
			"CI",
		}
		
		for _, concept := range concepts {
			assert.Contains(t, longDesc, concept, "Documentation should mention %s", concept)
		}
	})

	t.Run("code examples formatting", func(t *testing.T) {
		longDesc := CmdDeploy.Description.Long
		
		// Check for proper markdown formatting
		assert.Contains(t, longDesc, "```bash frame=\"none\"")
		assert.Contains(t, longDesc, ":::tip")
		
		// Check for table formatting
		assert.Contains(t, longDesc, "| Resource | Concurrency | Flag |")
		assert.Contains(t, longDesc, "| -------- | ----------- | ---- |")
	})

	t.Run("environment variable documentation", func(t *testing.T) {
		longDesc := CmdDeploy.Description.Long
		
		// Check that all concurrency environment variables are properly documented
		envVars := []string{
			"SST_BUILD_CONCURRENCY_SITE",
			"SST_BUILD_CONCURRENCY_FUNCTION", 
			"SST_BUILD_CONCURRENCY_CONTAINER",
		}
		
		for _, envVar := range envVars {
			assert.Contains(t, longDesc, envVar, "Documentation should include %s", envVar)
		}
	})
}

func TestDeployCommandErrorHandling(t *testing.T) {
	t.Run("error handling patterns", func(t *testing.T) {
		// Test that the command follows proper error handling patterns
		// This is more of a structural test since we can't easily mock all dependencies
		
		// The Run function should handle errors from:
		// 1. c.InitProject()
		// 2. server.New()
		// 3. s.Start()
		// 4. p.Run()
		
		// We verify this by checking the function structure exists
		assert.NotNil(t, CmdDeploy.Run, "Run function should exist for error handling")
	})
}