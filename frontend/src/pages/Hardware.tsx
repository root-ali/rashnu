/* Hardware / Assets inventory page */
import { useState } from "react";
import { Icon } from "../components/icons";
import { Badge, type BadgeKind, Field, Modal, Panel, Stat, Th, useSort } from "../components/ui";
import { IC } from "../lib/data";
import type { Asset, AssetRole, AssetType, Ctx, DataCenter } from "../lib/types";

export function HardwarePage({ ctx }: { ctx: Ctx }) {
  const { assets, dcs, derived, isAdmin, fmt, fmtFull, unit } = ctx;
  const [q, setQ] = useState("");
  const [typeF, setTypeF] = useState<string>("all");
  const [dcF, setDcF] = useState<string>("all");
  const [editing, setEditing] = useState<Partial<Asset> | null>(null);
  const [sort, setSort, applySort] = useSort({ col: "total", dir: "desc" });

  const dcName = (id: string) => dcs.find((d) => d.id === id)?.short || "—";
  const costOf = (id: string) => derived.assetCosts[id] || { depreciation: 0, colo: 0, total: 0 };

  let rows = assets.filter((a) => {
    if (typeF !== "all" && a.type !== typeF) return false;
    if (dcF !== "all" && a.dcId !== dcF) return false;
    if (q && !(a.name + a.vendor + a.model).toLowerCase().includes(q.toLowerCase())) return false;
    return true;
  });
  rows = applySort(rows, {
    name: (a) => a.name,
    type: (a) => a.type,
    dc: (a) => dcName(a.dcId),
    purchase: (a) => a.purchaseAmount,
    depr: (a) => costOf(a.id).depreciation,
    colo: (a) => costOf(a.id).colo,
    total: (a) => costOf(a.id).total,
    amort: (a) => IC.amortRemaining(a),
  });

  const sums = rows.reduce(
    (s, a) => {
      const c = costOf(a.id);
      s.capex += a.purchaseAmount;
      s.depr += c.depreciation;
      s.colo += c.colo;
      s.total += c.total;
      return s;
    },
    { capex: 0, depr: 0, colo: 0, total: 0 },
  );

  const roleBadge = (a: Asset) => {
    if (a.type === "switch")
      return (
        <Badge kind="neutral">
          <Icon name="switch" size={12} /> Switch
        </Badge>
      );
    if (a.type === "storage")
      return (
        <Badge kind="neutral">
          <Icon name="storage" size={12} /> Storage
        </Badge>
      );
    const m: Record<string, [BadgeKind, string]> = { "k8s-node": ["info", "K8s node"], "vm-host": ["purple", "VM host"], "bare-metal": ["neutral", "Bare metal"] };
    const [k, l] = m[a.role] || ["neutral", "Server"];
    return (
      <Badge kind={k}>
        <Icon name="server" size={12} /> {l}
      </Badge>
    );
  };

  return (
    <div className="content-inner">
      <div className="grid grid-4" style={{ marginBottom: 16 }}>
        <Stat label="Total CapEx" value={fmt(sums.capex)} unit={unit} icon="money" accent="var(--accent)" meta={<span>{rows.length} assets</span>} />
        <Stat label="Monthly depreciation" value={fmt(sums.depr)} unit={unit} icon="trend" accent="var(--info)" meta={<span>{assets.filter((a) => IC.depreciation(a) > 0).length} still amortizing</span>} />
        <Stat label="Monthly colo share" value={fmt(sums.colo)} unit={unit} icon="datacenter" accent="var(--warn)" meta={<span>allocated by power draw</span>} />
        <Stat label="Total monthly cost" value={fmt(sums.total)} unit={unit} icon="dollar" accent="var(--accent-2)" meta={<span>{fmt(sums.total / IC.HOURS_PER_MONTH)} {unit}/hr</span>} />
      </div>

      <Panel
        noBody
        title="Hardware inventory"
        sub="Servers, switches & storage — straight-line depreciation + colo allocation"
        right={
          <div className="row gap-8">
            <div className="search-box">
              <Icon name="search" className="ico" />
              <input className="input" placeholder="Search assets…" value={q} onChange={(e) => setQ(e.target.value)} />
            </div>
            <select className="select" value={typeF} onChange={(e) => setTypeF(e.target.value)} style={{ width: 130 }}>
              <option value="all">All types</option>
              <option value="server">Servers</option>
              <option value="switch">Switches</option>
              <option value="storage">Storage</option>
            </select>
            <select className="select" value={dcF} onChange={(e) => setDcF(e.target.value)} style={{ width: 150 }}>
              <option value="all">All data centers</option>
              {dcs.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.short}
                </option>
              ))}
            </select>
            {isAdmin && (
              <button className="btn primary" onClick={() => setEditing({})}>
                <Icon name="plus" size={15} /> Add asset
              </button>
            )}
          </div>
        }
      >
        <div className="tbl-wrap">
          <table className="tbl">
            <thead>
              <tr>
                <Th label="Asset" col="name" sort={sort} setSort={setSort} />
                <Th label="Type" col="type" sort={sort} setSort={setSort} />
                <Th label="DC" col="dc" sort={sort} setSort={setSort} />
                <th>Specs</th>
                <Th label="Purchase" col="purchase" sort={sort} setSort={setSort} align="r" />
                <Th label="Amort" col="amort" sort={sort} setSort={setSort} align="r" />
                <Th label="Depr / mo" col="depr" sort={sort} setSort={setSort} align="r" />
                <Th label="Colo / mo" col="colo" sort={sort} setSort={setSort} align="r" />
                <Th label="Total / mo" col="total" sort={sort} setSort={setSort} align="r" />
                {isAdmin && <th></th>}
              </tr>
            </thead>
            <tbody>
              {rows.map((a) => {
                const c = costOf(a.id);
                const rem = IC.amortRemaining(a);
                const done = IC.depreciation(a) === 0;
                return (
                  <tr key={a.id}>
                    <td>
                      <div className="cell-strong">{a.name}</div>
                      <div className="cell-sub">
                        {a.vendor} {a.model}
                      </div>
                    </td>
                    <td>{roleBadge(a)}</td>
                    <td>
                      <span className="tag">{dcName(a.dcId)}</span>
                    </td>
                    <td className="cell-sub">
                      {a.cpuCores ? (
                        <span className="cell-mono">
                          {a.cpuCores}c · {a.ramGB >= 1024 ? a.ramGB / 1024 + "TB" : a.ramGB + "GB"}
                          {(a.ssdGB > 0 || a.hddGB > 0) && (
                            <span className="faint">
                              {a.ssdGB > 0 ? ` · ${a.ssdGB >= 1024 ? a.ssdGB / 1024 + "TB" : a.ssdGB + "GB"} SSD` : ""}
                              {a.hddGB > 0 ? ` · ${a.hddGB >= 1024 ? a.hddGB / 1024 + "TB" : a.hddGB + "GB"} HDD` : ""}
                            </span>
                          )}
                        </span>
                      ) : (
                        <span className="faint">—</span>
                      )}
                      <div className="faint cell-mono" style={{ fontSize: 11 }}>
                        {a.powerWatts}W
                      </div>
                    </td>
                    <td className="r">
                      <div className="cell-mono cell-strong">{fmt(a.purchaseAmount)}</div>
                      <div className="cell-sub cell-mono">{a.purchaseDate}</div>
                    </td>
                    <td className="r">
                      {done ? (
                        <Badge kind="good">paid off</Badge>
                      ) : (
                        <span>
                          <span className="cell-mono">{rem}</span>
                          <span className="faint">/{a.amortMonths}mo</span>
                        </span>
                      )}
                    </td>
                    <td className="r cell-mono">{c.depreciation > 0 ? fmt(c.depreciation) : <span className="faint">0</span>}</td>
                    <td className="r cell-mono">{fmt(c.colo)}</td>
                    <td className="r cell-mono cell-strong" style={{ color: "var(--accent)" }}>
                      {fmt(c.total)}
                    </td>
                    {isAdmin && (
                      <td className="r">
                        <div className="row" style={{ justifyContent: "flex-end", gap: 2 }}>
                          <button className="icon-btn" onClick={() => setEditing(a)} title="Edit">
                            <Icon name="edit" size={15} />
                          </button>
                          <button className="icon-btn" onClick={() => ctx.deleteAsset(a.id)} title="Delete">
                            <Icon name="trash" size={15} />
                          </button>
                        </div>
                      </td>
                    )}
                  </tr>
                );
              })}
              {rows.length === 0 && (
                <tr>
                  <td colSpan={10} className="empty">
                    No assets match your filters.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Panel>

      {editing && <AssetForm asset={editing} dcs={dcs} unit={unit} fmtFull={fmtFull} onClose={() => setEditing(null)} onSave={(a) => { ctx.upsertAsset(a); setEditing(null); }} />}
    </div>
  );
}

function AssetForm({ asset, dcs, onClose, onSave, unit, fmtFull }: { asset: Partial<Asset>; dcs: DataCenter[]; onClose: () => void; onSave: (a: Asset) => void; unit: string; fmtFull: (v: number) => string }) {
  const isNew = !asset.id;
  const [f, setF] = useState<Asset>(() => ({
    id: asset.id || "srv-" + Math.random().toString(36).slice(2, 6),
    name: asset.name || "",
    type: asset.type || "server",
    role: asset.role || "k8s-node",
    dcId: asset.dcId || (dcs[0] && dcs[0].id) || "",
    vendor: asset.vendor || "",
    model: asset.model || "",
    cpuCores: asset.cpuCores || 0,
    ramGB: asset.ramGB || 0,
    ssdGB: asset.ssdGB || 0,
    hddGB: asset.hddGB || 0,
    powerWatts: asset.powerWatts || 300,
    purchaseAmount: asset.purchaseAmount || 0,
    purchaseDate: asset.purchaseDate || "2025-01-01",
    amortMonths: asset.amortMonths || 48,
    rackUnits: asset.rackUnits || 1,
  }));
  const set = <K extends keyof Asset>(k: K, v: Asset[K]) => setF((p) => ({ ...p, [k]: v }));
  const isServer = f.type === "server";
  const monthly = f.amortMonths ? f.purchaseAmount / f.amortMonths : 0;

  return (
    <Modal
      wide
      title={isNew ? "Add asset" : "Edit " + f.name}
      onClose={onClose}
      footer={
        <>
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn primary" onClick={() => onSave(f)} disabled={!f.name}>
            {isNew ? "Add asset" : "Save changes"}
          </button>
        </>
      }
    >
      <div className="field-row">
        <Field label="Asset name">
          <input className="input" value={f.name} onChange={(e) => set("name", e.target.value)} placeholder="k8s-node-12" />
        </Field>
        <Field label="Type">
          <select
            className="select"
            value={f.type}
            onChange={(e) => {
              const ty = e.target.value as AssetType;
              set("type", ty);
              set("role", ty === "switch" ? "network" : ty === "storage" ? "storage" : "k8s-node");
            }}
          >
            <option value="server">Server</option>
            <option value="switch">Switch</option>
            <option value="storage">Storage</option>
          </select>
        </Field>
      </div>
      <div className="field-row">
        {isServer && (
          <Field label="Role">
            <select className="select" value={f.role} onChange={(e) => set("role", e.target.value as AssetRole)}>
              <option value="k8s-node">Kubernetes node</option>
              <option value="vm-host">VM host</option>
              <option value="bare-metal">Bare metal</option>
            </select>
          </Field>
        )}
        <Field label="Data center">
          <select className="select" value={f.dcId} onChange={(e) => set("dcId", e.target.value)}>
            {dcs.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </select>
        </Field>
      </div>
      <div className="field-row">
        <Field label="Vendor">
          <input className="input" value={f.vendor} onChange={(e) => set("vendor", e.target.value)} placeholder="Dell" />
        </Field>
        <Field label="Model">
          <input className="input" value={f.model} onChange={(e) => set("model", e.target.value)} placeholder="PowerEdge R740" />
        </Field>
      </div>
      {isServer && (
        <>
          <div className="field-row">
            <Field label="CPU cores">
              <input className="input mono" type="number" value={f.cpuCores} onChange={(e) => set("cpuCores", +e.target.value)} />
            </Field>
            <Field label="RAM (GB)">
              <input className="input mono" type="number" value={f.ramGB} onChange={(e) => set("ramGB", +e.target.value)} />
            </Field>
          </div>
          <div className="field-row">
            <Field label="SSD capacity (GB)">
              <input className="input mono" type="number" value={f.ssdGB} onChange={(e) => set("ssdGB", +e.target.value)} />
            </Field>
            <Field label="HDD capacity (GB)">
              <input className="input mono" type="number" value={f.hddGB} onChange={(e) => set("hddGB", +e.target.value)} />
            </Field>
          </div>
        </>
      )}
      <div className="field-row">
        <Field label="Power draw (W)" hint="Used to allocate colo fee">
          <input className="input mono" type="number" value={f.powerWatts} onChange={(e) => set("powerWatts", +e.target.value)} />
        </Field>
        <Field label="Rack units">
          <input className="input mono" type="number" value={f.rackUnits} onChange={(e) => set("rackUnits", +e.target.value)} />
        </Field>
      </div>
      <div className="divider" />
      <div className="field-row">
        <Field label={"Purchase amount (" + unit + ")"} hint={f.purchaseAmount ? fmtFull(f.purchaseAmount) + " " + unit : "Enter full amount"}>
          <input className="input mono" type="number" value={f.purchaseAmount} onChange={(e) => set("purchaseAmount", +e.target.value)} />
        </Field>
        <Field label="Purchase date">
          <input className="input mono" type="date" value={f.purchaseDate} onChange={(e) => set("purchaseDate", e.target.value)} />
        </Field>
        <Field label="Amortization (months)">
          <input className="input mono" type="number" value={f.amortMonths} onChange={(e) => set("amortMonths", +e.target.value)} />
        </Field>
      </div>
      <div className="panel" style={{ background: "var(--bg-3)", marginTop: 4 }}>
        <div className="panel-body row" style={{ gap: 24 }}>
          <div>
            <div className="kpi-mini">Monthly depreciation</div>
            <div className="num" style={{ fontSize: 18, fontWeight: 700, color: "var(--accent)" }}>
              {IC.fmtMoney(monthly, { unit: unit as "Rial" | "Toman" })} <span className="faint" style={{ fontSize: 12 }}>{unit}</span>
            </div>
          </div>
          <div>
            <div className="kpi-mini">Hourly</div>
            <div className="num" style={{ fontSize: 18, fontWeight: 700 }}>
              {IC.fmtMoney(monthly / IC.HOURS_PER_MONTH, { unit: unit as "Rial" | "Toman" })}
            </div>
          </div>
          <div>
            <div className="kpi-mini">Method</div>
            <div style={{ fontSize: 13, marginTop: 4 }} className="muted">
              straight-line · {f.purchaseAmount ? fmtFull(f.purchaseAmount) : "—"} ÷ {f.amortMonths || 0}mo
            </div>
          </div>
        </div>
      </div>
    </Modal>
  );
}
