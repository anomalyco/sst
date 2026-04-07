import * as pulumi from "@pulumi/pulumi";
import { describe, expect, it } from "vitest";
import { Function } from "../../src/components/aws/function";
import { functionBuilder } from "../../src/components/aws/helpers/function-builder";

async function resolveOutputs<T extends readonly unknown[]>(values: T) {
  return await new Promise<T>((resolve, reject) => {
    pulumi.all(values as unknown as any[]).apply((result) => {
      try {
        resolve(result as unknown as T);
      } catch (error) {
        reject(error);
      }
    });
  });
}

function createMockFunction({
  durable,
  publish,
}: {
  durable: boolean;
  publish: boolean;
}) {
  const fn = Object.create(Function.prototype) as Function;
  Object.assign(fn as any, {
    durable,
    function: pulumi.output({
      arn: "arn:aws:lambda:us-east-1:123456789012:function:my-func",
      qualifiedArn:
        "arn:aws:lambda:us-east-1:123456789012:function:my-func:$LATEST",
      invokeArn:
        "arn:aws:apigateway:us-east-1:lambda:path/2015-03-31/functions/arn:aws:lambda:us-east-1:123456789012:function:my-func/invocations",
      qualifiedInvokeArn:
        "arn:aws:apigateway:us-east-1:lambda:path/2015-03-31/functions/arn:aws:lambda:us-east-1:123456789012:function:my-func:$LATEST/invocations",
      responseStreamingInvokeArn:
        "arn:aws:apigateway:us-east-1:lambda:path/2021-11-15/functions/arn:aws:lambda:us-east-1:123456789012:function:my-func/response-streaming-invocations",
      publish,
    }),
  });
  return fn;
}

describe("Function targets", () => {
  it("keeps arn raw and exposes qualified durable targets", async () => {
    const fn = createMockFunction({ durable: true, publish: false });
    const [arn, targetArn, qualifier, targetInvokeArn, targetResponse] =
      await resolveOutputs([
        fn.arn,
        fn.targetArn,
        fn.qualifier,
        fn.targetInvokeArn,
        fn.targetResponseStreamingInvokeArn,
      ] as const);

    expect({ arn, targetArn, qualifier, targetInvokeArn, targetResponse }).toEqual(
      {
        arn: "arn:aws:lambda:us-east-1:123456789012:function:my-func",
        targetArn:
          "arn:aws:lambda:us-east-1:123456789012:function:my-func:$LATEST",
        qualifier: "$LATEST",
        targetInvokeArn:
          "arn:aws:apigateway:us-east-1:lambda:path/2015-03-31/functions/arn:aws:lambda:us-east-1:123456789012:function:my-func:$LATEST/invocations",
        targetResponse:
          "arn:aws:apigateway:us-east-1:lambda:path/2021-11-15/functions/arn:aws:lambda:us-east-1:123456789012:function:my-func:$LATEST/response-streaming-invocations",
      },
    );
  });
});

describe("functionBuilder", () => {
  it("normalizes raw qualified function arns", async () => {
    const builder = functionBuilder(
      "MyFunction",
      "arn:aws:lambda:us-east-1:123456789012:function:my-func:live",
      {},
    );
    const [arn, targetArn, qualifier, targetInvokeArn, targetResponse] =
      await resolveOutputs([
        builder.arn,
        builder.targetArn,
        builder.qualifier,
        builder.targetInvokeArn,
        builder.targetResponseStreamingInvokeArn,
      ] as const);

    expect({ arn, targetArn, qualifier, targetInvokeArn, targetResponse }).toEqual(
      {
        arn: "arn:aws:lambda:us-east-1:123456789012:function:my-func",
        targetArn:
          "arn:aws:lambda:us-east-1:123456789012:function:my-func:live",
        qualifier: "live",
        targetInvokeArn:
          "arn:aws:apigateway:us-east-1:lambda:path/2015-03-31/functions/arn:aws:lambda:us-east-1:123456789012:function:my-func:live/invocations",
        targetResponse:
          "arn:aws:apigateway:us-east-1:lambda:path/2021-11-15/functions/arn:aws:lambda:us-east-1:123456789012:function:my-func:live/response-streaming-invocations",
      },
    );
  });
});
