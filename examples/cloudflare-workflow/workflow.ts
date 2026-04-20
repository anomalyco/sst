import {
  WorkflowEntrypoint,
  WorkflowEvent,
  WorkflowStep,
} from "cloudflare:workers";

type Params = {
  orderId: string;
};

export class OrderProcessor extends WorkflowEntrypoint<{}, Params> {
  async run(event: WorkflowEvent<Params>, step: WorkflowStep) {
    const validated = await step.do("validate", async () => {
      return { orderId: event.payload.orderId, valid: true };
    });

    await step.sleep("cooldown", "5 seconds");

    const charged = await step.do("charge", async () => {
      return { orderId: validated.orderId, chargedAt: Date.now() };
    });

    return charged;
  }
}
