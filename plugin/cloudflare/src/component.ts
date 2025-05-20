import {
  ComponentResource,
  ComponentResourceOptions,
  Inputs,
  output,
  Output,
} from "@pulumi/pulumi";
import { prefixName, physicalName } from "sst-plugin/naming";
import { VisibleError } from "sst-plugin/error";
import { Component as BaseComponent } from "sst-plugin/component";

export class Component extends BaseComponent {
  constructor(
    type: string,
    name: string,
    args?: Inputs,
    opts?: ComponentResourceOptions
  ) {
    super(type, name, args, {
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
                suffix?: () => Output<string>;
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
              `In "${name}" component, the physical name of "${args.name}" (${args.type}) is not prefixed`
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
                  Name: prefixName(length, args.name),
                },
              },
              opts: args.opts,
            };
          }

          // Handle prefix field is name
          const suffix = options?.suffix ? options.suffix() : output("");
          return {
            props: {
              ...args.props,
              [nameField]: suffix.apply((suffix) => {
                let v = options?.lower
                  ? physicalName(length, args.name, suffix).toLowerCase()
                  : physicalName(length, args.name, suffix);
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
      ],
      ...opts,
    });
  }
}
