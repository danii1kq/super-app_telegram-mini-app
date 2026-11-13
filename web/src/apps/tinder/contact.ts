export function telegramContact(username: string): string {
  return /^[A-Za-z][A-Za-z0-9_]{4,31}$/.test(username) ? `https://t.me/${username}` : '';
}
