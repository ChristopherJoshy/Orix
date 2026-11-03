import { friendlyError } from "../lib/errors";
import { useEffect, useId, useRef, useState } from "react";
import { apiOrigin, firebaseAuth, useSession } from "../lib/session";
import { searchContacts, type TravelContact } from "../lib/safety";
import { PixelMark } from "./PixelArt";
export function ContactPhoto({ person }: { person: TravelContact }) {
  const identity = useSession((s) => s.account?.user_id);
  const [source, setSource] = useState<string | null>(null);
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    let objectURL: string | undefined;
    setSource(null);
    setFailed(false);
    if (!person.photoUrl || !identity) return;
    try {
      const url = new URL(person.photoUrl, apiOrigin());
      if (url.username || url.password) return;
      if (
        url.origin === apiOrigin() &&
        url.pathname.startsWith("/api/v1/users/")
      ) {
        const user = firebaseAuth.currentUser;
        if (!user) return;
        void user
          .getIdToken()
          .then((token) =>
            fetch(url, {
              headers: { Authorization: `Bearer ${token}` },
              credentials: "omit",
              redirect: "error",
              signal: controller.signal,
            }),
          )
          .then(async (response) => {
            if (!response.ok) throw new Error();
            return response.blob();
          })
          .then((blob) => {
            if (
              controller.signal.aborted ||
              firebaseAuth.currentUser?.uid !== user.uid
            )
              return;
            objectURL = URL.createObjectURL(blob);
            setSource(objectURL);
          })
          .catch(() => {
            if (!controller.signal.aborted) setFailed(true);
          });
      } else if (url.protocol === "https:") setSource(url.href);
    } catch {
      /* Invalid photo uses the pixel avatar. */
    }
    return () => {
      controller.abort();
      if (objectURL) URL.revokeObjectURL(objectURL);
    };
  }, [identity, person.photoUrl, attempt]);
  return source && !failed ? (
    <img
      src={source}
      alt=""
      width="40"
      height="40"
      referrerPolicy="no-referrer"
      onError={() => setFailed(true)}
    />
  ) : person.photoUrl ? (
    <button
      type="button"
      className="photo-retry"
      aria-label={`Retry photo for ${person.name || person.email}`}
      onClick={() => setAttempt((value) => value + 1)}
    >
      Retry photo
    </button>
  ) : (
    <span className="photo-unavailable">No photo</span>
  );
}
export function PersonIdentity({ person }: { person: TravelContact }) {
  return (
    <span className="person-identity">
      <ContactPhoto
        key={`${person.userId}:${person.photoUrl}`}
        person={person}
      />
      <span>
        <strong>{person.email || "Email unavailable"}</strong>
        {person.name && <small>{person.name}</small>}
      </span>
    </span>
  );
}
export function EmailPersonPicker({
  selected,
  onSelect,
  label = "Email address",
  disabled = false,
}: {
  selected: TravelContact | null;
  onSelect: (person: TravelContact | null) => void;
  label?: string;
  disabled?: boolean;
}) {
  const id = useId();
  const identity = useSession((s) => s.account?.user_id);
  const [query, setQuery] = useState(selected?.email ?? "");
  const [results, setResults] = useState<TravelContact[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const previous = useRef(selected);
  useEffect(() => {
    if (selected && selected.email !== previous.current?.email)
      setQuery(selected.email);
    else if (!selected && previous.current) {
      const old = previous.current.email;
      setQuery((value) => (value === old ? "" : value));
    }
    previous.current = selected;
  }, [selected]);
  useEffect(() => {
    setResults([]);
    setError("");
    setLoading(false);
    if (selected || disabled || !identity || query.trim().length < 2) return;
    const controller = new AbortController();
    setLoading(true);
    const timer = window.setTimeout(() => {
      setLoading(true);
      void searchContacts(query.trim(), controller.signal)
        .then((rows) => {
          if (!controller.signal.aborted) setResults(rows);
        })
        .catch((problem) => {
          if (!controller.signal.aborted) setError(friendlyError(problem));
        })
        .finally(() => {
          if (!controller.signal.aborted) setLoading(false);
        });
    }, 350);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [query, selected, disabled, identity]);
  return (
    <div className="email-picker">
      <label htmlFor={id}>{label}</label>
      <input
        id={id}
        type="text"
        inputMode="email"
        autoComplete="off"
        spellCheck={false}
        value={query}
        disabled={disabled}
        maxLength={100}
        placeholder="Search a travel contact’s email"
        onChange={(event) => {
          setQuery(event.target.value);
          onSelect(null);
        }}
        aria-describedby={`${id}-hint`}
      />
      <small id={`${id}-hint`}>
        Search your travel contacts using a partial email or a small spelling
        correction.
      </small>
      {selected ? (
        <div className="email-selected">
          <PersonIdentity person={selected} />
          <button
            type="button"
            disabled={disabled}
            onClick={() => {
              setQuery("");
              onSelect(null);
            }}
          >
            Change
          </button>
        </div>
      ) : (
        query.trim().length >= 2 && (
          <div className="email-results" aria-label="Matching emails">
            {loading ? (
              <div role="status" aria-label="Searching emails">
                <PixelMark className="contact-search-brand" />
              </div>
            ) : error ? (
              <p role="alert">{error}</p>
            ) : results.length ? (
              results.map((person) => (
                <button
                  type="button"
                  key={person.userId}
                  onClick={() => {
                    setQuery(person.email);
                    onSelect(person);
                  }}
                >
                  <PersonIdentity person={person} />
                </button>
              ))
            ) : (
              <p role="status">No matching travel contact.</p>
            )}
          </div>
        )
      )}
    </div>
  );
}
