import { Resource } from "sst";
import { rollout } from "sst/aws/rollout";

export const handler = rollout.handler(async (event) => {
  const status = await validate();
  await rollout.report(event, status);
});

async function validate(): Promise<"Succeeded" | "Failed"> {
  try {
    const resp = await fetch(Resource.TestApi.url);
    const payload = await resp.text();

    if (resp.ok) {
      console.log("Health check passed:", payload);
      return "Succeeded";
    }
    console.log("Health check failed:", resp.status, payload);
    return "Failed";
  } catch (err) {
    console.error("Validation failed:", err);
    return "Failed";
  }
}
