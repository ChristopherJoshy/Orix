/** A small, static pixel vignette; decorative, not a live map. */
export function HeroJourney() {
  return (
    <svg
      viewBox="0 0 160 100"
      fill="none"
      shapeRendering="crispEdges"
      aria-hidden="true"
    >
      <path d="M8 82h144v2H8z" fill="#b8b8b8" />
      <path
        d="M18 90h12v2H18zm30 0h12v2H48zm30 0h12v2H78zm30 0h12v2h-12z"
        fill="#d8d8d8"
      />
      <path d="M120 40h4v42h-4z" fill="#777" />
      <path d="M114 25h16v5h5v14h-5v5h-21v-5h-5V30h10z" fill="#dedede" />
      <path d="M114 25h16v5h5v7h-9v-4h-12z" fill="#ccc" />
      <g fill="#292929">
        <path d="M43 46h10v3h3v10h-3v3H43v-3h-3V49h3zM43 65h13v5h4v6h-5v-5h-3v11h-5V71h-3v7h-5V68h4z" />
        <path d="M69 42h10v3h3v10h-3v3H69v-3h-3V45h3zM69 61h12v5h4v9h-5v-8h-3v15h-5V68h-3v10h-5V65h5z" />
      </g>
      <path d="M44 50h8v7h-8zM70 46h8v7h-8z" fill="#f7f7f7" />
      <path d="M32 76h4v4h-4zm57-2h4v4h-4z" fill="#bbb" />
      <path d="M22 24h4v4h-4zm65-8h3v3h-3z" fill="#aaa" />
    </svg>
  );
}
