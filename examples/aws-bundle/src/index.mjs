import { readFileSync } from "fs";

const data = readFileSync(new URL("./file.json", import.meta.url));

export async function handler() {
  return {
    statusCode: 200,
    body: data.toString(),
  };
}
