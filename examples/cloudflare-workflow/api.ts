import { Resource } from "sst";

export default {
  async fetch() {
    const instance = await Resource.SignupWorkflow.create();

    return Response.json({
      id: instance.id,
      message: "Workflow started. Check the logs to see it run.",
    });
  },
};
