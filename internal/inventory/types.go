package inventory

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type DataCenter struct {
	ID             uuid.UUID `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	Short          string    `json:"short" db:"short"`
	Provider       string    `json:"provider" db:"provider"`
	Region         string    `json:"region" db:"region"`
	Racks          int       `json:"racks" db:"racks"`
	PowerKW        float64   `json:"power_kw" db:"power_kw"`
	Tier           string    `json:"tier" db:"tier"`
	MonthlyColoFee int64     `json:"monthly_colo_fee" db:"monthly_colo_fee"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type Server struct {
	ID            uuid.UUID    `json:"id" db:"id"`
	DataCenterID  uuid.UUID    `json:"datacenter_id" db:"datacenter_id"`
	Hostname      string       `json:"hostname" db:"hostname"`
	Region        string       `json:"region" db:"region"`
	Vendor        string       `json:"vendor" db:"vendor"`
	Model         string       `json:"model" db:"model"`
	PurchasePrice int64        `json:"purchase_price" db:"purchase_price"`
	PurchaseDate  time.Time    `json:"purchase_date" db:"purchase_date"`
	AmortMonths   int          `json:"amort_months" db:"amort_months"`
	VCPUs         int          `json:"vcpus" db:"vcpus"`
	MemoryGB      int          `json:"memory_gb" db:"memory_gb"`
	SSDCapacityGB int          `json:"ssd_capacity_gb" db:"ssd_capacity_gb"`
	HDDCapacityGB int          `json:"hdd_capacity_gb" db:"hdd_capacity_gb"`
	GPUs          int          `json:"gpus" db:"gpus"`
	PowerWatts    int          `json:"power_watts" db:"power_watts"`
	RackUnits     int          `json:"rack_units" db:"rack_units"`
	Role          string       `json:"role" db:"role"`
	Status        ServerStatus `json:"status" db:"status"`
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at" db:"updated_at"`
}

// InfraSpecs is the canonical schema stored in infrastructure_hardwares.specs (JSONB).
type InfraSpecs struct {
	Vendor     string `json:"vendor"`
	PowerWatts int    `json:"power_watts"`
	RackUnits  int    `json:"rack_units"`
}

// InfrastructureHardware stores switches, routers, storage arrays and similar kit.
// The Specs field carries InfraSpecs (vendor, power_watts, rack_units).
type InfrastructureHardware struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	DataCenterID  uuid.UUID       `json:"datacenter_id" db:"datacenter_id"`
	Hostname      string          `json:"hostname" db:"hostname"`
	HardwareType  string          `json:"hardware_type" db:"hardware_type"`
	Region        string          `json:"region" db:"region"`
	PurchasePrice int64           `json:"purchase_price" db:"purchase_price"`
	PurchaseDate  time.Time       `json:"purchase_date" db:"purchase_date"`
	AmortMonths   int             `json:"amort_months" db:"amort_months"`
	Model         string          `json:"model" db:"model"`
	Specs         json.RawMessage `json:"specs" db:"specs"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

type ServerStatus string

const (
	StatusActive       ServerStatus = "active"
	StatusMaintenance  ServerStatus = "maintenance"
	StatusDecommission ServerStatus = "decommissioned"
)
