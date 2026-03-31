/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS OpenAuth React SPA
 *
 * This is a full-stack monorepo app shows the OpenAuth flow for a single-page app
 * and an authenticated API. It has:
 *
 * - React SPA built with Vite and the `StaticSite` component in the `packages/web`
 *   directory.
 *   ```ts title="infra/web.ts"
 *   export const web = new sst.aws.StaticSite("MyWeb", {
 *     path: "packages/web",
 *     build: {
 *       output: "dist",
 *       command: "npm run build",
 *     },
 *     environment: {
 *       VITE_API_URL: api.url,
 *       VITE_AUTH_URL: auth.url,
 *     },
 *   });
 *   ```
 *
 * - API with Hono and the `Function` component in `packages/functions/src/api.ts`.
 *   ```ts title="infra/api.ts"
 *   export const api = new sst.aws.Function("MyApi", {
 *     url: true,
 *     link: [auth],
 *     handler: "packages/functions/src/api.handler",
 *   });
 *   ```
 *
 * - OpenAuth with the `Auth` component in `packages/functions/src/auth.ts`.
 *   ```ts title="infra/auth.ts"
 *   export const auth = new sst.aws.Auth("MyAuth", {
 *     issuer: "packages/functions/src/auth.handler",
 *   });
 *   ```
 *
 * The React frontend uses a `AuthContext` provider to manage the auth flow.
 *
 * ```tsx title="packages/web/src/AuthContext.tsx"
 * <AuthContext.Provider
 *   value={{
 *     login,
 *     logout,
 *     userId,
 *     loaded,
 *     loggedIn,
 *     getToken,
 *   }}
 * >
 *   {children}
 * </AuthContext.Provider>
 * ```
 *
 * Now in `App.tsx`, we can use the `useAuth` hook.
 *
 * ```tsx title="packages/web/src/App.tsx"
 * const auth = useAuth();
 *
 * return !auth.loaded ? (
 *   <div>Loading...</div>
 * ) : (
 *   <div>
 *     {auth.loggedIn ? (
 *       <div>
 *         <p>
 *           <span>Logged in</span>
 *           {auth.userId && <span> as {auth.userId}</span>}
 *         </p>
 *       </div>
 *     ) : (
 *       <button onClick={auth.login}>Login with OAuth</button>
 *     )}
 *   </div>
 * );
 * ```
 *
 * Once authenticated, we can call our authenticated API by passing in the access
 * token.
 *
 * ```tsx title="packages/web/src/App.tsx" {3}
 * await fetch(`${import.meta.env.VITE_API_URL}me`, {
 *   headers: {
 *     Authorization: `Bearer ${await auth.getToken()}`,
 *   },
 * });
 * ```
 *
 * The API uses the OpenAuth client to verify the token.
 *
 * ```ts title="packages/functions/src/api.ts" {3}
 * const authHeader = c.req.header("Authorization");
 * const token = authHeader.split(" ")[1];
 * const verified = await client.verify(subjects, token);
 * ```
 *
 * The `sst.config.ts` dynamically imports all the `infra/` files.
 */
export default $config({
  app(input) {
    return {
      name: "aws-auth-react",
      removal: input?.stage === "production" ? "retain" : "remove",
      protect: ["production"].includes(input?.stage),
      home: "aws",
    };
  },
  async run() {
    await import("./infra/auth");
    await import("./infra/api");
    await import("./infra/web");
  },
});
