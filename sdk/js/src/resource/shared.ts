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
  for (let [key, value] of Object.entries(input ?? {})) {
    if (typeof value === "string") {
      environment[key] = value;
      if (!key.startsWith("SST_RESOURCE_") || !value) {
        continue;
      }
      raw[key.slice("SST_RESOURCE_".length)] = JSON.parse(value);
      continue;
    }

    raw[key] = value;
  }
}

export function loadResourceData(input?: Record<string, any>) {
  Object.assign(raw, input ?? {});
}

export function createResource<T extends Resource>(load: () => void) {
  let loaded = false;
  const loadData = () => {
    if (loaded) return;
    load();
    loaded = true;
  };

  return new Proxy(raw, {
    get(_target, prop: string | symbol) {
      loadData();
      if (prop in raw) {
        return raw[prop as string];
      }
      if (typeof prop !== "string") {
        return undefined;
      }
      if (!environment.SST_RESOURCE_App && !raw.App) {
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
    has(_target, prop: string | symbol) {
      loadData();
      return prop in raw;
    },
    ownKeys() {
      loadData();
      return Reflect.ownKeys(raw);
    },
    getOwnPropertyDescriptor(_target, prop: string | symbol) {
      loadData();
      return Object.getOwnPropertyDescriptor(raw, prop);
    },
  }) as T;
}
