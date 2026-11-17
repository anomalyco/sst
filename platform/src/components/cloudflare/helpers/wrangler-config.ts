import type { Input } from "@pulumi/pulumi";
import type { Unstable_RawConfig as RawConfig } from "wrangler";
import { VisibleError } from "../../error.js";

export type DeepInput<T> = T extends (...args: never[]) => unknown
  ? T
  : T extends readonly (infer U)[]
    ? readonly DeepInput<U>[]
    : T extends object
      ? { [K in keyof T]?: DeepInput<T[K]> }
      : Input<T>;

export type WranglerConfigSource = {
  owner: string;
  config: Partial<RawConfig>;
  linkScoped?: boolean;
};

export const LINK_NAME_PLACEHOLDER = "__sst_link_name__";

const LINK_SCOPED_REJECTED_PATHS = new Set([
  "name",
  "main",
  "account_id",
  "compatibility_date",
  "compatibility_flags",
  "workers_dev",
  "route",
  "routes",
  "triggers",
  "migrations",
  "exports",
  "assets",
  "observability",
  "placement",
  "limits",
  "tail_consumers",
  "streaming_tail_consumers",
  "env",
]);

type OwnedNode = {
  value: unknown;
  owner: string;
  children?: Map<string, OwnedNode>;
  entries?: OwnedNode[];
};

type ObjectKind = "map" | "container" | "singleton";

export function validateLinkScopedConfig(
  owner: string,
  config: Partial<RawConfig>,
): void {
  validateConfig(owner, config, true);
}

function validateConfig(
  owner: string,
  config: Partial<RawConfig>,
  linkScoped: boolean,
): void {
  const value = config as unknown;
  if (!isPlainObject(value)) {
    throw new VisibleError(
      `Invalid Wrangler config for owner ${owner}: expected an object`,
    );
  }

  if (linkScoped) {
    for (const key of Object.keys(value)) {
      if (LINK_SCOPED_REJECTED_PATHS.has(key)) {
        throw new VisibleError(
          `Wrangler config for owner ${owner} contains forbidden path ${key}`,
        );
      }
    }

    const queues = value.queues;
    if (isPlainObject(queues) && Object.hasOwn(queues, "consumers")) {
      throw new VisibleError(
        `Wrangler config for owner ${owner} contains forbidden path queues.consumers`,
      );
    }
    validatePlaceholders(value, owner, "", true);
  }
}

export function replaceLinkName(
  owner: string,
  config: Partial<RawConfig>,
  linkName: string,
): Partial<RawConfig> {
  validateLinkScopedConfig(owner, config);
  const replaced = materialize(
    replacePlaceholders(config, owner, linkName, "", true),
  ) as Partial<RawConfig>;
  validateIdentityDuplicates(replaced, owner, "");
  return replaced;
}

export function composeWranglerConfig(
  sources: readonly WranglerConfigSource[],
): Partial<RawConfig> {
  const result = new Map<string, OwnedNode>();

  for (const source of sources) {
    validateConfig(source.owner, source.config, source.linkScoped !== false);
    for (const [key, value] of Object.entries(source.config)) {
      const incoming = createNode(value, source.owner);
      const existing = result.get(key);
      result.set(
        key,
        existing ? mergeNodes(key, existing, incoming) : incoming,
      );
    }
  }

  return Object.fromEntries(
    [...result].map(([key, value]) => [key, materialize(value)]),
  ) as Partial<RawConfig>;
}

function validatePlaceholders(
  value: unknown,
  owner: string,
  path: string,
  root = false,
  arrayEntry = false,
): void {
  if (typeof value === "string") {
    if (
      value.includes(LINK_NAME_PLACEHOLDER) &&
      (!arrayEntry || value !== LINK_NAME_PLACEHOLDER)
    ) {
      invalidPlaceholder(owner, path);
    }
    return;
  }

  if (Array.isArray(value)) {
    const identity = getArrayIdentity(value);
    const identities = new Set<string>();
    value.forEach((entry, index) => {
      validatePlaceholders(
        entry,
        owner,
        `${path}[${index}]`,
        false,
        identity !== undefined,
      );
      if (identity) {
        const identityValue = String(
          (entry as Record<string, unknown>)[identity],
        );
        if (identities.has(identityValue)) {
          conflict(
            path,
            owner,
            owner,
            `duplicate ${identity} identity ${identityValue}`,
          );
        }
        identities.add(identityValue);
      }
    });
    return;
  }

  if (!isPlainObject(value)) return;

  const kind = root ? undefined : objectKind(value, arrayEntry, path);
  for (const [key, entry] of Object.entries(value)) {
    const childPath = path ? `${path}.${key}` : key;
    if (key.includes(LINK_NAME_PLACEHOLDER)) {
      if (kind !== "map" || key !== LINK_NAME_PLACEHOLDER) {
        invalidPlaceholder(owner, childPath);
      }
    }

    validatePlaceholders(
      entry,
      owner,
      childPath,
      false,
      kind === "singleton" && (key === "binding" || key === "name"),
    );
  }
}

function replacePlaceholders(
  value: unknown,
  owner: string,
  linkName: string,
  path: string,
  root = false,
  arrayEntry = false,
): OwnedNode {
  if (typeof value === "string") {
    return {
      owner,
      value: value.includes(LINK_NAME_PLACEHOLDER)
        ? arrayEntry && value === LINK_NAME_PLACEHOLDER
          ? linkName
          : invalidPlaceholder(owner, path)
        : value,
    };
  }

  if (Array.isArray(value)) {
    const identity = getArrayIdentity(value);
    return {
      owner,
      value,
      entries: value.map((entry, index) =>
        replacePlaceholders(
          entry,
          owner,
          linkName,
          `${path}[${index}]`,
          false,
          identity !== undefined,
        ),
      ),
    };
  }

  if (!isPlainObject(value)) return { owner, value };

  const kind = root ? undefined : objectKind(value, arrayEntry, path);
  const children = new Map<string, OwnedNode>();
  for (const [key, entry] of Object.entries(value)) {
    const childPath = path ? `${path}.${key}` : key;
    const replacementKey =
      kind === "map" && key === LINK_NAME_PLACEHOLDER ? linkName : key;
    if (children.has(replacementKey)) {
      conflict(
        `${path ? `${path}.` : ""}${replacementKey}`,
        owner,
        owner,
        "map key replacement collides with an existing key",
      );
    }
    children.set(
      replacementKey,
      replacePlaceholders(
        entry,
        owner,
        linkName,
        childPath,
        false,
        kind === "singleton" && (key === "binding" || key === "name"),
      ),
    );
  }

  return { owner, value, children };
}

function createNode(value: unknown, owner: string): OwnedNode {
  if (Array.isArray(value)) {
    return {
      owner,
      value,
      entries: value.map((entry) => createNode(entry, owner)),
    };
  }
  if (isPlainObject(value)) {
    return {
      owner,
      value,
      children: new Map(
        Object.entries(value).map(([key, entry]) => [
          key,
          createNode(entry, owner),
        ]),
      ),
    };
  }
  return { owner, value };
}

function mergeNodes(
  path: string,
  existing: OwnedNode,
  incoming: OwnedNode,
): OwnedNode {
  if (Array.isArray(existing.value) && Array.isArray(incoming.value)) {
    return {
      owner: existing.owner,
      value: existing.value,
      entries: mergeArrays(path, existing, incoming),
    };
  }

  if (isPlainObject(existing.value) && isPlainObject(incoming.value)) {
    if (Object.keys(existing.value).length === 0) return incoming;
    if (Object.keys(incoming.value).length === 0) return existing;

    const existingKind = objectKind(existing.value, false, path);
    const incomingKind = objectKind(incoming.value, false, path);
    if (existingKind === "singleton" || incomingKind === "singleton") {
      conflict(
        path,
        existing.owner,
        incoming.owner,
        "singleton values conflict",
      );
    }
    if (existingKind !== incomingKind) {
      conflict(path, existing.owner, incoming.owner, "object kinds conflict");
    }
    return {
      owner: existing.owner,
      value: existing.value,
      children:
        existingKind === "map"
          ? mergeObjectMap(path, existing, incoming)
          : mergeObjectContainer(path, existing, incoming),
    };
  }

  conflict(path, existing.owner, incoming.owner, "values conflict");
}

function mergeObjectContainer(
  path: string,
  existing: OwnedNode,
  incoming: OwnedNode,
) {
  const children = new Map(existing.children!);
  for (const [key, incomingChild] of incoming.children!) {
    const existingChild = children.get(key);
    children.set(
      key,
      existingChild
        ? mergeNodes(`${path}.${key}`, existingChild, incomingChild)
        : incomingChild,
    );
  }
  return children;
}

function mergeObjectMap(
  path: string,
  existing: OwnedNode,
  incoming: OwnedNode,
) {
  const children = new Map(existing.children!);
  for (const [key, incomingChild] of incoming.children!) {
    const existingChild = children.get(key);
    if (existingChild) {
      conflict(
        `${path}.${key}`,
        existingChild.owner,
        incomingChild.owner,
        "duplicate map key",
      );
    }
    children.set(key, incomingChild);
  }
  return children;
}

function mergeArrays(path: string, existing: OwnedNode, incoming: OwnedNode) {
  const existingEntries = existing.entries!;
  const incomingEntries = incoming.entries!;
  if (existingEntries.length === 0 || incomingEntries.length === 0) {
    const owners = new Set([
      existing.owner,
      incoming.owner,
      ...existingEntries.map((entry) => entry.owner),
      ...incomingEntries.map((entry) => entry.owner),
    ]);
    if (owners.size > 1) {
      const [firstOwner, secondOwner] = owners;
      conflict(
        path,
        firstOwner!,
        secondOwner!,
        "array has no uniform identity",
      );
    }
    return [...existingEntries, ...incomingEntries];
  }

  const identity = getArrayIdentity([
    ...existingEntries.map((entry) => entry.value),
    ...incomingEntries.map((entry) => entry.value),
  ]);
  if (!identity) {
    const owners = new Set(
      [...existingEntries, ...incomingEntries].map((entry) => entry.owner),
    );
    if (owners.size === 1) {
      return [...existingEntries, ...incomingEntries];
    }
    const [firstOwner, secondOwner] = owners;
    conflict(path, firstOwner!, secondOwner!, "array has no uniform identity");
  }

  const identities = new Map<string, OwnedNode>();
  for (const entry of [...existingEntries, ...incomingEntries]) {
    const identityValue = String(
      (entry.value as Record<string, unknown>)[identity],
    );
    const previous = identities.get(identityValue);
    if (previous) {
      conflict(
        path,
        previous.owner,
        entry.owner,
        `duplicate ${identity} identity ${identityValue}`,
      );
    }
    identities.set(identityValue, entry);
  }

  return [...existingEntries, ...incomingEntries];
}

function materialize(node: OwnedNode): unknown {
  if (node.entries) return node.entries.map(materialize);
  if (node.children) {
    return Object.fromEntries(
      [...node.children].map(([key, child]) => [key, materialize(child)]),
    );
  }
  return node.value;
}

function validateIdentityDuplicates(
  value: unknown,
  owner: string,
  path: string,
): void {
  if (Array.isArray(value)) {
    const identity = getArrayIdentity(value);
    if (identity) {
      const identities = new Set<string>();
      for (const entry of value) {
        const identityValue = String(
          (entry as Record<string, unknown>)[identity],
        );
        if (identities.has(identityValue)) {
          conflict(
            path,
            owner,
            owner,
            `duplicate ${identity} identity ${identityValue}`,
          );
        }
        identities.add(identityValue);
      }
    }
    value.forEach((entry, index) =>
      validateIdentityDuplicates(entry, owner, `${path}[${index}]`),
    );
    return;
  }

  if (isPlainObject(value)) {
    for (const [key, entry] of Object.entries(value)) {
      validateIdentityDuplicates(entry, owner, path ? `${path}.${key}` : key);
    }
  }
}

function getArrayIdentity(
  value: readonly unknown[],
): "binding" | "name" | undefined {
  if (value.length === 0 || !value.every(isPlainObject)) return undefined;
  if (value.every((entry) => typeof entry.binding === "string")) {
    return "binding";
  }
  if (value.every((entry) => typeof entry.name === "string")) return "name";
  return undefined;
}

function objectKind(
  value: Record<string, unknown>,
  arrayEntry = false,
  path = "",
): ObjectKind {
  if (path === "vars") return "map";
  if (isSingletonObject(value, arrayEntry)) return "singleton";
  if (Object.values(value).some(Array.isArray)) return "container";
  return "map";
}

function isSingletonObject(value: Record<string, unknown>, arrayEntry = false) {
  return (
    typeof value.binding === "string" ||
    (typeof value.name === "string" &&
      (arrayEntry ||
        Object.hasOwn(value, "namespace_id") ||
        Object.hasOwn(value, "simple")))
  );
}

function isPlainObject(value: unknown): value is Record<string, any> {
  if (value === null || typeof value !== "object") return false;
  const prototype = Object.getPrototypeOf(value);
  return prototype === Object.prototype || prototype === null;
}

function invalidPlaceholder(owner: string, path: string): never {
  throw new VisibleError(
    `Wrangler config placeholder is only allowed in binding/name identities or complete object map keys for owner ${owner} at ${
      path || "<root>"
    }`,
  );
}

function conflict(
  path: string,
  firstOwner: string,
  secondOwner: string,
  reason: string,
): never {
  throw new VisibleError(
    `Wrangler config conflict at ${path} between owners ${firstOwner} and ${secondOwner}: ${reason}`,
  );
}
