import { useEffect, useState } from "react";
import { Heart, Copy } from "lucide-react";
import { Modal } from "./Modal";
import { donationPayee, donationUpiId } from "../lib/donation-config";
import { donationDay, donationPromptKey, googlePayLink } from "../lib/donation";

let sessionDay = "";
export function DonationSupport({
  autoPrompt = false,
}: {
  autoPrompt?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [copyMessage, setCopyMessage] = useState("");
  function markShown() {
    sessionDay = donationDay();
    try {
      localStorage.setItem(donationPromptKey, sessionDay);
    } catch {
      /* Once this session if storage is blocked. */
    }
  }
  useEffect(() => {
    if (!autoPrompt) return;
    const check = () => {
      if (document.visibilityState !== "visible") return;
      const today = donationDay();
      if (sessionDay === today) return;
      try {
        if (localStorage.getItem(donationPromptKey) === today) return;
      } catch {
        /* Session deduplication remains active. */
      }
      // Avoid interrupting another dialog, onboarding, or an active form.
      if (
        document.querySelector('[role="dialog"], [role="alertdialog"]') ||
        /INPUT|TEXTAREA|SELECT/.test(document.activeElement?.tagName ?? "")
      )
        return;
      sessionDay = today;
      try {
        localStorage.setItem(donationPromptKey, today);
      } catch {
        /* Optional preference only. */
      }
      setOpen(true);
    };
    const timer = window.setTimeout(check, 1500);
    document.addEventListener("visibilitychange", check);
    return () => {
      clearTimeout(timer);
      document.removeEventListener("visibilitychange", check);
    };
  }, [autoPrompt]);
  async function copy() {
    try {
      await navigator.clipboard.writeText(donationUpiId);
      setCopyMessage("UPI ID copied.");
    } catch {
      setCopyMessage("Could not copy. Select the UPI ID above and copy it.");
    }
  }
  return (
    <>
      <button
        className="sidebar-link donation-trigger"
        onClick={() => {
          markShown();
          setOpen(true);
        }}
      >
        <Heart size={18} />
        <span>Donate</span>
      </button>
      <Modal
        title="A little help for ORIX."
        description="Help cover the running costs. Any amount is welcome."
        open={open}
        onOpenChange={setOpen}
      >
        <div className="donation-support">
          <div className="donation-qr">
            <img
              src="/art/donation-qr.png"
              width="246"
              height="246"
              alt="Scan to donate with any UPI app"
            />
            <p>Scan with any UPI app</p>
          </div>
          <div className="donation-mobile">
            <a
              className="button primary"
              href={googlePayLink(navigator.userAgent)}
            >
              Open Google Pay
            </a>
            <p>Opens Google Pay. Choose any amount there.</p>
          </div>
          <div className="donation-recipient">
            <span className="donation-recipient-label">Paying</span>
            <strong>{donationPayee}</strong>
            <span className="donation-upi">{donationUpiId}</span>
            <button className="button ghost" onClick={() => void copy()}>
              <Copy size={16} />
              Copy UPI ID
            </button>
            {copyMessage && <p role="status">{copyMessage}</p>}
          </div>
          <p className="donation-optional">
            Always optional. ORIX works the same either way.
          </p>
          <button className="button ghost" onClick={() => setOpen(false)}>
            Maybe later
          </button>
        </div>
      </Modal>
    </>
  );
}
