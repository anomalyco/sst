import { getPartitionOutput, apigateway, iam } from "@pulumi/aws";
import {
  ComponentResourceOptions,
  Output,
  ProviderResource,
  jsonStringify,
  interpolate,
} from "@pulumi/pulumi";
import { $print } from "../../component";
import { lazy } from "../../../util/lazy";

// The API Gateway account is a singleton per provider (account + region), so
// one Read resource per provider is enough. Reading it once per ApiGateway
// component re-reads it from AWS that many times on every update.
const useAccountCache = lazy(
  () => new Map<ProviderResource | undefined, Output<apigateway.Account>>(),
);

let cloudWatchRole: iam.Role | undefined;

function useCloudWatchRole(opts: ComponentResourceOptions) {
  const partition = getPartitionOutput(undefined, opts).partition;
  cloudWatchRole ??= new iam.Role(
    `APIGatewayPushToCloudWatchLogsRole`,
    {
      assumeRolePolicy: jsonStringify({
        Version: "2012-10-17",
        Statement: [
          {
            Effect: "Allow",
            Principal: {
              Service: "apigateway.amazonaws.com",
            },
            Action: "sts:AssumeRole",
          },
        ],
      }),
      managedPolicyArns: [
        interpolate`arn:${partition}:iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs`,
      ],
    },
    { retainOnDelete: true, provider: opts.provider },
  );
  return cloudWatchRole;
}

export function setupApiGatewayAccount(
  namePrefix: string,
  opts: ComponentResourceOptions,
) {
  const cache = useAccountCache();
  const existing = cache.get(opts.provider);
  if (existing) return existing;

  // The first provider keeps the bare name; later providers get a suffix to
  // keep URNs unique when gateways span multiple providers.
  const suffix = cache.size === 0 ? "" : `${cache.size + 1}`;
  const account = apigateway.Account.get(
    `APIGatewayAccount${suffix}`,
    "APIGatewayAccount",
    undefined,
    { provider: opts.provider },
  );

  const result = account.cloudwatchRoleArn.apply((arn) => {
    if (arn) return account;

    return new apigateway.Account(
      `${namePrefix}APIGatewayAccountSetup`,
      {
        cloudwatchRoleArn: useCloudWatchRole(opts).arn,
      },
      { retainOnDelete: true, provider: opts.provider },
    );
  });

  cache.set(opts.provider, result);
  return result;
}
