import { ComponentResourceOptions, all } from "@pulumi/pulumi";
import { Component } from "../component";

/**
 * The `Group` component lets you define a named set of components that can be
 * targeted together using `sst deploy --target` or excluded with `--exclude`.
 *
 * @example
 *
 * ```ts
 * const bucket = new sst.aws.Bucket("MyBucket");
 * const fn = new sst.aws.Function("MyFn", { handler: "src/handler.ts" });
 *
 * const api = new sst.x.Group("Api");
 * api.add(bucket, fn);
 * ```
 *
 * Now you can deploy just the `Api` group:
 * ```bash
 * sst deploy --target Api
 * ```
 */
export class Group extends Component {
  private members: Component[] = [];

  constructor(name: string, opts?: ComponentResourceOptions) {
    super("sst:sst:Group", name, {}, opts);
  }

  /**
   * Add components to this group.
   *
   * @example
   * ```ts
   * const bucket = new sst.aws.Bucket("MyBucket");
   * const fn = new sst.aws.Function("MyFn", { handler: "src/handler.ts" });
   *
   * const api = new sst.x.Group("Api");
   * api.add(bucket, fn);
   * ```
   */
  add(...members: Component[]) {
    this.members.push(...members);
    this.registerOutputs({
      _group: {
        members: all(this.members.map((m) => m.urn)),
      },
    });
  }
}
