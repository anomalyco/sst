import { CustomResourceOptions, Input, Output, dynamic, output } from "@pulumi/pulumi";
import { rpc } from "../rpc/rpc.js";
import { BaseSiteFileOptions } from "./base-site.js";

interface NormalizedFileOption {
  files: string[];
  ignore?: string[];
  cacheControl?: string;
  contentType?: string;
}

export interface StaticSiteManifestFile {
  source: string;
  key: string;
  hash: string;
  contentType: string;
  cacheControl?: string;
}

export interface StaticSiteManifestInputs {
  sitePath: Input<string>;
  outputPath: Input<string>;
  buildCommand?: Input<string | undefined>;
  environment?: Input<Record<string, Input<string>>>;
  fileOptions: Input<BaseSiteFileOptions[]>;
  textEncoding: Input<string>;
  keyPrefix?: Input<string | undefined>;
  assetPath?: Input<string | undefined>;
  assetRoutes?: Input<string[]>;
  bucketDomain?: Input<string | undefined>;
  errorPage?: Input<string | undefined>;
  base?: Input<string | undefined>;
  trigger: Input<string>;
}

export interface StaticSiteManifest {
  files: Output<StaticSiteManifestFile[]>;
  assetManifest: Output<Record<string, string>>;
  kvEntries: Output<Record<string, string>>;
  invalidationVersion: Output<string>;
  outputPath: Output<string>;
}

export class StaticSiteManifest extends dynamic.Resource {
  constructor(
    name: string,
    args: StaticSiteManifestInputs,
    opts?: CustomResourceOptions,
  ) {
    super(
      new rpc.Provider("StaticSite.Manifest"),
      `${name}.sst.StaticSiteManifest`,
      {
        ...args,
        files: undefined,
        assetManifest: undefined,
        kvEntries: undefined,
        invalidationVersion: undefined,
        fileOptions: output(args.fileOptions).apply(normalizeFileOptions),
      },
      opts,
    );
  }
}

function normalizeFileOptions(fileOptions: BaseSiteFileOptions[]) {
  return fileOptions.map((fileOption) => ({
    files: Array.isArray(fileOption.files) ? fileOption.files : [fileOption.files],
    ignore: !fileOption.ignore
      ? undefined
      : Array.isArray(fileOption.ignore)
        ? fileOption.ignore
        : [fileOption.ignore],
    cacheControl: fileOption.cacheControl,
    contentType: fileOption.contentType,
  })) satisfies NormalizedFileOption[];
}
