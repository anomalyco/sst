import { env } from "cloudflare:workers";

export const Resource = new Proxy(
  {},
  {
    get(_target, prop: string) {
      if (prop in env) {
        const value = env[prop];
        return typeof value === "string" ? JSON.parse(value) : value;
      }
      throw new Error(`"${prop}" is not linked in your sst.config.ts`);
    },
  }
) as Record<string, any>;
