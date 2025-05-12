import { request } from "http";
import { definitions } from "./resource.js";

export async function createRPC() {
  const url = new URL(process.env.SST_SERVER! + "/rpc/tunnel");
  const options = {
    hostname: url.hostname,
    port: url.port,
    path: url.pathname,
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
  };
  const req = request(options, (res) => {
    res.socket.unref();
    res.on("data", async (chunk) => {
      console.log(chunk.toString());
      const parsed = JSON.parse(chunk.toString());
      console.log(parsed);
      const method = parsed.method;
      const params = parsed.params;
      const result = await (async () => {
        switch (method) {
          case "resource.update": {
            const def = definitions[params.type];
            return (
              (await def.update?.(params.name, params.state, params.inputs)) ||
              params.state.outputs
            );
          }
          case "resource.create": {
            const def = definitions[params.type];
            return await def.create(params.inputs);
          }
          case "resource.delete": {
            const def = definitions[params.type];
            await def.delete?.(params.name, params.state);
            return;
          }
          default:
            throw new Error(`Unknown method "${method}"`);
        }
      })();
      fetch(process.env.SST_SERVER! + "/rpc/response", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          id: parsed.id,
          result,
        }),
      });
    });
  });

  req.end();
}
