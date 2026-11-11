/**
 * HTTP client for /api/v1. The session token lives in memory only and travels
 * in the Authorization header; nothing is written to localStorage.
 */

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly fields?: Record<string, string>;

  constructor(status: number, code: string, message: string, fields?: Record<string, string>) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
    this.fields = fields;
  }
}

let sessionToken = '';
let onUnauthorized: (() => void) | undefined;

export function setSessionToken(token: string): void {
  sessionToken = token;
}

export function clearSessionToken(): void {
  sessionToken = '';
}

export function setUnauthorizedHandler(handler: (() => void) | undefined): void {
  onUnauthorized = handler;
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE';
  body?: unknown;
  signal?: AbortSignal;
  /** Idempotency key for repeatable writes (messages, expenses). */
  clientId?: string;
}

export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (sessionToken) headers.Authorization = `Bearer ${sessionToken}`;
  if (options.body !== undefined) headers['Content-Type'] = 'application/json';
  if (options.clientId) headers['Idempotency-Key'] = options.clientId;

  const response = await fetch(`/api/v1${path}`, {
    method: options.method ?? 'GET',
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    signal: options.signal,
  });

  if (response.status === 401) {
    clearSessionToken();
    onUnauthorized?.();
  }

  const text = await response.text();
  const payload = text ? (JSON.parse(text) as unknown) : undefined;

  if (!response.ok) {
    const error = payload as { error?: { code?: string; message?: string; fields?: Record<string, string> } } | undefined;
    throw new ApiError(
      response.status,
      error?.error?.code ?? 'unknown',
      error?.error?.message ?? `Запрос не выполнен (${response.status})`,
      error?.error?.fields,
    );
  }

  return payload as T;
}

/** Server returns bigint IDs and money as strings; keep them as strings. */
export type Id = string;
