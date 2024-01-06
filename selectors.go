package main

import (
	"sort"
	"sync"
	"time"

	"github.com/netsec-ethz/scion-apps/pkg/pan"
)

type pathsMRU []*pan.Path

type remoteEntry struct {
	paths pathsMRU
	seen  time.Time
}

type RRReplySelector struct {
	pan.DefaultReplySelector
	mtx     sync.RWMutex
	remotes map[pan.UDPAddr]remoteEntry
	idx     int
}

func NewRRReplySelector() *RRReplySelector {
	return &RRReplySelector{
		remotes: make(map[pan.UDPAddr]remoteEntry),
		idx:     0,
	}
}

func NewSmartReplySelector(content_id int) *RRReplySelector {
	remotePaths := make(map[pan.UDPAddr]remoteEntry)

	for addr, entry := range remotePaths {
		entry.paths = filterPaths(entry.paths, content_id)
		remotePaths[addr] = entry
	}

	return &RRReplySelector{
		remotes: remotePaths,
		idx:     0,
	}
}

func (s *RRReplySelector) Path(remote pan.UDPAddr) *pan.Path {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	r, ok := s.remotes[remote]
	if !ok || len(r.paths) == 0 {
		return nil
	}
	s.idx += 1
	if s.idx > len(r.paths) {
		s.idx = 1
	}
	return r.paths[s.idx-1]
}

func filterPaths(paths pathsMRU, cid int) pathsMRU {
	var filtered pathsMRU
	var mtus []uint16
	var lats []int64
	var bws []uint64

	for idx := range paths {
		switch cid {
		case 0:
			var mtu uint16 = paths[idx].Metadata.MTU
			if mtu >= 1500 {
				filtered = append(filtered, paths[idx])
				mtus = append(mtus, mtu)
			}
		case 1:
			latencies := paths[idx].Metadata.Latency
			var lat int64 = 0
			for i := range latencies {
				lat += latencies[i].Milliseconds()
			}
			if lat <= 20 {
				filtered = append(filtered, paths[idx])
				lats = append(lats, lat)
			}
		case 2:
			bandwidths := paths[idx].Metadata.Bandwidth
			var bw uint64 = 0
			for i := range bandwidths {
				if bandwidths[i] < bw || bw == 0 {
					bw = bandwidths[i]
				}
			}
			if bw >= 100000 {
				filtered = append(filtered, paths[idx])
				bws = append(bws, bw)
			}
		}
	}

	// Sort filteredPaths by index
	switch cid {
	case 0:
		sort.Slice(filtered, func(i, j int) bool {
			return mtus[i] < mtus[j]
		})
	case 1:
		sort.Slice(filtered, func(i, j int) bool {
			return lats[i] < lats[j]
		})
	case 2:
		sort.Slice(filtered, func(i, j int) bool {
			return bws[i] > bws[j]
		})
	}

	return filtered
}
