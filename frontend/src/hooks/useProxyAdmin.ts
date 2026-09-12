import { useCallback, useEffect, useState } from 'react';
import { apiDelete, apiGet, apiPost, apiPut, ApiError } from '../utils/apiClient';
import type { ProxyProfile } from '../types/proxy';

interface ProxyListResponse {
  proxies: ProxyProfile[];
}

/** 管理后台代理配置的列表与 CRUD。地址只在创建/替换时进入请求体。 */
export function useProxyAdmin(isAuthenticated: boolean) {
  const [proxies, setProxies] = useState<ProxyProfile[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchProxies = useCallback(async () => {
    if (!isAuthenticated) return;
    setIsLoading(true);
    setError(null);
    try {
      const resp = await apiGet<ProxyListResponse>('/api/admin/proxies');
      setProxies(resp.proxies || []);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '加载代理配置失败');
    } finally {
      setIsLoading(false);
    }
  }, [isAuthenticated]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 鉴权后挂载即取代理列表，加载态由 fetchProxies 管理
    fetchProxies();
  }, [fetchProxies]);

  const createProxy = useCallback(async (name: string, url: string) => {
    setError(null);
    try {
      await apiPost<{ proxy: ProxyProfile }>('/api/admin/proxies', { name, url });
      await fetchProxies();
    } catch (e) {
      const err = e instanceof ApiError ? e.message : '创建代理配置失败';
      setError(err);
      throw e;
    }
  }, [fetchProxies]);

  const updateProxy = useCallback(async (id: string, name: string, url: string) => {
    setError(null);
    try {
      await apiPut<{ proxy: ProxyProfile }>(`/api/admin/proxies/${encodeURIComponent(id)}`, { name, url });
      await fetchProxies();
    } catch (e) {
      const err = e instanceof ApiError ? e.message : '更新代理配置失败';
      setError(err);
      throw e;
    }
  }, [fetchProxies]);

  const deleteProxy = useCallback(async (id: string) => {
    setError(null);
    try {
      await apiDelete(`/api/admin/proxies/${encodeURIComponent(id)}`);
      await fetchProxies();
    } catch (e) {
      const err = e instanceof ApiError ? e.message : '删除代理配置失败';
      setError(err);
      throw e;
    }
  }, [fetchProxies]);

  return { proxies, isLoading, error, fetchProxies, createProxy, updateProxy, deleteProxy };
}
