import * as sst from "sst-plugin";
import { Transform, transform } from "sst-plugin/internal/transform";
import { VisibleError } from "sst-plugin/error";
import { AWSComponent } from "./component";
import {
  buildKvNamespace,
  createKvRouteData,
  parsePattern,
  RouterBaseRouteArgs,
  updateKvRoutes,
} from "./router-base-route";
import { toSeconds } from "./util/duration";
import { RouterUrlRouteArgs } from "./router";

export interface Args extends RouterBaseRouteArgs {
  /**
   * The URL to route to.
   */
  url: sst.Input<string>;
  /**
   * Additional arguments for the route.
   */
  routeArgs?: sst.Input<RouterUrlRouteArgs>;
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
export class RouterUrlRoute extends sst.Component {
  constructor(name: string, args: Args, opts?: sst.ComponentOptions) {
    super(__pulumiType, name, args, opts);

    const self = this;

    sst
      .resolve([args.url, args.pattern, args.routeArgs])
      .apply(([url, pattern, routeArgs]) => {
        const u = new URL(url);
        const host = u.host;
        const protocol = u.protocol.slice(0, -1);

        const patternData = parsePattern(pattern);
        const namespace = buildKvNamespace(name);
        createKvRouteData(name, args, self, namespace, {
          host,
          rewrite: routeArgs?.rewrite,
          origin: {
            protocol: protocol === "https" ? undefined : protocol,
            connectionAttempts: routeArgs?.connectionAttempts,
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
      });
  }
}

const __pulumiType = "sst:aws:RouterUrlRoute";
// @ts-expect-error
RouterUrlRoute.__pulumiType = __pulumiType;
