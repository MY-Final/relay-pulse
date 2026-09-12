import { useState, useCallback, useEffect, useRef } from 'react';
import {
  apiGet, apiPost, apiPut, apiDelete, ApiError,
  setAdminCSRFToken, setAdminSessionExpiredHandler,
} from '../utils/apiClient';
import type {
  AdminSubmission,
  AdminListResponse,
  AdminDetailResponse,
  OnboardingTestResult,
  SubmissionStatus,
} from '../types/onboarding';

const SEARCH_DEBOUNCE_MS = 300;

interface UseAdminOptions {
  /** 私人部署后台不需要加载申请列表，但保留默认值兼容其它调用方。 */
  manageSubmissions?: boolean;
}

export function useAdmin({ manageSubmissions = true }: UseAdminOptions = {}) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isCheckingAuth, setIsCheckingAuth] = useState(true);

  // List state
  const [submissions, setSubmissions] = useState<AdminSubmission[]>([]);
  const [total, setTotal] = useState(0);
  const [statusFilter, setStatusFilter] = useState<SubmissionStatus | 'all'>('all');
  const [page, setPage] = useState(1);
  const [isLoading, setIsLoading] = useState(false);
  // searchQuery 跟随输入即时更新（供受控输入框回显），debouncedSearchQuery 才驱动请求。
  const [searchQuery, setSearchQuery] = useState('');
  const [debouncedSearchQuery, setDebouncedSearchQuery] = useState('');

  // Detail state
  const [selectedSubmission, setSelectedSubmission] = useState<AdminSubmission | null>(null);
  const [detailLoadingId, setDetailLoadingId] = useState<string | null>(null);
  const detailAbortRef = useRef<AbortController | null>(null);

  const [error, setError] = useState<string | null>(null);
  const [suggestedChannel, setSuggestedChannel] = useState<string>('');

  const authHeaders = useCallback((): HeadersInit => ({}), []);

  const login = useCallback(async (username: string, password: string) => {
    setError(null);
    try {
      const resp = await apiPost<{ authenticated: boolean; csrf_token?: string }>(
        '/api/admin/login',
        { username, password },
      );
      setAdminCSRFToken(resp.csrf_token || '');
      setIsAuthenticated(true);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '登录失败');
      throw e;
    }
  }, []);

  const logout = useCallback(async () => {
    try {
      await apiPost('/api/admin/logout', {});
    } finally {
      setAdminCSRFToken('');
      setIsAuthenticated(false);
    }
  }, []);

  useEffect(() => {
    const handleSessionExpired = () => {
      setAdminCSRFToken('');
      setIsAuthenticated(false);
    };
    setAdminSessionExpiredHandler(handleSessionExpired);
    return () => setAdminSessionExpiredHandler(null);
  }, []);

  useEffect(() => {
    let active = true;
    apiGet<{ authenticated: boolean; csrf_token?: string }>('/api/admin/me')
      .then(resp => {
        if (!active) return;
        setAdminCSRFToken(resp.csrf_token || '');
        setIsAuthenticated(!!resp.authenticated);
      })
      .catch(() => {
        if (!active) return;
        setAdminCSRFToken('');
        setIsAuthenticated(false);
      })
      .finally(() => {
        if (active) setIsCheckingAuth(false);
      });
    return () => { active = false; };
  }, []);

  // 输入做 debounce：稳定 300ms 后才更新驱动请求的值，并同时回到第 1 页。
  // 把 setPage(1) 放在这里（而非输入回调里）可避免「翻到第 N 页后开始搜索」时
  // 先打一次「旧关键词 + 第 1 页」的多余请求，从而消除潜在的旧响应覆盖新响应。
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearchQuery(searchQuery.trim());
      setPage(1);
    }, SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [searchQuery]);

  // Fetch list
  const fetchList = useCallback(async () => {
    if (!isAuthenticated || !manageSubmissions) return;
    setIsLoading(true);
    setError(null);

    try {
      const limit = 20;
      const params = new URLSearchParams({
        status: statusFilter,
        limit: String(limit),
        offset: String((page - 1) * limit),
      });
      if (debouncedSearchQuery) params.set('q', debouncedSearchQuery);
      const resp = await apiGet<AdminListResponse>(
        `/api/admin/submissions?${params.toString()}`,
        { headers: authHeaders() },
      );
      setSubmissions(resp.submissions || []);
      setTotal(resp.total);
    } catch (e) {
      if (e instanceof ApiError && (e.status === 401 || e.status === 403)) {
        setIsAuthenticated(false);
        setError('登录已失效，请重新登录');
      } else {
        setError(e instanceof ApiError ? e.message : '加载失败');
      }
    } finally {
      setIsLoading(false);
    }
  }, [isAuthenticated, manageSubmissions, statusFilter, page, debouncedSearchQuery, authHeaders]);

  // Auto-fetch on filter/page change
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 鉴权/筛选变更即取数：fetchList 在 await 前同步置 loading/清错误为有意
    if (isAuthenticated && manageSubmissions) fetchList();
  }, [isAuthenticated, manageSubmissions, fetchList]);

  // 拉取模板列表（供 SubmissionDetail 的模板下拉使用）
  //
  // 显式抛错：当后端不可达 / 鉴权失效 / 返回非 2xx 时，调用方应感知失败并降级
  // 为禁用控件，而不是静默回退到空数组。沿用 useMonitorAdmin 的"吞错返回 []"
  // 会让管理员看到"该服务类型暂无模板"的假象，掩盖配置故障。
  const fetchTemplates = useCallback(async (serviceType?: string): Promise<string[]> => {
    if (!isAuthenticated) return [];

    const normalized = serviceType?.trim() ?? '';
    const qs = normalized ? `?service_type=${encodeURIComponent(normalized)}` : '';
    const resp = await apiGet<{ templates: string[] }>(
      `/api/admin/templates${qs}`,
      { headers: authHeaders() },
    );
    return resp.templates ?? [];
  }, [isAuthenticated, authHeaders]);

  // Fetch detail
  const fetchDetail = useCallback(async (publicId: string) => {
    if (!isAuthenticated) return;
    detailAbortRef.current?.abort(); // 中止上一条在途详情，防止迟到响应覆盖新选中项
    const ac = new AbortController();
    detailAbortRef.current = ac;
    setDetailLoadingId(publicId);
    setError(null);
    try {
      const resp = await apiGet<AdminDetailResponse>(
        `/api/admin/submissions/${publicId}`,
        { headers: authHeaders(), signal: ac.signal },
      );
      if (ac.signal.aborted) return;
      setSelectedSubmission(resp.submission);
    } catch (e) {
      if (ac.signal.aborted || (e instanceof DOMException && e.name === 'AbortError')) return;
      setError(e instanceof ApiError ? e.message : '加载详情失败');
    } finally {
      // 仅当自己仍是最新一次请求时才清 loading：被后续请求取代的旧请求不得清掉新请求的 loading 态
      if (detailAbortRef.current === ac) setDetailLoadingId(null);
    }
  }, [isAuthenticated, authHeaders]);

  // 供切 tab / 返回列表时中止在途详情请求，避免迟到响应写回已离开的视图
  const cancelDetail = useCallback(() => {
    detailAbortRef.current?.abort();
    detailAbortRef.current = null;
    setDetailLoadingId(null);
  }, []);

  // Update submission
  const updateSubmission = useCallback(async (publicId: string, updates: Record<string, unknown>) => {
    if (!isAuthenticated) return;
    setError(null);

    try {
      const resp = await apiPut<{ submission: AdminSubmission }>(
        `/api/admin/submissions/${publicId}`,
        updates,
        { headers: authHeaders() },
      );
      setSelectedSubmission(resp.submission);
      fetchList(); // refresh list
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '更新失败');
    }
  }, [isAuthenticated, authHeaders, fetchList]);

  // Reject
  const rejectSubmission = useCallback(async (publicId: string, note: string) => {
    if (!isAuthenticated) return;
    setError(null);

    try {
      await apiPost(`/api/admin/submissions/${publicId}/reject`, { note }, { headers: authHeaders() });
      fetchList();
      setSelectedSubmission(null);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '驳回失败');
    }
  }, [isAuthenticated, authHeaders, fetchList]);

  // Test — inline probe, returns result synchronously
  const testSubmission = useCallback(async (publicId: string): Promise<OnboardingTestResult | null> => {
    if (!isAuthenticated) return null;
    setError(null);

    try {
      const resp = await apiPost<OnboardingTestResult>(
        `/api/admin/submissions/${publicId}/test`,
        {},
        { headers: authHeaders() },
      );
      return resp;
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '测试失败');
      return null;
    }
  }, [isAuthenticated, authHeaders]);

  // Delete
  const deleteSubmission = useCallback(async (publicId: string) => {
    if (!isAuthenticated) return;
    setError(null);

    try {
      await apiDelete(`/api/admin/submissions/${publicId}`, { headers: authHeaders() });
      fetchList();
      setSelectedSubmission(null);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : '删除失败');
    }
  }, [isAuthenticated, authHeaders, fetchList]);

  // Publish
  const publishSubmission = useCallback(async (publicId: string, board = 'hot') => {
    if (!isAuthenticated) return;
    setError(null);
    setSuggestedChannel('');

    try {
      await apiPost(`/api/admin/submissions/${publicId}/publish`, { board }, { headers: authHeaders() });
      fetchList();
      setSelectedSubmission(null);
    } catch (e) {
      if (e instanceof ApiError && e.status === 409 && e.data) {
        const suggested = e.data.suggested_channel;
        if (typeof suggested === 'string') {
          setSuggestedChannel(suggested);
        }
      }
      setError(e instanceof ApiError ? e.message : '上架失败');
    }
  }, [isAuthenticated, authHeaders, fetchList]);

  return {
    // Auth
    isAuthenticated,
    isCheckingAuth,
    login,
    logout,

    // List
    submissions,
    total,
    statusFilter,
    setStatusFilter,
    page,
    setPage,
    isLoading,
    searchQuery,
    setSearchQuery,
    fetchList,

    // Detail
    selectedSubmission,
    detailLoadingId,
    fetchDetail,
    cancelDetail,
    fetchTemplates,
    updateSubmission,
    testSubmission,
    rejectSubmission,
    deleteSubmission,
    publishSubmission,
    setSelectedSubmission,

    error,
    suggestedChannel,
  };
}
