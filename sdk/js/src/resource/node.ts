import crypto from "crypto";
import { readFileSync } from "fs";
import { env } from "process";

import {
  loadResourceData,
  loadResourceEnvironment,
  Resource,
} from "./shared.js";

const environment: Record<string, string | undefined> = {
  ...env,
  ...globalThis.process?.env,
};

loadResourceEnvironment(environment);

if (environment.SST_RESOURCES_JSON) {
  try {
    loadResourceData(JSON.parse(environment.SST_RESOURCES_JSON));
  } catch (error) {
    console.error("Failed to parse SST_RESOURCES_JSON:", error);
  }
}

if (environment.SST_KEY_FILE && environment.SST_KEY && !(globalThis as any).SST_KEY_FILE_DATA) {
  const key = Buffer.from(environment.SST_KEY, "base64");
  const encryptedData = readFileSync(environment.SST_KEY_FILE);
  const nonce = Buffer.alloc(12, 0);
  const decipher = crypto.createDecipheriv("aes-256-gcm", key, nonce);
  const authTag = encryptedData.subarray(-16);
  const actualCiphertext = encryptedData.subarray(0, -16);
  decipher.setAuthTag(authTag);
  let decrypted = decipher.update(actualCiphertext);
  decrypted = Buffer.concat([decrypted, decipher.final()]);
  loadResourceData(JSON.parse(decrypted.toString()));
}

if ((globalThis as any).SST_KEY_FILE_DATA) {
  loadResourceData((globalThis as any).SST_KEY_FILE_DATA);
}

export { Resource };
