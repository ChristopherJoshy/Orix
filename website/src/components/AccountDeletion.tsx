import { useState } from "react";
import { Link } from "react-router-dom";
import { GoogleAuthProvider, reauthenticateWithPopup } from "firebase/auth";
import {
  apiRequest,
  firebaseAuth,
  signOutAccount,
  useSession,
} from "../lib/session";
import { friendlyError } from "../lib/errors";
import { Button, Field } from "./ui";
import { Modal } from "./Modal";
export function AccountDeletion() {
  const account = useSession((s) => s.account);
  const [open, setOpen] = useState(false);
  const [confirmation, setConfirmation] = useState("");
  const [busy, setBusy] = useState(false);
  const [requested, setRequested] = useState(false);
  const [error, setError] = useState("");
  async function remove() {
    const user = firebaseAuth.currentUser;
    if (!user || busy || confirmation !== "DELETE") return;
    setBusy(true);
    setError("");
    try {
      await reauthenticateWithPopup(user, new GoogleAuthProvider());
      if (firebaseAuth.currentUser?.uid !== user.uid) return;
      await apiRequest("/users/me", { method: "DELETE" });
      setRequested(true);
      setOpen(false);
      await signOutAccount();
    } catch (problem) {
      setError(friendlyError(problem));
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="panel">
      <h2>{requested ? "Deletion requested." : "Request account deletion"}</h2>
      {requested ? (
        <p>
          Your request was accepted. Account cleanup continues in the
          background; this is not a confirmation that deletion has finished.
        </p>
      ) : account ? (
        <>
          <p>
            Deleting your account removes your access and starts cleanup of your
            account and associated data. You’ll confirm your Google identity
            again before the request is sent.
          </p>
          <Button variant="secondary" onClick={() => setOpen(true)}>
            Review deletion
          </Button>
        </>
      ) : (
        <Link className="button secondary" to="/login?next=/account-deletion">
          Sign in to request deletion
        </Link>
      )}
      {error && <p role="alert">{error}</p>}
      <Modal
        title="Delete your ORIX account?"
        description="This starts account cleanup. Make sure you are signed in to the account you want to delete."
        open={open}
        onOpenChange={(value) => {
          if (!busy) setOpen(value);
        }}
      >
        <Field label="Type DELETE to confirm">
          <input
            value={confirmation}
            autoComplete="off"
            disabled={busy}
            onChange={(event) => setConfirmation(event.target.value)}
          />
        </Field>
        <Button
          disabled={busy || confirmation !== "DELETE"}
          onClick={() => void remove()}
        >
          {busy ? "Requesting deletion…" : "Verify with Google and delete"}
        </Button>
        <Button
          variant="secondary"
          disabled={busy}
          onClick={() => setOpen(false)}
        >
          Keep my account
        </Button>
        {error && <p role="alert">{error}</p>}
      </Modal>
    </section>
  );
}
