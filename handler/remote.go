package handler

import (
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type RemotePrinter struct {
	MacAddress        string
	LastReceivedMsgID *int
	SessionStartTime  time.Time
	MsgTimestamps     []MsgEntry // Sorted list of (MsgID, RecvTimestamp)
	Mu                sync.Mutex
}

type MsgEntry struct {
	ID        int
	Timestamp float64
}

func NewRemotePrinter(mac string) *RemotePrinter {
	return &RemotePrinter{
		MacAddress:    mac,
		MsgTimestamps: make([]MsgEntry, 0),
	}
}

func (rp *RemotePrinter) RegisterReceivedMessage(msgID int) {
	rp.Mu.Lock()
	defer rp.Mu.Unlock()

	now := float64(time.Now().UnixNano()) / 1e9
	entry := MsgEntry{ID: msgID, Timestamp: now}

	// Binary search for insertion point (Sort.Search returns the index where it *should* be)
	idx := sort.Search(len(rp.MsgTimestamps), func(i int) bool {
		return rp.MsgTimestamps[i].ID >= msgID
	})

	// Duplicate check logic (simplified from python bisect logic)
	if idx < len(rp.MsgTimestamps) && rp.MsgTimestamps[idx].ID == msgID {
		log.Info().Msgf("detected duplicate datagram for %s", rp.MacAddress)
		return
	}

	// Insert
	rp.MsgTimestamps = append(rp.MsgTimestamps, MsgEntry{})
	copy(rp.MsgTimestamps[idx+1:], rp.MsgTimestamps[idx:])
	rp.MsgTimestamps[idx] = entry
}

func (rp *RemotePrinter) CreateReport(interval float64) map[string]interface{} {
	rp.Mu.Lock()
	defer rp.Mu.Unlock()

	rangeEnd := float64(time.Now().UnixNano()) / 1e9
	rangeStart := rangeEnd - interval

	// Drop older data
	dropIdx := 0
	for i, msg := range rp.MsgTimestamps {
		if msg.Timestamp >= rangeStart {
			dropIdx = i
			break
		}
		// If we reach the end and everything is old
		if i == len(rp.MsgTimestamps)-1 && msg.Timestamp < rangeStart {
			dropIdx = len(rp.MsgTimestamps)
		}
	}
	if dropIdx > 0 {
		rp.MsgTimestamps = rp.MsgTimestamps[dropIdx:]
	}

	if len(rp.MsgTimestamps) == 0 {
		return nil
	}

	lowest := rp.MsgTimestamps[0].ID
	highest := rp.MsgTimestamps[len(rp.MsgTimestamps)-1].ID
	expected := highest - lowest + 1
	received := len(rp.MsgTimestamps)

	dropRate := 1.0 - (float64(received) / float64(expected))

	return map[string]interface{}{
		"expected_number_of_messages": expected,
		"received_number_of_messages": received,
		"interval":                    interval,
		"drop_rate":                   dropRate,
	}
}
