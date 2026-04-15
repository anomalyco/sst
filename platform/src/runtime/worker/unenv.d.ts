export interface CloudflareUnenvInput {
  date?: string;
  flags?: string[];
}

export interface CloudflareUnenvConfig {
  alias: Record<string, string>;
  external: string[];
  polyfill: string[];
}

export function getCloudflareUnenvConfig(
  input?: CloudflareUnenvInput,
): CloudflareUnenvConfig;
