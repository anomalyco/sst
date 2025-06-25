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

type FunctionEnvironmentUpdateOutputs struct {
	Updated bool `json:"updated"`
}

func (r *FunctionEnvironmentUpdate) Create(input *FunctionEnvironmentUpdateInputs, output *CreateResult[FunctionEnvironmentUpdateOutputs]) error {
	if err := r.updateEnvironment(input); err != nil {
		return err
	}

	*output = CreateResult[FunctionEnvironmentUpdateOutputs]{
		ID: input.FunctionName,
		Outs: FunctionEnvironmentUpdateOutputs{
			Updated: true,
		},
	}
	return nil
}

func (r *FunctionEnvironmentUpdate) Update(input *UpdateInput[FunctionEnvironmentUpdateInputs, FunctionEnvironmentUpdateOutputs], output *UpdateResult[FunctionEnvironmentUpdateOutputs]) error {
	if err := r.updateEnvironment(&input.News); err != nil {
		return err
	}

	*output = UpdateResult[FunctionEnvironmentUpdateOutputs]{
		Outs: FunctionEnvironmentUpdateOutputs{
			Updated: true,
		},
	}
	return nil
}

func (r *FunctionEnvironmentUpdate) Read(input *ReadInput[FunctionEnvironmentUpdateInputs], output *ReadResult[FunctionEnvironmentUpdateOutputs]) error {
	*output = ReadResult[FunctionEnvironmentUpdateOutputs]{
		ID: input.ID,
		Outs: FunctionEnvironmentUpdateOutputs{
			Updated: false,
		},
	}
	return nil
}

func (r *FunctionEnvironmentUpdate) Diff(input *DiffInput[FunctionEnvironmentUpdateInputs, FunctionEnvironmentUpdateOutputs], output *DiffResult) error {
	*output = DiffResult{
		Changes: input.Olds.Updated != true,
	}
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