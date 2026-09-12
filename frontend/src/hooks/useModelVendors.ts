import { useEffect, useState } from 'react';
import { apiGet } from '../utils/apiClient';
import type { ModelVendorInfo } from '../types/onboarding';

/**
 * 拉取模型厂商受控词表（后端 internal/modelvendor 是唯一真相源，前端不自建一份）。
 *
 * 走独立的 /api/model-vendors：它不依赖 onboarding 开关，管理员始终能编辑厂商。
 * 模块级缓存一份，避免后台里每开一个详情页都重拉。
 *
 * 拉不到时返回空数组——调用方应据此降级成「只保留当前值」的下拉，而不是把字段变成不可编辑：
 * 词表拉不到是前端的麻烦，不该让管理员改不了这条申请。
 */
let cachedVendors: ModelVendorInfo[] | null = null;
let inflight: Promise<ModelVendorInfo[]> | null = null;

function fetchVendors(): Promise<ModelVendorInfo[]> {
  if (cachedVendors) return Promise.resolve(cachedVendors);
  if (!inflight) {
    inflight = apiGet<{ model_vendors?: ModelVendorInfo[] }>('/api/model-vendors')
      .then((resp) => {
        cachedVendors = resp.model_vendors ?? [];
        return cachedVendors;
      })
      .catch(() => [])
      .finally(() => { inflight = null; });
  }
  return inflight;
}

export function useModelVendors(): ModelVendorInfo[] {
  const [vendors, setVendors] = useState<ModelVendorInfo[]>(cachedVendors ?? []);

  useEffect(() => {
    let alive = true;
    fetchVendors().then((list) => { if (alive) setVendors(list); });
    return () => { alive = false; };
  }, []);

  return vendors;
}

/**
 * 组装厂商下拉选项：空值（未声明）恒在首位，当前值即使不在词表里也保留为选项——
 * 否则打开一条带历史/新厂商 code 的记录，一保存就把该字段悄悄清空了。
 */
export function buildVendorOptions(
  vendors: ModelVendorInfo[],
  current: string,
  emptyLabel: string,
): { value: string; label: string }[] {
  const options = [{ value: '', label: emptyLabel }];
  for (const vendor of vendors) {
    options.push({ value: vendor.code, label: `${vendor.label}（${vendor.code}）` });
  }
  if (current && !options.some((o) => o.value === current)) {
    options.push({ value: current, label: current });
  }
  return options;
}
