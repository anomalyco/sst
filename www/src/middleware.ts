import { defineMiddleware } from "astro:middleware";

export const onRequest = defineMiddleware((context, next) => {
  const accept = context.request.headers.get("accept") || "";
  const url = new URL(context.request.url);

  // Parse Accept header - format: "type/subtype;q=weight, ..."
  const acceptParts = accept.split(",").map(part => part.trim().split(";")[0]);

  // Find positions in Accept header (earlier = higher priority)
  const markdownIndex = acceptParts.findIndex(part =>
    part === "text/markdown" || part === "text/plain"
  );
  const htmlIndex = acceptParts.indexOf("text/html");

  // Prefer markdown if:
  // 1. Markdown/plain is present in Accept header
  // 2. HTML is not present, OR markdown comes before HTML
  const prefersMarkdown =
    markdownIndex !== -1 &&
    (htmlIndex === -1 || markdownIndex < htmlIndex);

  // Rewrite /docs.md and /blog.md to their index pages
  if (url.pathname === "/docs.md" || url.pathname === "/blog.md") {
    return context.rewrite(url.pathname.replace(".md", "/index.md"));
  }

  // Only rewrite docs and blog routes (but not .md files)
  if (
    prefersMarkdown &&
    (url.pathname.startsWith("/docs") || url.pathname.startsWith("/blog")) &&
    !url.pathname.endsWith(".md")
  ) {
    // Rewrite to markdown endpoint
    let markdownPath;

    if (url.pathname === "/docs" || url.pathname === "/blog") {
      markdownPath = `${url.pathname}/index.md`;
    } else if (url.pathname.endsWith("/")) {
      // Index pages: /docs/ → /docs/index.md
      markdownPath = `${url.pathname}index.md`;
    } else {
      // Regular pages: /docs/components → /docs/components.md
      markdownPath = `${url.pathname}.md`;
    }

    return context.rewrite(markdownPath);
  }

  return next();
});
