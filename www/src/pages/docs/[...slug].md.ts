import { getCollection, getEntry } from "astro:content";
import type { APIRoute } from "astro";
import { cleanMarkdown } from "../../util/markdown";
import changelog from "../../data/changelog.json";

function formatTag(tag: string): string {
  return tag.replace(/^v/, "");
}

function renderChangelog(): string {
  return (changelog as Array<{ tag: string; body: string }>)
    .map((r) => `## ${formatTag(r.tag)}\n\n${r.body}`)
    .join("\n\n");
}

export async function getStaticPaths() {
  const docs = await getCollection("docs");
  return docs
    .filter(
      (doc) =>
        doc.id.startsWith("docs/") &&
        doc.id !== "docs/index.mdx"
    )
    .map((doc) => ({
      params: { slug: doc.id.replace(/^docs\//, "").replace(/\.mdx?$/, "") },
    }));
}

export const GET: APIRoute = async ({ params }) => {
  const slug = params.slug!;
  const entry = await getEntry("docs", `docs/${slug}`);
  if (!entry?.body) return new Response("Not found", { status: 404 });

  let cleaned = cleanMarkdown(entry.body);
  if (slug === "changelog") {
    cleaned = cleaned.replace(/<Changelog\s*\/>/g, renderChangelog());
  }
  const markdown = `# ${entry.data.title}

${entry.data.description || ""}

Source: https://sst.dev/docs/${slug}

---

${cleaned}`;

  return new Response(markdown, {
    headers: {
      "Content-Type": "text/markdown; charset=utf-8",
      "Cache-Control": "public,max-age=0,s-maxage=86400,stale-while-revalidate=86400",
    },
  });
};
