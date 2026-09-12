import { useState } from 'react';
import { Pencil, Plus, Save, Trash2, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { ProxyProfile } from '../../types/proxy';
import { FormField } from './FormControls';

interface ProxyProfilesProps {
  proxies: ProxyProfile[];
  isLoading: boolean;
  onCreate: (name: string, url: string) => Promise<void>;
  onUpdate: (id: string, name: string, url: string) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
}

export function ProxyProfiles({ proxies, isLoading, onCreate, onUpdate, onDelete }: ProxyProfilesProps) {
  const { t } = useTranslation();
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [editingID, setEditingID] = useState<string | null>(null);
  const [editingName, setEditingName] = useState('');
  const [editingURL, setEditingURL] = useState('');
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const create = async (event: React.FormEvent) => {
    event.preventDefault();
    setFormError(null);
    setBusy(true);
    try {
      await onCreate(name.trim(), url.trim());
      setName('');
      setUrl('');
    } catch (e) {
      setFormError(e instanceof Error ? e.message : t('admin.proxies.saveFailed'));
    } finally {
      setBusy(false);
    }
  };

  const saveEdit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!editingID) return;
    setFormError(null);
    setBusy(true);
    try {
      await onUpdate(editingID, editingName.trim(), editingURL.trim());
      setEditingID(null);
    } catch (e) {
      setFormError(e instanceof Error ? e.message : t('admin.proxies.saveFailed'));
    } finally {
      setBusy(false);
    }
  };

  const remove = async (profile: ProxyProfile) => {
    if (!window.confirm(t('admin.proxies.confirmDelete', { name: profile.name }))) return;
    setFormError(null);
    setBusy(true);
    try {
      await onDelete(profile.id);
    } catch (e) {
      setFormError(e instanceof Error ? e.message : t('admin.proxies.deleteFailed'));
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className="space-y-5">
      <div>
        <h2 className="text-xl font-bold text-primary">{t('admin.proxies.title')}</h2>
        <p className="mt-1 text-sm text-muted">{t('admin.proxies.description')}</p>
      </div>

      {formError && <div className="p-3 bg-danger/10 border border-danger/20 rounded-lg text-danger text-sm">{formError}</div>}

      <form onSubmit={create} className="bg-surface rounded-lg border border-default p-4 space-y-3">
        <h3 className="text-sm font-semibold text-primary">{t('admin.proxies.addTitle')}</h3>
        <div className="grid grid-cols-1 md:grid-cols-[1fr_2fr_auto] gap-3 items-end">
          <FormField label={t('admin.proxies.name')} value={name} onChange={setName} placeholder={t('admin.proxies.namePlaceholder')} />
          <FormField label={t('admin.proxies.url')} value={url} onChange={setUrl} type="text" placeholder={t('admin.proxies.urlPlaceholder')} />
          <button type="submit" disabled={busy || !name.trim() || !url.trim()} className="inline-flex items-center justify-center gap-1.5 px-4 py-2 rounded-md bg-accent/10 text-accent text-sm font-medium hover:bg-accent/20 transition disabled:opacity-50">
            <Plus size={15} aria-hidden="true" />{t('admin.proxies.add')}
          </button>
        </div>
      </form>

      {isLoading ? (
        <div className="py-10 text-center text-sm text-muted">{t('admin.proxies.loading')}</div>
      ) : proxies.length === 0 ? (
        <div className="bg-surface rounded-lg border border-dashed border-default p-8 text-center text-sm text-muted">{t('admin.proxies.empty')}</div>
      ) : (
        <div className="bg-surface rounded-lg border border-default divide-y divide-default">
          {proxies.map(profile => editingID === profile.id ? (
            <form key={profile.id} onSubmit={saveEdit} className="p-4 grid grid-cols-1 md:grid-cols-[1fr_2fr_auto] gap-3 items-end">
              <FormField label={t('admin.proxies.name')} value={editingName} onChange={setEditingName} />
              <FormField label={t('admin.proxies.url')} value={editingURL} onChange={setEditingURL} placeholder={profile.url_masked} />
              <div className="flex gap-2">
                <button type="submit" disabled={busy || !editingName.trim()} className="inline-flex items-center gap-1 px-3 py-2 rounded-md bg-accent/10 text-accent text-sm hover:bg-accent/20 disabled:opacity-50"><Save size={15} aria-hidden="true" />{t('admin.proxies.save')}</button>
                <button type="button" onClick={() => setEditingID(null)} className="inline-flex items-center gap-1 px-3 py-2 rounded-md border border-default text-secondary text-sm hover:text-primary"><X size={15} aria-hidden="true" />{t('admin.detail.cancel')}</button>
              </div>
            </form>
          ) : (
            <div key={profile.id} className="p-4 flex flex-wrap items-center gap-x-5 gap-y-2">
              <div className="min-w-[12rem] flex-1">
                <div className="text-sm font-medium text-primary">{profile.name}</div>
                <div className="mt-1 text-xs text-muted font-mono break-all">{profile.url_masked}</div>
              </div>
              <div className="text-xs text-muted font-mono">{profile.id}</div>
              <div className="flex gap-2">
                <button type="button" onClick={() => { setEditingID(profile.id); setEditingName(profile.name); setEditingURL(''); setFormError(null); }} className="inline-flex items-center gap-1 px-3 py-1.5 rounded-md border border-default text-secondary text-xs hover:text-primary"><Pencil size={14} aria-hidden="true" />{t('admin.proxies.edit')}</button>
                <button type="button" disabled={busy} onClick={() => remove(profile)} className="inline-flex items-center gap-1 px-3 py-1.5 rounded-md border border-danger/30 text-danger text-xs hover:bg-danger/10 disabled:opacity-50"><Trash2 size={14} aria-hidden="true" />{t('admin.proxies.delete')}</button>
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

