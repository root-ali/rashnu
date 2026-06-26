/* Cost reports page — persisted backend service costs */
import { useEffect, useMemo, useState } from "react";
import { Icon } from "../components/icons";
import { Panel, PlatformBadge, Stat } from "../components/ui";
import { AreaChart, CHART_COLORS, Donut, HBarList, type SeriesDatum } from "../components/charts";
import { getCostSpendApi, type BackendSpendSummary } from "../lib/api";
import { IC } from "../lib/data";
import type { Ctx, Platform, ServiceMonthlyReport } from "../lib/types";

type SvcRow = ServiceMonthlyReport & { workloads: number; env: string };

export function ReportsPage({ ctx }: { ctx: Ctx }) {
  const { services, costReports, isAdmin, calculateDailyCosts, fulfillCostReports, fmt, unit, t } = ctx;
  const [platF, setPlatF] = useState<string>("all");
  const [rate, setRate] = useState<"monthly" | "hourly">("monthly");
  const [groupBy, setGroupBy] = useState<string>("team");
  const [localStyle, setLocalStyle] = useState<string | null>(null);
  const [groupSpend, setGroupSpend] = useState<BackendSpendSummary | null>(null);
  const style = localStyle || t.reportStyle || "mixed";

  const rateDiv = rate === "hourly" ? IC.HOURS_PER_MONTH : 1;
  const rmoney = (v: number) => fmt(v / rateDiv);
  const monthLabel = costReports.monthlyMonth || new Date().toISOString().slice(0, 7);

  const serviceMeta = useMemo(() => {
    const map: Record<string, { workloads: number; env: string }> = {};
    services.forEach((s) => {
      map[s.id] = { workloads: s.workloads.length, env: s.env };
    });
    return map;
  }, [services]);

  const list: SvcRow[] = useMemo(() => {
    return costReports.monthly
      .filter((r) => platF === "all" || r.platform === platF)
      .map((r) => ({
        ...r,
        workloads: serviceMeta[r.service_id]?.workloads ?? 0,
        env: serviceMeta[r.service_id]?.env ?? "—",
      }))
      .sort((a, b) => b.total_cost - a.total_cost);
  }, [costReports.monthly, platF, serviceMeta]);

  const total = list.reduce((s, r) => s + r.total_cost, 0);

  const byPlat: SeriesDatum[] = ["kubernetes", "vm"]
    .map((p) => ({
      label: p === "vm" ? "VM" : "Kubernetes",
      value: list.filter((r) => r.platform === p).reduce((a, r) => a + r.total_cost, 0),
      color: p === "vm" ? "var(--purple)" : "var(--info)",
    }))
    .filter((d) => d.value > 0);

  useEffect(() => {
    if (groupBy === "env") {
      setGroupSpend(null);
      return;
    }
    let cancelled = false;
    void getCostSpendApi({ month: monthLabel, groupBy, platform: platF })
      .then((res) => {
        if (!cancelled) setGroupSpend(res);
      })
      .catch(() => {
        if (!cancelled) setGroupSpend(null);
      });
    return () => {
      cancelled = true;
    };
  }, [groupBy, platF, monthLabel]);

  const groupData: SeriesDatum[] = useMemo(() => {
    if (groupBy === "env") {
      const groups: Record<string, number> = {};
      list.forEach((r) => {
        groups[r.env] = (groups[r.env] || 0) + r.total_cost;
      });
      return Object.entries(groups)
        .map(([label, value], i) => ({ label, value, color: CHART_COLORS[i % CHART_COLORS.length] }))
        .sort((a, b) => b.value - a.value);
    }
    return (groupSpend?.items ?? [])
      .map((item, i) => ({
        label: item.label,
        value: item.total_cost,
        color: CHART_COLORS[i % CHART_COLORS.length],
      }))
      .sort((a, b) => b.value - a.value);
  }, [groupBy, groupSpend, list]);

  const topServices: SeriesDatum[] = list.slice(0, 8).map((r) => ({
    label: r.service_name,
    sub: r.team,
    value: r.total_cost,
    color: r.platform === "vm" ? "var(--purple)" : "var(--info)",
  }));

  const trendData = costReports.trend.map((d) => ({
    month: IC.formatTrendMonth(d.month),
    value: d.total_cost / rateDiv,
  }));

  const exportCsv = () => {
    const hdr = ["service", "platform", "team", "env", "workloads", "monthly_" + unit, "hourly_" + unit];
    const lines = [hdr.join(",")].concat(
      list.map((r) =>
        [r.service_name, r.platform, r.team, r.env, r.workloads, Math.round(r.total_cost), Math.round(r.total_cost / IC.HOURS_PER_MONTH)].join(","),
      ),
    );
    const blob = new Blob([lines.join("\n")], { type: "text/csv" });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = "rashnu-cost-report.csv";
    a.click();
    ctx.toast("Exported cost report CSV");
  };

  const recalculate = () => {
    const defaultStart = new Date().toISOString().slice(0, 8) + "01";
    const startTime = window.prompt("Backfill cost reports from date (YYYY-MM-DD):", defaultStart);
    if (!startTime) return;
    if (!/^\d{4}-\d{2}-\d{2}$/.test(startTime)) {
      ctx.toast("Invalid date — use YYYY-MM-DD", "error");
      return;
    }
    void fulfillCostReports(startTime);
  };

  const k8sVal = byPlat.find((p) => p.label === "Kubernetes")?.value || 0;
  const vmVal = byPlat.find((p) => p.label === "VM")?.value || 0;
  const hasData = costReports.monthly.length > 0 || costReports.trend.length > 0;
  const monthTotal = costReports.monthly.reduce((s, r) => s + r.total_cost, 0);

  return (
    <div className="content-inner">
      <div className="row gap-8 wrap" style={{ marginBottom: 16 }}>
        <div className="seg">
          {["all", "kubernetes", "vm"].map((p) => (
            <button key={p} className={platF === p ? "on" : ""} onClick={() => setPlatF(p)}>
              {p === "all" ? "All" : p === "vm" ? "VM" : "K8s"}
            </button>
          ))}
        </div>
        <div className="seg">
          {(["monthly", "hourly"] as const).map((r) => (
            <button key={r} className={rate === r ? "on" : ""} onClick={() => setRate(r)}>
              {r === "monthly" ? "Monthly" : "Hourly"}
            </button>
          ))}
        </div>
        <div className="spacer" />
        {!hasData && <span className="faint" style={{ fontSize: 12 }}>Run pricing fulfillment, then Recalculate</span>}
        {isAdmin && (
          <>
            <button className="btn" onClick={() => calculateDailyCosts()}>
              Recalculate today
            </button>
            <button className="btn" onClick={recalculate}>
              Recalculate
            </button>
          </>
        )}
        <span className="kpi-mini">View</span>
        <div className="seg">
          {["charts", "tables", "mixed"].map((v) => (
            <button key={v} className={style === v ? "on" : ""} onClick={() => setLocalStyle(v)}>
              {v[0].toUpperCase() + v.slice(1)}
            </button>
          ))}
        </div>
        <button className="btn" onClick={exportCsv}>
          <Icon name="download" size={15} /> CSV
        </button>
      </div>

      <div className="row gap-8" style={{ marginBottom: 10, alignItems: "center" }}>
        <span style={{ fontWeight: 600, fontSize: 14 }}>Service costs</span>
        <span className="faint" style={{ fontSize: 12 }}>
          Persisted monthly totals from per-datacenter unit pricing · {monthLabel}
        </span>
      </div>

      <div className="grid grid-4" style={{ marginBottom: 16 }}>
        <Stat label={"Total service cost / " + (rate === "hourly" ? "hr" : "mo")} value={rmoney(total)} unit={unit} icon="dollar" accent="var(--accent)" meta={<span>{list.length} services</span>} />
        <Stat label="Kubernetes" value={rmoney(k8sVal)} unit={unit} icon="k8s" accent="var(--info)" meta={<span>{((k8sVal / (total || 1)) * 100).toFixed(0)}% of spend</span>} />
        <Stat label="VM" value={rmoney(vmVal)} unit={unit} icon="vm" accent="var(--purple)" meta={<span>{((vmVal / (total || 1)) * 100).toFixed(0)}% of spend</span>} />
        <Stat label="Services tracked" value={String(list.length)} unit="" icon="server" accent="var(--accent-2)" meta={<span>{costReports.monthly.length} with cost data</span>} />
      </div>

      {!hasData && (
        <Panel title="No cost data yet" style={{ marginBottom: 16 }}>
          <div className="faint" style={{ fontSize: 13 }}>
            No monthly cost reports yet. Run pricing fulfillment, then use Recalculate to backfill.
          </div>
        </Panel>
      )}

      {(style === "charts" || style === "mixed") && hasData && (
        <div className="grid grid-2" style={{ marginBottom: 16 }}>
          <Panel title="Spend by platform">
            <Donut data={byPlat} fmt={(v) => rmoney(v)} centerLabel={unit + " / " + (rate === "hourly" ? "hr" : "mo")} centerValue={rmoney(total)} />
          </Panel>
          <Panel
            title={"Spend by " + groupBy}
            right={
              <select className="select" value={groupBy} onChange={(e) => setGroupBy(e.target.value)} style={{ width: 150 }}>
                <option value="team">By team</option>
                <option value="platform">By platform</option>
                <option value="env">By environment</option>
                <option value="datacenter">By data center</option>
              </select>
            }
          >
            <HBarList data={groupData} fmt={(v) => rmoney(v)} />
          </Panel>
        </div>
      )}

      {(style === "charts" || style === "mixed") && hasData && (
        <Panel title="Top services by cost" sub="ranked, all platforms" style={{ marginBottom: 16 }}>
          <HBarList data={topServices} fmt={(v) => rmoney(v)} />
        </Panel>
      )}

      {(style === "tables" || style === "mixed") && (
        <Panel noBody title="Per-service cost report" sub={unit + " / " + (rate === "hourly" ? "hour" : "month") + " · " + monthLabel}>
          {list.length === 0 ? (
            <div className="faint" style={{ padding: 20, fontSize: 13 }}>No service cost data for this month.</div>
          ) : (
            <div className="tbl-wrap">
              <table className="tbl">
                <thead>
                  <tr>
                    <th>Service</th>
                    <th>Platform</th>
                    <th>Team</th>
                    <th className="r">Workloads</th>
                    <th className="r">Days</th>
                    <th className="r">Monthly cost</th>
                    <th>Share</th>
                  </tr>
                </thead>
                <tbody>
                  {list.map((r) => (
                    <tr key={r.service_id}>
                      <td>
                        <div className="cell-strong">{r.service_name}</div>
                        <div className="cell-sub">{r.platform === "kubernetes" ? "K8s" : r.env}</div>
                      </td>
                      <td>
                        <PlatformBadge platform={r.platform as Platform} />
                      </td>
                      <td className="cell-sub">{r.team}</td>
                      <td className="r cell-mono">{r.workloads}</td>
                      <td className="r cell-mono cell-sub">{r.days_recorded}</td>
                      <td className="r cell-mono cell-strong" style={{ color: "var(--accent)" }}>
                        {rmoney(r.total_cost)}
                      </td>
                      <td style={{ minWidth: 120 }}>
                        <div className="row gap-8">
                          <div className="minibar" style={{ flex: 1 }}>
                            <span style={{ width: ((r.total_cost / (list[0]?.total_cost || 1)) * 100) + "%", background: r.platform === "vm" ? "var(--purple)" : "var(--info)" }} />
                          </div>
                          <span className="cell-mono cell-sub" style={{ minWidth: 34 }}>
                            {(((r.total_cost / (total || 1)) * 100).toFixed(1))}%
                          </span>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr style={{ borderTop: "2px solid var(--border-2)" }}>
                    <td className="cell-strong" colSpan={5}>
                      Total · {list.length} services
                    </td>
                    <td className="r cell-mono cell-strong" style={{ color: "var(--accent)" }}>
                      {rmoney(total)}
                    </td>
                    <td></td>
                  </tr>
                </tfoot>
              </table>
            </div>
          )}
        </Panel>
      )}

      {(style === "tables" || style === "mixed") && costReports.monthly.length > 0 && (
        <Panel
          noBody
          title="Resource breakdown"
          sub={`${costReports.monthly.length} services · ${monthLabel}`}
          style={{ marginTop: 16 }}
        >
          <div className="tbl-wrap">
            <table className="tbl">
              <thead>
                <tr>
                  <th>Service</th>
                  <th>Team</th>
                  <th className="r">Days</th>
                  <th className="r">CPU</th>
                  <th className="r">RAM</th>
                  <th className="r">SSD</th>
                  <th className="r">HDD</th>
                  <th className="r">Daily avg</th>
                  <th className="r">Month total</th>
                </tr>
              </thead>
              <tbody>
                {costReports.monthly.map((r) => (
                  <>
                    <tr key={r.service_id}>
                      <td>
                        <div className="cell-strong">{r.service_name}</div>
                        <div className="cell-sub">{r.platform === "kubernetes" ? "K8s" : "VM"}</div>
                      </td>
                      <td className="cell-sub">{r.team}</td>
                      <td className="r cell-mono cell-sub">{r.days_recorded}</td>
                      <td className="r cell-mono cell-sub">{fmt(r.cpu_cost)} {unit}</td>
                      <td className="r cell-mono cell-sub">{fmt(r.ram_cost)} {unit}</td>
                      <td className="r cell-mono cell-sub">{fmt(r.ssd_cost)} {unit}</td>
                      <td className="r cell-mono cell-sub">{fmt(r.hdd_cost)} {unit}</td>
                      <td className="r cell-mono cell-sub">{fmt(r.daily_avg)} {unit}</td>
                      <td className="r cell-mono cell-strong" style={{ color: "var(--accent)" }}>
                        {fmt(r.total_cost)} {unit}
                      </td>
                    </tr>
                    {r.datacenter_breakdown.length > 1 &&
                      r.datacenter_breakdown.map((dc) => (
                        <tr key={r.service_id + dc.datacenter_id} style={{ background: "var(--surface-2)" }}>
                          <td style={{ paddingLeft: 24 }}>
                            <div className="cell-sub" style={{ fontSize: 12 }}>
                              ↳ {dc.datacenter_name}
                            </div>
                          </td>
                          <td />
                          <td />
                          <td className="r cell-mono cell-sub" style={{ fontSize: 12 }}>
                            {fmt(dc.cpu_cost)} {unit}
                          </td>
                          <td className="r cell-mono cell-sub" style={{ fontSize: 12 }}>
                            {fmt(dc.ram_cost)} {unit}
                          </td>
                          <td className="r cell-mono cell-sub" style={{ fontSize: 12 }}>
                            {fmt(dc.ssd_cost)} {unit}
                          </td>
                          <td className="r cell-mono cell-sub" style={{ fontSize: 12 }}>
                            {fmt(dc.hdd_cost)} {unit}
                          </td>
                          <td className="r cell-mono cell-sub" style={{ fontSize: 12 }}>
                            {fmt(dc.daily_avg)} {unit}
                          </td>
                          <td className="r cell-mono cell-sub" style={{ fontSize: 12 }}>
                            {fmt(dc.total_cost)} {unit}
                          </td>
                        </tr>
                      ))}
                  </>
                ))}
              </tbody>
              <tfoot>
                <tr style={{ borderTop: "2px solid var(--border-2)" }}>
                  <td className="cell-strong" colSpan={3}>
                    Total
                  </td>
                  <td className="r cell-mono">{fmt(costReports.monthly.reduce((a, r) => a + r.cpu_cost, 0))} {unit}</td>
                  <td className="r cell-mono">{fmt(costReports.monthly.reduce((a, r) => a + r.ram_cost, 0))} {unit}</td>
                  <td className="r cell-mono">{fmt(costReports.monthly.reduce((a, r) => a + r.ssd_cost, 0))} {unit}</td>
                  <td className="r cell-mono">{fmt(costReports.monthly.reduce((a, r) => a + r.hdd_cost, 0))} {unit}</td>
                  <td />
                  <td className="r cell-mono cell-strong" style={{ color: "var(--accent)" }}>
                    {fmt(monthTotal)} {unit}
                  </td>
                </tr>
              </tfoot>
            </table>
          </div>
        </Panel>
      )}

      {(style === "charts" || style === "mixed") && (
        <Panel title="Spend trend" sub={`trailing 12 months · persisted backend data · ${rate === "hourly" ? "hourly rate" : "monthly total"}`} style={{ marginTop: 16 }}>
          <AreaChart data={trendData} height={200} fmt={(v) => fmt(v) + " " + unit} />
        </Panel>
      )}
    </div>
  );
}
