package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/sst/sst/v3/cmd/sst/cli"
	"github.com/sst/sst/v3/cmd/sst/mosaic/ui"
	"github.com/sst/sst/v3/pkg/bus"
	"github.com/sst/sst/v3/pkg/project"
	"github.com/sst/sst/v3/pkg/server"
	"github.com/yalp/jsonpath"
	"golang.org/x/sync/errgroup"
)

var CmdDiff = &cli.Command{
	Name: "diff",
	Description: cli.Description{
		Short: "See what changes will be made",
		Long: strings.Join([]string{
			"Builds your app to see what changes will be made when you deploy it.",
			"",
			"It displays a list of resources that will be created, updated, or deleted.",
			"For each of these resources, it'll also show the properties that are changing.",
			"",
			":::tip",
			"Run a `sst diff` to see what changes will be made when you deploy your app.",
			":::",
			"",
			"This is useful for cases when you pull some changes from a teammate and want to",
			"see what will be deployed; before doing the actual deploy.",
			"",
			"Optionally, you can diff a specific component by passing in the name of the component from your `sst.config.ts`.",
			"",
			"```bash frame=\"none\"",
			"sst diff --target MyComponent",
			"```",
			"",
			"Alternatively, exclude a specific component from the diff.",
			"",
			"```bash frame=\"none\"",
			"sst diff --exclude MyComponent",
			"```",
			"",
			"By default, this compares to the last deploy of the given stage as it would be",
			"deployed using `sst deploy`. But if you are working in dev mode using `sst dev`,",
			"you can use the `--dev` flag.",
			"",
			"```bash frame=\"none\"",
			"sst diff --dev",
			"```",
			"",
			"This is useful because in dev mode, you app is deployed a little differently.",
		}, "\n"),
	},
	Flags: []cli.Flag{
		{
			Name: "target",
			Description: cli.Description{
				Short: "Run it only for a component",
				Long:  "Only run it for the given component.",
			},
		},
		{
			Name: "exclude",
			Type: "string",
			Description: cli.Description{
				Short: "Exclude a component",
				Long:  "Exclude the specified component from the operation.",
			},
		},
		{
			Name: "dev",
			Type: "bool",
			Description: cli.Description{
				Short: "Compare to sst dev",
				Long: strings.Join([]string{
					"Compare to the dev version of this stage.",
				}, "\n"),
			},
		},
		{
			Name: "policy",
			Type: "string",
			Description: cli.Description{
				Short: "Path to policy pack",
				Long:  "Run policy pack validation against the preview changes.",
			},
		},
		{
			Name: "json",
			Type: "bool",
			Description: cli.Description{
				Short: "Output as JSON",
				Long:  "Output the diff result as JSON to stdout. Useful for CI pipelines and scripting.",
			},
		},
	},
	Examples: []cli.Example{
		{
			Content: "sst diff --stage production",
			Description: cli.Description{
				Short: "See changes to production",
			},
		},
		{
			Content: "sst diff --stage production --policy ./policies/production",
			Description: cli.Description{
				Short: "See changes to production with policy validation",
			},
		},
		{
			Content: "sst diff --json",
			Description: cli.Description{
				Short: "Output changes as JSON",
			},
		},
	},
	Run: func(c *cli.Cli) error {
		jsonOutput := c.Bool("json")

		p, err := c.InitProject()
		if err != nil {
			return err
		}
		defer p.Cleanup()

		target := []string{}
		if c.String("target") != "" {
			target = strings.Split(c.String("target"), ",")
		}

		exclude := []string{}
		if c.String("exclude") != "" {
			exclude = strings.Split(c.String("exclude"), ",")
		}

		var wg errgroup.Group
		outputs := []*apitype.ResOutputsEvent{}
		uiOptions := []ui.Option{}
		if jsonOutput {
			// Keep stdout machine-readable when attached to a TTY.
			uiOptions = append(uiOptions, ui.WithSilent)
		}
		u := ui.New(c.Context, uiOptions...)
		s, err := server.New()
		if err != nil {
			return err
		}
		wg.Go(func() error {
			defer c.Cancel()
			return s.Start(c.Context, p)
		})

		events := bus.SubscribeAll()
		wg.Go(func() error {
			for evt := range events {
				if !jsonOutput {
					u.Event(evt)
				}
				switch evt := evt.(type) {
				case *apitype.ResOutputsEvent:
					outputs = append(outputs, evt)
				}
			}
			return nil
		})
		defer u.Destroy()
		err = p.Run(c.Context, &project.StackInput{
			Command:    "diff",
			ServerPort: s.Port,
			Dev:        c.Bool("dev"),
			Target:     target,
			Exclude:    exclude,
			Verbose:    c.Bool("verbose"),
			PolicyPath: c.String("policy"),
		})
		bus.Unsubscribe(events)
		close(events)
		c.Cancel()
		if waitErr := wg.Wait(); waitErr != nil && err == nil {
			err = waitErr
		}

		if jsonOutput {
			if jsonErr := renderDiffJSON(outputs); jsonErr != nil {
				return jsonErr
			}
			return err
		}
		if err != nil {
			return err
		}
		return renderDiffText(outputs, u)
	},
}

func textDiffIcon(op apitype.OpType) string {
	switch op {
	case apitype.OpImport, apitype.OpReplace, apitype.OpCreate:
		return ui.TEXT_SUCCESS_BOLD.Render("+")
	case apitype.OpDelete:
		return ui.TEXT_DANGER_BOLD.Render("-")
	case apitype.OpUpdate:
		return ui.TEXT_WARNING_BOLD.Render("*")
	default:
		return ""
	}
}

func renderDiffJSON(outputs []*apitype.ResOutputsEvent) error {
	filtered := make([]apitype.StepEventMetadata, 0, len(outputs))
	for _, output := range outputs {
		if output.Metadata.Op == apitype.OpSame {
			continue
		}
		filtered = append(filtered, output.Metadata)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(filtered)
}

func renderDiffText(outputs []*apitype.ResOutputsEvent, u *ui.UI) error {
	rendered := make([]*apitype.ResOutputsEvent, 0, len(outputs))
	for _, output := range outputs {
		if textDiffIcon(output.Metadata.Op) == "" {
			continue
		}
		rendered = append(rendered, output)
	}
	if len(rendered) == 0 {
		fmt.Println(
			ui.TEXT_HIGHLIGHT_BOLD.Render("➜"),
			ui.TEXT_NORMAL_BOLD.Render(" No changes"),
		)
		fmt.Println()
		return nil
	}
	for _, output := range rendered {
		icon := textDiffIcon(output.Metadata.Op)
		fmt.Println(icon, "", ui.TEXT_NORMAL_BOLD.Render(u.FormatURN(output.Metadata.URN)))
		sorted := make([]string, 0, len(output.Metadata.DetailedDiff))
		for path := range output.Metadata.DetailedDiff {
			sorted = append(sorted, path)
		}
		sort.Strings(sorted)
		for _, path := range sorted {
			diff := output.Metadata.DetailedDiff[path]
			label := ""
			if diff.Kind == apitype.DiffUpdate {
				label = ui.TEXT_WARNING_BOLD.Render("*")
			}
			if diff.Kind == apitype.DiffDelete {
				label = ui.TEXT_DANGER_BOLD.Render("-")
			}
			if diff.Kind == apitype.DiffAdd {
				label = ui.TEXT_SUCCESS_BOLD.Render("+")
			}
			if diff.Kind == apitype.DiffAddReplace {
				label = ui.TEXT_SUCCESS_BOLD.Render("+")
			}
			if diff.Kind == apitype.DiffUpdateReplace {
				label = ui.TEXT_WARNING_BOLD.Render("*")
			}
			if diff.Kind == apitype.DiffDeleteReplace {
				label = ui.TEXT_DANGER_BOLD.Render("-")
			}
			fmt.Print("   ", label+" ", strings.TrimSpace(path))
			value, _ := jsonpath.Read(output.Metadata.New.Outputs, "$."+path)
			if path == "__provider" {
				value = "code changed"
			}
			if value != nil {
				formatted := ""
				switch value.(type) {
				case string:
					formatted = value.(string)
				default:
					bytes, _ := json.MarshalIndent(value, "", "  ")
					formatted = string(bytes)
				}
				lines := strings.Split(string(formatted), "\n")
				fmt.Print(" = ")
				for index, line := range lines {
					if index > 0 {
						fmt.Print("     ")
					}
					fmt.Print(ui.TEXT_DIM.Render(line) + "\n")
				}
			} else {
				fmt.Println()
			}
		}
		fmt.Println()
	}
	return nil
}
