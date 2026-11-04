import assert from "node:assert/strict";
import {
  formatDonation,
  rupeesToPaise,
  validDonationRows,
} from "/tmp/opencode/orix-leaderboard-tests/donation-leaderboard.js";

for (const [value, want] of [
  ["1", 100],
  ["0.01", 1],
  ["125.50", 12550],
  [" 250 ", 25000],
  ["10000000", 1_000_000_000],
]) {
  assert.equal(rupeesToPaise(value), want);
}
for (const value of [
  "",
  "0",
  "-1",
  "1.001",
  "1e3",
  "NaN",
  "1,000",
  "10000000.01",
  "10000000000000",
  "1.2.3",
]) {
  assert.equal(rupeesToPaise(value), null, value);
}
assert.match(formatDonation(12550), /₹125\.50/);
assert.match(formatDonation(10000000), /₹1,00,000/);
assert.equal(validDonationRows([]), true);
assert.equal(validDonationRows([{ name: "Supporter", amount_paise: 1 }]), true);
assert.equal(
  validDonationRows([{ name: "Supporter", amount_paise: 1.5 }]),
  false,
);
assert.equal(validDonationRows([{ name: " ", amount_paise: 100 }]), false);
assert.equal(
  validDonationRows([{ name: "Supporter", amount_paise: -100 }]),
  false,
);
assert.equal(validDonationRows(null), false);
console.log(
  "Donation leaderboard money parsing, INR formatting and response validation passed.",
);
