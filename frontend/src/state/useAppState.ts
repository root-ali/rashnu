/* App-wide state: auth, data CRUD, theme, derived costs. Persisted to localStorage. */
import { useEffect, useMemo, useRef, useState } from "react";
import { IC } from "../lib/data";
import type { Asset, CostReports, Ctx, DataCenter, Derived, Service, ToastKind, Tweaks, User } from "../lib/types";
import type { ToastItem } from "../components/ui";
import {
  addPodWorkloadApi,
  addVMWorkloadApi,
  calculateDailyCostsApi,
  clearToken,
  createDataCenterApi,
  createInfraHardwareApi,
  createServerApi,
  createServiceApi,
  createUserApi,
  deleteDataCenterApi,
  deleteInfraHardwareApi,
  deletePodWorkloadApi,
  deleteServerApi,
  deleteServiceApi,
  fulfillCostReportsApi,
  deleteUserApi,
  deleteVMWorkloadApi,
  getCostTrendApi,
  getMonthlyCostReportsApi,
  getServiceApi,
  getToken,
  listDataCentersApi,
  listInfraHardwaresApi,
  listServersApi,
  listServicesApi,
  listUsersApi,
  updateDataCenterApi,
  updateInfraHardwareApi,
  updateServerApi,
  updateServiceApi,
  updateUserApi,
} from "../lib/api";
import {
  mapAssetToInfraCreate,
  mapAssetToInfraUpdate,
  mapAssetToServerCreate,
  mapAssetToServerUpdate,
  mapBackendDataCenter,
  mapBackendInfraHardware,
  mapBackendServer,
  mapBackendService,
  mapBackendUser,
  mapDataCenterToBackend,
} from "../lib/mappers";

export const ACCENTS: Record<string, string> = {
  "#3d8bff": "#00c2b3",
  "#7c6cff": "#22b8cf",
  "#ff7849": "#ffb648",
  "#2ec98a": "#39d4c4",
};
export const FONTS: Record<string, string> = {
  Archivo: '"Archivo", Helvetica, Arial, sans-serif',
  Helvetica: '"Helvetica Neue", Helvetica, Arial, sans-serif',
  "Space Grotesk": '"Space Grotesk", Helvetica, sans-serif',
};

function load<T>(key: string, fallback: T): T {
  try {
    const v = localStorage.getItem(key);
    return v ? (JSON.parse(v) as T) : fallback;
  } catch {
    return fallback;
  }
}
function save<T>(key: string, v: T): void {
  try {
    localStorage.setItem(key, JSON.stringify(v));
  } catch {
    /* ignore */
  }
}

export interface AppState {
  theme: string;
  setTheme: (v: string) => void;
  me: User | null;
  setMe: (u: User | null) => void;
  page: string;
  setPage: (p: string) => void;
  users: User[];
  toasts: ToastItem[];
  toast: (msg: string, kind?: ToastKind) => void;
  isAdmin: boolean;
  loading: boolean;
  derived: Derived;
  ctx: Ctx;
  resetData: () => void;
}

export function useAppState(t: Tweaks): AppState {
  const [theme, setTheme] = useState<string>(() => load("ic_theme", "dark"));
  const [me, setMeRaw] = useState<User | null>(() => load<User | null>("ic_me", null));
  const [page, setPage] = useState<string>(() => load("ic_page", "dashboard"));

  const [dcs, setDcs] = useState<DataCenter[]>(() => load("ic_dcs", []));
  const [assets, setAssets] = useState<Asset[]>(() => load("ic_assets", []));
  const [services, setServices] = useState<Service[]>(() => load("ic_services", []));
  const [users, setUsers] = useState<User[]>(() => load("ic_users", []));
  const [costReports, setCostReports] = useState<CostReports>({ monthly: [], trend: [] });
  const [toasts, setToasts] = useState<ToastItem[]>([]);
  const [loading, setLoading] = useState(false);

  const prevMeRef = useRef<User | null>(me);

  useEffect(() => save("ic_theme", theme), [theme]);
  useEffect(() => save("ic_me", me), [me]);
  useEffect(() => save("ic_page", page), [page]);
  useEffect(() => save("ic_dcs", dcs), [dcs]);
  useEffect(() => save("ic_assets", assets), [assets]);
  useEffect(() => save("ic_services", services), [services]);
  useEffect(() => save("ic_users", users), [users]);

  useEffect(() => {
    const r = document.documentElement;
    r.setAttribute("data-theme", theme);
    r.style.setProperty("--accent", t.accent);
    r.style.setProperty("--accent-2", ACCENTS[t.accent] || "#00c2b3");
    r.style.setProperty("--font-ui", FONTS[t.fontUi] || FONTS.Archivo);
  }, [theme, t.accent, t.fontUi]);

  // ── Toast helpers ─────────────────────────────────────────────────────────

  const addToast = (msg: string, kind?: ToastKind) => {
    const id = Math.random().toString(36).slice(2);
    setToasts((p) => [...p, { id, msg, kind }]);
    setTimeout(() => setToasts((p) => p.filter((x) => x.id !== id)), 2600);
  };

  // ── Backend data refresh ──────────────────────────────────────────────────

  const refreshData = async () => {
    if (!getToken()) return;
    setLoading(true);
    try {
      const [dcsRes, serversRes, hwsRes, svcsRes] = await Promise.all([
        listDataCentersApi(),
        listServersApi(),
        listInfraHardwaresApi(),
        listServicesApi(),
      ]);
      setDcs((dcsRes.datacenters ?? []).map(mapBackendDataCenter));
      setAssets([
        ...(serversRes.servers ?? []).map(mapBackendServer),
        ...(hwsRes.hardwares ?? []).map(mapBackendInfraHardware),
      ]);
      setServices((svcsRes.services ?? []).map(mapBackendService));
      if (me?.role === "admin") {
        const usersRes = await listUsersApi();
        setUsers((usersRes ?? []).map(mapBackendUser));
      }
    } catch (e) {
      addToast("Failed to load data: " + String(e), "error");
    } finally {
      setLoading(false);
    }

    // Cost reports are fetched independently — a failure here shows a warning
    // but never blocks the main inventory/services data from loading.
    try {
      const [monthlyRes, trendRes] = await Promise.all([
        getMonthlyCostReportsApi(),
        getCostTrendApi(12),
      ]);
      setCostReports({
        monthlyMonth: monthlyRes?.month ? String(monthlyRes.month).slice(0, 7) : undefined,
        monthly: (monthlyRes?.reports ?? []).map((r) => ({
          service_id: r.service_id,
          service_name: r.service_name,
          team: r.team,
          platform: r.platform,
          month: r.month,
          total_cost: r.total_cost,
          daily_avg: r.daily_avg,
          days_recorded: r.days_recorded,
          cpu_cost: r.cpu_cost ?? 0,
          ram_cost: r.ram_cost ?? 0,
          ssd_cost: r.ssd_cost ?? 0,
          hdd_cost: r.hdd_cost ?? 0,
          datacenter_breakdown: (r.datacenter_breakdown ?? []).map((d) => ({
            datacenter_id: d.datacenter_id,
            datacenter_name: d.datacenter_name,
            total_cost: d.total_cost,
            daily_avg: d.daily_avg,
            cpu_cost: d.cpu_cost ?? 0,
            ram_cost: d.ram_cost ?? 0,
            ssd_cost: d.ssd_cost ?? 0,
            hdd_cost: d.hdd_cost ?? 0,
          })),
        })),
        trend: (trendRes?.points ?? []).map((p) => ({
          month: p.month,
          total_cost: p.total_cost,
        })),
      });
    } catch (e) {
      addToast("Failed to load cost reports: " + String(e), "error");
    }
  };

  // Fetch from backend on page reload when a session token already exists.
  useEffect(() => {
    if (getToken()) refreshData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Fetch from backend on fresh login (me transitions null → non-null).
  useEffect(() => {
    const prev = prevMeRef.current;
    prevMeRef.current = me;
    if (me && !prev) refreshData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [me]);

  // ── Auth wrapper ──────────────────────────────────────────────────────────

  const setMe = (u: User | null) => {
    if (!u) {
      clearToken();
      setDcs([]);
      setAssets([]);
      setServices([]);
      setUsers([]);
    }
    setMeRaw(u);
  };

  const isAdmin = !!me && me.role === "admin";

  // ── Helpers ───────────────────────────────────────────────────────────────

  const dcRegion = (dcId: string): string => dcs.find((d) => d.id === dcId)?.region ?? "";

  // ── DataCenter CRUD ───────────────────────────────────────────────────────

  const upsertDc = async (item: DataCenter) => {
    const isNew = !dcs.some((x) => x.id === item.id);
    if (isNew) {
      try {
        await createDataCenterApi(mapDataCenterToBackend(item));
        await refreshData();
        addToast("Added " + item.name);
      } catch (e) {
        addToast("Failed to add data center: " + String(e), "error");
      }
    } else {
      setDcs((arr) => arr.map((x) => (x.id === item.id ? item : x)));
      try {
        await updateDataCenterApi({ id: item.id, ...mapDataCenterToBackend(item) });
        addToast("Saved " + item.name);
      } catch (e) {
        addToast("Failed to save: " + String(e), "error");
        await refreshData();
      }
    }
  };

  const deleteDc = async (id: string) => {
    if (!window.confirm("Delete this data center?")) return;
    setDcs((arr) => arr.filter((x) => x.id !== id));
    addToast("Deleted data center", "error");
    try {
      await deleteDataCenterApi(id);
    } catch (e) {
      addToast("Failed to delete: " + String(e), "error");
      await refreshData();
    }
  };

  // ── Asset (Server / Infra Hardware) CRUD ─────────────────────────────────

  const upsertAsset = async (item: Asset) => {
    const isNew = !assets.some((x) => x.id === item.id);
    const region = dcRegion(item.dcId);
    const isServer = item.type === "server";

    if (isNew) {
      try {
        if (isServer) {
          await createServerApi(mapAssetToServerCreate(item, region));
        } else {
          await createInfraHardwareApi(mapAssetToInfraCreate(item, region));
        }
        // Re-fetch so the local state uses the server-assigned UUID.
        await refreshData();
        addToast("Added " + item.name);
      } catch (e) {
        addToast("Failed to add asset: " + String(e), "error");
      }
    } else {
      setAssets((arr) => arr.map((x) => (x.id === item.id ? item : x)));
      try {
        if (isServer) {
          await updateServerApi(mapAssetToServerUpdate(item, region));
        } else {
          await updateInfraHardwareApi(mapAssetToInfraUpdate(item, region));
        }
        addToast("Saved " + item.name);
      } catch (e) {
        addToast("Failed to save: " + String(e), "error");
        await refreshData();
      }
    }
  };

  const deleteAsset = async (id: string) => {
    if (!window.confirm("Delete this asset?")) return;
    const target = assets.find((x) => x.id === id);
    setAssets((arr) => arr.filter((x) => x.id !== id));
    addToast("Deleted asset", "error");
    try {
      if (!target || target.type === "server") {
        await deleteServerApi(id);
      } else {
        await deleteInfraHardwareApi(id);
      }
    } catch (e) {
      addToast("Failed to delete: " + String(e), "error");
      await refreshData();
    }
  };

  // ── Service CRUD ──────────────────────────────────────────────────────────

  const syncWorkloads = async (serviceId: string, item: Service, currentAssets: typeof assets) => {
    for (let i = 0; i < item.workloads.length; i++) {
      const w = item.workloads[i];
      if (w.kind === "pod") {
        // Pods: hostId is a k8s-node server asset ID; derive datacenter from it.
        const asset = currentAssets.find((a) => a.id === w.hostId);
        const datacenter_id = asset?.dcId ?? "";
        await addPodWorkloadApi(serviceId, {
          service_id: serviceId,
          name: w.vmName || item.name,
          namespace: item.namespace || "default",
          vcpu: Math.round(w.cpuRequest),
          memory_gb: Math.round(w.ramRequestGB),
          ssd_gb: w.ssdGB ?? 0,
          hdd_gb: w.hddGB ?? 0,
          gpus: w.gpus ?? 0,
          cpu_usage: w.cpuUsage ?? 0,
          memory_usage_gb: w.ramUsageGB ?? 0,
          datacenter_id,
        });
      } else {
        // VMs: hostId IS the datacenter_id (selected from datacenter list in the form).
        await addVMWorkloadApi(serviceId, {
          service_id: serviceId,
          name: w.vmName || item.name + "-" + String(i + 1),
          vcpu: Math.round(w.cpuRequest),
          memory_gb: Math.round(w.ramRequestGB),
          ssd_gb: w.ssdGB ?? 0,
          hdd_gb: w.hddGB ?? 0,
          gpus: w.gpus ?? 0,
          cpu_usage: w.cpuUsage ?? 0,
          memory_usage_gb: w.ramUsageGB ?? 0,
          datacenter_id: w.hostId,
        });
      }
    }
  };

  const upsertService = async (item: Service) => {
    const isNew = !services.some((x) => x.id === item.id);
    if (isNew) {
      try {
        const { id } = await createServiceApi({
          name: item.name,
          description: "",
          platform: item.platform,
          team: item.team,
        });
        await syncWorkloads(id, item, assets);
        await refreshData();
        addToast("Added " + item.name);
      } catch (e) {
        addToast("Failed to add service: " + String(e), "error");
      }
    } else {
      try {
        await updateServiceApi({ id: item.id, name: item.name, description: "", team: item.team });
        // Replace all workloads: delete existing, create new
        const current = await getServiceApi(item.id);
        for (const p of current.pods ?? []) await deletePodWorkloadApi(item.id, p.id);
        for (const v of current.vms ?? []) await deleteVMWorkloadApi(item.id, v.id);
        await syncWorkloads(item.id, item, assets);
        await refreshData();
        addToast("Saved " + item.name);
      } catch (e) {
        addToast("Failed to save service: " + String(e), "error");
        await refreshData();
      }
    }
  };

  const deleteService = async (id: string) => {
    if (!window.confirm("Delete this service?")) return;
    setServices((arr) => arr.filter((x) => x.id !== id));
    try {
      await deleteServiceApi(id);
      addToast("Deleted service");
    } catch (e) {
      addToast("Failed to delete service: " + String(e), "error");
      await refreshData();
    }
  };

  // ── User CRUD ─────────────────────────────────────────────────────────────

  const upsertUser = async (item: User) => {
    const isNew = !users.some((x) => x.id === item.id);
    setUsers((arr) => {
      const i = arr.findIndex((x) => x.id === item.id);
      if (i === -1) return [...arr, item];
      const cp = [...arr];
      cp[i] = item;
      return cp;
    });
    if (isNew) {
      try {
        await createUserApi({
          email: item.email,
          full_name: item.name,
          password: item.password,
          role: item.role,
          status: item.active ? "active" : "inactive",
        });
        await refreshData();
        addToast("Added " + item.name);
      } catch (e) {
        addToast("Failed to create user: " + String(e), "error");
        await refreshData();
      }
    } else {
      try {
        await updateUserApi({
          id: item.id,
          email: item.email,
          full_name: item.name,
          role: item.role,
          status: item.active ? "active" : "inactive",
        });
        addToast("Saved " + item.name);
      } catch (e) {
        addToast("Failed to save user: " + String(e), "error");
        await refreshData();
      }
    }
  };

  const deleteUser = async (id: string) => {
    if (!window.confirm("Delete this user?")) return;
    setUsers((arr) => arr.filter((x) => x.id !== id));
    try {
      await deleteUserApi(id);
      addToast("Deleted user");
    } catch (e) {
      addToast("Failed to delete user: " + String(e), "error");
      await refreshData();
    }
  };

  // ── Derived calculations ──────────────────────────────────────────────────

  const derived: Derived = useMemo(() => {
    const assetCosts = IC.assetCosts(assets, dcs);
    const alloc = IC.allocate(assets, dcs, services, { method: t.allocMethod, cpuWeight: t.cpuWeight / 100, includeIdle: t.includeIdle });
    const totals = IC.totals(assets, dcs, services);
    const hostUtil = IC.hostUtil(assets, services);
    const trend = IC.trend(totals.monthlyTotal);
    return { assetCosts, alloc, totals, hostUtil, trend };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [assets, services, dcs, t.allocMethod, t.cpuWeight, t.includeIdle]);

  const fmt = (v: number) => IC.fmtMoney(v, { unit: t.unit });
  const fmtFull = (v: number) => IC.fmtMoneyFull(v, { unit: t.unit });
  const unit = IC.unitLabel(t.unit);

  const resetData = async () => {
    if (!getToken()) {
      addToast("Not logged in", "error");
      return;
    }
    if (!window.confirm("Refresh all data from the backend?")) return;
    await refreshData();
    addToast("Data refreshed");
  };

  const calculateDailyCosts = async (date?: string) => {
    try {
      await calculateDailyCostsApi(date);
      await refreshData();
      addToast("Daily costs calculated");
    } catch (e) {
      addToast("Failed to calculate costs: " + String(e), "error");
    }
  };

  const fulfillCostReports = async (startTime: string) => {
    try {
      await fulfillCostReportsApi(startTime);
      await refreshData();
      addToast("Cost reports backfilled from " + startTime);
    } catch (e) {
      addToast("Failed to backfill cost reports: " + String(e), "error");
    }
  };

  const ctx: Ctx = {
    dcs,
    assets,
    services,
    users,
    derived,
    me,
    isAdmin,
    t,
    fmt,
    fmtFull,
    unit,
    toast: addToast,
    nav: setPage,
    upsertAsset,
    deleteAsset,
    upsertDc,
    deleteDc,
    upsertService,
    deleteService,
    upsertUser,
    deleteUser,
    costReports,
    calculateDailyCosts,
    fulfillCostReports,
  };

  return { theme, setTheme, me, setMe, page, setPage, users, toasts, toast: addToast, isAdmin, loading, derived, ctx, resetData };
}