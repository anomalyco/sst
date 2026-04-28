import { env } from "cloudflare:workers";

import { createResource, loadFromCloudflareEnv } from "./shared.js";
import type { Resource as ResourceShape } from "./shared.js";

let environmentLoaded = false;

function loadCloudflareResources() {
  if (environmentLoaded) return;
  environmentLoaded = true;
  loadFromCloudflareEnv(env);
}

export function fromCloudflareEnv(input: any) {
  loadFromCloudflareEnv(input);
}

export function wrapCloudflareHandler(handler: any) {
  if (handler == null) {
    return undefined;
  }

  if (typeof handler === "function" && handler.hasOwnProperty("prototype")) {
    return class extends handler {
      constructor(ctx: any, env: any) {
        loadFromCloudflareEnv(env);
        super(ctx, env);
      }
    };
  }

  function wrap(fn: any) {
    return function (req: any, env: any, ...rest: any[]) {
      loadFromCloudflareEnv(env);
      return fn(req, env, ...rest);
    };
  }

  const result = {} as any;
  for (const [key, value] of Object.entries(handler)) {
    result[key] = wrap(value);
  }
  return result;
}

export interface Resource extends ResourceShape {}
export const Resource = createResource(loadCloudflareResources) as Resource;
