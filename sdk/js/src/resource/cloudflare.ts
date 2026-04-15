import { env } from "cloudflare:workers";

import { fromCloudflareEnv, Resource, wrapCloudflareHandler } from "./shared.js";

fromCloudflareEnv(env);

export { Resource, fromCloudflareEnv, wrapCloudflareHandler };
