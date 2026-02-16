import { getEntry } from "astro:content";
import type { APIRoute } from "astro";
import { cleanMarkdown } from "../../utils/markdown-clean";

export const GET: APIRoute = async () => {
  const entry = await getEntry("docs", "docs");
  if (!entry) return new Response("Not found", { status: 404 });

  const cleaned = cleanMarkdown(entry.body!);
  const markdown = `# ${entry.data.title}\n\n${entry.data.description || ""}\n\nSource: https://sst.dev/docs\n\n---\n\n${cleaned}`;

  return new Response(markdown, {
    headers: {
      "Content-Type": "text/markdown; charset=utf-8",
      "Cache-Control": "public, max-age=3600",
    },
  });
};
