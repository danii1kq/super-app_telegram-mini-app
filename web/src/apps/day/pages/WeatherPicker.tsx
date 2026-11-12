import { useState } from 'react';
import { useWeatherCities, type WeatherCity, type WeatherDay } from '../api';
import { localDayKey } from '../calendar';
import { useI18n, useT } from '@/shared/i18n';
import { Banner, Card, Icon } from '@/shared/ui';
import { deviceStorage } from '@/shared/telegram/sdk';

const CITY_KEY = 'srunal.weather.city';

export function loadWeatherCity(): WeatherCity | null {
  try {
    const raw = deviceStorage.get(CITY_KEY);
    if (!raw) return null;
    const city = JSON.parse(raw) as WeatherCity;
    return Number.isFinite(city.latitude) && Math.abs(city.latitude) <= 90 && Number.isFinite(city.longitude) && Math.abs(city.longitude) <= 180 && typeof city.name === 'string' && city.name.length <= 200 && Number.isSafeInteger(city.id) ? city : null;
  } catch { return null; }
}

export function WeatherPicker({ city, onChange }: { city: WeatherCity | null; onChange: (city: WeatherCity | null) => void }) {
  const t = useT();
  const { lang } = useI18n();
  const [draft, setDraft] = useState('');
  const [search, setSearch] = useState('');
  const [picking, setPicking] = useState(!city);
  const cities = useWeatherCities(picking ? search : '', lang);
  const select = (value: WeatherCity | null) => {
    deviceStorage.set(CITY_KEY, value ? JSON.stringify(value) : '');
    onChange(value); setPicking(!value); setSearch(''); setDraft('');
  };

  return <Card className="gap-2 p-3">
    <div className="flex items-center gap-2">
      <Icon name="partly_cloudy_day" size={22} className="shrink-0 text-accent" />
      <span className="min-w-0 flex-1 break-words text-body-strong text-ink">{city ? `${city.name}${city.country ? `, ${city.country}` : ''}` : t('dayWeatherChoose')}</span>
      {city ? <button type="button" onClick={() => setPicking(!picking)} aria-expanded={picking} className="min-h-11 px-2 text-body-secondary text-accent">{t('edit')}</button> : null}
    </div>
    {picking ? <>
      <form onSubmit={(event) => { event.preventDefault(); setSearch(draft.trim()); }} className="flex min-w-0 gap-2">
        <input value={draft} onChange={(event) => setDraft(event.target.value)} maxLength={80} aria-label={t('dayWeatherCity')} placeholder={t('dayWeatherCity')} className="h-12 min-w-0 flex-1 rounded-xl bg-neutral-soft/60 px-3 text-body-lg text-ink outline-none focus:ring-2 focus:ring-accent" />
        <button type="submit" disabled={draft.trim().length < 2} aria-label={t('search')} className="flex size-12 shrink-0 items-center justify-center rounded-xl bg-accent text-btn-text disabled:opacity-50"><Icon name="search" size={22} /></button>
      </form>
      <p className="text-caption text-ink-2">{t('dayWeatherPrivacy')}</p>
      {cities.isFetching ? <p role="status" className="text-caption text-ink-2">{t('loading')}</p> : null}
      {cities.isError ? <Banner tone="warning" icon="cloud_off" title={t('dayWeatherUnavailable')} action={<button type="button" onClick={() => void cities.refetch()} className="min-h-11">{t('retry')}</button>} /> : null}
      {cities.data?.map((item) => <button type="button" key={item.id} onClick={() => select(item)} className="min-h-12 rounded-xl px-2 py-2 text-left text-body-secondary text-ink active:bg-neutral-soft"><span className="block font-semibold">{item.name}</span><span className="text-caption text-ink-2">{[item.region, item.country].filter(Boolean).join(', ')}</span></button>)}
      {cities.data?.length === 0 ? <p className="text-caption text-ink-2">{t('noSearchResults')}</p> : null}
      {city ? <button type="button" onClick={() => select(null)} className="min-h-11 self-start px-2 text-body-secondary text-ink-2">{t('dayWeatherRemove')}</button> : null}
    </> : null}
    <a href="https://open-meteo.com/" target="_blank" rel="noreferrer" className="self-start text-caption text-accent">{t('dayWeatherSource')}</a>
  </Card>;
}

function weatherIcon(code: number): string {
  if (code === 0) return 'sunny';
  if (code <= 3) return 'partly_cloudy_day';
  if (code <= 48) return 'foggy';
  if (code >= 95) return 'thunderstorm';
  if ((code >= 71 && code <= 77) || code >= 85) return 'weather_snowy';
  return 'rainy';
}

export function DayWeather({ date, days, loading }: { date: Date; days: WeatherDay[] | undefined; loading: boolean }) {
  const t = useT();
  const weather = days?.find((item) => item.date === localDayKey(date));
  return <div className="mb-2 flex min-h-6 items-center gap-2 px-1 text-caption text-ink-2">
    {weather ? <><Icon name={weatherIcon(weather.code)} size={20} className="text-accent" /><span aria-label={t('dayWeatherTemperature')}>{Math.round(weather.min)}° / {Math.round(weather.max)}°C</span></> : <span>{t(loading ? 'loading' : 'dayWeatherOutside')}</span>}
  </div>;
}
