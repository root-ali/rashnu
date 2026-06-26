package cost_report

import (
	"testing"

	"github.com/google/uuid"
)

func TestAverageCPURAMPrices(t *testing.T) {
	dc1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	dc2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	dc3 := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	priceMap := map[uuid.UUID]*DailyUnitPrice{
		dc1: {DatacenterID: dc1, VCPUPrice: 100, MemoryPrice: 50},
		dc2: {DatacenterID: dc2, VCPUPrice: 200, MemoryPrice: 80},
		dc3: {DatacenterID: dc3, VCPUPrice: 300, MemoryPrice: 70},
	}

	t.Run("two datacenters", func(t *testing.T) {
		blend := averageCPURAMPrices([]uuid.UUID{dc1, dc2}, priceMap)
		if blend == nil {
			t.Fatal("expected blend")
		}
		if blend.VCPUPrice != 150 || blend.MemoryPrice != 65 {
			t.Fatalf("blend: cpu=%d ram=%d, want 150/65", blend.VCPUPrice, blend.MemoryPrice)
		}
	})

	t.Run("three datacenters", func(t *testing.T) {
		blend := averageCPURAMPrices([]uuid.UUID{dc1, dc2, dc3}, priceMap)
		if blend == nil {
			t.Fatal("expected blend")
		}
		if blend.VCPUPrice != 200 || blend.MemoryPrice != 66 {
			t.Fatalf("blend: cpu=%d ram=%d, want 200/66", blend.VCPUPrice, blend.MemoryPrice)
		}
	})

	t.Run("single datacenter returns nil", func(t *testing.T) {
		if blend := averageCPURAMPrices([]uuid.UUID{dc1}, priceMap); blend != nil {
			t.Fatalf("expected nil, got %+v", blend)
		}
	})

	t.Run("missing price for one datacenter still averages available", func(t *testing.T) {
		blend := averageCPURAMPrices([]uuid.UUID{dc1, dc2, uuid.New()}, priceMap)
		if blend == nil {
			t.Fatal("expected blend")
		}
		if blend.VCPUPrice != 150 || blend.MemoryPrice != 65 {
			t.Fatalf("blend: cpu=%d ram=%d, want 150/65", blend.VCPUPrice, blend.MemoryPrice)
		}
	})
}

func TestWorkloadResourceCosts(t *testing.T) {
	dc1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	p := &DailyUnitPrice{
		DatacenterID: dc1,
		VCPUPrice:    100,
		MemoryPrice:  50,
		SSDPrice:     10,
		HDDPrice:     5,
		GPUPrice:     1000,
	}
	w := WorkloadInput{
		VCPU:         4,
		MemoryGB:     8,
		SSDGB:        100,
		HDDGB:        200,
		GPUs:         1,
		DatacenterID: dc1,
	}

	cpu, ram, ssd, hdd, gpu := workloadResourceCosts(w, p, nil)
	if cpu != 400 || ram != 400 || ssd != 1000 || hdd != 1000 || gpu != 1000 {
		t.Fatalf("native costs: cpu=%d ram=%d ssd=%d hdd=%d gpu=%d", cpu, ram, ssd, hdd, gpu)
	}

	blend := &blendedCPURAM{VCPUPrice: 150, MemoryPrice: 60}
	cpu, ram, ssd, hdd, gpu = workloadResourceCosts(w, p, blend)
	if cpu != 600 || ram != 480 {
		t.Fatalf("blended cpu/ram: cpu=%d ram=%d, want 600/480", cpu, ram)
	}
	if ssd != 1000 || hdd != 1000 || gpu != 1000 {
		t.Fatalf("storage/gpu should stay DC-specific: ssd=%d hdd=%d gpu=%d", ssd, hdd, gpu)
	}
}

func TestShouldBlendCPURAM(t *testing.T) {
	if !shouldBlendCPURAM(platformKubernetes, 2) {
		t.Fatal("kubernetes with 2 DCs should blend")
	}
	if !shouldBlendCPURAM(platformKubernetes, 3) {
		t.Fatal("kubernetes with 3 DCs should blend")
	}
	if shouldBlendCPURAM(platformKubernetes, 1) {
		t.Fatal("single DC kubernetes should not blend")
	}
	if shouldBlendCPURAM("vm", 3) {
		t.Fatal("vm should not blend")
	}
}

func TestMultiDCServiceCostUsesBlendedCPURAM(t *testing.T) {
	dc1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	dc2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	priceMap := map[uuid.UUID]*DailyUnitPrice{
		dc1: {DatacenterID: dc1, VCPUPrice: 100, MemoryPrice: 50, SSDPrice: 10, HDDPrice: 5, GPUPrice: 0},
		dc2: {DatacenterID: dc2, VCPUPrice: 300, MemoryPrice: 90, SSDPrice: 20, HDDPrice: 8, GPUPrice: 0},
	}

	svc := &ServiceInput{
		Platform: platformKubernetes,
		Workloads: []WorkloadInput{
			{VCPU: 2, MemoryGB: 4, SSDGB: 10, DatacenterID: dc1, DatacenterName: "DC-1"},
			{VCPU: 2, MemoryGB: 4, SSDGB: 10, DatacenterID: dc2, DatacenterName: "DC-2"},
		},
	}

	serviceDCs := uniqueWorkloadDatacenters(svc.Workloads)
	blend := averageCPURAMPrices(serviceDCs, priceMap)
	if blend == nil {
		t.Fatal("expected blend")
	}
	if blend.VCPUPrice != 200 || blend.MemoryPrice != 70 {
		t.Fatalf("blend: cpu=%d ram=%d, want 200/70", blend.VCPUPrice, blend.MemoryPrice)
	}

	var totalCPU, totalRAM int64
	for _, w := range svc.Workloads {
		p := priceMap[w.DatacenterID]
		cpu, ram, _, _, _ := workloadResourceCosts(w, p, blend)
		totalCPU += cpu
		totalRAM += ram
	}

	// 2 pods × (2 vCPU × 200 + 4 GB × 70)
	if totalCPU != 800 || totalRAM != 560 {
		t.Fatalf("service totals: cpu=%d ram=%d, want 800/560", totalCPU, totalRAM)
	}

	// Per-pod costs differ by DC even though service totals match when workloads are symmetric.
	cpuDC1, _, _, _, _ := workloadResourceCosts(svc.Workloads[0], priceMap[dc1], blend)
	cpuDC2, _, _, _, _ := workloadResourceCosts(svc.Workloads[1], priceMap[dc2], blend)
	if cpuDC1 != cpuDC2 {
		t.Fatalf("blended per-pod CPU should be equal: dc1=%d dc2=%d", cpuDC1, cpuDC2)
	}
	nativeDC2, _, _, _, _ := workloadResourceCosts(svc.Workloads[1], priceMap[dc2], nil)
	if cpuDC2 == nativeDC2 {
		t.Fatalf("DC2 pod CPU should differ under blended vs native pricing")
	}
}
