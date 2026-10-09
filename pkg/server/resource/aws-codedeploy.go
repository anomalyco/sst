package resource

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	codedeploytypes "github.com/aws/aws-sdk-go-v2/service/codedeploy/types"
)

type handleDeploymentConflictInput struct {
	Client              *codedeploy.Client
	ApplicationName     string
	DeploymentGroupName string
	OnConflict          string
}

func handleDeploymentConflict(r *AwsResource, input handleDeploymentConflictInput) error {
	existingID, err := findActiveDeployment(r, input.Client, input.ApplicationName, input.DeploymentGroupName)
	if err != nil {
		return err
	}
	if existingID == "" {
		return nil
	}
	switch input.OnConflict {
	case "fail":
		return fmt.Errorf("deployment %s already in progress for deployment group %s", existingID, input.DeploymentGroupName)
	case "cancel":
		return stopDeployment(r, stopDeploymentInput{
			Client:       input.Client,
			DeploymentID: existingID,
			Rollback:     false,
		})
	case "rollback":
		return stopDeployment(r, stopDeploymentInput{
			Client:       input.Client,
			DeploymentID: existingID,
			Rollback:     true,
		})
	}
	return nil
}

type stopDeploymentInput struct {
	Client       *codedeploy.Client
	DeploymentID string
	Rollback     bool
}

func stopDeployment(r *AwsResource, input stopDeploymentInput) error {
	if input.Rollback {
		slog.Info("stopping existing deployment (rollback)", "deploymentId", input.DeploymentID)
	} else {
		slog.Info("stopping existing deployment (keep current state)", "deploymentId", input.DeploymentID)
	}
	_, err := input.Client.StopDeployment(r.context, &codedeploy.StopDeploymentInput{
		DeploymentId:        aws.String(input.DeploymentID),
		AutoRollbackEnabled: aws.Bool(input.Rollback),
	})
	if err != nil {
		return fmt.Errorf("failed to stop deployment %s: %w", input.DeploymentID, err)
	}
	return waitForDeploymentStopped(r, input.Client, input.DeploymentID)
}

func findActiveDeployment(r *AwsResource, client *codedeploy.Client, applicationName, deploymentGroupName string) (string, error) {
	var nextToken *string
	for {
		listResult, err := client.ListDeployments(r.context, &codedeploy.ListDeploymentsInput{
			ApplicationName:     aws.String(applicationName),
			DeploymentGroupName: aws.String(deploymentGroupName),
			IncludeOnlyStatuses: []codedeploytypes.DeploymentStatus{
				codedeploytypes.DeploymentStatusCreated,
				codedeploytypes.DeploymentStatusQueued,
				codedeploytypes.DeploymentStatusInProgress,
			},
			NextToken: nextToken,
		})
		if err != nil {
			return "", fmt.Errorf("failed to list deployments: %w", err)
		}
		if len(listResult.Deployments) > 0 {
			return listResult.Deployments[0], nil
		}
		if listResult.NextToken == nil {
			return "", nil
		}
		nextToken = listResult.NextToken
	}
}

type createDeploymentInput struct {
	Client              *codedeploy.Client
	ApplicationName     string
	DeploymentGroupName string
	AppSpecJSON         string
}

func createDeployment(r *AwsResource, input createDeploymentInput) (string, error) {
	slog.Info("creating CodeDeploy deployment", "application", input.ApplicationName, "group", input.DeploymentGroupName)
	result, err := input.Client.CreateDeployment(r.context, &codedeploy.CreateDeploymentInput{
		ApplicationName:     aws.String(input.ApplicationName),
		DeploymentGroupName: aws.String(input.DeploymentGroupName),
		Revision: &codedeploytypes.RevisionLocation{
			RevisionType: codedeploytypes.RevisionLocationTypeAppSpecContent,
			AppSpecContent: &codedeploytypes.AppSpecContent{
				Content: aws.String(input.AppSpecJSON),
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create deployment: %w", err)
	}

	deploymentID := *result.DeploymentId
	slog.Info("CodeDeploy deployment created", "deploymentId", deploymentID)
	return deploymentID, nil
}

type buildAppSpecInput struct {
	Resources       any
	BeforeTrafficFn string
	AfterTrafficFn  string
}

func buildAppSpec(input buildAppSpecInput) (string, error) {
	type hookEntry struct {
		BeforeAllowTraffic string `json:"BeforeAllowTraffic,omitempty"`
		AfterAllowTraffic  string `json:"AfterAllowTraffic,omitempty"`
	}
	type appSpec struct {
		Version   string      `json:"version"`
		Resources any         `json:"Resources"`
		Hooks     []hookEntry `json:"Hooks,omitempty"`
	}

	spec := appSpec{
		Version:   "0.0",
		Resources: input.Resources,
	}

	if input.BeforeTrafficFn != "" || input.AfterTrafficFn != "" {
		hook := hookEntry{}
		if input.BeforeTrafficFn != "" {
			hook.BeforeAllowTraffic = input.BeforeTrafficFn
		}
		if input.AfterTrafficFn != "" {
			hook.AfterAllowTraffic = input.AfterTrafficFn
		}
		spec.Hooks = []hookEntry{hook}
	}

	data, err := json.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("failed to marshal AppSpec: %w", err)
	}
	return string(data), nil
}

func waitForDeploymentStopped(r *AwsResource, client *codedeploy.Client, deploymentID string) error {
	start := time.Now()
	timeout := 5 * time.Minute

	for {
		result, err := client.GetDeployment(r.context, &codedeploy.GetDeploymentInput{
			DeploymentId: aws.String(deploymentID),
		})
		if err != nil {
			return fmt.Errorf("failed to get deployment %s: %w", deploymentID, err)
		}

		status := result.DeploymentInfo.Status
		if status == codedeploytypes.DeploymentStatusStopped ||
			status == codedeploytypes.DeploymentStatusFailed ||
			status == codedeploytypes.DeploymentStatusSucceeded {
			return nil
		}

		if time.Since(start) > timeout {
			return fmt.Errorf("timed out waiting for deployment %s to stop after 5 minutes", deploymentID)
		}

		time.Sleep(5 * time.Second)
	}
}
