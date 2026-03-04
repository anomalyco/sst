import { getCollection } from "astro:content";
import type { APIRoute } from "astro";

export const GET: APIRoute = async () => {
  const docs = await getCollection("docs");
  const filtered = docs
    .filter((doc) => doc.id.startsWith("docs/"))
    .sort((a, b) => a.id.localeCompare(b.id));

  const links = filtered
    .map((doc) => {
      const slug = doc.id.replace(/\.mdx?$/, "");
      const description = doc.data.description || "";
      return `- [${doc.data.title}](https://sst.dev/${slug})${description ? `: ${description}` : ""}`;
    })
    .join("\n");

  const body = `# SST

> SST is a framework for building full-stack apps on your own infrastructure with support for AWS, Cloudflare, and 150+ providers.

## Docs

${links}
`;

  return new Response(body, {
    headers: {
      "Content-Type": "text/plain; charset=utf-8",
      "Cache-Control": "public,max-age=0,s-maxage=86400,stale-while-revalidate=86400",
    },
  });
};
