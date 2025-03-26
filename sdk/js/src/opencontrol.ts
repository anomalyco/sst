import { tool } from "opencontrol/tool";
import { client } from "./aws/client.js";
import { Resource } from "./resource.js";
import { z } from "zod";
import AWS from "aws-sdk";

/**
 * A list of OpenControl tools provided by SST. Currently, it includes tools that
 * can:
 *
 * - Lists the resources in your SST app.
 * - Access the resources in your AWS account.
 *
 * You can add this tool to your OpenControl server by passing it to the `tools`
 * option when creating it.
 *
 * @example
 * ```js title="src/server.ts"
 * import { create } from "opencontrol";
 * import { tools } from "sst/opencontrol";
 *
 * const app = create({
 *   model: // ...
 *   tools: [...tools]
 * });
 * ```
 */
export const tools = [
  /*
  tool({
    name: "sst",
    description: "Get the resources in the current SST app",
    async run() {
      const c = await client();
      const stateBucket = await getStateBucket();
      if (!stateBucket)
        throw new Error(
          "Failed to find the SST state bucket in user's AWS account. Ask the user to make sure the AWS account has been bootstrapped with SST."
        );

      const state = await getStateFile();
      if (!state)
        throw new Error(
          "Failed to find the SST state file in user's AWS account."
        );

      const resources = state["checkpoint"]["latest"]["resources"];
      return resources
        .filter(
          (r: any) =>
            r.type !== "sst:sst:LinkRef" &&
            !r.type.startsWith("pulumi:provider:")
        )
        .map((r: any) => ({
          urn: r.urn,
          type: r.type,
          id: r.id,
          parent: r.parent,
        }));

      async function getStateBucket() {
        const res = await c.fetch(`https://ssm.${c.region}.amazonaws.com/`, {
          method: "POST",
          headers: {
            "X-Amz-Target": "AmazonSSM.GetParameter",
            "Content-Type": "application/x-amz-json-1.1",
          },
          body: JSON.stringify({
            Name: "/sst/bootstrap",
          }),
        });
        if (!res.ok) return;

        const data = (await res.json()) as {
          Parameter: {
            Value: string;
          };
        };
        return JSON.parse(data.Parameter.Value)["state"];
      }

      async function getStateFile() {
        const res = await c.fetch(
          `https://${stateBucket}.s3.${c.region}.amazonaws.com/app/${Resource.App.name}/${Resource.App.stage}.json`,
          {
            method: "GET",
          }
        );
        if (!res.ok) return;
        return (await res.json()) as any;
      }
    },
  }),
  */
  tool({
    name: "aws",
    description: "Make a call to the AWS SDK for JavaScript v2",
    args: z.object({
      client: z.string().describe("Class name of the client to use"),
      command: z.string().describe("Command to call on the client"),
      args: z
        .record(z.string(), z.any())
        .optional()
        .describe("Arguments to pass to the command"),
    }),
    async run(input) {
      // @ts-ignore
      const client = new AWS[input.client]();
      return await client[input.command](input.args).promise();
    },
  }),
  tool({
    name: "aws_batch",
    description:
      "Make multiple calls to the AWS SDK for JavaScript v2 with the same command but different arguments. This tools takes an array of arguments, and will call the command with each argument in the array in parallel. Use this over the aws tool when you need to call the same command multiple times with different arguments.",
    args: z.object({
      client: z.string().describe("Class name of the client to use"),
      command: z.string().describe("Command to call on the client"),
      args: z
        .array(
          z
            .record(z.string(), z.any())
            .optional()
            .describe("Arguments to pass to the command")
        )
        .describe(
          "An array of arguments. Each argument will be passed to the command in parallel."
        ),
    }),
    async run(input) {
      // @ts-ignore
      const client = new AWS[input.client]();
      return await Promise.all(
        input.args.map((arg: any) => client[input.command](arg).promise())
      );
    },
  }),
];
