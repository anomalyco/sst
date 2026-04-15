export interface Resource {
  App: {
    name: string;
    stage: string;
  };
}

const raw: Record<string, any> = {
  // @ts-expect-error
  ...globalThis.$SST_LINKS,
};

const environment: Record<string, string | undefined> = {};

export function loadResourceEnvironment(input?: Record<string, any>) {
  for (const [key, value] of Object.entries(input ?? {})) {
    if (typeof value === "string") {
      environment[key] = value;
    }
    if (!key.startsWith("SST_RESOURCE_") || !value) {
      continue;
    }
    raw[key.slice("SST_RESOURCE_".length)] = JSON.parse(value as string);
  }
}

export function loadResourceData(input?: Record<string, any>) {
  Object.assign(raw, input ?? {});
}

export function fromCloudflareEnv(input: any) {
  for (let [key, value] of Object.entries(input)) {
    if (typeof value === "string") {
      environment[key] = value;
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
    if (!environment.SST_RESOURCE_App) {
      throw new Error(
        "It does not look like SST links are active. If this is in local development and you are not starting this process through the multiplexer, wrap your command with `sst dev -- <command>`",
      );
    }
    let msg = `"${prop}" is not linked in your sst.config.ts`;
    if (environment.AWS_LAMBDA_FUNCTION_NAME) {
      msg += ` to ${environment.AWS_LAMBDA_FUNCTION_NAME}`;
    }
    throw new Error(msg);
  },
}) as Resource;
