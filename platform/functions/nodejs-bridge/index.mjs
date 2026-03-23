import { promisify } from "node:util";
import { execFile as _execFile } from "node:child_process";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const execFile = promisify(_execFile);

export const handler = async () => {
  const binaryPath = join(__dirname, "bootstrap");

  try {
    await execFile(binaryPath);
  } catch (err) {
    console.error(err);
    throw err;
  }
};
