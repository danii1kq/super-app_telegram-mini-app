import { useState } from 'react';
import { useSession } from '@/app/session';
import { api } from '@/shared/api/client';
import { AppHeader, Banner, Card, Hairline, Row, Screen, Segmented, SectionLabel, Switch } from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useI18n, type Lang } from '@/shared/i18n';
import { saveThemeMode, type ThemeMode } from '@/app/theme';

/** Profile and shell settings. */
export function ProfilePage() {
  const { me, status } = useSession();
  const { lang, setLang, t } = useI18n();
  const feedback = useFeedback();
  const [theme, setTheme] = useState<ThemeMode>(me?.theme ?? 'auto');

  const patch = async (body: Record<string, unknown>) => {
    try {
      await api('/me/settings', { method: 'PATCH', body });
      feedback('Сохранено');
    } catch {
      feedback('Не удалось сохранить', 'error');
    }
  };

  const changeTheme = (next: ThemeMode) => {
    setTheme(next);
    saveThemeMode(next);
    void patch({ theme: next });
  };

  const changeLang = (next: Lang) => {
    setLang(next);
    void patch({ lang: next });
  };

  return (
    <Screen header={<AppHeader title={t('profile')} leadingIcon="person" />}>
      {status === 'outside' ? (
        <div className="pt-2">
          <Banner tone="warning" icon="info" title="Откройте через Telegram">
            Профиль привязан к аккаунту Telegram и работает внутри Mini App.
          </Banner>
        </div>
      ) : null}

      <SectionLabel>Аккаунт</SectionLabel>
      <Card>
        <Row
          icon="person"
          title={me?.public_name ?? 'Гость'}
          subtitle={me?.username ? `@${me.username}` : 'Telegram не подключён'}
        />
      </Card>

      <SectionLabel>Оформление</SectionLabel>
      <Card className="p-3">
        <Segmented
          options={[
            { value: 'auto', label: 'Как в Telegram' },
            { value: 'light', label: 'Светлая' },
            { value: 'dark', label: 'Тёмная' },
          ]}
          value={theme}
          onChange={changeTheme}
        />
      </Card>

      <SectionLabel>Язык</SectionLabel>
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

      <SectionLabel>Уведомления</SectionLabel>
      <Card>
        <Row
          icon="notifications"
          title="Сообщения в чате бота"
          subtitle="Доставка не зависит от открытого приложения"
          trailing={<Switch checked label="Уведомления" onChange={() => feedback('Настройка появится позже')} />}
        />
        <Hairline />
        <Row
          icon="volume_off"
          title="Заглушить диалог"
          subtitle="Настраивается в самом диалоге"
          iconTone="ink-2"
        />
      </Card>

      <div className="px-1 pt-4 text-caption text-ink-2">
        Часовой пояс: {me?.timezone ?? 'Asia/Qyzylorda'}. Приложения остаются отдельными: их данные
        открываются только по выбранным вами разрешениям.
      </div>
    </Screen>
  );
}
