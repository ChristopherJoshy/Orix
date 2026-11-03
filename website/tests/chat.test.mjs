import assert from "node:assert/strict";
import { mergeMessages, isChatMessage } from "/tmp/orix-chat-tests/chat.js";
const message = (id, seconds, extra = {}) => ({
  message_id: id,
  content: id,
  is_mine: false,
  created_at: `2026-11-03T12:00:${String(seconds).padStart(2, "0")}Z`,
  ...extra,
});
const pending = message("local-a", 2, {
  is_mine: true,
  client_message_id: "a",
  pending: true,
});
const ack = message("server-a", 3, { is_mine: true, client_message_id: "a" });
let result = mergeMessages([pending], [ack]);
assert.deepEqual(
  result.map((m) => m.message_id),
  ["server-a"],
);
assert.deepEqual(mergeMessages(result, [ack]), result);
console.log(
  "PASS persisted ack replaces pending bubble; socket echo does not duplicate it",
);
result = mergeMessages(
  [message("old", 1), message("live", 5)],
  [message("snapshot", 3)],
);
assert.deepEqual(
  result.map((m) => m.message_id),
  ["old", "snapshot", "live"],
);
console.log("PASS stale snapshots preserve live arrivals and loaded history");
const failed = { ...pending, pending: false, failed: true };
assert.equal(
  mergeMessages([failed], [message("other", 1)]).at(-1).failed,
  true,
);
assert.deepEqual(mergeMessages([failed], [ack]), [ack]);
const theirs = { ...ack, is_mine: false };
assert.equal(mergeMessages([pending], [theirs]).length, 2);
console.log(
  "PASS failed drafts survive refresh; only own acknowledgements reconcile client IDs",
);
assert.equal(isChatMessage({ message_id: "x" }), false);
assert.equal(isChatMessage({ ...ack, created_at: "invalid" }), false);
assert.equal(isChatMessage(ack), true);
console.log("PASS malformed socket messages cannot enter the conversation");
