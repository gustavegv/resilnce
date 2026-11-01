export const base = '';
export const resolve = (path: string, params?: Record<string, string>) =>
  path.replace(/\[([^\]]+)\]/g, (_, key: string) => params?.[key] ?? `[${key}]`);
