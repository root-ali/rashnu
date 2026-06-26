package pricing

import (
	"time"

	"github.com/google/uuid"
)

// CreatePriceRequest represents the request to create pricing
type CreatePriceRequest struct {
	DatacenterID string `json:"datacenter_id" validate:"required"`
	Month        string `json:"month" validate:"required"` // format: "2006-01"
}

// PriceResponse represents the pricing data returned to clients
type PriceResponse struct {
	ID           uuid.UUID `json:"id"`
	DatacenterID uuid.UUID `json:"datacenter_id"`
	Month        string    `json:"month"` // format: "2006-01"
	CPUPrice     int64     `json:"cpu_price"`
	MemoryPrice  int64     `json:"memory_price"`
	SSDPrice     int64     `json:"ssd_price"`
	HDDPrice     int64     `json:"hdd_price"`
	GPUPrice     int64     `json:"gpu_price"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ListPricesResponse represents the response for listing all prices
type ListPricesResponse struct {
	Prices []*PriceResponse `json:"prices"`
	Count  int              `json:"count"`
}

// UpdatePriceRequest represents the request to update pricing
type UpdatePriceRequest struct {
	CPUPrice    *int64 `json:"cpu_price,omitempty"`
	MemoryPrice *int64 `json:"memory_price,omitempty"`
	SSDPrice    *int64 `json:"ssd_price,omitempty"`
	HDDPrice    *int64 `json:"hdd_price,omitempty"`
	GPUPrice    *int64 `json:"gpu_price,omitempty"`
}

// ToPriceResponse converts Price entity to PriceResponse DTO
func ToPriceResponse(price *Price) *PriceResponse {
	return &PriceResponse{
		ID:           price.ID,
		DatacenterID: price.DatacenterID,
		Month:        price.Date.Format("2006-01"),
		CPUPrice:     price.CPUPrice,
		MemoryPrice:  price.MemoryPrice,
		SSDPrice:     price.SSDPrice,
		HDDPrice:     price.HDDPrice,
		GPUPrice:     price.GPUPrice,
		CreatedAt:    price.CreatedAt,
		UpdatedAt:    price.UpdatedAt,
	}
}

// ToPriceResponseList converts a list of Price entities to PriceResponse DTOs
func ToPriceResponseList(prices []*Price) *ListPricesResponse {
	responses := make([]*PriceResponse, 0, len(prices))
	for _, price := range prices {
		responses = append(responses, ToPriceResponse(price))
	}

	return &ListPricesResponse{
		Prices: responses,
		Count:  len(responses),
	}
}

// FulfillPriceDataRequest represents the request to fulfill price data
type FulfillPriceDataRequest struct {
	StartTime string `json:"start_time" validate:"required"` // format: "2006-01-02"
}

// FulfillPriceDataResponse represents the response after fulfilling price data
type FulfillPriceDataResponse struct {
	Message     string    `json:"message"`
	StartTime   time.Time `json:"start_time"`
	MonthsCount int       `json:"months_count"`
	DaysCount   int       `json:"days_count"`
	CompletedAt time.Time `json:"completed_at"`
}
