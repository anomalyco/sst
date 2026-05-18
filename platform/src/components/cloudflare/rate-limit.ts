import {
  ComponentResourceOptions,
  output,
  type Input,
  type Output,
} from "@pulumi/pulumi";
import { Component } from "../component";
import { Link } from "../link";
import { binding } from "./binding";
import { toSeconds } from "../duration";
import { VisibleError } from "../error";

export interface RateLimitArgs {
  /**
   * A positive integer that uniquely defines this rate limiting namespace within your Cloudflare account.
   */
  namespaceId: Input<number>;
  /**
   * The number of allowed requests within the specified period of time.
   */
  limit: Input<number>;
  /**
   * The duration of the rate limit window. Must be either 10 seconds or 1 minute.
   */
  period: Input<"10 seconds" | "1 minute">;
}

/**
 * The `RateLimit` component lets you add a [Cloudflare Rate Limit](https://developers.cloudflare.com/workers/runtime-apis/bindings/rate-limit/) binding to
 * your app.
 *
 * @example
 *
 * #### Minimal example
 *
 * ```ts title="sst.config.ts"
 * const rateLimit = new sst.cloudflare.RateLimit("MyRateLimit", {
 *   namespaceId: 1001,
 *   limit: 100,
 *   period: "1 minute",
 * });
 * ```
 *
 * #### Link to a worker
 *
 * You can link RateLimit to a worker.
 *
 * ```ts {4} title="sst.config.ts"
 * new sst.cloudflare.Worker("MyWorker", {
 *   handler: "./index.ts",
 *   url: true
 *   link: [rateLimit],
 * });
 * ```
 *
 * Once linked, you can use the SDK to interact with the RateLimit binding.
 *
 * ```ts title="index.ts" {7}
 * import { Resource } from "sst/resource";
 *
 * export default {
 *   async fetch(req, env): Promise<Response> {
 *     const url = new URL(req.url);
 *
 *     const outcome = await Resource.MyRateLimit.limit({ key: url.pathname });
 *     if (!outcome.success) {
 *       return new Response(`Rate limit exceeded for ${url.pathname}`, { status: 429 });
 *     }
 *
 *     return new Response("OK", { status: 200 });
 *   }
 * }
 * ```
 */
export class RateLimit extends Component implements Link.Linkable {
  private _namespaceId: Output<string>;
  private _limit: Output<number>;
  private _period: Output<number>;

  constructor(
    name: string,
    args: RateLimitArgs,
    opts?: ComponentResourceOptions,
  ) {
    super(__pulumiType, name, args, opts);

    const namespaceId = normalizeNamespaceId();
    const limit = output(args.limit);
    const period = normalizePeriod();

    this._namespaceId = namespaceId;
    this._limit = limit;
    this._period = period;

    function normalizeNamespaceId() {
      return output(args.namespaceId).apply((namespaceId) => {
        if (!Number.isInteger(namespaceId) || namespaceId <= 0) {
          throw new VisibleError(
            "The `namespaceId` property must be a positive integer.",
          );
        }

        return namespaceId.toString();
      });
    }

    function normalizePeriod() {
      return output(args.period).apply(toSeconds);
    }
  }

  /**
   * A unique identifier for the rate limit namespace.
   */
  public get namespaceId() {
    return this._namespaceId;
  }

  /**
   * The number of allowed requests within the specified period of time.
   */
  public get limit() {
    return this._limit;
  }

  /**
   * The duration of the rate limit window, in seconds.
   */
  public get period() {
    return this._period;
  }

  /**
   * When you link a RateLimit binding, it will be available to the worker and you can
   * interact with it using its [API methods](https://developers.cloudflare.com/workers/runtime-apis/bindings/rate-limit/).
   *
   * @example
   * ```ts title="index.ts"
   * import { Resource } from "sst/resource";
   *
   * export default {
   *   async fetch(req, env): Promise<Response> {
   *     const url = new URL(req.url);
   *
   *     const outcome = await Resource.MyRateLimit.limit({ key: url.pathname });
   *     if (!outcome.success) {
   *       return new Response(`Rate limit exceeded for ${url.pathname}`, { status: 429 });
   *     }
   *
   *     return new Response("OK", { status: 200 });
   *   }
   * }
   * ```
   *
   * @internal
   */
  getSSTLink() {
    return {
      properties: {},
      include: [
        binding({
          type: "rateLimitBindings",
          properties: {
            namespaceId: this._namespaceId,
            simple: {
              limit: this._limit,
              period: this._period,
            },
          },
        }),
      ],
    };
  }
}

const __pulumiType = "sst:cloudflare:RateLimit";
// @ts-expect-error
RateLimit.__pulumiType = __pulumiType;
