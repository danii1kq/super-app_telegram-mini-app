/** Merge overlapping history pages without losing bigint sequence precision. */
export function orderedMessages<T extends { id: string; seq: string }>(pages: { messages: T[] }[]): T[] {
  const messages = new Map<string, T>();
  for (const page of pages.slice().reverse()) {
    for (const message of page.messages) messages.set(message.id, message);
  }
  return [...messages.values()].sort((a, b) => BigInt(a.seq) < BigInt(b.seq) ? -1 : BigInt(a.seq) > BigInt(b.seq) ? 1 : 0);
}
