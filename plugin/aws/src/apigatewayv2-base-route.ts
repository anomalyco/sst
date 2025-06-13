import { apigatewayv2 } from "@pulumi/aws";
import * as sst from "sst-plugin";
import { Transform, transform } from "sst-plugin/internal/transform";
import { ApiGatewayV2RouteArgs } from "./apigatewayv2";

export interface ApiGatewayV2BaseRouteArgs extends ApiGatewayV2RouteArgs {
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
  /**
   * The path for the route.
   */
  route: sst.Input<string>;
}

export function createApiRoute(
  name: string,
  args: ApiGatewayV2BaseRouteArgs,
  integrationId: sst.Output<string>,
  parent: sst.Component,
) {
  const authArgs = sst.output(args.auth).apply((auth) => {
    if (!auth) return { authorizationType: "NONE" };
    if (auth.iam) return { authorizationType: "AWS_IAM" };
    if (auth.lambda)
      return {
        authorizationType: "CUSTOM",
        authorizerId: auth.lambda,
      };
    if (auth.jwt)
      return {
        authorizationType: "JWT",
        authorizationScopes: auth.jwt.scopes,
        authorizerId: auth.jwt.authorizer,
      };
    return { authorizationType: "NONE" };
  });

  return authArgs.apply(
    (authArgs) =>
      new apigatewayv2.Route(
        ...transform(
          args.transform?.route,
          `${name}Route`,
          {
            apiId: sst.output(args.api).id,
            routeKey: args.route,
            target: sst.interpolate`integrations/${integrationId}`,
            ...authArgs,
          },
          { parent },
        ),
      ),
  );
}
