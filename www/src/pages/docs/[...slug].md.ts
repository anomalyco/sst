import { getEntry, getCollection } from "astro:content";
import type { APIRoute } from "astro";
import { cleanMarkdown } from "../../utils/markdown-clean";

export async function getStaticPaths() {
  const docs = await getCollection("docs");

  // Filter to only docs that start with "docs/"
  const filtered = docs.filter((doc) => {
    return doc.slug.startsWith("docs/") && !doc.slug.endsWith("/index");
  });

  return filtered.map((doc) => ({
    // Remove "docs/" prefix for the param
    params: { slug: doc.slug.replace(/^docs\//, '') },
  }));
}

export const GET: APIRoute = async ({ params }) => {
  // Add "docs/" prefix back to get the actual slug
  const slug = `docs/${params.slug}`;
  const entry = await getEntry("docs", slug);

  if (!entry) {
    return new Response("Not found", { status: 404 });
  }

  // Clean the markdown content (remove Astro components, imports, etc.)
  const cleanedBody = cleanMarkdown(entry.body);

  // Build clean markdown response
  const markdown = `# ${entry.data.title}

${entry.data.description || ''}

Source: https://sst.dev/${slug}

---

${cleanedBody}`;

  return new Response(markdown, {
    headers: {
      "Content-Type": "text/markdown; charset=utf-8",
      "Cache-Control": "public, max-age=3600"
    }
  });
};
