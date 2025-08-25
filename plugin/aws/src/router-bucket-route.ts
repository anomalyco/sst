import * as sst from "sst-plugin";
import { toSeconds } from "./util/duration.js";
import { Bucket } from "./bucket.js";
import {
  RouterBaseRouteArgs,
  parsePattern,
  buildKvNamespace,
  createKvRouteData,
  updateKvRoutes,
} from "./router-base-route.js";
import { RouterBucketRouteArgs } from "./router.js";
import { AWSComponent } from "./component.js";

export interface Args extends RouterBaseRouteArgs {
  /**
   * The bucket to route to.
   */
  bucket: sst.Input<Bucket>;
  /**
   * Additional arguments for the route.
   */
  routeArgs?: sst.Input<RouterBucketRouteArgs>;
}

/**
 * The `RouterBucketRoute` component is internally used by the `Router` component
 * to add routes.
 *
 * :::note
 * This component is not intended to be created directly.
 * :::
 *
 * You'll find this component returned by the `routeBucket` method of the `Router` component.
 */
export class RouterBucketRoute extends AWSComponent {
  constructor(name: string, args: Args, opts?: sst.ComponentOptions) {
    super(__pulumiType, name, args, opts);

    const self = this;

    sst
      .resolve([args.pattern, args.routeArgs])
      .apply(([pattern, routeArgs]) => {
        const patternData = parsePattern(pattern);
        const namespace = buildKvNamespace(name);
        createKvRouteData(name, args, self, namespace, {
          domain: sst.output(args.bucket).nodes.bucket.bucketRegionalDomainName,
          rewrite: routeArgs?.rewrite,
          origin: {
            connectionAttempts: routeArgs?.connectionAttempts,
            timeouts: {
              connectionTimeout:
                routeArgs?.connectionTimeout &&
                toSeconds(routeArgs?.connectionTimeout),
            },
          },
        });
        updateKvRoutes(name, args, self, "bucket", namespace, patternData);
      });
  }
}

const __pulumiType = "sst:aws:RouterBucketRoute";
// @ts-expect-error
RouterBucketRoute.__pulumiType = __pulumiType;
