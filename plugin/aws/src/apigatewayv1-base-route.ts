import { apigateway } from "@pulumi/aws";
import * as sst from "sst-plugin";
import { transform } from "sst-plugin/internal/transform";
import { ApiGatewayV1RouteArgs } from "./apigatewayv1.js";

export interface ApiGatewayV1BaseRouteArgs extends ApiGatewayV1RouteArgs {
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
   * The route method.
   */
  method: string;
  /**
   * The route path.
   */
  path: string;
  /**
   * The route resource ID.
   */
  resourceId: sst.Input<string>;
}

export function createMethod(
  name: string,
  args: ApiGatewayV1BaseRouteArgs,
  parent: sst.Component,
) {
  const { api, method, resourceId, auth, apiKey } = args;

  const authArgs = sst.output(auth).apply((auth) => {
    if (!auth) return { authorization: "NONE" };
    if (auth.iam) return { authorization: "AWS_IAM" };
    if (auth.custom)
      return { authorization: "CUSTOM", authorizerId: auth.custom };
    if (auth.cognito)
      return {
        authorization: "COGNITO_USER_POOLS",
        authorizerId: auth.cognito.authorizer,
        authorizationScopes: auth.cognito.scopes,
      };
    return { authorization: "NONE" };
  });

  return authArgs.apply(
    (authArgs) =>
      new apigateway.Method(
        ...transform(
          args.transform?.method,
          `${name}Method`,
          {
            restApi: sst.output(api).id,
            resourceId: resourceId,
            httpMethod: method,
            authorization: authArgs.authorization,
            authorizerId: authArgs.authorizerId,
            authorizationScopes: authArgs.authorizationScopes,
            apiKeyRequired: apiKey,
          },
          { parent },
        ),
      ),
  );
}
