package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/netsec-ethz/scion-apps/pkg/pan"
)

// works only if paths metaData is present
func filterPaths(paths pan.PathsMRU, cid int) pan.PathsMRU {
	var filtered pan.PathsMRU

	switch cid {
	case 0:
		var mtus []uint16
		for idx := 0; idx < len(paths); idx++ {
			//fmt.Printf("%d. Path's MTU:", idx+1)
			//fmt.Println(paths[idx].Metadata.MTU)
			var mtu uint16 = paths[idx].Metadata.MTU
			if mtu >= 1400 {
				filtered = append(filtered, paths[idx])
				mtus = append(mtus, mtu)
			}
		}
		sort.Slice(filtered, func(i, j int) bool {
			return mtus[i] < mtus[j]
		})
	case 1:
		for idx := 0; idx < len(paths); idx++ {
			// DEBUG: print non empty latency vectors
			if len(paths[idx].Metadata.Latency) > 0 {
				lats := paths[idx].Metadata.Latency
				for i := 0; i < len(lats); i++ {
					if lats[i] != 0 {
						fmt.Printf("%d. Path's Latencies:", idx+1)
						fmt.Println(lats)
						break
					}
				}
			}
			//fmt.Printf("%d. Path's Latencies:", idx+1)
			//fmt.Println(paths[idx].Metadata.Latency)
			lat, _ := paths[idx].Metadata.LatencySum()
			if lat.Milliseconds() <= 20 {
				filtered = append(filtered, paths[idx])
			}
		}
		sort.Slice(filtered, func(i, j int) bool {
			a, b := filtered[i].Metadata.LowerLatency(filtered[j].Metadata)
			return b && a
			//return lats[i] < lats[j]
		})
	case 2:
		for idx := 0; idx < len(paths); idx++ {
			// DEBUG: print non empty bandwidths vectors
			if len(paths[idx].Metadata.Bandwidth) > 0 {
				bws := paths[idx].Metadata.Bandwidth
				for i := 0; i < len(bws); i++ {
					if bws[i] != 0 {
						fmt.Printf("%d. Path's Bandwidths:", idx+1)
						fmt.Println(bws)
						break
					}
				}
			}
			//fmt.Printf("%d. Path's Bandwidths:", idx+1)
			//fmt.Println(paths[idx].Metadata.Bandwidth)
			bw, _ := paths[idx].Metadata.BandwidthMin()
			if bw >= 100000 {
				filtered = append(filtered, paths[idx])
			}
		}
		sort.Slice(filtered, func(i, j int) bool {
			a, b := filtered[i].Metadata.HigherBandwidth(filtered[j].Metadata)
			return b && a
			//return bws[i] > bws[j]
		})
	}

	//fmt.Printf("Found %d paths viable paths!\n", len(filtered))
	return filtered
}

type RRReplySelector struct {
	mtx     sync.RWMutex
	hctx    *pan.HostContext
	remotes map[pan.UDPAddr]pan.RemoteEntry
	idx     int
	lim     int
}

type SmartReplySelector struct {
	rrs *RRReplySelector
	cid int
}

func NewRRReplySelector(nr_rr_paths int) *RRReplySelector {
	return &RRReplySelector{
		hctx:    pan.Host(),
		remotes: make(map[pan.UDPAddr]pan.RemoteEntry),
		idx:     0,
		lim:     nr_rr_paths,
	}
}

func (s *RRReplySelector) Initialize(local pan.UDPAddr) {
	//fmt.Println(len(s.remotes[local].Paths))
}

func (s *RRReplySelector) Record(remote pan.UDPAddr, path *pan.Path) {
	if path == nil {
		return
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	r, ok := s.remotes[remote]
	if ok && len(r.Paths) > 0 {
		fmt.Println("No Paths found!")
		return
	}

	r.Seen = time.Now()
	paths, err := s.hctx.QueryPaths(context.Background(), remote.IA)
	if err != nil {
		fmt.Println("ERORR while querying Paths!")
		return
	}
	fmt.Printf("Found %d path(s)!\n", len(r.Paths))

	// limit to 5 or 10 best
	if len(paths) > s.lim {
		paths = paths[:s.lim]
	}

	fmt.Printf("Inserted %d path(s) into the record!\n", len(r.Paths))
	r.Paths = paths
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
	if !ok || len(r.Paths) == 0 {
		//fmt.Println("No Paths found!")
		return nil
	}
	s.idx += 1
	if s.idx > len(r.Paths) {
		s.idx = 1
	}
	//fmt.Printf("Choose %d. path of %d found paths!\n", s.idx, len(r.Paths))
	return r.Paths[s.idx-1]
}

func NewSmartReplySelector(content_id int, nr_rr_paths int) *SmartReplySelector {
	/*
	 => remotePaths are only set through Records method

	  remotePaths := make(map[new_pan.UDPAddr]pan.RemoteEntry)
	  fmt.Println(len(remotePaths))

	  for addr, entry := range remotePaths {
	  	entry.Paths = filterPaths(entry.Paths, content_id)
	  	remotePaths[addr] = entry
	  }
	*/

	return &SmartReplySelector{
		rrs: &RRReplySelector{
			remotes: make(map[pan.UDPAddr]pan.RemoteEntry),
			hctx:    pan.Host(),
			idx:     0,
			lim:     nr_rr_paths,
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

	r, ok := s.rrs.remotes[remote]
	if ok && len(r.Paths) > 0 {
		fmt.Println("No Paths found!")
		return
	}

	r.Seen = time.Now()
	paths, err := s.rrs.hctx.QueryPaths(context.Background(), remote.IA)
	if err != nil {
		fmt.Println("ERORR while querying Paths!")
		return
	}

	fmt.Printf("Found %d path(s)!\n", len(r.Paths))
	paths = filterPaths(paths, s.cid)

	// limit to 5 or 10 best
	if len(paths) > s.rrs.lim {
		paths = paths[:s.rrs.lim]
	}

	fmt.Printf("Inserted %d path(s) into the record!\n", len(r.Paths))
	r.Paths = paths
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
