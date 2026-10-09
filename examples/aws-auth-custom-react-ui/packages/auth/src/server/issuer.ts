import { streamHandle } from 'hono/aws-lambda'
import { issuer } from "@openauthjs/openauth";
import { PasswordProvider } from "@openauthjs/openauth/provider/password";
import { CustomPasswordUI } from "@openauthjs/openauth/ui/custom/password";
import { subjects } from "./subjects";

async function getUser(email: string) {
  return "123";
}

const app = issuer({
  subjects,
  allow: async () => true,
  providers: {
    password: PasswordProvider(
      CustomPasswordUI({
        sendCode: async (email, code) => {
          console.log(email, code);
        },
        loader: {
          login: async () => ({
            data: {
              serverText: "This came from the login loader",
            },
          }),
          register: async () => ({
            data: {
              serverText: "This came from the register loader",
            },
          }),
          change: async () => ({
            data: {
              serverText: "This came from the cahnge loader",
            },
          }),
        },
      }),
    ),
  },
  success: async (ctx, value) => {
    if (value.provider === "password") {
      return ctx.subject("user", {
        id: await getUser(value.email),
      });
    }
    throw new Error("Invalid provider");
  },
});
export const handler = streamHandle(app); // Remember! You should use an streamming handler (except you manually disable streamming in your server function)

export default app; // Always remember to have an default export with the app to enable the vite dev server.
