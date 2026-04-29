import { getCollection, getEntry } from "astro:content";
import type { APIRoute } from "astro";
import { cleanMarkdown } from "../../util/markdown";

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

async function buildExamplesCatalog(): Promise<string> {
  const docs = await getCollection("docs");
  const examples = docs
    .filter((doc) => doc.id.startsWith("docs/examples/"))
    .sort((a, b) => a.id.localeCompare(b.id));

  const lines = examples.map((doc) => {
    const slug = doc.id.replace(/\.mdx?$/, "").replace(/^docs\//, "");
    return `- [${doc.data.title}](/docs/${slug}/)`;
  });

  return lines.join("\n");
}

export const GET: APIRoute = async ({ params }) => {
  const slug = params.slug!;
  const entry = await getEntry("docs", `docs/${slug}`);
  if (!entry?.body) return new Response("Not found", { status: 404 });

  let content: string;
  if (slug === "examples") {
    // Serve as a catalog of links to individual example pages
    const catalog = await buildExamplesCatalog();
    content = `# ${entry.data.title}

${entry.data.description || ""}

Source: https://sst.dev/docs/${slug}

---

${catalog}`;
  } else {
    const cleaned = cleanMarkdown(entry.body);
    content = `# ${entry.data.title}

${entry.data.description || ""}

Source: https://sst.dev/docs/${slug}

---

${cleaned}`;
  }

  return new Response(content, {
    headers: {
      "Content-Type": "text/markdown; charset=utf-8",
      "Cache-Control": "public,max-age=0,s-maxage=86400,stale-while-revalidate=86400",
    },
  });
};
