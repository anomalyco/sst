import { execSync } from "child_process";
import { VisibleError } from "../components/error.js";
import { Input } from "../components/input.js";

let dockerChecked = false;
let dockerAvailable = false;

export function isDockerAvailable(): boolean {
  if (dockerChecked) return dockerAvailable;
  try {
    execSync("docker version", { stdio: "pipe", timeout: 5000 });
    dockerAvailable = true;
  } catch {
    dockerAvailable = false;
  }
  dockerChecked = true;
  return dockerAvailable;
}

export function requireDocker(): void {
  if (!isDockerAvailable()) {
    throw new VisibleError("Docker is required but not running.");
  }
}

export function needsLocalDocker(args: {
  image?: Input<string | { context?: Input<string> }>;
  containers?: Input<{ image?: Input<string | { context?: Input<string> }> }>[];
}): boolean {
  const { image, containers } = args;
  const isLocalImage = (img: typeof image) =>
    img && typeof img === "object" && "context" in img;
  return !!(
    isLocalImage(image) ||
    (Array.isArray(containers) &&
      containers.some(
        (c) =>
          c &&
          typeof c === "object" &&
          "image" in c &&
          isLocalImage(c.image as typeof image),
      ))
  );
}
