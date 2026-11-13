import { useNavigate } from 'react-router-dom';
import { AppHeader, Icon, Screen } from '@/shared/ui';
import { useT } from '@/shared/i18n';

/**
 * SruGame is the hub for the games, the way SruNal groups its apps.
 *
 * The catalogue used to list SruFarm directly, which left no room for a second
 * game: adding one meant another top-level card next to the productivity apps.
 * The hub keeps the catalogue short and gives the games a place of their own.
 */
type GameId = 'farm' | 'tinder';

interface GameEntry {
  id: GameId;
  path: string;
  title: string;
  tagline: 'appFarmTagline' | 'appTinderTagline';
  icon: string;
}

const GAMES: readonly GameEntry[] = [
  { id: 'farm', path: '/apps/game/farm', title: 'SruFarm', tagline: 'appFarmTagline', icon: 'potted_plant' },
  { id: 'tinder', path: '/apps/tinder', title: 'SruTinder', tagline: 'appTinderTagline', icon: 'favorite' },
] as const;

export function GameHomePage() {
  const t = useT();
  const navigate = useNavigate();

  return (
    <Screen header={<AppHeader title="SruGame" onBack={() => navigate('/apps')} leadingIcon="sports_esports" />}>
      <div className="grid grid-cols-2 gap-3 pt-2">
        {GAMES.map((game) => (
          <button
            key={game.id}
            type="button"
            onClick={() => navigate(game.path)}
            className="flex flex-col items-start gap-2 rounded-[18px] bg-surface p-4 text-left transition active:scale-[0.98]"
          >
            <span className="flex size-12 items-center justify-center rounded-2xl bg-accent-soft text-accent">
              <Icon name={game.icon} size={26} />
            </span>
            <span className="text-body-strong text-ink">{game.title}</span>
            <span className="text-caption text-ink-2">{t(game.tagline)}</span>
          </button>
        ))}
      </div>

      <div className="flex items-start gap-2 px-1 pt-4 text-caption text-ink-2">
        <Icon name="info" size={16} className="pt-0.5" />
        <span>{t('gameHint')}</span>
      </div>
    </Screen>
  );
}
