import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  formatCooldown,
  newRequestId,
  secondsLeft,
  useAuraAnswer,
  useAuraClass,
  useAuraEquip,
  useAuraFarm,
  useAuraStart,
  useAuraState,
  useAuraStat,
  type AuraRelic,
  type AuraSession,
} from '../api';
import {
  AppHeader,
  Banner,
  Card,
  Chip,
  EmptyState,
  Icon,
  PrimaryButton,
  Screen,
  Skeleton,
} from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useSession } from '@/app/session';
import { useT } from '@/shared/i18n';

const STAT_LABELS: Record<string, string> = {
  str: 'Сила',
  agi: 'Ловкость',
  int: 'Интеллект',
  vit: 'Живучесть',
  per: 'Восприятие',
};

const RANK_TONE: Record<string, 'success' | 'warning'> = {
  E: 'warning',
  D: 'warning',
  C: 'warning',
  B: 'success',
  A: 'success',
  S: 'success',
  SS: 'success',
  SSS: 'success',
};

/**
 * SruAura: the math-driven RPG.
 *
 * The screen is one loop: farm aura on a cooldown, spend it on challenges, and
 * answer questions to win. The question is always rendered from server data and
 * the answer is checked on the server, so the client cannot shortcut a run.
 */
export function AuraPage() {
  const t = useT();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const { me } = useSession();
  const state = useAuraState(Boolean(me));
  const farm = useAuraFarm();
  const start = useAuraStart();
  const answer = useAuraAnswer();
  const chooseClass = useAuraClass();
  const equip = useAuraEquip();
  const spendStat = useAuraStat();

  const [requestId, setRequestId] = useState(newRequestId);
  const [tick, setTick] = useState(Date.now());
  const [showRelics, setShowRelics] = useState(false);

  useEffect(() => {
    const timer = window.setInterval(() => setTick(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  const player = state.data?.player;
  const session: AuraSession | null = player?.session ?? null;

  // A new question means a new attempt id, so a retry of the previous answer
  // cannot be mistaken for this one.
  useEffect(() => {
    setRequestId(newRequestId());
  }, [session?.question.text]);

  const pick = (choice: number) => {
    if (answer.isPending) return;
    answer.mutate(
      { choice, requestId },
      {
        onSuccess: (result) => {
          if (result.finished) {
            const relic = result.relic ? ` ${t('auraRelic')}: ${result.relic.name}` : '';
            feedback(`${result.message}${relic}`);
          } else if (!result.correct) {
            feedback(result.hint ?? t('auraWrong'), 'error');
          }
        },
        onError: () => feedback(t('error'), 'error'),
      },
    );
  };

  const begin = (kind: string) => {
    start.mutate(kind, {
      onError: () => feedback(t('auraNotReady'), 'error'),
    });
  };

  if (state.isLoading) {
    return (
      <Screen header={<AppHeader title="SruAura" onBack={() => navigate('/apps/game')} leadingIcon="auto_awesome" />}>
        <Skeleton className="mt-3 h-40 w-full" />
      </Screen>
    );
  }

  if (state.isError || !player) {
    return (
      <Screen header={<AppHeader title="SruAura" onBack={() => navigate('/apps/game')} leadingIcon="auto_awesome" />}>
        <Banner tone="danger" icon="error" title={t('error')}>
          {t('retryConnection')}
        </Banner>
      </Screen>
    );
  }

  // Reading tick keeps the countdown live: the interval above re-renders once a
  // second, so the remaining time actually decreases.
  const now = tick;
  const farmLeft = formatCooldown(secondsLeft(player.farm_ready_at, now));
  const arenaLeft = formatCooldown(secondsLeft(player.arena_ready_at, now));
  const gateLeft = formatCooldown(secondsLeft(player.gate_ready_at, now));
  const towerLeft = formatCooldown(secondsLeft(player.tower_ready_at, now));

  return (
    <Screen header={<AppHeader title="SruAura" onBack={() => navigate('/apps/game')} leadingIcon="auto_awesome" />}>
      <Card className="mt-2 p-4">
        <div className="flex items-center gap-3">
          <span className="flex size-14 items-center justify-center rounded-2xl bg-accent-soft text-accent">
            <Icon name="auto_awesome" size={28} />
          </span>
          <div className="flex min-w-0 flex-1 flex-col">
            <span className="text-title-3 text-ink">
              {player.class_name || t('auraNoClass')}
            </span>
            <span className="text-caption text-ink-2">
              {t('auraLevel')} {player.level} · {player.xp} / {player.xp_to_next} XP
            </span>
          </div>
          <Chip tone="success" icon="bolt">
            {player.aura}
          </Chip>
        </div>

        {player.ability ? (
          <div className="mt-3 flex items-start gap-2 rounded-xl bg-neutral-soft/60 p-3">
            <Icon name={player.ability.icon} size={18} className="pt-0.5 text-accent" />
            <div className="flex flex-col">
              <span className="text-body-secondary-strong text-ink">{player.ability.name}</span>
              <span className="text-caption text-ink-2">{player.ability.text}</span>
            </div>
          </div>
        ) : null}

        {player.stat_points > 0 ? (
          <div className="mt-3">
            <div className="pb-1 text-caption text-ink-2">
              {t('auraPoints')}: {player.stat_points}
            </div>
            <div className="flex flex-wrap gap-1.5">
              {Object.entries(STAT_LABELS).map(([key, label]) => (
                <button
                  key={key}
                  type="button"
                  disabled={spendStat.isPending}
                  onClick={() => spendStat.mutate(key)}
                  className="h-8 rounded-full bg-neutral-soft px-3 text-caption text-ink-2"
                >
                  {label} {player.stats[key as keyof typeof player.stats]}
                </button>
              ))}
            </div>
          </div>
        ) : null}

        <div className="pt-3">
          <PrimaryButton onClick={() => farm.mutate()} disabled={Boolean(farmLeft)} progress={farm.isPending}>
            {farmLeft ? `${t('auraFarmIn')} ${farmLeft}` : t('auraFarm')}
          </PrimaryButton>
        </div>
      </Card>

      {session ? (
        <Card className="mt-3 p-4">
          <div className="flex items-center justify-between pb-2">
            <span className="text-body-secondary text-ink-2">
              {session.kind === 'arena' ? t('auraArena') : session.kind === 'gate' ? t('auraGate') : session.kind === 'tower' ? t('auraTower') : t('auraGrind')}
            </span>
            {session.boss_max_hp > 0 ? (
              <span className="text-caption text-ink-2">
                {session.boss_name} · {session.boss_hp} / {session.boss_max_hp}
              </span>
            ) : null}
          </div>

          {session.boss_max_hp > 0 ? (
            <div className="h-2 overflow-hidden rounded-full bg-neutral-soft">
              <div
                className="h-full rounded-full bg-accent"
                style={{ width: `${Math.max(0, (session.boss_hp / session.boss_max_hp) * 100)}%` }}
              />
            </div>
          ) : null}

          <div className="flex items-center gap-2 pt-3 text-caption text-ink-2">
            <Chip tone="success" icon="local_fire_department">
              {t('auraCombo')} {session.combo}
            </Chip>
            <span>{t('auraRound')} {session.round} / {session.rounds}</span>
          </div>

          <p className="pt-3 text-title-3 text-ink">{session.question.text}</p>
          <div className="grid grid-cols-2 gap-2 pt-3">
            {session.question.options.map((option) => (
              <button
                key={option}
                type="button"
                disabled={answer.isPending}
                onClick={() => pick(option)}
                className="h-12 rounded-xl bg-neutral-soft text-body-strong text-ink transition active:scale-[0.98]"
              >
                {option}
              </button>
            ))}
          </div>
          <div className="pt-2 text-center text-caption text-hint">{session.question.topic_name}</div>
        </Card>
      ) : (
        <div className="flex flex-col gap-2 pt-3">
          <button
            type="button"
            disabled={start.isPending || Boolean(arenaLeft)}
            onClick={() => begin('arena')}
            className="flex items-center justify-between rounded-[18px] bg-surface p-4 text-left disabled:opacity-60"
          >
            <span className="flex flex-col">
              <span className="text-body-strong text-ink">{t('auraArena')}</span>
              <span className="text-caption text-ink-2">{t('auraArenaHint')}</span>
            </span>
            <span className="text-caption text-accent">{arenaLeft || '→'}</span>
          </button>

          <button
            type="button"
            disabled={start.isPending || Boolean(gateLeft)}
            onClick={() => begin('gate')}
            className="flex items-center justify-between rounded-[18px] bg-surface p-4 text-left disabled:opacity-60"
          >
            <span className="flex flex-col">
              <span className="text-body-strong text-ink">{t('auraGate')}</span>
              <span className="text-caption text-ink-2">{t('auraGateHint')}</span>
            </span>
            <span className="text-caption text-accent">{gateLeft || '→'}</span>
          </button>

          <button
            type="button"
            disabled={start.isPending || Boolean(towerLeft)}
            onClick={() => begin('tower')}
            className="flex items-center justify-between rounded-[18px] bg-surface p-4 text-left disabled:opacity-60"
          >
            <span className="flex flex-col">
              <span className="text-body-strong text-ink">{t('auraTower')}</span>
              <span className="text-caption text-ink-2">{t('auraTowerHint')}</span>
            </span>
            <span className="text-caption text-accent">{towerLeft || '→'}</span>
          </button>

          <button
            type="button"
            disabled={start.isPending || player.grind_today >= player.grind_cap}
            onClick={() => begin('grind')}
            className="flex items-center justify-between rounded-[18px] bg-surface p-4 text-left disabled:opacity-60"
          >
            <span className="flex flex-col">
              <span className="text-body-strong text-ink">{t('auraGrind')}</span>
              <span className="text-caption text-ink-2">
                {t('auraGrindHint')} · {player.grind_today} / {player.grind_cap}
              </span>
            </span>
            <span className="text-caption text-accent">→</span>
          </button>
        </div>
      )}

      {!player.hero_class ? (
        <Card className="mt-3 p-4">
          <div className="pb-2 text-body-secondary-strong text-ink">{t('auraChooseClass')}</div>
          <div className="flex flex-col gap-1.5">
            {state.data?.classes.map((entry) => (
              <button
                key={entry.id}
                type="button"
                disabled={chooseClass.isPending}
                onClick={() => chooseClass.mutate(entry.id)}
                className="rounded-xl bg-neutral-soft px-3 py-2 text-left"
              >
                <span className="block text-body-secondary-strong text-ink">{entry.name}</span>
                <span className="block text-caption text-ink-2">{entry.description}</span>
              </button>
            ))}
          </div>
        </Card>
      ) : null}

      <div className="pt-3">
        <button
          type="button"
          onClick={() => setShowRelics((value) => !value)}
          className="flex items-center gap-1 text-body-secondary text-accent"
        >
          <Icon name={showRelics ? 'expand_less' : 'expand_more'} size={18} />
          {t('auraRelics')} {state.data?.relics.length ?? 0}
        </button>

        {showRelics ? (
          <div className="flex flex-col gap-1.5 pt-2">
            {state.data && state.data.relics.length === 0 ? (
              <EmptyState icon="auto_awesome" title={t('auraNoRelics')} text={t('auraNoRelicsHint')} />
            ) : null}
            {state.data?.relics.map((relic: AuraRelic) => (
              <div key={relic.key} className="flex items-center gap-2 rounded-xl bg-neutral-soft/60 px-3 py-2">
                <Chip tone={RANK_TONE[relic.rank] ?? 'warning'}>{relic.rank}</Chip>
                <div className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate text-body-secondary-strong text-ink">{relic.name}</span>
                  <span className="text-caption text-ink-2">
                    {relic.slot} · {t('auraRelicLevel')} {relic.level}
                  </span>
                </div>
                <button
                  type="button"
                  disabled={equip.isPending}
                  onClick={() => equip.mutate({ key: relic.key, slot: relic.worn ? '' : relic.slot })}
                  className="h-8 rounded-lg bg-accent-soft px-3 text-caption font-semibold text-accent"
                >
                  {relic.worn ? t('auraUnequip') : t('auraEquip')}
                </button>
              </div>
            ))}
          </div>
        ) : null}
      </div>

      <div className="px-1 pt-4 text-caption text-ink-2">{t('auraNote')}</div>
    </Screen>
  );
}
