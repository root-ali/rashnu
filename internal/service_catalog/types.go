package service_catalog

import (
	"time"

	"github.com/google/uuid"
)

const (
	PlatformKubernetes = "kubernetes"
	PlatformVM         = "vm"
)

// ResourceTotals is the sum of all workload resources for a service.
type ResourceTotals struct {
	VCPU     int `json:"vcpu"`
	MemoryGB int `json:"memory_gb"`
	SSDGB    int `json:"ssd_gb"`
	HDDGB    int `json:"hdd_gb"`
	GPUs     int `json:"gpus"`
}

type Service struct {
	ID          uuid.UUID      `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Platform    string         `json:"platform"` // "kubernetes" | "vm"
	Team        string         `json:"team"`
	Pods        []PodSpec      `json:"pods,omitempty"`
	VMs         []VMSpec       `json:"vms,omitempty"`
	Resources   ResourceTotals `json:"resources"`
	// LivePods is populated only when Prometheus is enabled and the service is Kubernetes-based.
	LivePods  []LivePodMetric `json:"live_pods,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	DeletedAt *time.Time      `json:"deleted_at,omitempty"`
}

func computeResources(pods []PodSpec, vms []VMSpec) ResourceTotals {
	var r ResourceTotals
	for _, p := range pods {
		r.VCPU += p.VCPU
		r.MemoryGB += p.MemoryGB
		r.SSDGB += p.SSDGB
		r.HDDGB += p.HDDGB
		r.GPUs += p.GPUs
	}
	for _, v := range vms {
		r.VCPU += v.VCPU
		r.MemoryGB += v.MemoryGB
		r.SSDGB += v.SSDGB
		r.HDDGB += v.HDDGB
		r.GPUs += v.GPUs
	}
	return r
}

type PodSpec struct {
	ID             uuid.UUID `json:"id"`
	ServiceID      uuid.UUID `json:"service_id"`
	Name           string    `json:"name"`
	Namespace      string    `json:"namespace"`
	VCPU           int       `json:"vcpu"`
	MemoryGB       int       `json:"memory_gb"`
	SSDGB          int       `json:"ssd_gb"`
	HDDGB          int       `json:"hdd_gb"`
	GPUs           int       `json:"gpus"`
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsageGB  float64   `json:"memory_usage_gb"`
	DatacenterID   uuid.UUID `json:"datacenter_id"`
	DatacenterName string    `json:"datacenter_name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type VMSpec struct {
	ID             uuid.UUID `json:"id"`
	ServiceID      uuid.UUID `json:"service_id"`
	Name           string    `json:"name"`
	VCPU           int       `json:"vcpu"`
	MemoryGB       int       `json:"memory_gb"`
	SSDGB          int       `json:"ssd_gb"`
	HDDGB          int       `json:"hdd_gb"`
	GPUs           int       `json:"gpus"`
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsageGB  float64   `json:"memory_usage_gb"`
	DatacenterID   uuid.UUID `json:"datacenter_id"`
	DatacenterName string    `json:"datacenter_name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
