import type { RideMode } from "./types";
export function localDate(offset = 0) {
  const date = new Date();
  date.setDate(date.getDate() + offset);
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}
export const modeLabels: Record<RideMode, string> = {
  auto: "Shared auto pool",
  partner: "Travel companion pool",
  lift: "Lift pool",
};
export const poolLabel = (mode: RideMode | null) =>
  mode ? modeLabels[mode] : "Waiting for a pool";
