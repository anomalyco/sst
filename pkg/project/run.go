package project

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/sst/sst/v3/pkg/bus"
	"github.com/sst/sst/v3/pkg/flag"
	"github.com/sst/sst/v3/pkg/global"
	"github.com/sst/sst/v3/pkg/id"
	"github.com/sst/sst/v3/pkg/js"
	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/project/provider"
	"github.com/sst/sst/v3/pkg/telemetry"
	"github.com/sst/sst/v3/pkg/types"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
)

func (p *Project) Run(ctx context.Context, input *StackInput) error {
	// if flag.SST_EXPERIMENTAL {
	// 	slog.Info("using next run system")
	// }
	// return p.RunOld(ctx, input)
	return p.RunNext(ctx, input)
}

func (p *Project) RunNext(ctx context.Context, input *StackInput) error {
	log := slog.Default().With("service", "project.run")
	log.Info("running stack command", "cmd", input.Command)

	if p.app.Protect && input.Command == "remove" {
		return ErrProtectedStage
	}

	bus.Publish(&StackCommandEvent{
		App:     p.app.Name,
		Stage:   p.app.Stage,
		Config:  p.PathConfig(),
		Command: input.Command,
		Version: p.Version(),
	})

	update := &provider.Update{
		ID: id.Descending(),
	}
	var err error
	if input.Command != "diff" {
		update, err = p.Lock(input.Command)
		if err != nil {
			if err == provider.ErrLockExists {
				bus.Publish(&ConcurrentUpdateEvent{})
			}
			return err
		}
		log = log.With("updateID", update.ID)
		defer p.Unlock()
	}

	workdir, err := p.NewWorkdir(update.ID)
	if err != nil {
		return err
	}
	defer workdir.Cleanup()

	passphrase, err := provider.Passphrase(p.home, p.app.Name, p.app.Stage)
	if err != nil {
		return err
	}

	outfile := filepath.Join(p.PathPlatformDir(), fmt.Sprintf("sst.config.%v.mjs", time.Now().UnixMilli()))
	os.WriteFile(
		filepath.Join(workdir.path, "Pulumi.yaml"),
		[]byte("name: "+p.app.Name+"\nruntime: nodejs\nmain: "+outfile+"\n"),
		0644,
	)
	pulumiStdout, err := os.Create(p.PathLog("pulumi"))
	if err != nil {
		return err
	}
	defer pulumiStdout.Close()
	pulumiStderr, err := os.Create(p.PathLog("pulumi.err"))
	if err != nil {
		return err
	}
	defer pulumiStderr.Close()
	_, err = workdir.Pull()
	if err != nil {
		if errors.Is(err, provider.ErrStateNotFound) {
			if input.Command != "deploy" {
				return ErrStageNotFound
			}
			cmd := process.Command(global.PulumiPath(), "stack", "init", "organization/"+p.app.Name+"/"+p.app.Stage)
			cmd.Stdout = pulumiStdout
			cmd.Stderr = pulumiStderr
			cmd.Dir = workdir.path
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env,
				"PULUMI_BACKEND_URL=file://"+workdir.Backend(),
				"PULUMI_CONFIG_PASSPHRASE="+passphrase,
			)
			err := cmd.Run()
			if err != nil {
				return err
			}

		} else {
			return err
		}
	}

	completed, err := getCompletedEvent(ctx, passphrase, workdir)
	if err != nil {
		bus.Publish(&BuildFailedEvent{
			Error: err.Error(),
		})
		log.Info("state file might be corrupted", "err", err)
		return err
	}
	completed.Finished = true
	completed.Old = true
	bus.Publish(completed)
	log.Info("got previous deployment")

	cli := map[string]interface{}{
		"command": input.Command,
		"dev":     input.Dev,
		"paths": map[string]string{
			"home":     global.ConfigDir(),
			"root":     p.PathRoot(),
			"work":     p.PathWorkingDir(),
			"platform": p.PathPlatformDir(),
		},
		"state": map[string]interface{}{
			"version": completed.Versions,
		},
	}
	cliBytes, err := json.Marshal(cli)
	if err != nil {
		return err
	}
	appBytes, err := json.Marshal(p.app)
	if err != nil {
		return err
	}

	providerShim := []string{}
	for _, entry := range p.lock {
		providerShim = append(providerShim, fmt.Sprintf("import * as %s from \"%s\";", entry.Alias, entry.Package))
	}
	providerShim = append(providerShim, fmt.Sprintf("import * as sst from \"%s\";", path.Join(p.PathPlatformDir(), "src/components")))

	buildResult, err := js.Build(js.EvalOptions{
		Dir:     p.PathRoot(),
		Outfile: outfile,
		Define: map[string]string{
			"$app": string(appBytes),
			"$cli": string(cliBytes),
			"$dev": fmt.Sprintf("%v", input.Dev),
		},
		Inject:  []string{filepath.Join(p.PathWorkingDir(), "platform/src/shim/run.js")},
		Globals: strings.Join(providerShim, "\n"),
		Code: fmt.Sprintf(`
      import { run } from "%v";
      import mod from "%v/sst.config.ts";
      const result = await run(mod.run);
      export default result;
    `,
			path.Join(p.PathWorkingDir(), "platform/src/auto/run.ts"),
			p.PathRoot(),
		),
	})
	if err != nil {
		bus.Publish(&BuildFailedEvent{
			Error: err.Error(),
		})
		log.Error("failed to build sst.config.ts", "err", err)
		return err
	}
	log.Info("built sst.config.ts", "to", outfile)
	if !flag.SST_NO_CLEANUP {
		defer js.Cleanup(buildResult)
	}

	// disable for now until we hash env too
	if input.SkipHash != "" && buildResult.OutputFiles[0].Hash == input.SkipHash && false {
		bus.Publish(&SkipEvent{})
		return nil
	}

	var meta = js.Metafile{}
	err = json.Unmarshal([]byte(buildResult.Metafile), &meta)
	if err != nil {
		return err
	}
	files := []string{}

	for key := range meta.Inputs {
		absPath, err := filepath.Abs(key)
		if err != nil {
			continue
		}
		files = append(files, absPath)
	}
	bus.Publish(&BuildSuccessEvent{
		Files: files,
		Hash:  buildResult.OutputFiles[0].Hash,
	})
	log.Info("tracked files")

	secrets := map[string]string{}
	fallback := map[string]string{}

	wg := errgroup.Group{}

	wg.Go(func() error {
		secrets, err = provider.GetSecrets(p.home, p.app.Name, p.app.Stage)
		if err != nil {
			return ErrPassphraseInvalid
		}
		return nil
	})

	wg.Go(func() error {
		fallback, err = provider.GetSecrets(p.home, p.app.Name, "")
		if err != nil {
			return ErrPassphraseInvalid
		}
		return nil
	})

	if err := wg.Wait(); err != nil {
		return err
	}

	env := os.Environ()
	for key, value := range p.Env() {
		env = append(env, fmt.Sprintf("%v=%v", key, value))
	}
	for key, value := range fallback {
		env = append(env, fmt.Sprintf("SST_SECRET_%v=%v", key, value))
	}
	for key, value := range secrets {
		env = append(env, fmt.Sprintf("SST_SECRET_%v=%v", key, value))
	}
	env = append(env,
		"PULUMI_CONFIG_PASSPHRASE="+passphrase,
		"PULUMI_SKIP_UPDATE_CHECK=true",
		"PULUMI_BACKEND_URL=file://"+workdir.Backend(),
		"PULUMI_DEBUG_COMMANDS=true",
		// "PULUMI_DISABLE_AUTOMATIC_PLUGIN_ACQUISITION=true",
		"NODE_OPTIONS=--enable-source-maps --no-deprecation",
		"PULUMI_HOME="+global.ConfigDir(),
	)
	if input.ServerPort != 0 {
		env = append(env, "SST_SERVER=http://127.0.0.1:"+fmt.Sprint(input.ServerPort))
	}
	pulumiPath := flag.SST_PULUMI_PATH
	if pulumiPath == "" {
		pulumiPath = filepath.Join(global.BinPath(), "..")
	}

	eventlogPath := workdir.EventLogPath()
	eventlog, err := os.OpenFile(eventlogPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer eventlog.Close()

	args := []string{
		"--stack", fmt.Sprintf("organization/%v/%v", p.app.Name, p.app.Stage),
		"--non-interactive",
		"--event-log", eventlogPath,
	}

	if input.Command == "deploy" || input.Command == "diff" {
		for provider, opts := range p.app.Providers {
			for key, value := range opts.(map[string]interface{}) {
				switch v := value.(type) {
				case map[string]interface{}:
					bytes, err := json.Marshal(v)
					if err != nil {
						return err
					}
					args = append(args, "--config", fmt.Sprintf("%v:%v=%v", provider, key, string(bytes)))
				case []interface{}:
					bytes, err := json.Marshal(v)
					if err != nil {
						return err
					}
					args = append(args, "--config", fmt.Sprintf("%v:%v=%v", provider, key, string(bytes)))
				case string:
					args = append(args, "--config", fmt.Sprintf("%v:%v=%v", provider, key, v))
				}
			}
		}
	}

	switch input.Command {
	case "diff":
		args = append([]string{"preview"}, args...)
	case "refresh":
		args = append([]string{"refresh", "--yes"}, args...)
	case "deploy":
		args = append([]string{"up", "--yes", "-f"}, args...)
	case "remove":
		args = append([]string{"destroy", "--yes", "-f"}, args...)
	}

	if input.Target != nil {
		for _, item := range input.Target {
			index := slices.IndexFunc(completed.Resources, func(res apitype.ResourceV3) bool {
				return res.URN.Name() == item
			})
			if index == -1 {
				return fmt.Errorf("Target not found: %v", item)
			}
			args = append(args, "--target", string(completed.Resources[index].URN))
		}
		if len(input.Target) > 0 {
			args = append(args, "--target-dependents")
		}
	}

	cmd := process.Command(filepath.Join(pulumiPath, "bin/pulumi"), args...)
	process.Detach(cmd)
	cmd.Env = env
	cmd.Stdout = pulumiStdout
	cmd.Stderr = pulumiStderr
	cmd.Dir = workdir.Backend()
	log.Info("starting pulumi", "args", cmd.Args)

	errors := []Error{}
	finished := false
	importDiffs := map[string][]ImportDiff{}

	partial := make(chan int, 1000)
	partialContext, partialCancel := context.WithCancel(ctx)
	defer partialCancel()
	partialDone := make(chan error)
	go func() {
		if input.Command == "diff" {
			return
		}
		for {
			select {
			case <-partialContext.Done():
				partialDone <- nil
				return
			case <-partial:
				workdir.PushPartial(update.ID)
			case <-time.After(time.Second * 5):
				workdir.PushPartial(update.ID)
				continue
			}
		}
	}()

	err = cmd.Start()
	if err != nil {
		return err
	}
	exited := make(chan struct{})
	go func() {
		cmd.Wait()
		log.Info("pulumi exited", "err", err)
		close(exited)
	}()

	go func() {
		select {
		case <-exited:
			return
		case <-ctx.Done():
			if cmd.Process != nil {
				log.Info("sending interrupt")
				err := cmd.Process.Signal(syscall.SIGINT)
				if err != nil {
					log.Error("failed to send interrupt", "err", err)
				}
			}
		}
	}()

	reader := bufio.NewReader(eventlog)

	eofs := 0
loop:
	for {
		bytes, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				select {
				case <-exited:
					log.Info("eof and exited", "eofs", eofs)
					eofs++
					if eofs < 2 {
						continue
					}
					log.Info("breaking out of tail loop")
					break loop
				case <-time.After(time.Millisecond * 100):
					continue
				}
			}
			continue
		}

		var event events.EngineEvent
		err = json.Unmarshal(bytes, &event)
		if err != nil {
			log.Error("failed to unmarshal event", "err", err)
			continue
		}

		if event.DiagnosticEvent != nil && event.DiagnosticEvent.Severity == "error" {
			if strings.HasPrefix(event.DiagnosticEvent.Message, "update failed") || strings.Contains(event.DiagnosticEvent.Message, "failed to register new resource") {
				continue
			}

			// check if the error is a common error
			help := []string{}
			for _, commonError := range CommonErrors {
				if strings.Contains(event.DiagnosticEvent.Message, commonError.Message) {
					help = append(help, commonError.Short...)
				}
			}

			exists := false
			if event.DiagnosticEvent.URN != "" {
				for _, item := range errors {
					if item.URN == event.DiagnosticEvent.URN {
						exists = true
						break
					}
				}
			}

			if exists {
				continue
			}

			if !exists {
				errors = append(errors, Error{
					Message: strings.TrimSpace(event.DiagnosticEvent.Message),
					URN:     event.DiagnosticEvent.URN,
					Help:    help,
				})
				log.Info("telemetry tracking error")
				telemetry.Track("cli.resource.error", map[string]interface{}{
					"error": event.DiagnosticEvent.Message,
					"urn":   event.DiagnosticEvent.URN,
				})
			}
		}

		if event.ResOpFailedEvent != nil {
			if event.ResOpFailedEvent.Metadata.Op == apitype.OpImport {
				for _, name := range event.ResOpFailedEvent.Metadata.Diffs {
					old := event.ResOpFailedEvent.Metadata.Old.Inputs[name]
					next := event.ResOpFailedEvent.Metadata.New.Inputs[name]
					diffs, ok := importDiffs[event.ResOpFailedEvent.Metadata.URN]
					if !ok {
						diffs = []ImportDiff{}
					}
					importDiffs[event.ResOpFailedEvent.Metadata.URN] = append(diffs, ImportDiff{
						URN:   event.ResOpFailedEvent.Metadata.URN,
						Input: name,
						Old:   old,
						New:   next,
					})
				}
			}
		}

		if event.ResOutputsEvent != nil || event.CancelEvent != nil || event.SummaryEvent != nil {
			partial <- 1
		}

		for _, field := range getNotNilFields(event) {
			bus.Publish(field)
		}

		if event.SummaryEvent != nil {
			finished = true
		}
	}

	log.Info("parsing state")
	complete, err := getCompletedEvent(context.Background(), passphrase, workdir)
	if err != nil {
		return err
	}
	complete.UpdateID = update.ID
	complete.Finished = finished
	complete.Errors = errors
	complete.ImportDiffs = importDiffs
	types.Generate(p.PathConfig(), complete.Links)
	defer bus.Publish(complete)

	if input.Command != "diff" {
		log.Info("canceling partial")
		partialCancel()
		log.Info("waiting for partial to exit")
		<-partialDone

		err = workdir.Push(update.ID)
		if err != nil {
			return err
		}
	}

	outputsFilePath := filepath.Join(p.PathWorkingDir(), "outputs.json")
	outputsFile, _ := os.Create(outputsFilePath)
	defer outputsFile.Close()
	json.NewEncoder(outputsFile).Encode(complete.Outputs)

	if input.Command != "diff " {
		update.TimeCompleted = time.Now().Format(time.RFC3339)
		for _, err := range errors {
			update.Errors = append(update.Errors, provider.SummaryError{
				URN:     err.URN,
				Message: err.Message,
			})
		}
		err = provider.PutUpdate(p.home, p.app.Name, p.app.Stage, update)
		if err != nil {
			return err
		}
	}

	log.Info("done running stack command")
	if cmd.ProcessState.ExitCode() > 0 {
		return ErrStackRunFailed
	}
	return nil
}
