import { cloudwatch } from "@pulumi/aws";
import * as sst from "sst-plugin";
import { transform, Transform } from "sst-plugin/internal/transform";
import { BusSubscriberArgs } from "./bus.js";

export interface BusBaseSubscriberArgs extends BusSubscriberArgs {
  /**
   * The bus to use.
   */
  bus: sst.Input<{
    /**
     * The ARN of the bus.
     */
    arn: sst.Input<string>;
    /**
     * The name of the bus.
     */
    name: sst.Input<string>;
  }>;
}

export function createRule(
  name: string,
  eventBusName: sst.Input<string>,
  args: BusBaseSubscriberArgs,
  parent: sst.Component,
) {
  return new cloudwatch.EventRule(
    ...transform(
      args?.transform?.rule,
      `${name}Rule`,
      {
        eventBusName,
        eventPattern: args.pattern
          ? sst.output(args.pattern).apply((pattern) =>
              JSON.stringify({
                "detail-type": pattern.detailType,
                source: pattern.source,
                detail: pattern.detail,
              }),
            )
          : JSON.stringify({
              source: [{ prefix: "" }],
            }),
      },
      { parent },
    ),
  );
}
