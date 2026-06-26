/* User management page */
import { useState } from "react";
import { Icon } from "../components/icons";
import { Badge, Field, Modal, Panel, Stat } from "../components/ui";
import type { Ctx, Role, User } from "../lib/types";

export function UsersPage({ ctx }: { ctx: Ctx }) {
  const { users, isAdmin, me } = ctx;
  const [editing, setEditing] = useState<Partial<User> | null>(null);

  const admins = users.filter((u) => u.role === "admin").length;
  const active = users.filter((u) => u.active).length;

  if (!isAdmin) {
    return (
      <div className="content-inner">
        <Panel>
          <div className="empty">
            <Icon name="lock" size={28} style={{ color: "var(--text-3)", marginBottom: 10 }} />
            <div style={{ fontSize: 15, fontWeight: 600, color: "var(--text-2)" }}>Admins only</div>
            <div style={{ marginTop: 4 }}>You need an administrator role to manage users.</div>
          </div>
        </Panel>
      </div>
    );
  }

  return (
    <div className="content-inner">
      <div className="grid grid-3" style={{ marginBottom: 16 }}>
        <Stat label="Total users" value={users.length} icon="users" accent="var(--accent)" meta={<span>{active} active</span>} />
        <Stat label="Administrators" value={admins} icon="shield" accent="var(--bad)" meta={<span>full read/write access</span>} />
        <Stat label="Viewers" value={users.length - admins} icon="eye" accent="var(--info)" meta={<span>read-only access</span>} />
      </div>

      <Panel
        noBody
        title="Users &amp; access"
        sub="Admins can edit infrastructure; viewers have read-only access"
        right={
          <button className="btn primary" onClick={() => setEditing({})}>
            <Icon name="plus" size={15} /> Invite user
          </button>
        }
      >
        <div className="tbl-wrap">
          <table className="tbl">
            <thead>
              <tr>
                <th>User</th>
                <th>Email</th>
                <th>Role</th>
                <th>Status</th>
                <th>Created</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id}>
                  <td>
                    <div className="row gap-8">
                      <div className="brand-mark" style={{ width: 30, height: 30, fontSize: 12, background: u.role === "admin" ? "var(--bad-bg)" : "var(--info-bg)", color: u.role === "admin" ? "var(--bad)" : "var(--info)", boxShadow: "none" }}>
                        {u.name.split(" ").map((p) => p[0]).join("").slice(0, 2)}
                      </div>
                      <div className="cell-strong">
                        {u.name}
                        {u.id === me?.id && <span className="tag" style={{ marginLeft: 7 }}>you</span>}
                      </div>
                    </div>
                  </td>
                  <td className="cell-mono cell-sub">{u.email}</td>
                  <td>{u.role === "admin" ? <Badge kind="bad"><Icon name="shield" size={11} /> Admin</Badge> : <Badge kind="info"><Icon name="eye" size={11} /> Viewer</Badge>}</td>
                  <td>{u.active ? <Badge kind="good" dot>Active</Badge> : <Badge kind="neutral">Disabled</Badge>}</td>
                  <td className="cell-mono cell-sub">{u.lastSeen}</td>
                  <td className="r">
                    <div className="row" style={{ justifyContent: "flex-end", gap: 2 }}>
                      <button className="icon-btn" onClick={() => setEditing(u)} title="Edit">
                        <Icon name="edit" size={15} />
                      </button>
                      {u.id !== me?.id && (
                        <button className="icon-btn" onClick={() => ctx.deleteUser(u.id)} title="Remove">
                          <Icon name="trash" size={15} />
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Panel>

      {editing && (
        <UserForm
          user={editing}
          onClose={() => setEditing(null)}
          onSave={(u) => {
            ctx.upsertUser(u);
            setEditing(null);
          }}
        />
      )}
    </div>
  );
}

function UserForm({ user, onClose, onSave }: { user: Partial<User>; onClose: () => void; onSave: (u: User) => void }) {
  const isNew = !user.id;
  const [f, setF] = useState<User>(() => ({
    id: user.id || "u" + Math.random().toString(36).slice(2, 6),
    name: user.name || "",
    email: user.email || "",
    role: user.role || "viewer",
    password: user.password || "viewer",
    active: user.active != null ? user.active : true,
    lastSeen: user.lastSeen || "2026-06-03",
  }));
  const set = <K extends keyof User>(k: K, v: User[K]) => setF((p) => ({ ...p, [k]: v }));
  return (
    <Modal
      title={isNew ? "Invite user" : "Edit " + f.name}
      onClose={onClose}
      footer={
        <>
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn primary" onClick={() => onSave(f)} disabled={!f.name || !f.email}>
            {isNew ? "Send invite" : "Save"}
          </button>
        </>
      }
    >
      <div className="field-row">
        <Field label="Full name">
          <input className="input" value={f.name} onChange={(e) => set("name", e.target.value)} placeholder="Sara Karimi" />
        </Field>
      </div>
      <Field label="Email">
        <input className="input mono" value={f.email} onChange={(e) => set("email", e.target.value)} placeholder="sara@rashnnu.io" />
      </Field>
      <div className="field-row">
        <Field label="Role" hint={f.role === "admin" ? "Can create, edit & delete everything" : "Read-only access to all data"}>
          <select className="select" value={f.role} onChange={(e) => set("role", e.target.value as Role)}>
            <option value="viewer">Viewer (read-only)</option>
            <option value="admin">Administrator</option>
          </select>
        </Field>
        <Field label="Status">
          <select className="select" value={f.active ? "1" : "0"} onChange={(e) => set("active", e.target.value === "1")}>
            <option value="1">Active</option>
            <option value="0">Disabled</option>
          </select>
        </Field>
      </div>
      <Field label="Password" hint="Demo only — stored in plaintext in local state">
        <input className="input mono" value={f.password} onChange={(e) => set("password", e.target.value)} />
      </Field>
    </Modal>
  );
}
