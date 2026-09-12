/** 管理后台支持的接入协议。模型厂商（model_vendor）是另一条正交字段。 */
export const MONITOR_SERVICE_TYPES = ['cc', 'cx', 'gm'] as const;

export function isKnownMonitorService(service?: string): boolean {
  const normalized = service?.trim().toLowerCase();
  return MONITOR_SERVICE_TYPES.includes(normalized as typeof MONITOR_SERVICE_TYPES[number]);
}

/**
 * 模板文件名的第一段就是接入协议（cc-/cx-/gm-）。未知旧 service 不过滤，
 * 这样管理员可以打开历史配置并把 xai 这类误填值修复为正确的 cx。
 */
export function mergeTemplateNames(
  names: string[],
  current: string | undefined,
  service?: string,
): string[] {
  const normalized = service?.trim().toLowerCase();
  const filtered = isKnownMonitorService(normalized)
    ? names.filter(name => name.startsWith(`${normalized}-`))
    : names;
  const merged = new Set(filtered.filter(Boolean));
  if (current) merged.add(current);
  return Array.from(merged).sort();
}
