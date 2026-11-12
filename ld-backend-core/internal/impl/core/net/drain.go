package ldnet

import "time"

const (
	// DrainWait is how long a drain waits for the next packet: the buffer is
	// empty once nothing arrives in this time, far less than a packet interval.
	DrainWait = time.Millisecond
	// DrainMaxPackets bounds a drain, in case packets keep arriving faster
	// than they are read.
	DrainMaxPackets = 100000
)

// Drain reads and discards the packets buffered in a stream, and returns how
// many were discarded. Every read gets its own deadline: a single deadline for
// the whole drain makes the reads fail once it passes, even with packets still
// buffered, so only part of the buffer would be discarded on a busy machine.
// The caller must restore the deadline the stream reads with afterwards.
func Drain(setReadDeadline func(time.Time) error, read func() error) int {
	count := 0
	for count < DrainMaxPackets {
		if err := setReadDeadline(time.Now().Add(DrainWait)); err != nil {
			break
		}
		if err := read(); err != nil {
			break // nothing arrived in time: the buffer is drained
		}
		count++
	}
	return count
}
