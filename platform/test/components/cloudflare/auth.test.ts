import { readFileSync } from "node:fs";
import * as pulumi from "@pulumi/pulumi";
import { describe, expect, it, vi } from "vitest";

const workerCalls = vi.hoisted(
  () => [] as { options?: pulumi.ComponentResourceOptions }[],
);

vi.mock("../../../src/components/cloudflare/worker.js", async () => {
  const pulumi =
    await vi.importActual<typeof import("@pulumi/pulumi")>("@pulumi/pulumi");

  class Worker extends pulumi.ComponentResource {
    public readonly url = pulumi.output("https://auth.example.com");
    public readonly nodes: { worker: pulumi.ComponentResource };

    constructor(
      name: string,
      _args: unknown,
      options?: pulumi.ComponentResourceOptions,
    ) {
      super("sst:cloudflare:Worker", name, {}, options);
      workerCalls.push({ options });
      this.nodes = {
        worker: new pulumi.ComponentResource(
          "cloudflare:index/workersScript:WorkersScript",
          `${name}Script`,
          { scriptName: `${name}-script` },
          { parent: this },
        ),
      };
    }
  }

  return { Worker };
});

// @ts-ignore
global.$app = { name: "app", stage: "test" };
// @ts-ignore
global.$util = pulumi;
// @ts-ignore
global.$dev = false;
// @ts-ignore
global.$cli = {
  paths: { root: process.cwd(), work: process.cwd() },
  state: { version: {} },
};

pulumi.runtime.setMocks(
  {
    newResource: (args: pulumi.runtime.MockResourceArgs) => ({
      id: `${args.name}_id`,
      state: {
        ...args.inputs,
        privateKeyPemPkcs8: "private-key",
        publicKeyPem: "public-key",
        url: `https://${args.name}.example.com`,
        scriptName: `${args.name}-script`,
        etag: "etag",
      },
    }),
    call: (args: pulumi.runtime.MockCallArgs) => args.inputs,
  },
  "project",
  "stack",
  false,
);

function resolveOutput<T>(value: pulumi.Output<T>): Promise<T> {
  return new Promise((resolve) => {
    value.apply((resolved) => {
      resolve(resolved);
      return resolved;
    });
  });
}

describe("Cloudflare Auth ownership", () => {
  it("parents its internal Worker and keeps it out of top-level dev services", () => {
    const authSource = readFileSync(
      new URL("../../../src/components/cloudflare/auth.ts", import.meta.url),
      "utf8",
    );
    const workerSource = readFileSync(
      new URL("../../../src/components/cloudflare/worker.ts", import.meta.url),
      "utf8",
    );

    expect(authSource).toMatch(/new Worker\(\s*`\$\{name\}Authenticator`/);
    const authenticatorWorker = authSource.match(
      /new Worker\(([\s\S]*?)\n\s+\);/,
    )?.[1];
    expect(authenticatorWorker).toBeDefined();
    expect(authenticatorWorker).toContain("`${name}Authenticator`");
    expect(authenticatorWorker).toMatch(
      /parent:\s*this,[\s\S]*aliases:\s*\[\{\s*parent:\s*rootStackResource\s*\}\]/,
    );
    expect(workerSource).toMatch(
      /if \(!opts\?\.parent\) \{\s*this\.devConfig = \{\s*services:/,
    );
    expect(authSource).not.toContain("services:");
  });

  it("aliases the old top-level Worker URN and propagates it to descendants", async () => {
    const { Auth } = await import("../../../src/components/cloudflare/auth.js");
    const auth = new Auth("Auth", {
      authenticator: { handler: "package.json" },
    });
    const authenticator = await resolveOutput(auth.authenticator);
    const worker = authenticator as unknown as {
      __parentResource: unknown;
      __aliases: pulumi.Output<string>[];
      nodes: { worker: { __aliases: pulumi.Output<string>[] } };
    };

    expect(worker.__parentResource).toBe(auth);
    expect(workerCalls.at(-1)?.options).toEqual({
      parent: auth,
      aliases: [{ parent: pulumi.rootStackResource }],
    });
    await expect(
      Promise.all(worker.__aliases.map(resolveOutput)),
    ).resolves.toContain(
      "urn:pulumi:stack::project::sst:cloudflare:Worker::AuthAuthenticator",
    );

    const propagatedAliases = pulumi.allAliases(
      [],
      "AuthAuthenticatorScript",
      "cloudflare:index/workersScript:WorkersScript",
      worker as unknown as pulumi.Resource,
      "AuthAuthenticator",
    );
    const resolvedPropagatedAliases = await Promise.all(
      propagatedAliases.map(resolveOutput),
    );
    expect(resolvedPropagatedAliases).toEqual([
      expect.stringMatching(
        /urn:pulumi:stack::project::sst:cloudflare:Worker\$.*::AuthAuthenticatorScript$/,
      ),
    ]);
  });
});
