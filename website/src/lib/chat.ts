export interface ChatMessage {
  message_id: string;
  content: string;
  is_mine: boolean;
  user_display_name?: string;
  created_at: string;
  client_message_id?: string;
  pending?: boolean;
  failed?: boolean;
}

/** REST acknowledgements and socket echoes converge on one persisted message. */
export function mergeMessages(
  current: ChatMessage[],
  incoming: ChatMessage[],
): ChatMessage[] {
  const confirmed = new Map(
    current
      .filter((m) => !m.pending && !m.failed)
      .map((m) => [m.message_id, m]),
  );
  for (const message of incoming) confirmed.set(message.message_id, message);
  const acknowledged = new Set(
    [...confirmed.values()]
      .filter((m) => m.is_mine && !m.pending && !m.failed)
      .map((m) => m.client_message_id)
      .filter(Boolean),
  );
  const local = current.filter(
    (m) =>
      (m.pending || m.failed) &&
      !acknowledged.has(m.client_message_id) &&
      !confirmed.has(m.message_id),
  );
  return [...confirmed.values(), ...local].sort(
    (a, b) =>
      Date.parse(a.created_at) - Date.parse(b.created_at) ||
      a.message_id.localeCompare(b.message_id),
  );
}
export function isChatMessage(value: unknown): value is ChatMessage {
  if (!value || typeof value !== "object") return false;
  const m = value as Partial<ChatMessage>;
  return (
    typeof m.message_id === "string" &&
    typeof m.content === "string" &&
    typeof m.is_mine === "boolean" &&
    typeof m.created_at === "string" &&
    Number.isFinite(Date.parse(m.created_at))
  );
}
