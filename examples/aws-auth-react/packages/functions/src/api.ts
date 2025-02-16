import { Hono } from "hono";
import { Resource } from "sst";
import { handle } from "hono/aws-lambda";
import { createClient } from "@openauthjs/openauth/client";
import { subjects } from "./subjects";

const client = createClient({
  clientID: "jwt-api",
  issuer: Resource.MyAuth.url,
});

async function getUserInfo(userId: string) {
  // Get user from database
  return {
    userId,
    name: "Patrick Star",
  };
}

const app = new Hono();

app.get("/me", async (c) => {
  const authHeader = c.req.header("Authorization");

  if (!authHeader) {
    return c.status(401);
  }

  const token = authHeader.split(" ")[1];
  const verified = await client.verify(subjects, token);

  if (verified.err) {
    return c.status(401);
  }

  return c.json(await getUserInfo(verified.subject.properties.id));
});

export const handler = handle(app);
