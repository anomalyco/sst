"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.iamEdit = iamEdit;
var pulumi_1 = require("@pulumi/pulumi");
var aws_1 = require("@pulumi/aws");
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
function iamEdit(policy, cb) {
    return (0, pulumi_1.output)(policy).apply(function (v) {
        var json = typeof v === "string" ? JSON.parse(v) : v;
        cb(json);
        return aws_1.iam.getPolicyDocumentOutput({
            sourcePolicyDocuments: [(0, pulumi_1.jsonStringify)(json)],
        }).json;
    });
}
