import { handle } from "hono/aws-lambda";
import { create } from "opencontrol";
import { tools } from "sst/opencontrol";

const app = create({
  key: process.env.OPENCONTROL_KEY,
  tools,
});

export const handler = handle(app);

//const tools = [workout, aws, ...sstTools];
//const batch = tool({
//  name: "batch",
//  description: "Run another tool multiple times with different arguments",
//  args: z.object({
//    tool: z
//      .enum(tools.map((t) => t.name) as [string, ...string[]])
//      .describe("The tool to run"),
//    args: z
//      .array(z.any())
//      .describe("An array of arguments to pass to the tool"),
//  }),
//  async run(input) {
//    const tool = tools.find((t) => t.name === input.tool);
//    if (!tool) throw new Error(`Tool ${input.tool} not found`);
//
//    // Validate each argument against the tool's schema if it exists
//    if (tool.args) {
//      input.args.forEach((arg, index) => {
//        try {
//          tool.args.parse(arg);
//        } catch (error) {
//          throw new Error(
//            `Argument at index ${index} is invalid for tool ${input.tool}: ${error.message}`
//          );
//        }
//      });
//    }
//
//    return await Promise.all(input.args.map((arg) => tool.run(arg)));
//  },
//});
