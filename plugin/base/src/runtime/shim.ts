export { config as $config } from "../config.js";
export {
  resolve as $resolve,
  json as $json,
  concat as $concat,
  interpolate as $interpolate,
} from "../util.js";

// deprecated
import { json } from "../util.js";
const { parse, stringify } = json;
export { parse as $jsonParse, stringify as $jsonStringify };
