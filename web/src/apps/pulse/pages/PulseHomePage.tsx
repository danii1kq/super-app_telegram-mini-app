import { useNavigate } from 'react-router-dom';
import { AppHeader, Screen } from '@/shared/ui';
import { Placeholder } from '@/shared/ui/Placeholder';

/** SruPulse entry screen. */
export function PulseHomePage() {
  const navigate = useNavigate();
  return (
    <Screen header={<AppHeader title="SruPulse" onBack={() => navigate('/apps')} leadingIcon="bolt" />}>
      <Placeholder title="SruPulse" icon="bolt" text="Статусы свободного времени появятся после подключения API." />
    </Screen>
  );
}
