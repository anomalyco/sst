"use strict";
var __extends = (this && this.__extends) || (function () {
    var extendStatics = function (d, b) {
        extendStatics = Object.setPrototypeOf ||
            ({ __proto__: [] } instanceof Array && function (d, b) { d.__proto__ = b; }) ||
            function (d, b) { for (var p in b) if (Object.prototype.hasOwnProperty.call(b, p)) d[p] = b[p]; };
        return extendStatics(d, b);
    };
    return function (d, b) {
        if (typeof b !== "function" && b !== null)
            throw new TypeError("Class extends value " + String(b) + " is not a constructor or null");
        extendStatics(d, b);
        function __() { this.constructor = d; }
        d.prototype = b === null ? Object.create(b) : (__.prototype = b.prototype, new __());
    };
})();
var __spreadArray = (this && this.__spreadArray) || function (to, from, pack) {
    if (pack || arguments.length === 2) for (var i = 0, l = from.length, ar; i < l; i++) {
        if (ar || !(i in from)) {
            if (!ar) ar = Array.prototype.slice.call(from, 0, i);
            ar[i] = from[i];
        }
    }
    return to.concat(ar || Array.prototype.slice.call(from));
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Cron = void 0;
var pulumi_1 = require("@pulumi/pulumi");
var component_1 = require("../component");
var aws_1 = require("@pulumi/aws");
var function_builder_1 = require("./helpers/function-builder");
var error_1 = require("../error");
/**
 * The `Cron` component lets you add cron jobs to your app
 * using [Amazon Event Bus](https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-event-bus.html). The cron job can invoke a `Function` or a container `Task`.
 *
 * @example
 * #### Cron job function
 *
 * Pass in a `schedule` and a `function` that'll be executed.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Cron("MyCronJob", {
 *   function: "src/cron.handler",
 *   schedule: "rate(1 minute)"
 * });
 * ```
 *
 * #### Cron job container task
 *
 * Create a container task and pass in a `schedule` and a `task` that'll be executed.
 *
 * ```ts title="sst.config.ts" {5}
 * const myCluster = new sst.aws.Cluster("MyCluster");
 * const myTask = new sst.aws.Task("MyTask", { cluster: myCluster });
 *
 * new sst.aws.Cron("MyCronJob", {
 *   task: myTask,
 *   schedule: "rate(1 day)"
 * });
 * ```
 *
 * #### Customize the function
 *
 * ```js title="sst.config.ts"
 * new sst.aws.Cron("MyCronJob", {
 *   schedule: "rate(1 minute)",
 *   function: {
 *     handler: "src/cron.handler",
 *     timeout: "60 seconds"
 *   }
 * });
 * ```
 */
var Cron = /** @class */ (function (_super) {
    __extends(Cron, _super);
    function Cron(name, args, opts) {
        var _a;
        var _this = _super.call(this, __pulumiType, name, args, opts) || this;
        var parent = _this;
        var fnArgs = normalizeFunction();
        var event = (0, pulumi_1.output)(args.event || {});
        normalizeTargets();
        var enabled = (0, pulumi_1.output)((_a = args.enabled) !== null && _a !== void 0 ? _a : true);
        var rule = createRule();
        var fn = createFunction();
        var role = createRole();
        var target = createTarget();
        _this.name = name;
        _this.fn = fn;
        _this.rule = rule;
        _this.target = target;
        function normalizeFunction() {
            var _a;
            if (args.job && args.function)
                throw new error_1.VisibleError("You cannot provide both \"job\" and \"function\" in the \"".concat(name, "\" Cron component. The \"job\" property has been deprecated. Use \"function\" instead."));
            var input = (_a = args.function) !== null && _a !== void 0 ? _a : args.job;
            return input ? (0, pulumi_1.output)(input) : undefined;
        }
        function normalizeTargets() {
            if (fnArgs && args.task)
                throw new error_1.VisibleError("You cannot provide both a function and a task in the \"".concat(name, "\" Cron component."));
        }
        function createRule() {
            var _a;
            var _b;
            return new ((_a = aws_1.cloudwatch.EventRule).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.rule, "".concat(name, "Rule"), {
                scheduleExpression: args.schedule,
                state: enabled.apply(function (v) { return (v ? "ENABLED" : "DISABLED"); }),
            }, { parent: parent }), false)))();
        }
        function createFunction() {
            if (!fnArgs)
                return;
            var fn = fnArgs.apply(function (fnArgs) {
                return (0, function_builder_1.functionBuilder)("".concat(name, "Handler"), fnArgs, {}, undefined, {
                    parent: parent,
                });
            });
            new aws_1.lambda.Permission("".concat(name, "Permission"), {
                action: "lambda:InvokeFunction",
                function: fn.arn,
                principal: "events.amazonaws.com",
                sourceArn: rule.arn,
            }, { parent: parent });
            return fn;
        }
        function createRole() {
            if (!args.task)
                return;
            return new aws_1.iam.Role("".concat(name, "TargetRole"), {
                assumeRolePolicy: aws_1.iam.assumeRolePolicyForPrincipal({
                    Service: "events.amazonaws.com",
                }),
                inlinePolicies: [
                    {
                        name: "inline",
                        policy: aws_1.iam.getPolicyDocumentOutput({
                            statements: [
                                {
                                    actions: ["ecs:RunTask"],
                                    resources: [args.task.nodes.taskDefinition.arn],
                                },
                                {
                                    actions: ["iam:PassRole"],
                                    resources: [
                                        args.task.nodes.executionRole.arn,
                                        args.task.nodes.taskRole.arn,
                                    ],
                                },
                            ],
                        }).json,
                    },
                ],
            }, { parent: parent });
        }
        function createTarget() {
            var _a;
            var _b;
            return new ((_a = aws_1.cloudwatch.EventTarget).bind.apply(_a, __spreadArray([void 0], (0, component_1.transform)((_b = args.transform) === null || _b === void 0 ? void 0 : _b.target, "".concat(name, "Target"), fn
                ? {
                    arn: fn.arn,
                    rule: rule.name,
                    input: event.apply(function (event) { return JSON.stringify(event); }),
                }
                : {
                    arn: args.task.cluster,
                    rule: rule.name,
                    ecsTarget: {
                        launchType: "FARGATE",
                        taskDefinitionArn: args.task.nodes.taskDefinition.arn,
                        networkConfiguration: {
                            subnets: args.task.subnets,
                            securityGroups: args.task.securityGroups,
                            assignPublicIp: args.task.assignPublicIp,
                        },
                    },
                    roleArn: role.arn,
                    input: (0, pulumi_1.all)([event, args.task.containers]).apply(function (_a) {
                        var event = _a[0], containers = _a[1];
                        return JSON.stringify({
                            containerOverrides: containers.map(function (name) { return ({
                                name: name,
                                environment: [
                                    {
                                        name: "SST_EVENT",
                                        value: JSON.stringify(event),
                                    },
                                ],
                            }); }),
                        });
                    }),
                }, { parent: parent }), false)))();
        }
        return _this;
    }
    Object.defineProperty(Cron.prototype, "nodes", {
        /**
         * The underlying [resources](/docs/components/#nodes) this component creates.
         */
        get: function () {
            var self = this;
            return {
                /**
                 * The AWS Lambda Function that'll be invoked when the cron job runs.
                 * @deprecated Use `nodes.function` instead.
                 */
                get job() {
                    if (!self.fn)
                        throw new error_1.VisibleError("No function created for the \"".concat(self.name, "\" cron job."));
                    return self.fn.apply(function (fn) { return fn.getFunction(); });
                },
                /**
                 * The AWS Lambda Function that'll be invoked when the cron job runs.
                 */
                get function() {
                    if (!self.fn)
                        throw new error_1.VisibleError("No function created for the \"".concat(self.name, "\" cron job."));
                    return self.fn.apply(function (fn) { return fn.getFunction(); });
                },
                /**
                 * The EventBridge Rule resource.
                 */
                rule: this.rule,
                /**
                 * The EventBridge Target resource.
                 */
                target: this.target,
            };
        },
        enumerable: false,
        configurable: true
    });
    return Cron;
}(component_1.Component));
exports.Cron = Cron;
var __pulumiType = "sst:aws:Cron";
// @ts-expect-error
Cron.__pulumiType = __pulumiType;
