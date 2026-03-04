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

type ContainerImage = Input<string | { context?: Input<string> }>;

function isLocalImage(image: ContainerImage | undefined) {
  return !!(image && typeof image === "object" && "context" in image);
}

export function needsLocalDocker(args: {
  image?: ContainerImage;
  containers?: Input<{ image?: ContainerImage }>[];
}) {
  const { image, containers } = args;
  return (
    isLocalImage(image) ||
    (Array.isArray(containers) &&
      containers.some(
        (c) =>
          c &&
          typeof c === "object" &&
          "image" in c &&
          isLocalImage(c.image),
      ))
  );
}
