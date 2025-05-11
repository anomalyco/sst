export {
  output,
  all as resolve,
  interpolate,
  concat,
  Input,
} from "@pulumi/pulumi";

import { jsonParse, jsonStringify } from "@pulumi/pulumi";

export const json = {
  parse: jsonParse,
  stringify: jsonStringify,
};
