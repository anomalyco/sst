# Cloudflare Worker Observability

Date: 2026-06-21

## Purpose

Expose Cloudflare Workers observability (logs and traces) as a first-class option across non-deprecated Worker-backed SST components, with attention to the SSR site family (Astro, ReactRouter, TanStackStart, SolidStart) and `StaticSiteV2`.

## Background

Cloudflare Workers supports observability at two levels:

- **Traces**: capture platform operations (fetch, KV, R2, Durable Object, Queue). Enabled via `observability.enabled = true` and `observability.traces.enabled = true` with optional `head_sampling_rate`. Custom spans can be added in Worker code via `ctx.tracing`.
- **Logs**: capture `console.*` messages and invocation metadata. Controlled via `observability.logs.enabled` + `invocationLogs`.

The `@pulumi/cloudflare` v6.15.0 `WorkersScript` resource exposes these through `WorkersScriptArgs.observability`:

```typescript
WorkersScriptObservability {
  enabled: boolean;
  headSamplingRate?: number;
  logs?: {
    enabled: boolean;
    headSamplingRate?: number;
    invocationLogs: boolean;
    destinations?: string[];
    persist?: boolean;
  };
  traces?: {
    enabled?: boolean;
    headSamplingRate?: number;
    destinations?: string[];
    persist?: boolean;
  };
}
```

SST's current `Worker` component does not expose any of this as a first-class arg. Users can only reach it via `transform.worker`.

## Worker Creation Chain

```
AstroArgs -> SsrSiteArgs -> SsrSite.createWorker() -> new Worker(args) -> cf.WorkersScript
```

Three tiers of components create or accept Workers:

1. **Direct `new Worker(args)`** -- `Worker` itself
2. **Internal Worker creation with selected pass-through args** -- `SsrSite`, `StaticSiteV2`, `Workflow`
3. **`workerBuilder(handler | WorkerArgs)`** -- `Cron`, `Queue.subscribe`
4. **WorkerArgs accepted directly** -- `Auth.authenticator`

The SSR subclasses (`Astro`, `ReactRouter`, `TanStackStart`, experimental `SolidStart`) each redeclare their args individually via `extends SsrSiteArgs`, picking specific properties with JSDoc. `SsrSite.createWorker()` builds the Worker with hardcoded `url: true`, `dev: false`, and resolved `handler`/`assets` paths.

`StaticSiteV2` is in scope for the first pass because it creates a Worker internally and exposes only selected Worker properties today. Adding `observability` only to `WorkerArgs` would not help `new StaticSiteV2(...)` users unless `StaticSiteV2Args` also gets a pass-through prop.

Deprecated `StaticSite` is out of scope for the first pass. `DurableObject` is also out of scope because it does not create a Worker; users attach Durable Objects to a `Worker` through `link` and configure observability on that Worker.

## Proposed API Shape

### On `WorkerArgs`

```typescript
observability?: Input<{
  logs?: Input<{
    enabled?: Input<boolean>;
    headSamplingRate?: Input<number>;
    invocationLogs?: Input<boolean>;
  }>;
  traces?: Input<{
    enabled?: Input<boolean>;
    headSamplingRate?: Input<number>;
  }>;
}>;
```

`destinations` and `persist` intentionally omitted from the initial shape. They can be added later without breaking changes.

### Component Pass-through Props

Components that create internal Workers but do not accept `WorkerArgs` directly should expose:

```typescript
observability?: WorkerArgs["observability"];
```

This applies to `SsrSiteArgs`, `StaticSiteV2Args`, and `WorkflowArgs`.

`Cron`, `Queue.subscribe`, and `Auth` do not need component-level args for the first pass because their Worker definitions already flow through `WorkerArgs`.

### Normalization

```typescript
function normalizeObservability() {
  if (!args.observability) return undefined;
  return output(args.observability).apply((v) => ({
    enabled: true,
    logs: v.logs
      ? { enabled: true, invocationLogs: true, ...v.logs }
      : undefined,
    traces: v.traces
      ? { enabled: true, ...v.traces }
      : undefined,
  }));
}
```

### Usage

```typescript
// Just traces
new Worker("MyWorker", {
  handler: "src/index.ts",
  observability: { traces: { enabled: true, headSamplingRate: 0.1 } },
});

// Logs + traces
new Worker("MyWorker", {
  handler: "src/index.ts",
  observability: {
    logs: { enabled: true, headSamplingRate: 0.5 },
    traces: { enabled: true, headSamplingRate: 0.1 },
  },
});

// Via SSR component (Astro, ReactRouter, TanStackStart, SolidStart)
new Astro("MySite", {
  observability: { traces: { enabled: true } },
});

// Via StaticSiteV2
new StaticSiteV2("MySite", {
  observability: { logs: { enabled: true } },
});

// Via Workflow
new Workflow("MyWorkflow", {
  handler: "src/workflow.ts",
  className: "OrderProcessor",
  observability: { traces: { enabled: true } },
});

// Via workerBuilder components (Cron, Queue.subscribe)
new Cron("MyJob", {
  worker: {
    handler: "src/job.handler",
    observability: { traces: { enabled: true } },
  },
});

queue.subscribe({
  handler: "src/consumer.ts",
  observability: { logs: { enabled: true } },
});
```

### Threading

1. Add `observability` to `WorkerArgs` in `worker.ts`
2. Add `normalizeObservability()` with top-level `enabled: true` and thread into the `cf.WorkersScript` call
3. Add to `SsrSiteArgs` in `ssr-site.ts` and thread through `createWorker()`
4. Subclasses (`AstroArgs`, `ReactRouterArgs`, `TanStackStartArgs`, `SolidStartArgs`) inherit via `extends SsrSiteArgs`
5. Add to `StaticSiteV2Args` and thread through its internal router Worker
6. Add to `WorkflowArgs` and thread through its internal Worker
7. `workerBuilder` components pass `WorkerArgs` through -- no changes needed
8. `Auth` accepts `authenticator: WorkerArgs` -- no changes needed

## Open Questions

1. **Top-level shorthand?** Should `observability: true` enable all observability with defaults?
2. **Destinations and persist?** Defer or include now?
3. **Go dynamic provider?** Has stale `observability` struct with `HeapSamplingRate`. It's deprecated -- leave it or clean up?
