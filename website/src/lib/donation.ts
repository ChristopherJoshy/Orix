import { donationUri } from "./donation-config.js";

// Prompt preference only. This never records or infers payment completion.
export const donationPromptKey = "orix.donation.lastPromptDay.v1";
export function donationDay(now = new Date()) {
  return `${now.getFullYear()}-${now.getMonth() + 1}-${now.getDate()}`;
}
export function googlePayLink(platform: string) {
  if (/android/i.test(platform)) {
    return `intent://pay?${donationUri.split("?")[1]}#Intent;scheme=upi;package=com.google.android.apps.nbu.paisa.user;end`;
  }
  return donationUri.replace("upi://", "gpay://upi/");
}
