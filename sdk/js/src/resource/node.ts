import crypto from "crypto";
import { readFileSync } from "fs";
import { env } from "process";

import {
  createResource,
  loadResourceData,
  loadResourceEnvironment,
} from "./shared.js";
import type { Resource as ResourceShape } from "./shared.js";

const state = globalThis as typeof globalThis & {
  SST_KEY_FILE_DATA?: Record<string, any>;
};

let environmentLoaded = false;
let resourcesJson: string | undefined;
let keyFile: string | undefined;
let keyFileData: Record<string, any> | undefined;

function loadNodeResources() {
  const environment: Record<string, string | undefined> = {
    ...env,
    ...globalThis.process?.env,
  };

  if (!environmentLoaded) {
    environmentLoaded = true;
    loadResourceEnvironment(environment);
  }

  if (
    environment.SST_RESOURCES_JSON &&
    environment.SST_RESOURCES_JSON !== resourcesJson
  ) {
    resourcesJson = environment.SST_RESOURCES_JSON;
    try {
      loadResourceData(JSON.parse(environment.SST_RESOURCES_JSON));
    } catch (error) {
      console.error("Failed to parse SST_RESOURCES_JSON:", error);
    }
  }

  if (
    environment.SST_KEY_FILE &&
    environment.SST_KEY &&
    !state.SST_KEY_FILE_DATA &&
    environment.SST_KEY_FILE !== keyFile
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
    keyFile = environment.SST_KEY_FILE;
  }

  if (state.SST_KEY_FILE_DATA && state.SST_KEY_FILE_DATA !== keyFileData) {
    keyFileData = state.SST_KEY_FILE_DATA;
    loadResourceData(keyFileData);
  }
}

export interface Resource extends ResourceShape {}
export const Resource = createResource(loadNodeResources) as Resource;
