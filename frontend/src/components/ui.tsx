/* rashnnu · shared UI primitives */
import React, { useEffect, useState } from "react";
import { Icon, type IconName } from "./icons";
import type { Crit, Platform, ToastKind } from "../lib/types";

export type BadgeKind = "good" | "warn" | "bad" | "info" | "purple" | "neutral";

export function Badge({ kind = "neutral", children, dot }: { kind?: BadgeKind; children: React.ReactNode; dot?: boolean }) {
  return (
    <span className={"badge " + kind}>
      {dot && <span className="dot" />}
      {children}
    </span>
  );
}

export function Panel({
  title,
  sub,
  right,
  children,
  style,
  bodyStyle,
  noBody,
}: {
  title?: React.ReactNode;
  sub?: React.ReactNode;
  right?: React.ReactNode;
  children: React.ReactNode;
  style?: React.CSSProperties;
  bodyStyle?: React.CSSProperties;
  noBody?: boolean;
}) {
  return (
    <div className="panel" style={style}>
      {(title || right) && (
        <div className="panel-head">
          <div>
            {title && <div className="panel-title">{title}</div>}
            {sub && <div className="panel-sub">{sub}</div>}
          </div>
          <div className="spacer" />
          {right}
        </div>
      )}
      {noBody ? (
        children
      ) : (
        <div className="panel-body" style={bodyStyle}>
          {children}
        </div>
      )}
    </div>
  );
}

export function Stat({
  label,
  value,
  unit,
  icon,
  accent,
  meta,
  delta,
}: {
  label: React.ReactNode;
  value: React.ReactNode;
  unit?: React.ReactNode;
  icon?: IconName;
  accent?: string;
  meta?: React.ReactNode;
  delta?: number;
}) {
  return (
    <div className="stat">
      {accent && <div className="accent-bar" style={{ background: accent }} />}
      <div className="label">
        {icon && <Icon name={icon} size={14} />} {label}
      </div>
      <div className="value">
        {value}
        {unit && <span className="u">{unit}</span>}
      </div>
      {(meta || delta != null) && (
        <div className="meta">
          {delta != null && (
            <span className={"delta " + (delta >= 0 ? "up" : "down")}>
              <Icon name={delta >= 0 ? "arrowUp" : "arrowDown"} size={12} style={{ verticalAlign: "-2px" }} />
              {Math.abs(delta).toFixed(1)}%
            </span>
          )}
          {meta}
        </div>
      )}
    </div>
  );
}

export function Modal({
  title,
  children,
  onClose,
  footer,
  wide,
}: {
  title: React.ReactNode;
  children: React.ReactNode;
  onClose: () => void;
  footer?: React.ReactNode;
  wide?: boolean;
}) {
  useEffect(() => {
    const h = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  return (
    <div className="modal-overlay" onMouseDown={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <div className={"modal" + (wide ? " wide" : "")}>
        <div className="modal-head">
          <h3>{title}</h3>
          <div className="spacer" />
          <button className="icon-btn" onClick={onClose}>
            <Icon name="close" size={18} />
          </button>
        </div>
        <div className="modal-body">{children}</div>
        {footer && <div className="modal-foot">{footer}</div>}
      </div>
    </div>
  );
}

export function Field({ label, hint, children }: { label?: React.ReactNode; hint?: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="field">
      {label && <label>{label}</label>}
      {children}
      {hint && <div className="hint">{hint}</div>}
    </div>
  );
}

export function UtilChip({ value }: { value: number }) {
  const pct = value * 100;
  let color = "var(--good)",
    bg = "var(--good-bg)";
  if (pct >= 85) {
    color = "var(--bad)";
    bg = "var(--bad-bg)";
  } else if (pct >= 65) {
    color = "var(--warn)";
    bg = "var(--warn-bg)";
  }
  return (
    <span className="util-chip" style={{ color, background: bg }}>
      {pct.toFixed(0)}%
    </span>
  );
}

export function UtilBar({ value, width = 80 }: { value: number; width?: number }) {
  const pct = Math.min(100, value * 100);
  let color = "var(--good)";
  if (pct >= 85) color = "var(--bad)";
  else if (pct >= 65) color = "var(--warn)";
  return (
    <div className="row gap-8">
      <div className="minibar" style={{ width }}>
        <span style={{ width: pct + "%", background: color }} />
      </div>
      <span className="cell-mono" style={{ fontSize: 11.5, color: "var(--text-2)", minWidth: 32 }}>
        {pct.toFixed(0)}%
      </span>
    </div>
  );
}

export function PlatformBadge({ platform }: { platform: Platform }) {
  return platform === "kubernetes" ? (
    <Badge kind="info">
      <Icon name="k8s" size={12} /> K8s
    </Badge>
  ) : (
    <Badge kind="purple">
      <Icon name="vm" size={12} /> VM
    </Badge>
  );
}

export function CritBadge({ crit }: { crit: Crit }) {
  const map: Record<string, BadgeKind> = { critical: "bad", high: "warn", med: "info", low: "neutral" };
  const label: Record<string, string> = { critical: "Critical", high: "High", med: "Medium", low: "Low" };
  return <Badge kind={map[crit] || "neutral"}>{label[crit] || crit}</Badge>;
}

export interface ToastItem {
  id: string;
  msg: string;
  kind?: ToastKind;
}

export function Toasts({ items }: { items: ToastItem[] }) {
  return (
    <div className="toast-wrap">
      {items.map((t) => (
        <div className="toast" key={t.id} style={{ borderLeftColor: t.kind === "error" ? "var(--bad)" : "var(--good)" }}>
          <Icon name={t.kind === "error" ? "alert" : "check"} size={15} style={{ color: t.kind === "error" ? "var(--bad)" : "var(--good)" }} />
          {t.msg}
        </div>
      ))}
    </div>
  );
}

export interface SortState {
  col: string;
  dir: "asc" | "desc";
}

export function Th({ label, col, sort, setSort, align }: { label: React.ReactNode; col: string; sort: SortState; setSort: (s: SortState) => void; align?: "r" }) {
  const active = sort.col === col;
  return (
    <th className={"sortable " + (align === "r" ? "r" : "")} onClick={() => setSort({ col, dir: active && sort.dir === "desc" ? "asc" : "desc" })}>
      <span style={{ display: "inline-flex", alignItems: "center", gap: 4 }}>
        {label}
        {active && <Icon name={sort.dir === "desc" ? "arrowDown" : "arrowUp"} size={11} style={{ color: "var(--accent)" }} />}
      </span>
    </th>
  );
}

export type SortApply = <T>(arr: T[], accessors: Record<string, (x: T) => string | number>) => T[];

export function useSort(initial: SortState): [SortState, (s: SortState) => void, SortApply] {
  const [sort, setSort] = useState<SortState>(initial);
  const apply = (<T,>(arr: T[], accessors: Record<string, (x: T) => string | number>): T[] => {
    const acc = accessors[sort.col];
    if (!acc) return arr;
    const sorted = [...arr].sort((a, b) => {
      const va = acc(a),
        vb = acc(b);
      if (typeof va === "string") return va.localeCompare(vb as string);
      return (va as number) - (vb as number);
    });
    return sort.dir === "desc" ? sorted.reverse() : sorted;
  }) as SortApply;
  return [sort, setSort, apply];
}
