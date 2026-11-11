import { useNavigate } from 'react-router-dom';
import { AppHeader, Screen } from '@/shared/ui';
import { Placeholder } from '@/shared/ui/Placeholder';

/** SruDay entry screen. */
export function DayHomePage() {
  const navigate = useNavigate();
  return (
    <Screen header={<AppHeader title="SruDay" onBack={() => navigate('/apps')} leadingIcon="calendar_month" />}>
      <Placeholder
        title="SruDay"
        icon="calendar_month"
        text="Календарь появится после подключения API SruDay."
      />
    </Screen>
  );
}
