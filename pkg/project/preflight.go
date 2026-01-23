package project

import (
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
)

type migrationNotice struct {
	providerName string
	fromVersion  string
	toVersion    string
	message      string
}

var migrationNotices = []migrationNotice{
	{
		providerName: "aws",
		fromVersion:  "6.0.0",
		toVersion:    "7.0.0",
		message:      "Detected AWS provider upgrade to v7.\n\nUpgrading from v6 to v7 introduces some breaking changes and requires state migration before deploying. Refer to the SST release notes and the pulumi migration guide to make the relevent changes before deploying: https://www.pulumi.com/registry/packages/aws/how-to-guides/7-0-migration \n\nThis notice will clear after migrating your state",
	},
}

func (p *Project) checkProviderUpgrade(resources []apitype.ResourceV3) []string {
	providerVersions := make(map[string]string)

	// We iterate backwards over the slice b/c when multiple provider versions are present (i.e. just after refreshing but before deploying)
	// the old provider version appears at the end of the array, and we want to override the version checkpoint with the new version.
	for i := len(resources) - 1; i >= 0; i-- {
		v := resources[i]
		name := strings.TrimPrefix(string(v.Type), "pulumi:providers:")
		if name == string(v.Type) {
			continue
		}
		versionOutput, ok := v.Outputs["version"].(string)
		if !ok {
			continue
		}
		providerVersions[name] = versionOutput
	}

	var messages []string

	for _, entry := range p.lock {
		currentVersionStr, ok := providerVersions[entry.Name]
		if !ok {
			continue
		}

		currentVersion, err := semver.NewVersion(currentVersionStr)
		if err != nil {
			continue
		}

		targetVersion, err := semver.NewVersion(entry.Version)
		if err != nil {
			continue
		}

		// Check if this is an upgrade
		if !currentVersion.LessThan(targetVersion) {
			continue
		}

		// Check against all applicable upgrade rules
		for _, rule := range migrationNotices {
			if rule.providerName != entry.Name {
				continue
			}

			ruleFrom, err := semver.NewVersion(rule.fromVersion)
			if err != nil {
				continue
			}

			ruleTo, err := semver.NewVersion(rule.toVersion)
			if err != nil {
				continue
			}

			// Check if the upgrade path crosses this rule's version range
			// Current version must be >= fromVersion AND < toVersion
			// Target version must be >= toVersion
			if currentVersion.Compare(ruleFrom) >= 0 &&
				currentVersion.Compare(ruleTo) < 0 &&
				targetVersion.Compare(ruleTo) >= 0 {
				messages = append(messages, rule.message)
			}
		}
	}

	return messages
}
