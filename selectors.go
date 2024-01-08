package main

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/netsec-ethz/scion-apps/pkg/pan"
)

type pathsMRU []*pan.Path

func (p *pathsMRU) insertp(path *pan.Path, maxEntries int) {
	paths := *p
	i := 0
	for ; i < len(paths); i++ {
		if paths[i].Fingerprint == path.Fingerprint {
			break
		}
	}
	if i == len(paths) {
		if len(paths) < maxEntries {
			*p = append(paths, nil)
			paths = *p
		} else {
			i = len(paths) - 1 // overwrite least recently used
		}
	}
	paths[i] = path

	// move most-recently-used to front
	if i != 0 {
		pi := paths[i]
		copy(paths[1:i+1], paths[0:i])
		paths[0] = pi
	}
}

// works only if paths metaData is present
func filterPaths(paths pathsMRU, cid int) pathsMRU {
	var filtered pathsMRU

	switch cid {
	case 0:
		var mtus []uint16
		for idx := 0; idx < len(paths); idx++ {
			fmt.Println(paths[idx])
			fmt.Println(paths[idx].Metadata)
			var mtu uint16 = paths[idx].Metadata.MTU
			if mtu >= 1500 {
				filtered = append(filtered, paths[idx])
				mtus = append(mtus, mtu)
			}
		}
		sort.Slice(filtered, func(i, j int) bool {
			return mtus[i] < mtus[j]
		})
	case 1:
		var lats []int64
		for idx := 0; idx < len(paths); idx++ {
			latencies := paths[idx].Metadata.Latency
			var lat int64 = 0 //simplify to paths[idx].Metadata.latencySum()[0].Milliseconds()
			for i := range latencies {
				lat += latencies[i].Milliseconds()
			}
			if lat <= 20 {
				filtered = append(filtered, paths[idx])
				lats = append(lats, lat)
			}
		}
		//could use LowerLatency comperator as shown below
		sort.Slice(filtered, func(i, j int) bool {
			return lats[i] < lats[j]
		})
	case 2:
		var bws []uint64
		for idx := 0; idx < len(paths); idx++ {
			bandwidths := paths[idx].Metadata.Bandwidth
			var bw uint64 = 0 //simplify to paths[idx].Metadata.bandwidthMin()[0]
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
		//could use HigherBandwidth comperator with lambda on filtered lambda x,y: x.Metadata.HigherBandwidth(y.Metadata)
		sort.Slice(filtered, func(i, j int) bool {
			return bws[i] > bws[j]
		})
	}

	fmt.Println(len(filtered))
	return filtered
}

type remoteEntry struct {
	paths pathsMRU
	seen  time.Time
}

type RRReplySelector struct {
	mtx     sync.RWMutex
	remotes map[pan.UDPAddr]remoteEntry
	idx     int
}

type SmartReplySelector struct {
	rrs *RRReplySelector
	cid int
}

func NewRRReplySelector() *RRReplySelector {
	return &RRReplySelector{
		remotes: make(map[pan.UDPAddr]remoteEntry),
		idx:     0,
	}
}

func (s *RRReplySelector) Initialize(local pan.UDPAddr) {
	//fmt.Println(len(s.remotes[local].paths))
}

func (s *RRReplySelector) Record(remote pan.UDPAddr, path *pan.Path) {
	if path == nil {
		return
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	r := s.remotes[remote]
	r.seen = time.Now()
	r.paths.insertp(path, 10)
	s.remotes[remote] = r
}

func (s *RRReplySelector) PathDown(pan.PathFingerprint, pan.PathInterface) {
	// TODO failover.
}

func (s *RRReplySelector) Close() error {
	return nil
}

func (s *RRReplySelector) Path(remote pan.UDPAddr) *pan.Path {
	s.mtx.RLock()
	defer s.mtx.RUnlock()
	r, ok := s.remotes[remote]
	if !ok || len(r.paths) == 0 {
		fmt.Println("No Paths found!")
		return nil
	}
	s.idx += 1
	if s.idx > len(r.paths) {
		s.idx = 1
	}
	fmt.Printf("Found %d paths!", len(r.paths))
	return r.paths[s.idx-1]
}

func NewSmartReplySelector(content_id int) *SmartReplySelector {
	/*
	 => remotePaths are only set through Records method

	  remotePaths := make(map[pan.UDPAddr]remoteEntry)
	  fmt.Println(len(remotePaths))

	  for addr, entry := range remotePaths {
	  	entry.paths = filterPaths(entry.paths, content_id)
	  	remotePaths[addr] = entry
	  }
	*/

	return &SmartReplySelector{
		rrs: &RRReplySelector{
			remotes: make(map[pan.UDPAddr]remoteEntry),
			idx:     0,
		},
		cid: content_id,
	}
}

func (s *SmartReplySelector) Initialize(local pan.UDPAddr) {
	s.rrs.Initialize(local)
}

func (s *SmartReplySelector) Record(remote pan.UDPAddr, path *pan.Path) {
	if path == nil {
		return
	}

	s.rrs.mtx.Lock()
	defer s.rrs.mtx.Unlock()

	r := s.rrs.remotes[remote]
	r.seen = time.Now()
	r.paths.insertp(path, 10)
	fmt.Printf("Inserted %d path(s) into the record!", len(r.paths))
	r.paths = filterPaths(r.paths, s.cid)
	s.rrs.remotes[remote] = r
}

func (s *SmartReplySelector) PathDown(pf pan.PathFingerprint, pi pan.PathInterface) {
	s.rrs.PathDown(pf, pi)
	// TODO failover.
}

func (s *SmartReplySelector) Close() error {
	return nil
}

func (s *SmartReplySelector) Path(remote pan.UDPAddr) *pan.Path {
	return s.rrs.Path(remote)
}
