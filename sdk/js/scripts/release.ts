#!/usr/bin/env bun

import { $ } from "bun";
import fs from "fs/promises";
import path from "path";

import metafile from "../../../dist/metadata.json";
import artifacts from "../../../dist/artifacts.json";
import pkg from "../package.json";
const nextPkg = JSON.parse(JSON.stringify(pkg));
nextPkg.version = metafile.version;
nextPkg.optionalDependencies = nextPkg.optionalDependencies || {};
const snapshot = nextPkg.version.includes("0.0.0");
if (snapshot) {
  console.log("snapshot mode");
}

console.log("publishing", nextPkg.version);
console.log(`npm dist-tag add ${pkg.name}@${pkg.version} ion`);

await fs.rmdir("dist", { recursive: true }).catch(() => {});
await $`bun run build`;

const cpus = {
  arm64: "arm64",
  amd64: "x64",
  "386": "x86",
};

const tmp = `tmp`;
const binaryPackages = [] as string[];
for (const artifact of artifacts) {
  if (artifact.type !== "Binary") continue;
  const os = artifact.goos === "windows" ? "win32" : artifact.goos;
  const cpu = cpus[artifact.goarch as keyof typeof cpus];
  if (!os || !cpu)
    throw new Error(`Invalid artifact: ${JSON.stringify(artifact)}`);
  const name = `${pkg.name}-${os}-${cpu}`;
  const dir = path.join(tmp, name);
  const binary = path.basename(artifact.path);
  await fs.mkdir(path.join(dir, "bin"), { recursive: true });
  await fs.cp(
    path.join("../../", artifact.path),
    path.join(dir, "bin", binary),
  );
  Bun.write(
    path.join(dir, "package.json"),
    JSON.stringify(
      {
        name,
        version: nextPkg.version,
        os: [os],
        cpu: [cpu],
      },
      null,
      2,
    ),
  );
  nextPkg.optionalDependencies[name] = nextPkg.version;
  binaryPackages.push(dir);
}

const tag = snapshot ? "snapshot" : "latest";
try {
  for (const dir of binaryPackages) {
    await $`cd ${dir} && npm publish --access public --tag ${tag}`;
  }
  console.log(nextPkg);
  await Bun.write("package.json", JSON.stringify(nextPkg, null, 2));
  await $`npm publish --access public --tag ${tag}`;
  if (!snapshot)
    await $`npm dist-tag add ${nextPkg.name}@${nextPkg.version} ion`;
} finally {
  await Bun.write("package.json", JSON.stringify(pkg, null, 2));
  await fs.rmdir(tmp, { recursive: true });
}
