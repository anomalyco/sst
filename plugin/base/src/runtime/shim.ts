export { config as $config } from "../config";
export {
  resolve as $resolve,
  json as $json,
  concat as $concat,
  interpolate as $interpolate,
} from "../util";
export { transform as $transform } from "../transform";

// deprecated
import { json } from "../util";
const { parse, stringify } = json;
export { parse as $jsonParse, stringify as $jsonStringify };
