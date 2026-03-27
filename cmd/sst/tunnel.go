package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/sst/sst/v3/cmd/sst/cli"
	"github.com/sst/sst/v3/cmd/sst/mosaic/dev"
	"github.com/sst/sst/v3/cmd/sst/mosaic/ui"
	"github.com/sst/sst/v3/internal/util"
	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/project"
	"github.com/sst/sst/v3/pkg/server"
	"github.com/sst/sst/v3/pkg/tunnel"
)

var CmdTunnel = &cli.Command{
	Name: "tunnel",
	Description: cli.Description{
		Short: "Start a tunnel",
		Long: strings.Join([]string{
			"Start a tunnel.",
			"",
			"```bash frame=\"none\"",
			"sst tunnel",
			"```",
			"",
			"If your app has a VPC with `bastion` enabled, you can use this to connect to it.",
			"This will forward traffic from the following ranges using either SSH or SSM, depending on your bastion configuration:",
			"- `10.0.4.0/22`",
			"- `10.0.12.0/22`",
			"- `10.0.0.0/22`",
			"- `10.0.8.0/22`",
			"",
			"The tunnel allows your local machine to access resources that are in the VPC.",
			"",
			":::note",
			"The tunnel is only available for apps that have a VPC with `bastion` enabled, or apps that have a Bastion component",
			":::",
			"",
			"If you are running `sst dev`, this tunnel will be started automatically under the",
			"_Tunnel_ tab in the sidebar.",
			"",
			":::tip",
			"This is automatically started when you run `sst dev`.",
			":::",
			"",
			"You can start this manually if you want to connect to a different stage.",
			"",
			"```bash frame=\"none\"",
			"sst tunnel --stage production",
			"```",
			"",
			"This needs a network interface on your local machine. You can create this",
			"with the `sst tunnel install` command.",
			"",
			":::note",
			"When using the Bastion component in SSM mode, the tunnel requires the AWS Session Manager Plugin to be installed.",
			"https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html",
			":::",
		}, "\n"),
	},
	Run: func(c *cli.Cli) error {
		if tunnel.NeedsInstall() {
			return util.NewReadableError(nil, "The sst tunnel needs to be installed or upgraded. Run `sudo sst tunnel install`")
		}

		if tunnel.IsRunning() {
			return util.NewReadableError(nil, "Another tunnel process is already running. Stop it before starting a new one.")
		}

		cfgPath, err := c.Discover()
		if err != nil {
			return err
		}

		slog.Info("starting tunnel")

		var completed *project.CompleteEvent

		stage, err := c.Stage(cfgPath)
		if err != nil {
			return err
		}

		if url, err := server.Discover(cfgPath, stage); err == nil {
			completed, err = dev.Completed(c.Context, url)
			if err != nil {
				return err
			}
		} else {
			proj, err := c.InitProject()
			if err != nil {
				return err
			}
			completed, err = proj.GetCompleted(c.Context)
			if err != nil {
				return err
			}
		}

		if err != nil {
			return err
		}
		if len(completed.Tunnels) == 0 {
			return util.NewReadableError(nil, "No tunnels found for stage "+stage)
		}

		var ssmConfigs []tunnel.SSMConfig
		var sshConfigs []tunnel.SSHConfig
		var allSubnets []string

		for name, tun := range completed.Tunnels {
			mode := tun.Mode

			// backwards compatible for vpc bastion v1
			if mode == "" {
				mode = "ssh"
			}

			if mode == "ssm" {
				if tun.InstanceID == "" {
					slog.Warn("SSM tunnel missing instance ID, skipping", "name", name)
					continue
				}
				if tun.Region == "" {
					slog.Warn("SSM tunnel missing region, skipping", "name", name)
					continue
				}
				ssmConfigs = append(ssmConfigs, tunnel.SSMConfig{
					InstanceID: tun.InstanceID,
					Region:     tun.Region,
					Subnets:    tun.Subnets,
				})
			} else if mode == "ssh" {
				if tun.IP == "" || tun.PrivateKey == "" {
					slog.Warn("SSH tunnel missing IP or private key, skipping", "name", name)
					continue
				}
				sshConfigs = append(sshConfigs, tunnel.SSHConfig{
					Host:       tun.IP,
					Username:   tun.Username,
					PrivateKey: tun.PrivateKey,
					Subnets:    tun.Subnets,
				})
			}

			allSubnets = append(allSubnets, tun.Subnets...)
		}

		if len(ssmConfigs) == 0 && len(sshConfigs) == 0 {
			return util.NewReadableError(nil, "No tunnels found. Make sure you have a bastion deployed.")
		}

		if len(ssmConfigs) > 0 {
			if _, err := exec.LookPath("session-manager-plugin"); err != nil {
				return util.NewReadableError(nil, "AWS Session Manager Plugin is required for SSM tunnels but was not found.\n\nInstall it from: https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html\n\nAlternatively, you can use SSH mode by setting `ssm: false` on your Bastion component.")
			}
		}

		args := []string{
			"-n", "-E",
			tunnel.BINARY_PATH, "tunnel", "start",
			"--subnets", strings.Join(allSubnets, ","),
			"--print-logs",
		}

		if len(ssmConfigs) > 0 {
			ssmJSON, err := json.Marshal(ssmConfigs)
			if err != nil {
				return fmt.Errorf("failed to serialize SSM config: %w", err)
			}
			args = append(args, "--ssm-config", string(ssmJSON))
		}

		if len(sshConfigs) > 0 {
			sshJSON, err := json.Marshal(sshConfigs)
			if err != nil {
				return fmt.Errorf("failed to serialize SSH config: %w", err)
			}
			args = append(args, "--ssh-config", string(sshJSON))
		}

		// run as root
		tunnelCmd := process.CommandContext(
			c.Context,
			"sudo",
			args...,
		)
		tunnelCmd.Env = append(
			os.Environ(),
			"SST_SKIP_LOCAL=true",
			"SST_SKIP_DEPENDENCY_CHECK=true",
			"SST_LOG="+strings.ReplaceAll(os.Getenv("SST_LOG"), ".log", "_sudo.log"),
		)
		tunnelCmd.Stdout = os.Stdout
		slog.Info("starting tunnel", "cmd", tunnelCmd.Args)
		fmt.Println(ui.TEXT_HIGHLIGHT_BOLD.Render("Tunnel"))
		fmt.Println()

		for _, cfg := range ssmConfigs {
			fmt.Print(ui.TEXT_HIGHLIGHT_BOLD.Render("▤"))
			fmt.Println(ui.TEXT_NORMAL.Render("  " + cfg.InstanceID + " (SSM, " + cfg.Region + ")"))
			for _, subnet := range cfg.Subnets {
				fmt.Println(ui.TEXT_DIM.Render("   " + subnet))
			}
			fmt.Println()
		}

		for _, cfg := range sshConfigs {
			fmt.Print(ui.TEXT_HIGHLIGHT_BOLD.Render("▤"))
			fmt.Println(ui.TEXT_NORMAL.Render("  " + cfg.Host + " (SSH)"))
			for _, subnet := range cfg.Subnets {
				fmt.Println(ui.TEXT_DIM.Render("   " + subnet))
			}
			fmt.Println()
		}

		fmt.Println(ui.TEXT_DIM.Render("Waiting for connections..."))
		fmt.Println()
		stderr, _ := tunnelCmd.StderrPipe()
		tunnelCmd.Start()
		output, _ := io.ReadAll(stderr)
		if strings.Contains(string(output), "password is required") {
			return util.NewReadableError(nil, "Make sure you have installed the tunnel with `sudo sst tunnel install`")
		}
		return nil
	},
	Children: []*cli.Command{
		{
			Name: "install",
			Description: cli.Description{
				Short: "Install the tunnel",
				Long: strings.Join([]string{
					"Install the tunnel.",
					"",
					"To be able to create a tunnel, SST needs to create a network interface on your local",

					"",
					"```bash \"sudo\"",
					"sudo sst tunnel install",
					"```",
					"",
					"You only need to run this once on your machine.",
				}, "\n"),
			},
			Run: func(c *cli.Cli) error {
				currentUser, err := user.Current()
				if err != nil {
					return err
				}
				if currentUser.Uid != "0" {
					return util.NewReadableError(nil, "You need to run this command as root")
				}
				err = tunnel.Install()
				if err != nil {
					return err
				}
				ui.Success("Tunnel installed successfully.")
				return nil
			},
		},
		{
			Name: "start",
			Description: cli.Description{
				Short: "Start the tunnel",
				Long: strings.Join([]string{
					"Start the tunnel.",
					"",
					"This will start the tunnel.",
					"",
					"This is required for the tunnel to work.",
				}, "\n"),
			},
			Hidden: true,
			Flags: []cli.Flag{
				{
					Name: "subnets",
					Type: "string",
					Description: cli.Description{
						Short: "The subnets to route through the tunnel",
						Long:  "The subnets to route through the tunnel",
					},
				},
				{
					Name: "ssm-config",
					Type: "string",
					Description: cli.Description{
						Short: "JSON-encoded SSM tunnel configurations",
						Long:  "JSON-encoded SSM tunnel configurations",
					},
				},
				{
					Name: "ssh-config",
					Type: "string",
					Description: cli.Description{
						Short: "JSON-encoded SSH tunnel configurations",
						Long:  "JSON-encoded SSH tunnel configurations",
					},
				},
			},
			Run: func(c *cli.Cli) error {
				subnets := strings.Split(c.String("subnets"), ",")
				ssmJSON := c.String("ssm-config")
				sshJSON := c.String("ssh-config")

				var ssmConfigs []tunnel.SSMConfig
				var sshConfigs []tunnel.SSHConfig

				if ssmJSON != "" {
					if err := json.Unmarshal([]byte(ssmJSON), &ssmConfigs); err != nil {
						return util.NewReadableError(err, "failed to parse SSM configuration")
					}
				}

				if sshJSON != "" {
					if err := json.Unmarshal([]byte(sshJSON), &sshConfigs); err != nil {
						return util.NewReadableError(err, "failed to parse SSH configuration")
					}
				}

				if len(ssmConfigs) == 0 && len(sshConfigs) == 0 {
					return util.NewReadableError(nil, "at least one SSM or SSH tunnel is required")
				}

				slog.Info("starting tunnel", "subnets", subnets, "ssm", len(ssmConfigs), "ssh", len(sshConfigs))
				err := tunnel.Start(subnets...)
				defer tunnel.Stop()
				if err != nil {
					return err
				}
				slog.Info("tunnel started")

				err = tunnel.StartProxy(c.Context, ssmConfigs, sshConfigs)

				if err != nil {
					slog.Error("failed to start tunnel", "error", err)
				}
				return nil
			},
		},
	},
}
