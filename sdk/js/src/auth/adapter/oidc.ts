import { BaseClient, generators, Issuer, TokenSet } from "openid-client";
import { Adapter } from "./adapter.js";
import { getCookie } from "hono/cookie";

export interface OidcBasicConfig {
  /**
   * The clientID provided by the third party oauth service
   */
  clientID: string;
  /**
   * Determines whether users will be prompted for reauthentication and consent
   */
  prompt?: string;
}

export interface OidcConfig extends OidcBasicConfig {
  issuer: Issuer;
  scope: string;
}

export const OidcAdapter = /* @__PURE__ */ (config: OidcConfig) => {
  return async function (routes, ctx) {
    routes.get("/authorize", async (c) => {
      const callback = new URL(c.req.url);
      callback.pathname = callback.pathname.replace(/authorize.*$/, "callback");
      callback.search = "";
      callback.host = c.req.header("x-forwarded-host") || callback.host;

      const client = new config.issuer.Client({
        client_id: config.clientID,
        redirect_uris: [callback.toString()],
        response_types: ["id_token"],
      });
      const nonce = generators.nonce();
      const state = generators.state();
      const url = client.authorizationUrl({
        scope: config.scope,
        response_mode: "form_post",
        nonce,
        state,
        prompt: config.prompt,
      });
      ctx.cookie(c, "auth_nonce", nonce, 60 * 10);
      ctx.cookie(c, "auth_state", state, 60 * 10);
      return c.redirect(url);
    });

    routes.post("/callback", async (c) => {
      const callback = c.req.url.replace(/authorize\/.*$/, "callback");
      const client = new config.issuer.Client({
        client_id: config.clientID,
        redirect_uris: [callback],
        response_types: ["id_token"],
      });

      const form = await c.req.formData();
      const nonce = getCookie(c, "auth_nonce");
      const state = getCookie(c, "auth_state");
      const tokenset = await client.callback(
        callback,
        Object.fromEntries(form as any),
        {
          nonce,
          state,
        }
      );

      return ctx.success(c, {
        tokenset,
        client,
      });
    });
  } satisfies Adapter<{
    tokenset: TokenSet;
    client: BaseClient;
  }>;
};
