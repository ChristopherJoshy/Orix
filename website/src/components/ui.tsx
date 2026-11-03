import type { ButtonHTMLAttributes, ReactNode } from "react";
import { ArrowUpRight, ArrowRight, Check, ChevronRight } from "lucide-react";
import { Link } from "react-router-dom";
import { Avatar, PixelMark } from "./PixelArt";
import { poolLabel } from "../lib/data";
import type { Ride } from "../lib/types";
export function Button({
  children,
  className = "",
  variant = "primary",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "secondary" | "ghost" | "danger";
}) {
  return (
    <button className={`button ${variant} ${className}`} {...props}>
      {children}
    </button>
  );
}
export function Logo({ inverted = false }: { inverted?: boolean }) {
  return (
    <Link
      to="/"
      className={`logo ${inverted ? "inverted" : ""}`}
      aria-label="ORIX home"
    >
      <img
        className="brand-wordmark"
        src="/orix-logo.svg"
        width="119"
        height="33"
        alt=""
      />
    </Link>
  );
}

export function LinkButton({
  to,
  children,
  secondary = false,
  className = "",
}: {
  to: string;
  children: ReactNode;
  secondary?: boolean;
  className?: string;
}) {
  return (
    <Link
      to={to}
      className={`button ${secondary ? "secondary" : "primary"} ${className}`}
    >
      {children}
    </Link>
  );
}
export function Badge({
  children,
  dark = false,
}: {
  children: ReactNode;
  dark?: boolean;
}) {
  return (
    <span className={`badge ${dark ? "dark" : ""}`}>
      <span className="status-pixel" />
      {children}
    </span>
  );
}
export function PageHead({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle?: string;
  children?: ReactNode;
}) {
  return (
    <div className="page-head">
      <div>
        {subtitle && <p className="eyebrow">{subtitle}</p>}
        <h1>{title}</h1>
      </div>
      {children}
    </div>
  );
}
export function Empty({
  title,
  text,
  children,
}: {
  title: string;
  text: string;
  children?: ReactNode;
}) {
  return (
    <div className="empty">
      <PixelMark className="empty-brand" />
      <h2>{title}</h2>
      <p>{text}</p>
      {children}
    </div>
  );
}
export function Field({
  label,
  children,
  hint,
}: {
  label: string;
  children: ReactNode;
  hint?: string;
}) {
  return (
    <label className="field">
      <span>{label}</span>
      {children}
      {hint && <small>{hint}</small>}
    </label>
  );
}
export function Toggle({
  label,
  description,
  checked,
  onChange,
}: {
  label: string;
  description?: string;
  checked: boolean;
  onChange: (value: boolean) => void;
}) {
  return (
    <div className="setting-row">
      <div>
        <strong>{label}</strong>
        {description && <p>{description}</p>}
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-label={label}
        className={`toggle ${checked ? "on" : ""}`}
        onClick={() => onChange(!checked)}
      >
        <span>{checked && <Check size={12} />}</span>
      </button>
    </div>
  );
}
export function RideCard({
  ride,
  children,
}: {
  ride: Ride;
  children?: ReactNode;
}) {
  return (
    <article className="ride-card">
      <div className="ride-card-top">
        <span className="mode-icon">
          <PixelMark className="pool-mark" />
        </span>
        <Badge>{poolLabel(ride.mode)}</Badge>
        <span className="muted ride-date">
          {new Date(`${ride.date}T12:00`).toLocaleDateString("en-IN", {
            day: "numeric",
            month: "short",
          })}
        </span>
      </div>
      <div className="journey">
        <div className="journey-line">
          <i />
          <span />
          <i />
        </div>
        <div>
          <strong>{ride.from}</strong>
          <p>to</p>
          <strong>{ride.to}</strong>
        </div>
        <div className="ride-time">
          <strong>{ride.time}</strong>
          <small>Departure</small>
        </div>
      </div>
      <div className="ride-card-bottom">
        <div className="avatar-stack">
          {ride.members.slice(0, 3).map((name) => (
            <Avatar key={name} name={name} size={28} />
          ))}
          <small>
            {ride.members.length}/{ride.seats} together
          </small>
        </div>
        {children || (
          <Link to={`/app/rides/${ride.id}`} className="text-link">
            View ride <ArrowUpRight size={16} />
          </Link>
        )}
      </div>
      {ride.womenOnly && (
        <small className="ride-safety">Women-only group</small>
      )}
    </article>
  );
}
export function SectionLink({
  to,
  children,
}: {
  to: string;
  children: ReactNode;
}) {
  return (
    <Link className="section-link" to={to}>
      {children}
      <ChevronRight size={16} />
    </Link>
  );
}
export { ArrowUpRight, ArrowRight };
