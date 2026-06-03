import { env } from "cloudflare:workers";

import { createResource, loadResourceEnvironment } from "./shared.js";
import type { Resource as BaseResource } from "./node.js";

function loadCloudflareResources() {
  loadResourceEnvironment(env);
}

export function wrapCloudflareHandler(handler: any) {
  if (handler == null) {
    return undefined;
  }

  if (typeof handler === "function" && handler.hasOwnProperty("prototype")) {
    return class extends handler {
      constructor(ctx: any, env: any) {
        loadResourceEnvironment(env);
        super(ctx, env);
      }
    };
  }

  function wrap(fn: any) {
    return function (req: any, env: any, ...rest: any[]) {
      loadResourceEnvironment(env);
      return fn(req, env, ...rest);
    };
  }

  const result = {} as any;
  for (const [key, value] of Object.entries(handler)) {
    result[key] = wrap(value);
  }
  return result;
}

// Keep an interface here so generated sst-env.d.ts can augment Resource.
export interface Resource extends BaseResource {}
export const Resource = createResource<Resource>(loadCloudflareResources);
