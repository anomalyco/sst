import { getCollection } from "astro:content";
import type { APIRoute } from "astro";
import { cleanMarkdown } from "../util/markdown";
import { isExcludedFromLlmsFull } from "../util/llms-filter";

export const GET: APIRoute = async () => {
  const docs = await getCollection("docs");
  const filtered = docs
    .filter((doc) => doc.id.startsWith("docs/"))
    .filter((doc) => !isExcludedFromLlmsFull(doc.id.replace(/\.mdx?$/, "")))
    .sort((a, b) => a.id.localeCompare(b.id));

  const pages = filtered.map((doc) => {
    const slug = doc.id.replace(/\.mdx?$/, "");
    const cleaned = cleanMarkdown(doc.body || "");
    return `## ${doc.data.title}

${doc.data.description || ""}

https://sst.dev/${slug}

${cleaned}`;
  });

  const body = `# SST Documentation

> The complete SST documentation for building full-stack applications on AWS and Cloudflare.

${pages.join("\n\n---\n\n")}
`;

  return new Response(body, {
    headers: {
      "Content-Type": "text/plain; charset=utf-8",
      "Cache-Control": "public,max-age=0,s-maxage=86400,stale-while-revalidate=86400",
    },
  });
};
