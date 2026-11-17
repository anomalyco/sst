package main

import (
	"strings"

	"github.com/sst/sst/v3/cmd/sst/cli"
	"github.com/sst/sst/v3/cmd/sst/mosaic/ui"
	"github.com/sst/sst/v3/pkg/types"
)

var CmdTypes = &cli.Command{
	Name: "types",
	Description: cli.Description{
		Short: "Generate resource types",
		Long: strings.Join([]string{
			"Generate the types for the linked resources in your app.",
			"",
			"```bash frame=\"none\"",
			"sst types",
			"```",
			"",
			"This writes the `sst-env.d.ts` file that gives you typesafe access to your linked",
			"resources in your code. If you are using a language other than JavaScript, it'll",
			"also generate the equivalent; like the `sst.pyi` file for Python.",
			"",
			"These types are generated automatically as a part of `sst dev` and `sst deploy`.",
			"So you typically don't need to run this. But it's useful when you want the types",
			"without having to run your app.",
			"",
			"For example, in your CI pipeline or right after you pull down a teammate's changes;",
			"where you want to typecheck before deploying.",
			"",
			"Generate the types for a specific stage by passing in the stage name.",
			"",
			"```bash frame=\"none\"",
			"sst types --stage production",
			"```",
			"",
			":::note",
			"The types are based on what's been deployed. So the stage needs to have been",
			"deployed at least once.",
			":::",
		}, "\n"),
	},
	Examples: []cli.Example{
		{
			Content: "sst types --stage production",
			Description: cli.Description{
				Short: "Generate types for production",
			},
		},
	},
	Run: func(c *cli.Cli) error {
		p, err := c.InitProject()
		if err != nil {
			return err
		}
		defer p.Cleanup()

		complete, err := p.GetCompleted(c.Context)
		if err != nil {
			return err
		}

		err = types.Generate(p.PathConfig(), complete.Links, p.App().Types.Ignore)
		if err != nil {
			return err
		}

		ui.Success("Generated types")
		return nil
	},
}
