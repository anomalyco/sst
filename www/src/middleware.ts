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

  // Only rewrite docs and blog routes (but not if already requesting .md directly)
  if (
    prefersMarkdown &&
    (url.pathname.startsWith("/docs/") || url.pathname.startsWith("/blog/")) &&
    !url.pathname.endsWith(".md")
  ) {
    // Rewrite to markdown endpoint with .md extension
    return context.rewrite(`${url.pathname}.md`);
  }

  return next();
});
