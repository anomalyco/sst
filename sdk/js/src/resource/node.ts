import crypto from "crypto";
import { readFileSync } from "fs";
import { env } from "process";

import {
  createResource,
  loadResourceData,
  loadResourceEnvironment,
} from "./shared.js";
import type { Resource as BaseResource } from "./shared.js";

const state = globalThis as typeof globalThis & {
  SST_KEY_FILE_DATA?: Record<string, any>;
};

function loadNodeResources() {
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

  if (
    environment.SST_KEY_FILE &&
    environment.SST_KEY &&
    !state.SST_KEY_FILE_DATA
  ) {
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

  if (state.SST_KEY_FILE_DATA) {
    loadResourceData(state.SST_KEY_FILE_DATA);
  }
}

// Keep an interface here so generated sst-env.d.ts can augment Resource.
export interface Resource extends BaseResource {}
export const Resource = createResource<Resource>(loadNodeResources);
