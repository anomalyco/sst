import path from "path";
import fs from "fs";
import { Output, output, all, ComponentResourceOptions } from "@pulumi/pulumi";
import { Input } from "../input.js";
import { Component, transform, type Transform } from "../component.js";
import { VisibleError } from "../error.js";
import { BaseSsrSiteArgs, buildApp } from "../base/base-ssr-site.js";
import { Worker, WorkerArgs } from "./worker.js";
import { normalizeCompatibility } from "./helpers/compatibility.js";
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

  protected abstract buildPlan(
    outputPath: Output<string>,
    name: string,
    args: SsrSiteArgs,
  ): Output<Plan>;

  constructor(
    type: string,
    name: string,
    args: SsrSiteArgs = {},
    opts: ComponentResourceOptions = {},
  ) {
    super(type, name, args, opts);
    const self = this;

    const sitePath = normalizeSitePath();
    const dev = normalizeDev();

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

    const outputPath = buildApp(self, name, args, sitePath);
    const plan = validatePlan(this.buildPlan(outputPath, name, args));
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
                compatibility: resolveCompatibility(),
              }
            : undefined,
          command: output(devArgs.command ?? "npm run dev"),
          autostart: output(devArgs.autostart ?? true),
          directory: output(devArgs.directory ?? sitePath),
          links: output(args.link || [])
            .apply(Link.build)
            .apply((links) => links.map((link) => link.name)),
        },
      };
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

    function normalizeSitePath() {
      return output(args.path).apply((sitePath) => {
        if (!sitePath) return ".";

        if (!fs.existsSync(sitePath)) {
          throw new VisibleError(
            `Site directory not found at "${path.resolve(
              sitePath,
            )}". Please check the path setting in your configuration.`,
          );
        }
        return sitePath;
      });
    }

    function validatePlan(plan: Output<Plan>) {
      return plan;
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
