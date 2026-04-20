import {
  WorkflowEntrypoint,
  type WorkflowEvent,
  type WorkflowStep,
} from "cloudflare:workers";

interface Env {}

export default {
  async fetch() {
    return new Response("Workflow worker is ready.");
  },
};

export class SignupWorkflow extends WorkflowEntrypoint<Env> {
  async run(_event: WorkflowEvent<unknown>, step: WorkflowStep) {
    await step.do("log workflow run", async () => {
      console.log("Cloudflare Workflow ran.");
    });
  }
}
