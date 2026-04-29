import { getCollection } from "astro:content";
import type { APIRoute } from "astro";
import { cleanMarkdown } from "../util/markdown";
import changelog from "../data/changelog.json";

function formatTag(tag: string): string {
  return tag.replace(/^v/, "");
}

function renderChangelog(): string {
  return (changelog as Array<{ tag: string; body: string }>)
    .map((r) => `## ${formatTag(r.tag)}\n\n${r.body}`)
    .join("\n\n");
}

export const GET: APIRoute = async () => {
  const docs = await getCollection("docs");
  const filtered = docs
    .filter((doc) => doc.id.startsWith("docs/"))
    .sort((a, b) => a.id.localeCompare(b.id));

  const pages = filtered.map((doc) => {
    const slug = doc.id.replace(/\.mdx?$/, "");
    let cleaned = cleanMarkdown(doc.body || "");
    if (slug === "docs/changelog") {
      cleaned = cleaned.replace(/<Changelog\s*\/>/g, renderChangelog());
    }
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
