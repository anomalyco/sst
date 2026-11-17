import { all } from "@pulumi/pulumi";
import type { ComponentResourceOptions, Input } from "@pulumi/pulumi";
import { Component } from "../component";
import {
  buildKvNamespace,
  createKvRouteData,
  parsePattern,
  updateKvRoutes,
} from "./router-base-route";
import type { RouterBaseRouteArgs } from "./router-base-route";
import type { ProtectionConfig, RouterUrlRouteArgs } from "./router";
import { toSeconds } from "../duration";

export interface Args extends RouterBaseRouteArgs {
  /**
   * The URL to route to.
   */
  url: Input<string>;
  /**
   * Additional arguments for the route.
   */
  routeArgs?: Input<RouterUrlRouteArgs>;
  /**
   * The protection configuration inherited from the Router.
   */
  protection: Input<ProtectionConfig>;
}

/**
 * The `RouterUrlRoute` component is internally used by the `Router` component
 * to add routes.
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `route` method of the `Router` component.
 */
export class RouterUrlRoute extends Component {
  constructor(name: string, args: Args, opts?: ComponentResourceOptions) {
    super(__pulumiType, name, args, opts);

    const self = this;

    all([args.url, args.pattern, args.routeArgs, args.protection]).apply(
      ([url, pattern, routeArgs, protection]) => {
        const u = new URL(url);
        const host = u.host;
        const protocol = u.protocol.slice(0, -1);
        const useOac =
          /^[a-z0-9]{32}\.lambda-url\.[a-z0-9-]+\.on\.aws$/.test(u.hostname) &&
          (protection.mode === "oac" ||
            protection.mode === "oac-with-edge-signing");

        const patternData = parsePattern(pattern);
        const namespace = buildKvNamespace(name);
        createKvRouteData(name, args, self, namespace, {
          host,
          rewrite: routeArgs?.rewrite,
          origin: {
            protocol: useOac || protocol === "https" ? undefined : protocol,
            connectionAttempts: routeArgs?.connectionAttempts,
            ...(useOac
              ? {
                  originAccessControlConfig: {
                    enabled: true,
                    signingBehavior: "always",
                    signingProtocol: "sigv4",
                    originType: "lambda",
                  },
                }
              : {}),
            timeouts: (() => {
              const timeouts = [
                "connectionTimeout" as const,
                "readTimeout" as const,
                "keepAliveTimeout" as const,
              ].flatMap((k) => {
                const value = routeArgs?.[k];
                return value ? [[k, toSeconds(value)]] : [];
              });
              return timeouts.length ? Object.fromEntries(timeouts) : undefined;
            })(),
          },
        });
        updateKvRoutes(name, args, self, "url", namespace, patternData);
      },
    );
  }
}

const __pulumiType = "sst:aws:RouterUrlRoute";
// @ts-expect-error
RouterUrlRoute.__pulumiType = __pulumiType;
