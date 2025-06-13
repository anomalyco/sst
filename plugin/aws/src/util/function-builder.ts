import * as sst from "sst-plugin";
import { Transform, transform } from "sst-plugin/internal/transform";
import { FunctionArgs, FunctionArn } from "../function";
import { VisibleError } from "sst-plugin/error";
import { Function } from "../function";

export type FunctionBuilder = sst.Output<{
  getFunction: () => Function;
  arn: sst.Output<string>;
  invokeArn: sst.Output<string>;
}>;

export function functionBuilder(
  name: string,
  definition: sst.Input<string | FunctionArn | FunctionArgs>,
  defaultArgs: Pick<
    FunctionArgs,
    "description" | "link" | "environment" | "permissions" | "url" | "_skipHint"
  >,
  argsTransform?: Transform<FunctionArgs>,
  opts?: sst.ComponentOptions,
): FunctionBuilder {
  return sst.output(definition).apply((definition) => {
    if (typeof definition === "string") {
      // Case 1: The definition is an ARN
      if (definition.startsWith("arn:")) {
        const parts = definition.split(":");
        return {
          getFunction: () => {
            throw new VisibleError(
              "Cannot access the created function because it is referenced as an ARN.",
            );
          },
          arn: sst.output(definition),
          invokeArn: sst.output(
            `arn:${parts[1]}:apigateway:${parts[3]}:lambda:path/2015-03-31/functions/${definition}/invocations`,
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
      return {
        getFunction: () => fn,
        arn: fn.arn,
        invokeArn: fn.nodes.function.invokeArn,
      };
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
            link: sst
              .resolve([defaultArgs?.link, definition.link])
              .apply(([defaultLink, link]) => [
                ...(defaultLink ?? []),
                ...(link ?? []),
              ]),
            environment: sst
              .resolve([defaultArgs?.environment, definition.environment])
              .apply(([defaultEnvironment, environment]) => ({
                ...(defaultEnvironment ?? {}),
                ...(environment ?? {}),
              })),
            permissions: sst
              .resolve([defaultArgs?.permissions, definition.permissions])
              .apply(([defaultPermissions, permissions]) => [
                ...(defaultPermissions ?? []),
                ...(permissions ?? []),
              ]),
          },
          opts || {},
        ),
      );
      return {
        getFunction: () => fn,
        arn: fn.arn,
        invokeArn: fn.nodes.function.invokeArn,
      };
    }
    throw new Error(`Invalid function definition for the "${name}" Function`);
  });
}
