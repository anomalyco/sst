/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## ECR lifecycle policy
 *
 * Apply a lifecycle policy to the SST bootstrap ECR repository.
 */
export default $config({
  app(input) {
    return {
      name: "aws-ecr-lifecycle",
      home: "aws",
      removal: input?.stage === "production" ? "retain" : "remove",
    };
  },
  async run() {
    new aws.ecr.LifecyclePolicy("SstAssetLifecycle", {
      repository: sst.aws.getECRRepository().apply((repo) => repo.name),
      policy: JSON.stringify({
        rules: [
          {
            rulePriority: 1,
            description: "Expire untagged images pushed over 30 days ago",
            selection: {
              tagStatus: "untagged",
              countType: "sinceImagePushed",
              countUnit: "days",
              countNumber: 30,
            },
            action: { type: "expire" },
          },
        ],
      }),
    });
  },
});
