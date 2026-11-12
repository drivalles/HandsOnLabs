package ldnet_test

import (
	"errors"
	ldnet "ld-backend/internal/impl/core/net"
	"testing"
	"time"
)

// testStream is a buffered stream whose reads fail once the deadline passed,
// even with packets left, like pion's packetio.Buffer and Go's sockets.
type testStream struct {
	buffered  int
	readDelay time.Duration
	deadline  time.Time
}

func (s *testStream) setReadDeadline(t time.Time) error {
	s.deadline = t
	return nil
}

func (s *testStream) read() error {
	if time.Now().After(s.deadline) {
		return errors.New("timeout")
	}
	if s.buffered == 0 {
		time.Sleep(time.Until(s.deadline))
		return errors.New("timeout")
	}
	// a busy machine: every read takes a while
	time.Sleep(s.readDelay)
	s.buffered--
	return nil
}

func TestDrainSlowReads(t *testing.T) {
	// 10s of audio, each read slower than the whole drain used to last
	stream := &testStream{buffered: 500, readDelay: 200 * time.Microsecond}
	count := ldnet.Drain(stream.setReadDeadline, stream.read)
	if count != 500 || stream.buffered != 0 {
		t.Fatalf("expected the 500 packets drained, got %d with %d left", count, stream.buffered)
	}
}

func TestDrainEmpty(t *testing.T) {
	stream := &testStream{}
	start := time.Now()
	if count := ldnet.Drain(stream.setReadDeadline, stream.read); count != 0 {
		t.Fatalf("expected nothing drained, got %d", count)
	}
	if elapsed := time.Since(start); elapsed > 20*time.Millisecond {
		t.Fatalf("an empty drain took %v", elapsed)
	}
}

func TestDrainBounded(t *testing.T) {
	stream := &testStream{buffered: ldnet.DrainMaxPackets + 10}
	if count := ldnet.Drain(stream.setReadDeadline, stream.read); count != ldnet.DrainMaxPackets {
		t.Fatalf("expected the drain to stop at %d, got %d", ldnet.DrainMaxPackets, count)
	}
}

func TestDrainDeadlineError(t *testing.T) {
	failing := func(time.Time) error { return errors.New("closed") }
	if count := ldnet.Drain(failing, func() error { return nil }); count != 0 {
		t.Fatalf("expected no reads when the deadline can not be set, got %d", count)
	}
}
