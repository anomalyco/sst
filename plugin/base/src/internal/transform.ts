import { CustomResourceOptions } from "@pulumi/pulumi";
import { ComponentOptions } from "../component.js";

export type Transform<T> =
  | Partial<T>
  | ((args: T, opts: ComponentOptions, name: string) => undefined);

export function transform<T extends object>(
  transform: Transform<T> | undefined,
  name: string,
  args: T,
  opts: CustomResourceOptions,
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
