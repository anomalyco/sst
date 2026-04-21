import fs from "fs";
import path from "path";

export type WranglerCompatibility = {
  date: string;
  flags: string[];
};

export type WranglerLinkInclude = {
  type: string;
  binding?: string;
  properties?: Record<string, unknown>;
};

export type WranglerLink = {
  name: string;
  include: WranglerLinkInclude[];
};

export function createWranglerConfig(input: {
  appStage: string;
  name: string;
  frameworkConfig?: Record<string, any>;
  compatibility: WranglerCompatibility;
  environment?: Record<string, string>;
  links?: WranglerLink[];
  accountID?: string;
}) {
  const config: Record<string, any> = {
    ...(input.frameworkConfig ?? {}),
    name: sanitizeWranglerName(`sst-${input.appStage}-${input.name}`),
    compatibility_date: input.compatibility.date,
    compatibility_flags: input.compatibility.flags,
  };

  if (input.accountID) {
    config.account_id = input.accountID;
  }

  const vars = { ...(input.environment ?? {}) };
  const kvNamespaces: Record<string, any>[] = [];
  const r2Buckets: Record<string, any>[] = [];
  const d1Databases: Record<string, any>[] = [];
  const hyperdrives: Record<string, any>[] = [];
  const services: Record<string, any>[] = [];
  const queueProducers: Record<string, any>[] = [];
  let ai: Record<string, any> | undefined;
  let versionMetadata: Record<string, any> | undefined;

  for (const link of input.links ?? []) {
    const binding = link.include.find(
      (item) => item.type === "cloudflare.binding",
    );
    if (!binding) continue;

    const properties = binding.properties ?? {};
    switch (binding.binding) {
      case "aiBindings":
        ai = {
          binding: link.name,
          remote: true,
        };
        break;
      case "kvNamespaceBindings":
        kvNamespaces.push({
          binding: link.name,
          id: stringValue(properties.namespaceId),
          remote: true,
        });
        break;
      case "secretTextBindings":
      case "plainTextBindings":
        vars[link.name] = stringValue(properties.text);
        break;
      case "serviceBindings":
        services.push({
          binding: link.name,
          service: stringValue(properties.service),
          remote: true,
        });
        break;
      case "queueBindings":
        queueProducers.push({
          binding: link.name,
          queue: stringValue(properties.queueName),
          remote: true,
        });
        break;
      case "r2BucketBindings":
        r2Buckets.push({
          binding: link.name,
          bucket_name: stringValue(properties.bucketName),
          remote: true,
        });
        break;
      case "d1DatabaseBindings":
        d1Databases.push({
          binding: link.name,
          database_id: stringValue(properties.id),
          remote: true,
        });
        break;
      case "hyperdriveBindings":
        hyperdrives.push({
          binding: link.name,
          id: stringValue(properties.id),
        });
        break;
      case "versionMetadataBindings":
        versionMetadata = {
          binding: link.name,
        };
        break;
    }
  }

  if (Object.keys(vars).length > 0) {
    config.vars = vars;
  }
  if (kvNamespaces.length > 0) {
    config.kv_namespaces = kvNamespaces;
  }
  if (r2Buckets.length > 0) {
    config.r2_buckets = r2Buckets;
  }
  if (d1Databases.length > 0) {
    config.d1_databases = d1Databases;
  }
  if (hyperdrives.length > 0) {
    config.hyperdrive = hyperdrives;
  }
  if (services.length > 0) {
    config.services = services;
  }
  if (queueProducers.length > 0) {
    config.queues = {
      producers: queueProducers,
    };
  }
  if (ai) {
    config.ai = ai;
  }
  if (versionMetadata) {
    config.version_metadata = versionMetadata;
  }

  return config;
}

export function writeWranglerConfig(args: {
  workDir: string;
  stage: string;
  name: string;
  config: Record<string, any>;
}) {
  const wranglerPath = path.join(
    args.workDir,
    "wrangler",
    args.stage,
    `${args.name}.jsonc`,
  );
  const contents = JSON.stringify(args.config, null, 2);

  fs.mkdirSync(path.dirname(wranglerPath), { recursive: true });
  if (
    !fs.existsSync(wranglerPath) ||
    fs.readFileSync(wranglerPath, "utf-8") !== contents
  ) {
    fs.writeFileSync(wranglerPath, contents);
  }

  return wranglerPath;
}

function stringValue(input: unknown) {
  return typeof input === "string" ? input : "";
}

const wranglerNameRegex = /[^a-z0-9-]+/g;

function sanitizeWranglerName(input: string) {
  const value = input.toLowerCase().replaceAll(wranglerNameRegex, "-");
  return value.replace(/^-+|-+$/g, "") || "sst";
}
