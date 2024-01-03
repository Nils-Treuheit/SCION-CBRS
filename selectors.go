package main

import (
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

type SmartReplySelector struct {
	pan.DefaultReplySelector
	mtx     sync.RWMutex
	remotes map[pan.UDPAddr]remoteEntry
	idx     int
	cid     int
}

func NewSmartReplySelector() *SmartReplySelector {
	return &SmartReplySelector{
		remotes: make(map[pan.UDPAddr]remoteEntry),
		idx:     0,
		cid:     0,
	}
}

func (s *SmartReplySelector) Path(remote pan.UDPAddr) *pan.Path {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	r, ok := s.remotes[remote]
	var chosenPath *pan.Path = nil
	if !ok || len(r.paths) == 0 {
		return nil
	}
	chosenPath = r.paths[0]
	s.idx += 1
	if s.idx > len(r.paths) {
		s.idx = 1
	}
	if s.cid == 0 {
		found := false
		for idx := s.idx - 1; idx < len(r.paths); idx++ {
			if r.paths[idx].Metadata.MTU >= 1500 {
				chosenPath = r.paths[idx]
				found = true
				break
			}
		}
		if !found && s.idx > 1 {
			for idx := 0; idx < s.idx-1; idx++ {
				if r.paths[idx].Metadata.MTU >= 1500 {
					s.idx = 1
					return r.paths[idx]
				}
			}
			s.idx = 1
		}
	} else if s.cid == 1 {
		found := false
		for idx := s.idx - 1; idx < len(r.paths); idx++ {
			lats := r.paths[idx].Metadata.Latency
			var lat int64 = 0
			for i := 0; i < len(lats); i++ {
				lat += lats[i].Milliseconds()
			}
			if lat <= 20 {
				chosenPath = r.paths[idx]
				found = true
				break
			}
		}
		if !found && s.idx > 1 {
			for idx := 0; idx < s.idx-1; idx++ {
				lats := r.paths[idx].Metadata.Latency
				var lat int64 = 0
				for i := 0; i < len(lats); i++ {
					lat += lats[i].Milliseconds()
				}
				if lat <= 20 {
					s.idx = 1
					return r.paths[idx]
				}
			}
			s.idx = 1
		}
	} else if s.cid == 2 {
		found := false
		for idx := s.idx - 1; idx < len(r.paths); idx++ {
			bws := r.paths[idx].Metadata.Bandwidth
			var bw uint64 = 0
			for i := 0; i < len(bws); i++ {
				if bws[i] < bw {
					bw = bws[i]
				}
			}
			if bw >= 100000 {
				chosenPath = r.paths[idx]
				found = true
				break
			}
		}
		if !found && s.idx > 1 {
			for idx := 0; idx < s.idx-1; idx++ {
				bws := r.paths[idx].Metadata.Bandwidth
				var bw uint64 = 0
				for i := 0; i < len(bws); i++ {
					if bws[i] < bw {
						bw = bws[i]
					}
				}
				if bw >= 100000 {
					s.idx = 1
					return r.paths[idx]
				}
			}
			s.idx = 1
		}
	}
	return chosenPath
}
