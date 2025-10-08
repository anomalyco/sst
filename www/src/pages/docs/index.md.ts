import { getEntry } from "astro:content";
import type { APIRoute } from "astro";
import { cleanMarkdown } from "../../utils/markdown-clean";

export const GET: APIRoute = async () => {
  const entry = await getEntry("docs", "docs");

  if (!entry) {
    return new Response("Not found", { status: 404 });
  }

  const cleanedBody = cleanMarkdown(entry.body);

  const markdown = `# ${entry.data.title}

${entry.data.description || ""}

Source: https://sst.dev/docs

---

${cleanedBody}`;

  return new Response(markdown, {
    headers: {
      "Content-Type": "text/markdown; charset=utf-8",
      "Cache-Control": "public, max-age=3600",
    },
  });
};
