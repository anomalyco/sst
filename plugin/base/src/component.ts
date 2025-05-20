import {
  ComponentResource,
  ComponentResourceOptions,
  Inputs,
  runtime,
  output,
  asset as pulumiAsset,
  Input,
  all,
  CustomResourceOptions,
} from "@pulumi/pulumi";
import { VisibleError } from "./error.js";
import path from "path";
import { statSync } from "fs";

/**
 * Helper type to inline nested types
 */
export type Prettify<T> = {
  [K in keyof T]: T[K];
} & {};

export type Transform<T> =
  | Partial<T>
  | ((args: T, opts: CustomResourceOptions, name: string) => undefined);

export function transform<T extends object>(
  transform: Transform<T> | undefined,
  name: string,
  args: T,
  opts: CustomResourceOptions
) {
  // Case: transform is a function
  if (typeof transform === "function") {
    transform(args, opts, name);
    return [name, args, opts] as const;
  }

  // Case: no transform
  // Case: transform is an argument
  return [name, { ...args, ...transform }, opts] as const;
}

export class Component extends ComponentResource {
  constructor(
    type: string,
    name: string,
    args?: Inputs,
    opts?: ComponentResourceOptions
  ) {
    const transforms = ComponentTransforms.get(type) ?? [];
    for (const transform of transforms) {
      transform({ name, props: args, opts });
    }
    super(type, name, args, {
      transformations: [
        (args) => {
          // Ensure component names do not contain spaces
          if (name.includes(" "))
            throw new Error(
              `Invalid component name "${name}" (${args.type}). Component names cannot contain spaces.`
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
              }`
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
      ...opts,
    });
  }
}

const ComponentTransforms = new Map<string, any[]>();
export function $transform<T, Args, Options>(
  resource: { new (name: string, args: Args, opts?: Options): T },
  cb: (args: Args, opts: Options, name: string) => void
) {
  // @ts-expect-error
  const type = resource.__pulumiType;
  if (type.startsWith("sst:")) {
    let transforms = ComponentTransforms.get(type);
    if (!transforms) {
      transforms = [];
      ComponentTransforms.set(type, transforms);
    }
    transforms.push((input: any) => {
      cb(input.props, input.opts, input.name);
      return input;
    });
    return;
  }
  runtime.registerStackTransformation((input) => {
    if (input.type !== type) return;
    cb(input.props as any, input.opts as any, input.name);
    return input;
  });
}

export function $asset(assetPath: string) {
  // TODO check process.cwd() resolves to the root of the project
  const fullPath = path.isAbsolute(assetPath)
    ? assetPath
    : path.join(process.cwd(), assetPath);

  try {
    return statSync(fullPath).isDirectory()
      ? new pulumiAsset.FileArchive(fullPath)
      : new pulumiAsset.FileAsset(fullPath);
  } catch (e) {
    throw new VisibleError(`Asset not found: ${fullPath}`);
  }
}

export function $lazy<T>(fn: () => T) {
  return output(undefined)
    .apply(async () => output(fn()))
    .apply((x) => x);
}

export function $print(...msg: Input<any>[]) {
  return all(msg).apply((msg) => console.log(...msg));
}
