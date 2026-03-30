/**
 * Build a listener key from protocol and port (e.g. "HTTPS443").
 */
export function listenerKey(protocol: string, port: number): string {
  return `${protocol.toUpperCase()}${port}`;
}

/**
 * Build a target group key from container name, protocol and port (e.g. "appHTTP3000").
 */
export function targetKey(
  container: string,
  protocol: string,
  port: number,
): string {
  return `${container}${protocol.toUpperCase()}${port}`;
}
