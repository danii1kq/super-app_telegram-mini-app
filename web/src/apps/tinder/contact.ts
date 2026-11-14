/**
 * Builds a Telegram deep link from a username, or an empty string when the
 * username is not a valid one.
 *
 * Telegram usernames are 5 to 32 characters, start with a letter and contain
 * only letters, digits and underscores. A profile can carry an arbitrary string
 * in that field, so the value is validated before it becomes a link: otherwise
 * a crafted username could point somewhere else entirely.
 */
export function telegramContact(username: string): string {
  return /^[A-Za-z][A-Za-z0-9_]{4,31}$/.test(username) ? `https://t.me/${username}` : '';
}
