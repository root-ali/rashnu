// Backend REST API client — all communication with Go backend goes through here.
// JWT Bearer token is stored in localStorage under TOKEN_KEY.

const TOKEN_KEY = "ic_token";

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY);
}
export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token);
}
export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY);
}

// Decode JWT payload without verifying signature (server verifies; client just reads claims).
export function parseJwt(token: string): Record<string, unknown> {
  try {
    return JSON.parse(atob(token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/")));
  } catch {
    return {};
  }
}

// ── Backend response types ────────────────────────────────────────────────────

export interface BackendDataCenter {
  id: string;
  name: string;
  short: string;
  provider: string;
  region: string;
  racks: number;
  power_kw: number;
  tier: string;
  monthly_colo_fee: number;
  created_at: string;
  updated_at: string;
}

export interface BackendServer {
  id: string;
  datacenter_id: string;
  hostname: string;
  region: string;
  vendor: string;
  model: string;
  purchase_price: number;
  purchase_date: string;
  amort_months: number;
  vcpus: number;
  memory_gb: number;
  ssd_capacity_gb: number;
  hdd_capacity_gb: number;
  gpus: number;
  power_watts: number;
  rack_units: number;
  role: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface InfraHardwareSpecs {
  vendor?: string;
  power_watts?: number;
  rack_units?: number;
}

export interface BackendInfraHardware {
  id: string;
  datacenter_id: string;
  hostname: string;
  hardware_type: string;
  region: string;
  purchase_price: number;
  purchase_date: string;
  amort_months: number;
  model: string;
  specs: InfraHardwareSpecs | null;
  created_at: string;
  updated_at: string;
}

// ── Base fetch ────────────────────────────────────────────────────────────────

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(init?.headers as Record<string, string> | undefined),
  };
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(`/api/v1${path}`, { ...init, headers });
  if (!res.ok) {
    if (res.status === 401) {
      clearToken();
      window.dispatchEvent(new CustomEvent("auth:token-expired"));
    }
    const text = await res.text().catch(() => "");
    let msg = `HTTP ${res.status}`;
    try {
      msg = (JSON.parse(text) as { error?: string }).error || msg;
    } catch {
      if (text) msg = text;
    }
    throw new Error(msg);
  }

  const text = await res.text();
  if (!text || text === "{}") return undefined as T;
  return JSON.parse(text) as T;
}

// ── Identity ──────────────────────────────────────────────────────────────────

export async function loginApi(email: string, password: string): Promise<{ token: string; role: string; email: string; full_name: string }> {
  return apiFetch("/user/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
}

export interface BackendUser {
  id: string;
  fullname: string;
  email: string;
  role: string;
  status: string;
  create_at: string;
  update_at: string;
}

export async function listUsersApi(): Promise<BackendUser[]> {
  return apiFetch("/user");
}

export async function createUserApi(req: {
  email: string;
  full_name: string;
  password: string;
  role: string;
  status: string;
}): Promise<void> {
  await apiFetch("/user/new", { method: "POST", body: JSON.stringify(req) });
}

export async function updateUserApi(req: {
  id: string;
  email: string;
  full_name: string;
  role: string;
  status: string;
}): Promise<void> {
  await apiFetch(`/user/${req.id}`, { method: "PUT", body: JSON.stringify(req) });
}

export async function deleteUserApi(id: string): Promise<void> {
  await apiFetch(`/user/${id}`, { method: "DELETE" });
}

export async function updatePasswordApi(userId: string, req: {
  old_passwrd: string;
  new_passwrd: string;
  confirm_password: string;
}): Promise<void> {
  await apiFetch(`/user/${userId}/password`, { method: "PUT", body: JSON.stringify(req) });
}

export async function logoutApi(): Promise<void> {
  await apiFetch("/user/logout", { method: "POST" });
}

// ── Service Catalog ───────────────────────────────────────────────────────────

export interface BackendResourceTotals {
  vcpu: number;
  memory_gb: number;
  ssd_gb: number;
  hdd_gb: number;
  gpus: number;
}

export interface BackendPodSpec {
  id: string;
  service_id: string;
  name: string;
  namespace: string;
  vcpu: number;
  memory_gb: number;
  ssd_gb: number;
  hdd_gb: number;
  gpus: number;
  cpu_usage: number;
  memory_usage_gb: number;
  datacenter_id: string;
  datacenter_name: string;
}

export interface BackendVMSpec {
  id: string;
  service_id: string;
  name: string;
  vcpu: number;
  memory_gb: number;
  ssd_gb: number;
  hdd_gb: number;
  gpus: number;
  cpu_usage: number;
  memory_usage_gb: number;
  datacenter_id: string;
  datacenter_name: string;
}

export interface BackendService {
  id: string;
  name: string;
  description: string;
  platform: string;
  team: string;
  pods: BackendPodSpec[] | null;
  vms: BackendVMSpec[] | null;
  resources: BackendResourceTotals;
  live_pods?: Array<{ pod: string; cpu_cores: number; memory_gb: number }>;
  created_at: string;
  updated_at: string;
}

export async function listServicesApi(): Promise<{ services: BackendService[] }> {
  return apiFetch("/services");
}

export async function getServiceApi(id: string): Promise<BackendService> {
  return apiFetch(`/services/${id}`);
}

export async function createServiceApi(req: {
  name: string;
  description: string;
  platform: string;
  team: string;
}): Promise<{ id: string }> {
  return apiFetch("/services", { method: "POST", body: JSON.stringify(req) });
}

export async function updateServiceApi(req: {
  id: string;
  name: string;
  description: string;
  team: string;
}): Promise<void> {
  await apiFetch(`/services/${req.id}`, { method: "PUT", body: JSON.stringify(req) });
}

export async function deleteServiceApi(id: string): Promise<void> {
  await apiFetch(`/services/${id}`, { method: "DELETE" });
}

export async function addPodWorkloadApi(serviceId: string, req: {
  service_id: string;
  name: string;
  namespace: string;
  vcpu: number;
  memory_gb: number;
  ssd_gb: number;
  hdd_gb: number;
  gpus: number;
  cpu_usage: number;
  memory_usage_gb: number;
  datacenter_id: string;
}): Promise<void> {
  await apiFetch(`/services/${serviceId}/pods`, { method: "POST", body: JSON.stringify(req) });
}

export async function deletePodWorkloadApi(serviceId: string, podId: string): Promise<void> {
  await apiFetch(`/services/${serviceId}/pods/${podId}`, { method: "DELETE" });
}

export async function addVMWorkloadApi(serviceId: string, req: {
  service_id: string;
  name: string;
  vcpu: number;
  memory_gb: number;
  ssd_gb: number;
  hdd_gb: number;
  gpus: number;
  cpu_usage: number;
  memory_usage_gb: number;
  datacenter_id: string;
}): Promise<void> {
  await apiFetch(`/services/${serviceId}/vms`, { method: "POST", body: JSON.stringify(req) });
}

export async function deleteVMWorkloadApi(serviceId: string, vmId: string): Promise<void> {
  await apiFetch(`/services/${serviceId}/vms/${vmId}`, { method: "DELETE" });
}

export async function saveWorkloadUsageApi(
  serviceId: string,
  workloadId: string,
  kind: "pod" | "vm",
  cpuUsage: number,
  memoryUsageGB: number,
): Promise<void> {
  const path = kind === "pod"
    ? `/services/${serviceId}/pods/${workloadId}`
    : `/services/${serviceId}/vms/${workloadId}`;
  // Re-uses the full update endpoint; only usage fields are changed — others keep their stored values
  // via a GET-then-PUT pattern handled in the component.
  await apiFetch(path, { method: "PUT", body: JSON.stringify({ cpu_usage: cpuUsage, memory_usage_gb: memoryUsageGB }) });
}

// ── Prometheus config ─────────────────────────────────────────────────────────

export interface PrometheusConfig {
  url: string;
  enabled: boolean;
  updated_at?: string;
}

export async function getPrometheusConfigApi(): Promise<PrometheusConfig> {
  return apiFetch("/prometheus/config");
}

export async function updatePrometheusConfigApi(req: { url: string; enabled: boolean }): Promise<void> {
  await apiFetch("/prometheus/config", { method: "PUT", body: JSON.stringify(req) });
}

// ── DataCenters ───────────────────────────────────────────────────────────────

export async function listDataCentersApi(): Promise<{ datacenters: BackendDataCenter[] }> {
  return apiFetch("/datacenters");
}

export async function createDataCenterApi(req: {
  name: string;
  short: string;
  provider: string;
  region: string;
  racks: number;
  power_kw: number;
  tier: string;
  monthly_colo_fee: number;
}): Promise<void> {
  await apiFetch("/datacenters", { method: "POST", body: JSON.stringify(req) });
}

export async function updateDataCenterApi(req: {
  id: string;
  name: string;
  short: string;
  provider: string;
  region: string;
  racks: number;
  power_kw: number;
  tier: string;
  monthly_colo_fee: number;
}): Promise<void> {
  await apiFetch("/datacenters", { method: "PUT", body: JSON.stringify(req) });
}

export async function deleteDataCenterApi(id: string): Promise<void> {
  await apiFetch(`/datacenters/${id}`, { method: "DELETE" });
}

// ── Servers ───────────────────────────────────────────────────────────────────

export async function listServersApi(): Promise<{ servers: BackendServer[] }> {
  return apiFetch("/servers");
}

export async function createServerApi(req: {
  datacenter_id: string;
  hostname: string;
  region: string;
  vendor: string;
  model: string;
  purchase_price: number;
  purchase_date: string;
  amort_months: number;
  vcpus: number;
  memory_gb: number;
  ssd_capacity_gb: number;
  hdd_capacity_gb: number;
  gpus: number;
  power_watts: number;
  rack_units: number;
  role: string;
}): Promise<void> {
  await apiFetch("/servers", { method: "POST", body: JSON.stringify(req) });
}

export async function updateServerApi(req: {
  id: string;
  datacenter_id: string;
  hostname: string;
  region: string;
  vendor: string;
  model: string;
  purchase_price: number;
  purchase_date: string;
  amort_months: number;
  vcpus: number;
  memory_gb: number;
  ssd_capacity_gb: number;
  hdd_capacity_gb: number;
  gpus: number;
  power_watts: number;
  rack_units: number;
  role: string;
  status: string;
}): Promise<void> {
  await apiFetch("/servers", { method: "PUT", body: JSON.stringify(req) });
}

export async function deleteServerApi(id: string): Promise<void> {
  await apiFetch(`/servers/${id}`, { method: "DELETE" });
}

// ── Infrastructure Hardwares ──────────────────────────────────────────────────

export async function listInfraHardwaresApi(): Promise<{ hardwares: BackendInfraHardware[] }> {
  return apiFetch("/infrastructure-hardwares");
}

export async function createInfraHardwareApi(req: {
  datacenter_id: string;
  hostname: string;
  hardware_type: string;
  region: string;
  purchase_price: number;
  purchase_date: string;
  amort_months: number;
  model: string;
  specs: unknown;
}): Promise<void> {
  await apiFetch("/infrastructure-hardwares", { method: "POST", body: JSON.stringify(req) });
}

export async function updateInfraHardwareApi(req: {
  id: string;
  datacenter_id: string;
  hostname: string;
  hardware_type: string;
  region: string;
  purchase_price: number;
  purchase_date: string;
  amort_months: number;
  model: string;
  specs: unknown;
}): Promise<void> {
  await apiFetch("/infrastructure-hardwares", { method: "PUT", body: JSON.stringify(req) });
}

export async function deleteInfraHardwareApi(id: string): Promise<void> {
  await apiFetch(`/infrastructure-hardwares/${id}`, { method: "DELETE" });
}

// ── Cost Reports ──────────────────────────────────────────────────────────────

export interface BackendMonthlyDatacenterCost {
  datacenter_id: string;
  datacenter_name: string;
  total_cost: number;
  daily_avg: number;
  cpu_cost: number;
  ram_cost: number;
  ssd_cost: number;
  hdd_cost: number;
}

export interface BackendMonthlyReport {
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
  datacenter_breakdown: BackendMonthlyDatacenterCost[];
}

export interface BackendCostTrendPoint {
  month: string;
  total_cost: number;
}

export interface BackendSpendItem {
  key: string;
  label: string;
  total_cost: number;
  service_count?: number;
  platform?: string;
  team?: string;
}

export interface BackendSpendSummary {
  month: string;
  group_by: string;
  total: number;
  items: BackendSpendItem[];
}

export async function getMonthlyCostReportsApi(month?: string): Promise<{ reports: BackendMonthlyReport[]; month?: string; total?: number }> {
  const q = month ? `?month=${month}` : "";
  return apiFetch(`/cost-reports/monthly${q}`);
}

export async function getCostTrendApi(months = 12): Promise<{ points: BackendCostTrendPoint[] }> {
  return apiFetch(`/cost-reports/trend?months=${months}`);
}

export async function getCostSpendApi(opts?: { month?: string; groupBy?: string; platform?: string }): Promise<BackendSpendSummary> {
  const params = new URLSearchParams();
  if (opts?.month) params.set("month", opts.month);
  if (opts?.groupBy) params.set("group_by", opts.groupBy);
  if (opts?.platform && opts.platform !== "all") params.set("platform", opts.platform);
  const q = params.toString();
  return apiFetch(`/cost-reports/spend${q ? `?${q}` : ""}`);
}

export async function calculateDailyCostsApi(date?: string): Promise<void> {
  const q = date ? `?date=${date}` : "";
  await apiFetch(`/cost-reports/calculate${q}`, { method: "POST" });
}

export async function fulfillCostReportsApi(startTime: string): Promise<void> {
  await apiFetch(`/cost-reports/fulfill`, { method: "POST", body: JSON.stringify({ start_time: startTime }) });
}

