export * from "./realtime/index.js";
export * from "./resource.js";
export * from "./vector/index.js";

import { format } from "util";

if (
  process.env?.ECS_CONTAINER_METADATA_URI_V4 &&
  !process.env.SST_DISABLE_LOG_POLYFILL
) {
  const log =
    (level: "INFO" | "WARN" | "TRACE" | "DEBUG" | "ERROR") =>
    (msg: string, ...rest: any[]) => {
      let line = `${level}\t${format(msg, ...rest)}`;
      line = line.replace(/\n/g, "\r");
      process.stdout.write(line + "\n");
    };
  console.log = log("INFO");
  console.warn = log("WARN");
  console.error = log("ERROR");
  console.trace = log("TRACE");
  console.debug = log("DEBUG");
}
