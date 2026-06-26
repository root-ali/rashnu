/* ============================================================
   rashnnu · Infrastructure Cost Manager — shared domain types
   ============================================================ */

export type AssetType = "server" | "switch" | "storage";
export type AssetRole = "k8s-node" | "vm-host" | "bare-metal" | "network" | "storage";

export interface Asset {
  id: string;
  name: string;
  type: AssetType;
  role: AssetRole;
  dcId: string;
  vendor: string;
  model: string;
  cpuCores: number;
  ramGB: number;
  ssdGB: number;
  hddGB: number;
  powerWatts: number;
  purchaseAmount: number;
  purchaseDate: string;
  amortMonths: number;
  rackUnits: number;
}

export interface DataCenter {
  id: string;
  name: string;
  short: string;
  provider: string;
  region: string;
  coloMonthly: number;
  racks: number;
  power_kw: number;
  tier: string;
}

export type WorkloadKind = "pod" | "vm";

export interface Workload {
  id?: string; // backend pod/vm ID — undefined for locally-added workloads
  hostId: string;
  kind: WorkloadKind;
  vmName?: string;
  ssdGB: number;
  hddGB: number;
  gpus: number;
  cpuRequest: number;
  ramRequestGB: number;
  cpuUsage: number;
  ramUsageGB: number;
}

export interface ServiceResources {
  vcpu: number;
  memory_gb: number;
  ssd_gb: number;
  hdd_gb: number;
  gpus: number;
}

export interface LivePodMetric {
  pod: string;
  cpu_cores: number;
  memory_gb: number;
}

export type Platform = "kubernetes" | "vm";
export type Crit = "critical" | "high" | "med" | "low";

export interface Service {
  id: string;
  name: string;
  team: string;
  platform: Platform;
  namespace: string | null;
  env: string;
  crit: Crit;
  manualWeight: number;
  workloads: Workload[];
  resources?: ServiceResources;
  live_pods?: LivePodMetric[];
}

export type Role = "admin" | "viewer";

export interface User {
  id: string;
  name: string;
  email: string;
  role: Role;
  password: string;
  active: boolean;
  lastSeen: string;
}

/* ---- cost engine result shapes ---- */

export interface AssetCost {
  depreciation: number;
  colo: number;
  total: number;
}

export interface Consumer {
  svcId: string;
  name: string;
  platform: Platform;
  manualWeight: number;
  cpuUsage: number;
  ramUsage: number;
  cpuReq: number;
  ramReq: number;
}

export interface HostUtil {
  consumers: number;
  cpuUsed: number;
  ramUsed: number;
  cpuReq: number;
  ramReq: number;
  cpuUtil: number;
  ramUtil: number;
}

export interface ServiceAllocBreakdown {
  hostId: string;
  cost: number;
  share: number;
}

export interface ServiceAlloc {
  monthly: number;
  breakdown: ServiceAllocBreakdown[];
  shared?: number;
  monthlyTotal?: number;
}

export interface AllocResult {
  byService: Record<string, ServiceAlloc>;
  idleByHost: Record<string, number>;
  sharedPool: number;
  totalComputeCost: number;
}

export type AllocMethod = "usage" | "requests" | "even" | "manual";

export interface AllocOpts {
  method?: AllocMethod;
  cpuWeight?: number;
  includeIdle?: boolean;
}

export interface Totals {
  monthlyDepreciation: number;
  monthlyColo: number;
  monthlyTotal: number;
  hourlyTotal: number;
  annualTotal: number;
  totalCapex: number;
  assetCount: number;
  serverCount: number;
  activeAmort: number;
}

export interface TrendPoint {
  month: string;
  value: number;
}

export type Unit = "Rial" | "Toman";

export interface MoneyOpts {
  unit?: Unit;
}

/* ---- app-level state shapes ---- */

export interface Tweaks {
  navStyle: "sidebar" | "topbar";
  vizStyle: "area" | "bars" | "donut";
  reportStyle: "charts" | "tables" | "mixed";
  allocMethod: AllocMethod;
  cpuWeight: number;
  includeIdle: boolean;
  unit: Unit;
  accent: string;
  fontUi: string;
}

export interface Derived {
  assetCosts: Record<string, AssetCost>;
  alloc: AllocResult;
  totals: Totals;
  hostUtil: Record<string, HostUtil>;
  trend: TrendPoint[];
}

export type ToastKind = "success" | "error";

export interface MonthlyDatacenterCostBreakdown {
  datacenter_id: string;
  datacenter_name: string;
  total_cost: number;
  daily_avg: number;
  cpu_cost: number;
  ram_cost: number;
  ssd_cost: number;
  hdd_cost: number;
}

export interface ServiceMonthlyReport {
  service_id: string;
  service_name: string;
  team: string;
  platform: string;
  month: string;
  total_cost: number;
  daily_avg: number;
  days_recorded: number;
  cpu_cost: number;
  ram_cost: number;
  ssd_cost: number;
  hdd_cost: number;
  datacenter_breakdown: MonthlyDatacenterCostBreakdown[];
}

export interface CostTrendPoint {
  month: string;
  total_cost: number;
}

export interface CostReports {
  monthly: ServiceMonthlyReport[];
  trend: CostTrendPoint[];
  monthlyMonth?: string;
}

/** Shared context object threaded into every page. */
export interface Ctx {
  dcs: DataCenter[];
  assets: Asset[];
  services: Service[];
  users: User[];
  derived: Derived;
  me: User | null;
  isAdmin: boolean;
  t: Tweaks;
  fmt: (v: number) => string;
  fmtFull: (v: number) => string;
  unit: string;
  toast: (msg: string, kind?: ToastKind) => void;
  nav: (page: string) => void;
  upsertAsset: (a: Asset) => void;
  deleteAsset: (id: string) => void;
  upsertDc: (d: DataCenter) => void;
  deleteDc: (id: string) => void;
  upsertService: (s: Service) => void;
  deleteService: (id: string) => void;
  upsertUser: (u: User) => void;
  deleteUser: (id: string) => void;
  costReports: CostReports;
  calculateDailyCosts: (date?: string) => Promise<void>;
  fulfillCostReports: (startTime: string) => Promise<void>;
}
