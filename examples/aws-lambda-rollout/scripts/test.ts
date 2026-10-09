interface Result {
  status: number;
  latency: number;
  version: string;
  error: boolean;
}

export async function loadTest(url: string, count = 50) {
  console.log(`URL: ${url}\n`);
  console.log(`Sending ${count} concurrent requests...\n`);

  const results: Result[] = await Promise.all(
    Array.from({ length: count }, async () => {
      const start = Date.now();
      try {
        const res = await fetch(url);
        const body = (await res.json()) as { version?: string };
        return {
          status: res.status,
          latency: Date.now() - start,
          version: body.version ?? "unknown",
          error: false,
        };
      } catch {
        return {
          status: 0,
          latency: Date.now() - start,
          version: "unknown",
          error: true,
        };
      }
    }),
  );

  const byVersion = new Map<string, Result[]>();
  for (const r of results) {
    if (!byVersion.has(r.version)) byVersion.set(r.version, []);
    byVersion.get(r.version)!.push(r);
  }

  for (const [version, vResults] of byVersion) {
    const succeeded = vResults.filter(
      (r) => !r.error && r.status >= 200 && r.status < 300,
    );
    const failed = vResults.filter((r) => r.error || r.status >= 400);
    const avgLatency =
      vResults.reduce((sum, r) => sum + r.latency, 0) / vResults.length;

    console.log(`Version ${version}:`);
    console.log(`  Requests:    ${vResults.length}`);
    console.log(`  Succeeded:   ${succeeded.length}`);
    console.log(`  Failed:      ${failed.length}`);
    console.log(`  Avg latency: ${Math.round(avgLatency)}ms`);

    if (failed.length > 0) {
      const statusCounts: Record<number, number> = {};
      for (const r of failed) {
        statusCounts[r.status] = (statusCounts[r.status] || 0) + 1;
      }
      console.log(`  Status codes:`, statusCounts);
    }
    console.log();
  }
}
