import * as util from "@pulumi/pulumi";
import { Link } from "../components/link.js";
import { $config } from "../config.js";
import { $transform, $asset } from "../components/component.js";

const $secrets = JSON.parse(process.env.SST_SECRETS || "{}");
const { output, apply, all, interpolate, concat, jsonParse, jsonStringify } =
  util;

const linkable = Link.makeLinkable;
export {
  linkable as "$linkable",
  output as "$output",
  apply as "$apply",
  all as "$resolve",
  interpolate as "$interpolate",
  concat as "$concat",
  jsonParse as "$jsonParse",
  jsonStringify as "$jsonStringify",
  util as "$util",
  $asset as "$asset",
  $config as "$config",
  $transform as "$transform",
  $secrets as "$secrets",
};
