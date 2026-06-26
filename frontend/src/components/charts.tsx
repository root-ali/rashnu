/* rashnnu · lightweight responsive SVG charts */
import React, { useEffect, useId, useRef, useState } from "react";

export function useMeasure(): [React.RefObject<HTMLDivElement>, number] {
  const ref = useRef<HTMLDivElement>(null);
  const [w, setW] = useState(600);
  useEffect(() => {
    if (!ref.current) return;
    const ro = new ResizeObserver((entries) => {
      const cw = entries[0].contentRect.width;
      if (cw > 0) setW(cw);
    });
    ro.observe(ref.current);
    return () => ro.disconnect();
  }, []);
  return [ref, w];
}

export interface TrendDatum {
  month: string;
  value: number;
}
export interface BarDatum {
  label: string;
  value: number;
  color?: string;
}
export interface SeriesDatum {
  label: string;
  value: number;
  color?: string;
  sub?: string;
}

type Fmt = (v: number) => React.ReactNode;
const ident: Fmt = (v) => v;

// ---------- Area / line trend ----------
export function AreaChart({ data, height = 180, color = "var(--accent)", fmt = ident }: { data: TrendDatum[]; height?: number; color?: string; fmt?: Fmt }) {
  const [ref, W] = useMeasure();
  const gid = "ag" + useId().replace(/:/g, "");
  const [hover, setHover] = useState<number | null>(null);
  if (!data.length) {
    return (
      <div ref={ref} className="faint" style={{ padding: "24px 8px", fontSize: 13 }}>
        No trend data yet
      </div>
    );
  }
  const padL = 8,
    padR = 8,
    padT = 14,
    padB = 22;
  const vals = data.map((d) => d.value);
  const max = Math.max(...vals, 1) * 1.08,
    min = Math.min(...vals, 0) * 0.9;
  const innerW = W - padL - padR,
    innerH = height - padT - padB;
  const x = (i: number) => padL + (innerW * i) / Math.max(data.length - 1, 1);
  const y = (v: number) => padT + innerH - (innerH * (v - min)) / (max - min || 1);
  const linePts = data.map((d, i) => `${x(i)},${y(d.value)}`).join(" ");
  const areaPts = `${padL},${padT + innerH} ${linePts} ${padL + innerW},${padT + innerH}`;
  return (
    <div ref={ref} style={{ width: "100%", position: "relative" }}>
      <svg
        width={W}
        height={height}
        style={{ display: "block" }}
        onMouseLeave={() => setHover(null)}
        onMouseMove={(e) => {
          const rect = e.currentTarget.getBoundingClientRect();
          const px = e.clientX - rect.left;
          let best = 0,
            bd = 1e9;
          data.forEach((_, i) => {
            const dd = Math.abs(x(i) - px);
            if (dd < bd) {
              bd = dd;
              best = i;
            }
          });
          setHover(best);
        }}
      >
        <defs>
          <linearGradient id={gid} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity="0.32" />
            <stop offset="100%" stopColor={color} stopOpacity="0.02" />
          </linearGradient>
        </defs>
        {[0, 0.5, 1].map((g, i) => (
          <line key={i} x1={padL} x2={padL + innerW} y1={padT + innerH * g} y2={padT + innerH * g} stroke="var(--grid-line)" strokeWidth="1" />
        ))}
        <polygon points={areaPts} fill={`url(#${gid})`} />
        <polyline points={linePts} fill="none" stroke={color} strokeWidth="2" strokeLinejoin="round" />
        {data.map((d, i) => (
          <text key={i} x={x(i)} y={height - 6} textAnchor="middle" fontSize="10" fill="var(--text-3)" fontFamily="var(--font-mono)">
            {d.month}
          </text>
        ))}
        {hover != null && (
          <>
            <line x1={x(hover)} x2={x(hover)} y1={padT} y2={padT + innerH} stroke={color} strokeWidth="1" strokeDasharray="3 3" opacity="0.6" />
            <circle cx={x(hover)} cy={y(data[hover].value)} r="4" fill={color} stroke="var(--panel)" strokeWidth="2" />
          </>
        )}
      </svg>
      {hover != null && (
        <div
          style={{
            position: "absolute",
            left: Math.min(W - 110, Math.max(0, x(hover) - 50)),
            top: 2,
            background: "var(--bg-3)",
            border: "1px solid var(--border-2)",
            borderRadius: 6,
            padding: "5px 9px",
            fontSize: 11.5,
            pointerEvents: "none",
            boxShadow: "var(--shadow)",
            whiteSpace: "nowrap",
          }}
        >
          <span className="muted">{data[hover].month} · </span>
          <span className="mono" style={{ fontWeight: 700 }}>
            {fmt(data[hover].value)}
          </span>
        </div>
      )}
    </div>
  );
}

// ---------- Vertical bars ----------
export function BarChart({ data, height = 180, color = "var(--accent)", fmt = ident }: { data: BarDatum[]; height?: number; color?: string; fmt?: Fmt }) {
  const [ref, W] = useMeasure();
  const padT = 14,
    padB = 22,
    padX = 6;
  const [hover, setHover] = useState<number | null>(null);
  const max = Math.max(...data.map((d) => d.value)) * 1.05 || 1;
  const innerH = height - padT - padB;
  const n = data.length;
  const gap = 6;
  const bw = (W - padX * 2 - gap * (n - 1)) / n;
  return (
    <div ref={ref} style={{ width: "100%", position: "relative" }}>
      <svg width={W} height={height} style={{ display: "block" }}>
        {[0, 0.5, 1].map((g, i) => (
          <line key={i} x1={padX} x2={W - padX} y1={padT + innerH * g} y2={padT + innerH * g} stroke="var(--grid-line)" strokeWidth="1" />
        ))}
        {data.map((d, i) => {
          const h = (innerH * d.value) / max;
          const xx = padX + i * (bw + gap);
          return (
            <g key={i} onMouseEnter={() => setHover(i)} onMouseLeave={() => setHover(null)}>
              <rect x={xx} y={padT + innerH - h} width={bw} height={h} rx="3" fill={hover === i ? color : `color-mix(in oklab, ${color} 78%, transparent)`} />
              <text x={xx + bw / 2} y={height - 6} textAnchor="middle" fontSize="10" fill="var(--text-3)" fontFamily="var(--font-mono)">
                {d.label}
              </text>
            </g>
          );
        })}
      </svg>
      {hover != null && (
        <div style={{ position: "absolute", left: padX + hover * (bw + gap) + bw / 2 - 50, top: 0, width: 100, textAlign: "center", pointerEvents: "none" }}>
          <span style={{ background: "var(--bg-3)", border: "1px solid var(--border-2)", borderRadius: 6, padding: "3px 7px", fontSize: 11.5, boxShadow: "var(--shadow)", fontWeight: 700 }} className="mono">
            {fmt(data[hover].value)}
          </span>
        </div>
      )}
    </div>
  );
}

// ---------- Donut ----------
export const CHART_COLORS = ["var(--accent)", "var(--accent-2)", "var(--purple)", "var(--warn)", "var(--good)", "var(--info)", "var(--bad)"];

export function Donut({
  data,
  size = 170,
  thickness = 22,
  fmt = ident,
  centerLabel,
  centerValue,
}: {
  data: SeriesDatum[];
  size?: number;
  thickness?: number;
  fmt?: Fmt;
  centerLabel?: React.ReactNode;
  centerValue?: React.ReactNode;
}) {
  const total = data.reduce((s, d) => s + d.value, 0) || 1;
  const r = (size - thickness) / 2;
  const cx = size / 2,
    cy = size / 2;
  const circ = 2 * Math.PI * r;
  let offset = 0;
  const [hover, setHover] = useState<number | null>(null);
  return (
    <div className="row" style={{ gap: 18, alignItems: "center" }}>
      <svg width={size} height={size} style={{ flexShrink: 0, transform: "rotate(-90deg)" }}>
        {data.map((d, i) => {
          const frac = d.value / total;
          const len = circ * frac;
          const seg = (
            <circle
              key={i}
              cx={cx}
              cy={cy}
              r={r}
              fill="none"
              stroke={d.color || CHART_COLORS[i % CHART_COLORS.length]}
              strokeWidth={hover === i ? thickness + 4 : thickness}
              strokeDasharray={`${len} ${circ - len}`}
              strokeDashoffset={-offset}
              onMouseEnter={() => setHover(i)}
              onMouseLeave={() => setHover(null)}
              style={{ transition: "stroke-width .12s", cursor: "pointer" }}
            />
          );
          offset += len;
          return seg;
        })}
        <text x={cx} y={cy - 4} textAnchor="middle" fontSize="20" fontWeight="700" fill="var(--text)" fontFamily="var(--font-mono)" transform={`rotate(90 ${cx} ${cy})`}>
          {hover != null ? fmt(data[hover].value) : centerValue || ""}
        </text>
        <text x={cx} y={cy + 14} textAnchor="middle" fontSize="10" fill="var(--text-2)" transform={`rotate(90 ${cx} ${cy})`}>
          {hover != null ? data[hover].label : centerLabel || ""}
        </text>
      </svg>
      <div className="legend" style={{ flexDirection: "column", gap: 7 }}>
        {data.map((d, i) => (
          <div className="li" key={i} style={{ color: hover === i ? "var(--text)" : "var(--text-2)" }} onMouseEnter={() => setHover(i)} onMouseLeave={() => setHover(null)}>
            <span className="sw" style={{ background: d.color || CHART_COLORS[i % CHART_COLORS.length] }} />
            <span style={{ minWidth: 90 }}>{d.label}</span>
            <span className="mono" style={{ fontWeight: 600, color: "var(--text)" }}>
              {fmt(d.value)}
            </span>
            <span className="faint mono" style={{ fontSize: 11 }}>
              {((d.value / total) * 100).toFixed(0)}%
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ---------- Horizontal bar ranking ----------
export function HBarList({ data, fmt = ident, color = "var(--accent)", max }: { data: SeriesDatum[]; fmt?: Fmt; color?: string; max?: number }) {
  const mx = max || Math.max(...data.map((d) => d.value)) || 1;
  return (
    <div className="col" style={{ gap: 11 }}>
      {data.map((d, i) => (
        <div key={i}>
          <div className="row" style={{ marginBottom: 4 }}>
            <span style={{ fontSize: 12.5, fontWeight: 500 }}>{d.label}</span>
            {d.sub && <span className="faint" style={{ fontSize: 11 }}>&nbsp;· {d.sub}</span>}
            <div className="spacer" />
            <span className="mono" style={{ fontSize: 12.5, fontWeight: 600 }}>
              {fmt(d.value)}
            </span>
          </div>
          <div className="minibar" style={{ height: 8 }}>
            <span style={{ width: (d.value / mx) * 100 + "%", background: d.color || color }} />
          </div>
        </div>
      ))}
    </div>
  );
}

// ---------- Stacked horizontal bar (composition) ----------
export function StackedBar({ segments, height = 10 }: { segments: SeriesDatum[]; height?: number }) {
  const total = segments.reduce((s, d) => s + d.value, 0) || 1;
  return (
    <div className="bar-track" style={{ height }}>
      {segments.map((s, i) => (
        <span key={i} title={s.label} style={{ width: (s.value / total) * 100 + "%", background: s.color || CHART_COLORS[i % CHART_COLORS.length] }} />
      ))}
    </div>
  );
}
