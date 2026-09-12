import { useState } from 'react';
import { Helmet } from 'react-helmet-async';
import { useTranslation } from 'react-i18next';
import { Loader2 } from 'lucide-react';
import { useAdmin } from '../hooks/useAdmin';
import { useMonitorAdmin } from '../hooks/useMonitorAdmin';
import { AdminAuth } from '../components/admin/AdminAuth';
import { MonitorList } from '../components/admin/MonitorList';
import { MonitorDetail } from '../components/admin/MonitorDetail';
import { MonitorForm } from '../components/admin/MonitorForm';
import { ProxyProfiles } from '../components/admin/ProxyProfiles';
import { useProxyAdmin } from '../hooks/useProxyAdmin';
import { APP_NAME } from '../constants';

type AdminTab = 'monitors' | 'proxies';

export default function AdminPage() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<AdminTab>('monitors');
  const [showCreateForm, setShowCreateForm] = useState(false);

  const {
    isAuthenticated, isCheckingAuth, login, logout,
    error: adminError,
  } = useAdmin({ manageSubmissions: false });

  const monitor = useMonitorAdmin(isAuthenticated);
  const proxyAdmin = useProxyAdmin(isAuthenticated);

  const handleTabChange = (tab: AdminTab) => {
    monitor.cancelDetail();
    setActiveTab(tab);
    setShowCreateForm(false);
    monitor.setSelectedMonitor(null);
    monitor.setSelectedKey(null);
  };

  return (
    <>
      <Helmet>
        <title>{`${t('admin.meta.title')} | ${APP_NAME}`}</title>
        <meta name="robots" content="noindex,nofollow" />
      </Helmet>

      <main className="min-h-screen bg-page py-8 px-4">
        <div className="max-w-5xl mx-auto space-y-6">
          {isCheckingAuth ? (
            <DetailLoading />
          ) : !isAuthenticated ? (
            <AdminAuth
              onSubmit={login}
              error={adminError}
            />
          ) : (
            <>
              {/* 顶栏 */}
              <header className="flex items-center justify-between">
                <h1 className="text-2xl font-bold text-primary">{t('admin.title')}</h1>
                <button
                  onClick={logout}
                  className="px-3 py-1.5 text-sm rounded-lg border border-default text-secondary hover:text-primary transition"
                >
                  {t('admin.logout')}
                </button>
              </header>

              {/* Tab 导航 */}
              <nav className="flex gap-1 border-b border-default">
                <TabButton
                  active={activeTab === 'monitors'}
                  onClick={() => handleTabChange('monitors')}
                  label={t('admin.tabs.monitors')}
                />
                <TabButton
                  active={activeTab === 'proxies'}
                  onClick={() => handleTabChange('proxies')}
                  label={t('admin.tabs.proxies')}
                />
              </nav>

              {/* 错误提示 */}
              {(adminError || monitor.error || proxyAdmin.error) && (
                <div className="p-4 bg-danger/10 border border-danger/20 rounded-lg">
                  <p className="text-danger font-medium">{adminError || monitor.error || proxyAdmin.error}</p>
                </div>
              )}

              {/* 通道管理 Tab */}
              {activeTab === 'monitors' && (
                showCreateForm ? (
                  <MonitorForm
                    fetchTemplates={monitor.fetchTemplates}
                    proxyProfiles={proxyAdmin.proxies}
                    onSave={async (file) => {
                      await monitor.createMonitor(file);
                      setShowCreateForm(false);
                    }}
                    onCancel={() => setShowCreateForm(false)}
                  />
                ) : monitor.detailLoadingId && !monitor.selectedMonitor ? (
                  <DetailLoading />
                ) : monitor.selectedMonitor && monitor.selectedKey ? (
                  <MonitorDetail
                    fetchTemplates={monitor.fetchTemplates}
                    proxyProfiles={proxyAdmin.proxies}
                    monitorFile={monitor.selectedMonitor}
                    monitorKey={monitor.selectedKey}
                    onBack={() => {
                      monitor.cancelDetail();
                      monitor.setSelectedMonitor(null);
                      monitor.setSelectedKey(null);
                    }}
                    onSave={async (file, revision) => {
                      await monitor.updateMonitor(monitor.selectedKey!, file, revision);
                    }}
                    onDelete={() => {
                      if (monitor.selectedKey) {
                        monitor.deleteMonitor(monitor.selectedKey);
                      }
                    }}
                    onToggle={(field, value) => {
                      if (monitor.selectedKey) {
                        monitor.toggleMonitor(monitor.selectedKey, field, value);
                      }
                    }}
                    onProbe={async (overrides, targetModel) => {
                      if (monitor.selectedKey) {
                        return monitor.probeMonitor(monitor.selectedKey, overrides, targetModel);
                      }
                      return null;
                    }}
                    fetchLogs={monitor.fetchMonitorLogs}
                    probeTargets={monitor.probeTargets}
                    probingTargets={monitor.probingTargets}
                    probeResults={monitor.probeResults}
                    probeErrors={monitor.probeErrors}
                  />
                ) : (
                  <div className="space-y-4">
                    <div className="flex justify-end">
                      <button
                        onClick={() => setShowCreateForm(true)}
                        className="px-4 py-2 rounded-lg bg-accent/10 text-accent text-sm font-medium hover:bg-accent/20 transition"
                      >
                        {t('admin.monitors.create')}
                      </button>
                    </div>
                    <MonitorList
                      monitors={monitor.monitors}
                      total={monitor.total}
                      isLoading={monitor.isLoading}
                      boardFilter={monitor.boardFilter}
                      setBoardFilter={monitor.setBoardFilter}
                      statusFilter={monitor.statusFilter}
                      setStatusFilter={monitor.setStatusFilter}
                      searchQuery={monitor.searchQuery}
                      setSearchQuery={monitor.setSearchQuery}
                      onSelect={(key) => monitor.fetchDetail(key)}
                      onRefresh={monitor.refreshList}
                    />
                  </div>
                )
              )}
              {/* 代理配置 Tab */}
              {activeTab === 'proxies' && (
                <ProxyProfiles
                  proxies={proxyAdmin.proxies}
                  isLoading={proxyAdmin.isLoading}
                  onCreate={proxyAdmin.createProxy}
                  onUpdate={proxyAdmin.updateProxy}
                  onDelete={proxyAdmin.deleteProxy}
                />
              )}
            </>
          )}
        </div>
      </main>
    </>
  );
}

function DetailLoading() {
  const { t } = useTranslation();
  return (
    <div className="flex items-center justify-center py-16 text-muted">
      <Loader2 size={20} className="animate-spin mr-2" />
      {t('admin.table.loading')}
    </div>
  );
}

function TabButton({ active, onClick, label }: { active: boolean; onClick: () => void; label: string }) {
  return (
    <button
      onClick={onClick}
      className={`px-4 py-2.5 text-sm font-medium transition border-b-2 -mb-px ${
        active
          ? 'border-accent text-accent'
          : 'border-transparent text-muted hover:text-secondary'
      }`}
    >
      {label}
    </button>
  );
}
