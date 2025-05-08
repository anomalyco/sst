"use strict";
var __makeTemplateObject = (this && this.__makeTemplateObject) || function (cooked, raw) {
    if (Object.defineProperty) { Object.defineProperty(cooked, "raw", { value: raw }); } else { cooked.raw = raw; }
    return cooked;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.setupApiGatewayAccount = setupApiGatewayAccount;
var aws_1 = require("@pulumi/aws");
var pulumi_1 = require("@pulumi/pulumi");
function setupApiGatewayAccount(namePrefix, opts) {
    var account = aws_1.apigateway.Account.get("".concat(namePrefix, "APIGatewayAccount"), "APIGatewayAccount", undefined, { provider: opts.provider });
    return account.cloudwatchRoleArn.apply(function (arn) {
        if (arn)
            return account;
        var partition = (0, aws_1.getPartitionOutput)(undefined, opts).partition;
        var role = new aws_1.iam.Role("APIGatewayPushToCloudWatchLogsRole", {
            assumeRolePolicy: (0, pulumi_1.jsonStringify)({
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
                (0, pulumi_1.interpolate)(templateObject_1 || (templateObject_1 = __makeTemplateObject(["arn:", ":iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"], ["arn:", ":iam::aws:policy/service-role/AmazonAPIGatewayPushToCloudWatchLogs"])), partition),
            ],
        }, { retainOnDelete: true, provider: opts.provider });
        return new aws_1.apigateway.Account("".concat(namePrefix, "APIGatewayAccountSetup"), {
            cloudwatchRoleArn: role.arn,
        }, { provider: opts.provider });
    });
}
var templateObject_1;
