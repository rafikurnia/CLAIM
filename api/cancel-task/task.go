package p

import (
	"crypto/rand"
	"fmt"

	"github.com/mr-tron/base58"
	"github.com/multiformats/go-multihash"
)

// Inspired from:
// https://github.com/libp2p/go-libp2p-core/blob/8293d284f2cdbd732ccbf5b648e99d6c5c5de7c0/test/peer.go#L12-L17
func randTaskID() (string, error) {
	buf := make([]byte, 16)
	rand.Read(buf)
	h, err := multihash.Sum(buf, multihash.SHA2_256, -1)
	if err != nil {
		return "", fmt.Errorf("multihash.Sum -> %w", err)
	}

	return base58.Encode([]byte(h)), nil
}

type task struct {
	ID               string
	VantagePoints    []string
	Probe            string
	Arguments        string
	Schedule         *schedule
	Type             string
	Status           string
	NumberOfSequence map[string]int
}

func newTask() (*task, error) {
	id, err := randTaskID()
	if err != nil {
		return nil, fmt.Errorf("randTaskID -> %w", err)
	}

	s, err := newSchedule("", "", "* * * * *")
	if err != nil {
		return nil, fmt.Errorf("newSchedule -> %w", err)
	}

	return &task{
		ID:               id,
		VantagePoints:    make([]string, 0),
		Schedule:         s,
		Status:           "scheduled",
		NumberOfSequence: make(map[string]int),
	}, nil
}
