/** Place names scan independently; full provider labels stay available on demand. */
export function JourneyRoute({ from, to }: { from: string; to: string }) {
  const name = (value: string) => value.split(",")[0].trim() || value;
  const expanded = name(from) !== from || name(to) !== to;
  return (
    <div className="journey-route">
      <dl className="journey-route-stops">
        <div>
          <dt>From</dt>
          <dd>{name(from)}</dd>
        </div>
        <div>
          <dt>To</dt>
          <dd>{name(to)}</dd>
        </div>
      </dl>
      {expanded && (
        <details className="journey-route-details">
          <summary>Full addresses</summary>
          <dl>
            <div>
              <dt>From</dt>
              <dd>{from}</dd>
            </div>
            <div>
              <dt>To</dt>
              <dd>{to}</dd>
            </div>
          </dl>
        </details>
      )}
    </div>
  );
}
