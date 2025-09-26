import { env } from "process";
import { readFileSync } from "fs";
import crypto from "crypto";

export interface Resource {
  App: {
    name: string;
    stage: string;
  };
}

const raw: Record<string, any> = {
  // @ts-expect-error,
  ...globalThis.$SST_LINKS,
};

const environment = {
  ...env,
  ...globalThis.process?.env,
};

// Handle consolidated resources JSON (for Windows with many resources)
if (environment.SST_RESOURCES_JSON) {
  try {
    const allResources = JSON.parse(environment.SST_RESOURCES_JSON);
    Object.assign(raw, allResources);
  } catch (error) {
    console.error("Failed to parse SST_RESOURCES_JSON:", error);
  }
}

// Handle individual SST_RESOURCE_ environment variables
for (const [key, value] of Object.entries(environment)) {
  if (key.startsWith("SST_RESOURCE_") && value) {
    raw[key.slice("SST_RESOURCE_".length)] = JSON.parse(value);
  }
}

// @ts-expect-error
if (env.SST_KEY_FILE && env.SST_KEY && !globalThis.SST_KEY_FILE_DATA) {
  const key = Buffer.from(env.SST_KEY, "base64");
  const encryptedData = readFileSync(env.SST_KEY_FILE);
  const nonce = Buffer.alloc(12, 0);
  const decipher = crypto.createDecipheriv("aes-256-gcm", key, nonce);
  const authTag = encryptedData.subarray(-16);
  const actualCiphertext = encryptedData.subarray(0, -16);
  decipher.setAuthTag(authTag);
  let decrypted = decipher.update(actualCiphertext);
  decrypted = Buffer.concat([decrypted, decipher.final()]);
  const decryptedData = JSON.parse(decrypted.toString());
  Object.assign(raw, decryptedData);
}

// @ts-expect-error
if (globalThis.SST_KEY_FILE_DATA) {
  // @ts-expect-error
  Object.assign(raw, globalThis.SST_KEY_FILE_DATA);
}
export function fromCloudflareEnv(input: any) {
  for (let [key, value] of Object.entries(input)) {
    if (typeof value === "string") {
      try {
        value = JSON.parse(value);
      } catch {}
    }
    raw[key] = value;
    if (key.startsWith("SST_RESOURCE_")) {
      raw[key.replace("SST_RESOURCE_", "")] = value;
    }
  }
}

export function wrapCloudflareHandler(handler: any) {
  if (typeof handler === "function" && handler.hasOwnProperty("prototype")) {
    return class extends handler {
      constructor(ctx: any, env: any) {
        fromCloudflareEnv(env);
        super(ctx, env);
      }
    };
  }

  function wrap(fn: any) {
    return function (req: any, env: any, ...rest: any[]) {
      fromCloudflareEnv(env);
      return fn(req, env, ...rest);
    };
  }

  const result = {} as any;
  for (const [key, value] of Object.entries(handler)) {
    result[key] = wrap(value);
  }
  return result;
}

export const Resource = new Proxy(raw, {
  get(_target, prop: string) {
    if (prop in raw) {
      return raw[prop];
    }
    if (!env.SST_RESOURCE_App) {
      throw new Error(
        "It does not look like SST links are active. If this is in local development and you are not starting this process through the multiplexer, wrap your command with `sst dev -- <command>`",
      );
    }
    let msg = `"${prop}" is not linked in your sst.config.ts`;
    if (env.AWS_LAMBDA_FUNCTION_NAME) {
      msg += ` to ${env.AWS_LAMBDA_FUNCTION_NAME}`;
    }
    throw new Error(msg);
  },
}) as Resource;
