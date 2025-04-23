package resource

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

type FunctionEnvironmentUpdate struct {
	*AwsResource
}

type FunctionEnvironmentUpdateInputs struct {
	FunctionName string            `json:"functionName"`
	Environment  map[string]string `json:"environment"`
	Region       string            `json:"region"`
}

func (r *FunctionEnvironmentUpdate) Create(input *FunctionEnvironmentUpdateInputs, output *CreateResult[struct{}]) error {
	if err := r.updateEnvironment(input); err != nil {
		return err
	}

	*output = CreateResult[struct{}]{
		ID: input.FunctionName,
	}
	return nil
}

func (r *FunctionEnvironmentUpdate) Update(input *UpdateInput[FunctionEnvironmentUpdateInputs, struct{}], output *UpdateResult[struct{}]) error {
	if err := r.updateEnvironment(&input.News); err != nil {
		return err
	}

	*output = UpdateResult[struct{}]{}
	return nil
}

func (r *FunctionEnvironmentUpdate) updateEnvironment(input *FunctionEnvironmentUpdateInputs) error {
	cfg, err := r.config()
	if err != nil {
		return err
	}

	// Use the specified region if provided
	if input.Region != "" {
		cfg.Region = input.Region
	}
	
	client := lambda.NewFromConfig(cfg)

	// Get the current function configuration to preserve other settings
	functionConfig, err := client.GetFunctionConfiguration(r.context, &lambda.GetFunctionConfigurationInput{
		FunctionName: aws.String(input.FunctionName),
	})
	if err != nil {
		return err
	}

	// Create environment variables map
	envVars := make(map[string]string)
	
	// If the function already has environment variables, preserve them
	if functionConfig.Environment != nil && functionConfig.Environment.Variables != nil {
		for k, v := range functionConfig.Environment.Variables {
			envVars[k] = v
		}
	}
	
	// Add or update with the new environment variables
	for k, v := range input.Environment {
		envVars[k] = v
	}

	// Update the function configuration with the new environment variables
	_, err = client.UpdateFunctionConfiguration(r.context, &lambda.UpdateFunctionConfigurationInput{
		FunctionName: aws.String(input.FunctionName),
		Environment: &types.Environment{
			Variables: envVars,
		},
	})
	
	return err
} 