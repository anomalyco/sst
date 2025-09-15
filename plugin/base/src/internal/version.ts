import { ComponentResource, ComponentResourceOptions } from "@pulumi/pulumi";

export class VersionComponent extends ComponentResource {
  constructor(target: string, version: number, opts: ComponentResourceOptions) {
    super("sst:sst:Version", target + "Version", {}, opts);
    this.registerOutputs({ target, version });
  }

  static parse(version: string): ComponentVersion {
    const [major, minor] = version.split(".");
    return { major: parseInt(major), minor: parseInt(minor) };
  }
}
export type ComponentVersion = { major: number; minor: number };
