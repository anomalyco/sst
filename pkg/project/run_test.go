package project

import (
	"strings"
	"testing"
)

func TestExtractPolicyViolations(t *testing.T) {
	tests := []struct {
		name       string
		logContent string
		want       []string // Strings that should be contained in the violations
	}{
		{
			name: "standard policy section format",
			logContent: `
Policies:
  ❌ [mandatory] aws-iam-no-inline-policies (aws-iam-no-inline-policies)
    Resource: arn:aws:iam::123456789012:role/my-role
    Description: IAM roles should not have inline policies
    Learn more: https://docs.aws.amazon.com/IAM/latest/UserGuide/best-practices.html

Diagnostics:
  aws:iam:Role (my-role):
    error: Preview failed
`,
			want: []string{
				"❌ [mandatory] aws-iam-no-inline-policies",
				"Resource: arn:aws:iam::123456789012:role/my-role",
				"Description: IAM roles should not have inline policies",
			},
		},
		{
			name: "multiple policy violations",
			logContent: `
Policies:
  ❌ [mandatory] aws-iam-no-inline-policies (aws-iam-no-inline-policies)
    Resource: arn:aws:iam::123456789012:role/my-role
    Description: IAM roles should not have inline policies

  ❌ [mandatory] aws-iam-permission-boundary (aws-iam-permission-boundary)
    Resource: arn:aws:iam::123456789012:role/my-role
    Description: IAM roles must have permission boundaries

Resources:
`,
			want: []string{
				"aws-iam-no-inline-policies",
				"aws-iam-permission-boundary",
				"IAM roles should not have inline policies",
				"IAM roles must have permission boundaries",
			},
		},
		{
			name: "alternative format with policy violation",
			logContent: `
error: Policy violation: IAM roles must have permission boundaries
Resource: arn:aws:iam::123456789012:role/my-role

Stack deployment failed.
`,
			want: []string{
				"Policy violation",
				"IAM roles must have permission boundaries",
				"Resource: arn:aws:iam::123456789012:role/my-role",
			},
		},
		{
			name: "policy check failed format",
			logContent: `
policy check failed: aws-iam-permission-boundary
Resource: arn:aws:iam::123456789012:role/my-role
error: Preview failed
`,
			want: []string{
				"policy check failed",
				"aws-iam-permission-boundary",
				"Resource: arn:aws:iam::123456789012:role/my-role",
			},
		},
		{
			name: "error line format",
			logContent: `
Running program '/path/to/policy-pack' failed with an unhandled exception:
Error: Policy violation detected in resource 'my-role'
`,
			want: []string{
				"Policy violation detected in resource 'my-role'",
			},
		},
		{
			name: "no policy violations",
			logContent: `
Previewing update (dev/app):
     Type                 Name          Plan       
 +   pulumi:pulumi:Stack  app-dev       create     
 +   └─ aws:s3:Bucket     my-bucket     create     
 
Resources:
    + 2 to create

Outputs:
    bucketName: "my-bucket-123"
`,
			want: []string{},
		},
		{
			name: "typescript compilation error",
			logContent: `
Unable to compile TypeScript:
/path/to/policy-pack/index.ts(10,12): error TS2304: Cannot find name 'PolicyPack'.
/path/to/policy-pack/index.ts(15,5): error TS2345: Argument of type 'string' is not assignable to parameter of type 'number'.

Cannot find module '@pulumi/policy'
`,
			want: []string{
				"Unable to compile TypeScript",
				"Cannot find name 'PolicyPack'",
				"Cannot find module '@pulumi/policy'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPolicyViolations(tt.logContent)

			for _, want := range tt.want {
				found := false
				for _, violation := range got {
					if strings.Contains(violation, want) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ExtractPolicyViolations() missing expected content: %q", want)
				}
			}

			if len(tt.want) == 0 && len(got) > 0 {
				t.Errorf("ExtractPolicyViolations() returned %d violations, expected none", len(got))
			}
		})
	}
}
