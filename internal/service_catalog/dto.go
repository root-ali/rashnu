package service_catalog

import "github.com/google/uuid"

type CreateServiceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Platform    string `json:"platform"` // "kubernetes" | "vm"
	Team        string `json:"team"`
}

type UpdateServiceRequest struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Team        string    `json:"team"`
}

type AddPodWorkloadRequest struct {
	ServiceID     uuid.UUID `json:"service_id"`
	Name          string    `json:"name"`
	Namespace     string    `json:"namespace"`
	VCPU          int       `json:"vcpu"`
	MemoryGB      int       `json:"memory_gb"`
	SSDGB         int       `json:"ssd_gb"`
	HDDGB         int       `json:"hdd_gb"`
	GPUs          int       `json:"gpus"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsageGB float64   `json:"memory_usage_gb"`
	DatacenterID  uuid.UUID `json:"datacenter_id"`
}

type UpdatePodWorkloadRequest struct {
	ID            uuid.UUID `json:"id"`
	ServiceID     uuid.UUID `json:"service_id"`
	Name          string    `json:"name"`
	Namespace     string    `json:"namespace"`
	VCPU          int       `json:"vcpu"`
	MemoryGB      int       `json:"memory_gb"`
	SSDGB         int       `json:"ssd_gb"`
	HDDGB         int       `json:"hdd_gb"`
	GPUs          int       `json:"gpus"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsageGB float64   `json:"memory_usage_gb"`
	DatacenterID  uuid.UUID `json:"datacenter_id"`
}

type AddVMWorkloadRequest struct {
	ServiceID     uuid.UUID `json:"service_id"`
	Name          string    `json:"name"`
	VCPU          int       `json:"vcpu"`
	MemoryGB      int       `json:"memory_gb"`
	SSDGB         int       `json:"ssd_gb"`
	HDDGB         int       `json:"hdd_gb"`
	GPUs          int       `json:"gpus"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsageGB float64   `json:"memory_usage_gb"`
	DatacenterID  uuid.UUID `json:"datacenter_id"`
}

type UpdateVMWorkloadRequest struct {
	ID            uuid.UUID `json:"id"`
	ServiceID     uuid.UUID `json:"service_id"`
	Name          string    `json:"name"`
	VCPU          int       `json:"vcpu"`
	MemoryGB      int       `json:"memory_gb"`
	SSDGB         int       `json:"ssd_gb"`
	HDDGB         int       `json:"hdd_gb"`
	GPUs          int       `json:"gpus"`
	CPUUsage      float64   `json:"cpu_usage"`
	MemoryUsageGB float64   `json:"memory_usage_gb"`
	DatacenterID  uuid.UUID `json:"datacenter_id"`
}
