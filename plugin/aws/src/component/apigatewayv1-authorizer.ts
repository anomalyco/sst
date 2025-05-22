import * as sst from "sst-plugin";
import { ApiGatewayV1AuthorizerArgs } from "./apigatewayv1.js";
import { AWSComponent } from "../component.js";
import { apigateway, lambda } from "@pulumi/aws";
import { functionBuilder, FunctionBuilder } from "./util/function-builder.js";
import { VisibleError } from "sst-plugin/error";
import { transform } from "sst-plugin/internal/transform";

export interface AuthorizerArgs extends ApiGatewayV1AuthorizerArgs {
  /**
   * The API Gateway to use for the route.
   */
  api: sst.Input<{
    /**
     * The name of the API Gateway.
     */
    name: sst.Input<string>;
    /**
     * The ID of the API Gateway.
     */
    id: sst.Input<string>;
    /**
     * The execution ARN of the API Gateway.
     */
    executionArn: sst.Input<string>;
  }>;
}

/**
 * The `ApiGatewayV1Authorizer` component is internally used by the `ApiGatewayV1` component
 * to add authorizers to [Amazon API Gateway REST API](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-rest-api.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `addAuthorizer` method of the `ApiGatewayV1` component.
 */
export class ApiGatewayV1Authorizer extends AWSComponent {
  private readonly authorizer: apigateway.Authorizer;
  private readonly fn?: FunctionBuilder;

  constructor(
    name: string,
    args: AuthorizerArgs,
    opts: sst.ComponentOptions = {},
  ) {
    super(__pulumiType, name, args, opts);

    const self = this;

    const api = sst.output(args.api);

    validateSingleAuthorizer();
    const type = getType();

    const fn = createFunction();
    const authorizer = createAuthorizer();
    createPermission();

    this.fn = fn;
    this.authorizer = authorizer;

    function validateSingleAuthorizer() {
      const authorizers = [
        args.requestFunction,
        args.tokenFunction,
        args.userPools,
      ].filter((e) => e);

      if (authorizers.length === 0)
        throw new VisibleError(
          `Please provide one of "requestFunction", "tokenFunction", or "userPools" for the ${args.name} authorizer.`,
        );

      if (authorizers.length > 1) {
        throw new VisibleError(
          `Please provide only one of "requestFunction", "tokenFunction", or "userPools" for the ${args.name} authorizer.`,
        );
      }
    }

    function getType() {
      if (args.tokenFunction) return "TOKEN";
      if (args.requestFunction) return "REQUEST";
      if (args.userPools) return "COGNITO_USER_POOLS";
    }

    function createFunction() {
      const fn = args.tokenFunction ?? args.requestFunction;
      if (!fn) return;

      return functionBuilder(
        `${name}Handler`,
        fn,
        {
          description: sst.interpolate`${api.name} authorizer`,
        },
        undefined,
        { parent: self },
      );
    }

    function createPermission() {
      if (!fn) return;

      return new lambda.Permission(
        `${name}Permission`,
        {
          action: "lambda:InvokeFunction",
          function: fn.arn,
          principal: "apigateway.amazonaws.com",
          sourceArn: sst.interpolate`${api.executionArn}/authorizers/${authorizer.id}`,
        },
        { parent: self },
      );
    }

    function createAuthorizer() {
      return new apigateway.Authorizer(
        ...transform(
          args.transform?.authorizer,
          `${name}Authorizer`,
          {
            restApi: api.id,
            type,
            name: args.name,
            providerArns: args.userPools,
            authorizerUri: fn?.invokeArn,
            authorizerResultTtlInSeconds: args.ttl,
            identitySource: args.identitySource,
          },
          { parent: self },
        ),
      );
    }
  }

  /**
   * The ID of the authorizer.
   */
  public get id() {
    return this.authorizer.id;
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    const self = this;
    return {
      /**
       * The API Gateway Authorizer.
       */
      authorizer: this.authorizer,
      /**
       * The Lambda function used by the authorizer.
       */
      get function() {
        if (!self.fn)
          throw new VisibleError(
            "Cannot access `nodes.function` because the data source does not use a Lambda function.",
          );
        return self.fn.apply((fn) => fn.getFunction());
      },
    };
  }
}

const __pulumiType = "sst:aws:ApiGatewayV1Authorizer";
// @ts-expect-error
ApiGatewayV1Authorizer.__pulumiType = __pulumiType;
