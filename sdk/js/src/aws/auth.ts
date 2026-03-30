import { Handler } from "aws-lambda";
import { Adapter } from "../auth/adapter/adapter.js";
import { AuthHandler } from "../auth/handler.js";
import { SessionBuilder, createSessionBuilder } from "../auth/session.js";
import { handle, streamHandle } from "hono/aws-lambda";

export namespace auth {
  export type Issuer = import("openid-client").Issuer;
  export function authorizer<
    Providers extends Record<string, Adapter<any>>,
    Sessions extends SessionBuilder,
  >(...args: Parameters<typeof AuthHandler<Providers, Sessions>>) {
    const hono = AuthHandler(...args);
    return (
      args[0].stream ? streamHandle(hono) : handle(hono)
    ) as Handler<any, any>;
  }
  export const sessions = createSessionBuilder;
}
