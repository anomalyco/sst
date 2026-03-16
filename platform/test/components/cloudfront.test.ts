import { createRequire } from "node:module";
import { readFileSync } from "node:fs";
import vm from "node:vm";
import { describe, expect, it } from "vitest";

import { CF_ROUTER_INJECTION } from "../../src/components/aws/router";

const require = createRequire(import.meta.url);
const routerSource = readFileSync(
  new URL("../../src/components/aws/router.ts", import.meta.url),
  "utf8",
);

type CloudFrontField = {
  value?: string;
  multiValue?: { value: string }[];
};

type CloudFrontRequest = {
  uri: string;
  headers: Record<string, CloudFrontField>;
  cookies: Record<string, CloudFrontField>;
  querystring: Record<string, unknown>;
  origin?: unknown;
};

type CloudFrontEvent = {
  request: CloudFrontRequest;
};

type TestContext = vm.Context & {
  Promise: PromiseConstructor;
  Math: Math;
  JSON: JSON;
  RegExp: RegExpConstructor;
  decodeURIComponent: typeof decodeURIComponent;
  encodeURIComponent: typeof encodeURIComponent;
  event: CloudFrontEvent;
  cf: {
    kvs: () => {
      get: (key: string) => Promise<string>;
    };
    updateRequestOrigin: (origin: unknown) => void;
  };
  require: NodeRequire;
  __routeSite?: (
    kvNamespace: string,
    metadata: Record<string, any>,
  ) => Promise<any>;
  __getRequestHeaderSize?: () => number;
  __matchRoute?: (routes: string[]) => Promise<any>;
  __handler?: (event: CloudFrontEvent) => Promise<any>;
};

type CreateContextInput = {
  uri: string;
  headers: Record<string, CloudFrontField>;
  cookies?: Record<string, CloudFrontField>;
  querystring?: Record<string, unknown>;
  kvGet?: (key: string) => Promise<string>;
  updateRequestOrigin?: (origin: unknown, event: CloudFrontEvent) => void;
};

function extractTemplateCode(
  start: string,
  end: string,
  context: Record<string, unknown> = {},
) {
  const startIndex = routerSource.indexOf(start);
  const endIndex = routerSource.indexOf(end, startIndex);

  if (startIndex === -1 || endIndex === -1) {
    throw new Error(`Failed to extract code between ${start} and ${end}`);
  }

  return vm.runInNewContext(`\`${routerSource.slice(startIndex, endIndex)}\``, context);
}

const MATCH_ROUTE_CODE = extractTemplateCode(
  "async function matchRoute(routes) {",
  "\n\n  // Look up the route",
);

const REQUEST_HANDLER_CODE = extractTemplateCode(
  "async function handler(event) {\n  ${userInjection}\n  ${blockCloudfrontUrlInjection}\n  ${CF_ROUTER_INJECTION}\n",
  "\n}`,\n            },\n            { parent: self },\n          );",
  {
    userInjection: "",
    blockCloudfrontUrlInjection: "",
    CF_ROUTER_INJECTION,
    kvNamespace: "router",
  },
);

function createContext(input: CreateContextInput) {
  const event: CloudFrontEvent = {
    request: {
      uri: input.uri,
      headers: input.headers,
      cookies: input.cookies ?? {},
      querystring: input.querystring ?? {},
    },
  };

  const context: TestContext = {
    Promise,
    Math,
    JSON,
    RegExp,
    decodeURIComponent,
    encodeURIComponent,
    event,
    cf: {
      kvs: () => ({
        get: input.kvGet ?? (async () => {
          throw new Error("missing");
        }),
      }),
      updateRequestOrigin(origin: any) {
        input.updateRequestOrigin?.(origin, event);
      },
    },
    require,
  };

  vm.createContext(context);
  return { context, event };
}

function loadRouteSite(input: {
  uri: string;
  headers: Record<string, CloudFrontField>;
  cookies?: Record<string, CloudFrontField>;
  querystring?: Record<string, unknown>;
}) {
  const { context, event } = createContext(input);

  new vm.Script(
    `${CF_ROUTER_INJECTION.replace(
      /async function routeSite\(([^)]*)\)\{/,
      "async function routeSite($1){globalThis.__getRequestHeaderSize=getRequestHeaderSize;",
    )};globalThis.__routeSite = routeSite;`,
  ).runInContext(context);

  return {
    event,
    routeSite: context.__routeSite!,
    getRequestHeaderSize: () => context.__getRequestHeaderSize!(),
  };
}

function loadRouteMatcher(input: {
  uri: string;
  headers: Record<string, CloudFrontField>;
  metadata: Record<string, any>;
}) {
  const { context } = createContext({
    ...input,
    kvGet: async (key: string) => {
      if (!key.endsWith(":metadata")) throw new Error("missing");
      const routeNs = key.slice(0, -":metadata".length);
      const metadata = input.metadata[routeNs];
      if (!metadata) throw new Error("missing");
      return JSON.stringify(metadata);
    },
  });

  new vm.Script(`${MATCH_ROUTE_CODE};globalThis.__matchRoute = matchRoute;`).runInContext(
    context,
  );

  return context.__matchRoute!;
}

function loadHandler(input: {
  uri: string;
  headers: Record<string, CloudFrontField>;
  routes: string[];
  metadata: Record<string, any>;
  cookies?: Record<string, CloudFrontField>;
  querystring?: Record<string, unknown>;
}) {
  const { context, event } = createContext({
    ...input,
    kvGet: async (key: string) => {
      if (key === "router:routes") return JSON.stringify(input.routes);
      if (!key.endsWith(":metadata")) throw new Error("missing");

      const routeNs = key.slice(0, -":metadata".length);
      const metadata = input.metadata[routeNs];
      if (!metadata) throw new Error("missing");
      return JSON.stringify(metadata);
    },
    updateRequestOrigin(origin, currentEvent) {
      currentEvent.request.origin = origin;
    },
  });

  new vm.Script(`${REQUEST_HANDLER_CODE}
}
globalThis.__handler = handler;`).runInContext(
    context,
  );

  return {
    event,
    handler: context.__handler!,
  };
}

async function selectRoute(requestUri: string, routePaths: string[]) {
  const metadata = Object.fromEntries(
    routePaths.map((path, index) => [`route${index}`, { path }]),
  );

  const matchRoute = loadRouteMatcher({
    uri: requestUri,
    headers: {
      host: { value: "example.com" },
    },
    metadata,
  });

  const match = await matchRoute(
    routePaths.map((path, index) => `url,route${index},,${path}`),
  );

  return match?.metadata.path;
}

async function pathMatches(requestUri: string, routePath: string) {
  return (await selectRoute(requestUri, [routePath])) === routePath;
}

describe("CloudFront router", () => {
  describe("path matching", () => {
    it("matches exact paths", async () => {
      expect(await pathMatches("/api", "/api")).toBe(true);
      expect(await pathMatches("/travel-plan", "/travel-plan")).toBe(true);
      expect(await pathMatches("/", "/")).toBe(true);
    });

    it("matches paths followed by slashes", async () => {
      expect(await pathMatches("/api/", "/api")).toBe(true);
      expect(await pathMatches("/api/users", "/api")).toBe(true);
      expect(await pathMatches("/api/users/123", "/api")).toBe(true);
    });

    it("matches nested paths", async () => {
      expect(await pathMatches("/travel-plan/abc", "/travel-plan")).toBe(
        true,
      );
      expect(await pathMatches("/travel-plan/abc/def", "/travel-plan")).toBe(
        true,
      );
      expect(await pathMatches("/uploads/file.txt", "/uploads")).toBe(true);
    });

    it("treats root as catch-all", async () => {
      expect(await pathMatches("/", "/")).toBe(true);
      expect(await pathMatches("/anything", "/")).toBe(true);
      expect(await pathMatches("/foo/bar", "/")).toBe(true);
      expect(await pathMatches("/api", "/")).toBe(true);
    });

    it("does not match non-segment continuations", async () => {
      expect(await pathMatches("/api-docs", "/api")).toBe(false);
      expect(await pathMatches("/apiv2", "/api")).toBe(false);
      expect(await pathMatches("/travel-plans", "/travel-plan")).toBe(false);
      expect(await pathMatches("/travel-planning", "/travel-plan")).toBe(
        false,
      );
    });

    it("does not match different paths", async () => {
      expect(await pathMatches("/users", "/api")).toBe(false);
      expect(await pathMatches("/v2/api", "/api")).toBe(false);
      expect(await pathMatches("/files", "/uploads")).toBe(false);
    });

    it("does not match shorter paths", async () => {
      expect(await pathMatches("/ap", "/api")).toBe(false);
      expect(await pathMatches("/a", "/api")).toBe(false);
    });

    it("handles trailing slash routes", async () => {
      expect(await pathMatches("/api/", "/api")).toBe(true);
      expect(await pathMatches("/api/users", "/api/")).toBe(true);
      expect(await pathMatches("/public/2025-11/image.jpg", "/public/")).toBe(
        true,
      );
      expect(await pathMatches("/public/", "/public/")).toBe(true);
      expect(await pathMatches("/publicfile", "/public/")).toBe(false);
    });

    it("handles special characters", async () => {
      expect(await pathMatches("/api/users-list", "/api")).toBe(true);
      expect(await pathMatches("/api_v2", "/api")).toBe(false);
      expect(await pathMatches("/api.json", "/api")).toBe(false);
    });

    it("handles deeply nested paths", async () => {
      expect(await pathMatches("/a/b/c/d/e/f", "/a")).toBe(true);
      expect(await pathMatches("/a/b/c/d/e/f", "/a/b")).toBe(true);
      expect(await pathMatches("/a/b/c/d/e/f", "/a/b/c")).toBe(true);
    });

    it("handles real-world boundary cases", async () => {
      expect(await pathMatches("/travel-plan/abc123", "/travel-plan")).toBe(
        true,
      );
      expect(await pathMatches("/travel-plans", "/travel-plan")).toBe(false);
      expect(await pathMatches("/travel-plans/123", "/travel-plan")).toBe(
        false,
      );
      expect(await pathMatches("/v1/users", "/v1")).toBe(true);
      expect(await pathMatches("/v1-beta", "/v1")).toBe(false);
      expect(await pathMatches("/v1-beta/users", "/v1")).toBe(false);
      expect(await pathMatches("/uploads/file.pdf", "/uploads")).toBe(true);
      expect(await pathMatches("/uploads-backup", "/uploads")).toBe(false);
      expect(await pathMatches("/uploads-backup/file.pdf", "/uploads")).toBe(
        false,
      );
    });

    it("picks the longest matching path", async () => {
      expect(await selectRoute("/api/users", ["/api", "/api-docs"])).toBe(
        "/api",
      );
      expect(await selectRoute("/api-docs/intro", ["/api", "/api-docs"])).toBe(
        "/api-docs",
      );
      expect(
        await selectRoute("/api/v2/users", ["/api", "/api/v2", "/api/v2/users", "/api/v3"]),
      ).toBe("/api/v2/users");
    });
  });

  describe("header sizing", () => {
    it("counts request headers and cookies", async () => {
      const { getRequestHeaderSize, routeSite } = loadRouteSite({
        uri: "/",
        headers: {
          host: { value: "example.com" },
          accept: { value: "text/html" },
        },
        cookies: {
          session: { value: "abc" },
          theme: { value: "dark" },
        },
      });

      await routeSite("test", {});
      expect(getRequestHeaderSize()).toBe(71);
    });

    it("counts multi-value headers and cookies", async () => {
      const { getRequestHeaderSize, routeSite } = loadRouteSite({
        uri: "/",
        headers: {
          accept: {
            multiValue: [{ value: "text/html" }, { value: "application/json" }],
          },
        },
        cookies: {
          session: {
            multiValue: [{ value: "abc" }, { value: "def" }],
          },
        },
      });

      await routeSite("test", {});
      expect(getRequestHeaderSize()).toBe(79);
    });

    it("returns 431 for oversized image requests", async () => {
      const { routeSite } = loadRouteSite({
        uri: "/_next/image",
        headers: {
          host: { value: "example.com" },
          accept: { value: "image/webp" },
        },
        cookies: {
          session: { value: "x".repeat(10000) },
        },
      });

      const response = await routeSite("test", {
        image: { route: "/_next/image", host: "image.example.com" },
      });

      expect(response.statusCode).toBe(431);
      expect(response.statusDescription).toBe("Request Header Fields Too Large");
      expect(response.body.data).toContain("Reduce cookie size");
    });

    it("returns 431 for oversized server requests", async () => {
      const { routeSite } = loadRouteSite({
        uri: "/",
        headers: {
          host: { value: "example.com" },
        },
        cookies: {
          session: { value: "x".repeat(10000) },
        },
      });

      const response = await routeSite("test", {
        servers: [["server.example.com", 0, 0]],
        origin: {},
      });

      expect(response.statusCode).toBe(431);
      expect(response.statusDescription).toBe("Request Header Fields Too Large");
      expect(response.body.data).toContain("Reduce cookie size");
    });

    it("returns 431 for oversized image requests through handler", async () => {
      const { event, handler } = loadHandler({
        uri: "/_next/image",
        headers: {
          host: { value: "example.com" },
          accept: { value: "image/webp" },
        },
        cookies: {
          session: { value: "x".repeat(10000) },
        },
        routes: ["site,route0,,/"],
        metadata: {
          route0: {
            image: { route: "/_next/image", host: "image.example.com" },
          },
        },
      });

      const response = await handler(event);

      expect(response.statusCode).toBe(431);
      expect(response.statusDescription).toBe("Request Header Fields Too Large");
      expect(response.body.data).toContain("Reduce cookie size");
    });

    it("returns 431 for oversized server requests through handler", async () => {
      const { event, handler } = loadHandler({
        uri: "/",
        headers: {
          host: { value: "example.com" },
        },
        cookies: {
          session: { value: "x".repeat(10000) },
        },
        routes: ["site,route0,,/"],
        metadata: {
          route0: {
            servers: [["server.example.com", 0, 0]],
            origin: {},
          },
        },
      });

      const response = await handler(event);

      expect(response.statusCode).toBe(431);
      expect(response.statusDescription).toBe("Request Header Fields Too Large");
      expect(response.body.data).toContain("Reduce cookie size");
    });
  });
});
