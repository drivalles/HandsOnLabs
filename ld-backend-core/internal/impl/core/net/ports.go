package ldnet

import (
	"fmt"
	"ld-backend/internal/spec"
	"math/rand/v2"
	"sync"
)

type PortsSelector struct {
	usedPorts map[uint16]bool
	portRange ldspec.PortRange
	mu        sync.Mutex
}

func NewPortsSelector(portRange ldspec.PortRange) *PortsSelector {
	return &PortsSelector{
		usedPorts: make(map[uint16]bool),
		portRange: portRange,
		mu:        sync.Mutex{},
	}
}

func (s *PortsSelector) GetPort() (uint16, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	totalPorts := int(s.portRange.MaxPort - s.portRange.MinPort)
	if len(s.usedPorts) >= totalPorts {
		return 0, fmt.Errorf("no ports available")
	}

	usagePercent := float64(len(s.usedPorts)) / float64(totalPorts)
	if usagePercent > 0.8 {
		return s.peekFirst()
	}

	return s.peekRandom()
}

func (s *PortsSelector) GetRTPPorts() (uint16, uint16, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	totalPorts := int(s.portRange.MaxPort - s.portRange.MinPort)
	if len(s.usedPorts) >= totalPorts {
		return 0, 0, fmt.Errorf("no ports available")
	}

	usagePercent := float64(len(s.usedPorts)) / float64(totalPorts)
	if usagePercent > 80 {
		return s.peekFirstPair()
	}

	return s.peekRandomPair()

}

func (s *PortsSelector) ReleasePort(port uint16) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.usedPorts, port)
}

func (s *PortsSelector) GetUsedPorts() []uint16 {
	s.mu.Lock()
	defer s.mu.Unlock()

	ports := make([]uint16, 0, len(s.usedPorts))
	for port := range s.usedPorts {
		ports = append(ports, port)
	}
	return ports
}

func (s *PortsSelector) randInt() uint16 {
	return uint16(uint32(s.portRange.MinPort) + rand.Uint32N(uint32(s.portRange.MaxPort-s.portRange.MinPort)))
}

func (s *PortsSelector) peekFirst() (uint16, error) {
	for i := s.portRange.MinPort; i <= s.portRange.MaxPort; i++ {
		if !s.usedPorts[i] {
			s.usedPorts[i] = true
			return i, nil
		}
	}
	return 0, fmt.Errorf("no ports available")
}

func (s *PortsSelector) peekFirstPair() (uint16, uint16, error) {
	for i := s.portRange.MinPort; i < s.portRange.MaxPort; i++ {
		if i%2 == 0 && !s.usedPorts[i] && !s.usedPorts[i+1] {
			s.usedPorts[i] = true
			s.usedPorts[i+1] = true
			return i, i + 1, nil
		}
	}
	return 0, 0, fmt.Errorf("no ports available")
}

func (s *PortsSelector) peekRandom() (uint16, error) {
	port := s.randInt()
	for s.usedPorts[port] {
		port = s.randInt()
	}
	s.usedPorts[port] = true
	return port, nil
}

func (s *PortsSelector) peekRandomPair() (uint16, uint16, error) {
	port := s.randInt()
	for port%2 != 0 || s.usedPorts[port] || s.usedPorts[port+1] {
		port = s.randInt()
	}
	s.usedPorts[port] = true
	s.usedPorts[port+1] = true
	return port, port + 1, nil
}
