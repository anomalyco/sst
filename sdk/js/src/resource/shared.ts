export interface Resource {
  App: {
    name: string;
    stage: string;
  };
}

const state = globalThis as typeof globalThis & {
  __SST_RESOURCE_RAW__?: Record<string, any>;
  __SST_RESOURCE_ENVIRONMENT__?: Record<string, string | undefined>;
};

const raw: Record<string, any> = (state.__SST_RESOURCE_RAW__ ??= {
  // @ts-expect-error
  ...globalThis.$SST_LINKS,
});

const environment: Record<string, string | undefined> =
  (state.__SST_RESOURCE_ENVIRONMENT__ ??= {});

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

export function loadFromCloudflareEnv(input: any) {
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
