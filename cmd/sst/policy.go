package main

import (
	"errors"
	"fmt"

	"github.com/sst/sst/v3/cmd/sst/mosaic/ui"
	"github.com/sst/sst/v3/pkg/project"
)

func handlePolicyError(err error) error {
	if errors.Is(err, project.ErrPolicyViolation) {
		fmt.Println(
			ui.TEXT_DANGER_BOLD.Render("✕"),
			ui.TEXT_DANGER_BOLD.Render(" Policy Violations Detected"),
		)
		fmt.Println()

		fmt.Println(ui.TEXT_DANGER.Render("The deployment would violate policy constraints."))
		fmt.Println(ui.TEXT_DANGER.Render("Review the policy violations and update your infrastructure accordingly."))

		if policyErr, ok := err.(*project.PolicyViolationError); ok {
			fmt.Println()
			fmt.Println(ui.TEXT_DANGER_BOLD.Render("Violations:"))

			if len(policyErr.Violations) > 0 {
				for _, violation := range policyErr.Violations {
					fmt.Println()
					fmt.Println(ui.TEXT_DANGER.Render(violation))
				}

				fmt.Println()
				fmt.Println(ui.TEXT_WARNING.Render("To fix these violations, update your infrastructure to comply with the policy requirements."))
			} else {
				fmt.Println()
				fmt.Println(ui.TEXT_DANGER.Render("No detailed violation information available."))
				fmt.Println(ui.TEXT_DANGER.Render("Check the logs for more details."))
			}
		}

		fmt.Println()
		return err
	} else if errors.Is(err, project.ErrPolicyConfigError) {
		fmt.Println(
			ui.TEXT_DANGER_BOLD.Render("✕"),
			ui.TEXT_DANGER_BOLD.Render(" Policy Configuration Error"),
		)
		fmt.Println()

		if configErr, ok := err.(*project.PolicyConfigError); ok && configErr.Message != "" {
			fmt.Println(ui.TEXT_DANGER.Render("Error details:"))
			fmt.Println(ui.TEXT_DANGER.Render(configErr.Message))
			fmt.Println()
		}

		return err
	}

	return err
}
