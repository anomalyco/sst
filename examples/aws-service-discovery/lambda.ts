import { Resource } from "sst";

export async function handler() {
  const response = await fetch(
    `http://${Resource.MyService.service}`
  );

  return {
    statusCode: 200,
    body: await response.text(),
  };
}
