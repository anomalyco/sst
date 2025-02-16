package main

import (
	"strings"

	"github.com/sst/sst/v3/cmd/sst/cli"
	"github.com/sst/sst/v3/cmd/sst/mosaic/ui"
	"github.com/sst/sst/v3/pkg/bus"
	"github.com/sst/sst/v3/pkg/project"
	"github.com/sst/sst/v3/pkg/server"
	"golang.org/x/sync/errgroup"
)

var CmdDeploy = &cli.Command{
	Name: "deploy",
	Description: cli.Description{
		Short: "Deploy your application",
		Long: strings.Join([]string{
			"Deploy your application. By default, it deploys to your personal stage.",
			"You typically want to deploy it to a specific stage.",
			"",
			"```bash frame=\"none\"",
			"sst deploy --stage production",
			"```",
			"",
			"Optionally, deploy a specific component by passing in the name of the component from your `sst.config.ts`.",
			"",
			"```bash frame=\"none\"",
			"sst deploy --target MyComponent",
			"```",
			"",
			"All the resources are deployed as concurrently as possible, based on their dependencies.",
			"For resources like your container images, sites, and functions; it first builds them and then deploys the generated assets.",
			"",
			":::tip",
			"Configure the concurrency if your CI builds are running out of memory.",
			":::",
			"",
			"Since the build processes for some of these resources take a lot of memory, their concurrency is limited by default.",
			"However, this can be configured.",
			"",
			"| Resource | Concurrency | Flag |",
			"| -------- | ----------- | ---- |",
			"| Sites | 1 | `SST_BUILD_CONCURRENCY_SITE` |",
			"| Functions | 4 | `SST_BUILD_CONCURRENCY_FUNCTION` |",
			"| Containers | 1 | `SST_BUILD_CONCURRENCY_CONTAINER` |",
			"",
			"So only one site is built at a time, 4 functions are built at a time, and only 1 container is built at a time.",
			"",
			"You can set the above environment variables to change this when you run `sst deploy`. This is useful for CI",
			"environments where you want to control this based on how much memory your CI machine has.",
			"",
			"For example, to build a maximum of 2 sites concurrently.",
			"",
			"```bash frame=\"none\"",
			"SST_BUILD_CONCURRENCY_SITE=2 sst deploy",
			"```",
			" Or to configure all these together.",
			"",
			"```bash frame=\"none\"",
			"SST_BUILD_CONCURRENCY_SITE=2 SST_BUILD_CONCURRENCY_CONTAINER=2 SST_BUILD_CONCURRENCY_FUNCTION=8 sst deploy",
			"```",
			"Typically, this command exits when there's an error deploying a resource.",
			"But sometimes you want to be able to `--continue` deploying as many resources as possible;",
			"",
			"```bash frame=\"none\"",
			"sst deploy --continue",
			"```",
			"",
			"This is useful when deploying a new stage with a lot of resources. You want",
			"to be able to deploy as many resources as possible and then come back and",
			"fix the errors.",
			"",
			"The `sst dev` command deploys your resources a little differently. It skips",
			"deploying resources that are going to be run locally. Sometimes you want to",
			"deploy a personal stage without starting `sst dev`.",
			"",
			"```bash frame=\"none\"",
			"sst deploy --dev",
			"```",
			"The `--dev` flag will deploy your resources as if you were running `sst dev`.",
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
			Name: "continue",
			Type: "bool",
			Description: cli.Description{
				Short: "Continue on error",
				Long:  "Continue on error and try to deploy as many resources as possible.",
			},
		},
		{
			Name: "dev",
			Type: "bool",
			Description: cli.Description{
				Short: "Deploy in dev mode",
				Long:  "Deploy resources like `sst dev` would.",
			},
		},
	},
	Examples: []cli.Example{
		{
			Content: "sst deploy --stage production",
			Description: cli.Description{
				Short: "Deploy to production",
			},
		},
	},
	Run: func(c *cli.Cli) error {
		p, err := c.InitProject()
		if err != nil {
			return err
		}
		defer p.Cleanup()

		target := []string{}
		if c.String("target") != "" {
			target = strings.Split(c.String("target"), ",")
		}

		var wg errgroup.Group
		defer wg.Wait()
		out := make(chan interface{})
		defer close(out)
		ui := ui.New(c.Context)
		s, err := server.New()
		if err != nil {
			return err
		}
		wg.Go(func() error {
			defer c.Cancel()
			return s.Start(c.Context, p)
		})
		events := bus.SubscribeAll()
		defer close(events)
		wg.Go(func() error {
			for evt := range events {
				ui.Event(evt)
			}
			return nil
		})
		defer ui.Destroy()
		defer c.Cancel()
		err = p.Run(c.Context, &project.StackInput{
			Command:    "deploy",
			Target:     target,
			Dev:        c.Bool("dev"),
			ServerPort: s.Port,
			Verbose:    c.Bool("verbose"),
			Continue:   c.Bool("continue"),
		})
		if err != nil {
			return err
		}
		return nil
	},
}
