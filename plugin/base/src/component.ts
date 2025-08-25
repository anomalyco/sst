import type { ComponentResourceOptions } from "@pulumi/pulumi";

import {
  ComponentResource,
  Inputs,
  runtime,
  output,
  Input,
  all,
} from "@pulumi/pulumi";

export type ComponentOptions = ComponentResourceOptions;

export class Component extends ComponentResource {
  constructor(
    type: string,
    name: string,
    args?: Inputs,
    opts?: ComponentOptions,
  ) {
    const transforms = ComponentTransforms.get(type) ?? [];
    for (const transform of transforms) {
      transform({ name, props: args, opts });
    }
    super(type, name, args, {
      ...opts,
      transformations: [
        (args) => {
          // Ensure component names do not contain spaces
          if (name.includes(" "))
            throw new Error(
              `Invalid component name "${name}" (${args.type}). Component names cannot contain spaces.`,
            );

          // Ensure names are prefixed with parent's name
          if (
            args.type !== type &&
            // @ts-expect-error
            !args.name.startsWith(args.opts.parent!.__name)
          ) {
            throw new Error(
              `In "${name}" component, the logical name of "${args.name}" (${
                args.type
              }) is not prefixed with parent's name ${
                // @ts-expect-error
                args.opts.parent!.__name
              }`,
            );
          }

          return {
            props: args.props,
            opts: args.opts,
          };
        },
        // Set child resources `retainOnDelete` if set on component
        (args) => ({
          props: args.props,
          opts: {
            ...args.opts,
            retainOnDelete: args.opts.retainOnDelete ?? opts?.retainOnDelete,
          },
        }),
        ...(opts?.transformations ?? []),
      ],
    });
  }
}

export const ComponentTransforms = new Map<string, any[]>();

export function $lazy<T>(fn: () => T) {
  return output(undefined)
    .apply(async () => output(fn()))
    .apply((x) => x);
}

export function $print(...msg: Input<any>[]) {
  return all(msg).apply((msg) => console.log(...msg));
}
