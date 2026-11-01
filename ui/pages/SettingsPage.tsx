import React, { useEffect, useState } from 'react';
import { KeyRound, Monitor, RefreshCw, Shield, User } from 'lucide-react';
import { useTranslation } from '@/store/translations';
import { useAuthStore } from '@/store/authStore';
import { useUIStore, type Language } from '@/store/uiStore';
import { authService } from '@/services/authService';
import { Button } from '@/components/ui';
import { configStyles } from '@/styles/configStyles';

type Section = 'profile' | 'security' | 'appearance';

const MIN_PASSWORD_LENGTH = 8;

export const SettingsPage: React.FC = () => {
  const t = useTranslation();
  const [section, setSection] = useState<Section>('profile');

  const nav: Array<{ id: Section; label: string; icon: React.ReactNode }> = [
    { id: 'profile', label: t.settings.profile, icon: <User size={18} /> },
    { id: 'security', label: t.settings.security, icon: <Shield size={18} /> },
    { id: 'appearance', label: t.settings.appearance, icon: <Monitor size={18} /> },
  ];

  return (
    <div className="flex-1 h-full overflow-y-auto bg-slate-50 dark:bg-slate-950 p-6 md:p-8">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-2xl font-bold text-slate-900 dark:text-white mb-6">{t.settings.title}</h1>

        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <nav className="md:col-span-1 space-y-1" aria-label={t.settings.title}>
            {nav.map((item) => (
              <button
                key={item.id}
                type="button"
                onClick={() => setSection(item.id)}
                aria-current={section === item.id ? 'page' : undefined}
                className={`w-full flex items-center px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                  section === item.id
                    ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-400'
                    : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800'
                }`}
              >
                <span className="mr-3">{item.icon}</span>
                {item.label}
              </button>
            ))}
          </nav>

          <div className="md:col-span-3 space-y-6">
            {section === 'profile' && <ProfileSection />}
            {section === 'security' && <SecuritySection />}
            {section === 'appearance' && <AppearanceSection />}
          </div>
        </div>
      </div>
    </div>
  );
};

const Card: React.FC<{ title: string; description: string; children: React.ReactNode; footer?: React.ReactNode }> = ({
  title,
  description,
  children,
  footer,
}) => (
  <section className="bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 rounded-xl shadow-sm overflow-hidden">
    <div className="p-6 border-b border-slate-100 dark:border-slate-800">
      <h2 className="text-lg font-bold text-slate-900 dark:text-white">{title}</h2>
      <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">{description}</p>
    </div>
    <div className="p-6">{children}</div>
    {footer && (
      <div className="p-4 bg-slate-50 dark:bg-slate-900/50 border-t border-slate-100 dark:border-slate-800 flex justify-end">
        {footer}
      </div>
    )}
  </section>
);

const ProfileSection: React.FC = () => {
  const t = useTranslation();
  const user = useAuthStore((state) => state.user);
  const updateUser = useAuthStore((state) => state.updateUser);
  const [loading, setLoading] = useState(false);
  const [failed, setFailed] = useState(false);

  const reload = async () => {
    setLoading(true);
    try {
      updateUser(await authService.getCurrentUser());
      setFailed(false);
    } catch {
      setFailed(true);
    } finally {
      setLoading(false);
    }
  };

  // Always show what GET /auth/me returns, not a stale persisted copy.
  useEffect(() => {
    void reload();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const formatDate = (value?: string) => (value ? new Date(value).toLocaleString() : '-');

  const rows: Array<[string, React.ReactNode]> = user
    ? [
        [t.settings.fullName, user.full_name || '-'],
        [t.settings.username, user.username || '-'],
        [t.settings.email, user.email],
        [t.settings.accountType, user.is_admin ? t.settings.administrator : t.settings.regularUser],
        [t.settings.roles, user.roles?.length ? user.roles.join(', ') : t.settings.noRoles],
        [t.settings.memberSince, formatDate(user.created_at)],
        [t.settings.lastLogin, formatDate(user.last_login_at)],
      ]
    : [];

  return (
    <Card
      title={t.settings.profileInfo}
      description={t.settings.profileDesc}
      footer={
        <Button
          variant="outline"
          size="sm"
          onClick={() => void reload()}
          loading={loading}
          icon={<RefreshCw size={16} />}
        >
          {t.settings.reload}
        </Button>
      }
    >
      {failed && !user && (
        <p role="alert" className="text-sm text-red-600 dark:text-red-400">
          {t.settings.notLoaded}
        </p>
      )}
      {user && (
        <dl className="grid grid-cols-1 sm:grid-cols-3 gap-x-4 gap-y-3 text-sm">
          {rows.map(([label, value]) => (
            <React.Fragment key={label}>
              <dt className="text-slate-500 dark:text-slate-400">{label}</dt>
              <dd className="sm:col-span-2 font-medium text-slate-800 dark:text-slate-200 break-all">{value}</dd>
            </React.Fragment>
          ))}
        </dl>
      )}
    </Card>
  );
};

const SecuritySection: React.FC = () => {
  const t = useTranslation();
  const changePassword = useAuthStore((state) => state.changePassword);
  const isLoading = useAuthStore((state) => state.isLoading);
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirm, setConfirm] = useState('');
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!oldPassword) return setError(t.settings.passwordRequired);
    if (newPassword.length < MIN_PASSWORD_LENGTH) return setError(t.settings.passwordTooShort);
    if (newPassword !== confirm) return setError(t.settings.passwordMismatch);
    setError(null);
    const ok = await changePassword({ old_password: oldPassword, new_password: newPassword });
    if (ok) {
      setOldPassword('');
      setNewPassword('');
      setConfirm('');
    }
  };

  const input = (id: string, label: string, value: string, onChange: (v: string) => void, autoComplete: string) => (
    <div>
      <label htmlFor={id} className={configStyles.label}>
        {label}
      </label>
      <input
        id={id}
        type="password"
        value={value}
        autoComplete={autoComplete}
        onChange={(e) => onChange(e.target.value)}
        className={configStyles.input}
      />
    </div>
  );

  return (
    <Card title={t.settings.security} description={t.settings.securityDesc}>
      <form onSubmit={handleSubmit} className="space-y-4" noValidate>
        {input('current-password', t.settings.currentPassword, oldPassword, setOldPassword, 'current-password')}
        {input('new-password', t.settings.newPassword, newPassword, setNewPassword, 'new-password')}
        {input('confirm-password', t.settings.confirmPassword, confirm, setConfirm, 'new-password')}
        {error && (
          <p role="alert" className="text-sm text-red-600 dark:text-red-400">
            {error}
          </p>
        )}
        <div className="flex items-center justify-between gap-4">
          <span className="text-xs text-slate-500 dark:text-slate-400">{t.settings.passwordChangeNote}</span>
          <Button type="submit" variant="primary" size="sm" loading={isLoading} icon={<KeyRound size={16} />}>
            {t.settings.changePassword}
          </Button>
        </div>
      </form>
    </Card>
  );
};

const AppearanceSection: React.FC = () => {
  const t = useTranslation();
  const { theme, setTheme, language, setLanguage } = useUIStore();

  const choice = <T extends string>(current: T, options: Array<{ value: T; label: string }>, onChange: (v: T) => void, name: string) => (
    <div role="radiogroup" aria-label={name} className="inline-flex rounded-lg border border-slate-200 dark:border-slate-700 overflow-hidden">
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          role="radio"
          aria-checked={current === option.value}
          onClick={() => onChange(option.value)}
          className={`px-4 py-1.5 text-sm ${
            current === option.value
              ? 'bg-blue-600 text-white'
              : 'bg-white dark:bg-slate-900 text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800'
          }`}
        >
          {option.label}
        </button>
      ))}
    </div>
  );

  return (
    <Card title={t.settings.appearance} description={t.settings.appearanceDesc}>
      <div className="space-y-5">
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-slate-700 dark:text-slate-300">{t.settings.theme}</span>
          {choice(theme, [
            { value: 'light', label: t.settings.light },
            { value: 'dark', label: t.settings.dark },
          ], setTheme, t.settings.theme)}
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm font-medium text-slate-700 dark:text-slate-300">{t.settings.language}</span>
          {choice<Language>(language, [
            { value: 'en', label: 'English' },
            { value: 'ru', label: 'Русский' },
          ], setLanguage, t.settings.language)}
        </div>
      </div>
    </Card>
  );
};
