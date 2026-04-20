export * from "./session.js";
export * from "./handler.js";
export { Issuer } from "openid-client";

import { AuthHandler } from "./handler.js";
import { createSessionBuilder } from "./session.js";

export namespace auth {
  export type Issuer = import("openid-client").Issuer;
  export const authorizer = AuthHandler;
  export const sessions = createSessionBuilder;
}
