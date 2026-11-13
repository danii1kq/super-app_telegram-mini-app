import { useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  formatMinor,
  useSplitGroups,
} from '../api';
import { newRequestId } from '../mutations';
import { receiptPreview, parseReceiptMinor, MAX_RECEIPT_MINOR, MAX_RECEIPT_ITEMS } from '../receiptPreview';
import { ApiError } from '@/shared/api/client';
import { ReceiptPhotoEditor } from '../ReceiptPhotoEditor';
import {
  useGroupMembers,
  useManualReceipt,
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

} from '@/shared/ui';
import { useFeedback } from '@/shared/ui/toast';
import { useSession } from '@/app/session';
import { usePeople } from '@/core/social/api';
import { useT, type DictKey } from '@/shared/i18n';

/** One editable line of the draft. */
interface DraftItem {
  id?: string;
  key: string;
  name: string;
  quantity: number;
  amount: string;
  unclear: boolean;
  units: Record<string, number>;
}

/** The categories the receipt screen offers. */
const CATEGORIES = ['groceries', 'cafe', 'transport', 'home', 'fun', 'other'] as const;

const CATEGORY_KEY: Record<string, DictKey> = {
  groceries: 'catGroceries',
  cafe: 'catCafe',
  transport: 'catTransport',
  home: 'catHome',
  fun: 'catFun',
  other: 'catOther',
};

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
  const groupMembers = useGroupMembers(groupId);
  const people = usePeople(Boolean(groupId));
  const upload = useUploadReceipt(groupId);
  const manualReceipt = useManualReceipt(groupId);

  const [receiptId, setReceiptId] = useState('');
  const [version, setVersion] = useState(1);
  const [items, setItems] = useState<DraftItem[]>([]);
  const [total, setTotal] = useState('');
  const [category, setCategory] = useState('cafe');
  const [discount, setDiscount] = useState('');
  const [service, setService] = useState('');
  const [payerId, setPayerId] = useState('');
  const [ocrError, setOcrError] = useState<string>('');
  const [manual, setManual] = useState(false);
  const [saveKey, setSaveKey] = useState(newRequestId);
  const [selectedPhoto, setSelectedPhoto] = useState<File | null>(null);
  const [originalPhoto, setOriginalPhoto] = useState<File | null>(null);
  const [photoEditing, setPhotoEditing] = useState(false);
  const [photoUrl, setPhotoUrl] = useState('');
  const [consent, setConsent] = useState(false);
  const [reviewedSignature, setReviewedSignature] = useState('');
  const [uncertainSave, setUncertainSave] = useState(false);
  const uploading = useRef(false);
  const submitting = useRef(false);
  const saveAttempt = useRef<{ payerId: string; clientRequestId: string; description: string; category: string } | null>(null);
  const draftSignature = JSON.stringify([items, total, discount, service, payerId, category]);
  const reviewed = reviewedSignature === draftSignature;

  const fileInput = useRef<HTMLInputElement | null>(null);

  const updateReceipt = useUpdateReceipt(receiptId);
  const saveReceipt = useSaveReceipt(receiptId, groupId);
  const draftLocked = updateReceipt.isPending || saveReceipt.isPending || uncertainSave;

  useEffect(() => {
    if (!selectedPhoto) { setPhotoUrl(''); return; }
    const url = URL.createObjectURL(selectedPhoto); setPhotoUrl(url);
    return () => URL.revokeObjectURL(url);
  }, [selectedPhoto]);


  const group = groups.data?.find((candidate) => candidate.id === groupId);
  // Only the group's own members: a stranger must never appear in a picker.
  // The viewer is added from the session, because the people directory leaves
  // the viewer out on purpose and the payer is often the viewer.
  const members = useMemo(() => {
    const ids = new Set(groupMembers.data ?? []);
    const fromDirectory = (people.data ?? []).filter((person) => ids.has(person.id));
    if (me && ids.has(me.id) && !fromDirectory.some((person) => person.id === me.id)) {
      return [{ id: me.id, public_name: me.public_name, username: me.username, following: false, is_me: true }, ...fromDirectory];
    }
    return fromDirectory;
  }, [people.data, groupMembers.data, me]);

  const nameOf = (userId: string): string => {
    if (userId === me?.id) return t('you');
    return members.find((person) => person.id === userId)?.public_name ?? `${t('member')} ${userId}`;
  };

  const applyReceipt = (receipt: {
    id: string;
    version: number;
    total_minor: string;
    discount_minor: string;
    service_minor: string;
    ocr_error?: string;
    image_bytes?: number;
    items: { id: string; name: string; quantity: number; amount_minor: string; unclear: boolean; units: Record<string, number> }[];
  }) => {
    setReceiptId(receipt.id);
    setVersion(receipt.version);
    setTotal(receipt.total_minor === '0' ? '' : formatMinor(receipt.total_minor));
    setDiscount(receipt.discount_minor === '0' ? '' : formatMinor(receipt.discount_minor));
    setService(receipt.service_minor === '0' ? '' : formatMinor(receipt.service_minor));
    setOcrError(receipt.ocr_error ?? '');
    setItems(
      receipt.items.map((item) => ({
        key: nextKey(),
        id: item.id,
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
    if (!file || upload.isPending) return;
    if (!file.size || file.size > 20 * 1024 * 1024 || !['image/jpeg', 'image/png', 'image/webp'].includes(file.type)) {
      feedback(t('receiptImageLimit'), 'error'); return;
    }
    setOriginalPhoto(file); setSelectedPhoto(file); setPhotoEditing(false); setConsent(false);
  };

  const recognize = () => {
    if (!selectedPhoto || !consent || uploading.current) return;
    uploading.current = true;
    upload.mutate(selectedPhoto, {
      onSuccess: (receipt) => {
        applyReceipt(receipt);
        if (receipt.ocr_error || !receipt.items.length) {
          setManual(true);
          setItems([{ key: nextKey(), name: '', quantity: 1, amount: '', unclear: false, units: {} }]);
          feedback(t('receiptOcrFailed'), 'error');
        }
      },
      onError: (error) => feedback(error instanceof ApiError ? error.message : t('receiptUploadFailed'), 'error'),
      onSettled: () => { uploading.current = false; },
    });
  };

  const startManual = () => {
    // A hand-typed receipt is a real draft on the server, so it saves the same
    // way a recognized one does.
    manualReceipt.mutate(undefined, {
      onSuccess: (receipt) => {
        setManual(true);
        setReceiptId(receipt.id);
        setItems([{ key: nextKey(), name: '', quantity: 1, amount: '', unclear: false, units: {} }]);
        if (me) setPayerId(me.id);
      },
      onError: () => feedback(t('receiptSaveFailed'), 'error'),
    });
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

  const itemsTotal = items.reduce((sum, item) => sum + (parseReceiptMinor(item.amount) ?? 0), 0);
  const totalMinor = parseReceiptMinor(total) ?? itemsTotal;

  const preview = useMemo(() => receiptPreview(
    items.map((item) => ({ key: item.key, amount: parseReceiptMinor(item.amount) ?? 0, units: item.units })),
    parseReceiptMinor(discount) ?? 0, parseReceiptMinor(service) ?? 0, totalMinor, payerId,
  ), [items, discount, service, totalMinor, payerId]);
  const validDraft = preview.valid && items.length > 0 && items.length <= MAX_RECEIPT_ITEMS && items.every((item) => {
    const amount = parseReceiptMinor(item.amount);
    const units = Object.values(item.units);
    return units.reduce((sum, value) => sum + value, 0) <= Math.max(item.quantity, units.length) && item.name.trim() && item.name.trim().length <= 200 && Number.isInteger(item.quantity) && item.quantity >= 1 && item.quantity <= 999 && amount !== null && Number.isSafeInteger(amount) && amount <= MAX_RECEIPT_MINOR;
  }) && [total, discount, service].every((value) => {
    if (!value.trim() || /^0([.,]0{1,2})?$/.test(value.trim())) return true;
    const amount = parseReceiptMinor(value); return amount !== null && Number.isSafeInteger(amount) && amount <= MAX_RECEIPT_MINOR;
  });

  const commitExpense = () => {
    const input = saveAttempt.current;
    if (!input) return;
    saveReceipt.mutate(input, {
      onSuccess: () => {
        saveAttempt.current = null; setUncertainSave(false); setSaveKey(newRequestId());
        feedback(t('receiptSaved')); navigate(`/apps/split/${groupId}`);
      },
      onError: (error) => {
        const uncertain = !(error instanceof ApiError) || error.status >= 500;
        setUncertainSave(uncertain);
        if (!uncertain) { saveAttempt.current = null; setSaveKey(newRequestId()); }
        feedback(uncertain ? t('receiptSaveUncertain') : error instanceof ApiError ? error.message : t('receiptSaveFailed'), 'error');
      },
      onSettled: () => { submitting.current = false; },
    });
  };

  const save = () => {
    if (submitting.current) return;
    if (uncertainSave && saveAttempt.current) { submitting.current = true; commitExpense(); return; }
    if (!reviewed || !validDraft || !payerId || !receiptId || preview.unallocated.length || preview.difference !== 0) {
      feedback(t('receiptConfirmRequired'), 'error'); return;
    }
    submitting.current = true;
    updateReceipt.mutate({
      version,
      totalMinor: String(totalMinor), discountMinor: String(parseReceiptMinor(discount) ?? 0), serviceMinor: String(parseReceiptMinor(service) ?? 0),
      items: items.map((item) => ({ id: item.id, name: item.name.trim(), quantity: item.quantity, amountMinor: String(parseReceiptMinor(item.amount) ?? 0), unclear: false, units: item.units })),
    }, {
      onSuccess: () => {
        saveAttempt.current = { payerId, clientRequestId: saveKey, description: group?.title ?? t('receiptTitle'), category };
        commitExpense();
      },
      onError: (error) => { submitting.current = false; feedback(error instanceof ApiError ? error.message : t('receiptSaveFailed'), 'error'); },
    });
  };

  const difference = preview.difference;

  return (
    <Screen
      header={
        <AppHeader title={t('receiptTitle')} onBack={() => navigate(`/apps/split/${groupId}`)} />
      }
      bottomSpace
    >
      {!receiptId && originalPhoto ? <ReceiptPhotoEditor key={`${originalPhoto.name}:${originalPhoto.lastModified}`} file={originalPhoto} onChange={(file) => { setSelectedPhoto(file); setConsent(false); }} onEditing={setPhotoEditing} /> : null}
      {photoUrl ? <Card className="mt-2 gap-2 p-3">
        <img src={photoUrl} alt={t('receiptPhotoPreview')} className="max-h-64 w-full rounded-xl object-contain" />
        <p className="text-caption text-ink-2">{t(receiptId ? 'receiptServerPhotoDeleted' : 'receiptLocalPreview')}</p>
      </Card> : null}
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
            onChange={(event) => { onPickFile(event.target.files?.[0]); event.target.value = ''; }}
          />
          <div className="flex flex-col gap-2 pt-2">
            <PrimaryButton onClick={() => fileInput.current?.click()} full disabled={upload.isPending}>
              {selectedPhoto ? t('receiptChooseAnother') : t('receiptTakePhoto')}
            </PrimaryButton>
            {selectedPhoto ? <>
              <label className="flex min-h-11 items-start gap-2 py-2 text-caption text-ink-2">
                <input type="checkbox" checked={consent} onChange={(event) => setConsent(event.target.checked)} disabled={upload.isPending} className="mt-0.5 size-5 shrink-0 accent-accent" />
                {t('receiptDeepSeekConsent')}
              </label>
              <PrimaryButton onClick={recognize} full disabled={!consent || photoEditing} progress={upload.isPending}>{upload.isPending ? t('receiptRecognizing') : t('receiptRecognize')}</PrimaryButton>
            </> : null}
            <SecondaryButton onClick={startManual} full disabled={upload.isPending}>
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
          <fieldset disabled={draftLocked} className="m-0 min-w-0 border-0 p-0">
          <SectionLabel>{t('receiptItems')}</SectionLabel>
          <div className="flex flex-col gap-2">
            {items.map((item) => (
              <Card key={item.key} className="p-3">
                <div className="flex items-start gap-2">
                  <input
                    value={item.name}
                    onChange={(event) => patchItem(item.key, { name: event.target.value })}
                    placeholder={t('receiptItemName')}
                    aria-label={t('receiptItemName')}
                    maxLength={200}
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
                    max={999}
                    aria-label={t('receiptQuantity')}
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
                    className="min-h-11 max-w-full rounded-full bg-accent-soft px-3 text-caption font-semibold text-accent"
                  >
                    {t('receiptEveryone')}
                  </button>
                  {members.map((person) => {
                    const units = item.units[person.id] ?? 0;
                    return (
                      <span key={person.id} className="flex max-w-full flex-wrap items-center gap-1">
                        <button
                          type="button"
                          aria-pressed={units > 0}
                          onClick={() => toggleUnit(item.key, person.id)}
                          className={
                            units > 0
                              ? 'min-h-11 max-w-full whitespace-normal break-words rounded-full bg-accent-soft px-3 text-caption font-semibold text-accent'
                              : 'min-h-11 max-w-full whitespace-normal break-words rounded-full bg-neutral-soft px-3 text-caption text-ink-2'
                          }
                        >
                          {person.public_name}
                        </button>
                        {units > 0 && item.quantity > 1 ? (
                          <span className="flex items-center gap-0.5">
                            <button
                              type="button"
                              aria-label={`${t('receiptLessUnits')}: ${person.public_name}`}
                              onClick={() => setUnits(item.key, person.id, units - 1)}
                              className="flex size-11 items-center justify-center rounded-full bg-neutral-soft text-ink-2"
                            >
                              <Icon name="remove" size={14} />
                            </button>
                            <span className="min-w-4 text-center text-caption text-ink">{units}</span>
                            <button
                              type="button"
                              aria-label={`${t('receiptMoreUnits')}: ${person.public_name}`}
                              onClick={() => setUnits(item.key, person.id, units + 1)}
                              className="flex size-11 items-center justify-center rounded-full bg-neutral-soft text-ink-2"
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
            disabled={items.length >= MAX_RECEIPT_ITEMS}
            className="mt-2 flex w-full items-center gap-2 rounded-[14px] bg-surface px-4 py-3 text-left text-body-lg text-hint"
          >
            <Icon name="add_circle" size={20} className="text-accent" />
            {t('receiptAddItem')}
          </button>

          <SectionLabel>{t('receiptTotals')}</SectionLabel>
          <Card className="p-3">
            <label className="text-caption text-ink-2" htmlFor="receipt-category">
              {t('category')}
            </label>
            <div className="flex flex-wrap gap-1.5 pt-1">
              {CATEGORIES.map((name) => (
                <button
                  key={name}
                  type="button"
                  onClick={() => setCategory(name)}
                  className={
                    category === name
                      ? 'min-h-11 max-w-full whitespace-normal break-words rounded-full bg-accent-soft px-3 text-caption font-semibold text-accent'
                      : 'min-h-11 max-w-full whitespace-normal break-words rounded-full bg-neutral-soft px-3 text-caption text-ink-2'
                  }
                >
                  {t(CATEGORY_KEY[name] ?? 'catOther')}
                </button>
              ))}
            </div>

            <label className="pt-3 text-caption text-ink-2" htmlFor="receipt-total">
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

          <SectionLabel>{t('receiptResult')}</SectionLabel>
          <Card className="p-4">
            <div className="flex items-center justify-between">
              <span className="text-body-secondary text-ink-2">{t('receiptTotal')}</span>
              <span className="text-body-strong text-ink">{formatMinor(String(totalMinor))} ₸</span>
            </div>
            {(parseReceiptMinor(discount) ?? 0) > 0 ? (
              <div className="flex items-center justify-between pt-1">
                <span className="text-body-secondary text-ink-2">{t('receiptDiscount')}</span>
                <span className="text-body-secondary text-success">
                  -{formatMinor(String(parseReceiptMinor(discount) ?? 0))} ₸
                </span>
              </div>
            ) : null}
            {(parseReceiptMinor(service) ?? 0) > 0 ? (
              <div className="flex items-center justify-between pt-1">
                <span className="text-body-secondary text-ink-2">{t('receiptService')}</span>
                <span className="text-body-secondary text-warning">
                  +{formatMinor(String(parseReceiptMinor(service) ?? 0))} ₸
                </span>
              </div>
            ) : null}

            <div className="pt-3 text-caption text-ink-2">{t('receiptShares')}</div>
            {Object.entries(preview.perPerson).map(([userId, share]) => (
              <div key={userId} className="flex items-center justify-between pt-1">
                <span className="text-body-secondary text-ink">{nameOf(userId)}</span>
                <span className="text-body-secondary text-ink">{formatMinor(String(share))} ₸</span>
              </div>
            ))}

            <div className="pt-3 text-caption text-ink-2">{t('receiptOwes')}</div>
            {Object.entries(preview.owes).length === 0 ? (
              <div className="pt-1 text-caption text-ink-2">{t('receiptNobodyOwes')}</div>
            ) : null}
            {Object.entries(preview.owes).map(([userId, share]) => (
              <div key={userId} className="flex items-center justify-between pt-1">
                <span className="text-body-secondary text-ink">
                  {nameOf(userId)} → {nameOf(payerId)}
                </span>
                <Chip tone="warning">{formatMinor(String(share))} ₸</Chip>
              </div>
            ))}

            {preview.unallocated.length > 0 ? (
              <div className="pt-3">
                <Banner tone="warning" icon="info" title={t('receiptUnassigned')} />
              </div>
            ) : null}

            {difference !== 0 ? (
              <div className="pt-3">
                <Banner tone="danger" icon="error" title={t('receiptMismatch')}>
                  {t('receiptDifference')}: {formatMinor(String(difference))} ₸
                </Banner>
              </div>
            ) : null}
          </Card>

          <label className="mt-3 flex min-h-11 items-start gap-2 rounded-xl bg-surface p-3 text-caption text-ink">
            <input type="checkbox" checked={reviewed} onChange={(event) => setReviewedSignature(event.target.checked ? draftSignature : '')} className="mt-0.5 size-5 shrink-0 accent-accent" />
            {t('receiptReviewConfirmed')}
          </label>
          {!validDraft ? <p role="alert" className="pt-2 text-caption text-danger">{t('receiptDraftInvalid')}</p> : null}
          </fieldset>
          {uncertainSave ? <Banner tone="warning" icon="info" title={t('receiptSaveUncertain')} /> : null}
          <div className="pt-3">
            <PrimaryButton
              onClick={save}
              full
              progress={saveReceipt.isPending || updateReceipt.isPending}
              disabled={!uncertainSave && (difference !== 0 || !payerId || !reviewed || !validDraft || preview.unallocated.length > 0)}
            >
              {t(uncertainSave ? 'receiptRetrySave' : 'receiptSave')}
            </PrimaryButton>
          </div>
        </>
      ) : null}
    </Screen>
  );
}
