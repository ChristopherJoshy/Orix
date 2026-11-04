import { useResource } from "../lib/resource";
import { PublicError } from "../lib/errors";
import { apiRequest } from "../lib/session";
import {
  formatDonation,
  validDonationRows,
  type DonationLeader,
} from "../lib/donation-leaderboard";
import { Button, PageHead } from "../components/ui";
import { PixelStatus } from "../components/PixelStatus";
import "./leaderboard.css";

// A 32 × 32 native-grid keeps the thank-you mark crisp at its 4× display size.
function PixelThanks() {
  return (
    <svg
      className="leaderboard-pixel"
      viewBox="0 0 32 32"
      width="128"
      height="128"
      aria-hidden="true"
      shapeRendering="crispEdges"
    >
      <path
        fill="currentColor"
        d="M7 10h18v3h4v9h-3v3h-6v3h5v3H7v-3h5v-3H6v-3H3v-9h4z"
      />
      <path
        fill="#fff"
        d="M5 15h2v5H5zm20 0h2v5h-2zM10 12h12v9h-3v3h-6v-3h-3z"
      />
      <path
        fill="currentColor"
        d="M12 2h3v2h2V2h3v2h2v4h-2v2h-2v2h-4v-2h-2V8h-2V4h2z"
      />
      <path fill="#a8a8a8" d="M10 14h2v6h-2zm4 12h4v2h-4z" />
      <path fill="currentColor" d="M2 4h2v2H2zm26 3h2v2h-2zM4 28h2v2H4z" />
    </svg>
  );
}

export function LeaderboardView({
  entries,
  loading,
  error,
  retry,
}: {
  entries: DonationLeader[] | null;
  loading: boolean;
  error: string;
  retry: () => void;
}) {
  return (
    <div className="donation-leaderboard">
      <PageHead
        title="Leaderboard"
        subtitle="A little support. A big difference."
      />
      <section
        className="leaderboard-thanks"
        aria-labelledby="leaderboard-thanks-title"
      >
        <div>
          <span className="eyebrow">Good people. Good journeys.</span>
          <h2 id="leaderboard-thanks-title">You keep us going.</h2>
          <p>
            To everyone who has donated: thank you for putting a little kindness
            into the journey. Your support helps keep ORIX free for everyone.
          </p>
        </div>
        <PixelThanks />
      </section>
      <section
        className="leaderboard-list"
        aria-labelledby="leaderboard-list-title"
        aria-busy={loading}
      >
        <h2 id="leaderboard-list-title">Our top supporters</h2>
        {error ? (
          <div className="leaderboard-state" role="alert">
            <PixelStatus state="error" />
            <p>{error}</p>
            <Button variant="secondary" onClick={retry} disabled={loading}>
              Retry
            </Button>
          </div>
        ) : entries === null ? (
          <div className="leaderboard-state">
            <PixelStatus state="loading" label="Loading our supporters" />
          </div>
        ) : entries.length === 0 ? (
          <div className="leaderboard-state">
            <p>
              No donations have been published yet. Every bit of kindness
              counts.
            </p>
          </div>
        ) : (
          <table className="leaderboard-table">
            <caption className="sr-only">
              Supporters, ordered by highest total donation
            </caption>
            <thead>
              <tr>
                <th scope="col">Name</th>
                <th scope="col">Amount</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((entry, index) => (
                <tr key={`${entry.name}-${index}`}>
                  <td>{entry.name}</td>
                  <td>{formatDonation(entry.amount_paise)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </div>
  );
}

async function readLeaderboard(path: string) {
  const result = await apiRequest<{ entries: DonationLeader[] }>(path);
  if (!validDonationRows(result?.entries))
    throw new PublicError("The leaderboard couldn’t load. Please try again.");
  return result;
}

export default function Leaderboard() {
  const { data, loading, error, load } = useResource(
    "/donations/leaderboard",
    0,
    readLeaderboard,
  );
  return (
    <LeaderboardView
      entries={data?.entries ?? null}
      loading={loading}
      error={error}
      retry={() => void load()}
    />
  );
}
