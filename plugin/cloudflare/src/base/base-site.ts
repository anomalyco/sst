export interface BaseSiteFileOptions {
  files: string | string[];
  ignore?: string | string[];
  cacheControl?: string;
  contentType?: string;
}

export function getContentType(file: string, encoding: string): string {
  const ext = file.split('.').pop()?.toLowerCase();
  
  const mimeTypes: Record<string, string> = {
    'html': 'text/html',
    'css': 'text/css',
    'js': 'application/javascript',
    'json': 'application/json',
    'png': 'image/png',
    'jpg': 'image/jpeg',
    'jpeg': 'image/jpeg',
    'gif': 'image/gif',
    'svg': 'image/svg+xml',
    'ico': 'image/x-icon',
    'txt': 'text/plain',
    'pdf': 'application/pdf',
    'zip': 'application/zip',
  };

  const mimeType = mimeTypes[ext || ''] || 'application/octet-stream';
  
  if (encoding !== 'none' && (mimeType.startsWith('text/') || mimeType === 'application/javascript' || mimeType === 'application/json')) {
    return `${mimeType}; charset=${encoding}`;
  }
  
  return mimeType;
}

export function toPosix(filePath: string): string {
  return filePath.replace(/\\/g, '/');
}