/**
 * Clean markdown content by removing Astro components, imports, and HTML tags
 * while preserving core markdown structure and code blocks.
 */
export function cleanMarkdown(source: string | undefined): string {
  // Handle undefined or null input
  if (!source) {
    return '';
  }

  let cleaned = source;

  // Remove frontmatter (--- ... ---)
  cleaned = cleaned.replace(/^---[\s\S]*?---\n*/m, '');

  // Remove import statements
  cleaned = cleaned.replace(/^import\s+.+?from\s+.+?;?\s*$/gm, '');

  // Remove Astro component tags but keep their content
  // Match <ComponentName ...> and </ComponentName>
  cleaned = cleaned.replace(/<[A-Z][a-zA-Z]*[^>]*>/g, '');
  cleaned = cleaned.replace(/<\/[A-Z][a-zA-Z]*>/g, '');

  // Remove self-closing Astro components
  cleaned = cleaned.replace(/<[A-Z][a-zA-Z]*[^>]*\/>/g, '');

  // Remove HTML comments
  cleaned = cleaned.replace(/<!--[\s\S]*?-->/g, '');

  // Remove inline HTML tags (but preserve markdown)
  cleaned = cleaned.replace(/<\/?[a-z][^>]*>/gi, '');

  // Clean up multiple blank lines (more than 2)
  cleaned = cleaned.replace(/\n{3,}/g, '\n\n');

  // Trim leading/trailing whitespace
  cleaned = cleaned.trim();

  return cleaned;
}

/**
 * Extract just the text content, removing all markdown syntax
 * Useful for summaries or plain text output
 */
export function stripMarkdown(markdown: string): string {
  let text = markdown;

  // Remove code blocks
  text = text.replace(/```[\s\S]*?```/g, '');
  text = text.replace(/`[^`]+`/g, '');

  // Remove headers
  text = text.replace(/^#{1,6}\s+/gm, '');

  // Remove bold/italic
  text = text.replace(/(\*\*|__)(.*?)\1/g, '$2');
  text = text.replace(/(\*|_)(.*?)\1/g, '$2');

  // Remove links but keep text
  text = text.replace(/\[([^\]]+)\]\([^)]+\)/g, '$1');

  // Remove images
  text = text.replace(/!\[([^\]]*)\]\([^)]+\)/g, '$1');

  // Remove list markers
  text = text.replace(/^[\*\-\+]\s+/gm, '');
  text = text.replace(/^\d+\.\s+/gm, '');

  // Remove blockquotes
  text = text.replace(/^>\s+/gm, '');

  // Clean up whitespace
  text = text.replace(/\n{2,}/g, '\n');
  text = text.trim();

  return text;
}
