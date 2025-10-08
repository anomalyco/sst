import { getEntry, getCollection } from "astro:content";
import type { APIRoute } from "astro";
import { cleanMarkdown } from "../../utils/markdown-clean";

export async function getStaticPaths() {
  const docs = await getCollection("docs");

  // Filter docs that start with "blog/" or "docs/"
  const filtered = docs.filter((doc) => {
    return (
      (doc.slug.startsWith("blog/") || doc.slug.startsWith("docs/")) &&
      !doc.slug.endsWith("/index")
    );
  });

  return filtered.map((doc) => {
    const [section, ...slugParts] = doc.slug.split("/");
    const slug = slugParts.join("/");

    return {
      params: { section, slug },
    };
  });
}

export const GET: APIRoute = async ({ params }) => {
  const slug = `${params.section}/${params.slug}`;
  const entry = await getEntry("docs", slug);

  if (!entry) {
    return new Response("Not found", { status: 404 });
  }

  const cleanedBody = cleanMarkdown(entry.body);

  const markdown = `# ${entry.data.title}

${entry.data.description || ""}

Source: https://sst.dev/${slug}

---

${cleanedBody}`;

  return new Response(markdown, {
    headers: {
      "Content-Type": "text/markdown; charset=utf-8",
      "Cache-Control": "public, max-age=3600",
    },
  });
};
