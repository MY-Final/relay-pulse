import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { CopyPlus } from 'lucide-react';
import type { TFunction } from 'i18next';
import type { LatestProbeSnapshot, MonitorSummary } from '../../types/monitor';

interface MonitorListProps {
  monitors: MonitorSummary[];
  total: number;
  isLoading: boolean;
  boardFilter: string;
  setBoardFilter: (v: string) => void;
  statusFilter: string;
  setStatusFilter: (v: string) => void;
  searchQuery: string;
  setSearchQuery: (v: string) => void;
  onSelect: (key: string) => void;
  onRefresh: () => void;
  onDuplicate: (key: string) => Promise<void>;
}

export function MonitorList({
  monitors, total, isLoading,
  boardFilter, setBoardFilter,
  statusFilter, setStatusFilter,
  searchQuery, setSearchQuery,
  onSelect, onRefresh, onDuplicate,
}: MonitorListProps) {
  const { t } = useTranslation();
  const [duplicatingKey, setDuplicatingKey] = useState<string | null>(null);

  const handleDuplicate = async (key: string) => {
    setDuplicatingKey(key);
    try {
      await onDuplicate(key);
    } finally {
      setDuplicatingKey(null);
    }
  };

  return (
    <div className="space-y-4">
      {/* 过滤栏 */}
      <div className="flex flex-wrap gap-3 items-center">
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          placeholder={t('admin.monitors.searchPlaceholder')}
          className="flex-1 min-w-[200px] px-3 py-2 rounded-lg bg-elevated border border-default text-primary text-sm focus:outline-none focus:border-accent"
        />

        <select
          value={boardFilter}
          onChange={(e) => setBoardFilter(e.target.value)}
          className="px-3 py-2 rounded-lg bg-elevated border border-default text-primary text-sm"
        >
          <option value="">{t('admin.monitors.allBoards')}</option>
          <option value="hot">Hot</option>
          <option value="secondary">Secondary</option>
          <option value="cold">Cold</option>
        </select>

        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="px-3 py-2 rounded-lg bg-elevated border border-default text-primary text-sm"
        >
          <option value="">{t('admin.monitors.allStatus')}</option>
          <option value="active">{t('admin.monitors.statusActive')}</option>
          <option value="disabled">{t('admin.monitors.statusDisabled')}</option>
          <option value="hidden">{t('admin.monitors.statusHidden')}</option>
        </select>

        <button
          onClick={onRefresh}
          className="px-3 py-2 rounded-lg border border-default text-secondary hover:text-primary text-sm transition"
        >
          {t('admin.monitors.refresh')}
        </button>
      </div>

      {/* 统计 */}
      <div className="text-sm text-muted">
        {t('admin.monitors.total', { count: total })}
      </div>

      {/* 表格 */}
      {isLoading ? (
        <div className="text-center py-8 text-muted">{t('admin.table.loading')}</div>
      ) : monitors.length === 0 ? (
        <div className="text-center py-8 text-muted">{t('admin.table.empty')}</div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-default text-left text-muted">
                <th className="py-2 px-3">{t('admin.monitors.colProvider')}</th>
                <th className="py-2 px-3">{t('admin.monitors.colService')}</th>
                <th className="py-2 px-3">{t('admin.monitors.colChannel')}</th>
                <th className="py-2 px-3">{t('admin.monitors.colModels')}</th>
                <th className="py-2 px-3 text-center">{t('admin.monitors.colLatestStatus')}</th>
                <th className="py-2 px-3 text-right">{t('admin.monitors.colLatestLatency')}</th>
                <th className="py-2 px-3">{t('admin.monitors.colLatestTime')}</th>
                <th className="py-2 px-3">{t('admin.monitors.colBoard')}</th>
                <th className="py-2 px-3">{t('admin.monitors.colStatus')}</th>
                <th className="py-2 px-3">{t('admin.monitors.colSource')}</th>
                <th className="py-2 px-3 text-right">{t('admin.monitors.colActions')}</th>
              </tr>
            </thead>
            <tbody>
              {monitors.map((m) => (
                <tr
                  key={m.key}
                  onClick={() => onSelect(m.key)}
                  className="border-b border-default/50 hover:bg-elevated/50 cursor-pointer transition"
                >
                  <td className="py-2.5 px-3 text-primary font-medium">{m.provider}</td>
                  <td className="py-2.5 px-3 text-secondary">{m.service}</td>
                  <td className="py-2.5 px-3 text-secondary">{m.channel_name || m.channel}</td>
                  <td className="py-2.5 px-3 text-muted">{m.model_count}</td>
                  <td className="py-2.5 px-3 text-center">
                    <LatestStatusDot probe={m.latest_probe} />
                  </td>
                  <td className="py-2.5 px-3 text-right text-secondary">
                    {m.latest_probe && m.latest_probe.latency > 0
                      ? `${m.latest_probe.latency}ms`
                      : <span className="text-muted">-</span>}
                  </td>
                  <td className="py-2.5 px-3 text-muted text-xs">
                    {m.latest_probe
                      ? formatRelativeTime(m.latest_probe.timestamp, t)
                      : '-'}
                  </td>
                  <td className="py-2.5 px-3">
                    <BoardBadge board={m.board} />
                  </td>
                  <td className="py-2.5 px-3">
                    <StatusBadge disabled={m.disabled} hidden={m.hidden} />
                  </td>
                  <td className="py-2.5 px-3 text-muted text-xs">{m.source}</td>
                  <td className="py-2.5 px-3 text-right">
                    <button
                      type="button"
                      onClick={(e) => { e.stopPropagation(); void handleDuplicate(m.key); }}
                      disabled={duplicatingKey !== null}
                      className="inline-flex items-center gap-1.5 rounded-lg border border-accent/30 px-2.5 py-1 text-xs text-accent transition hover:bg-accent/10 disabled:opacity-50"
                      title={t('admin.monitors.duplicate')}
                      aria-label={t('admin.monitors.duplicate')}
                    >
                      <CopyPlus className="h-3.5 w-3.5" aria-hidden="true" />
                      {duplicatingKey === m.key ? t('admin.monitors.duplicating') : t('admin.monitors.duplicate')}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function BoardBadge({ board }: { board: string }) {
  const { t } = useTranslation();
  const colors: Record<string, string> = {
    hot: 'bg-success/20 text-success',
    secondary: 'bg-warning/20 text-warning',
    cold: 'bg-muted/20 text-muted',
  };
  const labels: Record<string, string> = {
    hot: t('admin.monitors.boardHot'),
    secondary: t('admin.monitors.boardSecondary'),
    cold: t('admin.monitors.boardCold'),
  };
  // 空 board 字段视为 hot（与服务端默认语义一致），保证颜色与文字始终匹配
  const effectiveBoard = board || 'hot';
  return (
    <span className={`px-2 py-0.5 rounded text-xs font-medium ${colors[effectiveBoard] || colors.hot}`}>
      {labels[effectiveBoard] || labels.hot}
    </span>
  );
}

function LatestStatusDot({ probe }: { probe?: LatestProbeSnapshot }) {
  const { t } = useTranslation();
  if (!probe) {
    return <span className="text-muted text-xs" title={t('admin.monitors.latestProbe.none')}>-</span>;
  }
  const color =
    probe.status === 1 ? 'bg-success' :
    probe.status === 2 ? 'bg-warning' :
    probe.status === 0 ? 'bg-danger' :
    'bg-muted';
  // tooltip 显示 sub_status + http_code，方便鼠标停留看细节
  const detailParts = [
    probe.sub_status,
    probe.http_code ? `HTTP ${probe.http_code}` : '',
  ].filter(Boolean);
  const title = detailParts.length > 0 ? detailParts.join(' · ') : undefined;
  return <span className={`inline-block w-2.5 h-2.5 rounded-full ${color}`} title={title} />;
}

// formatRelativeTime 把 Unix 秒转成"X 分钟前 / X 小时前 / YYYY-MM-DD"。
// 与现状页 Tooltip 的时间格式保持一致：分钟级精度即可，不必到秒。
function formatRelativeTime(unixSec: number, t: TFunction): string {
  const now = Date.now() / 1000;
  const diff = Math.max(0, now - unixSec);
  if (diff < 60) return t('admin.monitors.latestProbe.justNow');
  if (diff < 3600) return t('admin.monitors.latestProbe.minutesAgo', { count: Math.floor(diff / 60) });
  if (diff < 86400) return t('admin.monitors.latestProbe.hoursAgo', { count: Math.floor(diff / 3600) });
  if (diff < 7 * 86400) return t('admin.monitors.latestProbe.daysAgo', { count: Math.floor(diff / 86400) });
  // 超过 7 天回退到绝对日期，避免"30 天前"造成的歧义
  const d = new Date(unixSec * 1000);
  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, '0');
  const dd = String(d.getDate()).padStart(2, '0');
  return `${yyyy}-${mm}-${dd}`;
}

function StatusBadge({ disabled, hidden }: { disabled: boolean; hidden: boolean }) {
  const { t } = useTranslation();
  if (disabled) {
    return <span className="px-2 py-0.5 rounded text-xs font-medium bg-danger/20 text-danger">{t('admin.monitors.statusDisabled')}</span>;
  }
  if (hidden) {
    return <span className="px-2 py-0.5 rounded text-xs font-medium bg-warning/20 text-warning">{t('admin.monitors.statusHidden')}</span>;
  }
  return <span className="px-2 py-0.5 rounded text-xs font-medium bg-success/20 text-success">{t('admin.monitors.statusActive')}</span>;
}
