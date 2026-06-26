package cost_report

import "github.com/google/uuid"

const platformKubernetes = "kubernetes"

// blendedCPURAM holds averaged CPU and RAM unit prices for multi-DC Kubernetes services.
type blendedCPURAM struct {
	VCPUPrice   int64
	MemoryPrice int64
}

// uniqueWorkloadDatacenters returns distinct datacenter IDs from workloads, preserving first-seen order.
func uniqueWorkloadDatacenters(workloads []WorkloadInput) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(workloads))
	ids := make([]uuid.UUID, 0, len(workloads))
	for _, w := range workloads {
		if _, ok := seen[w.DatacenterID]; ok {
			continue
		}
		seen[w.DatacenterID] = struct{}{}
		ids = append(ids, w.DatacenterID)
	}
	return ids
}

// shouldBlendCPURAM returns true when a Kubernetes service spans two or more datacenters.
func shouldBlendCPURAM(platform string, datacenterCount int) bool {
	return platform == platformKubernetes && datacenterCount >= 2
}

// averageCPURAMPrices averages CPU and RAM unit prices across datacenters that have pricing for the day.
// Returns nil when fewer than two datacenters have price data.
func averageCPURAMPrices(datacenterIDs []uuid.UUID, priceMap map[uuid.UUID]*DailyUnitPrice) *blendedCPURAM {
	if len(datacenterIDs) < 2 {
		return nil
	}

	var sumCPU, sumRAM int64
	var count int
	for _, dcID := range datacenterIDs {
		p, ok := priceMap[dcID]
		if !ok {
			continue
		}
		sumCPU += p.VCPUPrice
		sumRAM += p.MemoryPrice
		count++
	}
	if count < 2 {
		return nil
	}

	return &blendedCPURAM{
		VCPUPrice:   sumCPU / int64(count),
		MemoryPrice: sumRAM / int64(count),
	}
}

// workloadResourceCosts computes per-resource costs for one workload.
// When blend is non-nil, CPU and RAM use the blended prices; storage and GPU stay datacenter-specific.
func workloadResourceCosts(w WorkloadInput, p *DailyUnitPrice, blend *blendedCPURAM) (cpu, ram, ssd, hdd, gpu int64) {
	cpuPrice := p.VCPUPrice
	ramPrice := p.MemoryPrice
	if blend != nil {
		cpuPrice = blend.VCPUPrice
		ramPrice = blend.MemoryPrice
	}

	cpu = int64(w.VCPU) * cpuPrice
	ram = int64(w.MemoryGB) * ramPrice
	ssd = int64(w.SSDGB) * p.SSDPrice
	hdd = int64(w.HDDGB) * p.HDDPrice
	gpu = int64(w.GPUs) * p.GPUPrice
	return cpu, ram, ssd, hdd, gpu
}
