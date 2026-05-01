import {
  all,
  ComponentResourceOptions,
  Input,
  Output,
  output,
} from "@pulumi/pulumi";
import { Function, FunctionArgs, FunctionArn } from "../function.js";
import { Workflow } from "../workflow.js";
import { transform, Transform } from "../../component";
import { VisibleError } from "../../error";
import { splitQualifiedFunctionArn } from "./arn.js";

export type FunctionBuilder = Output<{
  getFunction: () => Function;
  arn: Output<string>;
  targetArn: Output<string>;
  qualifier: Output<string | undefined>;
  targetInvokeArn: Output<string>;
  targetResponseStreamingInvokeArn: Output<string>;
}>;

export function functionBuilder(
  name: string,
  definition: Input<string | Workflow | Function | FunctionArgs | FunctionArn>,
  defaultArgs: Pick<
    FunctionArgs,
    | "description"
    | "link"
    | "environment"
    | "permissions"
    | "url"
    | "streaming"
    | "_skipHint"
  >,
  argsTransform?: Transform<FunctionArgs>,
  opts?: ComponentResourceOptions,
): FunctionBuilder {
  function buildResult(fn: Function) {
    return {
      getFunction: () => fn,
      arn: fn.arn,
      targetArn: fn.targetArn,
      qualifier: fn.qualifier,
      targetInvokeArn: fn.targetInvokeArn,
      targetResponseStreamingInvokeArn: fn.targetResponseStreamingInvokeArn,
    };
  }

  return output(definition).apply((definition) => {
    if (definition instanceof Workflow) {
      return buildResult(definition.getFunction());
    }

    if (definition instanceof Function) {
      return buildResult(definition);
    }

    if (typeof definition === "string") {
      // Case 1: The definition is an ARN
      if (definition.startsWith("arn:")) {
        const { unqualifiedArn, qualifier } = splitQualifiedFunctionArn(
          definition,
        );
        const parts = definition.split(":");
        return {
          getFunction: () => {
            throw new VisibleError(
              "Cannot access the created function because it is referenced as an ARN.",
            );
          },
          arn: output(unqualifiedArn),
          targetArn: output(definition),
          qualifier: output(qualifier),
          targetInvokeArn: output(
            `arn:${parts[1]}:apigateway:${parts[3]}:lambda:path/2015-03-31/functions/${definition}/invocations`,
          ),
          targetResponseStreamingInvokeArn: output(
            `arn:${parts[1]}:apigateway:${parts[3]}:lambda:path/2021-11-15/functions/${definition}/response-streaming-invocations`,
          ),
        };
      }

      // Case 2: The definition is a handler
      const fn = new Function(
        ...transform(
          argsTransform,
          name,
          { handler: definition, ...defaultArgs },
          opts || {},
        ),
      );
      return buildResult(fn);
    }

    // Case 3: The definition is a FunctionArgs
    else if (definition.handler) {
      const fn = new Function(
        ...transform(
          argsTransform,
          name,
          {
            ...defaultArgs,
            ...definition,
            link: all([defaultArgs?.link, definition.link]).apply(
              ([defaultLink, link]) => [
                ...(defaultLink ?? []),
                ...(link ?? []),
              ],
            ),
            environment: all([
              defaultArgs?.environment,
              definition.environment,
            ]).apply(([defaultEnvironment, environment]) => ({
              ...(defaultEnvironment ?? {}),
              ...(environment ?? {}),
            })),
            permissions: all([
              defaultArgs?.permissions,
              definition.permissions,
            ]).apply(([defaultPermissions, permissions]) => [
              ...(defaultPermissions ?? []),
              ...(permissions ?? []),
            ]),
          },
          opts || {},
        ),
      );
      return buildResult(fn);
    }
    throw new Error(`Invalid function definition for the "${name}" Function`);
  });
}
