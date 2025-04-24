import Redis from "ioredis";
import { Resource } from "sst";

const client = new Redis({
  host: Resource.MyRedis.host,
  port: Resource.MyRedis.port,
  username: Resource.MyRedis.username,
  password: Resource.MyRedis.password,
  tls: {
    checkServerIdentity: () => undefined,
  },
});

export async function handler() {
  await client.set("foo", `bar-${Date.now()}`);
  return {
    statusCode: 200,
    body: JSON.stringify({
      foo: await client.get("foo"),
    }),
  };
}
