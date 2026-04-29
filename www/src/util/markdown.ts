function stripOutsideCodeBlocks(
  source: string,
  pattern: RegExp,
  replacement: string
): string {
  const parts = source.split(/(```[\s\S]*?```)/g);
  return parts
    .map((part, i) =>
      // Odd indices are code blocks — leave them alone
      i % 2 === 1 ? part : part.replace(pattern, replacement)
    )
    .join("");
}

export function cleanMarkdown(source: string): string {
  let result = source;
  // Remove import/export statements outside code blocks
  result = stripOutsideCodeBlocks(result, /^import\s+.*$/gm, "");
  result = stripOutsideCodeBlocks(result, /^export\s+.*$/gm, "");
  return (
    result
      // Remove JSX comments {/* ... */} (single and multiline)
      .replace(/\{\/\*[\s\S]*?\*\/\}/g, "")
      // Remove <Image ... /> self-closing tags
      .replace(/<Image\s+[^>]*\/>/g, "")
      // Remove <VideoAside ... /> self-closing tags
      .replace(/<VideoAside\s+[^>]*\/>/g, "")
      // Remove <LinkCard ... /> self-closing tags
      .replace(/<LinkCard\s+[^>]*\/>/g, "")
      // Remove <Icon ... /> self-closing tags
      .replace(/<Icon\s+[^>]*\/>/g, "")
      // Remove tsdoc component tags (opening and closing)
      .replace(/<\/?(?:Section|Segment|InlineSection)(?:\s+[^>]*)?>/g, "")
      // Convert <NestedTitle ...>content</NestedTitle> to just content
      .replace(/<NestedTitle[^>]*>([\s\S]*?)<\/NestedTitle>/g, "$1")
      // Remove <div class="tsdoc"> and </div>
      .replace(/<div\s+class="tsdoc">/g, "")
      .replace(/^<\/div>\s*$/gm, "")
      // Merge adjacent <code> tags into one (for type expressions like Input<string>)
      .replace(/<\/code><code class="[^"]*">/g, "")
      // Convert innermost <code class="...">content</code> → `content`
      .replace(/<code class="[^"]*">([^<]*)<\/code>/g, "`$1`")
      // Strip remaining outer <code class="..."> and </code> from nested structures
      .replace(/<code class="[^"]*">/g, "")
      // Convert plain <code>content</code> → `content`
      .replace(/<code>([^<]*)<\/code>/g, "`$1`")
      .replace(/<\/code>/g, "")
      // Remove <p> and </p> wrapper tags
      .replace(/<\/?p>/g, "")
      // Decode common HTML entities
      .replace(/&lt;/g, "<")
      .replace(/&gt;/g, ">")
      .replace(/&amp;/g, "&")
      .replace(/&ldquo;/g, '"')
      .replace(/&rdquo;/g, '"')
      .replace(/&#123;/g, "{")
      .replace(/&#125;/g, "}")
      .replace(/&lcub;/g, "{")
      .replace(/&rcub;/g, "}")
      // Convert <Tabs>/<TabItem> to labeled sections
      .replace(/<Tabs>/g, "")
      .replace(/<\/Tabs>/g, "")
      .replace(/<TabItem\s+label="([^"]*)">/g, "**$1**\n")
      .replace(/<\/TabItem>/g, "")
      // Ensure blank line before and after code blocks, horizontal rules, and headings
      .replace(/([^\n])\n(```)/g, "$1\n\n$2")
      .replace(/(```)\n([^\n])/g, "$1\n\n$2")
      .replace(/([^\n])\n(---)\n/g, "$1\n\n$2\n")
      .replace(/\n(---)\n([^\n])/g, "\n$1\n\n$2")
      .replace(/([^\n])\n(#{1,6} )/g, "$1\n\n$2")
      .replace(/\n(#{1,6} .+)\n([^\n])/g, "\n$1\n\n$2")
      // Collapse 3+ consecutive blank lines to 2
      .replace(/\n{3,}/g, "\n\n")
      .trim()
  );
}
