package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pulumi/pulumi/pkg/v3/resource/deploy"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/sst/sst/v3/cmd/sst/cli"
	"github.com/sst/sst/v3/cmd/sst/mosaic/ui"
	"github.com/sst/sst/v3/internal/util"
	"github.com/sst/sst/v3/pkg/id"
	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/project/provider"
	"github.com/sst/sst/v3/pkg/state"
)

var CmdState = &cli.Command{
	Name: "state",
	Description: cli.Description{
		Short: "Manage state of your app",
	},
	Children: []*cli.Command{
		{
			Name: "edit",
			Description: cli.Description{
				Short: "Edit the state of your app",
				Long: strings.Join([]string{
					"Edit the raw state of your app directly.",
					"",
					"This opens your state file in your local editor (`$EDITOR`, or `vim` by default).",
					"When you save and exit, SST pushes those changes back to your backend.",
					"",
					":::danger",
					"This command is dangerous. If you make an invalid change, you can corrupt your state and break deploys.",
					"Only use this if you understand the state format and know exactly what you are changing.",
					"Consider using safer commands like `sst state remove` or `sst state repair` first.",
					":::",
				}, "\n"),
			},
			Run: func(c *cli.Cli) error {
				p, err := c.InitProject()
				if err != nil {
					return err
				}
				defer p.Cleanup()

				update, err := p.Lock("edit")
				if err != nil {
					return util.NewReadableError(err, "Could not lock state")
				}
				defer p.Unlock()
				defer func() {
					update.TimeCompleted = time.Now().UTC().Format(time.RFC3339)
					provider.PutUpdate(p.Backend(), p.App().Name, p.App().Stage, update)
				}()
				workdir, err := p.NewWorkdir(update.ID)
				if err != nil {
					return err
				}
				defer workdir.Cleanup()

				path, err := workdir.Pull()
				if err != nil {
					return util.NewReadableError(err, "Could not pull state")
				}
				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "vim"
				}
				editorArgs := append(strings.Fields(editor), path)
				fmt.Println(editorArgs)
				cmd := process.Command(editorArgs[0], editorArgs[1:]...)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				if err := cmd.Start(); err != nil {
					return util.NewReadableError(err, "Could not start editor")
				}
				if err := cmd.Wait(); err != nil {
					return util.NewReadableError(err, "Editor exited with error")
				}

				return workdir.Push(update.ID)
			},
		},
		{
			Name: "export",
			Flags: []cli.Flag{
				{
					Name: "decrypt",
					Type: "bool",
					Description: cli.Description{
						Short: "Decrypt the state",
						Long:  "Decrypt the state before printing it out.",
					},
				},
			},
			Description: cli.Description{
				Short: "Prints the state of your app",
				Long: strings.Join([]string{
					"Prints the state of your app.",
					"",
					"This pull the state of your app from the cloud provider and then prints it out.",
					"You can write this to a file or view it directly in your terminal.",
					"",
					"This can be run for specific stages as well.",
					"",
					"```bash frame=\"none\"",
					"sst state export --stage production",
					"```",
					"",
					"By default, it runs on your personal stage.",
				}, "\n"),
			},
			Run: func(c *cli.Cli) error {
				p, err := c.InitProject()
				if err != nil {
					return err
				}
				defer p.Cleanup()
				workdir, err := p.NewWorkdir(id.Descending())
				if err != nil {
					return err
				}
				defer workdir.Cleanup()

				_, err = workdir.Pull()
				if err != nil {
					return util.NewReadableError(err, "Could not pull state")
				}
				exported, err := workdir.Export()
				if err != nil {
					return err
				}
				if c.Bool("decrypt") {
					passphrase, err := provider.Passphrase(p.Backend(), p.App().Name, p.App().Stage)
					if err != nil {
						return err
					}
					exported, err = state.Decrypt(c.Context, passphrase, exported)
					if err != nil {
						return err
					}
				}
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(exported)
			},
		},
		{
			Name: "list",
			Description: cli.Description{
				Short: "List all deployed stages",
				Long: strings.Join([]string{
					"Lists all the stages of your app for the current set of credentials.",
					"",
					":::note",
					"This does not list the stages that are deployed in other accounts.",
					":::",
					"",
					"This pulls the state of your app from the cloud provider and then prints out all the stages that are listed in the state.",
				}, "\n"),
			},
			Run: func(c *cli.Cli) error {
				p, err := c.InitProject()
				if err != nil {
					return err
				}
				defer p.Cleanup()
				backend := p.Backend()
				currentStage := p.App().Stage

				stages, err := provider.ListStages(backend, p.App().Name)
				if err != nil {
					return err
				}

				lines, err := provider.Info(backend)
				if err != nil {
					ui.Error("Failed to load provider information")
					return err
				}

				renderKeyValue("App", p.App().Name)

				for _, line := range lines {
					renderKeyValue(line.Key, line.Value)
				}

				if len(stages) == 0 {
					fmt.Println(
						ui.TEXT_NORMAL_BOLD.Render(indent("Stages:")) +
							ui.TEXT_NORMAL.Render(currentStage) + " " + ui.TEXT_WARNING_DIM.Render("(not deployed)"),
					)
					return nil
				}

				currentDeployed := false
				for i, stage := range stages {
					rendered := ui.TEXT_GRAY.Render(stage)
					if stage == currentStage {
						rendered = ui.TEXT_NORMAL.Render(stage)
						currentDeployed = true
					}

					if i == 0 {
						fmt.Println(ui.TEXT_NORMAL_BOLD.Render(indent("Stages:")) + rendered)
						continue
					}

					fmt.Println(indent("") + rendered)
				}

				if !currentDeployed {
					fmt.Println(indent("") + ui.TEXT_NORMAL.Render(currentStage) + " " + ui.TEXT_WARNING_DIM.Render("(not deployed)"))
				}

				return nil
			},
		},
		{
			Name: "remove",
			Args: []cli.Argument{
				{
					Name:     "target",
					Required: true,
					Description: cli.Description{
						Short: "The name of the resource to remove",
						Long:  "The name of the resource to remove.",
					},
				},
			},
			Description: cli.Description{
				Short: "Remove a resource from only the state",
				Long: strings.Join([]string{
					"Removes the reference for the given resource from the state.",
					"",
					":::note",
					"This does not remove the resource itself from your cloud account.",
					":::",
					"",
					"This does not delete the resource from your cloud provider; it only edits",
					"the state of your app.",
					"",
					"```bash frame=\"none\"",
					"sst state remove MyBucket",
					"```",
					"",
					"Here, `MyBucket` is the name of the resource as defined in your `sst.config.ts`.",
					"",
					"```ts title=\"sst.config.ts\"",
					"new sst.aws.Bucket(\"MyBucket\");",
					"```",
					"",
					"This command will:",
					"",
					"1. Find every resource with the given name in the state.",
					"2. Remove each one along with **all of its dependents** — children, explicit",
					"   `dependsOn`, property dependencies, `deletedWith` and `replaceWith`. This",
					"   keeps the state internally consistent.",
					"3. Run a `repair` afterwards to clean up any remaining dangling references.",
					"",
					"Protected resources are not deleted; if any are encountered the command will",
					"abort with an error so you can decide how to handle them.",
					"",
					"You can run this for specific stages as well.",
					"",
					"```bash frame=\"none\"",
					"sst state remove MyBucket --stage production",
					"```",
					"",
					"By default, it runs on your personal stage.",
				}, "\n"),
			},
			Run: func(c *cli.Cli) error {
				p, err := c.InitProject()
				if err != nil {
					return err
				}
				defer p.Cleanup()

				update, err := p.Lock("edit")
				if err != nil {
					return util.NewReadableError(err, "Could not lock state")
				}
				defer p.Unlock()
				defer func() {
					update.TimeCompleted = time.Now().UTC().Format(time.RFC3339)
					provider.PutUpdate(p.Backend(), p.App().Name, p.App().Stage, update)
				}()
				workdir, err := p.NewWorkdir(update.ID)
				if err != nil {
					return err
				}
				defer workdir.Cleanup()

				_, err = workdir.Pull()
				if err != nil {
					return util.NewReadableError(err, "Could not pull state")
				}

				checkpoint, err := workdir.Export()
				if err != nil {
					return util.NewReadableError(err, "Could not export state")
				}

				passphrase, err := provider.Passphrase(p.Backend(), p.App().Name, p.App().Stage)
				if err != nil {
					return util.NewReadableError(err, "Could not load passphrase")
				}

				target := c.Positional(0)
				result, err := state.Remove(c.Context, passphrase, target, checkpoint)
				if err != nil {
					return util.NewReadableError(err, err.Error())
				}
				if err := confirmRemove(result); err != nil {
					return err
				}

				if err := workdir.Import(checkpoint); err != nil {
					return util.NewReadableError(err, "Could not import state")
				}

				if err := workdir.Push(update.ID); err != nil {
					return err
				}
				ui.Success("Resource removed")
				return nil
			},
		},
		{
			Name: "repair",
			Description: cli.Description{
				Short: "Repair the state of your app",
				Long: strings.Join([]string{
					"Repairs the state of your app if it's corrupted.",
					"",
					"Sometimes, if something goes wrong with your app, or if the state was directly",
					"edited, the state can become corrupted. This will cause your `sst deploy` command",
					"to fail with a `snapshot integrity` error.",
					"",
					"This command looks for the following issues and fixes them.",
					"",
					"1. Since the state is a list of resources, if one resource depends on another,",
					"   it needs to be listed after the one it depends on. This command finds resources",
					"   that depend on each other but are not ordered correctly and **reorders them**.",
					"",
					"2. If resource B depends on resource A but resource A is not listed in the state,",
					"   it'll **remove the dangling reference**. This applies to all dependency types:",
					"   parent, dependencies, property dependencies, `deletedWith` and `replaceWith`.",
					"",
					"3. If a child resource has a parent that no longer exists, the child is",
					"   **unparented** (its URN is rewritten and it becomes a top-level resource).",
					"",
					"You can run this for specific stages as well.",
					"",
					"```bash frame=\"none\"",
					"sst state repair --stage production",
					"```",
					"",
					"By default, it runs on your personal stage.",
				}, "\n"),
			},
			Run: func(c *cli.Cli) error {
				p, err := c.InitProject()
				if err != nil {
					return err
				}
				defer p.Cleanup()

				update, err := p.Lock("repair")
				if err != nil {
					return util.NewReadableError(err, "Could not lock state")
				}
				defer p.Unlock()
				defer func() {
					update.TimeCompleted = time.Now().UTC().Format(time.RFC3339)
					provider.PutUpdate(p.Backend(), p.App().Name, p.App().Stage, update)
				}()
				workdir, err := p.NewWorkdir(update.ID)
				if err != nil {
					return err
				}
				defer workdir.Cleanup()

				_, err = workdir.Pull()
				if err != nil {
					return util.NewReadableError(err, "Could not pull state")
				}

				checkpoint, err := workdir.Export()
				if err != nil {
					return util.NewReadableError(err, "Could not export state")
				}

				passphrase, err := provider.Passphrase(p.Backend(), p.App().Name, p.App().Stage)
				if err != nil {
					return util.NewReadableError(err, "Could not load passphrase")
				}

				result, err := state.Repair(c.Context, passphrase, checkpoint)
				if err != nil {
					return util.NewReadableError(err, err.Error())
				}
				if err := confirmRepair(result); err != nil {
					return err
				}

				if err := workdir.Import(checkpoint); err != nil {
					return util.NewReadableError(err, "Could not import state")
				}

				if err := workdir.Push(update.ID); err != nil {
					return err
				}
				ui.Success("State repaired")
				return nil
			},
		},
	},
}

func confirmRepair(result state.RepairResult) error {
	if result.IsEmpty() {
		return util.NewReadableError(nil, "No changes needed")
	}

	if result.Reordered {
		fmt.Println("Reordered resources to satisfy dependency order.")
		fmt.Println()
	}

	if len(result.Pruned) > 0 {
		fmt.Println("Modified:")
		for _, p := range result.Pruned {
			renderPruneResult(p)
		}
	}

	return promptConfirm()
}

func confirmRemove(result state.RemoveResult) error {
	if result.IsEmpty() {
		return util.NewReadableError(nil, "No changes made")
	}

	if len(result.Removed) > 0 {
		fmt.Println("Removed:")
		for _, urn := range result.Removed {
			fmt.Printf("  - %s → %s\n", urn.Type().DisplayName(), urn.Name())
		}
	}

	if len(result.Pruned) > 0 {
		fmt.Println()
		fmt.Println("Modified:")
		for _, p := range result.Pruned {
			renderPruneResult(p)
		}
	}

	return promptConfirm()
}

func renderPruneResult(p deploy.PruneResult) {
	if p.OldURN != p.NewURN {
		fmt.Printf("  - %s → %s (unparented)\n", p.OldURN.Type().DisplayName(), p.OldURN.Name())
	} else {
		fmt.Printf("  - %s → %s\n", p.OldURN.Type().DisplayName(), p.OldURN.Name())
	}
	for _, dep := range p.RemovedDependencies {
		switch dep.Type {
		case resource.ResourceParent:
			fmt.Printf("      removed parent: %s → %s\n", dep.URN.Type().DisplayName(), dep.URN.Name())
		case resource.ResourceDependency:
			fmt.Printf("      removed dependency: %s → %s\n", dep.URN.Type().DisplayName(), dep.URN.Name())
		case resource.ResourcePropertyDependency:
			fmt.Printf("      removed property %q dependency: %s → %s\n", dep.Key, dep.URN.Type().DisplayName(), dep.URN.Name())
		case resource.ResourceDeletedWith:
			fmt.Printf("      removed deletedWith: %s → %s\n", dep.URN.Type().DisplayName(), dep.URN.Name())
		case resource.ResourceReplaceWith:
			fmt.Printf("      removed replaceWith: %s → %s\n", dep.URN.Type().DisplayName(), dep.URN.Name())
		}
	}
}

func promptConfirm() error {
	fmt.Println()
	fmt.Print("Do you want to commit these changes? (Y/n): ")
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		return util.NewReadableError(err, "failed to read user input")
	}
	if strings.ToLower(response) != "y" {
		return util.NewReadableError(nil, "Abandoning changes")
	}
	return nil
}

func indent(key string) string {
	return fmt.Sprintf("%-12s", key)
}

func renderKeyValue(key string, value string) {
	fmt.Println(ui.TEXT_NORMAL_BOLD.Render(indent(key+":")) + ui.TEXT_GRAY.Render(value))
}
