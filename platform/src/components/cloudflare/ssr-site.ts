import path from "path";
import fs from "fs";
import { Output, output, all, ComponentResourceOptions } from "@pulumi/pulumi";
import { Input } from "../input.js";
import { Component, transform, type Transform } from "../component.js";
import { VisibleError } from "../error.js";
import { BaseSsrSiteArgs, buildApp } from "../base/base-ssr-site.js";
import { Worker, WorkerArgs } from "./worker.js";
import { normalizeCompatibility } from "./helpers/compatibility.js";
import {
  createWranglerConfig,
  writeWranglerConfig,
} from "./helpers/wrangler.js";
import { Link } from "../link.js";
import { URL_UNAVAILABLE } from "../aws/linkable.js";

export type Plan = {
  server: string;
  assets: string;
};

export interface SsrSiteArgs extends BaseSsrSiteArgs {
  domain?: Input<string>;
  /**
   * [Transform](/docs/components#transform) how this component creates its underlying
   * resources.
   */
  transform?: {
    /**
     * Transform the Worker component used for handling the server-side rendering.
     */
    server?: Transform<WorkerArgs>;
  };
}

export abstract class SsrSite extends Component implements Link.Linkable {
  private server?: Worker;
  private devUrl?: Output<string>;

  protected validate(_sitePath: string): void {}

  protected abstract buildPlan(outputPath: Output<string>): Output<Plan>;

  protected buildWrangler(
    _sitePath: string,
  ): Input<Record<string, Input<any>> | undefined> {
    return undefined;
  }

  constructor(
    type: string,
    name: string,
    args: SsrSiteArgs = {},
    opts: ComponentResourceOptions = {},
  ) {
    super(type, name, args, opts);
    const self = this;

    const sitePath = normalizeSitePath();
    const compatibility = resolveCompatibility();
    const frameworkConfig = resolveFrameworkConfig();
    const wranglerConfig = resolveWranglerConfig();
    const dev = normalizeDev();
    this.validate(sitePath);

    if (dev.enabled) {
      this.devUrl = dev.url;
      this.registerOutputs({
        _dev: dev.outputs,
        _metadata: {
          mode: "placeholder",
          path: sitePath,
        },
      });
      return;
    }

    const outputPath = buildApp(
      self,
      name,
      args,
      sitePath,
      undefined,
      resolveBuildEnvironment(),
    );
    const plan = this.buildPlan(outputPath);
    const worker = createWorker();

    this.server = worker;

    this.registerOutputs({
      _hint: this.url,
      _metadata: {
        mode: "deployed",
        path: sitePath,
      },
    });

    function normalizeDev() {
      const enabled = $dev && args.dev !== false;
      const devArgs = args.dev || {};

      return {
        enabled,
        url: output(devArgs.url ?? URL_UNAVAILABLE),
        outputs: {
          title: devArgs.title,
          environment: args.environment,
          cloudflare: enabled
            ? {
                path: resolveDevWranglerPath(),
              }
            : undefined,
          command: output(devArgs.command ?? "npm run dev"),
          autostart: output(devArgs.autostart ?? true),
          directory: devArgs.directory ?? sitePath,
          links: output(args.link || [])
            .apply(Link.build)
            .apply((links) => links.map((link) => link.name)),
        },
      };
    }

    function resolveFrameworkConfig() {
      return output(self.buildWrangler(sitePath)).apply((config) => config ?? {});
    }

    function resolveCompatibility() {
      const [, workerArgs] = transform(
        args.transform?.server,
        `${name}Worker`,
        {
          environment: args.environment,
          link: args.link,
          url: true,
          dev: false,
          domain: args.domain,
          handler: output(""),
          assets: {
            directory: output(""),
          },
        },
        { parent: self },
      );
      return normalizeCompatibility(workerArgs);
    }

    function resolveBuildEnvironment() {
      return resolveBuildWranglerPath().apply((wranglerPath) => ({
        SST_WRANGLER_PATH: wranglerPath,
      }));
    }

    function resolveDevWranglerPath() {
      return wranglerConfig.apply((config) => {
        return writeWranglerConfig({
          workDir: $cli.paths.work,
          stage: $app.stage,
          name,
          config,
        });
      });
    }

    function resolveWranglerConfig() {
      return all([
        output(args.environment ?? {}),
        resolveLinkBindings(),
        compatibility,
        frameworkConfig,
      ]).apply(([environment, links, compatibility, frameworkConfig]) => {
        return createWranglerConfig({
          appName: $app.name,
          appStage: $app.stage,
          name,
          frameworkConfig,
          compatibility,
          environment,
          links,
          accountID: process.env.CLOUDFLARE_DEFAULT_ACCOUNT_ID,
        });
      });
    }

    function resolveBuildWranglerPath() {
      return wranglerConfig.apply((config) => {
        return writeWranglerConfig({
          workDir: $cli.paths.work,
          stage: $app.stage,
          name,
          config,
        });
      });
    }

    function resolveLinkBindings() {
      return output(args.link ?? []).apply((links) => {
        const linkBindings = links.filter(Link.isLinkable).map((link) =>
          output({
            urn: link.urn,
            link: link.getSSTLink(),
          }).apply(({ urn, link }) => ({
            name: urn.split("::").at(-1)!,
            include: link.include ?? [],
            properties: link.properties,
          })),
        );
        return linkBindings.length > 0 ? all(linkBindings) : [];
      });
    }

    function normalizeSitePath() {
      const sitePath = args.path ?? ".";

      if (!fs.existsSync(sitePath)) {
        throw new VisibleError(
          `Site directory not found at "${path.resolve(
            sitePath,
          )}". Please check the path setting in your configuration.`,
        );
      }

      return sitePath;
    }

    function createWorker() {
      return new Worker(
        ...transform(
          args.transform?.server,
          `${name}Worker`,
          {
            environment: args.environment,
            link: args.link,
            url: true,
            dev: false,
            domain: args.domain,
            handler: all([outputPath, plan.server]).apply(
              ([outputPath, server]) => path.join(outputPath, server),
            ),
            assets: {
              directory: all([outputPath, plan.assets]).apply(
                ([outputPath, assets]) => path.join(outputPath, assets),
              ),
            },
          },
          { parent: self },
        ),
      );
    }
  }

  /**
   * The URL of the site.
   *
   * If the `domain` is set, this is the URL with the custom domain.
   * Otherwise, it's the auto-generated Worker URL.
   */
  public get url() {
    if (this.server) return this.server.url;
    return this.devUrl!;
  }

  /**
   * The underlying [resources](/docs/components/#nodes) this component creates.
   */
  public get nodes() {
    return {
      /**
       * The Cloudflare Worker that renders the site.
       */
      server: this.server,
    };
  }

  /** @internal */
  public getSSTLink() {
    return {
      properties: {
        url: this.url,
      },
    };
  }
}
