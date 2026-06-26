/* Overview dashboard — viz style driven by Tweaks */
import { Badge, Panel, Stat } from "../components/ui";
import { AreaChart, BarChart, CHART_COLORS, Donut, HBarList, StackedBar } from "../components/charts";
import { IC } from "../lib/data";
import type { Ctx } from "../lib/types";

export function DashboardPage({ ctx }: { ctx: Ctx }) {
  const { assets, services, dcs, derived, costReports, fmt, unit, t, nav } = ctx;
  const tot = derived.totals;
  const viz = t.vizStyle || "area";

  const composition = [
    { label: "Depreciation", value: tot.monthlyDepreciation, color: "var(--accent)" },
    { label: "Colocation", value: tot.monthlyColo, color: "var(--warn)" },
  ];

  // cost by DC (compute + colo of assets in DC)
  const byDc = dcs
    .map((d, i) => {
      const v = assets.filter((a) => a.dcId === d.id).reduce((s, a) => s + (derived.assetCosts[a.id]?.total || 0), 0);
      return { label: d.short, sub: d.provider, value: v, color: CHART_COLORS[i % CHART_COLORS.length] };
    })
    .sort((a, b) => b.value - a.value);

  const topSvc = services
    .map((s) => ({ name: s.name, platform: s.platform, v: derived.alloc.byService[s.id]?.monthlyTotal || 0 }))
    .sort((a, b) => b.v - a.v)
    .slice(0, 6)
    .map((s) => ({ label: s.name, value: s.v, color: s.platform === "vm" ? "var(--purple)" : "var(--info)" }));

  const paidOff = assets.filter((a) => IC.depreciation(a) === 0).length;
  const backendTrend = costReports.trend.map((p) => ({
    month: IC.formatTrendMonth(p.month),
    value: p.total_cost,
  }));
  const trendData = backendTrend.length > 0 ? backendTrend : derived.trend;
  const barTrend = trendData.map((d) => ({ label: d.month, value: d.value }));

  // avg utilization across compute hosts
  const computeHosts = assets.filter((a) => a.role === "k8s-node" || a.role === "vm-host");
  const avgCpu = computeHosts.reduce((s, a) => s + (derived.hostUtil[a.id]?.cpuUtil || 0), 0) / (computeHosts.length || 1);
  const avgRam = computeHosts.reduce((s, a) => s + (derived.hostUtil[a.id]?.ramUtil || 0), 0) / (computeHosts.length || 1);

  return (
    <div className="content-inner">
      <div className="grid grid-4" style={{ marginBottom: 16 }}>
        <Stat label="Monthly run-rate" value={fmt(tot.monthlyTotal)} unit={unit} icon="dollar" accent="var(--accent)" delta={3.4} meta={<span>vs last month</span>} />
        <Stat label="Hourly cost" value={fmt(tot.hourlyTotal)} unit={unit} icon="clock" accent="var(--accent-2)" meta={<span>{fmt(tot.monthlyTotal)} {unit}/mo</span>} />
        <Stat label="Annual run-rate" value={fmt(tot.annualTotal)} unit={unit} icon="trend" accent="var(--info)" meta={<span>projected 12mo</span>} />
        <Stat label="Total CapEx deployed" value={fmt(tot.totalCapex)} unit={unit} icon="server" accent="var(--purple)" meta={<span>{tot.assetCount} assets · {tot.activeAmort} amortizing</span>} />
      </div>

      <div className="grid" style={{ gridTemplateColumns: "1.7fr 1fr", marginBottom: 16 }}>
        <Panel title="Infrastructure spend trend" sub={backendTrend.length > 0 ? "trailing 12 months · actual service costs" : "trailing 12 months · estimated run-rate"} right={<Badge kind="good" dot>live</Badge>}>
          {viz === "bars" ? <BarChart data={barTrend} height={210} fmt={(v) => fmt(v) + " " + unit} /> : <AreaChart data={trendData} height={210} fmt={(v) => fmt(v) + " " + unit} />}
        </Panel>
        <Panel title="Cost composition" sub="depreciation vs colocation">
          {viz === "donut" ? (
            <Donut data={composition} fmt={(v) => fmt(v)} centerLabel={unit + "/mo"} centerValue={fmt(tot.monthlyTotal)} />
          ) : (
            <div>
              <div className="row" style={{ justifyContent: "space-between", marginBottom: 6 }}>
                <span className="num" style={{ fontSize: 24, fontWeight: 700 }}>
                  {fmt(tot.monthlyTotal)} <span className="faint" style={{ fontSize: 13 }}>{unit}/mo</span>
                </span>
              </div>
              <StackedBar segments={composition} height={14} />
              <div className="legend" style={{ marginTop: 14, flexDirection: "column", gap: 9 }}>
                {composition.map((c, i) => (
                  <div className="li" key={i} style={{ width: "100%" }}>
                    <span className="sw" style={{ background: c.color }} />
                    <span>{c.label}</span>
                    <div className="spacer" />
                    <span className="mono cell-strong" style={{ color: "var(--text)" }}>
                      {fmt(c.value)}
                    </span>
                    <span className="faint mono" style={{ fontSize: 11 }}>
                      {((c.value / tot.monthlyTotal) * 100).toFixed(0)}%
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </Panel>
      </div>

      <div className="grid grid-3" style={{ marginBottom: 16 }}>
        <Panel title="Cost by data center" right={<span className="linkish" style={{ fontSize: 12 }} onClick={() => nav("datacenters")}>View all</span>}>
          {viz === "donut" ? <Donut data={byDc} size={150} fmt={(v) => fmt(v)} centerLabel="total/mo" centerValue={fmt(byDc.reduce((s, d) => s + d.value, 0))} /> : <HBarList data={byDc} fmt={(v) => fmt(v)} />}
        </Panel>
        <Panel title="Top services by cost" right={<span className="linkish" style={{ fontSize: 12 }} onClick={() => nav("reports")}>Report</span>}>
          <HBarList data={topSvc} fmt={(v) => fmt(v)} />
        </Panel>
        <Panel title="Fleet health">
          <div className="col" style={{ gap: 16 }}>
            <div>
              <div className="kpi-mini" style={{ marginBottom: 6, display: "flex", justifyContent: "space-between" }}>
                <span>Avg CPU utilization</span>
                <span className="mono">{(avgCpu * 100).toFixed(0)}%</span>
              </div>
              <div className="minibar" style={{ height: 8 }}>
                <span style={{ width: avgCpu * 100 + "%", background: avgCpu > 0.85 ? "var(--bad)" : avgCpu > 0.65 ? "var(--warn)" : "var(--good)" }} />
              </div>
            </div>
            <div>
              <div className="kpi-mini" style={{ marginBottom: 6, display: "flex", justifyContent: "space-between" }}>
                <span>Avg RAM utilization</span>
                <span className="mono">{(avgRam * 100).toFixed(0)}%</span>
              </div>
              <div className="minibar" style={{ height: 8 }}>
                <span style={{ width: avgRam * 100 + "%", background: avgRam > 0.85 ? "var(--bad)" : avgRam > 0.65 ? "var(--warn)" : "var(--good)" }} />
              </div>
            </div>
            <div className="divider" style={{ margin: 0 }} />
            <div className="row">
              <span className="muted" style={{ fontSize: 13 }}>Assets fully paid off</span>
              <div className="spacer" />
              <Badge kind="good">{paidOff} / {assets.length}</Badge>
            </div>
            <div className="row">
              <span className="muted" style={{ fontSize: 13 }}>Still amortizing</span>
              <div className="spacer" />
              <Badge kind="info">{tot.activeAmort}</Badge>
            </div>
            <div className="row">
              <span className="muted" style={{ fontSize: 13 }}>Kubernetes nodes</span>
              <div className="spacer" />
              <span className="cell-mono">{assets.filter((a) => a.role === "k8s-node").length}</span>
            </div>
            <div className="row">
              <span className="muted" style={{ fontSize: 13 }}>VM hosts</span>
              <div className="spacer" />
              <span className="cell-mono">{assets.filter((a) => a.role === "vm-host").length}</span>
            </div>
          </div>
        </Panel>
      </div>
    </div>
  );
}
