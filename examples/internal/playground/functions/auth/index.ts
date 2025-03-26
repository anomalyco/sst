import { Resource } from "sst";
import { authorizer } from "@openauthjs/openauth";
import { handle } from "hono/aws-lambda";
import { subjects } from "./subjects.js";
import { PasswordAdapter } from "@openauthjs/openauth/adapter/password";
import { PasswordUI } from "@openauthjs/openauth/ui/password";
import { GoogleAdapter } from "@openauthjs/openauth/adapter/google";
import { DynamoStorage } from "@openauthjs/openauth/storage/dynamo";

const app = authorizer({
  subjects,
  storage: DynamoStorage({
    table: Resource.RouterAuthStorage.name,
    pk: "pk",
    sk: "sk",
  }),
  providers: {
    password: PasswordAdapter(
      PasswordUI({
        sendCode: async (email, code) => {
          console.log(email, code);
        },
      })
    ),
    google: GoogleAdapter({
      clientID: Resource.GOOGLE_CLIENT_ID.value,
      clientSecret: Resource.GOOGLE_CLIENT_SECRET.value,
      scopes: ["email"],
    }),
  },
  success: async (ctx, value) => {
    if (value.provider === "password") {
      return ctx.subject("user", {
        email: value.email,
      });
    }
    if (value.provider === "google") {
      return ctx.subject("user", {
        email: value.tokenset.access,
      });
    }
    throw new Error("Invalid provider");
  },
});

// @ts-ignore
export const handler = handle(app);
