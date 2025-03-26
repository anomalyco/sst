import { all, ComponentResourceOptions, Input, output } from "@pulumi/pulumi";
import { Component } from "../component";
import { KvKeys } from "./providers/kv-keys";
import {
  buildKvNamespace,
  RouterBaseRouteArgs,
  updateKvRoutes,
  parsePattern,
} from "./router-base-route";
import { VisibleError } from "../error";
import { SsrSite } from "./ssr-site";
import { StaticSite } from "./static-site";
import { DistributionInvalidation } from "./providers/distribution-invalidation";
import { TanStackStart } from "./tan-stack-start.js";

export interface Args extends RouterBaseRouteArgs {
  /**
   * The site to route to.
   */
  site: Input<SsrSite | StaticSite>;
  /**
   * ID of the CloudFront distribution.
   */
  distributionId: Input<string>;
}

/**
 * The `RouterSiteRoute` component is internally used by the `Router` component
 * to add routes.
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `routeSite` method of the `Router` component.
 */
export class RouterSiteRoute extends Component {
  constructor(name: string, args: Args, opts?: ComponentResourceOptions) {
    super(__pulumiType, name, args, opts);

    const self = this;

    output(args.site).cdnData.apply((cdnData) => {
      // Site is not deployed (ie. dev mode)
      if (!cdnData) return;

      all([args.pattern, args.site, cdnData]).apply(
        ([pattern, site, cdnData]) => {
          const patternData = parsePattern(pattern);

          // Check if the site base path matches the route pattern
          if (patternData.path !== "/") {
            if (site.constructor.name === "TanStackStart")
              throw new VisibleError(
                `TanStack Start can only be routed from the root "/" and does not currently support base paths. Follow this thread on TanStack Discord to track progress: https://discord.com/channels/719702312431386674/1351676051964690452`,
              );

            if (!cdnData.base)
              throw new VisibleError(
                `No base path found for site. You must configure the base path to match the route pattern "${patternData.path}".`,
              );

            if (!cdnData.base.startsWith(patternData.path))
              throw new VisibleError(
                `The site base path "${cdnData.base}" must start with the route pattern "${patternData.path}".`,
              );
          }

          const namespace = buildKvNamespace(name);
          const kvUpdated = createKvRouteDataOverride();
          const kvRoutesUpdated = updateKvRoutes(
            name,
            args,
            self,
            "site",
            namespace,
            patternData,
          );
          createInvalidation();

          function createKvRouteDataOverride() {
            return new KvKeys(
              `${name}KvKeys`,
              {
                store: args.store,
                namespace,
                entries: cdnData.entries,
                purge: cdnData.purge,
              },
              { parent: self },
            );
          }

          function createInvalidation() {
            if (!cdnData.invalidation) return;

            new DistributionInvalidation(
              `${name}Invalidation`,
              {
                distributionId: args.distributionId,
                paths: cdnData.invalidation.paths,
                version: cdnData.invalidation.version,
                wait: cdnData.invalidation.wait,
              },
              {
                parent: self,
                dependsOn: [
                  ...cdnData.invalidationDependsOn,
                  kvUpdated,
                  kvRoutesUpdated,
                ],
              },
            );
          }
        },
      );
    });
  }
}

const __pulumiType = "sst:aws:RouterSiteRoute";
// @ts-expect-error
RouterSiteRoute.__pulumiType = __pulumiType;
