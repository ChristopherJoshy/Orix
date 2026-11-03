import { useEffect, useRef } from "react";
import { toast } from "sonner";

export type PixelState =
  | "loading"
  | "searching"
  | "found"
  | "no-match"
  | "lift-offered"
  | "cancelled"
  | "travelling"
  | "completed"
  | "report-sent"
  | "uploading"
  | "uploaded"
  | "error"
  | "connection-lost";
const Person = ({ x = 0 }: { x?: number }) => (
  <g transform={`translate(${x} 0)`}>
    <path
      d="M3 0h8v3h3v8h-3v3H3v-3H0V3h3zm0 3v7h8V3zM0 17h14v4H0zm0 4h4v10H0zm10 0h4v10h-4zm-6 0h6v4H4z"
      fillRule="evenodd"
    />
  </g>
);
const Car = () => (
  <g>
    <path
      d="M5 0h20v4h6v8h3v9h-4v4h-6v-4H9v4H3v-4h-5v-9h3V4h4zm-1 5v7h11V5zm14 0v7h9V5z"
      fillRule="evenodd"
    />
    <path d="M0 15h3v3H0zm29 0h3v3h-3z" fill="#aaa" />
  </g>
);
const Check = () => (
  <path d="M0 8h4v4h4v-4h4V4h4V0h4v4h-4v4h-4v4H8v4H4v-4H0z" />
);
export function PixelStatus({
  state,
  label,
  compact = false,
}: {
  state: PixelState;
  label?: string;
  compact?: boolean;
}) {
  const ref = useRef<SVGSVGElement>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    let visible = true;
    const sync = () => {
      el.dataset.active = String(visible && !document.hidden);
    };
    const observer = new IntersectionObserver((entries) => {
      visible = entries[0].isIntersecting;
      sync();
    });
    observer.observe(el);
    document.addEventListener("visibilitychange", sync);
    sync();
    return () => {
      observer.disconnect();
      document.removeEventListener("visibilitychange", sync);
    };
  }, [state]);
  return (
    <div
      className={`pixel-status ${state === "loading" ? "pixel-loader" : ""} ${compact ? "compact" : ""}`}
    >
      <svg
        ref={ref}
        key={state}
        viewBox={state === "loading" ? "30 23 52 25" : "0 0 120 72"}
        className={`pixel-state state-${state}`}
        fill="#222"
        shapeRendering="crispEdges"
        role={label ? "img" : undefined}
        aria-label={label}
        aria-hidden={label ? undefined : true}
      >
        {state !== "loading" && (
          <path
            d="M8 61h7v2H8zm11 0h7v2h-7zm11 0h7v2h-7zm11 0h7v2h-7zm11 0h7v2h-7zm11 0h7v2h-7zm11 0h7v2h-7zm11 0h7v2h-7zm11 0h7v2h-7z"
            fill="#ddd"
          />
        )}
        {state === "connection-lost" && (
          <g className="connection-chaya">
            <path d="M30 34h28v4H30zm0 4h4v18h20V38h4v22H30zm28 2h10v12H58v-4h6v-4h-6zM26 62h36v3H26z" />
            <g className="chaya-steam" fill="#888">
              <path d="M36 19h4v10h-4zm11-5h4v12h-4z" />
            </g>
            <path
              d="M78 22h20v4H78zm-5 5h5v4h-5zm25 0h5v4h-5zM82 32h12v4H82zm-4 5h4v4h-4zm16 0h4v4h-4zM86 42h4v4h-4z"
              fill="#999"
            />
            <path d="M74 44h4v4h-4zm4 4h4v4h-4zm4 4h4v4h-4zm4-4h4v4h-4zm4-4h4v4h-4z" />
          </g>
        )}
        {state === "loading" && (
          <g className="loading-pixels">
            {[0, 1, 2, 3].map((i) => (
              <rect
                key={i}
                x={38 + i * 12}
                y="34"
                width="6"
                height="6"
                style={{ animationDelay: `${i * 0.15}s` }}
              />
            ))}
          </g>
        )}
        {(state === "uploading" || state === "uploaded") && (
          <>
            <path fillRule="evenodd" d="M25 14h42v43H25zm4 4v35h34V18z" />
            <g transform="translate(39 20)">
              <Person />
            </g>
            {state === "uploading" ? (
              <path
                className="upload-arrow"
                d="M83 25h4v28h-4zm-8 8h4v-4h4v-4h4v4h4v4h4v4h-4v-4h-4v-4h-4v4h-4v4h-4z"
              />
            ) : (
              <g transform="translate(77 29)">
                <Check />
              </g>
            )}
          </>
        )}
        {state === "searching" && (
          <>
            <g transform="translate(12 27)" fill="#999">
              <Person />
            </g>
            <g transform="translate(94 27)" fill="#999">
              <Person />
            </g>
            <g className="search-lens">
              <path
                d="M53 15h14v2h2v12h-2v2H53v-2h-2V17h2zm0 2v12h14V17zM68 30h7v3h-7z"
                fillRule="evenodd"
              />
            </g>
            <g className="search-signals">
              {[0, 1, 2, 3].map((i) => (
                <rect
                  key={i}
                  x={38 + i * 11}
                  y="48"
                  width="4"
                  height="4"
                  style={{ animationDelay: `${i * 0.2}s` }}
                />
              ))}
            </g>
          </>
        )}
        {state === "found" && (
          <>
            <g className="found-left">
              <g transform="translate(35 27)">
                <Person />
              </g>
            </g>
            <g className="found-right">
              <g transform="translate(71 27)">
                <Person />
              </g>
            </g>
            <g transform="translate(52 7)">
              <g className="found-check">
                <Check />
              </g>
            </g>
            <path
              className="found-sparks"
              d="M27 15h3v3h-3zm61 2h3v3h-3zM57 1h3v3h-3z"
              fill="#999"
            />
          </>
        )}
        {state === "no-match" && (
          <>
            <g transform="translate(15 27)" fill="#999">
              <Person />
            </g>
            <path
              className="empty-route"
              d="M49 45h16v2H49zm19 0h5v2h-5z"
              fill="#999"
            />
            <path
              className="empty-place"
              d="M87 42h13v2H87zm0 10h13v2H87zm-2-8h2v8h-2zm15 0h2v8h-2z"
              fill="#999"
            />
            <path d="M51 24h2v2h-2zm5 0h2v2h-2zm5 0h2v2h-2z" />
          </>
        )}
        {state === "lift-offered" && (
          <>
            <g transform="translate(10 28)" fill="#999">
              <Person />
            </g>
            <g className="offered-car">
              <g transform="translate(37 34)">
                <Car />
              </g>
            </g>
            <path
              className="seat-arrow"
              d="M84 20h3v20h-3zm-6 9h3v3h-3zm3-3h3v3h-3zm6 0h3v3h-3zm3 3h3v3h-3z"
            />
          </>
        )}
        {state === "cancelled" && (
          <>
            <g transform="translate(12 27)" fill="#999">
              <Person />
            </g>
            <path className="cancel-route" d="M34 48h30v2H34z" fill="#999" />
            <path
              className="cancel-cross"
              d="M73 25h4v4h4v4h4v-4h4v-4h4v4h-4v4h-4v4h4v4h4v4h-4v-4h-4v-4h-4v4h-4v4h-4v-4h4v-4h4v-4h-4v-4h-4z"
            />
          </>
        )}
        {state === "travelling" && (
          <>
            <g className="travel-car">
              <g transform="translate(43 32)">
                <Car />
              </g>
            </g>
            <g className="travel-road" fill="#999">
              <path d="M5 56h12v2H5zm27 0h12v2H32zm27 0h12v2H59zm27 0h12v2H86zm27 0h12v2h-12z" />
            </g>
            <path d="M14 37h14v2H14zm-6 6h17v2H8z" fill="#ddd" />
          </>
        )}
        {state === "completed" && (
          <>
            <g transform="translate(26 27)">
              <Person />
            </g>
            <path d="M85 16h3v44h-3z" />
            <g className="finish-flag">
              {[0, 1, 2, 3].map((x) =>
                [0, 1, 2].map((y) => (
                  <rect
                    key={`${x}-${y}`}
                    x={88 + x * 4}
                    y={16 + y * 4}
                    width="4"
                    height="4"
                    fill={(x + y) % 2 ? "#ddd" : "#222"}
                  />
                )),
              )}
            </g>
            <g transform="translate(52 26)">
              <g className="finish-check">
                <Check />
              </g>
            </g>
          </>
        )}
        {state === "report-sent" && (
          <>
            <g className="sent-envelope">
              <path d="M36 20h46v26H36zm4 4v18h38V24z" fillRule="evenodd" />
              <path d="M40 24h4v2h4v2h4v2h4v2h6v-2h4v-2h4v-2h4v-2h4v2h-4v2h-4v2h-4v2h-4v2h-6v-2h-4v-2h-4v-2h-4v-2h-4z" />
            </g>
            <g transform="translate(85 21)">
              <g className="sent-check">
                <Check />
              </g>
            </g>
          </>
        )}
        {state === "error" && (
          <>
            <g transform="translate(14 27)" fill="#999">
              <Person />
            </g>
            <path d="M40 56h18v3H40zm34 0h28v3H74z" />
            <path
              className="error-break"
              d="M57 55h3v6h-3zm15 3h3v6h-3z"
              fill="#999"
            />
            <path className="error-mark" d="M64 24h4v15h-4zm0 20h4v4h-4z" />
          </>
        )}
      </svg>
    </div>
  );
}
/** Only successful API acknowledgements emit these outcomes. */
export function LifecycleFeedback() {
  useEffect(() => {
    const outcomes: Record<string, { state: PixelState; label: string }> = {
      searching: { state: "searching", label: "Finding your people" },
      found: { state: "found", label: "Your group is ready" },
      "lift-offered": { state: "lift-offered", label: "Lift registered" },
      cancelled: { state: "cancelled", label: "Request cancelled" },
      travelling: { state: "travelling", label: "Journey started" },
      completed: { state: "completed", label: "Destination reached" },
      "report-sent": { state: "report-sent", label: "Private report created" },
    };
    const listener = (event: Event) => {
      const outcome =
        outcomes[(event as CustomEvent<{ state: string }>).detail?.state];
      if (outcome)
        toast(
          <div className="pixel-feedback">
            <PixelStatus state={outcome.state} compact />
            <strong>{outcome.label}</strong>
          </div>,
          { duration: 4200 },
        );
    };
    window.addEventListener("orix:outcome", listener);
    return () => window.removeEventListener("orix:outcome", listener);
  }, []);
  return null;
}
