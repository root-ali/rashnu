package cost_report

import (
	"time"

	"github.com/google/uuid"
)

// DatacenterCost is the cost contribution of a single datacenter within a service's daily report.
type DatacenterCost struct {
	DatacenterID   uuid.UUID `json:"datacenter_id"`
	DatacenterName string    `json:"datacenter_name"`
	WorkloadCount  int       `json:"workload_count"`
	ComputeCost    int64     `json:"compute_cost"`
	CPUCost        int64     `json:"cpu_cost"`
	RAMCost        int64     `json:"ram_cost"`
	SSDCost        int64     `json:"ssd_cost"`
	HDDCost        int64     `json:"hdd_cost"`
}

// MonthlyDatacenterCost aggregates a datacenter's contribution for a calendar month.
type MonthlyDatacenterCost struct {
	DatacenterID   uuid.UUID `json:"datacenter_id"`
	DatacenterName string    `json:"datacenter_name"`
	TotalCost      int64     `json:"total_cost"`
	DailyAvg       int64     `json:"daily_avg"`
	CPUCost        int64     `json:"cpu_cost"`
	RAMCost        int64     `json:"ram_cost"`
	SSDCost        int64     `json:"ssd_cost"`
	HDDCost        int64     `json:"hdd_cost"`
}

// DailyReport is the computed daily cost for a single service.
type DailyReport struct {
	ID                  uuid.UUID        `json:"id"`
	ServiceID           uuid.UUID        `json:"service_id"`
	ServiceName         string           `json:"service_name"`
	Team                string           `json:"team"`
	Platform            string           `json:"platform"`
	WorkloadCount       int              `json:"workload_count"`
	Date                time.Time        `json:"date"`
	ComputeCost         int64            `json:"compute_cost"`
	CPUCost             int64            `json:"cpu_cost"`
	RAMCost             int64            `json:"ram_cost"`
	SSDCost             int64            `json:"ssd_cost"`
	HDDCost             int64            `json:"hdd_cost"`
	DatacenterBreakdown []DatacenterCost `json:"datacenter_breakdown"`
	CreatedAt           time.Time        `json:"created_at"`
}

// MonthlyReport aggregates daily records for a service over a calendar month.
type MonthlyReport struct {
	ServiceID           uuid.UUID               `json:"service_id"`
	ServiceName         string                  `json:"service_name"`
	Team                string                  `json:"team"`
	Platform            string                  `json:"platform"`
	Month               time.Time               `json:"month"`
	TotalCost           int64                   `json:"total_cost"`
	DailyAvg            int64                   `json:"daily_avg"`
	DaysRecorded        int                     `json:"days_recorded"`
	CPUCost             int64                   `json:"cpu_cost"`
	RAMCost             int64                   `json:"ram_cost"`
	SSDCost             int64                   `json:"ssd_cost"`
	HDDCost             int64                   `json:"hdd_cost"`
	DatacenterBreakdown []MonthlyDatacenterCost `json:"datacenter_breakdown"`
}

// ServiceInput is the cost engine's view of a service and its workloads.
type ServiceInput struct {
	ID        uuid.UUID
	Name      string
	Team      string
	Platform  string
	Workloads []WorkloadInput
}

// WorkloadInput holds the resource allocation of a single workload (pod or VM).
type WorkloadInput struct {
	VCPU           int
	MemoryGB       int
	SSDGB          int
	HDDGB          int
	GPUs           int
	DatacenterID   uuid.UUID
	DatacenterName string
}

// DailyUnitPrice is the per-resource price for one datacenter on one day.
type DailyUnitPrice struct {
	DatacenterID uuid.UUID
	VCPUPrice    int64
	MemoryPrice  int64
	SSDPrice     int64
	HDDPrice     int64
	GPUPrice     int64
}
