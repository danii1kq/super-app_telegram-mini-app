import { useNavigate } from 'react-router-dom';
import { AppHeader, Screen } from '@/shared/ui';
import { Placeholder } from '@/shared/ui/Placeholder';

/** SruDoc entry screen. */
export function DocHomePage() {
  const navigate = useNavigate();
  return (
    <Screen header={<AppHeader title="SruDoc" onBack={() => navigate('/apps')} leadingIcon="description" />}>
      <Placeholder
        title="SruDoc"
        icon="description"
        text="Заказы документов появятся после подключения API SruDoc."
      />
    </Screen>
  );
}
