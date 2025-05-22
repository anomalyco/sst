import * as sst from "sst-plugin";
import { apigateway } from "@pulumi/aws";
import { ApiGatewayV1ApiKeyArgs } from "./apigatewayv1.js";
import { AWSComponent } from "../component.js";

export interface ApiKeyArgs extends ApiGatewayV1ApiKeyArgs {
  /**
   * The API Gateway REST API to use for the API key.
   */
  apiId: sst.Input<string>;
  /**
   * The API Gateway Usage Plan to use for the API key.
   */
  usagePlanId: sst.Input<string>;
}

/**
 * The `ApiGatewayV1ApiKey` component is internally used by the `ApiGatewayV1UsagePlan` component
 * to add API keys to [Amazon API Gateway REST API](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-rest-api.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `addApiKey` method of the `ApiGatewayV1UsagePlan` component.
 */
export class ApiGatewayV1ApiKey extends AWSComponent implements sst.Linkable {
  private readonly key: apigateway.ApiKey;

  constructor(name: string, args: ApiKeyArgs, opts: sst.ComponentOptions = {}) {
    super(__pulumiType, name, args, opts);

    const self = this;

    this.key = new apigateway.ApiKey(
      `${name}ApiKey`,
      {
        value: args.value,
      },
      { parent: self },
    );

    new apigateway.UsagePlanKey(
      `${name}UsagePlanKey`,
      {
        keyId: this.key.id,
        keyType: "API_KEY",
        usagePlanId: args.usagePlanId,
      },
      { parent: self },
    );
  }

  /**
   * The API key value.
   */
  public get value() {
    return this.key.value;
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The API Gateway API Key.
       */
      apiKey: this.key,
    };
  }

  /** @internal */
  public getSSTLink() {
    return {
      properties: {
        value: this.value,
      },
    };
  }
}

const __pulumiType = "sst:aws:ApiGatewayV1ApiKey";
// @ts-expect-error
ApiGatewayV1ApiKey.__pulumiType = __pulumiType;
