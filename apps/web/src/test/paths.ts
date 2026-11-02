export const base = '';
export const asset = (path: string) => path;
export const resolve = (path: string, params?: Record<string, string>) =>
  path.replace(/\[([^\]]+)\]/g, (_, key: string) => params?.[key] ?? `[${key}]`);
