package inventory

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type DatacenterCreateRequest struct {
	Name           string  `json:"name"`
	Short          string  `json:"short"`
	Provider       string  `json:"provider"`
	Region         string  `json:"region"`
	Racks          int     `json:"racks"`
	PowerKW        float64 `json:"power_kw"`
	Tier           string  `json:"tier"`
	MonthlyColoFee int64   `json:"monthly_colo_fee"`
}

type DatacenterUpdateRequest struct {
	ID             uuid.UUID `json:"id" db:"id"`
	Name           string    `json:"name"`
	Short          string    `json:"short"`
	Provider       string    `json:"provider"`
	Region         string    `json:"region"`
	Racks          int       `json:"racks"`
	PowerKW        float64   `json:"power_kw"`
	Tier           string    `json:"tier"`
	MonthlyColoFee int64     `json:"monthly_colo_fee"`
}

type ServerCreateRequest struct {
	DataCenterID  uuid.UUID `json:"datacenter_id"`
	Hostname      string    `json:"hostname"`
	Region        string    `json:"region"`
	Vendor        string    `json:"vendor"`
	Model         string    `json:"model"`
	PurchasePrice int64     `json:"purchase_price"`
	PurchaseDate  Date      `json:"purchase_date"`
	AmortMonths   int       `json:"amort_months"`
	VCPUs         int       `json:"vcpus"`
	MemoryGB      int       `json:"memory_gb"`
	SSDCapacityGB int       `json:"ssd_capacity_gb"`
	HDDCapacityGB int       `json:"hdd_capacity_gb"`
	GPUs          int       `json:"gpus"`
	PowerWatts    int       `json:"power_watts"`
	RackUnits     int       `json:"rack_units"`
	Role          string    `json:"role"`
}

type ServerUpdateRequest struct {
	ID            uuid.UUID    `json:"id"`
	DataCenterID  uuid.UUID    `json:"datacenter_id"`
	Hostname      string       `json:"hostname"`
	Region        string       `json:"region"`
	Vendor        string       `json:"vendor"`
	Model         string       `json:"model"`
	PurchasePrice int64        `json:"purchase_price"`
	PurchaseDate  Date         `json:"purchase_date"`
	AmortMonths   int          `json:"amort_months"`
	VCPUs         int          `json:"vcpus"`
	MemoryGB      int          `json:"memory_gb"`
	SSDCapacityGB int          `json:"ssd_capacity_gb"`
	HDDCapacityGB int          `json:"hdd_capacity_gb"`
	GPUs          int          `json:"gpus"`
	PowerWatts    int          `json:"power_watts"`
	RackUnits     int          `json:"rack_units"`
	Role          string       `json:"role"`
	Status        ServerStatus `json:"status"`
}

type InfrastructureHardwareCreateRequest struct {
	DataCenterID  uuid.UUID       `json:"datacenter_id"`
	Hostname      string          `json:"hostname"`
	HardwareType  string          `json:"hardware_type"`
	Region        string          `json:"region"`
	PurchasePrice int64           `json:"purchase_price"`
	PurchaseDate  Date            `json:"purchase_date"`
	AmortMonths   int             `json:"amort_months"`
	Model         string          `json:"model"`
	Specs         json.RawMessage `json:"specs"`
}

type InfrastructureHardwareUpdateRequest struct {
	ID            uuid.UUID       `json:"id"`
	DataCenterID  uuid.UUID       `json:"datacenter_id"`
	Hostname      string          `json:"hostname"`
	HardwareType  string          `json:"hardware_type"`
	Region        string          `json:"region"`
	PurchasePrice int64           `json:"purchase_price"`
	PurchaseDate  Date            `json:"purchase_date"`
	AmortMonths   int             `json:"amort_months"`
	Model         string          `json:"model"`
	Specs         json.RawMessage `json:"specs"`
}

// DatacenterWithHardware represents a datacenter with all its associated hardware
type DatacenterWithHardware struct {
	DataCenter              *DataCenter               `json:"datacenter"`
	Servers                 []*Server                 `json:"servers"`
	InfrastructureHardwares []*InfrastructureHardware `json:"infrastructure_hardwares"`
}

// Date is a custom type that unmarshals dates in YYYY-MM-DD format
type Date struct {
	time.Time
}

// UnmarshalJSON implements json.Unmarshaler to parse dates in YYYY-MM-DD format
func (d *Date) UnmarshalJSON(b []byte) error {
	s := string(b)
	// Remove quotes
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}
	d.Time = t
	return nil
}

// MarshalJSON implements json.Marshaler to output dates in YYYY-MM-DD format
func (d *Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Time.Format("2006-01-02"))
}
