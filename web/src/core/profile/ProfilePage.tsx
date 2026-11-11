import { useState } from 'react';
import { useSession } from '@/app/session';
import { api } from '@/shared/api/client';
import { AppHeader, Card, Hairline, Row, Screen, Segmented, SectionLabel, Switch } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useI18n, type Lang } from '@/shared/i18n';
import { saveThemeMode, applyThemeMode, type ThemeMode } from '@/app/theme';

/** The build the user is running. Shown in the profile, not hidden in a menu. */
export const APP_VERSION = 'v1.0.0';

/** Profile and shell settings. */
export function ProfilePage() {
  const { me } = useSession();
  const { lang, setLang, t } = useI18n();
  const feedback = useFeedback();
  const [theme, setTheme] = useState<ThemeMode>(me?.theme ?? 'auto');

  const patch = async (body: Record<string, unknown>) => {
    try {
      await api('/me/settings', { method: 'PATCH', body });
      feedback(t('saved'));
    } catch {
      feedback(t('saveFailed'), 'error');
    }
  };

  const changeTheme = (next: ThemeMode) => {
    setTheme(next);
    saveThemeMode(next);
    applyThemeMode(next);
    void patch({ theme: next });
  };

  const changeLang = (next: Lang) => {
    setLang(next);
    void patch({ lang: next });
  };

  return (
    <Screen header={<AppHeader title={t('profile')} leadingIcon="person" />}>
      <SectionLabel>{t('account')}</SectionLabel>
      <Card>
        <Row
          icon="person"
          title={me?.public_name ?? t('guest')}
          subtitle={me?.username ? `@${me.username}` : t('telegramNotLinked')}
        />
      </Card>

      <SectionLabel>{t('appearance')}</SectionLabel>
      <Card className="p-3">
        <Segmented
          options={[
            { value: 'auto', label: t('themeAuto') },
            { value: 'light', label: t('themeLight') },
            { value: 'dark', label: t('themeDark') },
          ]}
          value={theme}
          onChange={changeTheme}
        />
      </Card>

      <SectionLabel>{t('language')}</SectionLabel>
      <Card className="p-3">
        <Segmented
          options={[
            { value: 'ru', label: 'Русский' },
            { value: 'kk', label: 'Қазақша' },
            { value: 'en', label: 'English' },
          ]}
          value={lang}
          onChange={changeLang}
        />
      </Card>

      <SectionLabel>{t('notifications')}</SectionLabel>
      <Card>
        <Row
          icon="notifications"
          title={t('notificationsTitle')}
          subtitle={t('notificationsSubtitle')}
          trailing={<Switch checked label={t('notifications')} onChange={() => feedback(t('soon'))} />}
        />
        <Hairline />
        <Row
          icon="volume_off"
          title={t('muteDialog')}
          subtitle={t('muteDialogSubtitle')}
          iconTone="ink-2"
        />
      </Card>

      <SectionLabel>{t('timezone')}</SectionLabel>
      <Card>
        <Row icon="schedule" title={me?.timezone ?? 'Asia/Almaty'} subtitle="GMT+5" />
      </Card>

      <SectionLabel>{t('version')}</SectionLabel>
      <Card>
        <Row icon="info" title={APP_VERSION} subtitle="SruNal" iconTone="ink-2" />
      </Card>

      <div className="px-1 pt-4 text-caption text-ink-2">{t('profileNote')}</div>
    </Screen>
  );
}
