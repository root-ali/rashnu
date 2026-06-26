/* Login page */
import { useState } from "react";
import { Icon } from "../components/icons";
import { Field } from "../components/ui";
import { loginApi, setToken } from "../lib/api";
import type { Role, User } from "../lib/types";

function nameFromEmail(email: string): string {
  const local = email.split("@")[0] ?? email;
  return local.charAt(0).toUpperCase() + local.slice(1);
}

export function LoginPage({ onLogin, theme, toggleTheme }: { onLogin: (u: User) => void; theme: string; toggleTheme: () => void }) {
  const [email, setEmail] = useState("");
  const [pw, setPw] = useState("");
  const [err, setErr] = useState("");
  const [loading, setLoading] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr("");
    setLoading(true);
    try {
      const resp = await loginApi(email.trim().toLowerCase(), pw);
      setToken(resp.token);

      const user: User = {
        id: resp.email,
        name: resp.full_name || nameFromEmail(resp.email),
        email: resp.email,
        role: resp.role as Role,
        password: "",
        active: true,
        lastSeen: new Date().toISOString().slice(0, 10),
      };
      onLogin(user);
    } catch (e) {
      setErr(e instanceof Error ? e.message : "Login failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-wrap">
      <div className="login-grid-bg" />
      <button className="icon-btn" onClick={toggleTheme} style={{ position: "fixed", top: 18, right: 18, zIndex: 5 }} title="Toggle theme">
        <Icon name={theme === "dark" ? "sun" : "moon"} size={18} />
      </button>
      <div className="login-card" style={{ position: "relative", zIndex: 2 }}>
        <div className="row" style={{ gap: 11, marginBottom: 22 }}>
          <img src="/logo.png" alt="rashnu" style={{ width: 38, height: 38, borderRadius: 10, objectFit: "cover", flexShrink: 0 }} />
          <div>
            <div className="brand-name" style={{ fontSize: 17 }}>
              rashnnu
            </div>
            <div className="brand-sub">Infrastructure Cost Manager</div>
          </div>
        </div>
        <div style={{ fontSize: 20, fontWeight: 700, letterSpacing: "-0.02em", marginBottom: 4 }}>Sign in</div>
        <div className="muted" style={{ fontSize: 13, marginBottom: 20 }}>
          Track depreciation, colo &amp; per-service cost.
        </div>
        <form onSubmit={submit}>
          <Field label="Email">
            <input className="input" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="you@rashnnu.io" autoFocus disabled={loading} />
          </Field>
          <Field label="Password">
            <input className="input" type="password" value={pw} onChange={(e) => setPw(e.target.value)} placeholder="••••••" disabled={loading} />
          </Field>
          {err && (
            <div style={{ color: "var(--bad)", fontSize: 12.5, marginBottom: 12, display: "flex", gap: 6, alignItems: "center" }}>
              <Icon name="alert" size={14} /> {err}
            </div>
          )}
          <button className="btn primary" type="submit" disabled={loading} style={{ width: "100%", justifyContent: "center", padding: "10px" }}>
            {loading ? (
              "Signing in…"
            ) : (
              <>
                <Icon name="logout" size={16} style={{ transform: "scaleX(-1)" }} /> Sign in
              </>
            )}
          </button>
        </form>
      </div>
    </div>
  );
}