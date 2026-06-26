// Mappers between backend REST shapes and the frontend domain types.

import type { Asset, AssetRole, AssetType, DataCenter, LivePodMetric, Platform, Role, Service, User, Workload, WorkloadKind } from "./types";
import type { BackendDataCenter, BackendInfraHardware, BackendServer, BackendService, BackendUser } from "./api";

// ── DataCenter ────────────────────────────────────────────────────────────────

export function mapBackendDataCenter(dc: BackendDataCenter): DataCenter {
  return {
    id: dc.id,
    name: dc.name,
    short: dc.short || dc.name.slice(0, 3).toUpperCase(),
    provider: dc.provider || "",
    region: dc.region,
    coloMonthly: dc.monthly_colo_fee,
    racks: dc.racks || 0,
    power_kw: dc.power_kw || 0,
    tier: dc.tier || "",
  };
}

export function mapDataCenterToBackend(dc: DataCenter) {
  return {
    name: dc.name,
    short: dc.short,
    provider: dc.provider,
    region: dc.region,
    racks: dc.racks,
    power_kw: dc.power_kw,
    tier: dc.tier,
    monthly_colo_fee: dc.coloMonthly,
  };
}

// ── Server ────────────────────────────────────────────────────────────────────

function toAssetRole(role: string): AssetRole {
  const map: Record<string, AssetRole> = {
    "k8s-node": "k8s-node",
    "vm-host": "vm-host",
    "bare-metal": "bare-metal",
    bare_metal: "bare-metal",
    storage: "storage",
    network: "network",
  };
  return (map[role] as AssetRole) ?? "bare-metal";
}

export function mapBackendServer(s: BackendServer): Asset {
  return {
    id: s.id,
    name: s.hostname,
    type: "server",
    role: toAssetRole(s.role),
    dcId: s.datacenter_id,
    vendor: s.vendor || "",
    model: s.model || "",
    cpuCores: s.vcpus,
    ramGB: s.memory_gb,
    ssdGB: s.ssd_capacity_gb || 0,
    hddGB: s.hdd_capacity_gb || 0,
    powerWatts: s.power_watts || 0,
    purchaseAmount: s.purchase_price,
    purchaseDate: (s.purchase_date ?? "").slice(0, 10),
    amortMonths: s.amort_months,
    rackUnits: s.rack_units || 1,
  };
}

export function mapAssetToServerCreate(a: Asset, region: string) {
  return {
    datacenter_id: a.dcId,
    hostname: a.name,
    region,
    vendor: a.vendor,
    model: a.model,
    purchase_price: a.purchaseAmount,
    purchase_date: a.purchaseDate,
    amort_months: a.amortMonths,
    vcpus: a.cpuCores,
    memory_gb: a.ramGB,
    ssd_capacity_gb: a.ssdGB,
    hdd_capacity_gb: a.hddGB,
    gpus: 0,
    power_watts: a.powerWatts,
    rack_units: a.rackUnits,
    role: a.role,
  };
}

export function mapAssetToServerUpdate(a: Asset, region: string) {
  return { ...mapAssetToServerCreate(a, region), id: a.id, status: "active" };
}

// ── Infrastructure Hardware ───────────────────────────────────────────────────

function toInfraAssetType(hwType: string): AssetType {
  if (hwType === "storage") return "storage";
  return "switch";
}

function toInfraAssetRole(hwType: string): AssetRole {
  if (hwType === "storage") return "storage";
  return "network";
}

export function mapBackendInfraHardware(h: BackendInfraHardware): Asset {
  const specs = h.specs || {};
  return {
    id: h.id,
    name: h.hostname,
    type: toInfraAssetType(h.hardware_type),
    role: toInfraAssetRole(h.hardware_type),
    dcId: h.datacenter_id,
    vendor: specs.vendor ?? "",
    model: h.model,
    cpuCores: 0,
    ramGB: 0,
    ssdGB: 0,
    hddGB: 0,
    powerWatts: specs.power_watts ?? 0,
    purchaseAmount: h.purchase_price,
    purchaseDate: (h.purchase_date ?? "").slice(0, 10),
    amortMonths: h.amort_months,
    rackUnits: specs.rack_units ?? 1,
  };
}

export function mapAssetToInfraCreate(a: Asset, region: string) {
  return {
    datacenter_id: a.dcId,
    hostname: a.name,
    hardware_type: a.type === "storage" ? "storage" : "switch",
    region,
    purchase_price: a.purchaseAmount,
    purchase_date: a.purchaseDate,
    amort_months: a.amortMonths,
    model: a.model,
    specs: { vendor: a.vendor, power_watts: a.powerWatts, rack_units: a.rackUnits },
  };
}

export function mapAssetToInfraUpdate(a: Asset, region: string) {
  return { ...mapAssetToInfraCreate(a, region), id: a.id };
}

// ── Service Catalog ───────────────────────────────────────────────────────────

export function mapBackendService(s: BackendService): Service {
  const pods = s.pods ?? [];
  const vms = s.vms ?? [];

  const workloads: Workload[] = [
    ...pods.map((p) => ({
      id: p.id,
      hostId: p.datacenter_id,
      kind: "pod" as WorkloadKind,
      ssdGB: p.ssd_gb ?? 0,
      hddGB: p.hdd_gb ?? 0,
      gpus: p.gpus ?? 0,
      cpuRequest: p.vcpu,
      ramRequestGB: p.memory_gb,
      cpuUsage: p.cpu_usage ?? 0,
      ramUsageGB: p.memory_usage_gb ?? 0,
    })),
    ...vms.map((v) => ({
      id: v.id,
      hostId: v.datacenter_id,
      kind: "vm" as WorkloadKind,
      vmName: v.name,
      ssdGB: v.ssd_gb ?? 0,
      hddGB: v.hdd_gb ?? 0,
      gpus: v.gpus ?? 0,
      cpuRequest: v.vcpu,
      ramRequestGB: v.memory_gb,
      cpuUsage: v.cpu_usage ?? 0,
      ramUsageGB: v.memory_usage_gb ?? 0,
    })),
  ];

  const livePods: LivePodMetric[] | undefined = s.live_pods?.map((p) => ({
    pod: p.pod,
    cpu_cores: p.cpu_cores,
    memory_gb: p.memory_gb,
  }));

  return {
    id: s.id,
    name: s.name,
    team: s.team,
    platform: s.platform as Platform,
    namespace: pods[0]?.namespace ?? null,
    env: "prod",
    crit: "med",
    manualWeight: 1,
    workloads,
    resources: s.resources,
    live_pods: livePods,
  };
}

// ── User ──────────────────────────────────────────────────────────────────────

export function mapBackendUser(u: BackendUser): User {
  return {
    id: u.id,
    name: u.fullname,
    email: u.email,
    role: u.role as Role,
    password: "",
    active: u.status === "active",
    lastSeen: (u.create_at ?? "").slice(0, 10),
  };
}