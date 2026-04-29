import * as fs from "fs";
import * as path from "path";

const REPO = "sst/sst";
const MIN_MAJOR = 4;
const PER_PAGE = 100;
const OUTPUT_PATH = path.join(
  import.meta.dir ?? __dirname,
  "src/data/changelog.json",
);

type GithubRelease = {
  tag_name: string;
  published_at: string;
  created_at: string;
  html_url: string;
  body: string | null;
  draft: boolean;
  prerelease: boolean;
};

type ChangelogEntry = {
  tag: string;
  publishedAt: string;
  url: string;
  body: string;
};

async function fetchReleases(): Promise<GithubRelease[]> {
  const headers: Record<string, string> = {
    Accept: "application/vnd.github+json",
    "X-GitHub-Api-Version": "2022-11-28",
    "User-Agent": "sst-docs-changelog-generator",
  };
  const token = process.env.GITHUB_TOKEN;
  if (token) headers.Authorization = `Bearer ${token}`;

  const all: GithubRelease[] = [];
  for (let page = 1; ; page++) {
    const url = `https://api.github.com/repos/${REPO}/releases?per_page=${PER_PAGE}&page=${page}`;
    const res = await fetch(url, { headers });
    if (!res.ok) {
      const text = await res.text().catch(() => "");
      throw new Error(
        `GitHub API ${res.status} ${res.statusText} on ${url}: ${text}`,
      );
    }
    const batch = (await res.json()) as GithubRelease[];
    all.push(...batch);
    if (batch.length < PER_PAGE) break;

    // Stop early once we've gone past the v4 cutoff to save calls.
    const passedCutoff = batch.every((r) => {
      const major = parseMajor(r.tag_name);
      return major !== null && major < MIN_MAJOR;
    });
    if (passedCutoff) break;
  }
  return all;
}

function parseMajor(tag: string): number | null {
  const m = /^v(\d+)\./.exec(tag);
  return m ? Number(m[1]) : null;
}

function cleanBody(body: string | null): string {
  if (!body) return "";
  let text = body.replace(/\r\n/g, "\n");

  // Strip the trailing "## Changelog" section heading and any blank lines
  // immediately preceding it, but keep the bullets that follow.
  text = text.replace(/\n*^##\s+Changelog\s*$/im, "");

  const lines = text.split("\n").map((line) => {
    // Match a leading bullet, an optional [hash](url) or bare hash,
    // an optional ":" or whitespace separator, then the subject.
    // Handles all observed formats:
    //   * abc1234 subject
    //   * abc1234: subject
    //   * [abc1234](https://...): subject
    //   * [abc1234](https://...) subject
    const m = line.match(
      /^(\s*[-*]\s+)(?:\[[0-9a-f]{7,40}\](?:\([^)]+\))?|[0-9a-f]{7,40})[:\s]\s*(.*)$/i,
    );
    if (m) return `${m[1]}${m[2]}`;
    return line;
  });

  let result = lines.join("\n");

  // Strip trailing "(@username)" author attributions from each line.
  result = result.replace(/[ \t]*\(@[\w-]+\)[ \t]*$/gm, "");

  result = result.trim();

  // Linkify GitHub PR/issue references (#1234) so they remain clickable.
  // Skip refs that are already inside a markdown link `[#1234]` or part of
  // a URL path (`/issues/1234#anchor` etc).
  result = result.replace(
    /(^|[^\[\/\w])#(\d{2,})(?!\])/g,
    (_, prefix, num) =>
      `${prefix}[#${num}](https://github.com/${REPO}/issues/${num})`,
  );

  // Unwrap parentheses around inline PR/issue links: "([#1234](url))" → "[#1234](url)".
  result = result.replace(/\(\[#(\d+)\]\(([^)]+)\)\)/g, "[#$1]($2)");

  return result;
}

async function main() {
  console.log(`Fetching releases from ${REPO}...`);
  const releases = await fetchReleases();
  console.log(`  fetched ${releases.length} total releases`);

  const filtered = releases
    .filter((r) => !r.draft)
    .filter((r) => {
      const major = parseMajor(r.tag_name);
      return major !== null && major >= MIN_MAJOR;
    })
    .sort(
      (a, b) =>
        new Date(b.published_at).getTime() -
        new Date(a.published_at).getTime(),
    );

  console.log(`  ${filtered.length} releases match v${MIN_MAJOR}+`);

  const entries: ChangelogEntry[] = filtered.map((r) => ({
    tag: r.tag_name,
    publishedAt: r.published_at,
    url: r.html_url,
    body: cleanBody(r.body),
  }));

  fs.mkdirSync(path.dirname(OUTPUT_PATH), { recursive: true });
  fs.writeFileSync(OUTPUT_PATH, JSON.stringify(entries, null, 2) + "\n");
  console.log(`Wrote ${entries.length} entries to ${OUTPUT_PATH}`);
}

main().catch((err) => {
  console.error("Failed to generate changelog:", err);
  process.exit(1);
});
