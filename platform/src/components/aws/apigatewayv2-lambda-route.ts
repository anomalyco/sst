import {
  ComponentResourceOptions,
  Input,
  Output,
  interpolate,
  output,
} from "@pulumi/pulumi";
import { Component, Transform, transform } from "../component";
import { Function, FunctionArgs, FunctionArn } from "./function.js";
import { apigatewayv2, lambda } from "@pulumi/aws";
import {
  ApiGatewayV2BaseRouteArgs,
  createApiRoute,
} from "./apigatewayv2-base-route";
import { FunctionBuilder, functionBuilder } from "./helpers/function-builder";

export interface Args extends ApiGatewayV2BaseRouteArgs {
  /**
   * The route function.
   *
   * Takes the handler path, the function args, or a function ARN.
   */
  handler: Input<string | FunctionArgs | FunctionArn>;
  /**
   * The resources to link to the route function.
   */
  handlerLink?: FunctionArgs["link"];
  /**
   * @internal
   */
  handlerTransform?: Transform<FunctionArgs>;
  /**
   * Reuse an already-created Lambda function, invoke permission, and
   * `apigatewayv2.Integration` instead of creating new ones.
   *
   * Populated by `ApiGatewayV2.route()` when `dedupeHandlers: true` is set on
   * the parent API and the current call's handler string matches an earlier
   * call's. The component type (and therefore the child `apigatewayv2.Route`
   * URN) stays the same whether this field is set or not — that's what keeps
   * Pulumi from trying to create a second API-Gateway route with the same
   * key during a dedup migration.
   *
   * @internal
   */
  sharedIntegration?: {
    integration: apigatewayv2.Integration;
    lambdaFunction: Output<Function>;
    permission: lambda.Permission;
  };
}

/**
 * The `ApiGatewayV2LambdaRoute` component is internally used by the `ApiGatewayV2` component
 * to add routes to your [API Gateway HTTP API](https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api.html).
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `route` method of the `ApiGatewayV2` component.
 */
export class ApiGatewayV2LambdaRoute extends Component {
  private readonly fn: FunctionBuilder | undefined;
  private readonly sharedLambdaFunction: Output<Function> | undefined;
  private readonly permission: lambda.Permission;
  private readonly apiRoute: Output<apigatewayv2.Route>;
  private readonly integration: apigatewayv2.Integration;

  constructor(name: string, args: Args, opts?: ComponentResourceOptions) {
    super(__pulumiType, name, args, opts);

    const self = this;
    const api = output(args.api);
    const route = output(args.route);
    const shared = args.sharedIntegration;

    if (shared) {
      // Dedup path: reuse the first-call's Function, Permission, and Integration.
      // Only the `apigatewayv2.Route` resource is created here, targeting the
      // existing integration. The component URN stays `ApiGatewayV2LambdaRoute`
      // so Pulumi doesn't treat a dedup-migration as a resource replacement.
      this.fn = undefined;
      this.sharedLambdaFunction = shared.lambdaFunction;
      this.permission = shared.permission;
      this.integration = shared.integration;
      this.apiRoute = createApiRoute(name, args, output(shared.integration.id), self);
      return;
    }

    const fn = functionBuilder(
      `${name}Handler`,
      args.handler,
      {
        description: interpolate`${api.name} route ${route}`,
        link: args.handlerLink,
      },
      args.handlerTransform,
      { parent: self },
    );

    const permission = new lambda.Permission(
      `${name}Permissions`,
      {
        action: "lambda:InvokeFunction",
        function: fn.arn,
        qualifier: fn.qualifier.apply((qualifier) => qualifier!),
        principal: "apigateway.amazonaws.com",
        sourceArn: interpolate`${api.executionArn}/*`,
      },
      { parent: self },
    );

    const integration = new apigatewayv2.Integration(
      ...transform(
        args.transform?.integration,
        `${name}Integration`,
        {
          apiId: api.id,
          integrationType: "AWS_PROXY",
          integrationUri: fn.targetArn,
          payloadFormatVersion: "2.0",
        },
        { parent: self, dependsOn: [permission] },
      ),
    );

    this.fn = fn;
    this.sharedLambdaFunction = undefined;
    this.permission = permission;
    this.apiRoute = createApiRoute(name, args, integration.id, self);
    this.integration = integration;
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    const self = this;
    return {
      /**
       * The Lambda function.
       */
      get function() {
        if (self.sharedLambdaFunction) return self.sharedLambdaFunction;
        return self.fn!.apply((fn) => fn.getFunction());
      },
      /**
       * The Lambda permission.
       */
      permission: this.permission,
      /**
       * The API Gateway HTTP API route.
       */
      route: this.apiRoute,
      /**
       * The API Gateway HTTP API integration.
       */
      integration: this.integration,
    };
  }
}

const __pulumiType = "sst:aws:ApiGatewayV2LambdaRoute";
// @ts-expect-error
ApiGatewayV2LambdaRoute.__pulumiType = __pulumiType;
