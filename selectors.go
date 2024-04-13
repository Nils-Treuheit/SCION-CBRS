package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/netsec-ethz/scion-apps/pkg/pan"
	"github.com/scionproto/scion/pkg/addr"
	"github.com/scionproto/scion/pkg/daemon"
	"github.com/scionproto/scion/pkg/sock/reliable"
	"github.com/scionproto/scion/scion/showpaths"
)

// hopPathTrace inspired by "github.com/netsec-ethz/scion-apps/pkg/pan/path_metadata.go"
func hopPath(pm *pan.PathMetadata) pan.PathHopSet {
	pathHops := make(pan.PathHopSet)
	for i := 0; i < len(pm.Interfaces)-1; i++ {
		pathHops[pan.PathHop{A: pm.Interfaces[i], B: pm.Interfaces[i+1]}] = struct{}{}
	}
	return pathHops
}

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
			if lat.Milliseconds() <= 25 {
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
	case 3:
		var hopPaths []pan.PathHopSet
		for idx := 0; idx < len(paths); idx++ {
			hopPaths = append(hopPaths, hopPath(paths[idx].Metadata))
			filtered = append(filtered, paths[idx])
		}
		sort.Slice(filtered, func(i, j int) bool {
			return hopPaths[i].SubsetOf(hopPaths[j])
		})
	}

	//fmt.Printf("Found %d paths viable paths!\n", len(filtered))
	return filtered
}

// round-robin reply selector
type RRReplySelector struct {
	mtx     sync.RWMutex
	hctx    *pan.HostContext
	remotes map[pan.UDPAddr]pan.RemoteEntry
	idx     int
	itcount int
	lim     int
	its     int
}

// content-based reply selector
type CBReplySelector struct {
	rrrs *RRReplySelector
	cid  int
}

// used for selected path or path range strategies
type StrategicReplySelector struct {
	cbrs    *CBReplySelector
	pathIDs []int
}

func NewRRReplySelector(nr_rr_paths int, rep_its int) *RRReplySelector {
	return &RRReplySelector{
		hctx:    pan.Host(),
		remotes: make(map[pan.UDPAddr]pan.RemoteEntry),
		idx:     1,
		itcount: 0,
		lim:     nr_rr_paths,
		its:     rep_its,
	}
}

func NewCBReplySelector(content_id int, nr_rr_paths int, rep_its int) *CBReplySelector {
	return &CBReplySelector{
		rrrs: &RRReplySelector{
			remotes: make(map[pan.UDPAddr]pan.RemoteEntry),
			hctx:    pan.Host(),
			idx:     1,
			itcount: 0,
			lim:     nr_rr_paths,
			its:     rep_its,
		},
		cid: content_id,
	}
}

func NewPathRangeReplySelector(content_id int, prange []int, rep_its int) *StrategicReplySelector {
	var pathRange []int
	range_start := prange[0]
	range_end := prange[1]
	for val := range_start; val < range_end; val++ {
		pathRange = append(pathRange, val)
	}
	return &StrategicReplySelector{
		cbrs: &CBReplySelector{
			rrrs: &RRReplySelector{
				remotes: make(map[pan.UDPAddr]pan.RemoteEntry),
				hctx:    pan.Host(),
				idx:     1,
				itcount: 0,
				lim:     len(pathRange),
				its:     rep_its,
			},
			cid: content_id,
		},
		pathIDs: pathRange,
	}
}

func NewSelectivePathReplySelector(content_id int, selectedPaths []int, rep_its int) *StrategicReplySelector {
	return &StrategicReplySelector{
		cbrs: &CBReplySelector{
			rrrs: &RRReplySelector{
				remotes: make(map[pan.UDPAddr]pan.RemoteEntry),
				hctx:    pan.Host(),
				idx:     1,
				itcount: 0,
				lim:     len(selectedPaths),
				its:     rep_its,
			},
			cid: content_id,
		},
		pathIDs: selectedPaths,
	}
}

func printShowpathsMetadata(local net.IP, remote addr.IA) {
	address, ok := os.LookupEnv("SCION_DAEMON_ADDRESS")
	if !ok {
		address = daemon.DefaultAPIAddress
	}
	dispatcher, ok := os.LookupEnv("SCION_DISPATCHER_SOCKET")
	if !ok {
		dispatcher = reliable.DefaultDispPath
	}
	var cfg showpaths.Config = showpaths.Config{
		Local:      local,
		Daemon:     address,
		MaxPaths:   showpaths.DefaultMaxPaths,
		Refresh:    false,
		NoProbe:    false,
		Sequence:   "",
		Dispatcher: dispatcher,
		Epic:       false,
	}
	extensivePathsResults, err := showpaths.Run(context.Background(), remote, cfg)
	if err != nil {
		fmt.Println(err)
	} else {
		extensivePathsResults.Human(os.Stdout, true, false)
	}
}

/*
Change Records method of the ReplySelectors
=> remotePaths are only set through Records method
*/
func (srs *StrategicReplySelector) Record(remote pan.UDPAddr, path *pan.Path) {
	if path == nil {
		return
	}

	srs.cbrs.rrrs.mtx.Lock()
	defer srs.cbrs.rrrs.mtx.Unlock()

	r, ok := srs.cbrs.rrrs.remotes[remote]
	if ok && len(r.Paths) > 0 {
		//fmt.Println("Paths already populated!")
		return
	}

	r.Seen = time.Now()
	// Check Showpaths Meta-Data Fields
	printShowpathsMetadata(srs.cbrs.rrrs.hctx.HostInLocalAS, addr.IA(remote.IA))
	// TODO: create better method to populate Meta-Data Fields
	paths, err := srs.cbrs.rrrs.hctx.QueryPaths(context.Background(), remote.IA)
	if err != nil {
		fmt.Println("ERORR while querying Paths, likely: No Paths found!")
		return
	}
	fmt.Printf("Found %d path(s)!\n", len(paths))

	paths = filterPaths(paths, srs.cbrs.cid)
	fmt.Printf("Filtered out %d paths.\n", len(paths))

	var newPaths []*pan.Path
	//for did, idx := range srs.pathIDs {
	for _, idx := range srs.pathIDs {
		//fmt.Printf("%d. Path ID is %d ", did, idx)
		if len(paths) > idx {
			newPaths = append(newPaths, paths[idx])
		}
		//fmt.Printf("and new paths array is now %d elements long.\n", len(newPaths))
	}

	fmt.Printf("Inserted %d path(s) into the record!\n", len(newPaths))
	r.Paths = newPaths
	srs.cbrs.rrrs.remotes[remote] = r
}

func (cbrs *CBReplySelector) Record(remote pan.UDPAddr, path *pan.Path) {
	if path == nil {
		return
	}

	cbrs.rrrs.mtx.Lock()
	defer cbrs.rrrs.mtx.Unlock()

	r, ok := cbrs.rrrs.remotes[remote]
	if ok && len(r.Paths) > 0 {
		//fmt.Println("Paths already populated!")
		return
	}

	r.Seen = time.Now()
	// Check Showpaths Meta-Data Fields
	printShowpathsMetadata(cbrs.rrrs.hctx.HostInLocalAS, addr.IA(remote.IA))
	// TODO: create better method to populate Meta-Data Fields
	paths, err := cbrs.rrrs.hctx.QueryPaths(context.Background(), remote.IA)
	if err != nil {
		fmt.Println("ERORR while querying Paths, likely: No Paths found!")
		return
	}
	fmt.Printf("Found %d path(s)!\n", len(paths))

	paths = filterPaths(paths, cbrs.cid)
	fmt.Printf("Filtered out %d paths.\n", len(paths))

	// limit to 5 or 10 best
	if len(paths) > cbrs.rrrs.lim {
		paths = paths[:cbrs.rrrs.lim]
	}

	fmt.Printf("Inserted %d path(s) into the record!\n", len(paths))
	r.Paths = paths
	cbrs.rrrs.remotes[remote] = r
}

// The Round-Robin_ReplySelector does not need a content based filter step
func (s *RRReplySelector) Record(remote pan.UDPAddr, path *pan.Path) {
	if path == nil {
		return
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	r, ok := s.remotes[remote]
	if ok && len(r.Paths) > 0 {
		//fmt.Println("Paths already populated!")
		return
	}

	r.Seen = time.Now()
	paths, err := s.hctx.QueryPaths(context.Background(), remote.IA)
	if err != nil {
		fmt.Println("ERORR while querying Paths, likely: No Paths found!")
		return
	}
	fmt.Printf("Found %d path(s)!\n", len(paths))

	// limit to 5 or 10 best
	if len(paths) > s.lim {
		paths = paths[:s.lim]
	}

	fmt.Printf("Inserted %d path(s) into the record!\n", len(paths))
	r.Paths = paths
	s.remotes[remote] = r
}

/*
Implement round-robin path selection
-> this should allow to emulate the default ReplySelector when round-robin path limit is set to 1
*/
func (rrrs *RRReplySelector) Path(remote pan.UDPAddr) *pan.Path {
	rrrs.mtx.RLock()
	defer rrrs.mtx.RUnlock()
	r, ok := rrrs.remotes[remote]
	if !ok || len(r.Paths) == 0 {
		//fmt.Println("No Paths found!")
		return nil
	}
	if rrrs.itcount < rrrs.its {
		rrrs.itcount += 1
	} else {
		rrrs.itcount = 0
		rrrs.idx += 1
		if rrrs.idx > len(r.Paths) {
			rrrs.idx = 1
		}
	}
	//fmt.Printf("Choose %d. path of %d found paths!\n", s.idx, len(r.Paths))
	return r.Paths[rrrs.idx-1]
}

/*
All following functions have been kept according to the default ReplySelector
as their function should be the same
-> side note the SmartReplySelector is a wrapper of the RRReplySelector as such it just calls for the wrapped ReplySelector's behavior
*/
func (rrrs *RRReplySelector) Initialize(local pan.UDPAddr) {
	//fmt.Println(len(s.remotes[local].Paths))
}

func (rrrs *RRReplySelector) PathDown(pan.PathFingerprint, pan.PathInterface) {
	// TODO failover.
}

func (rrrs *RRReplySelector) Close() error {
	return nil
}

func (cbrs *CBReplySelector) Initialize(local pan.UDPAddr) {
	cbrs.rrrs.Initialize(local)
}

func (cbrs *CBReplySelector) PathDown(pf pan.PathFingerprint, pi pan.PathInterface) {
	cbrs.rrrs.PathDown(pf, pi)
	// TODO failover.
}

func (cbrs *CBReplySelector) Close() error {
	return nil
}

func (cbrs *CBReplySelector) Path(remote pan.UDPAddr) *pan.Path {
	return cbrs.rrrs.Path(remote)
}

func (srs *StrategicReplySelector) Initialize(local pan.UDPAddr) {
	srs.cbrs.rrrs.Initialize(local)
}

func (srs *StrategicReplySelector) PathDown(pf pan.PathFingerprint, pi pan.PathInterface) {
	srs.cbrs.rrrs.PathDown(pf, pi)
	// TODO failover.
}

func (srs *StrategicReplySelector) Close() error {
	return nil
}

func (srs *StrategicReplySelector) Path(remote pan.UDPAddr) *pan.Path {
	return srs.cbrs.rrrs.Path(remote)
}
