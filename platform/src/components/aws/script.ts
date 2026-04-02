import {
  ComponentResourceOptions,
  CustomResourceOptions,
  dynamic,
} from "@pulumi/pulumi";
import { Component } from "../component.js";
import { FunctionArgs, FunctionArn } from "./function.js";
import { Input } from "../input.js";
import {
  FunctionBuilder,
  functionBuilder,
} from "./helpers/function-builder.js";
import { LambdaClient, InvokeCommand } from "@aws-sdk/client-lambda";
import { getRegionOutput } from "@pulumi/aws";
import { VisibleError } from "../error.js";

export interface ScriptArgs {
  /**
   * The function that'll be executed when the script is created.
   *
   * @example
   *
   * ```ts
   * {
   *   onCreate: "src/function.create"
   * }
   * ```
   *
   * You can pass in the full function props.
   *
   * ```ts
   * {
   *   onCreate: {
   *     handler: "src/function.create",
   *     timeout: "60 seconds"
   *   }
   * }
   * ```
   *
   * You can also pass in a function ARN.
   *
   * ```ts
   * {
   *   onCreate: "arn:aws:lambda:us-east-1:000000000000:function:my-sst-app-jayair-MyFunction",
   * }
   * ```
   */
  onCreate?: Input<string | FunctionArgs | FunctionArn>;
  /**
   * The function that'll be executed when the script is updated.
   */
  onUpdate?: Input<string | FunctionArgs | FunctionArn>;
  /**
   * The function that'll be executed when the script is deleted.
   */
  onDelete?: Input<string | FunctionArgs | FunctionArn>;
  /**
   * The event that'll be passed to the functions.
   *
   * @example
   * ```ts
   * {
   *   event: {
   *     foo: "bar",
   *   }
   * }
   * ```
   *
   * ```ts
   * function handler(event) {
   *   console.log(event.foo);
   * }
   * ```
   */
  event?: Input<Record<string, Input<string>>>;
  /**
   * By default, the onUpdate function runs during each deployment. If a version is provided, it will only run when the version changes.
   *
   * @example
   *
   * ```ts
   * {
   *   version: "4"
   * }
   * ```
   */
  version?: Input<string>;
}

type Inputs = {
  createFunctionArn?: string;
  updateFunctionArn?: string;
  deleteFunctionArn?: string;
  version?: string;
  event?: Record<string, any>;
  region: string;
};

type Outputs = Inputs & {
  id: string;
};

export interface ScriptInvocationInputs {
  createFunctionArn: Input<Inputs["createFunctionArn"]>;
  updateFunctionArn: Input<Inputs["updateFunctionArn"]>;
  deleteFunctionArn: Input<Inputs["deleteFunctionArn"]>;
  version: Input<Inputs["version"]>;
  event: Input<Inputs["event"]>;
  region: Input<Inputs["region"]>;
}

class Provider implements dynamic.ResourceProvider {
  private async invoke({
    functionName,
    event,
    region,
  }: {
    functionName: string;
    region: string;
    event?: Record<string, any>;
  }) {
    const client = new LambdaClient({ region });

    const command = new InvokeCommand({
      FunctionName: functionName,
      Payload: JSON.stringify(event ?? {}),
    });

    const response = await client.send(command);

    if (response.FunctionError) {
      const payload = JSON.parse(Buffer.from(response.Payload!).toString()) as { errorType?: string; errorMessage?: string; trace?: string[] };

      throw new VisibleError(`Script invocation failed: ${payload.errorMessage}`);
    }
  }

  async create(inputs: Inputs): Promise<dynamic.CreateResult<Outputs>> {
    if (inputs.createFunctionArn) {
      await this.invoke({
        functionName: inputs.createFunctionArn,
        event: inputs.event,
        region: inputs.region,
      });
    }

    const id = Date.now().toString();

    return {
      id,
      outs: {
        id,
        ...inputs,
      },
    };
  }

  async update(
    id: string,
    olds: Inputs,
    news: Inputs,
  ): Promise<dynamic.UpdateResult<Outputs>> {
    if (news.updateFunctionArn) {
      await this.invoke({
        functionName: news.updateFunctionArn,
        event: news.event,
        region: news.region,
      });
    }

    return {
      outs: {
        id,
        ...news,
      },
    };
  }

  async delete(id: string, props: Inputs) {
    if (props.deleteFunctionArn) {
      await this.invoke({
        functionName: props.deleteFunctionArn,
        event: props.event,
        region: props.region,
      });
    }
  }

  async diff(id: string, olds: Inputs, news: Inputs) {
    let changes = true;

    if (news.version) {
      changes = news.version !== olds.version;
    }

    return {
      changes,
      replaces: [],
      stables: [],
      deleteBeforeReplace: false,
    };
  }
}

export class ScriptInvocation extends dynamic.Resource {
  constructor(
    name: string,
    args: ScriptInvocationInputs,
    opts?: CustomResourceOptions,
  ) {
    super(
      new Provider(),
      `${name}.sst.aws.ScriptInvocation`,
      { ...args },
      opts,
    );
  }
}

/**
 * The `Script` component lets you invoke different lambda functions for every lifecycle event of the resource (created, updated or deleted). This can be useful to running code.
 *
 * :::note
 * The IAM user deploying your app needs to have `lambda:InvokeFunction` permission.
 * :::
 *
 * @example
 *
 * #### Minimal config
 *
 * ```ts title="sst.config.ts"
 * new Script("MyScript", {
 *   onCreate: "src/function.create",
 *   onUpdate: "src/function.update",
 *   onDelete: "src/function.delete",
 * });
 * ```
 */
export class Script extends Component {
  private readonly name: string;
  private readonly createFunctionBuilder?: FunctionBuilder;
  private readonly updateFunctionBuilder?: FunctionBuilder;
  private readonly deleteFunctionBuilder?: FunctionBuilder;

  constructor(name: string, args: ScriptArgs, opts?: ComponentResourceOptions) {
    super(__pulumiType, name, args, opts);

    const parent = this;

    const createFunction = createCreateFunction();
    const updateFunction = createUpdateFunction();
    const deleteFunction = createDeleteFunction();

    this.name = name;
    this.createFunctionBuilder = createFunction;
    this.updateFunctionBuilder = updateFunction;
    this.deleteFunctionBuilder = deleteFunction;

    function createCreateFunction() {
      if (!args.onCreate) {
        return undefined;
      }

      return functionBuilder(`${name}OnCreate`, args.onCreate, {}, undefined, {
        parent,
      });
    }

    function createUpdateFunction() {
      if (!args.onUpdate) {
        return undefined;
      }

      return functionBuilder(`${name}OnUpdate`, args.onUpdate, {}, undefined, {
        parent,
      });
    }

    function createDeleteFunction() {
      if (!args.onDelete) {
        return undefined;
      }

      return functionBuilder(`${name}OnDelete`, args.onDelete, {}, undefined, {
        parent,
      });
    }

    new ScriptInvocation(
      name,
      {
        createFunctionArn: createFunction?.arn,
        updateFunctionArn: updateFunction?.arn,
        deleteFunctionArn: deleteFunction?.arn,
        version: args.version,
        event: args.event,
        region: getRegionOutput(undefined, { parent: this }).name,
      },
      { parent },
    );
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    const self = this;
    return {
      /**
       * The AWS Lambda Function that'll be invoked when the script is created.
       */
      get createFunction() {
        if (!self.createFunctionBuilder) {
          throw new VisibleError(
            `No created function created for the "${self.name}" script.`,
          );
        }

        return self.createFunctionBuilder.apply((fn) => fn.getFunction());
      },
      /**
       * The AWS Lambda Function that'll be invoked when the script is updated.
       */
      get updateFunction() {
        if (!self.updateFunctionBuilder) {
          throw new VisibleError(
            `No updated function created for the "${self.name}" script.`,
          );
        }

        return self.updateFunctionBuilder.apply((fn) => fn.getFunction());
      },
      /**
       * The AWS Lambda Function that'll be invoked when the script is deleted.
       */
      get deleteFunction() {
        if (!self.deleteFunctionBuilder) {
          throw new VisibleError(
            `No deleted function created for the "${self.name}" script.`,
          );
        }

        return self.deleteFunctionBuilder.apply((fn) => fn.getFunction());
      },
    };
  }
}

const __pulumiType = "sst:aws:Script";
// @ts-expect-error
Script.__pulumiType = __pulumiType;
