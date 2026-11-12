package ldnet_test

import (
	ldnet "ld-backend/internal/impl/core/net"
	"ld-backend/internal/spec"
	"sync"
	"testing"
)

func TestPortsSelector(t *testing.T) {
	pRange := ldspec.PortRange{MinPort: 10000, MaxPort: 10010}
	selector := ldnet.NewPortsSelector(pRange)
	ports := make(map[uint16]bool)
	for i := 0; i < 10; i++ {
		port, err := selector.GetPort()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ports[port] {
			t.Fatalf("port %d was already used", port)
		}
		ports[port] = true
	}
	_, err := selector.GetPort()
	if err == nil {
		t.Fatal("expected error")
	}

	usedPorts := selector.GetUsedPorts()
	if len(usedPorts) != 10 {
		t.Fatalf("expected 10 used ports, got %d", len(usedPorts))
	}

	for _, port := range usedPorts {
		selector.ReleasePort(port)
	}

	usedPorts = selector.GetUsedPorts()
	if len(usedPorts) != 0 {
		t.Fatalf("expected 0 used ports, got %d", len(usedPorts))
	}
}

func TestPortsSelectorConcurrent(t *testing.T) {
	pRange := ldspec.PortRange{MinPort: 10000, MaxPort: 10010}
	selector := ldnet.NewPortsSelector(pRange)
	ports := sync.Map{}
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go testPortsGetPort(selector, t, &ports, &wg)
	}
	wg.Wait()
	_, err := selector.GetPort()
	if err == nil {
		t.Fatal("expected error")
	}

	usedPorts := selector.GetUsedPorts()
	if len(usedPorts) != 10 {
		t.Fatalf("expected 10 used ports, got %d", len(usedPorts))
	}

	for _, port := range usedPorts {
		selector.ReleasePort(port)
	}

	usedPorts = selector.GetUsedPorts()
	if len(usedPorts) != 0 {
		t.Fatalf("expected 0 used ports, got %d", len(usedPorts))
	}
}

func testPortsGetPort(selector *ldnet.PortsSelector, t *testing.T, ports *sync.Map, wg *sync.WaitGroup) {
	port, err := selector.GetPort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, ok := ports.Load(port)
	if ok {
		t.Fatalf("port %d was already used", port)
	}
	ports.Store(port, true)
	wg.Done()
}
