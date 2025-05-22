import * as sst from "sst-plugin";
import { iam } from "@pulumi/aws";
import { jsonStringify } from "@pulumi/pulumi";
import { Prettify } from "sst-plugin/internal/prettify";

type PartialUnwrappedPolicyDocument = {
  Id?: string;
  Version: "2008-10-17" | "2012-10-17";
  Statement: sst.Input<iam.PolicyStatement>[];
};

/**
 * A helper to modify the AWS IAM policy.
 *
 * The IAM policy document is normally in the form of a JSON string. This helper decodes
 * the string into a JSON object and passes it to the callback. Allowing you to modify the
 * policy document in a type-safe way.
 *
 * @example
 *
 * For example, this comes in handy when you are transforming the policy of a component.
 *
 * ```ts title="sst.config.ts" "sst.aws.iamEdit"
 * new sst.aws.Bucket("MyBucket", {
 *   transform: {
 *     policy: (args) => {
 *       args.policy = sst.aws.iamEdit(args.policy, (policy) => {
 *         policy.Statement.push({
 *           Effect: "Allow",
 *           Action: "s3:PutObject",
 *           Principal: { Service: "ses.amazonaws.com" },
 *           Resource: $interpolate`arn:aws:s3:::${args.bucket}/*`,
 *         });
 *       });
 *     },
 *   },
 * });
 * ```
 */
export function iamEdit(
  policy: sst.Input<iam.PolicyDocument | string>,
  cb: (doc: Prettify<PartialUnwrappedPolicyDocument>) => void,
) {
  return sst.output(policy).apply((v) => {
    const json = typeof v === "string" ? JSON.parse(v) : v;
    cb(json);
    return iam.getPolicyDocumentOutput({
      sourcePolicyDocuments: [jsonStringify(json)],
    }).json;
  });
}
