import { env } from "cloudflare:workers";

import { loadFromCloudflareEnv, Resource } from "./shared.js";

loadFromCloudflareEnv(env);

export function fromCloudflareEnv(input: any) {
  loadFromCloudflareEnv(input);
}

export function wrapCloudflareHandler(handler: any) {
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

export { Resource };
