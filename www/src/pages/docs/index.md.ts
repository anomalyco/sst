import { getEntry } from "astro:content";
import type { APIRoute } from "astro";
import { cleanMarkdown } from "../../util/markdown";

export const GET: APIRoute = async () => {
  const entry = await getEntry("docs", "docs");
  if (!entry?.body) return new Response("Not found", { status: 404 });

  const cleaned = cleanMarkdown(entry.body);
  const markdown = `# ${entry.data.title}

${entry.data.description || ""}

Source: https://sst.dev/docs

---

${cleaned}`;

  return new Response(markdown, {
    headers: {
      "Content-Type": "text/markdown; charset=utf-8",
      "Cache-Control": "public,max-age=0,s-maxage=86400,stale-while-revalidate=86400",
    },
  });
};
