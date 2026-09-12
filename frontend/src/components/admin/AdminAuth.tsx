import { useState } from 'react';
import { useTranslation } from 'react-i18next';

interface AdminAuthProps {
  onSubmit: (username: string, password: string) => Promise<void>;
  error?: string | null;
}

export const AdminAuth: React.FC<AdminAuthProps> = ({ onSubmit, error }) => {
  const { t } = useTranslation();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim() || !password || isSubmitting) return;
    setIsSubmitting(true);
    try {
      await onSubmit(username.trim(), password);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-page flex items-center justify-center px-4">
      <div className="w-full max-w-sm bg-surface border border-default rounded-lg p-8">
        <h1 className="text-xl font-semibold text-primary text-center mb-6">
          {t('admin.auth.title')}
        </h1>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label
              htmlFor="admin-username"
              className="block text-sm font-medium text-secondary mb-1.5"
            >
              {t('admin.auth.usernameLabel')}
            </label>
            <input
              id="admin-username"
              type="text"
              autoComplete="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder={t('admin.auth.usernamePlaceholder')}
              className="w-full px-3 py-2 bg-elevated border border-default rounded-md
                         text-primary placeholder:text-muted text-sm
                         focus:outline-none focus:border-accent focus:ring-1 focus:ring-accent
                         transition-colors"
              disabled={isSubmitting}
            />
          </div>

          <div>
            <label
              htmlFor="admin-password"
              className="block text-sm font-medium text-secondary mb-1.5"
            >
              {t('admin.auth.passwordLabel')}
            </label>
            <input
              id="admin-password"
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={t('admin.auth.passwordPlaceholder')}
              className="w-full px-3 py-2 bg-elevated border border-default rounded-md
                         text-primary placeholder:text-muted text-sm
                         focus:outline-none focus:border-accent focus:ring-1 focus:ring-accent
                         transition-colors"
              disabled={isSubmitting}
            />
          </div>

          {error && <p className="text-sm text-danger" role="alert">{error}</p>}

          <button
            type="submit"
            disabled={!username.trim() || !password || isSubmitting}
            className="w-full px-4 py-2.5 text-sm font-medium rounded-md border
                       bg-accent/10 border-accent/40 text-accent
                       hover:bg-accent/20 transition-colors
                       disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {isSubmitting ? t('admin.auth.submitting') : t('admin.auth.login')}
          </button>
        </form>
      </div>
    </div>
  );
};
