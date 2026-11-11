import { useNavigate } from 'react-router-dom';
import { AppHeader, Screen } from '@/shared/ui';
import { Placeholder } from '@/shared/ui/Placeholder';

/** SruSplit entry screen. */
export function SplitHomePage() {
  const navigate = useNavigate();
  return (
    <Screen header={<AppHeader title="SruSplit" onBack={() => navigate('/apps')} leadingIcon="receipt_long" />}>
      <Placeholder title="SruSplit" icon="receipt_long" text="Группы расходов появятся после подключения API." />
    </Screen>
  );
}
