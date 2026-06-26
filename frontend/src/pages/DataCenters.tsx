/* Data Centers + colo fees page */
import { useState } from "react";
import { Icon } from "../components/icons";
import { Badge, Field, Modal, Panel, Stat } from "../components/ui";
import type { Ctx, DataCenter } from "../lib/types";

export function DataCentersPage({ ctx }: { ctx: Ctx }) {
  const { dcs, assets, derived, isAdmin, fmt, fmtFull, unit } = ctx;
  const [editing, setEditing] = useState<Partial<DataCenter> | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);

  const totalColo = dcs.reduce((s, d) => s + d.coloMonthly, 0);
  const totalRacks = dcs.reduce((s, d) => s + d.racks, 0);
  const totalKw = dcs.reduce((s, d) => s + d.power_kw, 0);

  return (
    <div className="content-inner">
      <div className="grid grid-4" style={{ marginBottom: 16 }}>
        <Stat label="Data centers" value={dcs.length} icon="datacenter" accent="var(--accent)" meta={<span>{assets.length} assets hosted</span>} />
        <Stat label="Total colo / mo" value={fmt(totalColo)} unit={unit} icon="money" accent="var(--warn)" meta={<span>{fmt(totalColo * 12)} {unit}/yr</span>} />
        <Stat label="Racks" value={totalRacks} icon="server" accent="var(--info)" meta={<span>across {dcs.length} facilities</span>} />
        <Stat label="Provisioned power" value={totalKw} unit="kW" icon="power" accent="var(--accent-2)" meta={<span>contracted capacity</span>} />
      </div>

      <div className="row" style={{ marginBottom: 14 }}>
        <div className="page-title" style={{ fontSize: 15 }}>Facilities</div>
        <div className="spacer" />
        {isAdmin && (
          <button className="btn primary" onClick={() => setEditing({})}>
            <Icon name="plus" size={15} /> Add data center
          </button>
        )}
      </div>

      <div className="grid grid-3">
        {dcs.map((dc) => {
          const dcAssets = assets.filter((a) => a.dcId === dc.id);
          const usedW = dcAssets.reduce((s, a) => s + a.powerWatts, 0);
          const usedKw = usedW / 1000;
          const powerPct = Math.min(100, (usedKw / dc.power_kw) * 100);
          const open = expanded === dc.id;
          return (
            <div className="panel" key={dc.id} style={{ gridColumn: open ? "1 / -1" : "auto" }}>
              <div className="panel-body">
                <div className="row" style={{ alignItems: "flex-start" }}>
                  <div className="brand-mark" style={{ background: "var(--info-bg)", color: "var(--info)", boxShadow: "none" }}>
                    <Icon name="datacenter" size={17} />
                  </div>
                  <div style={{ marginLeft: 4 }}>
                    <div style={{ fontWeight: 700, fontSize: 14.5 }}>{dc.name}</div>
                    <div className="cell-sub">
                      {dc.provider} · {dc.region}
                    </div>
                  </div>
                  <div className="spacer" />
                  {isAdmin && (
                    <button className="icon-btn" onClick={() => setEditing(dc)} title="Edit">
                      <Icon name="edit" size={15} />
                    </button>
                  )}
                </div>

                <div className="row gap-8" style={{ marginTop: 12, flexWrap: "wrap" }}>
                  <span className="tag">{dc.short}</span>
                  <Badge kind="neutral">{dc.tier}</Badge>
                  <Badge kind="info">{dcAssets.length} assets</Badge>
                </div>

                <div className="divider" />
                <div className="row" style={{ justifyContent: "space-between", marginBottom: 10 }}>
                  <div>
                    <div className="kpi-mini">Colo fee / month</div>
                    <div className="num" style={{ fontSize: 21, fontWeight: 700, color: "var(--warn)" }}>
                      {fmt(dc.coloMonthly)} <span className="faint" style={{ fontSize: 12 }}>{unit}</span>
                    </div>
                  </div>
                  <div className="right">
                    <div className="kpi-mini">Per asset avg</div>
                    <div className="num" style={{ fontSize: 14, fontWeight: 600 }}>
                      {dcAssets.length ? fmt(dc.coloMonthly / dcAssets.length) : "—"}
                    </div>
                  </div>
                </div>

                <div className="kpi-mini" style={{ marginBottom: 5, display: "flex", justifyContent: "space-between" }}>
                  <span>Power utilization</span>
                  <span className="mono">
                    {usedKw.toFixed(1)} / {dc.power_kw} kW
                  </span>
                </div>
                <div className="minibar" style={{ height: 8 }}>
                  <span style={{ width: powerPct + "%", background: powerPct > 85 ? "var(--bad)" : "var(--accent)" }} />
                </div>

                <button className="btn sm" style={{ marginTop: 14, width: "100%", justifyContent: "center" }} onClick={() => setExpanded(open ? null : dc.id)}>
                  <Icon name="chevronDown" size={14} style={{ transform: open ? "rotate(180deg)" : "none", transition: "transform .15s" }} />
                  {open ? "Hide" : "Show"} hosted assets &amp; colo split
                </button>

                {open && (
                  <div className="tbl-wrap" style={{ marginTop: 12 }}>
                    <table className="tbl">
                      <thead>
                        <tr>
                          <th>Asset</th>
                          <th>Type</th>
                          <th className="r">Power</th>
                          <th className="r">Power share</th>
                          <th className="r">Colo / mo</th>
                        </tr>
                      </thead>
                      <tbody>
                        {dcAssets.map((a) => {
                          const share = usedW ? a.powerWatts / usedW : 0;
                          return (
                            <tr key={a.id}>
                              <td className="cell-strong">{a.name}</td>
                              <td className="cell-sub">
                                {a.vendor} {a.model}
                              </td>
                              <td className="r cell-mono">{a.powerWatts}W</td>
                              <td className="r cell-mono">{(share * 100).toFixed(1)}%</td>
                              <td className="r cell-mono cell-strong" style={{ color: "var(--warn)" }}>
                                {fmt(derived.assetCosts[a.id]?.colo || 0)}
                              </td>
                            </tr>
                          );
                        })}
                        {dcAssets.length === 0 && (
                          <tr>
                            <td colSpan={5} className="empty">
                              No assets assigned to this facility yet.
                            </td>
                          </tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {editing && (
        <DCForm
          dc={editing}
          unit={unit}
          fmtFull={fmtFull}
          onClose={() => setEditing(null)}
          onSave={(d) => {
            ctx.upsertDc(d);
            setEditing(null);
          }}
          onDelete={
            editing.id
              ? () => {
                  ctx.deleteDc(editing.id!);
                  setEditing(null);
                }
              : null
          }
        />
      )}
    </div>
  );
}

function DCForm({ dc, onClose, onSave, onDelete, unit, fmtFull }: { dc: Partial<DataCenter>; onClose: () => void; onSave: (d: DataCenter) => void; onDelete: (() => void) | null; unit: string; fmtFull: (v: number) => string }) {
  const isNew = !dc.id;
  const [f, setF] = useState<DataCenter>(() => ({
    id: dc.id || "dc-" + Math.random().toString(36).slice(2, 6),
    name: dc.name || "",
    short: dc.short || "",
    provider: dc.provider || "",
    region: dc.region || "",
    tier: dc.tier || "Tier III",
    coloMonthly: dc.coloMonthly || 0,
    racks: dc.racks || 1,
    power_kw: dc.power_kw || 10,
  }));
  const set = <K extends keyof DataCenter>(k: K, v: DataCenter[K]) => setF((p) => ({ ...p, [k]: v }));
  return (
    <Modal
      title={isNew ? "Add data center" : "Edit " + f.name}
      onClose={onClose}
      footer={
        <>
          {onDelete && (
            <button className="btn danger" onClick={onDelete}>
              <Icon name="trash" size={14} /> Delete
            </button>
          )}
          <div className="spacer" />
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn primary" onClick={() => onSave(f)} disabled={!f.name}>
            {isNew ? "Add" : "Save"}
          </button>
        </>
      }
    >
      <div className="field-row">
        <Field label="Name">
          <input className="input" value={f.name} onChange={(e) => set("name", e.target.value)} placeholder="Tehran · Pardis" />
        </Field>
        <Field label="Short code">
          <input className="input mono" value={f.short} onChange={(e) => set("short", e.target.value)} placeholder="THR-1" style={{ maxWidth: 120 }} />
        </Field>
      </div>
      <div className="field-row">
        <Field label="Provider">
          <input className="input" value={f.provider} onChange={(e) => set("provider", e.target.value)} placeholder="Pardis IDC" />
        </Field>
        <Field label="Region">
          <input className="input" value={f.region} onChange={(e) => set("region", e.target.value)} placeholder="IR-Tehran" />
        </Field>
      </div>
      <Field label="Colo fee per month" hint={f.coloMonthly ? fmtFull(f.coloMonthly) + " " + unit + " · split across hosted assets by power draw" : "Monthly colocation invoice"}>
        <div className="input-affix">
          <input className="input mono" type="number" value={f.coloMonthly} onChange={(e) => set("coloMonthly", +e.target.value)} />
          <span className="affix">{unit}</span>
        </div>
      </Field>
      <div className="field-row">
        <Field label="Tier">
          <select className="select" value={f.tier} onChange={(e) => set("tier", e.target.value)}>
            <option>Tier I</option>
            <option>Tier II</option>
            <option>Tier III</option>
            <option>Tier IV</option>
          </select>
        </Field>
        <Field label="Racks">
          <input className="input mono" type="number" value={f.racks} onChange={(e) => set("racks", +e.target.value)} />
        </Field>
        <Field label="Power (kW)">
          <input className="input mono" type="number" value={f.power_kw} onChange={(e) => set("power_kw", +e.target.value)} />
        </Field>
      </div>
    </Modal>
  );
}
