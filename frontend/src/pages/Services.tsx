/* Service definitions + resources page */
import { Fragment, useEffect, useState } from "react";
import { Icon } from "../components/icons";
import { CritBadge, Field, Modal, Panel, PlatformBadge, Stat } from "../components/ui";
import { IC } from "../lib/data";
import { getServiceApi } from "../lib/api";
import { mapBackendService } from "../lib/mappers";
import type { Asset, Crit, Ctx, DataCenter, LivePodMetric, Platform, Service, ServiceAlloc, Workload } from "../lib/types";

export function ServicesPage({ ctx }: { ctx: Ctx }) {
  const { services, assets, dcs, derived, isAdmin, fmt, unit, upsertService, toast } = ctx;
  const [q, setQ] = useState("");
  const [platF, setPlatF] = useState<string>("all");
  const [expanded, setExpanded] = useState<string | null>(null);
  const [editing, setEditing] = useState<Partial<Service> | null>(null);
  const assetName = (id: string) => assets.find((a) => a.id === id)?.name || id;

  const rows = services.filter((s) => {
    if (platF !== "all" && s.platform !== platF) return false;
    if (q && !(s.name + s.team + (s.namespace || "")).toLowerCase().includes(q.toLowerCase())) return false;
    return true;
  });
  const k8sN = services.filter((s) => s.platform === "kubernetes").length;
  const vmN = services.length - k8sN;
  const totalCost = services.reduce((s, sv) => s + (derived.alloc.byService[sv.id]?.monthlyTotal || 0), 0);

  const res = (s: Service) => ({
    vcpu: s.resources?.vcpu ?? s.workloads.reduce((a, w) => a + w.cpuRequest, 0),
    memGB: s.resources?.memory_gb ?? s.workloads.reduce((a, w) => a + w.ramRequestGB, 0),
    storGB: (s.resources?.ssd_gb ?? 0) + (s.resources?.hdd_gb ?? 0),
    gpus: s.resources?.gpus ?? 0,
  });

  return (
    <div className="content-inner">
      <div className="grid grid-4" style={{ marginBottom: 16 }}>
        <Stat label="Services" value={services.length} icon="services" accent="var(--accent)" meta={<span>{services.reduce((s, x) => s + x.workloads.length, 0)} workloads</span>} />
        <Stat label="On Kubernetes" value={k8sN} icon="k8s" accent="var(--info)" meta={<span className="row gap-6"><span className="dot-live" />metrics via Prometheus</span>} />
        <Stat label="On VMs" value={vmN} icon="vm" accent="var(--purple)" meta={<span>dedicated guests</span>} />
        <Stat label="Allocated / mo" value={fmt(totalCost)} unit={unit} icon="dollar" accent="var(--accent-2)" meta={<span>{fmt(totalCost / IC.HOURS_PER_MONTH)} {unit}/hr</span>} />
      </div>

      <Panel
        noBody
        title="Service catalog"
        sub="Resource definitions & cost attribution per service"
        right={
          <div className="row gap-8">
            <div className="search-box">
              <Icon name="search" className="ico" />
              <input className="input" placeholder="Search services…" value={q} onChange={(e) => setQ(e.target.value)} />
            </div>
            <div className="seg">
              {["all", "kubernetes", "vm"].map((p) => (
                <button key={p} className={platF === p ? "on" : ""} onClick={() => setPlatF(p)}>
                  {p === "all" ? "All" : p === "vm" ? "VM" : "K8s"}
                </button>
              ))}
            </div>
            {isAdmin && (
              <button className="btn primary" onClick={() => setEditing({})}>
                <Icon name="plus" size={15} /> Define service
              </button>
            )}
          </div>
        }
      >
        <div className="tbl-wrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>Service</th>
                <th>Platform</th>
                <th>Team</th>
                <th>Criticality</th>
                <th className="r">Workloads</th>
                <th className="r">vCPU</th>
                <th className="r">RAM (GB)</th>
                <th className="r">Storage (GB)</th>
                <th className="r">Cost / mo</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {rows.map((s) => {
                const cost = derived.alloc.byService[s.id] || { monthly: 0, breakdown: [] };
                const open = expanded === s.id;
                const r = res(s);
                return (
                  <Fragment key={s.id}>
                    <tr style={{ cursor: "pointer" }} onClick={() => setExpanded(open ? null : s.id)}>
                      <td>
                        <div className="row gap-8">
                          <Icon name="chevron" size={13} style={{ transform: open ? "rotate(90deg)" : "none", transition: "transform .15s", color: "var(--text-3)" }} />
                          <div>
                            <div className="cell-strong">{s.name}</div>
                            <div className="cell-sub">
                              {s.platform === "kubernetes" ? "ns: " + (s.namespace || "—") : s.workloads.length + " VM" + (s.workloads.length > 1 ? "s" : "")} · {s.env}
                            </div>
                          </div>
                        </div>
                      </td>
                      <td>
                        <PlatformBadge platform={s.platform} />
                      </td>
                      <td className="cell-sub">{s.team}</td>
                      <td>
                        <CritBadge crit={s.crit} />
                      </td>
                      <td className="r cell-mono">{s.workloads.length}</td>
                      <td className="r cell-mono">{r.vcpu}</td>
                      <td className="r cell-mono">{r.memGB}</td>
                      <td className="r cell-mono">{r.storGB > 0 ? r.storGB : "—"}</td>
                      <td className="r cell-mono cell-strong" style={{ color: "var(--accent)" }}>
                        {fmt(cost.monthlyTotal || 0)}
                      </td>
                      <td className="r">
                        {isAdmin && (
                          <button
                            className="icon-btn"
                            onClick={(e) => { e.stopPropagation(); setEditing(s); }}
                            title="Edit"
                          >
                            <Icon name="edit" size={15} />
                          </button>
                        )}
                      </td>
                    </tr>
                    {open && (
                      <tr>
                        <td colSpan={10} style={{ background: "var(--bg-3)", padding: 0 }}>
                          <ServiceDetail
                            s={s}
                            dcs={dcs}
                            assetName={assetName}
                            cost={cost}
                            fmt={fmt}
                            unit={unit}
                            onSaveUsage={(updated) => { upsertService(updated); toast("Usage saved"); }}
                          />
                        </td>
                      </tr>
                    )}
                  </Fragment>
                );
              })}
              {rows.length === 0 && (
                <tr>
                  <td colSpan={10} className="empty">
                    No services match.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </Panel>

      {editing && (
        <ServiceForm
          svc={editing}
          assets={assets}
          dcs={dcs}
          onClose={() => setEditing(null)}
          onSave={(s) => {
            ctx.upsertService(s);
            setEditing(null);
          }}
          onDelete={
            editing.id
              ? () => {
                  ctx.deleteService(editing.id!);
                  setEditing(null);
                }
              : null
          }
        />
      )}
    </div>
  );
}

function ServiceDetail({
  s, dcs, assetName, cost, fmt, unit, onSaveUsage,
}: {
  s: Service;
  dcs: DataCenter[];
  assetName: (id: string) => string;
  cost: ServiceAlloc;
  fmt: (v: number) => string;
  unit: string;
  onSaveUsage: (updated: Service) => void;
}) {
  const isK8s = s.platform === "kubernetes";
  const dcName = (id: string) => dcs.find((d) => d.id === id)?.short || id.slice(0, 8);

  // Fetch enriched service data (with live_pods when Prometheus is on)
  const [enriched, setEnriched] = useState<Service>(s);
  const [loading, setLoading] = useState(false);
  const [fetchErr, setFetchErr] = useState("");

  const fetchEnriched = () => {
    setLoading(true);
    setFetchErr("");
    getServiceApi(s.id)
      .then((raw) => setEnriched(mapBackendService(raw)))
      .catch((e) => setFetchErr(String(e)))
      .finally(() => setLoading(false));
  };

  useEffect(() => { fetchEnriched(); }, [s.id]); // eslint-disable-line react-hooks/exhaustive-deps

  const livePods: LivePodMetric[] | undefined = enriched.live_pods;
  const hasLive = !!livePods?.length;

  // ── Manual usage edit ─────────────────────────────────────────────────────
  const [usage, setUsage] = useState<{ cpu: number; ram: number }[]>(() =>
    enriched.workloads.map((w) => ({ cpu: w.cpuUsage, ram: w.ramUsageGB }))
  );
  const [dirty, setDirty] = useState(false);

  const setWUsage = (i: number, field: "cpu" | "ram", v: number) => {
    setUsage((prev) => { const cp = [...prev]; cp[i] = { ...cp[i], [field]: v }; return cp; });
    setDirty(true);
  };

  const saveUsage = () => {
    const updated: Service = {
      ...enriched,
      workloads: enriched.workloads.map((w, i) => ({
        ...w, cpuUsage: usage[i]?.cpu ?? w.cpuUsage, ramUsageGB: usage[i]?.ram ?? w.ramUsageGB,
      })),
    };
    onSaveUsage(updated);
    setDirty(false);
  };

  return (
    <div style={{ padding: 16 }}>
      <div className="grid grid-2" style={{ gap: 16, alignItems: "start" }}>

        {/* ── Workload / live pod table ── */}
        <div>
          <div className="row" style={{ marginBottom: 8 }}>
            <span className="panel-title" style={{ fontSize: 12.5 }}>
              {isK8s ? (hasLive ? "Live pods from Prometheus" : "Pods / replicas") : "Virtual machines"}
            </span>
            <span className="spacer" />
            {loading && <span className="cell-sub" style={{ fontSize: 11 }}>Loading…</span>}
            {hasLive && (
              <span className="badge info" style={{ fontSize: 10.5, marginLeft: 6 }}>
                <span className="dot-live" /> live
              </span>
            )}
            <button className="btn sm" onClick={fetchEnriched} disabled={loading} style={{ marginLeft: 6 }}>
              <Icon name="trend" size={12} /> Refresh
            </button>
            {!hasLive && dirty && (
              <button className="btn sm primary" onClick={saveUsage} style={{ marginLeft: 6 }}>
                Save usage
              </button>
            )}
          </div>

          {fetchErr && (
            <div style={{ padding: "8px 12px", color: "var(--bad)", fontSize: 12.5, marginBottom: 8 }}>
              <Icon name="alert" size={13} /> {fetchErr}
            </div>
          )}

          {/* ── Prometheus live pods ── */}
          {hasLive && (
            <table className="tbl" style={{ background: "var(--panel)", borderRadius: 8, overflow: "hidden", border: "1px solid var(--border)" }}>
              <thead>
                <tr>
                  <th>Pod</th>
                  <th className="r">CPU (cores)</th>
                  <th className="r">Memory (GB)</th>
                </tr>
              </thead>
              <tbody>
                {livePods!.map((p, i) => (
                  <tr key={i}>
                    <td className="cell-mono" style={{ fontSize: 12 }}>{p.pod}</td>
                    <td className="r cell-mono">{p.cpu_cores.toFixed(3)}</td>
                    <td className="r cell-mono">{p.memory_gb.toFixed(2)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          {/* ── Manual / stored workloads ── */}
          {!hasLive && (
            <table className="tbl" style={{ background: "var(--panel)", borderRadius: 8, overflow: "hidden", border: "1px solid var(--border)" }}>
              <thead>
                <tr>
                  <th>{isK8s ? "Pod" : "VM"}</th>
                  <th className="r">CPU req</th>
                  <th className="r">CPU use</th>
                  <th className="r">RAM req</th>
                  <th className="r">RAM use</th>
                </tr>
              </thead>
              <tbody>
                {enriched.workloads.length === 0 && (
                  <tr><td colSpan={5} className="empty">No workloads defined</td></tr>
                )}
                {enriched.workloads.map((w, i) => (
                  <tr key={i}>
                    <td>
                      <div className="cell-strong" style={{ fontSize: 12.5 }}>{w.vmName || enriched.name + "-" + (i + 1)}</div>
                      <div className="cell-sub cell-mono">
                        {isK8s ? assetName(w.hostId) : dcName(w.hostId)}
                      </div>
                    </td>
                    <td className="r cell-mono">{w.cpuRequest}c</td>
                    <td className="r">
                      <input className="input mono" type="number" step="0.01"
                        value={usage[i]?.cpu ?? w.cpuUsage}
                        onChange={(e) => setWUsage(i, "cpu", +e.target.value)}
                        style={{ width: 68, textAlign: "right", padding: "2px 6px", fontSize: 12 }}
                      />
                    </td>
                    <td className="r cell-mono">{w.ramRequestGB}g</td>
                    <td className="r">
                      <input className="input mono" type="number" step="0.1"
                        value={usage[i]?.ram ?? w.ramUsageGB}
                        onChange={(e) => setWUsage(i, "ram", +e.target.value)}
                        style={{ width: 68, textAlign: "right", padding: "2px 6px", fontSize: 12 }}
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        {/* ── Right column: Prometheus info card + cost ── */}
        <div className="col" style={{ gap: 12 }}>
          {isK8s && (
            <div className="panel" style={{ background: "var(--panel)" }}>
              <div className="panel-body" style={{ padding: 13 }}>
                <div className="row gap-8" style={{ marginBottom: 8 }}>
                  <Icon name="trend" size={14} style={{ color: hasLive ? "var(--good)" : "var(--text-3)" }} />
                  <span className="panel-title" style={{ fontSize: 12.5 }}>Prometheus</span>
                  <span className="spacer" />
                  <span className="badge" style={{ fontSize: 10.5,
                    background: hasLive ? "var(--good-dim)" : "var(--bg-3)",
                    color: hasLive ? "var(--good)" : "var(--text-3)" }}>
                    {hasLive ? "live" : "manual"}
                  </span>
                </div>
                <div className="mono" style={{ fontSize: 11, color: "var(--text-2)", background: "var(--bg)", borderRadius: 6, padding: "8px 10px", border: "1px solid var(--border)", lineHeight: 1.7 }}>
                  <div>namespace = <span style={{ color: "var(--good)" }}>"{enriched.namespace || "—"}"</span></div>
                  <div>service&nbsp;&nbsp; = <span style={{ color: "var(--info)" }}>"{enriched.name}"</span></div>
                </div>
                {!hasLive && (
                  <div className="cell-sub" style={{ marginTop: 8 }}>
                    Enable Prometheus in the ⚙ tweaks panel and set the server URL to pull live metrics.
                  </div>
                )}
              </div>
            </div>
          )}

          <div className="panel" style={{ background: "var(--panel)" }}>
            <div className="panel-body" style={{ padding: 13 }}>
              <div className="panel-title" style={{ fontSize: 12.5, marginBottom: 10 }}>Cost breakdown</div>
              <div className="col" style={{ gap: 8 }}>
                <div className="row">
                  <span className="muted" style={{ fontSize: 12.5 }}>Compute (allocated)</span>
                  <div className="spacer" />
                  <span className="cell-mono cell-strong">{fmt(cost.monthly || 0)} {unit}</span>
                </div>
                <div className="row">
                  <span className="muted" style={{ fontSize: 12.5 }}>Shared (network/storage)</span>
                  <div className="spacer" />
                  <span className="cell-mono">{fmt(cost.shared || 0)} {unit}</span>
                </div>
                <div className="divider" style={{ margin: "4px 0" }} />
                <div className="row">
                  <span style={{ fontSize: 13, fontWeight: 700 }}>Total monthly</span>
                  <div className="spacer" />
                  <span className="cell-mono cell-strong" style={{ color: "var(--accent)", fontSize: 15 }}>
                    {fmt(cost.monthlyTotal || 0)} {unit}
                  </span>
                </div>
                <div className="row">
                  <span className="muted" style={{ fontSize: 12 }}>Hourly</span>
                  <div className="spacer" />
                  <span className="cell-mono muted">{fmt((cost.monthlyTotal || 0) / IC.HOURS_PER_MONTH)} {unit}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function ServiceForm({ svc, assets, dcs, onClose, onSave, onDelete }: { svc: Partial<Service>; assets: Asset[]; dcs: DataCenter[]; onClose: () => void; onSave: (s: Service) => void; onDelete: (() => void) | null }) {
  const isNew = !svc.id;
  const k8sHosts = assets.filter((a) => a.role === "k8s-node");
  const [f, setF] = useState<Service>(() => ({
    id: svc.id || "svc-" + Math.random().toString(36).slice(2, 6),
    name: svc.name || "",
    team: svc.team || "",
    platform: svc.platform || "kubernetes",
    namespace: svc.namespace || "default",
    env: svc.env || "prod",
    crit: svc.crit || "med",
    manualWeight: svc.manualWeight || 3,
    workloads: svc.workloads
      ? IC.clone(svc.workloads).map((w: Workload) => ({
          ...w,
          vmName: w.vmName ?? "",
          ssdGB: w.ssdGB ?? 0,
          hddGB: w.hddGB ?? 0,
          gpus: w.gpus ?? 0,
        }))
      : [{ hostId: k8sHosts[0]?.id || "", kind: "pod", vmName: "", ssdGB: 0, hddGB: 0, gpus: 0, cpuRequest: 1, ramRequestGB: 2, cpuUsage: 0.6, ramUsageGB: 1.2 }],
  }));
  const set = <K extends keyof Service>(k: K, v: Service[K]) => setF((p) => ({ ...p, [k]: v }));
  const setW = <K extends keyof Workload>(i: number, k: K, v: Workload[K]) =>
    setF((p) => {
      const w = [...p.workloads];
      w[i] = { ...w[i], [k]: v };
      return { ...p, workloads: w };
    });
  const addW = () => {
    const isVm = f.platform === "vm";
    setF((p) => ({
      ...p,
      workloads: [
        ...p.workloads,
        {
          hostId: isVm ? (dcs[0]?.id || "") : (k8sHosts[0]?.id || ""),
          kind: isVm ? "vm" : "pod",
          vmName: "",
          ssdGB: 0,
          hddGB: 0,
          gpus: 0,
          cpuRequest: 1,
          ramRequestGB: 2,
          cpuUsage: 0.6,
          ramUsageGB: 1.2,
        },
      ],
    }));
  };
  const rmW = (i: number) => setF((p) => ({ ...p, workloads: p.workloads.filter((_, j) => j !== i) }));

  return (
    <Modal
      wide
      title={isNew ? "Define service" : "Edit " + f.name}
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
            {isNew ? "Create" : "Save"}
          </button>
        </>
      }
    >
      <div className="field-row">
        <Field label="Service name">
          <input className="input" value={f.name} onChange={(e) => set("name", e.target.value)} placeholder="api-gateway" />
        </Field>
        <Field label="Team">
          <input className="input" value={f.team} onChange={(e) => set("team", e.target.value)} placeholder="Platform" />
        </Field>
      </div>
      <div className="field-row">
        <Field label="Runs on">
          <select className="select" value={f.platform} onChange={(e) => set("platform", e.target.value as Platform)}>
            <option value="kubernetes">Kubernetes (metrics from Prometheus)</option>
            <option value="vm">Virtual machines</option>
          </select>
        </Field>
        {f.platform === "kubernetes" ? (
          <Field label="Namespace">
            <input className="input mono" value={f.namespace || ""} onChange={(e) => set("namespace", e.target.value)} />
          </Field>
        ) : (
          <Field label="Environment">
            <input className="input" value={f.env} onChange={(e) => set("env", e.target.value)} />
          </Field>
        )}
        <Field label="Criticality">
          <select className="select" value={f.crit} onChange={(e) => set("crit", e.target.value as Crit)}>
            <option value="critical">Critical</option>
            <option value="high">High</option>
            <option value="med">Medium</option>
            <option value="low">Low</option>
          </select>
        </Field>
      </div>
      <Field label="Manual cost weight" hint="Used only when allocation method is set to 'manual weight'">
        <input className="input mono" type="number" value={f.manualWeight} onChange={(e) => set("manualWeight", +e.target.value)} style={{ maxWidth: 120 }} />
      </Field>
      <div className="divider" />
      <div className="row" style={{ marginBottom: 10 }}>
        <span className="panel-title" style={{ fontSize: 13 }}>
          {f.platform === "kubernetes" ? "Pods / replicas" : "Virtual machines"}
        </span>
        <span className="spacer" />
        <button className="btn sm" onClick={addW}>
          <Icon name="plus" size={13} /> Add {f.platform === "kubernetes" ? "pod" : "VM"}
        </button>
      </div>
      <div className="col" style={{ gap: 8 }}>
        {f.workloads.map((w, i) => (
          <div key={i} style={{ background: "var(--bg-3)", padding: 8, borderRadius: 7, border: "1px solid var(--border)" }}>
            {/* ── Row 1: host/DC selector + compute resources ── */}
            <div className="row gap-8" style={{ marginBottom: w.kind === "vm" ? 6 : 0 }}>
              {w.kind === "vm" ? (
                <select className="select" value={w.hostId} onChange={(e) => setW(i, "hostId", e.target.value)} style={{ flex: 2, minWidth: 130 }} title="Datacenter">
                  {dcs.map((d) => (
                    <option key={d.id} value={d.id}>{d.short} — {d.name}</option>
                  ))}
                </select>
              ) : (
                <select className="select" value={w.hostId} onChange={(e) => setW(i, "hostId", e.target.value)} style={{ flex: 2, minWidth: 130 }} title="K8s node">
                  {k8sHosts.map((h) => (
                    <option key={h.id} value={h.id}>{h.name}</option>
                  ))}
                </select>
              )}
              <div className="input-affix" style={{ flex: 1 }}>
                <input className="input mono" type="number" step="0.1" value={w.cpuRequest} onChange={(e) => setW(i, "cpuRequest", +e.target.value)} title="vCPU request" />
                <span className="affix">c req</span>
              </div>
              <div className="input-affix" style={{ flex: 1 }}>
                <input className="input mono" type="number" step="0.1" value={w.cpuUsage} onChange={(e) => setW(i, "cpuUsage", +e.target.value)} title="CPU usage" />
                <span className="affix">c use</span>
              </div>
              <div className="input-affix" style={{ flex: 1 }}>
                <input className="input mono" type="number" value={w.ramRequestGB} onChange={(e) => setW(i, "ramRequestGB", +e.target.value)} title="RAM request (GB)" />
                <span className="affix">g req</span>
              </div>
              <div className="input-affix" style={{ flex: 1 }}>
                <input className="input mono" type="number" value={w.ramUsageGB} onChange={(e) => setW(i, "ramUsageGB", +e.target.value)} title="RAM usage (GB)" />
                <span className="affix">g use</span>
              </div>
              <button className="icon-btn" onClick={() => rmW(i)}>
                <Icon name="trash" size={14} />
              </button>
            </div>
            {/* ── Row 2 (VM only): VM name + storage + GPU ── */}
            {w.kind === "vm" && (
              <div className="row gap-8">
                <input
                  className="input mono"
                  style={{ flex: 2, minWidth: 130 }}
                  value={w.vmName || ""}
                  onChange={(e) => setW(i, "vmName", e.target.value)}
                  placeholder={f.name ? f.name + "-vm-" + (i + 1) : "vm-name"}
                  title="VM name"
                />
                <div className="input-affix" style={{ flex: 1 }}>
                  <input className="input mono" type="number" value={w.ssdGB} onChange={(e) => setW(i, "ssdGB", +e.target.value)} title="SSD (GB)" />
                  <span className="affix">SSD</span>
                </div>
                <div className="input-affix" style={{ flex: 1 }}>
                  <input className="input mono" type="number" value={w.hddGB} onChange={(e) => setW(i, "hddGB", +e.target.value)} title="HDD (GB)" />
                  <span className="affix">HDD</span>
                </div>
                <div className="input-affix" style={{ flex: 1 }}>
                  <input className="input mono" type="number" value={w.gpus} onChange={(e) => setW(i, "gpus", +e.target.value)} title="GPUs" />
                  <span className="affix">GPU</span>
                </div>
                {/* spacer to align with delete button above */}
                <div style={{ width: 28 }} />
              </div>
            )}
          </div>
        ))}
      </div>
    </Modal>
  );
}
