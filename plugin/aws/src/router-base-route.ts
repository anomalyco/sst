import * as sst from "sst-plugin";
import { Transform, transform } from "sst-plugin/internal/transform";
import { VisibleError } from "sst-plugin/error";
import { AWSComponent } from "./component.js";
import { jsonStringify } from "@pulumi/pulumi";
import { KvKeys } from "./providers/kv-keys.js";
import { KvRoutesUpdate } from "./providers/kv-routes-update.js";
import crypto from "crypto";

export interface RouterBaseRouteArgs {
  /**
   * The KV Namespace to use.
   */
  routerNamespace: sst.Input<string>;
  /**
   * The KV Store to use.
   */
  store: sst.Input<string>;
  /**
   * The pattern to match.
   */
  pattern: sst.Input<string>;
}

export function parsePattern(pattern: string) {
  const [host, ...path] = pattern.split("/");
  return {
    host: host
      .replace(/[.+?^${}()|[\]\\]/g, "\\$&") // Escape special regex chars
      .replace(/\*/g, ".*"), // Replace * with .*
    path: "/" + path.join("/"),
  };
}

export function buildKvNamespace(name: string) {
  // In the case multiple sites use the same kv store, we need to namespace the keys
  return crypto
    .createHash("md5")
    .update(`${sst.app.name}-${sst.app.stage}-${name}`)
    .digest("hex")
    .substring(0, 4);
}

export function createKvRouteData(
  name: string,
  args: RouterBaseRouteArgs,
  parent: sst.Component,
  routeNs: string,
  data: any,
) {
  new KvKeys(
    `${name}RouteKey`,
    {
      store: args.store,
      namespace: routeNs,
      entries: {
        metadata: jsonStringify(data),
      },
      purge: false,
    },
    { parent },
  );
}

export function updateKvRoutes(
  name: string,
  args: RouterBaseRouteArgs,
  parent: sst.Component,
  routeType: "url" | "bucket" | "site",
  routeNs: string,
  pattern: {
    host: string;
    path: string;
  },
) {
  return new KvRoutesUpdate(
    `${name}RoutesUpdate`,
    {
      store: args.store,
      namespace: args.routerNamespace,
      key: "routes",
      entry: [routeType, routeNs, pattern.host, pattern.path].join(","),
    },
    { parent },
  );
}
