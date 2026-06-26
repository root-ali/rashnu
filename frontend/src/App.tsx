/* rashnu · Infrastructure Cost Manager — app shell, routing, auth */
import { useState, useEffect } from "react";
import { updatePasswordApi, logoutApi, getPrometheusConfigApi, updatePrometheusConfigApi, type PrometheusConfig } from "./lib/api";
import { Icon, type IconName } from "./components/icons";
import { Badge, Field, Modal, Toasts } from "./components/ui";
import { TweaksPanel, TweakButton, TweakColor, TweakRadio, TweakSection, TweakSelect, TweakSlider, TweakToggle } from "./components/TweaksPanel";
import { useTweaks } from "./lib/tweaks";
import { useAppState, ACCENTS, FONTS } from "./state/useAppState";
import type { Tweaks } from "./lib/types";
import { LoginPage } from "./pages/Login";
import { DashboardPage } from "./pages/Dashboard";
import { HardwarePage } from "./pages/Hardware";
import { DataCentersPage } from "./pages/DataCenters";
import { ServicesPage } from "./pages/Services";
import { ReportsPage } from "./pages/Reports";
import { UsersPage } from "./pages/Users";

const TWEAK_DEFAULTS: Tweaks = {
  navStyle: "sidebar",
  vizStyle: "area",
  reportStyle: "mixed",
  allocMethod: "usage",
  cpuWeight: 50,
  includeIdle: false,
  unit: "Rial",
  accent: "#3d8bff",
  fontUi: "Archivo",
};

interface NavEntry {
  id: string;
  label: string;
  icon: IconName;
}
const NAV: NavEntry[] = [
  { id: "dashboard", label: "Overview", icon: "dashboard" },
  { id: "hardware", label: "Hardware", icon: "server" },
  { id: "datacenters", label: "Data Centers", icon: "datacenter" },
  { id: "services", label: "Services", icon: "services" },
  { id: "reports", label: "Cost Reports", icon: "report" },
];
const ADMIN_NAV: NavEntry[] = [{ id: "users", label: "Users", icon: "users" }];

const PAGES = {
  dashboard: DashboardPage,
  hardware: HardwarePage,
  datacenters: DataCentersPage,
  services: ServicesPage,
  reports: ReportsPage,
  users: UsersPage,
} as const;

const SUBTITLES: Record<string, string> = {
  dashboard: "Total infrastructure cost & trends",
  hardware: "Servers, switches & storage with amortization",
  datacenters: "Facilities & colocation fees",
  services: "Service definitions & resource attribution",
  reports: "Per-service cost — Kubernetes & VM",
  users: "Access control & roles",
};

export default function App() {
  const [t, setTweak] = useTweaks<Tweaks>(TWEAK_DEFAULTS);
  const app = useAppState(t);
  const { theme, setTheme, me, setMe, page, setPage, users, toasts, toast, isAdmin, ctx, resetData } = app;
  const [menuOpen, setMenuOpen] = useState(false);
  const [tweaksOpen, setTweaksOpen] = useState(false);
  const [changePwOpen, setChangePwOpen] = useState(false);
  const [promCfg, setPromCfg] = useState<PrometheusConfig>({ url: "", enabled: false });
  const [promSaving, setPromSaving] = useState(false);

  useEffect(() => {
    const handler = () => setMe(null);
    window.addEventListener("auth:token-expired", handler);
    return () => window.removeEventListener("auth:token-expired", handler);
  }, [setMe]);

  // Load Prometheus config when logged in
  useEffect(() => {
    if (!me) return;
    getPrometheusConfigApi().then(setPromCfg).catch(() => {});
  }, [me]);

  const savePromCfg = async () => {
    setPromSaving(true);
    try {
      await updatePrometheusConfigApi(promCfg);
      toast("Prometheus config saved");
    } catch (e) {
      toast("Failed to save Prometheus config: " + String(e), "error");
    } finally {
      setPromSaving(false);
    }
  };

  const tweaksUI = (
    <TweaksPanel open={tweaksOpen} onClose={() => setTweaksOpen(false)}>
      <TweakSection label="Layout" />
      <TweakRadio label="Navigation" value={t.navStyle} options={["sidebar", "topbar"]} onChange={(v) => setTweak("navStyle", v as Tweaks["navStyle"])} />
      <TweakRadio label="Dashboard viz" value={t.vizStyle} options={["area", "bars", "donut"]} onChange={(v) => setTweak("vizStyle", v as Tweaks["vizStyle"])} />
      <TweakRadio label="Report view" value={t.reportStyle} options={["charts", "tables", "mixed"]} onChange={(v) => setTweak("reportStyle", v as Tweaks["reportStyle"])} />

      <TweakSection label="Cost allocation model" />
      <TweakSelect
        label="Split host cost by"
        value={t.allocMethod}
        options={[
          { value: "usage", label: "CPU/RAM usage (Prometheus)" },
          { value: "requests", label: "Reserved requests" },
          { value: "even", label: "Even split" },
          { value: "manual", label: "Manual weight" },
        ]}
        onChange={(v) => setTweak("allocMethod", v as Tweaks["allocMethod"])}
      />
      <TweakSlider label="CPU ⟷ RAM weight" value={t.cpuWeight} min={0} max={100} unit="% CPU" onChange={(v) => setTweak("cpuWeight", v)} />
      <TweakToggle label="Show idle / unallocated headroom" value={t.includeIdle} onChange={(v) => setTweak("includeIdle", v)} />

      <TweakSection label="Display" />
      <TweakRadio label="Currency" value={t.unit} options={["Rial", "Toman"]} onChange={(v) => setTweak("unit", v as Tweaks["unit"])} />
      <TweakColor label="Accent" value={t.accent} options={Object.keys(ACCENTS)} onChange={(v) => setTweak("accent", v as string)} />
      <TweakSelect label="UI font" value={t.fontUi} options={Object.keys(FONTS)} onChange={(v) => setTweak("fontUi", v)} />
      <TweakToggle label="Dark mode" value={theme === "dark"} onChange={(v) => setTheme(v ? "dark" : "light")} />
      <TweakButton label="Reset sample data" onClick={resetData} />
      {isAdmin && (
        <>
          <TweakSection label="Integrations" />
          <TweakToggle
            label="Prometheus metrics"
            value={promCfg.enabled}
            onChange={(v) => setPromCfg((p) => ({ ...p, enabled: v }))}
          />
          {promCfg.enabled && (
            <div className="col" style={{ gap: 4 }}>
              <label style={{ fontSize: 11, color: "var(--text-2)", fontWeight: 500 }}>Prometheus URL</label>
              <input
                className="input mono"
                style={{ fontSize: 11.5 }}
                placeholder="http://prometheus:9090"
                value={promCfg.url}
                onChange={(e) => setPromCfg((p) => ({ ...p, url: e.target.value }))}
              />
            </div>
          )}
          <TweakButton label={promSaving ? "Saving…" : "Save Prometheus config"} onClick={savePromCfg} />
        </>
      )}
    </TweaksPanel>
  );

  const tweaksFab = (
    <button
      className="icon-btn"
      onClick={() => setTweaksOpen((o) => !o)}
      title="Tweaks"
      style={{ position: "fixed", left: 16, bottom: 16, zIndex: 100, width: 38, height: 38, background: "var(--panel)", border: "1px solid var(--border-2)", boxShadow: "var(--shadow)", color: "var(--text-2)" }}
    >
      <Icon name="settings" size={18} />
    </button>
  );

  if (!me) {
    return (
      <>
        <LoginPage
          theme={theme}
          toggleTheme={() => setTheme(theme === "dark" ? "light" : "dark")}
          onLogin={(u) => {
            setMe(u);
            setPage("dashboard");
            toast("Welcome back, " + u.name.split(" ")[0]);
          }}
        />
        {tweaksFab}
        {tweaksUI}
      </>
    );
  }

  const navItems = isAdmin ? [...NAV, ...ADMIN_NAV] : NAV;
  const current = navItems.find((n) => n.id === page) || NAV[0];
  const PageComp = (PAGES as Record<string, (p: { ctx: typeof ctx }) => JSX.Element>)[page] || DashboardPage;

  const initials = (name: string) =>
    name
      .split(" ")
      .map((p) => p[0])
      .join("")
      .slice(0, 2);

  const Brand = (
    <div className="brand">
      <img src="/logo.png" alt="rashnu" style={{ width: 36, height: 36, borderRadius: 8, objectFit: "cover", flexShrink: 0 }} />
      <div>
        <div className="brand-name">rashnnu</div>
        <div className="brand-sub">cost manager</div>
      </div>
    </div>
  );

  const NavList = (
    <nav className="nav">
      {t.navStyle === "sidebar" && <div className="nav-label">Monitoring</div>}
      {navItems.map((n) => (
        <div key={n.id} className={"nav-item" + (page === n.id ? " active" : "")} onClick={() => setPage(n.id)}>
          <Icon name={n.icon} size={17} className="ico" /> {n.label}
        </div>
      ))}
    </nav>
  );

  const UserMenu = (
    <div className="user-menu">
      <button className="btn ghost" onClick={() => setMenuOpen(!menuOpen)} style={{ paddingLeft: 6 }}>
        <div className="brand-mark" style={{ width: 26, height: 26, fontSize: 11, boxShadow: "none" }}>
          {initials(me.name)}
        </div>
        <span style={{ fontWeight: 600 }}>{me.name.split(" ")[0]}</span>
        <span className={"badge " + (isAdmin ? "info" : "neutral")} style={{ fontSize: 10 }}>
          {isAdmin ? "admin" : "viewer"}
        </span>
        <Icon name="chevronDown" size={14} />
      </button>
      {menuOpen && (
        <>
          <div style={{ position: "fixed", inset: 0, zIndex: 40 }} onClick={() => setMenuOpen(false)} />
          <div className="user-pop">
            <div style={{ padding: "11px 14px", borderBottom: "1px solid var(--border)" }}>
              <div className="cell-strong">{me.name}</div>
              <div className="cell-sub">{me.email}</div>
            </div>
            <div
              className="item"
              onClick={() => {
                setTheme(theme === "dark" ? "light" : "dark");
                setMenuOpen(false);
              }}
            >
              <Icon name={theme === "dark" ? "sun" : "moon"} size={16} /> {theme === "dark" ? "Light" : "Dark"} mode
            </div>
            <div
              className="item"
              onClick={() => { setChangePwOpen(true); setMenuOpen(false); }}
            >
              <Icon name="edit" size={16} /> Change password
            </div>
            <div
              className="item"
              onClick={() => {
                setMe(null);
                setMenuOpen(false);
              }}
              style={{ color: "var(--bad)" }}
            >
              <Icon name="logout" size={16} /> Sign out
            </div>
          </div>
        </>
      )}
    </div>
  );

  return (
    <div className={"app" + (t.navStyle === "topbar" ? " topbar-layout" : "")}>
      {t.navStyle === "sidebar" ? (
        <div className="sidebar">
          {Brand}
          {NavList}
          <div className="nav-foot">
            <button className="btn" style={{ width: "100%", justifyContent: "center" }} onClick={async () => { await logoutApi().catch(() => {}); setMe(null); }}>
              <Icon name="logout" size={15} /> Sign out
            </button>
          </div>
        </div>
      ) : (
        <div className="topnav">
          {Brand}
          {NavList}
          <div className="spacer" />
          {UserMenu}
        </div>
      )}
      <div className="main">
        <div className="topbar">
          <div>
            <div className="page-title">{current.label}</div>
            <div className="page-sub">{SUBTITLES[page]}</div>
          </div>
          <div className="topbar-spacer" />
          {!isAdmin && (
            <span className="readonly-banner">
              <Icon name="eye" size={13} /> Read-only — viewer role
            </span>
          )}
          <Badge kind="neutral">
            <Icon name="money" size={12} /> {ctx.unit}
          </Badge>
          <button className="icon-btn" onClick={() => setTheme(theme === "dark" ? "light" : "dark")} title="Toggle theme">
            <Icon name={theme === "dark" ? "sun" : "moon"} size={18} />
          </button>
          {t.navStyle === "sidebar" && UserMenu}
        </div>
        <div className="content">
          <PageComp ctx={ctx} />
        </div>
      </div>
      <Toasts items={toasts} />
      {tweaksFab}
      {tweaksUI}
      {changePwOpen && (
        <ChangePasswordModal
          userId={me.id}
          onClose={() => setChangePwOpen(false)}
          onSuccess={() => {
            setChangePwOpen(false);
            toast("Password updated — signing out for security");
            setTimeout(async () => { await logoutApi().catch(() => {}); setMe(null); }, 2000);
          }}
          onError={(msg) => toast(msg, "error")}
        />
      )}
    </div>
  );
}

function ChangePasswordModal({ userId, onClose, onSuccess, onError }: {
  userId: string;
  onClose: () => void;
  onSuccess: () => void;
  onError: (msg: string) => void;
}) {
  const [f, setF] = useState({ old: "", next: "", confirm: "" });
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState("");
  const set = (k: keyof typeof f, v: string) => { setF((p) => ({ ...p, [k]: v })); setErr(""); };

  const submit = async () => {
    if (f.next !== f.confirm) { setErr("New passwords do not match"); return; }
    if (f.next.length < 6) { setErr("Password must be at least 6 characters"); return; }
    setLoading(true);
    try {
      await updatePasswordApi(userId, { old_passwrd: f.old, new_passwrd: f.next, confirm_password: f.confirm });
      onSuccess();
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Failed to update password";
      setErr(msg);
      onError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title="Change password"
      onClose={onClose}
      footer={
        <>
          <button className="btn" onClick={onClose}>Cancel</button>
          <button className="btn primary" onClick={submit} disabled={loading || !f.old || !f.next || !f.confirm}>
            {loading ? "Saving…" : "Update password"}
          </button>
        </>
      }
    >
      <Field label="Current password">
        <input className="input mono" type="password" value={f.old} onChange={(e) => set("old", e.target.value)} autoFocus />
      </Field>
      <Field label="New password">
        <input className="input mono" type="password" value={f.next} onChange={(e) => set("next", e.target.value)} />
      </Field>
      <Field label="Confirm new password">
        <input className="input mono" type="password" value={f.confirm} onChange={(e) => set("confirm", e.target.value)} />
      </Field>
      {err && (
        <div style={{ color: "var(--bad)", fontSize: 12.5, display: "flex", gap: 6, alignItems: "center", marginTop: 4 }}>
          <Icon name="alert" size={14} /> {err}
        </div>
      )}
    </Modal>
  );
}
