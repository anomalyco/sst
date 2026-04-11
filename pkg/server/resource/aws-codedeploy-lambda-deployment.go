package resource

import (
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codedeploy"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

type CodeDeployLambdaDeployment struct {
	*AwsResource
}

type CodeDeployLambdaDeploymentInputs struct {
	ApplicationName     string `json:"applicationName"`
	DeploymentGroupName string `json:"deploymentGroupName"`
	FunctionName        string `json:"functionName"`
	AliasName           string `json:"aliasName"`
	TargetVersion       string `json:"targetVersion"`
	OnConflict          string `json:"onConflict"`
	BeforeTrafficFnArn  string `json:"beforeTrafficFnArn,omitempty"`
	AfterTrafficFnArn   string `json:"afterTrafficFnArn,omitempty"`
}

type CodeDeployLambdaDeploymentOutputs struct {
	DeploymentID  string `json:"deploymentId"`
	TargetVersion string `json:"targetVersion"`
}

func (r *CodeDeployLambdaDeployment) Create(input *CodeDeployLambdaDeploymentInputs, output *CreateResult[CodeDeployLambdaDeploymentOutputs]) error {
	outs, err := r.handle(input)
	if err != nil {
		return err
	}
	*output = CreateResult[CodeDeployLambdaDeploymentOutputs]{
		ID:   "deployment",
		Outs: outs,
	}
	return nil
}

func (r *CodeDeployLambdaDeployment) Update(input *UpdateInput[CodeDeployLambdaDeploymentInputs, CodeDeployLambdaDeploymentOutputs], output *UpdateResult[CodeDeployLambdaDeploymentOutputs]) error {
	if input.News.TargetVersion != "" && input.Olds.TargetVersion == input.News.TargetVersion {
		slog.Info("target version unchanged, skipping deployment", "version", input.News.TargetVersion)
		*output = UpdateResult[CodeDeployLambdaDeploymentOutputs]{
			Outs: input.Olds,
		}
		return nil
	}
	outs, err := r.handle(&input.News)
	if err != nil {
		return err
	}
	*output = UpdateResult[CodeDeployLambdaDeploymentOutputs]{
		Outs: outs,
	}
	return nil
}

func (r *CodeDeployLambdaDeployment) handle(input *CodeDeployLambdaDeploymentInputs) (CodeDeployLambdaDeploymentOutputs, error) {
	cfg, err := r.config()
	if err != nil {
		return CodeDeployLambdaDeploymentOutputs{}, err
	}
	cdClient := codedeploy.NewFromConfig(cfg)

	if err := handleDeploymentConflict(r.AwsResource, handleDeploymentConflictInput{
		Client:              cdClient,
		ApplicationName:     input.ApplicationName,
		DeploymentGroupName: input.DeploymentGroupName,
		OnConflict:          input.OnConflict,
	}); err != nil {
		return CodeDeployLambdaDeploymentOutputs{}, err
	}

	appSpecJSON, err := r.buildAppSpec(input, cfg)
	if err != nil {
		return CodeDeployLambdaDeploymentOutputs{}, err
	}

	if appSpecJSON == "" {
		slog.Info("skipping CodeDeploy deployment (no version change)")
		return CodeDeployLambdaDeploymentOutputs{
			DeploymentID:  "",
			TargetVersion: input.TargetVersion,
		}, nil
	}

	deploymentID, err := createDeployment(r.AwsResource, createDeploymentInput{
		Client:              cdClient,
		ApplicationName:     input.ApplicationName,
		DeploymentGroupName: input.DeploymentGroupName,
		AppSpecJSON:         appSpecJSON,
	})
	if err != nil {
		return CodeDeployLambdaDeploymentOutputs{}, err
	}

	return CodeDeployLambdaDeploymentOutputs{
		DeploymentID:  deploymentID,
		TargetVersion: input.TargetVersion,
	}, nil
}

func (r *CodeDeployLambdaDeployment) buildAppSpec(input *CodeDeployLambdaDeploymentInputs, cfg aws.Config) (string, error) {
	lambdaClient := lambda.NewFromConfig(cfg)

	aliasResult, err := lambdaClient.GetAlias(r.context, &lambda.GetAliasInput{
		FunctionName: aws.String(input.FunctionName),
		Name:         aws.String(input.AliasName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get alias %s for function %s: %w", input.AliasName, input.FunctionName, err)
	}
	currentVersion := *aliasResult.FunctionVersion

	if currentVersion == input.TargetVersion {
		return "", nil
	}

	slog.Info("CodeDeploy Lambda version shift",
		"function", input.FunctionName,
		"from", currentVersion,
		"to", input.TargetVersion,
	)

	type properties struct {
		Name           string `json:"Name"`
		Alias          string `json:"Alias"`
		CurrentVersion string `json:"CurrentVersion"`
		TargetVersion  string `json:"TargetVersion"`
	}
	type resource struct {
		Type       string     `json:"Type"`
		Properties properties `json:"Properties"`
	}

	resources := []map[string]resource{
		{
			input.FunctionName: {
				Type: "AWS::Lambda::Function",
				Properties: properties{
					Name:           input.FunctionName,
					Alias:          input.AliasName,
					CurrentVersion: currentVersion,
					TargetVersion:  input.TargetVersion,
				},
			},
		},
	}

	return buildAppSpec(buildAppSpecInput{
		Resources:       resources,
		BeforeTrafficFn: input.BeforeTrafficFnArn,
		AfterTrafficFn:  input.AfterTrafficFnArn,
	})
}
