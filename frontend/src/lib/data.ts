/* ============================================================
   rashnnu · Infrastructure Cost Manager — data model + cost engine
   Seed data, cost math and formatters. Pure TS, no React.
   ============================================================ */

import type {
  Asset,
  AssetCost,
  AssetRole,
  AllocOpts,
  AllocResult,
  Consumer,
  DataCenter,
  HostUtil,
  MoneyOpts,
  Service,
  ServiceAlloc,
  Totals,
  TrendPoint,
  Unit,
  User,
  Workload,
} from "./types";

// ---- seeded RNG so sample data is stable across reloads ----
function mulberry32(a: number): () => number {
  return function () {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}
const rnd = mulberry32(20260529);
const pick = <T>(arr: T[]): T => arr[Math.floor(rnd() * arr.length)];
const rint = (lo: number, hi: number): number => lo + Math.floor(rnd() * (hi - lo + 1));
const rfloat = (lo: number, hi: number): number => lo + rnd() * (hi - lo);

// ---- constants ----
const HOURS_PER_MONTH = 730; // 365.25/12*24 ≈ 730.5, use 730
const NOW = new Date("2026-05-29");

// ===========================================================
// DATA CENTERS
// ===========================================================
const datacenters: DataCenter[] = [
  { id: "dc-thr1", name: "Tehran · Pardis", short: "THR-1", provider: "Pardis IDC", region: "IR-Tehran", coloMonthly: 1_850_000_000, racks: 4, power_kw: 28, tier: "Tier III" },
  { id: "dc-thr2", name: "Tehran · Asiatech", short: "THR-2", provider: "Asiatech", region: "IR-Tehran", coloMonthly: 1_240_000_000, racks: 3, power_kw: 19, tier: "Tier II" },
  { id: "dc-fra1", name: "Frankfurt · Edge", short: "FRA-1", provider: "Hetzner", region: "EU-Frankfurt", coloMonthly: 940_000_000, racks: 2, power_kw: 12, tier: "Tier III" },
];

// ===========================================================
// ASSETS (servers, switches, storage)
// ===========================================================
interface ServerModel { vendor: string; model: string; cpu: number; ram: number; watt: number; }
interface SwitchModel { vendor: string; model: string; watt: number; }
interface StorageModel { vendor: string; model: string; watt: number; ram: number; cpu: number; }

const serverModels: ServerModel[] = [
  { vendor: "Dell", model: "PowerEdge R740", cpu: 64, ram: 512, watt: 620 },
  { vendor: "Dell", model: "PowerEdge R640", cpu: 48, ram: 384, watt: 480 },
  { vendor: "HPE", model: "ProLiant DL380 G10", cpu: 56, ram: 512, watt: 560 },
  { vendor: "HPE", model: "ProLiant DL360 G10", cpu: 40, ram: 256, watt: 430 },
  { vendor: "Supermicro", model: "AS-2124US", cpu: 128, ram: 1024, watt: 780 },
  { vendor: "Supermicro", model: "SYS-1029U", cpu: 32, ram: 192, watt: 360 },
  { vendor: "Lenovo", model: "ThinkSystem SR650", cpu: 48, ram: 384, watt: 500 },
];
const switchModels: SwitchModel[] = [
  { vendor: "Cisco", model: "Nexus 93180YC", watt: 320 },
  { vendor: "Juniper", model: "QFX5120-48Y", watt: 280 },
  { vendor: "MikroTik", model: "CCR2216-1G", watt: 120 },
  { vendor: "Arista", model: "7050SX3-48", watt: 240 },
];
const storageModels: StorageModel[] = [
  { vendor: "NetApp", model: "AFF A250", watt: 540, ram: 0, cpu: 0 },
  { vendor: "Dell", model: "PowerStore 500T", watt: 600, ram: 0, cpu: 0 },
];

// role drives what can run on the box
// k8s-node | vm-host | storage | network
const assets: Asset[] = [];
let aid = 0;
const mkDate = (y: number, m: number): string => new Date(y, m, rint(1, 27)).toISOString().slice(0, 10);

// distribute servers across DCs
const dcWeights = ["dc-thr1", "dc-thr1", "dc-thr1", "dc-thr2", "dc-thr2", "dc-fra1"];
const k8sCount = 10,
  vmCount = 9,
  bareCount = 4; // 23 servers

function makeServer(role: Asset["role"]): Asset {
  const m = pick(serverModels);
  const dcId = pick(dcWeights);
  const y = rint(2021, 2025);
  const amort = pick([36, 48, 60]);
  // price in Rial scaled by spec
  const base = m.cpu * 22_000_000 + m.ram * 4_200_000;
  const price = Math.round((base * rfloat(0.9, 1.25)) / 10_000_000) * 10_000_000;
  aid++;
  return {
    id: "srv-" + String(aid).padStart(2, "0"),
    name: (role === "k8s-node" ? "k8s-node-" : role === "vm-host" ? "vmhost-" : "bm-") + String(aid).padStart(2, "0"),
    type: "server",
    role,
    dcId,
    vendor: m.vendor,
    model: m.model,
    cpuCores: m.cpu,
    ramGB: m.ram,
    ssdGB: 0,
    hddGB: 0,
    powerWatts: m.watt,
    purchaseAmount: price,
    purchaseDate: mkDate(y, rint(0, 11)),
    amortMonths: amort,
    rackUnits: m.cpu >= 96 ? 2 : 1,
  };
}
for (let i = 0; i < k8sCount; i++) assets.push(makeServer("k8s-node"));
for (let i = 0; i < vmCount; i++) assets.push(makeServer("vm-host"));
for (let i = 0; i < bareCount; i++) assets.push(makeServer("bare-metal"));

// switches
for (let i = 0; i < 6; i++) {
  const m = pick(switchModels);
  const y = rint(2021, 2024);
  aid++;
  assets.push({
    id: "sw-" + String(i + 1).padStart(2, "0"),
    name: "switch-" + String(i + 1).padStart(2, "0"),
    type: "switch",
    role: "network",
    dcId: pick(dcWeights),
    vendor: m.vendor,
    model: m.model,
    cpuCores: 0,
    ramGB: 0,
    ssdGB: 0,
    hddGB: 0,
    powerWatts: m.watt,
    purchaseAmount: Math.round((rfloat(180, 880) / 10) * 10) * 1_000_000,
    purchaseDate: mkDate(y, rint(0, 11)),
    amortMonths: 60,
    rackUnits: 1,
  });
}
// storage
for (let i = 0; i < 3; i++) {
  const m = pick(storageModels);
  const y = rint(2022, 2025);
  assets.push({
    id: "stor-" + String(i + 1).padStart(2, "0"),
    name: "storage-" + String(i + 1).padStart(2, "0"),
    type: "storage",
    role: "storage",
    dcId: pick(dcWeights),
    vendor: m.vendor,
    model: m.model,
    cpuCores: 0,
    ramGB: 0,
    ssdGB: 0,
    hddGB: 0,
    powerWatts: m.watt,
    purchaseAmount: Math.round((rfloat(2200, 5400) / 10) * 10) * 1_000_000,
    purchaseDate: mkDate(y, rint(0, 11)),
    amortMonths: 60,
    rackUnits: 2,
  });
}

// ===========================================================
// SERVICES + workloads (Prometheus-style mock metrics)
// ===========================================================
const k8sNodes = assets.filter((a) => a.role === "k8s-node");
const vmHosts = assets.filter((a) => a.role === "vm-host");
// track remaining capacity per host for realistic packing
const cap: Record<string, { cpu: number; ram: number }> = {};
assets.forEach((a) => {
  cap[a.id] = { cpu: a.cpuCores * 0.85, ram: a.ramGB * 0.85 };
});

function placeOn(hosts: Asset[], cpuReq: number, ramReq: number): string {
  // find a host with room
  const shuffled = [...hosts].sort(() => rnd() - 0.5);
  for (const h of shuffled) {
    if (cap[h.id].cpu >= cpuReq && cap[h.id].ram >= ramReq) {
      cap[h.id].cpu -= cpuReq;
      cap[h.id].ram -= ramReq;
      return h.id;
    }
  }
  // fallback: least loaded
  const h = shuffled[0];
  cap[h.id].cpu -= cpuReq;
  cap[h.id].ram -= ramReq;
  return h.id;
}

interface SvcDef {
  name: string;
  team: string;
  platform: "kubernetes" | "vm";
  ns?: string;
  env: string;
  crit: Service["crit"];
  repl?: number;
}

const svcDefs: SvcDef[] = [];

const services: Service[] = svcDefs.map((s, i) => {
  const id = "svc-" + String(i + 1).padStart(2, "0");
  const workloads: Workload[] = [];
  if (s.platform === "kubernetes") {
    const pods = s.repl || rint(2, 5);
    // each replica: cpu request 0.5-6 cores, ram 0.5-12 GB
    for (let p = 0; p < pods; p++) {
      const cpuReq = +rfloat(0.5, 5.5).toFixed(2);
      const ramReq = +rfloat(0.5, 11).toFixed(1);
      const nodeId = placeOn(k8sNodes, cpuReq, ramReq);
      const util = rfloat(0.35, 0.92);
      workloads.push({
        hostId: nodeId,
        kind: "pod",
        ssdGB: 0,
        hddGB: 0,
        gpus: 0,
        cpuRequest: cpuReq,
        ramRequestGB: ramReq,
        cpuUsage: +(cpuReq * util).toFixed(2),
        ramUsageGB: +(ramReq * rfloat(0.45, 0.95)).toFixed(1),
      });
    }
  } else {
    const vms = rint(1, 4);
    for (let v = 0; v < vms; v++) {
      const vcpu = pick([4, 8, 8, 16, 16, 32]);
      const ramGB = vcpu * pick([2, 4, 4, 8]);
      const hostId = placeOn(vmHosts, vcpu, ramGB);
      const util = rfloat(0.3, 0.8);
      workloads.push({
        hostId,
        kind: "vm",
        vmName: s.name + "-vm" + (v + 1),
        ssdGB: 0,
        hddGB: 0,
        gpus: 0,
        cpuRequest: vcpu,
        ramRequestGB: ramGB,
        cpuUsage: +(vcpu * util).toFixed(2),
        ramUsageGB: +(ramGB * rfloat(0.4, 0.85)).toFixed(1),
      });
    }
  }
  return { id, name: s.name, team: s.team, platform: s.platform, namespace: s.ns || null, env: s.env, crit: s.crit, manualWeight: rint(1, 5), workloads };
});

// ===========================================================
// USERS
// ===========================================================
const users: User[] = [
  { id: "u1", name: "Arman Rashidi", email: "arman@rashnnu.io", role: "admin", password: "admin", active: true, lastSeen: "2026-05-29" },
  { id: "u2", name: "Sara Karimi", email: "sara@rashnnu.io", role: "admin", password: "admin", active: true, lastSeen: "2026-05-28" },
  { id: "u3", name: "Mehdi Tabesh", email: "mehdi@rashnnu.io", role: "viewer", password: "viewer", active: true, lastSeen: "2026-05-29" },
  { id: "u4", name: "Niloofar Ahmadi", email: "niloofar@rashnnu.io", role: "viewer", password: "viewer", active: true, lastSeen: "2026-05-22" },
  { id: "u5", name: "Pooya Nasiri", email: "pooya@rashnnu.io", role: "viewer", password: "viewer", active: false, lastSeen: "2026-04-30" },
];

// ===========================================================
// COST ENGINE
// ===========================================================
function monthsSince(dateStr: string): number {
  const d = new Date(dateStr);
  return (NOW.getFullYear() - d.getFullYear()) * 12 + (NOW.getMonth() - d.getMonth());
}

// monthly depreciation for an asset (0 once fully amortized)
function depreciation(asset: Asset): number {
  const elapsed = monthsSince(asset.purchaseDate);
  if (elapsed >= asset.amortMonths) return 0;
  return asset.purchaseAmount / asset.amortMonths;
}
function amortRemaining(asset: Asset): number {
  return Math.max(0, asset.amortMonths - monthsSince(asset.purchaseDate));
}

// colo fee allocated per asset, by power draw within its DC.
// Pass the live datacenter list so real DC IDs and fees are used.
function coloAllocation(assetList: Asset[], dcList: DataCenter[]): Record<string, number> {
  const byDc: Record<string, Asset[]> = {};
  assetList.forEach((a) => {
    (byDc[a.dcId] = byDc[a.dcId] || []).push(a);
  });
  const out: Record<string, number> = {};
  dcList.forEach((dc) => {
    const list = byDc[dc.id] || [];
    const totalW = list.reduce((s, a) => s + (a.powerWatts || 0), 0) || 1;
    list.forEach((a) => {
      out[a.id] = dc.coloMonthly * ((a.powerWatts || 0) / totalW);
    });
  });
  return out;
}

// full monthly cost per asset = depreciation + colo share
function assetCosts(assetList: Asset[], dcList: DataCenter[]): Record<string, AssetCost> {
  const colo = coloAllocation(assetList, dcList);
  const out: Record<string, AssetCost> = {};
  assetList.forEach((a) => {
    const dep = depreciation(a);
    const co = colo[a.id] || 0;
    out[a.id] = { depreciation: dep, colo: co, total: dep + co };
  });
  return out;
}

function isComputeHost(role: AssetRole): boolean {
  return role === "k8s-node" || role === "vm-host" || role === "bare-metal";
}

function resolveWorkloadDcId(hostId: string, assetList: Asset[], dcList: DataCenter[]): string | undefined {
  const asset = assetList.find((a) => a.id === hostId);
  if (asset) return asset.dcId;
  if (dcList.some((d) => d.id === hostId)) return hostId;
  return undefined;
}

// utilization / consumers per host (workloads pinned to a specific server asset)
function consumersByHost(serviceList: Service[]): Record<string, Consumer[]> {
  const map: Record<string, Consumer[]> = {};
  serviceList.forEach((s) => {
    s.workloads.forEach((w) => {
      (map[w.hostId] = map[w.hostId] || []).push({
        svcId: s.id,
        name: s.name,
        platform: s.platform,
        manualWeight: s.manualWeight,
        cpuUsage: w.cpuUsage,
        ramUsage: w.ramUsageGB,
        cpuReq: w.cpuRequest,
        ramReq: w.ramRequestGB,
      });
    });
  });
  return map;
}

// workloads grouped by datacenter (backend stores datacenter_id on pods/VMs)
function consumersByDatacenter(serviceList: Service[], assetList: Asset[], dcList: DataCenter[]): Record<string, Consumer[]> {
  const map: Record<string, Consumer[]> = {};
  serviceList.forEach((s) => {
    s.workloads.forEach((w) => {
      const dcId = resolveWorkloadDcId(w.hostId, assetList, dcList);
      if (!dcId) return;
      (map[dcId] = map[dcId] || []).push({
        svcId: s.id,
        name: s.name,
        platform: s.platform,
        manualWeight: s.manualWeight,
        cpuUsage: w.cpuUsage,
        ramUsage: w.ramUsageGB,
        cpuReq: w.cpuRequest,
        ramReq: w.ramRequestGB,
      });
    });
  });
  return map;
}

// host utilization summary
function hostUtil(assetList: Asset[], serviceList: Service[]): Record<string, HostUtil> {
  const cons = consumersByHost(serviceList);
  const out: Record<string, HostUtil> = {};
  assetList.forEach((a) => {
    const list = cons[a.id] || [];
    const cpuUsed = list.reduce((s, c) => s + c.cpuUsage, 0);
    const ramUsed = list.reduce((s, c) => s + c.ramUsage, 0);
    const cpuReq = list.reduce((s, c) => s + c.cpuReq, 0);
    const ramReq = list.reduce((s, c) => s + c.ramReq, 0);
    out[a.id] = {
      consumers: list.length,
      cpuUsed,
      ramUsed,
      cpuReq,
      ramReq,
      cpuUtil: a.cpuCores ? cpuUsed / a.cpuCores : 0,
      ramUtil: a.ramGB ? ramUsed / a.ramGB : 0,
    };
  });
  return out;
}

/* Allocate each datacenter's compute cost across services with workloads there. */
function allocate(assetList: Asset[], dcList: DataCenter[], serviceList: Service[], opts: AllocOpts = {}): AllocResult {
  const method = opts.method || "usage";
  const cpuW = opts.cpuWeight == null ? 0.5 : opts.cpuWeight;
  const ramW = 1 - cpuW;
  const costs = assetCosts(assetList, dcList);
  const consByDc = consumersByDatacenter(serviceList, assetList, dcList);

  const byService: Record<string, ServiceAlloc> = {};
  serviceList.forEach((s) => (byService[s.id] = { monthly: 0, breakdown: [] }));
  const idleByHost: Record<string, number> = {};

  const computeHostsByDc: Record<string, Asset[]> = {};
  assetList.forEach((a) => {
    if (!isComputeHost(a.role)) return;
    (computeHostsByDc[a.dcId] = computeHostsByDc[a.dcId] || []).push(a);
  });

  let totalComputeCost = 0;
  dcList.forEach((dc) => {
    const hosts = computeHostsByDc[dc.id] || [];
    const dcCost = hosts.reduce((s, a) => s + costs[a.id].total, 0);
    totalComputeCost += dcCost;
    const dcCpu = hosts.reduce((s, a) => s + a.cpuCores, 0);
    const dcRam = hosts.reduce((s, a) => s + a.ramGB, 0);
    const list = consByDc[dc.id] || [];
    if (list.length === 0) {
      hosts.forEach((a) => {
        idleByHost[a.id] = costs[a.id].total;
      });
      return;
    }

    const weights = list.map((c) => {
      if (method === "even") return 1;
      if (method === "manual") return c.manualWeight;
      if (method === "requests") return cpuW * (dcCpu ? c.cpuReq / dcCpu : 0) + ramW * (dcRam ? c.ramReq / dcRam : 0);
      return cpuW * (dcCpu ? c.cpuUsage / dcCpu : 0) + ramW * (dcRam ? c.ramUsage / dcRam : 0);
    });
    const totalW = weights.reduce((s, w) => s + w, 0) || 1;

    let allocFrac = 1;
    if (opts.includeIdle && (method === "usage" || method === "requests")) {
      const utilSum = list.reduce((s, c) => {
        const cu = method === "usage" ? c.cpuUsage : c.cpuReq;
        const ru = method === "usage" ? c.ramUsage : c.ramReq;
        return s + cpuW * (dcCpu ? cu / dcCpu : 0) + ramW * (dcRam ? ru / dcRam : 0);
      }, 0);
      allocFrac = Math.min(1, utilSum);
      const idleCost = dcCost * (1 - allocFrac);
      hosts.forEach((a) => {
        idleByHost[a.id] = idleCost * (costs[a.id].total / (dcCost || 1));
      });
    }

    list.forEach((c, i) => {
      const share = weights[i] / totalW;
      const cost = dcCost * allocFrac * share;
      byService[c.svcId].monthly += cost;
      byService[c.svcId].breakdown.push({ hostId: dc.id, cost, share });
    });
  });

  serviceList.forEach((s) => {
    byService[s.id].monthlyTotal = byService[s.id].monthly;
  });

  return { byService, idleByHost, sharedPool: 0, totalComputeCost };
}

// grand totals
function totals(assetList: Asset[], dcList: DataCenter[], _serviceList?: Service[]): Totals {
  const costs = assetCosts(assetList, dcList);
  let dep = 0,
    colo = 0,
    capex = 0;
  assetList.forEach((a) => {
    dep += costs[a.id].depreciation;
    colo += costs[a.id].colo;
    capex += a.purchaseAmount;
  });
  return {
    monthlyDepreciation: dep,
    monthlyColo: colo,
    monthlyTotal: dep + colo,
    hourlyTotal: (dep + colo) / HOURS_PER_MONTH,
    annualTotal: (dep + colo) * 12,
    totalCapex: capex,
    assetCount: assetList.length,
    serverCount: assetList.filter((a) => a.type === "server").length,
    activeAmort: assetList.filter((a) => depreciation(a) > 0).length,
  };
}

// synthesize a 12-month spend trend (slowly rising w/ noise) ending at current monthly
function rfloat2(r: () => number, lo: number, hi: number): number {
  return lo + r() * (hi - lo);
}

function trend(monthlyNow: number): TrendPoint[] {
  const out: TrendPoint[] = [];
  const now = new Date();
  let v = monthlyNow * 0.78;
  const r2 = mulberry32(7);
  for (let i = 11; i >= 0; i--) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
    const label = d.toLocaleString("en-US", { month: "short", year: "2-digit" });
    if (i === 0) {
      out.push({ month: label, value: monthlyNow });
    } else {
      v = v * (1 + rfloat2(r2, 0.005, 0.045));
      out.push({ month: label, value: Math.round(v) });
    }
  }
  return out;
}

// ===========================================================
// FORMATTERS  (Rial — large numbers, abbreviated)
// ===========================================================
function fmtMoney(n: number, opts: MoneyOpts = {}): string {
  const unit: Unit = opts.unit || "Rial";
  let v = Math.abs(n || 0);
  if (unit === "Toman") v = v / 10;
  const sign = n < 0 ? "-" : "";
  let str: string,
    suffix: string;
  if (v >= 1e12) {
    str = (v / 1e12).toFixed(2);
    suffix = "T";
  } else if (v >= 1e9) {
    str = (v / 1e9).toFixed(2);
    suffix = "B";
  } else if (v >= 1e6) {
    str = (v / 1e6).toFixed(1);
    suffix = "M";
  } else if (v >= 1e3) {
    str = (v / 1e3).toFixed(0);
    suffix = "K";
  } else {
    str = v.toFixed(0);
    suffix = "";
  }
  return sign + str + (suffix ? " " + suffix : "");
}
function fmtMoneyFull(n: number, opts: MoneyOpts = {}): string {
  let v = Math.round(n || 0);
  if ((opts.unit || "Rial") === "Toman") v = Math.round(v / 10);
  return v.toLocaleString("en-US");
}
function unitLabel(unit: Unit): string {
  return unit === "Toman" ? "Toman" : "Rial";
}
function fmtNum(n: number, d?: number): string {
  return (n || 0).toLocaleString("en-US", { maximumFractionDigits: d == null ? 0 : d });
}
function fmtPct(n: number): string {
  return (n * 100).toFixed(0) + "%";
}

function formatTrendMonth(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso.slice(0, 7);
  return d.toLocaleString("en-US", { month: "short", year: "2-digit" });
}

// deep clone for editable in-memory store
function clone<T>(x: T): T {
  return JSON.parse(JSON.stringify(x));
}

export const IC = {
  HOURS_PER_MONTH,
  NOW,
  seed: { datacenters, assets, services, users },
  monthsSince,
  depreciation,
  amortRemaining,
  coloAllocation,
  assetCosts,
  consumersByHost,
  hostUtil,
  allocate,
  totals,
  trend,
  formatTrendMonth,
  fmtMoney,
  fmtMoneyFull,
  fmtNum,
  fmtPct,
  unitLabel,
  clone,
};

export type ICType = typeof IC;
