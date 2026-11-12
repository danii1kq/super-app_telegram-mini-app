import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { formatMinor } from '../api';
import { monthKey, monthLabel, shiftMonth, useReport } from '../report';
import {
  AppHeader,
  Banner,
  Card,
  Chip,
  EmptyState,
  Hairline,
  Icon,
  Row,
  Screen,
  SectionLabel,
  Skeleton,
} from '@/shared/ui';
import { useI18n, useT, type DictKey } from '@/shared/i18n';
import { formatDayMonth } from '@/shared/i18n/date';
import { localeOf } from '@/shared/i18n/date';

/** Category labels, in the order the report shows them. */
const CATEGORY_ORDER = ['groceries', 'cafe', 'transport', 'home', 'fun', 'other'] as const;

const CATEGORY_KEY: Record<string, DictKey> = {
  groceries: 'catGroceries',
  cafe: 'catCafe',
  transport: 'catTransport',
  home: 'catHome',
  fun: 'catFun',
  other: 'catOther',
};

/**
 * The month report: what this person actually spent.
 *
 * "Мои траты" is the sum of their own shares, not the receipts they paid for.
 * Loans and repayments are shown separately and never added to it.
 */
export function SplitReportPage() {
  const t = useT();
  const { lang } = useI18n();
  const navigate = useNavigate();
  const [month, setMonth] = useState(() => monthKey(new Date()));
  const report = useReport(month);

  const data = report.data;
  const total = data ? Number(data.my_spending) : 0;

  return (
    <Screen
      header={<AppHeader title={t('reportTitle')} onBack={() => navigate('/apps/split')} />}
    >
      <div className="flex items-center justify-between pt-2">
        <button
          type="button"
          onClick={() => setMonth((current) => shiftMonth(current, -1))}
          aria-label={t('reportPrevMonth')}
          className="flex size-10 items-center justify-center rounded-xl bg-neutral-soft text-ink"
        >
          <Icon name="chevron_left" size={22} />
        </button>
        <span className="text-body-strong text-ink">{monthLabel(month, localeOf(lang))}</span>
        <button
          type="button"
          onClick={() => setMonth((current) => shiftMonth(current, 1))}
          aria-label={t('reportNextMonth')}
          className="flex size-10 items-center justify-center rounded-xl bg-neutral-soft text-ink"
        >
          <Icon name="chevron_right" size={22} />
        </button>
      </div>

      <div className="px-1 pt-2 text-caption text-ink-2">{t('reportScope')}</div>

      {report.isLoading ? (
        <div className="flex flex-col gap-2 pt-3">
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-32 w-full" />
        </div>
      ) : null}

      {report.isError ? (
        <div className="pt-3">
          <Banner tone="danger" icon="error" title={t('error')}>
            {t('retryConnection')}
          </Banner>
        </div>
      ) : null}

      {data ? (
        <>
          <SectionLabel>{t('reportMine')}</SectionLabel>
          <Card className="p-4">
            <div className="text-caption text-ink-2">{t('reportMySpending')}</div>
            <div className="pt-1 text-headline-xl text-ink">{formatMinor(data.my_spending)} ₸</div>
            <div className="flex items-center justify-between pt-3">
              <span className="text-body-secondary text-ink-2">{t('reportPaidByMe')}</span>
              <span className="text-body-secondary text-ink">{formatMinor(data.paid_by_me)} ₸</span>
            </div>
            <div className="px-0 pt-1 text-caption text-ink-2">{t('reportPaidByMeHint')}</div>
          </Card>

          {data.lines.length === 0 ? (
            <EmptyState icon="receipt_long" title={t('reportEmpty')} text={t('reportEmptyText')} />
          ) : null}

          {data.lines.length > 0 ? (
            <>
              <SectionLabel>{t('reportCategories')}</SectionLabel>
              <Card className="p-4">
                {CATEGORY_ORDER.filter((name) => Number(data.by_category[name] ?? 0) > 0).map((name) => {
                  const value = Number(data.by_category[name] ?? 0);
                  const share = total > 0 ? Math.round((value / total) * 100) : 0;
                  return (
                    <div key={name} className="pt-2 first:pt-0">
                      <div className="flex items-center justify-between">
                        <span className="text-body-secondary text-ink">
                          {CATEGORY_KEY[name] ? t(CATEGORY_KEY[name]) : name}
                        </span>
                        <span className="text-body-secondary text-ink">{formatMinor(String(value))} ₸</span>
                      </div>
                      <div className="mt-1 h-2 w-full overflow-hidden rounded-full bg-neutral-soft">
                        <div
                          className="h-full rounded-full bg-accent"
                          style={{ width: `${Math.max(share, 2)}%` }}
                        />
                      </div>
                    </div>
                  );
                })}
              </Card>

              <SectionLabel>{t('reportLines')}</SectionLabel>
              <Card>
                {data.lines.map((line, index) => (
                  <div key={line.expense_id}>
                    {index > 0 ? <Hairline /> : null}
                    <Row
                      icon="receipt_long"
                      title={line.description || t('docUntitled')}
                      subtitle={`${formatDayMonth(line.spent_at, lang)} · ${line.group_title}`}
                      onClick={() => navigate(`/apps/split/${line.group_id}`)}
                      trailing={
                        <span className="text-body-secondary text-ink">
                          {formatMinor(line.my_share)} ₸
                        </span>
                      }
                    />
                  </div>
                ))}
              </Card>
            </>
          ) : null}

          <SectionLabel>{t('reportDebts')}</SectionLabel>
          <Card className="p-4">
            <div className="flex items-center justify-between">
              <span className="text-body-secondary text-ink-2">{t('reportLentOut')}</span>
              <span className="text-body-secondary text-ink">{formatMinor(data.lent_out)} ₸</span>
            </div>
            <div className="flex items-center justify-between pt-1">
              <span className="text-body-secondary text-ink-2">{t('reportReturnedToMe')}</span>
              <span className="text-body-secondary text-success">
                {formatMinor(data.returned_to_me)} ₸
              </span>
            </div>
            <div className="flex items-center justify-between pt-1">
              <span className="text-body-secondary text-ink-2">{t('reportReturnedByMe')}</span>
              <span className="text-body-secondary text-ink-2">{formatMinor(data.returned_by_me)} ₸</span>
            </div>
            <div className="px-0 pt-2 text-caption text-ink-2">{t('reportDebtsHint')}</div>
          </Card>

          <SectionLabel>{t('reportNow')}</SectionLabel>
          <Card>
            <Row
              icon="south_west"
              iconTone="success"
              title={t('reportOwedToMe')}
              value={`${formatMinor(data.owed_to_me_now)} ₸`}
            />
            <Hairline />
            <Row
              icon="north_east"
              iconTone="danger"
              title={t('reportIOwe')}
              value={`${formatMinor(data.i_owe_now)} ₸`}
            />
          </Card>
          <div className="px-1 pt-2 text-caption text-ink-2">{t('reportNowHint')}</div>

          <div className="flex items-center gap-1 px-1 pt-3 text-caption text-ink-2">
            <Icon name="info" size={14} />
            <Chip tone="neutral">{t('reportOnlySplit')}</Chip>
          </div>
        </>
      ) : null}
    </Screen>
  );
}
