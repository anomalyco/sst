import type { APIEvent } from "@solidjs/start/server";
import { Resource } from "../util/resource";

export async function GET(input: APIEvent) {
  return Response.json(
    {
      secret: Resource.MySecret.value,
      foo: process.env.FOO,
    },
    { status: 200 }
  );
}
