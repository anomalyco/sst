import { getCollection } from "astro:content";
import type { APIRoute } from "astro";
import { cleanMarkdown } from "../../utils/markdown-clean";

export const GET: APIRoute = async () => {
  // Get all documentation
  const docs = await getCollection("docs");

  // Sort by slug for consistent ordering
  const sortedDocs = docs.sort((a, b) => a.slug.localeCompare(b.slug));

  // Build complete documentation in one file
  const content = `# SST Complete Documentation

> This file contains the complete SST documentation for AI agents and LLMs.
> Last generated: ${new Date().toISOString()}

## Table of Contents

${sortedDocs.map((doc, i) => `${i + 1}. [${doc.data.title}](#${doc.slug.replace(/\//g, '-')})`).join('\n')}

---

${sortedDocs
  .map((doc) => {
    const title = doc.data.title;
    const description = doc.data.description || '';
    const url = `https://sst.dev/${doc.slug}`;
    const cleanedBody = cleanMarkdown(doc.body);

    return `## ${title} {#${doc.slug.replace(/\//g, '-')}}

${description}

**URL:** ${url}

${cleanedBody}`;
  })
  .join('\n\n---\n\n')}

---

*End of SST Documentation*
`;

  return new Response(content, {
    headers: {
      "Content-Type": "text/plain; charset=utf-8",
      "Cache-Control": "public, max-age=3600"
    }
  });
};
