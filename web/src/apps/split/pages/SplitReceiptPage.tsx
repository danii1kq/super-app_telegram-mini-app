import { useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  formatMinor,
  parseMinor,
  useSplitGroups,
} from '../api';
import { newRequestId } from '../mutations';
import {
  useAllocation,
  useSaveReceipt,
  useUpdateReceipt,
  useUploadReceipt,
} from '../receipts';
import {
  AppHeader,
  Banner,
  Card,
  Chip,
  Icon,
  PrimaryButton,
  Row,
  Screen,
  SecondaryButton,
  SectionLabel,
  Skeleton,
} from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useSession } from '@/app/session';
import { usePeople } from '@/core/social/api';
import { useT } from '@/shared/i18n';

/** One editable line of the draft. */
interface DraftItem {
  key: string;
  name: string;
  quantity: number;
  amount: string;
  unclear: boolean;
  units: Record<string, number>;
}

let draftKey = 0;
function nextKey(): string {
  draftKey += 1;
  return `item-${draftKey}`;
}

/**
 * Receipt flow: photo, check, who took what, total, save.
 *
 * The recognizer only fills a draft. Every value is editable, unclear lines are
 * marked instead of guessed, and the split is computed by the server.
 */
export function SplitReceiptPage() {
  const { groupId = '' } = useParams();
  const t = useT();
  const navigate = useNavigate();
  const feedback = useFeedback();
  const { me } = useSession();
  const groups = useSplitGroups();
  const people = usePeople(Boolean(groupId));
  const upload = useUploadReceipt(groupId);

  const [receiptId, setReceiptId] = useState('');
  const [items, setItems] = useState<DraftItem[]>([]);
  const [total, setTotal] = useState('');
  const [discount, setDiscount] = useState('');
  const [service, setService] = useState('');
  const [payerId, setPayerId] = useState('');
  const [ocrError, setOcrError] = useState<string>('');
  const [manual, setManual] = useState(false);
  const [saveKey, setSaveKey] = useState(newRequestId);
  const fileInput = useRef<HTMLInputElement | null>(null);

  const updateReceipt = useUpdateReceipt(receiptId);
  const saveReceipt = useSaveReceipt(receiptId, groupId);
  const allocation = useAllocation(receiptId, payerId);

  const group = groups.data?.find((candidate) => candidate.id === groupId);
  const members = useMemo(() => people.data ?? [], [people.data]);

  const nameOf = (userId: string): string => {
    if (userId === me?.id) return t('you');
    return members.find((person) => person.id === userId)?.public_name ?? `${t('member')} ${userId}`;
  };

  const applyReceipt = (receipt: {
    id: string;
    total_minor: string;
    discount_minor: string;
    service_minor: string;
    ocr_error?: string;
    items: { name: string; quantity: number; amount_minor: string; unclear: boolean; units: Record<string, number> }[];
  }) => {
    setReceiptId(receipt.id);
    setTotal(receipt.total_minor === '0' ? '' : formatMinor(receipt.total_minor));
    setDiscount(receipt.discount_minor === '0' ? '' : formatMinor(receipt.discount_minor));
    setService(receipt.service_minor === '0' ? '' : formatMinor(receipt.service_minor));
    setOcrError(receipt.ocr_error ?? '');
    setItems(
      receipt.items.map((item) => ({
        key: nextKey(),
        name: item.name,
        quantity: item.quantity,
        amount: formatMinor(item.amount_minor),
        unclear: item.unclear,
        units: item.units,
      })),
    );
    if (me) setPayerId(me.id);
  };

  const onPickFile = (file: File | undefined) => {
    if (!file) return;
    upload.mutate(file, {
      onSuccess: (receipt) => {
        applyReceipt(receipt);
        if (receipt.ocr_error) {
          setManual(true);
          feedback(t('receiptOcrFailed'), 'error');
        }
      },
      onError: () => feedback(t('receiptUploadFailed'), 'error'),
    });
  };

  const startManual = () => {
    setManual(true);
    setItems([{ key: nextKey(), name: '', quantity: 1, amount: '', unclear: false, units: {} }]);
    if (me) setPayerId(me.id);
  };

  const patchItem = (key: string, patch: Partial<DraftItem>) => {
    setItems((current) => current.map((item) => (item.key === key ? { ...item, ...patch } : item)));
  };

  const toggleUnit = (key: string, userId: string) => {
    setItems((current) =>
      current.map((item) => {
        if (item.key !== key) return item;
        const units = { ...item.units };
        if (units[userId]) {
          delete units[userId];
        } else {
          units[userId] = 1;
        }
        return { ...item, units };
      }),
    );
  };

  const setUnits = (key: string, userId: string, count: number) => {
    setItems((current) =>
      current.map((item) => {
        if (item.key !== key) return item;
        const units = { ...item.units };
        if (count <= 0) {
          delete units[userId];
        } else {
          units[userId] = Math.min(count, item.quantity);
        }
        return { ...item, units };
      }),
    );
  };

  const takeAll = (key: string) => {
    setItems((current) =>
      current.map((item) => {
        if (item.key !== key) return item;
        const units: Record<string, number> = {};
        for (const person of members) units[person.id] = 1;
        return { ...item, units };
      }),
    );
  };

  const itemsTotal = items.reduce((sum, item) => sum + (parseMinor(item.amount) ?? 0), 0);
  const totalMinor = parseMinor(total) ?? itemsTotal;

  const persist = (): boolean => {
    if (!receiptId) return true;
    const payload = {
      totalMinor: String(totalMinor),
      discountMinor: String(parseMinor(discount) ?? 0),
      serviceMinor: String(parseMinor(service) ?? 0),
      items: items.map((item) => ({
        name: item.name.trim(),
        quantity: item.quantity,
        amountMinor: String(parseMinor(item.amount) ?? 0),
        unclear: item.unclear,
        units: item.units,
      })),
    };
    updateReceipt.mutate(payload);
    return true;
  };

  const save = () => {
    if (!payerId) {
      feedback(t('receiptPickPayer'), 'error');
      return;
    }
    const unassigned = items.some((item) => Object.keys(item.units).length === 0);
    if (unassigned) {
      feedback(t('receiptUnassigned'), 'error');
      return;
    }
    if (!receiptId) {
      feedback(t('receiptNeedPhoto'), 'error');
      return;
    }
    updateReceipt.mutate(
      {
        totalMinor: String(totalMinor),
        discountMinor: String(parseMinor(discount) ?? 0),
        serviceMinor: String(parseMinor(service) ?? 0),
        items: items.map((item) => ({
          name: item.name.trim(),
          quantity: item.quantity,
          amountMinor: String(parseMinor(item.amount) ?? 0),
          unclear: item.unclear,
          units: item.units,
        })),
      },
      {
        onSuccess: () => {
          saveReceipt.mutate(
            { payerId, clientRequestId: saveKey, description: group?.title ?? t('receiptTitle') },
            {
              onSuccess: () => {
                setSaveKey(newRequestId());
                feedback(t('receiptSaved'));
                navigate(`/apps/split/${groupId}`);
              },
              onError: () => feedback(t('receiptSaveFailed'), 'error'),
            },
          );
        },
        onError: () => feedback(t('receiptSaveFailed'), 'error'),
      },
    );
  };

  const difference = allocation.data ? Number(allocation.data.difference) : 0;

  return (
    <Screen
      header={
        <AppHeader title={t('receiptTitle')} onBack={() => navigate(`/apps/split/${groupId}`)} />
      }
      bottomSpace
    >
      {!receiptId && !manual ? (
        <Card className="mt-2 p-4">
          <div className="flex flex-col items-center gap-2 py-4 text-center">
            <Icon name="photo_camera" size={44} className="text-accent" />
            <span className="text-headline-sm text-ink">{t('receiptAdd')}</span>
            <span className="text-caption text-ink-2">{t('receiptAddHint')}</span>
          </div>
          <input
            ref={fileInput}
            type="file"
            accept="image/jpeg,image/png,image/webp"
            className="hidden"
            onChange={(event) => onPickFile(event.target.files?.[0])}
          />
          <div className="flex flex-col gap-2 pt-2">
            <PrimaryButton onClick={() => fileInput.current?.click()} full progress={upload.isPending}>
              {upload.isPending ? t('receiptRecognizing') : t('receiptTakePhoto')}
            </PrimaryButton>
            <SecondaryButton onClick={startManual} full>
              {t('receiptManual')}
            </SecondaryButton>
          </div>
        </Card>
      ) : null}

      {ocrError ? (
        <div className="pt-2">
          <Banner tone="warning" icon="error" title={t('receiptOcrFailed')}>
            {ocrError === 'unavailable' ? t('receiptOcrUnavailable') : t('receiptOcrUnreadable')}
          </Banner>
        </div>
      ) : null}

      {items.length > 0 ? (
        <>
          <SectionLabel>{t('receiptItems')}</SectionLabel>
          <div className="flex flex-col gap-2">
            {items.map((item) => (
              <Card key={item.key} className="p-3">
                <div className="flex items-start gap-2">
                  <input
                    value={item.name}
                    onChange={(event) => patchItem(item.key, { name: event.target.value })}
                    placeholder={t('receiptItemName')}
                    className={`min-w-0 flex-1 rounded-lg px-3 py-2 text-body-strong text-ink outline-none placeholder:text-hint ${
                      item.unclear ? 'bg-warning-soft' : 'bg-neutral-soft/60'
                    }`}
                  />
                  <button
                    type="button"
                    aria-label={t('delete')}
                    onClick={() => setItems((current) => current.filter((entry) => entry.key !== item.key))}
                    className="flex size-9 shrink-0 items-center justify-center text-danger"
                  >
                    <Icon name="delete" size={20} />
                  </button>
                </div>

                <div className="flex items-center gap-2 pt-2">
                  <input
                    type="number"
                    min={1}
                    value={item.quantity}
                    onChange={(event) =>
                      patchItem(item.key, { quantity: Math.max(1, Number(event.target.value) || 1) })
                    }
                    className="w-16 rounded-lg bg-neutral-soft/60 px-2 py-2 text-center text-body-lg text-ink outline-none"
                  />
                  <input
                    value={item.amount}
                    onChange={(event) => patchItem(item.key, { amount: event.target.value })}
                    placeholder={t('expenseAmount')}
                    inputMode="decimal"
                    className={`min-w-0 flex-1 rounded-lg px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint ${
                      item.unclear ? 'bg-warning-soft' : 'bg-neutral-soft/60'
                    }`}
                  />
                </div>

                {item.unclear ? (
                  <div className="flex items-center gap-1 pt-1 text-caption text-warning">
                    <Icon name="help" size={14} />
                    {t('receiptUnclear')}
                  </div>
                ) : null}

                <div className="pt-2 text-caption text-ink-2">{t('receiptWho')}</div>
                <div className="flex flex-wrap gap-1.5 pt-1">
                  <button
                    type="button"
                    onClick={() => takeAll(item.key)}
                    className="h-8 rounded-full bg-accent-soft px-3 text-caption font-semibold text-accent"
                  >
                    {t('receiptEveryone')}
                  </button>
                  {members.map((person) => {
                    const units = item.units[person.id] ?? 0;
                    return (
                      <span key={person.id} className="flex items-center gap-1">
                        <button
                          type="button"
                          onClick={() => toggleUnit(item.key, person.id)}
                          className={
                            units > 0
                              ? 'h-8 rounded-full bg-accent-soft px-3 text-caption font-semibold text-accent'
                              : 'h-8 rounded-full bg-neutral-soft px-3 text-caption text-ink-2'
                          }
                        >
                          {person.public_name}
                        </button>
                        {units > 0 && item.quantity > 1 ? (
                          <span className="flex items-center gap-0.5">
                            <button
                              type="button"
                              aria-label="minus"
                              onClick={() => setUnits(item.key, person.id, units - 1)}
                              className="flex size-6 items-center justify-center rounded-full bg-neutral-soft text-ink-2"
                            >
                              <Icon name="remove" size={14} />
                            </button>
                            <span className="min-w-4 text-center text-caption text-ink">{units}</span>
                            <button
                              type="button"
                              aria-label="plus"
                              onClick={() => setUnits(item.key, person.id, units + 1)}
                              className="flex size-6 items-center justify-center rounded-full bg-neutral-soft text-ink-2"
                            >
                              <Icon name="add" size={14} />
                            </button>
                          </span>
                        ) : null}
                      </span>
                    );
                  })}
                </div>
              </Card>
            ))}
          </div>

          <button
            type="button"
            onClick={() =>
              setItems((current) => [
                ...current,
                { key: nextKey(), name: '', quantity: 1, amount: '', unclear: false, units: {} },
              ])
            }
            className="mt-2 flex w-full items-center gap-2 rounded-[14px] bg-surface px-4 py-3 text-left text-body-lg text-hint"
          >
            <Icon name="add_circle" size={20} className="text-accent" />
            {t('receiptAddItem')}
          </button>

          <SectionLabel>{t('receiptTotals')}</SectionLabel>
          <Card className="p-3">
            <label className="text-caption text-ink-2" htmlFor="receipt-total">
              {t('receiptTotal')}
            </label>
            <input
              id="receipt-total"
              value={total}
              onChange={(event) => setTotal(event.target.value)}
              placeholder={formatMinor(String(itemsTotal))}
              inputMode="decimal"
              className="mt-1 w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none placeholder:text-hint"
            />
            <div className="flex gap-2 pt-2">
              <div className="flex-1">
                <label className="text-caption text-ink-2" htmlFor="receipt-discount">
                  {t('receiptDiscount')}
                </label>
                <input
                  id="receipt-discount"
                  value={discount}
                  onChange={(event) => setDiscount(event.target.value)}
                  inputMode="decimal"
                  className="mt-1 w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none"
                />
              </div>
              <div className="flex-1">
                <label className="text-caption text-ink-2" htmlFor="receipt-service">
                  {t('receiptService')}
                </label>
                <input
                  id="receipt-service"
                  value={service}
                  onChange={(event) => setService(event.target.value)}
                  inputMode="decimal"
                  className="mt-1 w-full rounded-lg bg-neutral-soft/60 px-3 py-2 text-body-lg text-ink outline-none"
                />
              </div>
            </div>
          </Card>

          <SectionLabel>{t('receiptPayer')}</SectionLabel>
          <Card>
            {members.map((person, index) => (
              <div key={person.id}>
                {index > 0 ? <div className="hairline" /> : null}
                <Row
                  title={person.public_name}
                  onClick={() => {
                    setPayerId(person.id);
                    persist();
                  }}
                  trailing={
                    payerId === person.id ? (
                      <Icon name="check_circle" size={22} className="text-accent" filled />
                    ) : undefined
                  }
                />
              </div>
            ))}
          </Card>

          {allocation.isLoading ? <Skeleton className="mt-3 h-24 w-full" /> : null}

          {allocation.data ? (
            <>
              <SectionLabel>{t('receiptResult')}</SectionLabel>
              <Card className="p-4">
                <div className="flex items-center justify-between">
                  <span className="text-body-secondary text-ink-2">{t('receiptTotal')}</span>
                  <span className="text-body-strong text-ink">
                    {formatMinor(allocation.data.total_minor)} ₸
                  </span>
                </div>
                {Number(allocation.data.discount_minor) > 0 ? (
                  <div className="flex items-center justify-between pt-1">
                    <span className="text-body-secondary text-ink-2">{t('receiptDiscount')}</span>
                    <span className="text-body-secondary text-success">
                      -{formatMinor(allocation.data.discount_minor)} ₸
                    </span>
                  </div>
                ) : null}
                {Number(allocation.data.service_minor) > 0 ? (
                  <div className="flex items-center justify-between pt-1">
                    <span className="text-body-secondary text-ink-2">{t('receiptService')}</span>
                    <span className="text-body-secondary text-warning">
                      +{formatMinor(allocation.data.service_minor)} ₸
                    </span>
                  </div>
                ) : null}

                <div className="pt-3 text-caption text-ink-2">{t('receiptShares')}</div>
                {Object.entries(allocation.data.per_person).map(([userId, share]) => (
                  <div key={userId} className="flex items-center justify-between pt-1">
                    <span className="text-body-secondary text-ink">{nameOf(userId)}</span>
                    <span className="text-body-secondary text-ink">{formatMinor(share)} ₸</span>
                  </div>
                ))}

                <div className="pt-3 text-caption text-ink-2">{t('receiptOwes')}</div>
                {Object.entries(allocation.data.owes_to_payer).length === 0 ? (
                  <div className="pt-1 text-caption text-ink-2">{t('receiptNobodyOwes')}</div>
                ) : null}
                {Object.entries(allocation.data.owes_to_payer).map(([userId, share]) => (
                  <div key={userId} className="flex items-center justify-between pt-1">
                    <span className="text-body-secondary text-ink">
                      {nameOf(userId)} → {nameOf(payerId)}
                    </span>
                    <Chip tone="warning">{formatMinor(share)} ₸</Chip>
                  </div>
                ))}

                {difference !== 0 ? (
                  <div className="pt-3">
                    <Banner tone="danger" icon="error" title={t('receiptMismatch')}>
                      {t('receiptDifference')}: {formatMinor(String(difference))} ₸
                    </Banner>
                  </div>
                ) : null}
              </Card>
            </>
          ) : null}

          <div className="pt-3">
            <PrimaryButton
              onClick={save}
              full
              progress={saveReceipt.isPending || updateReceipt.isPending}
              disabled={difference !== 0 || !payerId}
            >
              {t('receiptSave')}
            </PrimaryButton>
          </div>
        </>
      ) : null}
    </Screen>
  );
}
