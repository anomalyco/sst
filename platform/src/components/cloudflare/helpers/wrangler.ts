import fs from "fs";
import path from "path";
import type { Unstable_RawConfig as RawConfig } from "wrangler";
import { VisibleError } from "../../error.js";
import {
  composeWranglerConfig,
  replaceLinkName,
  type WranglerConfigSource,
} from "./wrangler-config.js";

export type WranglerCompatibility = {
  date: string;
  flags: string[];
};

export type WranglerLinkInclude = {
  type: string;
  config?: Partial<RawConfig>;
  [field: string]: unknown;
};

export type WranglerLink = {
  name: string;
  include: WranglerLinkInclude[];
  properties?: Record<string, unknown>;
};

export function createWranglerConfig(input: {
  appName: string;
  appStage: string;
  name: string;
  frameworkConfig?: Record<string, any>;
  compatibility: WranglerCompatibility;
  environment?: Record<string, string>;
  links?: WranglerLink[];
  accountId?: string;
}) {
  const baseConfig: Partial<RawConfig> = {
    name: sanitizeWranglerName(`sst-${input.appStage}-${input.name}`),
    compatibility_date: input.compatibility.date,
    compatibility_flags: input.compatibility.flags,
    ...(input.accountId ? { account_id: input.accountId } : {}),
    vars: {
      ...(input.environment ?? {}),
      SST_RESOURCE_App: JSON.stringify({
        name: input.appName,
        stage: input.appStage,
      }),
    },
  };

  const sources: WranglerConfigSource[] = [
    {
      owner: "framework",
      config: (input.frameworkConfig ?? {}) as Partial<RawConfig>,
      linkScoped: false,
    },
    { owner: "sst", config: baseConfig, linkScoped: false },
  ];

  for (const link of input.links ?? []) {
    const devConfigs = link.include
      .filter((item) => item.type === "cloudflare.dev")
      .map((item) => item.config);
    const binding = link.include.find(
      (item) => item.type === "cloudflare.binding",
    );

    if (devConfigs.length === 0) {
      // Non-native links are exposed as vars for local Wrangler consumers.
      if (!binding) {
        sources.push({
          owner: link.name,
          config: {
            vars: {
              [`SST_RESOURCE_${link.name}`]: JSON.stringify(
                link.properties ?? {},
              ),
            },
          },
          linkScoped: false,
        });
      }
      continue;
    }

    for (const config of devConfigs) {
      if (!config) {
        throw new VisibleError(
          `Cloudflare dev config is missing for ${link.name}`,
        );
      }
      sources.push({
        owner: link.name,
        config: replaceLinkName(link.name, config, link.name),
      });
    }
  }

  return composeWranglerConfig(sources);
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

const wranglerNameRegex = /[^a-z0-9-]+/g;

function sanitizeWranglerName(input: string) {
  const value = input.toLowerCase().replaceAll(wranglerNameRegex, "-");
  return value.replace(/^-+|-+$/g, "") || "sst";
}
