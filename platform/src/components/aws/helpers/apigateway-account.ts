import { getPartitionOutput, apigateway, iam } from "@pulumi/aws";
import {
  ComponentResourceOptions,
  ProviderResource,
  jsonStringify,
  interpolate,
} from "@pulumi/pulumi";
import { $print } from "../../component";
import { lazy } from "../../../util/lazy";

// The account is a singleton per provider, so gateways on the same provider
// share one read. The read keeps the name of the first gateway that asks for
// it, which is a name this helper already registered before the cache.
const useAccountReads = lazy(
  () => new Map<ProviderResource | undefined, apigateway.Account>(),
);

function useAccountRead(namePrefix: string, opts: ComponentResourceOptions) {
  const reads = useAccountReads();
  const existing = reads.get(opts.provider);
  if (existing) return existing;

  const account = apigateway.Account.get(
    `${namePrefix}APIGatewayAccount`,
    "APIGatewayAccount",
    undefined,
    { provider: opts.provider },
  );
  reads.set(opts.provider, account);
  return account;
}

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
  const account = useAccountRead(namePrefix, opts);

  return account.cloudwatchRoleArn.apply((arn) => {
    if (arn) return account;

    return new apigateway.Account(
      `${namePrefix}APIGatewayAccountSetup`,
      {
        cloudwatchRoleArn: useCloudWatchRole(opts).arn,
      },
      { retainOnDelete: true, provider: opts.provider },
    );
  });
}
