import { memo, useEffect, useRef, useId } from "react";
import { useReducedMotion } from "motion/react";
import { useStore } from "../lib/store";

const sprite = [
  "     1111111111        ",
  "   11111111111111      ",
  "  1122222222222111     ",
  "  11222222222221111    ",
  "  113333133333111211   ",
  "  1133331333331112211  ",
  "  11333313333311122211 ",
  "  11111111111111111111 ",
  "  11222222222221122221 ",
  "  11222222222221122221 ",
  "  11222222222221122221 ",
  "1111111111111111111111 ",
  "1111444111111111444111 ",
  "  1444441111111444441  ",
  "  1442441     1442441  ",
  "   14441       14441   ",
];
const colors = ["transparent", "#202020", "#e9e9e9", "#898989", "#111111"];
export function Rickshaw({
  className = "",
  label,
}: {
  className?: string;
  label?: string;
}) {
  return (
    <svg
      className={`rickshaw ${className}`}
      viewBox="0 0 24 17"
      role={label ? "img" : undefined}
      aria-label={label}
      aria-hidden={!label}
      shapeRendering="crispEdges"
    >
      {sprite.flatMap((row, y) =>
        [...row].map((value, x) =>
          value !== " " ? (
            <rect
              key={`${x}-${y}`}
              x={x}
              y={y}
              width="1"
              height="1"
              fill={colors[Number(value)]}
            />
          ) : null,
        ),
      )}
    </svg>
  );
}
export function PixelMark({ className = "" }: { className?: string }) {
  return (
    <svg
      className={className}
      viewBox="0 0 24 24"
      width="24"
      height="24"
      fill="currentColor"
      aria-hidden="true"
      shapeRendering="crispEdges"
    >
      <path d="M6 2h12v4h4v12h-4v4H6v-4H2V6h4zm2 6v8h8V8z" fillRule="evenodd" />
      <rect x="18" y="18" width="6" height="6" />
    </svg>
  );
}
export function Avatar({ name, size = 40 }: { name: string; size?: number }) {
  const seed = [...name].reduce((sum, c) => sum + c.charCodeAt(0), 0);
  return (
    <span
      className="avatar"
      style={{ width: size, height: size }}
      role="img"
      aria-label={`${name}'s pixel avatar`}
    >
      <svg viewBox="0 0 10 10" shapeRendering="crispEdges" aria-hidden="true">
        <rect width="10" height="10" fill="#e5e5e5" />
        {Array.from({ length: 5 }, (_, y) =>
          Array.from({ length: 3 }, (_, x) => {
            const filled = ((seed * (y + 3) * (x + 5)) ^ (seed >> x)) % 3 !== 0;
            return filled ? (
              <g key={`${x}-${y}`} fill={y === 0 ? "#656565" : "#292929"}>
                <rect x={x + 2} y={y + 2} width="1" height="1" />
                <rect x={7 - x} y={y + 2} width="1" height="1" />
              </g>
            ) : null;
          }),
        )}
      </svg>
    </span>
  );
}

// Seeded geometry is painted once. The animation loop only composites cached art and moving sprites.
export const PixelCity = memo(function PixelCity({
  compact = false,
}: {
  compact?: boolean;
}) {
  const ref = useRef<HTMLCanvasElement>(null);
  const reduced = useReducedMotion();
  const motion = useStore((s) => s.motion);
  useEffect(() => {
    const canvas = ref.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    const width = 800,
      height = 310;
    canvas.width = width;
    canvas.height = height;
    const background = document.createElement("canvas");
    background.width = width;
    background.height = height;
    const b = background.getContext("2d")!;
    b.imageSmoothingEnabled = false;
    const iso = (x: number, y: number) => [
      400 + (x - y) * 24,
      -70 + (x + y) * 12,
    ];
    const poly = (points: number[][], fill: string) => {
      b.fillStyle = fill;
      b.beginPath();
      points.forEach(([x, y], i) => (i ? b.lineTo(x, y) : b.moveTo(x, y)));
      b.closePath();
      b.fill();
    };
    for (let sum = 0; sum < 35; sum++)
      for (let x = 0; x < 18; x++) {
        const y = sum - x;
        if (y < 0 || y > 17) continue;
        const [px, py] = iso(x, y);
        const road = x === 7 || x === 8 || y === 9 || y === 10;
        poly(
          [
            [px, py],
            [px + 24, py + 12],
            [px, py + 24],
            [px - 24, py + 12],
          ],
          road ? "#b2b2b2" : (x * 5 + y) % 4 === 0 ? "#e2e2e2" : "#ececec",
        );
        if (road && (x + y) % 2 === 0) {
          b.fillStyle = "#f8f8f8";
          b.fillRect(px - 3, py + 10, 6, 2);
        }
      }
    const building = (
      x: number,
      y: number,
      w: number,
      d: number,
      h: number,
      shade: string,
    ) => {
      const [px, py] = iso(x, y);
      const [rightX, rightY] = iso(x + w, y);
      const [bottomX, bottomY] = iso(x + w, y + d);
      const [leftX, leftY] = iso(x, y + d);
      poly(
        [
          [leftX, leftY],
          [bottomX, bottomY],
          [bottomX + 9, bottomY + 4],
          [leftX + 10, leftY + 7],
        ],
        "#d1d1d1",
      );
      poly(
        [
          [px, py - h],
          [leftX, leftY - h],
          [bottomX, bottomY - h],
          [rightX, rightY - h],
        ],
        shade,
      );
      poly(
        [
          [leftX, leftY - h],
          [bottomX, bottomY - h],
          [bottomX, bottomY],
          [leftX, leftY],
        ],
        "#a7a7a7",
      );
      poly(
        [
          [bottomX, bottomY - h],
          [rightX, rightY - h],
          [rightX, rightY],
          [bottomX, bottomY],
        ],
        "#d3d3d3",
      );
      b.fillStyle = "#f5f5f5";
      for (let j = 0; j < Math.floor(h / 12); j++)
        for (let k = 0; k < Math.floor(w * 2); k++)
          b.fillRect(
            bottomX + 5 + k * 8,
            bottomY - h + 8 + j * 11 - k * 4,
            3,
            4,
          );
      b.fillStyle = "#636363";
      b.fillRect(bottomX - 8, bottomY - 10, 5, 9);
      if (h > 35) {
        b.fillStyle = "#737373";
        b.fillRect(px - 3, py - h - 6, 9, 6);
        b.fillStyle = "#ededed";
        b.fillRect(px, py - h - 6, 6, 2);
      }
    };
    const tree = (x: number, y: number) => {
      const [px, py] = iso(x, y);
      b.fillStyle = "#727272";
      b.fillRect(px, py - 28, 3, 36);
      b.fillStyle = "#404040";
      [
        [-18, -30, 14, 4],
        [-12, -34, 12, 5],
        [2, -34, 16, 4],
        [8, -29, 16, 4],
        [-5, -39, 4, 9],
        [-21, -25, 6, 4],
        [20, -24, 5, 4],
      ].forEach(([dx, dy, w, h]) => b.fillRect(px + dx, py + dy, w, h));
    };
    for (let sum = 1; sum < 32; sum++)
      for (let x = 1; x < 16; x += 3) {
        const y = sum - x;
        if (
          y < 1 ||
          y > 15 ||
          y % 3 !== 1 ||
          [7, 8].includes(x) ||
          [7, 8, 9, 10].includes(y)
        )
          continue;
        if ((x + y) % 4 === 0) tree(x, y);
        else
          building(
            x,
            y,
            1.6,
            1.35,
            16 + ((x * 37 + y * 13) % 48),
            (x + y) % 3 ? "#f9f9f9" : "#bababa",
          );
      }
    tree(6.1, 8.2);
    tree(11, 8.3);
    tree(9.4, 13);
    let frame = 0,
      last = 0,
      elapsed = 0,
      visible = true,
      previous = 0;
    const draw = () => {
      ctx.clearRect(0, 0, width, height);
      ctx.imageSmoothingEnabled = false;
      ctx.drawImage(background, 0, 0);
      [
        [(elapsed * 0.0006) % 16, 9.7],
        [7.5, 16 - ((elapsed * 0.00042) % 16)],
      ].forEach(([x, y], i) => {
        const [px, py] = iso(x, y);
        ctx.fillStyle = "#9e9e9e";
        ctx.fillRect(px - 9, py + 12, 24, 4);
        sprite.forEach((row, sy) =>
          [...row].forEach((v, sx) => {
            if (v !== " ") {
              ctx.fillStyle = colors[Number(v)];
              ctx.fillRect(
                Math.round(px - 10 + sx),
                Math.round(
                  py - 5 + sy + (i ? 0 : Math.sin(elapsed / 160) > 0 ? 1 : 0),
                ),
                1,
                1,
              );
            }
          }),
        );
      });
    };
    // No animation work survives an offscreen canvas, hidden tab, or motion preference.
    let running = false;
    const frameBudget = 1000 / (window.innerWidth < 700 ? 18 : 24);
    const canAnimate = () => visible && !document.hidden && !reduced && motion;
    const tick = (time: number) => {
      if (!canAnimate()) {
        running = false;
        previous = 0;
        return;
      }
      if (time - last >= frameBudget) {
        elapsed += previous ? Math.min(time - previous, 90) : 0;
        previous = time;
        last = time;
        draw();
      }
      frame = requestAnimationFrame(tick);
    };
    const syncAnimation = () => {
      if (!canAnimate()) {
        cancelAnimationFrame(frame);
        running = false;
        previous = 0;
      } else if (!running) {
        running = true;
        frame = requestAnimationFrame(tick);
      }
    };
    draw();
    syncAnimation();
    const observer = new IntersectionObserver((entries) => {
      visible = entries[0].isIntersecting;
      syncAnimation();
    });
    observer.observe(canvas);
    document.addEventListener("visibilitychange", syncAnimation);
    return () => {
      cancelAnimationFrame(frame);
      observer.disconnect();
      document.removeEventListener("visibilitychange", syncAnimation);
    };
  }, [reduced, motion]);
  return (
    <div className={`pixel-city ${compact ? "compact" : ""}`}>
      <canvas
        ref={ref}
        aria-label="A tiny isometric Kochi neighbourhood, with pixel autorickshaws travelling along shared streets"
        role="img"
      />
      <div className="city-fade" />
    </div>
  );
});
export function RouteArt({
  from = "Kakkanad",
  to = "Edappally",
  small = false,
}: {
  from?: string;
  to?: string;
  small?: boolean;
}) {
  const patternId = useId();
  return (
    <div
      className={`route-art ${small ? "small" : ""}`}
      aria-label={`Illustrative route from ${from} to ${to}; not a navigation map`}
      role="img"
    >
      <svg
        viewBox="0 0 600 300"
        aria-hidden="true"
        preserveAspectRatio="xMidYMid slice"
      >
        <defs>
          <pattern
            id={patternId}
            width="80"
            height="65"
            patternUnits="userSpaceOnUse"
          >
            <path d="M0 12h58v37H0z" fill="#e1e1e1" />
            <path d="M8 20h42v21H8z" fill="#ececec" />
          </pattern>
        </defs>
        <rect width="600" height="300" fill="#f6f6f6" />
        <rect width="600" height="300" fill={`url(#${patternId})`} />
        <path
          d="M0 220h175v-85h145V68h280"
          stroke="#fff"
          strokeWidth="28"
          fill="none"
        />
        <path
          className="route-dash"
          d="M95 220h80v-85h145V68h175"
          stroke="#222"
          strokeWidth="4"
          strokeDasharray="5 5"
          fill="none"
        />
        <rect x="86" y="211" width="18" height="18" fill="#222" />
        <rect
          x="488"
          y="61"
          width="14"
          height="14"
          fill="#fff"
          stroke="#222"
          strokeWidth="4"
        />
      </svg>
      <span className="map-label origin">{from}</span>
      <span className="map-label destination">{to}</span>
      <Rickshaw className="map-auto" />
      <span className="map-disclaimer">Illustrated route</span>
    </div>
  );
}
