/**
 * Simple path-to-regex utility for converting path patterns to regular expressions
 */
export function pathToRegexp(paths: string | string[]): RegExp {
  if (Array.isArray(paths)) {
    if (paths.length === 0) {
      return /(?:)/; // Match nothing
    }
    
    // Join multiple paths with OR
    const pattern = paths.map(path => pathToRegexpSingle(path)).join('|');
    return new RegExp(`^(?:${pattern})$`);
  }
  
  return new RegExp(`^${pathToRegexpSingle(paths)}$`);
}

function pathToRegexpSingle(path: string): string {
  // Escape special regex characters except * and ?
  let pattern = path
    .replace(/[.+^${}()|[\]\\]/g, '\\$&')
    .replace(/\*/g, '.*')  // * becomes .*
    .replace(/\?/g, '.');  // ? becomes .
  
  return pattern;
}