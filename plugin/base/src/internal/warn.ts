export namespace warn {
  const alreadyWarned = new Set<string>();

  export function once(message: string) {
    if (alreadyWarned.has(message)) return;
    alreadyWarned.add(message);
    console.warn(message);
  }
}
