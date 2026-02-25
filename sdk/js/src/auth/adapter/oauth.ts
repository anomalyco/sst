import { BaseClient, generators, Issuer, TokenSet } from "openid-client";
import { Adapter, AdapterError } from "./adapter.js";
import { Context } from "hono";
import { getCookie } from "hono/cookie";

export interface OauthBasicConfig {
  /**
   * The clientID provided by the third party oauth service
   */
  clientID: string;
  /**
   * The clientSecret provided by the third party oauth service
   */
  clientSecret: string;
  /**
   * Various scopes requested for the access token
   */
  scope: string;
  /**
   * Determines whether users will be prompted for reauthentication and consent
   */
  prompt?: string;
  /**
   * Additional parameters to be passed to the authorization endpoint
   */
  params?: Record<string, string>;
}

export interface OauthConfig extends OauthBasicConfig {
  issuer: Issuer;
}

export class OauthError extends AdapterError {}

export const OauthAdapter =
  /* @__PURE__ */
  (config: OauthConfig) => {
    return async function (routes, ctx) {
      function getClient(c: Context) {
        const callback = new URL(c.req.url);
        callback.pathname = callback.pathname.replace(
          /authorize.*$/,
          "callback"
        );
        callback.search = "";
        callback.host = c.req.header("x-forwarded-host") || callback.host;
        return [
          callback,
          new config.issuer.Client({
            client_id: config.clientID,
            client_secret: config.clientSecret,
            redirect_uris: [callback.toString()],
            response_types: ["code"],
          }),
        ] as const;
      }

      routes.get("/authorize", async (c) => {
        const [_, client] = getClient(c);
        const code_verifier = generators.codeVerifier();
        const state = generators.state();
        const code_challenge = generators.codeChallenge(code_verifier);
        const url = client.authorizationUrl({
          scope: config.scope,
          code_challenge: code_challenge,
          code_challenge_method: "S256",
          state,
          prompt: config.prompt,
          ...config.params,
        });
        ctx.cookie(c, "auth_code_verifier", code_verifier, 60 * 10);
        ctx.cookie(c, "auth_state", state, 60 * 10);
        return c.redirect(url);
      });

      routes.get("/callback", async (c) => {
        const [callback, client] = getClient(c);
        const query = c.req.query();
        if (query.error) {
          throw new OauthError(query.error);
        }
        const code_verifier = getCookie(c, "auth_code_verifier");
        const state = getCookie(c, "auth_state");
        const tokenset = await client[
          config.issuer.metadata.userinfo_endpoint
            ? "callback"
            : "oauthCallback"
        ](callback.toString(), query, {
          code_verifier,
          state,
        });
        return ctx.success(c, {
          client,
          tokenset,
        });
      });

      // response_mode=form_post
      routes.post("/callback", async (c) => {
        const [callback, client] = getClient(c);
        const form = await c.req.formData();
        if (form.get("error")) {
          throw new OauthError(form.get("error")!.toString());
        }
        const code_verifier = getCookie(c, "auth_code_verifier");
        const state = getCookie(c, "auth_state");
        const tokenset = await client[
          config.issuer.metadata.userinfo_endpoint
            ? "callback"
            : "oauthCallback"
        ](callback.toString(), Object.fromEntries(form as any), {
          code_verifier,
          state,
        });
        return ctx.success(c, {
          client,
          tokenset,
        });
      });
    } satisfies Adapter<{
      tokenset: TokenSet;
      client: BaseClient;
    }>;
  };
