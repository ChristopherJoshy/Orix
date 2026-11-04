export interface DonationLeader {
  name: string;
  amount_paise: number;
}
export interface DonationLeaderboard {
  version: number;
  entries: (DonationLeader & { id: string })[];
}

export const maxDonationPaise = 1_000_000_000;

/** Parse rupees without accepting exponents or rounding away fractional paise. */
export function rupeesToPaise(value: string): number | null {
  const match = /^(\d{1,8})(?:\.(\d{1,2}))?$/.exec(value.trim());
  if (!match) return null;
  const amount =
    Number(match[1]) * 100 + Number((match[2] ?? "").padEnd(2, "0"));
  return Number.isSafeInteger(amount) &&
    amount > 0 &&
    amount <= maxDonationPaise
    ? amount
    : null;
}

export function formatDonation(amountPaise: number): string {
  return new Intl.NumberFormat("en-IN", {
    style: "currency",
    currency: "INR",
    minimumFractionDigits: amountPaise % 100 ? 2 : 0,
    maximumFractionDigits: 2,
  }).format(amountPaise / 100);
}

export function validDonationRows(value: unknown): value is DonationLeader[] {
  return (
    Array.isArray(value) &&
    value.length <= 50 &&
    value.every(
      (row) =>
        row &&
        typeof row.name === "string" &&
        row.name.trim() &&
        Number.isSafeInteger(row.amount_paise) &&
        row.amount_paise > 0 &&
        row.amount_paise <= maxDonationPaise,
    )
  );
}
