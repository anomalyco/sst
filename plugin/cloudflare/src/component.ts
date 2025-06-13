import * as sst from "sst-plugin";
import { VisibleError } from "sst-plugin/error";
import { Component as BaseComponent } from "sst-plugin/component";
import { ComponentResource, ComponentResourceOptions } from "@pulumi/pulumi";

export class CloudflareComponent extends BaseComponent {
  private componentType: string;
  private componentName: string;

  constructor(
    type: string,
    name: string,
    args?: Record<string, sst.Input<any>>,
    opts?: sst.ComponentOptions,
  ) {
    super(type, name, args, {
      ...opts,
      transformations: [
        // Ensure logical and physical names are prefixed
        (args) => {
          // Ensure physical names are prefixed with app/stage
          // note: We are setting the default names here instead of inline when creating
          //       the resource is b/c the physical name is inferred from the logical name.
          //       And it's convenient to access the logical name here.
          if (args.type.startsWith("sst:")) return;
          if (
            [
              "cloudflare:index/record:Record",
              "cloudflare:index/workerCronTrigger:WorkerCronTrigger",
              "cloudflare:index/workerDomain:WorkerDomain",
            ].includes(args.type)
          )
            return;

          const namingRules: Record<
            string,
            [
              string,
              number,
              {
                lower?: boolean;
                replace?: (name: string) => string;
                suffix?: () => sst.Output<string>;
              }?,
            ]
          > = {
            "cloudflare:index/d1Database:D1Database": [
              "name",
              64,
              { lower: true },
            ],
            "cloudflare:index/r2Bucket:R2Bucket": ["name", 64, { lower: true }],
            "cloudflare:index/workerScript:WorkerScript": [
              "name",
              64,
              { lower: true },
            ],
            "cloudflare:index/queue:Queue": ["name", 64, { lower: true }],
            "cloudflare:index/workersKvNamespace:WorkersKvNamespace": [
              "title",
              64,
              { lower: true },
            ],
          };

          const rule = namingRules[args.type];
          if (!rule)
            throw new VisibleError(
              `In "${name}" component, the physical name of "${args.name}" (${args.type}) is not prefixed`,
            );

          // name is already set
          const nameField = rule[0];
          const length = rule[1];
          const options = rule[2];
          if (args.props[nameField] && args.props[nameField] !== "") return;

          // Handle prefix field is tags
          if (nameField === "tags") {
            return {
              props: {
                ...args.props,
                tags: {
                  // @ts-expect-error
                  ...args.tags,
                  Name: sst.naming.prefix(length, args.name),
                },
              },
              opts: args.opts,
            };
          }

          // Handle prefix field is name
          const suffix = options?.suffix ? options.suffix() : sst.output("");
          return {
            props: {
              ...args.props,
              [nameField]: suffix.apply((suffix) => {
                let v = options?.lower
                  ? sst.naming.physical(length, args.name, suffix).toLowerCase()
                  : sst.naming.physical(length, args.name, suffix);
                if (options?.replace) v = options.replace(v);
                return v;
              }),
            },
            opts: {
              ...args.opts,
              ignoreChanges: [...(args.opts.ignoreChanges ?? []), nameField],
            },
          };
        },
        ...(opts?.transformations || []),
      ],
    });

    this.componentType = type;
    this.componentName = name;
  }

  /** @internal */
  protected registerVersion(input: {
    new: number;
    old?: number;
    message?: string;
    forceUpgrade?: `v${number}`;
  }) {
    // Check component version
    const oldVersion = input.old;
    const newVersion = input.new ?? 1;
    if (oldVersion) {
      const className = this.componentType.replaceAll(":", ".");
      // Invalid forceUpgrade value
      if (input.forceUpgrade && input.forceUpgrade !== `v${newVersion}`) {
        throw new VisibleError(
          [
            `The value of "forceUpgrade" does not match the version of "${className}" component.`,
            `Set "forceUpgrade" to "v${newVersion}" to upgrade to the new version.`,
          ].join("\n"),
        );
      }
      // Version upgraded without forceUpgrade
      if (oldVersion < newVersion && !input.forceUpgrade) {
        throw new VisibleError(input.message ?? "");
      }
      // Version downgraded
      if (oldVersion > newVersion) {
        throw new VisibleError(
          [
            `It seems you are trying to use an older version of "${className}".`,
            `You need to recreate this component to rollback - https://sst.dev/docs/components/#versioning`,
          ].join("\n"),
        );
      }
    }

    // Set version
    if (newVersion > 1) {
      new Version(this.componentName, newVersion, { parent: this });
    }
  }
}

export class Version extends ComponentResource {
  constructor(target: string, version: number, opts: ComponentResourceOptions) {
    super("sst:sst:Version", target + "Version", {}, opts);
    this.registerOutputs({ target, version });
  }
}

// Keep backward compatibility with the old Component export
export const Component = CloudflareComponent;
