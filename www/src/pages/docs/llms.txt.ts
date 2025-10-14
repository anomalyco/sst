import { getCollection } from "astro:content";
import type { APIRoute } from "astro";
import { cleanMarkdown } from "../../utils/markdown-clean";

export const GET: APIRoute = async () => {
  const docs = await getCollection("docs");

  const filteredDocs = docs.filter((doc) => !doc.slug.startsWith("blog/"));
  
  const sortedDocs = filteredDocs.sort((a, b) => a.slug.localeCompare(b.slug));

  const content = `# SST Documentation

> The SST documentation for building full-stack applications on AWS and Cloudflare. Learn about components, deployment, and infrastructure as code.

${sortedDocs
  .map((doc) => {
    const url = `https://sst.dev/${doc.slug}`;
    const title = doc.data.title;
    const description = doc.data.description || '';

    const cleanedBody = cleanMarkdown(doc.body);

    return `## ${title}

${description}

${url}

${cleanedBody}`;
  })
  .join('\n\n---\n\n')}
`;

  return new Response(content, {
    headers: {
      "Content-Type": "text/plain; charset=utf-8",
      "Cache-Control": "public, max-age=3600"
    }
  });
};
