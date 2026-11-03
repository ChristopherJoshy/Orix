import { SceneArt } from "../components/SceneArt";
import "./Auth.css";
import { friendlyError } from "../lib/errors";
import { useEffect, useState } from "react";
import { Link, Navigate, useNavigate, useSearchParams } from "react-router-dom";
import { Button, Field, Logo, Toggle } from "../components/ui";
import { PixelStatus } from "../components/PixelStatus";
import { OrixTour } from "../components/OrixTour";
import {
  apiRequest,
  signInGoogle,
  useSession,
  verifyAccount,
} from "../lib/session";
export function Login() {
  const { account, loading, error: sessionError } = useSession();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const requestedNext = params.get("next");
  const next =
    requestedNext &&
    /^\/(?:app(?:\/|$)|account-deletion$)/.test(requestedNext) &&
    !requestedNext.includes("\\")
      ? requestedNext
      : "/app";
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  if (account)
    return (
      <Navigate
        to={
          account.onboarding_completed
            ? next
            : `/onboarding?next=${encodeURIComponent(next)}`
        }
        replace
      />
    );
  return (
    <div className="login-illustrated">
      <header className="login-topbar">
        <Link to="/">Back to Orix</Link>
      </header>
      <main id="main" className="login-composition">
        <figure className="login-art">
          <SceneArt scene="welcome" priority />
        </figure>
        <div className="login-content">
          <Logo />
          <h1>Go together.</h1>
          <p>
            A familiar route. A little company.
            <br />
            Sign in to find your people.
          </p>
          <Button
            className="google-signin wide"
            disabled={busy || loading}
            onClick={async () => {
              setBusy(true);
              setError("");
              try {
                const user = await signInGoogle();
                if (user)
                  navigate(
                    user.onboarding_completed
                      ? next
                      : `/onboarding?next=${encodeURIComponent(next)}`,
                    {
                      replace: true,
                    },
                  );
              } catch (problem) {
                setError(friendlyError(problem));
              } finally {
                setBusy(false);
              }
            }}
          >
            {busy ? "Connecting to Google…" : "Continue with Google"}
          </Button>
          {(error || sessionError) && (
            <p role="alert">{error || sessionError}</p>
          )}
          <p className="login-consent">
            By continuing, you agree to our <Link to="/terms">terms</Link> and{" "}
            <Link to="/privacy">privacy policy</Link>.
          </p>
        </div>
      </main>
    </div>
  );
}
export function Onboarding({ embedded = false }: { embedded?: boolean }) {
  const { account, loading } = useSession();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const target = params.get("next");
  const next =
    target &&
    /^\/(?:app(?:\/|$)|account-deletion$)/.test(target) &&
    !target.includes("\\")
      ? target
      : "/app";
  const [step, setStep] = useState(0);
  const [phone, setPhone] = useState("");
  const [gender, setGender] = useState("");
  const [occupation, setOccupation] = useState("");
  const [photo, setPhoto] = useState("");
  const [photoClear, setPhotoClear] = useState(false);
  const [uploaded, setUploaded] = useState(false);
  const [offerLifts, setOfferLifts] = useState(false);
  const [accepted, setAccepted] = useState(false);
  const [privacy, setPrivacy] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [permission, setPermission] = useState("");
  const [tour, setTour] = useState(false);
  useEffect(
    () => () => {
      if (photo) URL.revokeObjectURL(photo);
    },
    [photo],
  );
  if (loading)
    return (
      <main className="loading-screen">
        <PixelStatus state="loading" label="Verifying account" />
      </main>
    );
  if (!account) return <Navigate to="/login" replace />;
  if (account.onboarding_completed && !embedded)
    return <Navigate to={next} replace />;
  const Container = embedded ? "section" : "main";
  const titles = [
    "Your details",
    "Your photo",
    "Travel preferences",
    "Ready to travel",
  ];
  const hasPhoto = uploaded || !!account.photo_url;
  async function upload(file: File) {
    if (
      !["image/jpeg", "image/png", "image/webp"].includes(file.type) ||
      file.size > 2 * 1024 * 1024
    ) {
      setError("Choose a JPEG, PNG or WebP photo smaller than 2 MB.");
      return;
    }
    setBusy(true);
    setError("");
    setPhotoClear(false);
    try {
      const body = new FormData();
      body.append("file", file);
      await apiRequest("/users/me/avatar", { method: "POST", body });
      setPhoto(URL.createObjectURL(file));
      setUploaded(true);
    } catch (problem) {
      setError(friendlyError(problem));
    } finally {
      setBusy(false);
    }
  }
  async function finish() {
    if (!accepted || !privacy || !photoClear) {
      setError(
        "Confirm your photo and read the safety and privacy information.",
      );
      return;
    }
    setBusy(true);
    setError("");
    try {
      await apiRequest("/auth/onboard", {
        method: "POST",
        body: JSON.stringify({
          phone: phone.replace(/[^+0-9]/g, ""),
          gender: gender || null,
          safety_consent: accepted,
          privacy_consent: privacy,
          photo_clear: true,
          offer_lifts: offerLifts,
          tour_version: 1,
          occupation,
        }),
      });
      await verifyAccount();
      navigate(next, { replace: true });
    } catch (problem) {
      setError(friendlyError(problem));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Container
      className="standalone-form introduction"
      id={embedded ? undefined : "main"}
    >
      <Logo />
      <p className="muted">Step {step + 1} of 4</p>
      <h1>{titles[step]}</h1>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          setError("");
          if (
            step === 0 &&
            !/^\+[1-9][0-9]{9,14}$/.test(phone.replace(/[^+0-9]/g, ""))
          ) {
            setError("Enter your phone number with country code.");
            return;
          }
          if (step === 1 && (!hasPhoto || !photoClear)) {
            setError("Add or confirm a clear profile photo.");
            return;
          }
          if (step === 3) {
            void finish();
            return;
          }
          if (step === 2) setTour(true);
          setStep(step + 1);
        }}
      >
        {step === 0 && (
          <>
            <SceneArt scene="companions" className="intro-scene" />
            <p>
              {account.display_name}
              <br />
              {account.email}
            </p>
            <Field label="I am a">
              <select
                value={occupation}
                required
                onChange={(e) => setOccupation(e.target.value)}
              >
                <option value="" disabled>
                  Choose one
                </option>
                <option value="student">Student</option>
                <option value="teacher">Teacher</option>
                <option value="other">Other</option>
              </select>
            </Field>
            <Field label="Phone number">
              <input
                type="tel"
                autoComplete="tel"
                placeholder="+91 98765 43210"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                maxLength={20}
                required
              />
            </Field>
            <Field label="Gender (optional)">
              <select
                value={gender}
                onChange={(e) => setGender(e.target.value)}
              >
                <option value="">Prefer not to say</option>
                <option value="F">Woman</option>
                <option value="M">Man</option>
                <option value="O">Another identity</option>
              </select>
            </Field>
            <p className="muted">
              Women-only matching depends on verified eligibility.
            </p>
          </>
        )}
        {step === 1 && (
          <>
            <p>
              Use a recent photo with your face clearly visible. Travel partners
              use it to recognise you and may decline a match if they cannot
              identify you.
            </p>
            {photo ? (
              <img
                className="intro-photo"
                src={photo}
                alt="Your selected profile photo"
              />
            ) : (
              account.photo_url && (
                <p>
                  Your Google profile photo is available. Replace it if your
                  face is not clear.
                </p>
              )
            )}
            <Field label="Profile photo">
              <input
                type="file"
                accept="image/jpeg,image/png,image/webp"
                disabled={busy}
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  if (file) void upload(file);
                }}
              />
            </Field>
            {(busy || uploaded) && (
              <PixelStatus
                state={busy ? "uploading" : "uploaded"}
                label={busy ? "Uploading photo" : "Photo uploaded"}
              />
            )}
            <label className="checkbox-label">
              <input
                type="checkbox"
                disabled={!hasPhoto || busy}
                checked={photoClear}
                onChange={(e) => setPhotoClear(e.target.checked)}
              />
              <span>My profile photo clearly shows my face</span>
            </label>
          </>
        )}
        {step === 2 && (
          <>
            <Toggle
              label="I would like to offer lifts"
              description="Register a route, time and seats before being matched as a driver."
              checked={offerLifts}
              onChange={setOfferLifts}
            />
            <h2>Optional permissions</h2>
            <p>
              Notifications can alert you to updates. Location can fill your
              starting point. You can continue without either and choose places
              manually.
            </p>
            <div className="intro-actions">
              <Button
                variant="secondary"
                type="button"
                onClick={async () => {
                  if (!("Notification" in window)) {
                    setPermission(
                      "Notifications are unavailable in this browser.",
                    );
                    return;
                  }
                  try {
                    const result = await Notification.requestPermission();
                    setPermission(
                      result === "granted"
                        ? "Notification permission enabled."
                        : "Notifications are off. You can change this in browser settings.",
                    );
                  } catch {
                    setPermission(
                      "Notifications are unavailable. You can continue without them.",
                    );
                  }
                }}
              >
                Enable notifications
              </Button>
              <Button
                variant="secondary"
                type="button"
                onClick={() => {
                  if (!navigator.geolocation) {
                    setPermission(
                      "Location is unavailable. Choose places manually.",
                    );
                    return;
                  }
                  navigator.geolocation.getCurrentPosition(
                    () => setPermission("Location permission enabled."),
                    () =>
                      setPermission(
                        "Location is off. You can select places manually.",
                      ),
                    { timeout: 10000, maximumAge: 60000 },
                  );
                }}
              >
                Enable location
              </Button>
            </div>
            {permission && <p role="status">{permission}</p>}
          </>
        )}
        {step === 3 && (
          <>
            <Button
              type="button"
              variant="secondary"
              onClick={() => setTour(true)}
            >
              Replay the tour
            </Button>
            <p>
              Check the person and pickup plan before travelling. Agree on costs
              directly. You can block contacts by email or start a private
              report conversation.
            </p>
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={accepted}
                onChange={(e) => setAccepted(e.target.checked)}
              />
              <span>
                I agree to the <Link to="/terms">community safety rules</Link>
              </span>
            </label>
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={privacy}
                onChange={(e) => setPrivacy(e.target.checked)}
              />
              <span>
                I have read the <Link to="/privacy">privacy policy</Link>
              </span>
            </label>
          </>
        )}
        {error && <p role="alert">{error}</p>}
        <div className="intro-actions">
          {step > 0 && (
            <Button
              variant="ghost"
              type="button"
              disabled={busy}
              onClick={() => setStep(step - 1)}
            >
              Back
            </Button>
          )}
          <Button disabled={busy || (step === 3 && (!accepted || !privacy))}>
            {busy ? "Saving…" : step === 3 ? "Finish setup" : "Continue"}
          </Button>
        </div>
      </form>
      <OrixTour open={tour} onClose={() => setTour(false)} />
    </Container>
  );
}
