import { friendlyError } from "../lib/errors";
import { useEffect, useId, useRef, useState } from "react";
import { apiRequest } from "../lib/session";
import type { SelectedPlace } from "../lib/types";
import { PixelStatus } from "./PixelStatus";
export function PlaceAutocomplete({
  label,
  value,
  onChange,
  pickup = false,
  disabled = false,
}: {
  label: string;
  value: SelectedPlace | null;
  onChange: (value: SelectedPlace | null) => void;
  pickup?: boolean;
  disabled?: boolean;
}) {
  const id = useId();
  const [query, setQuery] = useState(value?.label ?? "");
  const [results, setResults] = useState<SelectedPlace[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const epoch = useRef(0);
  const [credit, setCredit] = useState<{ text: string; url: string } | null>(
    null,
  );
  useEffect(() => {
    if (value) {
      setQuery(value.label);
      setResults([]);
      setError("");
    }
  }, [value]);
  useEffect(() => {
    if (value || query.trim().length < 3) {
      setResults([]);
      setBusy(false);
      return;
    }
    const current = ++epoch.current;
    const controller = new AbortController();
    const timer = setTimeout(async () => {
      setBusy(true);
      setError("");
      try {
        const data = await apiRequest<{
          places: {
            place_id: string;
            label: string;
            latitude: number;
            longitude: number;
          }[];
          attribution: string;
          attribution_url: string;
        }>(`/places/search?query=${encodeURIComponent(query.trim())}`, {
          signal: controller.signal,
        });
        const places: SelectedPlace[] = data.places
          .filter(
            (p) =>
              p.place_id &&
              p.label &&
              Number.isFinite(p.latitude) &&
              Math.abs(p.latitude) <= 90 &&
              Number.isFinite(p.longitude) &&
              Math.abs(p.longitude) <= 180,
          )
          .map((p) => ({
            id: p.place_id,
            label: p.label,
            latitude: p.latitude,
            longitude: p.longitude,
          }));
        if (epoch.current === current && !controller.signal.aborted) {
          setResults(places.slice(0, 5));
          setCredit({ text: data.attribution, url: data.attribution_url });
          if (!places.length)
            setError("No places found. Include a city or area.");
        }
      } catch (error) {
        if (!controller.signal.aborted && epoch.current === current)
          setError(
            friendlyError(error),
          );
      } finally {
        if (!controller.signal.aborted && epoch.current === current)
          setBusy(false);
      }
    }, 450);
    return () => {
      clearTimeout(timer);
      controller.abort();
      epoch.current++;
    };
  }, [query, value]);
  return (
    <div className="place-autocomplete">
      <label htmlFor={id}>{label}</label>
      <input
        id={id}
        value={query}
        disabled={disabled}
        autoComplete="off"
        placeholder="Type a place or address"
        maxLength={200}
        onChange={(event) => {
          setQuery(event.target.value);
          onChange(null);
          setError("");
        }}
        aria-describedby={`${id}-hint`}
        aria-expanded={results.length > 0}
        aria-controls={results.length ? `${id}-results` : undefined}
      />
      {busy && <PixelStatus state="loading" compact label="Searching places" />}
      {error && <p role="status">{error}</p>}
      {results.length > 0 && (
        <ul
          id={`${id}-results`}
          className="place-results"
          aria-label={`${label} suggestions`}
        >
          {results.map((place) => (
            <li key={place.id}>
              <button
                type="button"
                onClick={() => {
                  onChange(place);
                  setQuery(place.label);
                  setResults([]);
                }}
              >
                <strong>{place.label}</strong>
                <small>
                  {place.latitude.toFixed(5)}, {place.longitude.toFixed(5)}
                </small>
              </button>
            </li>
          ))}
        </ul>
      )}
      <small id={`${id}-hint`}>
        {value
          ? pickup
            ? "Match pickups within 300 m and destinations within 100 m."
            : "Destinations must be within 100 m of each other."
          : "Choose a suggestion to set the place and coordinates."}
      </small>
      {credit && /^https:\/\//.test(credit.url) && (
        <a
          className="place-credit"
          href={credit.url}
          target="_blank"
          rel="noreferrer"
        >
          {credit.text}
        </a>
      )}
    </div>
  );
}
