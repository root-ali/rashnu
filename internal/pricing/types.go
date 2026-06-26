package pricing

import (
	"time"

	"github.com/google/uuid"
)

type Price struct {
	ID           uuid.UUID
	DatacenterID uuid.UUID
	Date         time.Time
	CPUPrice     int64
	MemoryPrice  int64
	SSDPrice     int64
	HDDPrice     int64
	GPUPrice     int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type perMonthPrice struct {
	DatacenterID uuid.UUID
	Region       string
	Date         time.Time
	TotalPrice   int64
	VCPUPrice    int64
	MemoryPrice  int64
	SSDPrice     int64
	HDDPrice     int64
	GPUPrice     int64
}

type perDayPrice struct {
	DatacenterID uuid.UUID
	Region       string
	Date         time.Time
	TotalPrice   int64
	VCPUPrice    int64
	MemoryPrice  int64
	SSDPrice     int64
	HDDPrice     int64
	GPUPrice     int64
}

// DailyPriceEntry is the exported per-unit daily price for a datacenter,
// used by the cost reporting service.
type DailyPriceEntry struct {
	DatacenterID uuid.UUID
	VCPUPrice    int64
	MemoryPrice  int64
	SSDPrice     int64
	HDDPrice     int64
	GPUPrice     int64
}
