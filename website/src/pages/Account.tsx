import { enableWebPush, disableWebPush } from "../lib/webPush";
import { friendlyError } from "../lib/errors";
import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import {
  apiRequest,
  signOutAccount,
  useSession,
  verifyAccount,
} from "../lib/session";
import { useResource } from "../lib/resource";
import { refreshJourneys } from "../lib/journeys";
import { useStore } from "../lib/store";
import { PersonIdentity } from "../components/EmailPersonPicker";
import { TourLauncher } from "../components/OrixTour";
import { PixelStatus } from "../components/PixelStatus";
import { Button, Field, PageHead, Toggle } from "../components/ui";
import { Modal } from "../components/Modal";
interface ProfileData {
  user_id: string;
  email: string;
  display_name: string;
  photo_url?: string;
  occupation?: string;
  phone?: string;
  gender?: string;
  is_rider: boolean;
}
interface Preferences {
  ride_reminders: boolean;
  chat_messages: boolean;
  group_updates: boolean;
  promotions: boolean;
}
const preferenceLabels: [keyof Preferences, string, string][] = [
  [
    "ride_reminders",
    "Journey reminders",
    "A reminder before your planned departure.",
  ],
  ["chat_messages", "Group messages", "New messages from your travel group."],
  [
    "group_updates",
    "Group activity",
    "Matching, confirmation and pickup updates.",
  ],
  ["promotions", "ORIX news", "Optional product updates."],
];
export function Profile() {
  const owner = useSession((s) => s.account?.user_id);
  const { data: profile, error, load } = useResource<ProfileData>("/users/me");
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState("");
  const [occupation, setOccupation] = useState("other");
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const [saved, setSaved] = useState(false);
  const sending = useRef(false);
  useEffect(() => {
    setEditing(false);
    setName("");
    setOccupation("other");
    setBusy(false);
    setFailure("");
    setSaved(false);
    sending.current = false;
  }, [owner]);
  async function update(
    body: FormData | { display_name: string; occupation: string },
  ) {
    if (sending.current) return;
    sending.current = true;
    setBusy(true);
    setFailure("");
    setSaved(false);
    try {
      await apiRequest(
        "/users/me" + (body instanceof FormData ? "/avatar" : ""),
        {
          method: body instanceof FormData ? "POST" : "PUT",
          body: body instanceof FormData ? body : JSON.stringify(body),
        },
      );
      if (owner !== useSession.getState().account?.user_id) return;
      setEditing(false);
      setSaved(true);
      await verifyAccount();
      refreshJourneys();
      await load();
    } catch (problem) {
      if (owner === useSession.getState().account?.user_id)
        setFailure(friendlyError(problem));
    } finally {
      if (owner === useSession.getState().account?.user_id) {
        sending.current = false;
        setBusy(false);
      }
    }
  }
  return (
    <>
      <PageHead
        title="Your profile."
        subtitle="Help your travel partners recognise you"
      />
      {(error || failure) && (
        <p role="alert">
          {failure || error}{" "}
          <Button variant="secondary" onClick={() => void load()}>
            Refresh profile
          </Button>
        </p>
      )}
      {!profile && !error ? (
        <PixelStatus state="loading" label="Opening your profile" />
      ) : (
        profile && (
          <>
            <section className="panel">
              <PersonIdentity
                person={{
                  userId: profile.user_id,
                  email: profile.email,
                  name: profile.display_name,
                  photoUrl: profile.photo_url,
                }}
              />
              <p>
                {profile.occupation === "student"
                  ? "Student"
                  : profile.occupation === "teacher"
                    ? "Teacher"
                    : profile.occupation === "other"
                      ? "Other"
                      : "Occupation not provided"}
              </p>
              <p>{profile.phone}</p>
              <Button
                variant="secondary"
                disabled={busy}
                onClick={() => {
                  setName(profile.display_name);
                  setOccupation(profile.occupation || "other");
                  setEditing(true);
                  setFailure("");
                }}
              >
                Edit details
              </Button>
            </section>
            <section className="panel">
              <h2>A clear face. A familiar hello.</h2>
              <p>
                Choose a recent photo that clearly shows your face. Others may
                decline a group if they cannot recognise you.
              </p>
              <Field label="Replace profile photo">
                <input
                  type="file"
                  accept="image/jpeg,image/png,image/webp"
                  disabled={busy}
                  onChange={(event) => {
                    const file = event.target.files?.[0];
                    event.target.value = "";
                    if (!file) return;
                    if (
                      !["image/jpeg", "image/png", "image/webp"].includes(
                        file.type,
                      ) ||
                      file.size > 2 * 1024 * 1024
                    ) {
                      setFailure(
                        "Choose a JPEG, PNG or WebP photo up to 2 MB.",
                      );
                      return;
                    }
                    const body = new FormData();
                    body.append("file", file);
                    void update(body);
                  }}
                />
              </Field>
              {busy && <PixelStatus state="uploading" label="Saving profile" />}
              {saved && !busy && (
                <PixelStatus state="uploaded" compact label="Profile saved" />
              )}
            </section>
            <section className="panel">
              <h2>Your account</h2>
              <p>
                Your email comes from your verified Google account. Your
                Student, Teacher or Other label is shown to your group and does
                not grant special permissions.
              </p>
              <div className="intro-actions">
                <Link className="button secondary" to="/app/settings">
                  Account settings
                </Link>
                <Link className="button secondary" to="/app/rider">
                  Manage lifts
                </Link>
                <TourLauncher />
              </div>
            </section>
          </>
        )
      )}
      <Modal
        open={editing}
        onOpenChange={(value) => {
          if (!busy) setEditing(value);
        }}
        title="Edit your details"
        description="Your name and occupation are visible to your travel partners."
      >
        <form
          onSubmit={(event) => {
            event.preventDefault();
            void update({ display_name: name.trim(), occupation });
          }}
        >
          <Field label="Display name">
            <input
              required
              minLength={1}
              maxLength={100}
              value={name}
              disabled={busy}
              onChange={(event) => setName(event.target.value)}
            />
          </Field>
          <Field label="I am a">
            <select
              value={occupation}
              disabled={busy}
              onChange={(event) => setOccupation(event.target.value)}
            >
              <option value="student">Student</option>
              <option value="teacher">Teacher</option>
              <option value="other">Other</option>
            </select>
          </Field>
          {failure && <p role="alert">{failure}</p>}
          <Button disabled={busy || !name.trim()}>
            {busy ? "Saving…" : "Save details"}
          </Button>
        </form>
      </Modal>
    </>
  );
}
export function Settings() {
  const owner = useSession((s) => s.account?.user_id);
  const { data, error, load } = useResource<Preferences>(
    "/users/me/notification-preferences",
  );
  const { motion, setMotion } = useStore();
  const [busy, setBusy] = useState(false);
  const [failure, setFailure] = useState("");
  const [saved, setSaved] = useState("");
  const sending = useRef(false);
  useEffect(() => {
    setBusy(false);
    setFailure("");
    setSaved("");
    sending.current = false;
  }, [owner]);
  async function change(key: keyof Preferences, value: boolean) {
    if (sending.current) return;
    sending.current = true;
    setBusy(true);
    setFailure("");
    setSaved("");
    try {
      await apiRequest("/users/me/notification-preferences", {
        method: "PUT",
        body: JSON.stringify({ [key]: value }),
      });
      if (owner === useSession.getState().account?.user_id) {
        setSaved("Notification preference saved.");
        refreshJourneys();
        await load();
      }
    } catch (problem) {
      if (owner === useSession.getState().account?.user_id)
        setFailure(friendlyError(problem));
    } finally {
      if (owner === useSession.getState().account?.user_id) {
        sending.current = false;
        setBusy(false);
      }
    }
  }
  return (
    <>
      <PageHead
        title="Your settings."
        subtitle="Your preferences, across your devices"
      />
      {(error || failure) && (
        <p role="alert">
          {failure || error}{" "}
          <Button variant="secondary" onClick={() => void load()}>
            Refresh
          </Button>
        </p>
      )}
      <section className="panel">
        <h2>Stay in the loop.</h2>
        {!data && !error ? (
          <PixelStatus state="loading" label="Opening preferences" />
        ) : (
          data && (
            <fieldset
              disabled={busy}
              style={{ border: 0, padding: 0, margin: 0, minWidth: 0 }}
            >
              {preferenceLabels.map(([key, label, description]) => (
                <Toggle
                  key={key}
                  label={label}
                  description={description}
                  checked={data[key]}
                  onChange={(value) => void change(key, value)}
                />
              ))}
            </fieldset>
          )
        )}
        {saved && <p role="status">{saved}</p>}
        <p>
          These preferences are saved to your account. Delivery also depends on
          your device and browser permissions.
        </p>
      </section>
      <section className="panel">
        <h2>Browser notifications</h2>
        <p>
          Enable notifications on this browser to receive an alert when an
          update arrives.
        </p>
        <div className="intro-actions">
          <Button
            variant="secondary"
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              setFailure("");
              setSaved("");
              try {
                await enableWebPush();
                if (owner === useSession.getState().account?.user_id)
                  setSaved("Browser notifications enabled.");
              } catch (problem) {
                if (owner === useSession.getState().account?.user_id)
                  setFailure(friendlyError(problem));
              } finally {
                if (owner === useSession.getState().account?.user_id)
                  setBusy(false);
              }
            }}
          >
            Enable on this browser
          </Button>
          <Button
            variant="secondary"
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              setFailure("");
              setSaved("");
              try {
                await disableWebPush();
                if (owner === useSession.getState().account?.user_id)
                  setSaved("Browser notifications disabled.");
              } catch (problem) {
                if (owner === useSession.getState().account?.user_id)
                  setFailure(friendlyError(problem));
              } finally {
                if (owner === useSession.getState().account?.user_id)
                  setBusy(false);
              }
            }}
          >
            Disable on this browser
          </Button>
        </div>
      </section>
      <section className="panel">
        <h2>Motion</h2>
        <Toggle
          label="Play animations"
          description="Saved on this browser. Your device’s reduced-motion preference is always respected."
          checked={motion}
          onChange={setMotion}
        />
        <TourLauncher />
      </section>
      <section className="panel">
        <h2>Privacy and account</h2>
        <div className="intro-actions">
          <Link className="button secondary" to="/privacy">
            Privacy policy
          </Link>
          <Link className="button secondary" to="/terms">
            Terms
          </Link>
          <Link className="button secondary" to="/account-deletion">
            Delete my account
          </Link>
          <Button
            variant="secondary"
            onClick={() =>
              void signOutAccount().catch(() =>
                setFailure("Could not sign out. Try again."),
              )
            }
          >
            Sign out
          </Button>
        </div>
      </section>
    </>
  );
}
