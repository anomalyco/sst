export function cleanMarkdown(source: string): string {
  return (
    source
      // Remove import statements
      .replace(/^import\s+.*$/gm, "")
      // Remove export statements
      .replace(/^export\s+.*$/gm, "")
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
      // Convert <Tabs>/<TabItem> to labeled sections
      .replace(/<Tabs>/g, "")
      .replace(/<\/Tabs>/g, "")
      .replace(/<TabItem\s+label="([^"]*)">/g, "**$1**\n")
      .replace(/<\/TabItem>/g, "")
      // Collapse 3+ consecutive blank lines to 2
      .replace(/\n{3,}/g, "\n\n")
      .trim()
  );
}
