import {
  ComponentResourceOptions,
  Output,
  all,
  interpolate,
  output,
} from "@pulumi/pulumi";
import { Component, Prettify, Transform, transform } from "../component.js";
import { dns as awsDns } from "./dns.js";
import { VisibleError } from "../error.js";
import { DnsValidatedCertificate } from "./dns-validated-certificate.js";
import { Link } from "../link.js";
import { URL_UNAVAILABLE } from "./linkable.js";
import {
  appautoscaling,
  ec2,
  ecs,
  getRegionOutput,
  iam,
  lb,
  servicediscovery,
} from "@pulumi/aws";
import { Vpc } from "./vpc.js";
import { DevCommand } from "../experimental/dev-command.js";
import { DurationMinutes, toSeconds } from "../duration.js";
import { Input } from "../input.js";
import {
  FargateBaseArgs,
  FargateContainerArgs,
  createExecutionRole,
  createTaskDefinition,
  createTaskRole,
  normalizeArchitecture,
  normalizeContainers,
  normalizeCpu,
  normalizeMemory,
  normalizeStorage,
} from "./fargate.js";
import { Dns } from "../dns.js";
import { hashStringToPrettyString } from "../naming.js";
import { Alb } from "./alb.js";
import { listenerKey, targetKey } from "./helpers/load-balancer.js";

type Port = `${number}/${"http" | "https" | "tcp" | "udp" | "tcp_udp" | "tls"}`;

interface ServiceRules {
  /**
   * The port and protocol the service listens on. Uses the format `{port}/{protocol}`.
   *
   * @example
   * ```js
   * {
   *   listen: "80/http"
   * }
   * ```
   */
  listen: Input<Port>;
  /**
   * The port and protocol of the container the service forwards the traffic to. Uses the
   * format `{port}/{protocol}`.
   *
   * @example
   * ```js
   * {
   *   forward: "80/http"
   * }
   * ```
   * @default The same port and protocol as `listen`.
   */
  forward?: Input<Port>;
  /**
   * The name of the container to forward the traffic to. This maps to the `name` defined in the
   * `container` prop.
   *
   * You only need this if there's more than one container. If there's only one container, the
   * traffic is automatically forwarded there.
   */
  container?: Input<string>;
  /**
   * The port and protocol to redirect the traffic to. Uses the format `{port}/{protocol}`.
   *
   * @example
   * ```js
   * {
   *   redirect: "80/http"
   * }
   * ```
   */
  redirect?: Input<Port>;
  /**
   * @deprecated Use `conditions.path` instead.
   */
  path?: Input<string>;
  /**
   * The conditions for the redirect. Only applicable to `http` and `https` protocols.
   */
  conditions?: Input<{
    /**
     * Configure path-based routing. Only requests matching the path are forwarded to
     * the container.
     *
     * ```js
     * {
     *   path: "/api/*"
     * }
     * ```
     *
     * The path pattern is case-sensitive, supports wildcards, and can be up to 128
     * characters.
     * - `*` matches 0 or more characters. For example, `/api/*` matches `/api/` or
     *   `/api/orders`.
     * - `?` matches exactly 1 character. For example, `/api/?.png` matches `/api/a.png`.
     *
     * @default Requests to all paths are forwarded.
     */
    path?: Input<string>;
    /**
     * Configure query string based routing. Only requests matching one of the query
     * string conditions are forwarded to the container.
     *
     * Takes a list of `key`, the name of the query string parameter, and `value` pairs.
     * Where `value` is the value of the query string parameter. But it can be a pattern as well.
     *
     * If multiple `key` and `value` pairs are provided, it'll match requests with **any** of the
     * query string parameters.
     *
     * @default Query string is not checked when forwarding requests.
     *
     * @example
     *
     * For example, to match requests with query string `version=v1`.
     *
     * ```js
     * {
     *   query: [
     *     { key: "version", value: "v1" }
     *   ]
     * }
     * ```
     *
     * Or match requests with query string matching `env=test*`.
     *
     * ```js
     * {
     *   query: [
     *     { key: "env", value: "test*" }
     *   ]
     * }
     * ```
     *
     * Match requests with query string `version=v1` **or** `env=test*`.
     *
     * ```js
     * {
     *   query: [
     *     { key: "version", value: "v1" },
     *     { key: "env", value: "test*" }
     *   ]
     * }
     * ```
     *
     * Match requests with any query string key with value `example`.
     *
     * ```js
     * {
     *   query: [
     *     { value: "example" }
     *   ]
     * }
     * ```
     */
    query?: Input<
      Input<{
        /**
         * The name of the query string parameter.
         */
        key?: Input<string>;
        /**
         * The value of the query string parameter.
         *
         * If no `key` is provided, it'll match any request where a query string parameter with
         * the given value exists.
         */
        value: Input<string>;
      }>[]
    >;
    /**
     * Configure header based routing. Only requests matching the header
     * name and values are forwarded to the container.
     *
     * Both the header name and values are case insensitive.
     *
     * @default Header is not checked when forwarding requests.
     *
     * @example
     *
     * For example, if you specify `X-Custom-Header` as the name and `Value1`
     * as a value, it will match requests with the header
     * `x-custom-header: value1` as well.
     *
     * ```js
     * {
     *   header: {
     *     name: "X-Custom-Header",
     *     values: ["Value1", "Value2", "Prefix*"]
     *   }
     * }
     * ```
     */
    header?: Input<{
      /**
       * The name of the HTTP header field to check. This is case-insensitive.
       */
      name: Input<string>;

      /**
       * The values to match against the header value. The rule matches if the
       * request header matches any of these values. Values are case-insensitive
       * and support wildcards (`*` and `?`) for pattern matching.
       */
      values: Input<Input<string>>[];
    }>;
  }>;
}

type AlbPort = `${number}/${"http" | "https"}`;

export interface ServiceAlbRule {
  /**
   * The port and protocol to listen on, in `{port}/{protocol}` format. Must match a listener on the ALB.
   *
   * @example
   * ```js
   * {
   *   listen: "443/https"
   * }
   * ```
   */
  listen: AlbPort;
  /**
   * The container port and protocol to forward traffic to. Uses the format `{port}/{protocol}`.
   *
   * The protocol must match what the container actually speaks — using `"3000/https"` when
   * the container speaks HTTP will cause health check failures.
   *
   * @example
   * ```js
   * {
   *   forward: "8080/http"
   * }
   * ```
   */
  forward: AlbPort;
  /**
   * The name of the container to forward the traffic to. Required when multiple containers
   * are configured.
   */
  container?: string;
  /**
   * The conditions for the listener rule. At least one condition (path, query, or header)
   * must be specified. The ALB owns the default action — services only add conditional rules.
   *
   * @example
   * ```js
   * {
   *   conditions: {
   *     path: "/api/*"
   *   }
   * }
   * ```
   */
  conditions: {
    /**
     * Path pattern to match. Supports wildcards (`*` and `?`).
     */
    path?: Input<string>;
    /**
     * Query string conditions.
     */
    query?: Input<
      Input<{
        key?: Input<string>;
        value: Input<string>;
      }>[]
    >;
    /**
     * HTTP header condition.
     */
    header?: Input<{
      name: Input<string>;
      values: Input<Input<string>>[];
    }>;
  };
  /**
   * Explicit priority for the listener rule (1–50000).
   * Must be unique per listener across ALL services sharing the ALB.
   * Use non-overlapping ranges per service (e.g., Service A: 100-199, Service B: 200-299).
   *
   * @example
   * ```js
   * {
   *   priority: 100
   * }
   * ```
   */
  priority: number;
}

interface ServiceContainerArgs extends FargateContainerArgs {
  /**
   * Configure the health check for the container. Same as the top-level
   * [`health`](#health).
   */
  health?: ServiceArgs["health"];
  /**
   * Configure how this container works in `sst dev`. Same as the top-level
   * [`dev`](#dev).
   */
  dev?: {
    /**
     * The command that `sst dev` runs to start this in dev mode. Same as the top-level
     * [`dev.command`](#dev-command).
     */
    command: Input<string>;
    /**
     * Configure if you want to automatically start this when `sst dev` starts. Same as the
     * top-level [`dev.autostart`](#dev-autostart).
     */
    autostart?: Input<boolean>;
    /**
     * Change the directory from where the `command` is run. Same as the top-level
     * [`dev.directory`](#dev-directory).
     */
    directory?: Input<string>;
  };
}

export interface ServiceArgs extends FargateBaseArgs {
  /**
   * Configure how this component works in `sst dev`.
   *
   * :::note
   * In `sst dev` your service is not deployed.
   * :::
   *
   * By default, your service in not deployed in `sst dev`. Instead, you can set the
   * `dev.command` and it'll be started locally in a separate tab in the
   * `sst dev` multiplexer. Read more about [`sst dev`](/docs/reference/cli/#dev).
   *
   * This makes it so that the container doesn't have to be redeployed on every change. To
   * disable this and deploy your service in `sst dev`, pass in `false`.
   */
  dev?:
    | false
    | {
        /**
         * The `url` when this is running in dev mode.
         *
         * Since this component is not deployed in `sst dev`, there is no real URL. But if you are
         * using this component's `url` or linking to this component's `url`, it can be useful to
         * have a placeholder URL. It avoids having to handle it being `undefined`.
         * @default `"http://url-unavailable-in-dev.mode"`
         */
        url?: Input<string>;
        /**
         * The command that `sst dev` runs to start this in dev mode. This is the command you run
         * when you want to run your service locally.
         */
        command?: Input<string>;
        /**
         * Configure if you want to automatically start this when `sst dev` starts. You can still
         * start it manually later.
         * @default `true`
         */
        autostart?: Input<boolean>;
        /**
         * Change the directory from where the `command` is run.
         * @default Uses the `image.dockerfile` path
         */
        directory?: Input<string>;
      };
  /**
   * Configure a public endpoint for the service. When configured, a load balancer
   * will be created to route traffic to the containers. By default, the endpoint is an
   * auto-generated load balancer URL.
   *
   * You can also add a custom domain for the public endpoint.
   * @deprecated Use `loadBalancer` instead.
   * @example
   *
   * ```js
   * {
   *   public: {
   *     domain: "example.com",
   *     rules: [
   *       { listen: "80/http" },
   *       { listen: "443/https", forward: "80/http" }
   *     ]
   *   }
   * }
   * ```
   */
  public?: Input<{
    /**
     * Set a custom domain for your public endpoint.
     *
     * Automatically manages domains hosted on AWS Route 53, Cloudflare, and Vercel. For other
     * providers, you'll need to pass in a `cert` that validates domain ownership and add the
     * DNS records.
     *
     * :::tip
     * Built-in support for AWS Route 53, Cloudflare, and Vercel. And manual setup for other
     * providers.
     * :::
     *
     * @example
     *
     * By default this assumes the domain is hosted on Route 53.
     *
     * ```js
     * {
     *   domain: "example.com"
     * }
     * ```
     *
     * For domains hosted on Cloudflare.
     *
     * ```js
     * {
     *   domain: {
     *     name: "example.com",
     *     dns: sst.cloudflare.dns()
     *   }
     * }
     * ```
     */
    domain?: Input<
      | string
      | {
          /**
           * The custom domain you want to use.
           *
           * @example
           * ```js
           * {
           *   domain: {
           *     name: "example.com"
           *   }
           * }
           * ```
           *
           * Can also include subdomains based on the current stage.
           *
           * ```js
           * {
           *   domain: {
           *     name: `${$app.stage}.example.com`
           *   }
           * }
           * ```
           */
          name: Input<string>;
          /**
           * Alias domains that should be used.
           *
           * @example
           * ```js {4}
           * {
           *   domain: {
           *     name: "app1.example.com",
           *     aliases: ["app2.example.com"]
           *   }
           * }
           * ```
           */
          aliases?: Input<string[]>;
          /**
           * The ARN of an ACM (AWS Certificate Manager) certificate that proves ownership of the
           * domain. By default, a certificate is created and validated automatically.
           *
           * :::tip
           * You need to pass in a `cert` for domains that are not hosted on supported `dns` providers.
           * :::
           *
           * To manually set up a domain on an unsupported provider, you'll need to:
           *
           * 1. [Validate that you own the domain](https://docs.aws.amazon.com/acm/latest/userguide/domain-ownership-validation.html) by creating an ACM certificate. You can either validate it by setting a DNS record or by verifying an email sent to the domain owner.
           * 2. Once validated, set the certificate ARN as the `cert` and set `dns` to `false`.
           * 3. Add the DNS records in your provider to point to the load balancer endpoint.
           *
           * @example
           * ```js
           * {
           *   domain: {
           *     name: "example.com",
           *     dns: false,
           *     cert: "arn:aws:acm:us-east-1:112233445566:certificate/3a958790-8878-4cdc-a396-06d95064cf63"
           *   }
           * }
           * ```
           */
          cert?: Input<string>;
          /**
           * The DNS provider to use for the domain. Defaults to the AWS.
           *
           * Takes an adapter that can create the DNS records on the provider. This can automate
           * validating the domain and setting up the DNS routing.
           *
           * Supports Route 53, Cloudflare, and Vercel adapters. For other providers, you'll need
           * to set `dns` to `false` and pass in a certificate validating ownership via `cert`.
           *
           * @default `sst.aws.dns`
           *
           * @example
           *
           * Specify the hosted zone ID for the Route 53 domain.
           *
           * ```js
           * {
           *   domain: {
           *     name: "example.com",
           *     dns: sst.aws.dns({
           *       zone: "Z2FDTNDATAQYW2"
           *     })
           *   }
           * }
           * ```
           *
           * Use a domain hosted on Cloudflare, needs the Cloudflare provider.
           *
           * ```js
           * {
           *   domain: {
           *     name: "example.com",
           *     dns: sst.cloudflare.dns()
           *   }
           * }
           * ```
           *
           * Use a domain hosted on Vercel, needs the Vercel provider.
           *
           * ```js
           * {
           *   domain: {
           *     name: "example.com",
           *     dns: sst.vercel.dns()
           *   }
           * }
           * ```
           */
          dns?: Input<false | (Dns & {})>;
        }
    >;
    /** @deprecated Use `rules` instead. */
    ports?: Input<Prettify<ServiceRules>[]>;
    /**
     * Configure the mapping for the ports the public endpoint listens to and forwards to
     * the service.
     * This supports two types of protocols:
     *
     * 1. Application Layer Protocols: `http` and `https`. This'll create an [Application Load Balancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/introduction.html).
     * 2. Network Layer Protocols: `tcp`, `udp`, `tcp_udp`, and `tls`. This'll create a [Network Load Balancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/network/introduction.html).
     *
     * :::note
     * If you are listening  on `https` or `tls`, you need to specify a custom `public.domain`.
     * :::
     *
     * You can **not** configure both application and network layer protocols for the same
     * service.
     *
     * @example
     * Here we are listening on port `80` and forwarding it to the service on port `8080`.
     * ```js
     * {
     *   public: {
     *     rules: [
     *       { listen: "80/http", forward: "8080/http" }
     *     ]
     *   }
     * }
     * ```
     *
     * The `forward` port and protocol defaults to the `listen` port and protocol. So in this
     * case both are `80/http`.
     *
     * ```js
     * {
     *   public: {
     *     rules: [
     *       { listen: "80/http" }
     *     ]
     *   }
     * }
     * ```
     *
     * If multiple containers are configured via the `containers` argument, you need to
     * specify which container the traffic should be forwarded to.
     *
     * ```js
     * {
     *   public: {
     *     rules: [
     *       { listen: "80/http", container: "app" },
     *       { listen: "8000/http", container: "admin" },
     *     ]
     *   }
     * }
     * ```
     */
    rules?: Input<Prettify<ServiceRules>[]>;
  }>;
  /**
   * Configure a load balancer to route traffic to the containers.
   *
   * While you can expose a service through API Gateway, it's better to use a load balancer
   * for most traditional web applications. It is more expensive to start but at higher
   * levels of traffic it ends up being more cost effective.
   *
   * Also, if you need to listen on network layer protocols like `tcp` or `udp`, you have to
   * expose it through a load balancer.
   *
   * By default, the endpoint is an auto-generated load balancer URL. You can also add a
   * custom domain for the endpoint.
   *
   * @default Load balancer is not created
   * @example
   *
   * ```js
   * {
   *   loadBalancer: {
   *     domain: "example.com",
   *     rules: [
   *       { listen: "80/http", redirect: "443/https" },
   *       { listen: "443/https", forward: "80/http" }
   *     ]
   *   }
   * }
   * ```
   */
  loadBalancer?: Input<
    | {
    /**
     * Configure if the load balancer should be public or private.
     *
     * When set to `false`, the load balancer endpoint will only be accessible within the
     * VPC.
     *
     * @default `true`
     */
    public?: Input<boolean>;
    /**
     * Set a custom domain for your load balancer endpoint.
     *
     * Automatically manages domains hosted on AWS Route 53, Cloudflare, and Vercel. For other
     * providers, you'll need to pass in a `cert` that validates domain ownership and add the
     * DNS records.
     *
     * :::tip
     * Built-in support for AWS Route 53, Cloudflare, and Vercel. And manual setup for other
     * providers.
     * :::
     *
     * @example
     *
     * By default this assumes the domain is hosted on Route 53.
     *
     * ```js
     * {
     *   domain: "example.com"
     * }
     * ```
     *
     * For domains hosted on Cloudflare.
     *
     * ```js
     * {
     *   domain: {
     *     name: "example.com",
     *     dns: sst.cloudflare.dns()
     *   }
     * }
     * ```
     */
    domain?: Input<
      | string
      | {
          /**
           * The custom domain you want to use.
           *
           * @example
           * ```js
           * {
           *   domain: {
           *     name: "example.com"
           *   }
           * }
           * ```
           *
           * Can also include subdomains based on the current stage.
           *
           * ```js
           * {
           *   domain: {
           *     name: `${$app.stage}.example.com`
           *   }
           * }
           * ```
           *
           * Wildcard domains are supported.
           *
           * ```js
           * {
           *   domain: {
           *     name: "*.example.com"
           *   }
           * }
           * ```
           */
          name: Input<string>;
          /**
           * Alias domains that should be used.
           *
           * @example
           * ```js {4}
           * {
           *   domain: {
           *     name: "app1.example.com",
           *     aliases: ["app2.example.com"]
           *   }
           * }
           * ```
           */
          aliases?: Input<string[]>;
          /**
           * The ARN of an ACM (AWS Certificate Manager) certificate that proves ownership of the
           * domain. By default, a certificate is created and validated automatically.
           *
           * :::tip
           * You need to pass in a `cert` for domains that are not hosted on supported `dns` providers.
           * :::
           *
           * To manually set up a domain on an unsupported provider, you'll need to:
           *
           * 1. [Validate that you own the domain](https://docs.aws.amazon.com/acm/latest/userguide/domain-ownership-validation.html) by creating an ACM certificate. You can either validate it by setting a DNS record or by verifying an email sent to the domain owner.
           * 2. Once validated, set the certificate ARN as the `cert` and set `dns` to `false`.
           * 3. Add the DNS records in your provider to point to the load balancer endpoint.
           *
           * @example
           * ```js
           * {
           *   domain: {
           *     name: "example.com",
           *     dns: false,
           *     cert: "arn:aws:acm:us-east-1:112233445566:certificate/3a958790-8878-4cdc-a396-06d95064cf63"
           *   }
           * }
           * ```
           */
          cert?: Input<string>;
          /**
           * The DNS provider to use for the domain. Defaults to the AWS.
           *
           * Takes an adapter that can create the DNS records on the provider. This can automate
           * validating the domain and setting up the DNS routing.
           *
           * Supports Route 53, Cloudflare, and Vercel adapters. For other providers, you'll need
           * to set `dns` to `false` and pass in a certificate validating ownership via `cert`.
           *
           * @default `sst.aws.dns`
           *
           * @example
           *
           * Specify the hosted zone ID for the Route 53 domain.
           *
           * ```js
           * {
           *   domain: {
           *     name: "example.com",
           *     dns: sst.aws.dns({
           *       zone: "Z2FDTNDATAQYW2"
           *     })
           *   }
           * }
           * ```
           *
           * Use a domain hosted on Cloudflare, needs the Cloudflare provider.
           *
           * ```js
           * {
           *   domain: {
           *     name: "example.com",
           *     dns: sst.cloudflare.dns()
           *   }
           * }
           * ```
           *
           * Use a domain hosted on Vercel, needs the Vercel provider.
           *
           * ```js
           * {
           *   domain: {
           *     name: "example.com",
           *     dns: sst.vercel.dns()
           *   }
           * }
           * ```
           */
          dns?: Input<false | (Dns & {})>;
        }
    >;
    /** @deprecated Use `rules` instead. */
    ports?: Input<Prettify<ServiceRules>[]>;
    /**
     * Configure the mapping for the ports the load balancer listens to, forwards, or redirects to
     * the service.
     * This supports two types of protocols:
     *
     * 1. Application Layer Protocols: `http` and `https`. This'll create an [Application Load Balancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/introduction.html).
     * 2. Network Layer Protocols: `tcp`, `udp`, `tcp_udp`, and `tls`. This'll create a [Network Load Balancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/network/introduction.html).
     *
     * :::note
     * If you want to listen on `https` or `tls`, you need to specify a custom
     * `loadBalancer.domain`.
     * :::
     *
     * You **can not configure** both application and network layer protocols for the same
     * service.
     *
     * @example
     * Here we are listening on port `80` and forwarding it to the service on port `8080`.
     * ```js
     * {
     *   rules: [
     *     { listen: "80/http", forward: "8080/http" }
     *   ]
     * }
     * ```
     *
     * The `forward` port and protocol defaults to the `listen` port and protocol. So in this
     * case both are `80/http`.
     *
     * ```js
     * {
     *   rules: [
     *     { listen: "80/http" }
     *   ]
     * }
     * ```
     *
     * If multiple containers are configured via the `containers` argument, you need to
     * specify which container the traffic should be forwarded to.
     *
     * ```js
     * {
     *   rules: [
     *     { listen: "80/http", container: "app" },
     *     { listen: "8000/http", container: "admin" }
     *   ]
     * }
     * ```
     *
     * You can also route the same port to multiple containers via path-based routing.
     *
     * ```js
     * {
     *   rules: [
     *     {
     *       listen: "80/http",
     *       container: "app",
     *       conditions: { path: "/api/*" }
     *     },
     *     {
     *       listen: "80/http",
     *       container: "admin",
     *       conditions: { path: "/admin/*" }
     *     }
     *   ]
     * }
     * ```
     *
     * Additionally, you can redirect traffic from one port to another. This is
     * commonly used to redirect http to https.
     *
     * ```js
     * {
     *   rules: [
     *     { listen: "80/http", redirect: "443/https" },
     *     { listen: "443/https", forward: "80/http" }
     *   ]
     * }
     * ```
     */
    rules?: Input<Prettify<ServiceRules>[]>;
    /**
     * Configure the health check that the load balancer runs on your containers.
     *
     * :::tip
     * This health check is different from the [`health`](#health) check.
     * :::
     *
     * This health check is run by the load balancer. While, `health` is run by ECS. This
     * cannot be disabled if you are using a load balancer. While the other is off by default.
     *
     * Since this cannot be disabled, here are some tips on how to debug an unhealthy
     * health check.
     *
     * <details>
     * <summary>How to debug a load balancer health check</summary>
     *
     * If you notice a `Unhealthy: Health checks failed` error, it's because the health
     * check has failed. When it fails, the load balancer will terminate the containers,
     * causing any requests to fail.
     *
     * Here's how to debug it:
     *
     * 1. Verify the health check path.
     *
     *    By default, the load balancer checks the `/` path. Ensure it's accessible in your
     *    containers. If your application runs on a different path, then update the path in
     *    the health check config accordingly.
     *
     * 2. Confirm the containers are operational.
     *
     *    Navigate to **ECS console** > select the **cluster** > go to the **Tasks tab** >
     *    choose **Any desired status** under the **Filter desired status** dropdown > select
     *    a task and check for errors under the **Logs tab**. If it has error that means that
     *    the container failed to start.
     *
     * 3. If the container was terminated by the load balancer while still starting up, try
     *    increasing the health check interval and timeout.
     * </details>
     *
     * For `http` and `https` the default is:
     *
     * ```js
     * {
     *   path: "/",
     *   healthyThreshold: 5,
     *   successCodes: "200",
     *   timeout: "5 seconds",
     *   unhealthyThreshold: 2,
     *   interval: "30 seconds"
     * }
     * ```
     *
     * For `tcp` and `udp` the default is:
     *
     * ```js
     * {
     *   healthyThreshold: 5,
     *   timeout: "6 seconds",
     *   unhealthyThreshold: 2,
     *   interval: "30 seconds"
     * }
     * ```
     *
     * @example
     *
     * To configure the health check, we use the _port/protocol_ format. Here we are
     * configuring a health check that pings the `/health` path on port `8080`
     * every 10 seconds.
     *
     * ```js
     * {
     *   rules: [
     *     { listen: "80/http", forward: "8080/http" }
     *   ],
     *   health: {
     *     "8080/http": {
     *       path: "/health",
     *       interval: "10 seconds"
     *     }
     *   }
     * }
     * ```
     *
     */
    health?: Input<
      Record<
        Port,
        Input<{
          /**
           * The URL path to ping on the service for health checks. Only applicable to
           * `http` and `https` protocols.
           * @default `"/"`
           */
          path?: Input<string>;
          /**
           * The time period between each health check request. Must be between `5 seconds`
           * and `300 seconds`.
           * @default `"30 seconds"`
           */
          interval?: Input<DurationMinutes>;
          /**
           * The timeout for each health check request. If no response is received within this
           * time, it is considered failed. Must be between `2 seconds` and `120 seconds`.
           * @default `"5 seconds"`
           */
          timeout?: Input<DurationMinutes>;
          /**
           * The number of consecutive successful health check requests required to consider the
           * target healthy. Must be between 2 and 10.
           * @default `5`
           */
          healthyThreshold?: Input<number>;
          /**
           * The number of consecutive failed health check requests required to consider the
           * target unhealthy. Must be between 2 and 10.
           * @default `2`
           */
          unhealthyThreshold?: Input<number>;
          /**
           * One or more HTTP response codes the health check treats as successful. Only
           * applicable to `http` and `https` protocols.
           *
           * @default `"200"`
           * @example
           * ```js
           * {
           *   successCodes: "200-299"
           * }
           * ```
           */
          successCodes?: Input<string>;
        }>
      >
    >;
  }
    | {
    /**
     * The `Alb` instance to attach this service to. When provided, the service creates
     * target groups and listener rules on the shared ALB instead of creating its own
     * load balancer.
     *
     * ECS tasks use the VPC's default security group, which allows all traffic within the
     * VPC CIDR. For tighter security, add an explicit security group ingress rule from the
     * ALB's security group using `transform`.
     *
     * @example
     * ```js
     * {
     *   loadBalancer: {
     *     instance: alb,
     *     rules: [
     *       { listen: "443/https", forward: "8080/http", conditions: { path: "/api/*" }, priority: 100 }
     *     ]
     *   }
     * }
     * ```
     */
    instance: Alb;
    /**
     * The rules for routing traffic from the ALB to this service's containers.
     * Each rule must have explicit conditions and priority.
     */
    rules: Prettify<ServiceAlbRule>[];
    /**
     * Configure health checks for the target groups. Uses the same format as the inline
     * health check config, keyed by `{port}/{protocol}`.
     */
    health?: Record<
      AlbPort,
      Input<{
        path?: Input<string>;
        interval?: Input<DurationMinutes>;
        timeout?: Input<DurationMinutes>;
        healthyThreshold?: Input<number>;
        unhealthyThreshold?: Input<number>;
        successCodes?: Input<string>;
      }>
    >;
  }
  >;
  /**
   * Configure the CloudMap service registry for the service.
   *
   * This creates an `srv` record in the CloudMap service. This is needed if you want to connect
   * an `ApiGatewayV2` VPC link to the service.
   *
   * API Gateway will forward requests to the given port on the service.
   *
   * @example
   * ```js
   * {
   *   serviceRegistry: {
   *     port: 80
   *   }
   * }
   * ```
   */
  serviceRegistry?: Input<{
    /**
     * The port in the service to forward requests to.
     */
    port: number;
  }>;
  /**
   * Configure the service to automatically scale up or down based on the CPU or memory
   * utilization of a container. By default, scaling is disabled and the service will run
   * in a single container.
   *
   * @default `{ min: 1, max: 1 }`
   *
   * @example
   * ```js
   * {
   *   scaling: {
   *     min: 4,
   *     max: 16,
   *     cpuUtilization: 50,
   *     memoryUtilization: 50
   *   }
   * }
   * ```
   */
  scaling?: Input<{
    /**
     * The minimum number of containers to scale down to.
     * @default `1`
     * @example
     * ```js
     * {
     *   scaling: {
     *     min: 4
     *   }
     * }
     * ```
     */
    min?: Input<number>;
    /**
     * The maximum number of containers to scale up to.
     * @default `1`
     * @example
     * ```js
     * {
     *   scaling: {
     *     max: 16
     *   }
     * }
     * ```
     */
    max?: Input<number>;
    /**
     * The target CPU utilization percentage to scale up or down. It'll scale up
     * when the CPU utilization is above the target and scale down when it's below the target.
     * @default `70`
     * @example
     * ```js
     * {
     *   scaling: {
     *     cpuUtilization: 50
     *   }
     * }
     * ```
     */
    cpuUtilization?: Input<false | number>;
    /**
     * The target memory utilization percentage to scale up or down. It'll scale up
     * when the memory utilization is above the target and scale down when it's below the target.
     * @default `70`
     * @example
     * ```js
     * {
     *   scaling: {
     *     memoryUtilization: 50
     *   }
     * }
     * ```
     */
    memoryUtilization?: Input<false | number>;
    /**
     * The target request count to scale up or down. It'll scale up when the request count is
     * above the target and scale down when it's below the target.
     * @default `false`
     * @example
     * ```js
     * {
     *   scaling: {
     *     requestCount: 1500
     *   }
     * }
     * ```
     */
    requestCount?: Input<false | number>;
    /**
     * The amount of time, in seconds, after a scale-in activity completes before another scale-in activity can start.
     * This prevents the auto scaler from removing too many tasks too quickly.
     * @example
     * ```js
     * {
     *   scaling: {
     *     scaleInCooldown: "60 seconds"
     *   }
     * }
     * ```
     */
    scaleInCooldown?: Input<DurationMinutes>;
    /**
     * The amount of time, in seconds, after a scale-out activity completes before another scale-out activity can start.
     * This prevents the auto scaler from adding too many tasks too quickly.
     * @example
     * ```js
     * {
     *   scaling: {
     *     scaleOutCooldown: "60 seconds"
     *   }
     * }
     * ```
     */
    scaleOutCooldown?: Input<DurationMinutes>;
  }>;
  /**
   * Configure the capacity provider; regular Fargate or Fargate Spot, for this service.
   *
   * :::tip
   * Fargate Spot is a good option for dev or PR environments.
   * :::
   *
   * Fargate Spot allows you to run containers on spare AWS capacity at around 50% discount
   * compared to regular Fargate. [Learn more about Fargate
   * pricing](https://aws.amazon.com/fargate/pricing/).
   *
   * :::note
   * AWS might shut down Fargate Spot instances to reclaim capacity.
   * :::
   *
   * There are a couple of caveats:
   *
   * 1. AWS may reclaim this capacity and **turn off your service** after a two-minute warning.
   *    This is rare, but it can happen.
   * 2. If there's no spare capacity, you'll **get an error**.
   *
   * This makes Fargate Spot a good option for dev or PR environments. You can set this using.
   *
   * ```js
   * {
   *   capacity: "spot"
   * }
   * ```
   *
   * You can also configure the % of regular vs spot capacity you want through the `weight` prop.
   * And optionally set the `base` or first X number of tasks that'll be started using a given
   * capacity.
   *
   * For example, the `base: 1` says that the first task uses regular Fargate, and from that
   * point on there will be an even split between the capacity providers.
   *
   * ```js
   * {
   *   capacity: {
   *     fargate: { weight: 1, base: 1 },
   *     spot: { weight: 1 }
   *   }
   * }
   * ```
   *
   * The `base` works in tandem with the `scaling` prop. So setting `base` to X doesn't mean
   * it'll start those tasks right away. It means that as your service scales up, according to
   * the `scaling` prop, it'll ensure that the first X tasks will be with the given capacity.
   *
   * :::caution
   * Changing `capacity` requires taking down and recreating the ECS service.
   * :::
   *
   * And this is why you can only set the `base` for only one capacity provider. So you
   * are not allowed to do the following.
   *
   * ```js
   * {
   *   capacity: {
   *     fargate: { weight: 1, base: 1 },
   *     // This will give you an error
   *     spot: { weight: 1, base: 1 }
   *   }
   * }
   * ```
   *
   * When you change the `capacity`, the ECS service is terminated and recreated. This will
   * cause some temporary downtime.
   *
   * @default Regular Fargate
   *
   * @example
   *
   * Here are some examples settings.
   *
   * - Use only Fargate Spot.
   *
   *   ```js
   *   {
   *     capacity: "spot"
   *   }
   *   ```
   * - Use 50% regular Fargate and 50% Fargate Spot.
   *
   *   ```js
   *   {
   *     capacity: {
   *       fargate: { weight: 1 },
   *       spot: { weight: 1 }
   *     }
   *   }
   *   ```
   * - Use 50% regular Fargate and 50% Fargate Spot. And ensure that the first 2 tasks use
   *   regular Fargate.
   *
   *   ```js
   *   {
   *     capacity: {
   *       fargate: { weight: 1, base: 2 },
   *       spot: { weight: 1 }
   *     }
   *   }
   *   ```
   */
  capacity?: Input<
    | "spot"
    | {
        /**
         * Configure how the regular Fargate capacity is allocated.
         */
        fargate?: Input<{
          /**
           * Start the first `base` number of tasks with the given capacity.
           *
           * :::caution
           * You can only specify `base` for one capacity provider.
           * :::
           */
          base?: Input<number>;
          /**
           * Ensure the given ratio of tasks are started for this capacity.
           */
          weight: Input<number>;
        }>;
        /**
         * Configure how the Fargate spot capacity is allocated.
         */
        spot?: Input<{
          /**
           * Start the first `base` number of tasks with the given capacity.
           *
           * :::caution
           * You can only specify `base` for one capacity provider.
           * :::
           */
          base?: Input<number>;
          /**
           * Ensure the given ratio of tasks are started for this capacity.
           */
          weight: Input<number>;
        }>;
      }
  >;
  /**
   * Configure the health check that ECS runs on your containers.
   *
   * :::tip
   * This health check is different from the [`loadBalancer.health`](#loadbalancer-health) check.
   * :::
   *
   * This health check is run by ECS. While, `loadBalancer.health` is run by the load balancer,
   * if you are using one. This is off by default. While the load balancer one
   * cannot be disabled.
   *
   * This config maps to the `HEALTHCHECK` parameter of the `docker run` command. Learn
   * more about [container health checks](https://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_HealthCheck.html).
   *
   * @default Health check is disabled
   * @example
   * ```js
   * {
   *   health: {
   *     command: ["CMD-SHELL", "curl -f http://localhost:3000/ || exit 1"],
   *     startPeriod: "60 seconds",
   *     timeout: "5 seconds",
   *     interval: "30 seconds",
   *     retries: 3
   *   }
   * }
   * ```
   */
  health?: Input<{
    /**
     * A string array representing the command that the container runs to determine if it is
     * healthy.
     *
     * It must start with `CMD` to run the command arguments directly. Or `CMD-SHELL` to run
     * the command with the container's default shell.
     *
     * @example
     * ```js
     * {
     *   command: ["CMD-SHELL", "curl -f http://localhost:3000/ || exit 1"]
     * }
     * ```
     */
    command: Input<string[]>;
    /**
     * The grace period to provide containers time to bootstrap before failed health checks
     * count towards the maximum number of retries. Must be between `0 seconds` and
     * `300 seconds`.
     * @default `"0 seconds"`
     */
    startPeriod?: Input<DurationMinutes>;
    /**
     * The maximum time to allow one command to run. Must be between `2 seconds` and
     * `60 seconds`.
     * @default `"5 seconds"`
     */
    timeout?: Input<DurationMinutes>;
    /**
     * The time between running the command for the health check. Must be between `5 seconds`
     * and `300 seconds`.
     * @default `"30 seconds"`
     */
    interval?: Input<DurationMinutes>;
    /**
     * The number of consecutive failures required to consider the check to have failed. Must
     * be between `1` and `10`.
     * @default `3`
     */
    retries?: Input<number>;
  }>;
  /**
   * The containers to run in the service.
   *
   * :::tip
   * You can optionally run multiple containers in a service.
   * :::
   *
   * By default this starts a single container. To add multiple containers in the service, pass
   * in an array of containers args.
   *
   * ```ts
   * {
   *   containers: [
   *     {
   *       name: "app",
   *       image: "nginxdemos/hello:plain-text"
   *     },
   *     {
   *       name: "admin",
   *       image: {
   *         context: "./admin",
   *         dockerfile: "Dockerfile"
   *       }
   *     }
   *   ]
   * }
   * ```
   *
   * If you specify `containers`, you cannot list the above args at the top-level. For example,
   * you **cannot** pass in `image` at the top level.
   *
   * ```diff lang="ts"
   * {
   * -  image: "nginxdemos/hello:plain-text",
   *   containers: [
   *     {
   *       name: "app",
   *       image: "nginxdemos/hello:plain-text"
   *     },
   *     {
   *       name: "admin",
   *       image: "nginxdemos/hello:plain-text"
   *     }
   *   ]
   * }
   * ```
   *
   * You will need to pass in `image` as a part of the `containers`.
   */
  containers?: Input<Prettify<ServiceContainerArgs>>[];
  /**
   * Configure if `sst deploy` should wait for the service to be stable.
   *
   * :::tip
   * For non-prod environments it might make sense to pass in `false`.
   * :::
   *
   * Waiting for this process to finish ensures that new content will be available after
   * the deploy finishes. However, this process can sometimes take more than 5 mins.
   * @default `false`
   * @example
   * ```js
   * {
   *   wait: true
   * }
   * ```
   */
  wait?: Input<boolean>;
  /**
   * [Transform](/docs/components#transform) how this component creates its underlying
   * resources.
   */
  transform?: Prettify<
    FargateBaseArgs["transform"] & {
      /**
       * Transform the ECS Service resource.
       */
      service?: Transform<ecs.ServiceArgs>;
      /**
       * Transform the AWS Load Balancer resource.
       */
      loadBalancer?: Transform<lb.LoadBalancerArgs>;
      /**
       * Transform the AWS Security Group resource for the Load Balancer.
       */
      loadBalancerSecurityGroup?: Transform<ec2.SecurityGroupArgs>;
      /**
       * Transform the AWS Load Balancer listener resource.
       */
      listener?: Transform<lb.ListenerArgs>;
      /**
       * Transform the AWS Load Balancer target group resource.
       */
      target?: Transform<lb.TargetGroupArgs>;
      /**
       * Transform the AWS Application Auto Scaling target resource.
       */
      autoScalingTarget?: Transform<appautoscaling.TargetArgs>;
      /**
       * Transform the AWS Load Balancer listener rule resource. Only applies when
       * attaching to an external ALB via the `loadBalancer.instance` prop.
       */
      listenerRule?: Transform<lb.ListenerRuleArgs>;
    }
  >;
}

/**
 * The `Service` component lets you create containers that are always running, like web or
 * application servers. It uses [Amazon ECS](https://aws.amazon.com/ecs/) on [AWS Fargate](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/AWS_Fargate.html).
 *
 * @example
 *
 * #### Create a Service
 *
 * Services are run inside an ECS Cluster. If you haven't already, create one.
 *
 * ```ts title="sst.config.ts"
 * const vpc = new sst.aws.Vpc("MyVpc");
 * const cluster = new sst.aws.Cluster("MyCluster", { vpc });
 * ```
 *
 * Add the service to it.
 *
 * ```ts title="sst.config.ts"
 * const service = new sst.aws.Service("MyService", { cluster });
 * ```
 *
 * #### Configure the container image
 *
 * By default, the service will look for a Dockerfile in the root directory. Optionally
 * configure the image context and dockerfile.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Service("MyService", {
 *   cluster,
 *   image: {
 *     context: "./app",
 *     dockerfile: "Dockerfile"
 *   }
 * });
 * ```
 *
 * To add multiple containers in the service, pass in an array of containers args.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Service("MyService", {
 *   cluster,
 *   containers: [
 *     {
 *       name: "app",
 *       image: "nginxdemos/hello:plain-text"
 *     },
 *     {
 *       name: "admin",
 *       image: {
 *         context: "./admin",
 *         dockerfile: "Dockerfile"
 *       }
 *     }
 *   ]
 * });
 * ```
 *
 * This is useful for running sidecar containers.
 *
 * #### Enable auto-scaling
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Service("MyService", {
 *   cluster,
 *   scaling: {
 *     min: 4,
 *     max: 16,
 *     cpuUtilization: 50,
 *     memoryUtilization: 50
 *   }
 * });
 * ```
 *
 * #### Expose through API Gateway
 *
 * You can give your service a public URL by exposing it through API Gateway HTTP API. You can
 * also optionally give it a custom domain.
 *
 * ```ts title="sst.config.ts"
 * const service = new sst.aws.Service("MyService", {
 *   cluster,
 *   serviceRegistry: {
 *     port: 80
 *   }
 * });
 *
 * const api = new sst.aws.ApiGatewayV2("MyApi", {
 *   vpc,
 *   domain: "example.com"
 * });
 * api.routePrivate("$default", service.nodes.cloudmapService.arn);
 * ```
 *
 * #### Add a load balancer
 *
 * You can also expose your service by adding a load balancer to it and optionally
 * adding a custom domain.
 *
 * ```ts title="sst.config.ts"
 * new sst.aws.Service("MyService", {
 *   cluster,
 *   loadBalancer: {
 *     domain: "example.com",
 *     rules: [
 *       { listen: "80/http" },
 *       { listen: "443/https", forward: "80/http" }
 *     ]
 *   }
 * });
 * ```
 *
 * #### Link resources
 *
 * [Link resources](/docs/linking/) to your service. This will grant permissions
 * to the resources and allow you to access it in your app.
 *
 * ```ts {5} title="sst.config.ts"
 * const bucket = new sst.aws.Bucket("MyBucket");
 *
 * new sst.aws.Service("MyService", {
 *   cluster,
 *   link: [bucket]
 * });
 * ```
 *
 * You can use the [SDK](/docs/reference/sdk/) to access the linked resources in your service.
 *
 * ```ts title="app.ts"
 * import { Resource } from "sst";
 *
 * console.log(Resource.MyBucket.name);
 * ```
 *
 * #### Service discovery
 *
 * This component automatically creates a Cloud Map service host name for the
 * service. So anything in the same VPC can access it using the service's host name.
 *
 * For example, if you link the service to a Lambda function that's in the same VPC.
 *
 * ```ts title="sst.config.ts" {2,4}
 * new sst.aws.Function("MyFunction", {
 *   vpc,
 *   url: true,
 *   link: [service],
 *   handler: "lambda.handler"
 * });
 * ```
 *
 * You can access the service by its host name using the [SDK](/docs/reference/sdk/).
 *
 * ```ts title="lambda.ts"
 * import { Resource } from "sst";
 *
 * await fetch(`http://${Resource.MyService.service}`);
 * ```
 *
 * [Check out an example](/docs/examples/#aws-cluster-service-discovery).
 *
 * ---
 *
 * ### Cost
 *
 * By default, this uses a _Linux/X86_ _Fargate_ container with 0.25 vCPUs at $0.04048 per
 * vCPU per hour and 0.5 GB of memory at $0.004445 per GB per hour. It includes 20GB of
 * _Ephemeral Storage_ for free with additional storage at $0.000111 per GB per hour. Each
 * container also gets a public IPv4 address at $0.005 per hour.
 *
 * It works out to $0.04048 x 0.25 x 24 x 30 + $0.004445 x 0.5 x 24 x 30 + $0.005
 * x 24 x 30 or **$12 per month**.
 *
 * If you are using all Fargate Spot instances with `capacity: "spot"`, it's $0.01218784 x 0.25
 * x 24 x 30 + $0.00133831 x 0.5 x 24 x 30 + $0.005 x 24 x 30 or **$6 per month**
 *
 * Adjust this for the `cpu`, `memory` and `storage` you are using. And
 * check the prices for _Linux/ARM_ if you are using `arm64` as your `architecture`.
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [Fargate pricing](https://aws.amazon.com/fargate/pricing/) and the
 * [Public IPv4 Address pricing](https://aws.amazon.com/vpc/pricing/) for more details.
 *
 * #### Scaling
 *
 * By default, `scaling` is disabled. If enabled, adjust the above for the number of containers.
 *
 * #### API Gateway
 *
 * If you expose your service through API Gateway, you'll need to add the cost of
 * [API Gateway HTTP API](https://aws.amazon.com/api-gateway/pricing/#HTTP_APIs) as well.
 * For services that don't get a lot of traffic, this ends up being a lot cheaper since API
 * Gateway is pay per request.
 *
 * Learn more about using
 * [Cluster with API Gateway](/docs/examples/#aws-cluster-with-api-gateway).
 *
 * #### Application Load Balancer
 *
 * If you add `loadBalancer` _HTTP_ or _HTTPS_ `rules`, an ALB is created at $0.0225 per hour,
 * $0.008 per LCU-hour, and $0.005 per hour if HTTPS with a custom domain is used. Where LCU
 * is a measure of how much traffic is processed.
 *
 * That works out to $0.0225 x 24 x 30 or **$16 per month**. Add $0.005 x 24 x 30 or **$4 per
 * month** for HTTPS. Also add the LCU-hour used.
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [Application Load Balancer pricing](https://aws.amazon.com/elasticloadbalancing/pricing/)
 * for more details.
 *
 * #### Network Load Balancer
 *
 * If you add `loadBalancer` _TCP_, _UDP_, or _TLS_ `rules`, an NLB is created at $0.0225 per hour and
 * $0.006 per NLCU-hour. Where NCLU is a measure of how much traffic is processed.
 *
 * That works out to $0.0225 x 24 x 30 or **$16 per month**. Also add the NLCU-hour used.
 *
 * The above are rough estimates for _us-east-1_, check out the
 * [Network Load Balancer pricing](https://aws.amazon.com/elasticloadbalancing/pricing/)
 * for more details.
 */
export class Service extends Component implements Link.Linkable {
  private readonly _name: string;
  private readonly _service?: Output<ecs.Service>;
  private readonly cloudmapNamespace?: Output<string | undefined>;
  private readonly cloudmapService?: Output<
    servicediscovery.Service | undefined
  >;
  private readonly executionRole?: iam.Role;
  private readonly taskRole: iam.Role;
  private readonly taskDefinition?: Output<ecs.TaskDefinition>;
  private readonly loadBalancer?: lb.LoadBalancer;
  private readonly autoScalingTarget?: appautoscaling.Target;
  private readonly domain?: Output<string | undefined>;
  private readonly _url?: Output<string>;
  private readonly devUrl?: Output<string>;
  private readonly dev: boolean;

  constructor(
    name: string,
    args: ServiceArgs,
    opts: ComponentResourceOptions = {},
  ) {
    super(__pulumiType, name, args, opts);
    this._name = name;

    const self = this;
    const clusterArn = args.cluster.nodes.cluster.arn;
    const clusterName = args.cluster.nodes.cluster.name;
    const region = getRegionOutput({}, opts).region;
    const dev = normalizeDev();
    const wait = output(args.wait ?? false);
    const architecture = normalizeArchitecture(args);
    const cpu = normalizeCpu(args);
    const memory = normalizeMemory(cpu, args);
    const storage = normalizeStorage(args);
    const containers = normalizeContainers("service", args, name, architecture);
    const albAttachment = detectAlbAttachment();
    const lbArgs = albAttachment ? undefined : normalizeLoadBalancer();
    const scaling = normalizeScaling();
    const capacity = normalizeCapacity();
    const vpc = normalizeVpc();

    const taskRole = createTaskRole(name, args, opts, self, !!dev);

    this.dev = !!dev;
    this.cloudmapNamespace = vpc.cloudmapNamespaceName;
    this.taskRole = taskRole;

    if (dev) {
      this.devUrl = (!lbArgs && !args.loadBalancer) ? undefined : dev.url;
      registerReceiver();
      return;
    }

    const executionRole = createExecutionRole(name, args, opts, self);
    const taskDefinition = createTaskDefinition(
      name,
      args,
      opts,
      self,
      containers,
      architecture,
      cpu,
      memory,
      storage,
      taskRole,
      executionRole,
    );
    let loadBalancer: lb.LoadBalancer | undefined;
    let targetGroups: ReturnType<typeof createTargets>;
    let targetEntries: Output<{ targetGroup: lb.TargetGroup; containerName: string; containerPort: number }[]>;
    let effectiveLbArn: Output<string> | undefined;
    let effectiveDomain: Output<string | undefined>;
    let effectiveDnsName: Output<string> | undefined;
    const certificateArn = albAttachment ? output(undefined) : createSsl();
    if (albAttachment) {
      all([albAttachment.instance._vpc, vpc.id]).apply(
        ([albVpcId, clusterVpcId]) => {
          if (albVpcId !== clusterVpcId) {
            throw new VisibleError(
              `The ALB VPC "${albVpcId}" does not match the cluster VPC "${clusterVpcId}" in Service "${name}". The ALB and cluster must be in the same VPC.`,
            );
          }
        },
      );
      const { targets: albTargets, entries: albEntries } = createAlbTargetsAndEntries(albAttachment);
      targetGroups = output(albTargets);
      targetEntries = albEntries;
      createAlbListenerRules(albAttachment, albTargets);
      effectiveLbArn = albAttachment.instance.arn;
      effectiveDomain = output(undefined);
      effectiveDnsName = albAttachment.instance.dnsName;
    } else {
      loadBalancer = createLoadBalancer();
      targetGroups = createTargets();
      targetEntries = computeTargetEntries();
      createListeners();
      createDnsRecords();
      if (loadBalancer) {
        effectiveLbArn = loadBalancer.arn;
        effectiveDnsName = loadBalancer.dnsName;
      }
      effectiveDomain = lbArgs?.domain?.apply((d) => d?.name) ?? output(undefined);
    }
    const cloudmapService = createCloudmapService();
    const service = createService();
    const autoScalingTarget = createAutoScaling();

    this._service = service;
    this.cloudmapService = cloudmapService;
    this.executionRole = executionRole;
    this.taskDefinition = taskDefinition;
    this.loadBalancer = loadBalancer ?? albAttachment?.instance.nodes.loadBalancer;
    this.autoScalingTarget = autoScalingTarget;
    this.domain = effectiveDomain;
    this._url = effectiveDnsName
      ? all([effectiveDomain, effectiveDnsName]).apply(
          ([domain, dnsName]) =>
            domain ? `https://${domain}/` : `http://${dnsName}`,
        )
      : undefined;

    this.registerOutputs({ _hint: this._url });
    registerReceiver();

    function normalizeDev() {
      if (!$dev) return undefined;
      if (args.dev === false) return undefined;

      return {
        url: output(args.dev?.url ?? URL_UNAVAILABLE),
      };
    }

    function normalizeVpc() {
      // "vpc" is a Vpc component
      if (args.cluster.vpc instanceof Vpc) {
        const vpc = args.cluster.vpc;
        return {
          isSstVpc: true,
          id: vpc.id,
          loadBalancerSubnets: lbArgs?.pub.apply((v) =>
            v ? vpc.publicSubnets : vpc.privateSubnets,
          ),
          containerSubnets: vpc.publicSubnets,
          securityGroups: vpc.securityGroups,
          cloudmapNamespaceId: vpc.nodes.cloudmapNamespace.id,
          cloudmapNamespaceName: vpc.nodes.cloudmapNamespace.name,
        };
      }

      // "vpc" is object
      return output(args.cluster.vpc).apply((vpc) => ({
        isSstVpc: false,
        ...vpc,
      }));
    }

    function normalizeScaling() {
      // External ALB is always "application" type
      const lbType = albAttachment ? output("application" as const) : lbArgs?.type;
      return all([lbType, args.scaling]).apply(([type, v]) => {
        if (type !== "application" && v?.requestCount)
          throw new VisibleError(
            `Request count scaling is only supported for http/https protocols.`,
          );

        return {
          min: v?.min ?? 1,
          max: v?.max ?? 1,
          cpuUtilization: v?.cpuUtilization ?? 70,
          memoryUtilization: v?.memoryUtilization ?? 70,
          requestCount: v?.requestCount ?? false,
          scaleInCooldown: v?.scaleInCooldown ? toSeconds(v.scaleInCooldown) : undefined,
          scaleOutCooldown: v?.scaleOutCooldown ? toSeconds(v.scaleOutCooldown) : undefined,
        };
      });
    }

    function normalizeCapacity() {
      if (!args.capacity) return;

      return output(args.capacity).apply((v) => {
        if (v === "spot")
          return { spot: { weight: 1 }, fargate: { weight: 0 } };
        return v;
      });
    }

    function normalizeLoadBalancer() {
      const loadBalancer = args.loadBalancer ?? args.public;
      if (!loadBalancer) return;
      // ALB attachment case is handled by detectAlbAttachment() before this is called
      const inlineLoadBalancer = output(loadBalancer).apply(
        (lb) => lb as Exclude<typeof lb, { instance: Alb }>,
      );

      // normalize rules
      const rules = all([inlineLoadBalancer, containers]).apply(
        ([lb, containers]) => {
          // validate rules
          const lbRules = lb.rules ?? lb.ports;
          if (!lbRules || lbRules.length === 0)
            throw new VisibleError(
              `You must provide the ports to expose via "loadBalancer.rules".`,
            );

          // validate container defined when multiple containers exists
          if (containers.length > 1) {
            lbRules.forEach((v) => {
              if (!v.container)
                throw new VisibleError(
                  `You must provide a container name in "loadBalancer.rules" when there is more than one container.`,
                );
            });
          }

          // parse protocols and ports
          const rules = lbRules.map((v) => {
            const listenParts = v.listen.split("/");
            const listenPort = parseInt(listenParts[0]);
            const listenProtocol = listenParts[1];
            const listenConditions =
              v.conditions || v.path
                ? {
                    path: v.conditions?.path ?? v.path,
                    query: v.conditions?.query,
                    header: v.conditions?.header,
                  }
                : undefined;
            if (protocolType(listenProtocol) === "network" && listenConditions)
              throw new VisibleError(
                `Invalid rule conditions for listen protocol "${v.listen}". Only "http" protocols support conditions.`,
              );

            const redirectParts = v.redirect?.split("/");
            const redirectPort = redirectParts && parseInt(redirectParts[0]);
            const redirectProtocol = redirectParts && redirectParts[1];
            if (redirectPort && redirectProtocol) {
              if (
                protocolType(listenProtocol) !== protocolType(redirectProtocol)
              )
                throw new VisibleError(
                  `The listen protocol "${v.listen}" must match the redirect protocol "${v.redirect}".`,
                );
              return {
                type: "redirect" as const,
                listenPort,
                listenProtocol,
                listenConditions,
                redirectPort,
                redirectProtocol,
              };
            }

            const forwardParts = v.forward ? v.forward.split("/") : listenParts;
            const forwardPort = forwardParts && parseInt(forwardParts[0]);
            const forwardProtocol = forwardParts && forwardParts[1];
            if (protocolType(listenProtocol) !== protocolType(forwardProtocol))
              throw new VisibleError(
                `The listen protocol "${v.listen}" must match the forward protocol "${v.forward}".`,
              );
            return {
              type: "forward" as const,
              listenPort,
              listenProtocol,
              listenConditions,
              forwardPort,
              forwardProtocol,
              container: v.container ?? containers[0].name,
            };
          });

          // validate protocols are consistent
          const appProtocols = rules.filter(
            (rule) => protocolType(rule.listenProtocol) === "application",
          );
          if (appProtocols.length > 0 && appProtocols.length < rules.length)
            throw new VisibleError(
              `Protocols must be either all http/https, or all tcp/udp/tcp_udp/tls.`,
            );

          // validate certificate exists for https/tls protocol
          rules.forEach((rule) => {
            if (["https", "tls"].includes(rule.listenProtocol) && !lb.domain) {
              throw new VisibleError(
                `You must provide a custom domain for ${rule.listenProtocol.toUpperCase()} protocol.`,
              );
            }
          });

          return rules;
        },
      );

      // normalize domain
      const domain = output(inlineLoadBalancer).apply((lb) => {
        if (!lb.domain) return undefined;

        // normalize domain
        const domain =
          typeof lb.domain === "string" ? { name: lb.domain } : lb.domain;
        return {
          name: domain.name,
          aliases: domain.aliases ?? [],
          dns: domain.dns === false ? undefined : domain.dns ?? awsDns(),
          cert: domain.cert,
        };
      });

      // normalize type
      const type = output(rules).apply((rules) =>
        rules[0].listenProtocol.startsWith("http") ? "application" : "network",
      );

      // normalize public/private
      const pub = output(inlineLoadBalancer).apply((lb) =>
        "public" in lb ? lb.public ?? true : true,
      );

      // normalize health check
      const health = all([type, rules, inlineLoadBalancer]).apply(
        ([type, rules, lb]) =>
          Object.fromEntries(
            Object.entries(("health" in lb ? lb.health : {}) ?? {}).map(
              ([k, v]) => {
                if (
                  !rules.find(
                    (r) => `${r.forwardPort}/${r.forwardProtocol}` === k,
                  )
                )
                  throw new VisibleError(
                    `Cannot configure health check for "${k}". Make sure it is defined in "loadBalancer.ports".`,
                  );
                const protocol = k.split("/")[1];
                return [
                  k,
                  {
                    path: ["http", "https"].includes(protocol)
                      ? v.path ?? "/"
                      : undefined,
                    interval: v.interval ? toSeconds(v.interval) : 30,
                    timeout: v.timeout
                      ? toSeconds(v.timeout)
                      : type === "application"
                        ? 5
                        : 6,
                    healthyThreshold: v.healthyThreshold ?? 5,
                    unhealthyThreshold: v.unhealthyThreshold ?? 2,
                    protocol: ["http", "https"].includes(protocol)
                      ? undefined
                      : "TCP",
                    matcher: ["http", "https"].includes(protocol)
                      ? v.successCodes ?? "200"
                      : undefined,
                  },
                ];
              },
            ),
          ),
      );

      return { type, rules, domain, pub, health };
    }

    function createLoadBalancer() {
      if (!lbArgs) return;

      const securityGroup = new ec2.SecurityGroup(
        ...transform(
          args?.transform?.loadBalancerSecurityGroup,
          `${name}LoadBalancerSecurityGroup`,
          {
            description: "Managed by SST",
            vpcId: vpc.id,
            egress: [
              {
                fromPort: 0,
                toPort: 0,
                protocol: "-1",
                cidrBlocks: ["0.0.0.0/0"],
              },
            ],
            ingress: [
              {
                fromPort: 0,
                toPort: 0,
                protocol: "-1",
                cidrBlocks: ["0.0.0.0/0"],
              },
            ],
          },
          { parent: self },
        ),
      );

      return new lb.LoadBalancer(
        ...transform(
          args.transform?.loadBalancer,
          `${name}LoadBalancer`,
          {
            internal: lbArgs.pub.apply((v) => !v),
            loadBalancerType: lbArgs.type,
            subnets: vpc.loadBalancerSubnets,
            securityGroups: [securityGroup.id],
            enableCrossZoneLoadBalancing: true,
          },
          { parent: self },
        ),
      );
    }

    function createTargets() {
      if (!loadBalancer || !lbArgs) return;

      return all([lbArgs.rules, lbArgs.health]).apply(([rules, health]) => {
        const targets: Record<string, lb.TargetGroup> = {};

        rules.forEach((r) => {
          if (r.type !== "forward") return;

          const container = r.container;
          const forwardProtocol = r.forwardProtocol.toUpperCase();
          const forwardPort = r.forwardPort;
          const targetId = targetKey(container!, forwardProtocol, forwardPort);
          const target =
            targets[targetId] ??
            new lb.TargetGroup(
              ...transform(
                args.transform?.target,
                `${name}Target${targetId}`,
                {
                  // AWS enforces a 6-char limit on namePrefix for target groups.
                  // "TCP_UDP" is 7 chars, so strip the underscore to fit.
                  namePrefix: forwardProtocol.replace("_", ""),
                  port: forwardPort,
                  protocol: forwardProtocol,
                  targetType: "ip",
                  vpcId: vpc.id,
                  healthCheck: health[`${r.forwardPort}/${r.forwardProtocol}`],
                },
                { parent: self },
              ),
            );
          targets[targetId] = target;
        });
        return targets;
      });
    }

    function computeTargetEntries() {
      if (!lbArgs || !targetGroups)
        return output(
          [] as {
            targetGroup: lb.TargetGroup;
            containerName: string;
            containerPort: number;
          }[],
        );
      return all([lbArgs.rules, targetGroups]).apply(([rules, targets]) => {
        const entries: {
          targetGroup: lb.TargetGroup;
          containerName: string;
          containerPort: number;
        }[] = [];
        const seen = new Set<string>();
        for (const rule of rules) {
          if (rule.type !== "forward") continue;
          const targetId = targetKey(rule.container!, rule.forwardProtocol, rule.forwardPort);
          if (seen.has(targetId)) continue;
          seen.add(targetId);
          entries.push({
            targetGroup: targets[targetId],
            containerName: rule.container!,
            containerPort: rule.forwardPort,
          });
        }
        return entries;
      });
    }

    function createListeners() {
      if (!lbArgs || !loadBalancer || !targetGroups) return;

      return all([lbArgs.rules, targetGroups, certificateArn]).apply(
        ([rules, targets, cert]) => {
          // Group listeners by protocol and port
          // Because listeners with the same protocol and port but different path
          // are just rules of the same listener.
          const listenersById: Record<string, typeof rules> = {};
          rules.forEach((r) => {
            const listenProtocol = r.listenProtocol.toUpperCase();
            const listenPort = r.listenPort;
            const listenerId = listenerKey(listenProtocol, listenPort);
            listenersById[listenerId] = listenersById[listenerId] ?? [];
            listenersById[listenerId].push(r);
          });

          // Create listeners
          return Object.entries(listenersById).map(([listenerId, rules]) => {
            const listenProtocol = rules[0].listenProtocol.toUpperCase();
            const listenPort = rules[0].listenPort;
            const defaultRule = rules.find((r) => !r.listenConditions);
            const customRules = rules.filter((r) => r.listenConditions);
            const buildActions = (r?: (typeof rules)[number]) => [
              ...(!r
                ? [
                    {
                      type: "fixed-response",
                      fixedResponse: {
                        statusCode: "403",
                        contentType: "text/plain",
                        messageBody: "Forbidden",
                      },
                    },
                  ]
                : []),
              ...(r?.type === "forward"
                ? [
                    {
                      type: "forward",
                      targetGroupArn:
                        targets[
                          `${r.container}${r.forwardProtocol.toUpperCase()}${
                            r.forwardPort
                          }`
                        ].arn,
                    },
                  ]
                : []),
              ...(r?.type === "redirect"
                ? [
                    {
                      type: "redirect",
                      redirect: {
                        port: r.redirectPort.toString(),
                        protocol: r.redirectProtocol.toUpperCase(),
                        statusCode: "HTTP_301",
                      },
                    },
                  ]
                : []),
            ];
            const listener = new lb.Listener(
              ...transform(
                args.transform?.listener,
                `${name}Listener${listenerId}`,
                {
                  loadBalancerArn: loadBalancer.arn,
                  port: listenPort,
                  protocol: listenProtocol,
                  certificateArn: ["HTTPS", "TLS"].includes(listenProtocol)
                    ? cert
                    : undefined,
                  defaultActions: buildActions(defaultRule),
                },
                { parent: self },
              ),
            );

            customRules.forEach(
              (r) =>
                new lb.ListenerRule(
                  `${name}Listener${listenerId}Rule${hashStringToPrettyString(
                    JSON.stringify(r.listenConditions),
                    4,
                  )}`,
                  {
                    listenerArn: listener.arn,
                    actions: buildActions(r),
                    conditions: [
                      {
                        pathPattern: r.listenConditions!.path
                          ? { values: [r.listenConditions!.path!] }
                          : undefined,
                        queryStrings: r.listenConditions!.query,
                        httpHeader: r.listenConditions!.header
                          ? {
                              httpHeaderName: r.listenConditions!.header.name,
                              values: r.listenConditions!.header.values,
                            }
                          : undefined,
                      },
                    ],
                  },
                  { parent: self },
                ),
            );

            return listener;
          });
        },
      );
    }

    function createSsl() {
      if (!lbArgs) return output(undefined);

      return lbArgs.domain.apply((domain) => {
        if (!domain) return output(undefined);
        if (domain.cert) return output(domain.cert);

        return new DnsValidatedCertificate(
          `${name}Ssl`,
          {
            domainName: domain.name,
            alternativeNames: domain.aliases,
            dns: domain.dns!,
          },
          { parent: self },
        ).arn;
      });
    }

    function createCloudmapService() {
      return output(vpc.cloudmapNamespaceId).apply((cloudmapNamespaceId) => {
        if (!cloudmapNamespaceId) return;

        return new servicediscovery.Service(
          `${name}CloudmapService`,
          {
            name: `${name}.${$app.stage}.${$app.name}`,
            namespaceId: output(vpc.cloudmapNamespaceId).apply((id) => id!),
            forceDestroy: true,
            dnsConfig: {
              namespaceId: output(vpc.cloudmapNamespaceId).apply((id) => id!),
              dnsRecords: [
                ...(args.serviceRegistry ? [{ ttl: 60, type: "SRV" }] : []),
                { ttl: 60, type: "A" },
              ],
            },
          },
          { parent: self },
        );
      });
    }

    function createService() {
      return cloudmapService.apply(
        (cloudmapService) =>
          new ecs.Service(
            ...transform(
              args.transform?.service,
              `${name}Service`,
              {
                name,
                cluster: clusterArn,
                taskDefinition: taskDefinition.arn,
                desiredCount: scaling.min,
                ...(capacity
                  ? {
                      // setting `forceNewDeployment` ensures that the service is not recreated
                      // when the capacity provider config changes.
                      forceNewDeployment: true,
                      capacityProviderStrategies: capacity.apply((v) => [
                        ...(v.fargate
                          ? [
                              {
                                capacityProvider: "FARGATE",
                                base: v.fargate?.base,
                                weight: v.fargate?.weight,
                              },
                            ]
                          : []),
                        ...(v.spot
                          ? [
                              {
                                capacityProvider: "FARGATE_SPOT",
                                base: v.spot?.base,
                                weight: v.spot?.weight,
                              },
                            ]
                          : []),
                      ]),
                    }
                  : // @deprecated do not use `launchType`, set `capacityProviderStrategies`
                    // to `[{ capacityProvider: "FARGATE", weight: 1 }]` instead
                    {
                      launchType: "FARGATE",
                    }),
                networkConfiguration: {
                  // If the vpc is an SST vpc, services are automatically deployed to the public
                  // subnets. So we need to assign a public IP for the service to be accessible.
                  assignPublicIp: vpc.isSstVpc,
                  subnets: vpc.containerSubnets,
                  securityGroups: vpc.securityGroups,
                },
                deploymentCircuitBreaker: {
                  enable: true,
                  rollback: true,
                },
                loadBalancers: targetEntries.apply((entries) =>
                  entries.map((e) => ({
                    targetGroupArn: e.targetGroup.arn,
                    containerName: e.containerName,
                    containerPort: e.containerPort,
                  })),
                ),
                enableExecuteCommand: true,
                serviceRegistries: cloudmapService && {
                  registryArn: cloudmapService.arn,
                  port: args.serviceRegistry
                    ? output(args.serviceRegistry).port
                    : undefined,
                },
                waitForSteadyState: wait,
              },
              { parent: self },
            ),
          ),
      );
    }

    function createAutoScaling() {
      const target = new appautoscaling.Target(
        ...transform(
          args.transform?.autoScalingTarget,
          `${name}AutoScalingTarget`,
          {
            serviceNamespace: "ecs",
            scalableDimension: "ecs:service:DesiredCount",
            resourceId: interpolate`service/${clusterName}/${service.name}`,
            maxCapacity: scaling.max,
            minCapacity: scaling.min,
          },
          { parent: self },
        ),
      );

      all([scaling.cpuUtilization, scaling.scaleInCooldown, scaling.scaleOutCooldown]).apply(
        ([cpuUtilization, scaleInCooldown, scaleOutCooldown]) => {
          if (cpuUtilization === false) return;
          new appautoscaling.Policy(
            `${name}AutoScalingCpuPolicy`,
            {
              serviceNamespace: target.serviceNamespace,
              scalableDimension: target.scalableDimension,
              resourceId: target.resourceId,
              policyType: "TargetTrackingScaling",
              targetTrackingScalingPolicyConfiguration: {
                predefinedMetricSpecification: {
                  predefinedMetricType: "ECSServiceAverageCPUUtilization",
                },
                targetValue: cpuUtilization,
                scaleInCooldown,
                scaleOutCooldown,
              },
            },
            { parent: self },
          );
        }
      );

      all([scaling.memoryUtilization, scaling.scaleInCooldown, scaling.scaleOutCooldown]).apply(
        ([memoryUtilization, scaleInCooldown, scaleOutCooldown]) => {
          if (memoryUtilization === false) return;
          new appautoscaling.Policy(
            `${name}AutoScalingMemoryPolicy`,
            {
              serviceNamespace: target.serviceNamespace,
              scalableDimension: target.scalableDimension,
              resourceId: target.resourceId,
              policyType: "TargetTrackingScaling",
              targetTrackingScalingPolicyConfiguration: {
                predefinedMetricSpecification: {
                  predefinedMetricType: "ECSServiceAverageMemoryUtilization",
                },
                targetValue: memoryUtilization,
                scaleInCooldown,
                scaleOutCooldown,
              },
            },
            { parent: self },
          );
        }
      );

      all([scaling.requestCount, scaling.scaleInCooldown, scaling.scaleOutCooldown, targetGroups]).apply(
        ([requestCount, scaleInCooldown, scaleOutCooldown, targetGroups]) => {
          if (requestCount === false) return;
          if (!targetGroups) return;

          const targetGroup = Object.values(targetGroups)[0];

          new appautoscaling.Policy(
            `${name}AutoScalingRequestCountPolicy`,
            {
              serviceNamespace: target.serviceNamespace,
              scalableDimension: target.scalableDimension,
              resourceId: target.resourceId,
              policyType: "TargetTrackingScaling",
              targetTrackingScalingPolicyConfiguration: {
                predefinedMetricSpecification: {
                  predefinedMetricType: "ALBRequestCountPerTarget",
                  resourceLabel: all([effectiveLbArn!, targetGroup.arn]).apply(
                    ([lbArn, targetGroupArn]) => {
                      // arn:...:loadbalancer/app/frank-MyServiceLoadBalan/005af2ad12da1e52
                      // => app/frank-MyServiceLoadBalan/005af2ad12da1e52
                      const lbPart = lbArn
                        .split(":")
                        .pop()
                        ?.split("/")
                        .slice(1)
                        .join("/");
                      // arn:...:targetgroup/HTTP20250103004618450100000001/e0811b8cf3a60762
                      // => targetgroup/HTTP20250103004618450100000001
                      const tgPart = targetGroupArn.split(":").pop();
                      return `${lbPart}/${tgPart}`;
                    },
                  ),
                },
                targetValue: requestCount,
                scaleInCooldown,
                scaleOutCooldown,
              },
            },
            { parent: self },
          );
        },
      );

      return target;
    }

    function createDnsRecords() {
      if (!lbArgs) return;

      lbArgs.domain.apply((domain) => {
        if (!domain?.dns) return;

        for (const recordName of [domain.name, ...domain.aliases]) {
          const namePrefix =
            recordName === domain.name ? name : `${name}${recordName}`;
          domain.dns.createAlias(
            namePrefix,
            {
              name: recordName,
              aliasName: loadBalancer!.dnsName,
              aliasZone: loadBalancer!.zoneId,
            },
            { parent: self },
          );
        }
      });
    }

    function detectAlbAttachment() {
      const loadBalancer = args.loadBalancer;
      if (
        !loadBalancer ||
        typeof loadBalancer !== "object" ||
        !("instance" in loadBalancer) ||
        !(loadBalancer.instance instanceof Alb)
      ) {
        return undefined;
      }
      return loadBalancer;
    }

    function createAlbTargetsAndEntries(
      attachment: NonNullable<typeof albAttachment>,
    ) {
      const rules = attachment.rules;
      const health = attachment.health ?? {};

      if (rules.length === 0) {
        throw new VisibleError(
          `You must provide at least one rule in "loadBalancer.rules" when using an external ALB in Service "${name}".`,
        );
      }

      // Validate container names (no resources created here)
      containers.apply((ctrs) => {
        const containerNames = new Set(ctrs.map((c) => c.name));
        if (ctrs.length > 1) {
          for (const rule of rules) {
            if (!rule.container) {
              throw new VisibleError(
                `You must provide a "container" name in each rule when there is more than one container in Service "${name}".`,
              );
            }
          }
        }
        for (const rule of rules) {
          const cn = rule.container ?? ctrs[0].name;
          if (!containerNames.has(cn)) {
            throw new VisibleError(
              `Container "${cn}" in "loadBalancer.rules" does not match any container in Service "${name}". Available: ${[...containerNames].join(", ")}.`,
            );
          }
        }
      });

      // Create target groups in a plain loop (no apply)
      const targets: Record<string, lb.TargetGroup> = {};
      const rawEntries: {
        targetGroup: lb.TargetGroup;
        explicitContainer: string | undefined;
        containerPort: number;
      }[] = [];

      for (const rule of rules) {
        const parts = rule.forward.split("/");
        const forwardPort = parseInt(parts[0]);
        const forwardProtocol = parts[1].toUpperCase();
        // Use explicit container or component name for keying/naming
        const containerNameForKey = rule.container ?? name;
        const tgtId = targetKey(containerNameForKey, forwardProtocol, forwardPort);

        if (!targets[tgtId]) {
          const healthKey = `${forwardPort}/${parts[1]}` as AlbPort;
          const healthCheck = health[healthKey];
          const normalizedHealth = healthCheck
            ? output(healthCheck).apply((h) => ({
                path: h.path ?? "/",
                interval: h.interval ? toSeconds(h.interval) : 30,
                timeout: h.timeout ? toSeconds(h.timeout) : 5,
                healthyThreshold: h.healthyThreshold ?? 5,
                unhealthyThreshold: h.unhealthyThreshold ?? 2,
                matcher: h.successCodes ?? "200",
              }))
            : {
                path: "/",
                interval: 30,
                timeout: 5,
                healthyThreshold: 5,
                unhealthyThreshold: 2,
                matcher: "200",
              };

          const tg = new lb.TargetGroup(
            ...transform(
              args.transform?.target,
              `${name}Target${tgtId}`,
              {
                namePrefix: forwardProtocol,
                port: forwardPort,
                protocol: forwardProtocol,
                targetType: "ip",
                vpcId: attachment.instance._vpc,
                healthCheck: normalizedHealth,
              },
              { parent: self },
            ),
          );
          targets[tgtId] = tg;
          rawEntries.push({
            targetGroup: tg,
            explicitContainer: rule.container,
            containerPort: forwardPort,
          });
        }
      }

      // Resolve actual container names for ECS registration
      const entries = containers.apply((ctrs) =>
        rawEntries.map((e) => ({
          targetGroup: e.targetGroup,
          containerName: e.explicitContainer ?? ctrs[0].name,
          containerPort: e.containerPort,
        })),
      );

      return { targets, entries };
    }

    function createAlbListenerRules(
      attachment: NonNullable<typeof albAttachment>,
      albTargets: Record<string, lb.TargetGroup>,
    ) {
      const rules = attachment.rules;
      const prioritiesByListener = new Map<string, Set<number>>();

      for (const rule of rules) {
        if (rule.priority < 1 || rule.priority > 50000) {
          throw new VisibleError(
            `Priority ${rule.priority} must be between 1 and 50000 in Service "${name}". When sharing an ALB, ensure non-overlapping priority ranges across services.`,
          );
        }

        const seen =
          prioritiesByListener.get(rule.listen) ?? new Set();
        if (seen.has(rule.priority)) {
          throw new VisibleError(
            `Duplicate priority ${rule.priority} on listener "${rule.listen}" in Service "${name}".`,
          );
        }
        seen.add(rule.priority);
        prioritiesByListener.set(rule.listen, seen);

        if (
          !rule.conditions?.path &&
          !rule.conditions?.query &&
          !rule.conditions?.header
        ) {
          throw new VisibleError(
            `At least one condition (path, query, or header) must be set for rules on an external ALB in Service "${name}".`,
          );
        }

        const listenerParts = rule.listen.split("/");
        const listenerPort = parseInt(listenerParts[0]);
        const listenerProtocol = listenerParts[1];

        const forwardParts = rule.forward.split("/");
        const forwardPort = parseInt(forwardParts[0]);
        const forwardProtocol = forwardParts[1].toUpperCase();
        const containerNameForKey = rule.container ?? name;
        const tgtId = targetKey(containerNameForKey, forwardProtocol, forwardPort);

        const targetGroup = albTargets[tgtId];
        if (!targetGroup) {
          throw new VisibleError(
            `Target group "${tgtId}" not found. Ensure the forward port matches in Service "${name}".`,
          );
        }

        const listenerResource =
          attachment.instance.getListener(listenerProtocol, listenerPort);

        new lb.ListenerRule(
          ...transform(
            args.transform?.listenerRule,
            `${name}AlbRule${listenerProtocol.toUpperCase()}${listenerPort}P${rule.priority}`,
            {
              listenerArn: listenerResource.arn,
              priority: rule.priority,
              actions: [
                {
                  type: "forward",
                  targetGroupArn: targetGroup.arn,
                },
              ],
              conditions: [
                {
                  pathPattern: rule.conditions.path
                    ? { values: [rule.conditions.path] }
                    : undefined,
                  queryStrings: rule.conditions.query,
                  httpHeader: rule.conditions.header
                    ? output(rule.conditions.header).apply((h) => ({
                        httpHeaderName: h.name,
                        values: h.values,
                      }))
                    : undefined,
                },
              ],
            },
            { parent: self },
          ),
        );
      }
    }

    function registerReceiver() {
      all([containers]).apply(([val]) => {
        for (const container of val) {
          const title = val.length == 1 ? name : `${name}${container.name}`;
          new DevCommand(`${title}Dev`, {
            link: args.link,
            dev: {
              title,
              autostart: true,
              directory: (() => {
                if (!container.image) return "";
                if (typeof container.image === "string") return "";
                if (container.image.context) return container.image.context;
                return "";
              })(),
              ...container.dev,
            },
            environment: {
              ...container.environment,
              AWS_REGION: region,
            },
            aws: {
              role: taskRole.arn,
            },
          });
        }
      });
    }
  }

  /**
   * The URL of the service.
   *
   * If `public.domain` is set, this is the URL with the custom domain.
   * Otherwise, it's the auto-generated load balancer URL.
   */
  public get url() {
    const errorMessage =
      "Cannot access the URL because no public ports are exposed.";
    if (this.dev) {
      if (!this.devUrl) throw new VisibleError(errorMessage);
      return this.devUrl;
    }

    if (!this._url) throw new VisibleError(errorMessage);
    return this._url;
  }

  /**
   * The name of the Cloud Map service. This is useful for service discovery.
   */
  public get service() {
    return all([this.cloudmapNamespace, this.cloudmapService]).apply(
      ([namespace, service]) => {
        if (!namespace)
          throw new VisibleError(
            `Cannot access the AWS Cloud Map service name for the "${this._name}" Service. Cloud Map is not configured for the cluster.`,
          );

        return this.dev
          ? interpolate`dev.${namespace}`
          : interpolate`${service!.name}.${namespace}`;
      },
    );
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    const self = this;
    return {
      /**
       * The Amazon ECS Service.
       */
      get service() {
        if (self.dev)
          throw new VisibleError("Cannot access `nodes.service` in dev mode.");
        return self._service!;
      },
      /**
       * The Amazon ECS Execution Role.
       */
      executionRole: this.executionRole,
      /**
       * The Amazon ECS Task Role.
       */
      taskRole: this.taskRole,
      /**
       * The Amazon ECS Task Definition.
       */
      get taskDefinition() {
        if (self.dev)
          throw new VisibleError(
            "Cannot access `nodes.taskDefinition` in dev mode.",
          );
        return self.taskDefinition!;
      },
      /**
       * The Amazon Elastic Load Balancer.
       */
      get loadBalancer() {
        if (self.dev)
          throw new VisibleError(
            "Cannot access `nodes.loadBalancer` in dev mode.",
          );
        if (!self.loadBalancer)
          throw new VisibleError(
            "Cannot access `nodes.loadBalancer` when no public ports are exposed.",
          );
        return self.loadBalancer;
      },
      /**
       * The Amazon Application Auto Scaling target.
       */
      get autoScalingTarget() {
        if (self.dev)
          throw new VisibleError(
            "Cannot access `nodes.autoScalingTarget` in dev mode.",
          );
        return self.autoScalingTarget!;
      },
      /**
       * The Amazon Cloud Map service.
       */
      get cloudmapService() {
        if (self.dev)
          throw new VisibleError(
            "Cannot access `nodes.cloudmapService` in dev mode.",
          );

        return output(self.cloudmapService).apply((service) => {
          if (!service)
            throw new VisibleError(
              `Cannot access "nodes.cloudmapService" for the "${self._name}" Service. Cloud Map is not configured for the cluster.`,
            );
          return service;
        });
      },
    };
  }

  /** @internal */
  public getSSTLink() {
    return {
      properties: {
        url: this.dev ? this.devUrl : this._url,
        service: output(this.cloudmapNamespace).apply((namespace) =>
          namespace ? this.service : undefined,
        ),
      },
    };
  }
}

function protocolType(protocol: string) {
  return ["http", "https"].includes(protocol)
    ? ("application" as const)
    : ("network" as const);
}

const __pulumiType = "sst:aws:Service";
// @ts-expect-error
Service.__pulumiType = __pulumiType;
