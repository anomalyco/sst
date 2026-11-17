package resource

import (
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	codedeploytypes "github.com/aws/aws-sdk-go-v2/service/codedeploy/types"
)

type CodeDeployDeploymentWaiter struct {
	*AwsResource
}

type CodeDeployDeploymentWaiterInputs struct {
	DeploymentID string `json:"deploymentId"`
	Wait         bool   `json:"wait"`
	Trigger      string `json:"trigger"`
}

type CodeDeployDeploymentWaiterOutputs struct {
	DeploymentID string `json:"deploymentId"`
	Status       string `json:"status"`
}

func (r *CodeDeployDeploymentWaiter) Create(input *CodeDeployDeploymentWaiterInputs, output *CreateResult[CodeDeployDeploymentWaiterOutputs]) error {
	outs := r.handle(input)
	*output = CreateResult[CodeDeployDeploymentWaiterOutputs]{
		ID:   "waiter",
		Outs: outs,
	}
	return nil
}

// Update is called on every deploy because the TypeScript side passes a
// changing "trigger" input (e.g. Date.now()). We ignore the trigger and only
// compare deploymentId to decide whether there's a real new deployment to
// wait on.
//
// This solves a key problem: after a failed deployment, the waiter's status
// output is "Failed". If we didn't force Update to run, re-deploying with no
// code changes would leave the stale "Failed" status in Pulumi state, causing
// any downstream .apply() check to keep throwing. By always running Update
// and returning "Skipped" when deploymentId hasn't changed, we reset the
// status so the next deploy succeeds cleanly.
func (r *CodeDeployDeploymentWaiter) Update(input *UpdateInput[CodeDeployDeploymentWaiterInputs, CodeDeployDeploymentWaiterOutputs], output *UpdateResult[CodeDeployDeploymentWaiterOutputs]) error {
	if input.News.DeploymentID == input.Olds.DeploymentID {
		slog.Info("deployment ID unchanged, skipping wait", "deploymentId", input.News.DeploymentID)
		*output = UpdateResult[CodeDeployDeploymentWaiterOutputs]{
			Outs: CodeDeployDeploymentWaiterOutputs{
				DeploymentID: input.News.DeploymentID,
				Status:       "Skipped",
			},
		}
		return nil
	}
	outs := r.handle(&input.News)
	*output = UpdateResult[CodeDeployDeploymentWaiterOutputs]{
		Outs: outs,
	}
	return nil
}

// handle polls the deployment status until it reaches a terminal state.
//
// This function never returns an error. If it did, Pulumi would not persist
// the waiter's outputs. On the next deploy, the waiter's old deployment ID
// would be stale, causing it to re-check a dead deployment and fail again —
// even if no code changed. By always returning successfully, we ensure the
// deployment ID is saved so the Update handler can skip unchanged deployments.
//
// Failure status is surfaced to the user on the TypeScript side by checking
// the status output.
func (r *CodeDeployDeploymentWaiter) handle(input *CodeDeployDeploymentWaiterInputs) CodeDeployDeploymentWaiterOutputs {
	out := func(status string) CodeDeployDeploymentWaiterOutputs {
		return CodeDeployDeploymentWaiterOutputs{
			DeploymentID: input.DeploymentID,
			Status:       status,
		}
	}

	if !input.Wait || input.DeploymentID == "" {
		return out("Skipped")
	}

	cfg, err := r.config()
	if err != nil {
		slog.Error("failed to get AWS config", "error", err)
		return out("Error")
	}
	client := codedeploy.NewFromConfig(cfg)

	start := time.Now()
	timeout := 60 * time.Minute

	for {
		result, err := client.GetDeployment(r.context, &codedeploy.GetDeploymentInput{
			DeploymentId: aws.String(input.DeploymentID),
		})
		if err != nil {
			slog.Error("failed to get deployment", "deploymentId", input.DeploymentID, "error", err)
			return out("Error")
		}

		if terminal, status := checkDeploymentStatus(result.DeploymentInfo, input.DeploymentID); terminal {
			return out(status)
		}

		if time.Since(start) > timeout {
			slog.Warn("CodeDeploy deployment timed out", "deploymentId", input.DeploymentID)
			return out("TimedOut")
		}

		time.Sleep(10 * time.Second)
	}
}

func checkDeploymentStatus(info *codedeploytypes.DeploymentInfo, deploymentID string) (terminal bool, status string) {
	switch info.Status {
	case codedeploytypes.DeploymentStatusSucceeded:
		slog.Info("CodeDeploy deployment succeeded", "deploymentId", deploymentID)
		return true, string(info.Status)
	case codedeploytypes.DeploymentStatusFailed:
		errMsg := "unknown error"
		if info.ErrorInformation != nil && info.ErrorInformation.Message != nil {
			errMsg = *info.ErrorInformation.Message
		}
		slog.Warn("CodeDeploy deployment failed", "deploymentId", deploymentID, "error", errMsg)
		return true, string(info.Status)
	case codedeploytypes.DeploymentStatusStopped:
		slog.Warn("CodeDeploy deployment was stopped", "deploymentId", deploymentID)
		return true, string(info.Status)
	}
	return false, ""
}
