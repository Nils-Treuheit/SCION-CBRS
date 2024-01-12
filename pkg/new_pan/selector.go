// Copyright 2021 ETH Zurich
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package new_pan

import (
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/scionproto/scion/pkg/addr"
)

// Selector controls the path used by a single **dialed** socket. Stateful.
type Selector interface {
	// Path selects the path for the next packet.
	// Invoked for each packet sent with Write.
	Path() *Path
	// Initialize the selector for a connection with the initial list of paths,
	// filtered/ordered by the Policy.
	// Invoked once during the creation of a Conn.
	Initialize(local, remote UDPAddr, paths []*Path)
	// Refresh updates the paths. This is called whenever the Policy is changed or
	// when paths were about to expire and are refreshed from the SCION daemon.
	// The set and order of paths may differ from previous invocations.
	Refresh([]*Path)
	// PathDown is called whenever an SCMP down notification is received on any
	// connection so that the selector can adapt its path choice. The down
	// notification may be for unrelated paths not used by this selector.
	PathDown(PathFingerprint, PathInterface)
	Close() error
}

// DefaultSelector is a Selector for a single dialed socket.
// This will keep using the current path, starting with the first path chosen
// by the policy, as long possible.
// Faults are detected passively via SCMP down notifications; whenever such
// a down notification affects the current path, the DefaultSelector will
// switch to the first path (in the order defined by the policy) that is not
// affected by down notifications.
type DefaultSelector struct {
	mutex   sync.Mutex
	paths   []*Path
	current int
}

func NewDefaultSelector() *DefaultSelector {
	return &DefaultSelector{}
}

func (s *DefaultSelector) Path() *Path {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.paths) == 0 {
		return nil
	}
	return s.paths[s.current]
}

func (s *DefaultSelector) Initialize(local, remote UDPAddr, paths []*Path) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.paths = paths
	s.current = 0
}

func (s *DefaultSelector) Refresh(paths []*Path) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	newcurrent := 0
	if len(s.paths) > 0 {
		currentFingerprint := s.paths[s.current].Fingerprint
		for i, p := range paths {
			if p.Fingerprint == currentFingerprint {
				newcurrent = i
				break
			}
		}
	}
	s.paths = paths
	s.current = newcurrent
}

func (s *DefaultSelector) PathDown(pf PathFingerprint, pi PathInterface) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if current := s.paths[s.current]; isInterfaceOnPath(current, pi) || pf == current.Fingerprint {
		fmt.Println("down:", s.current, len(s.paths))
		better := stats.FirstMoreAlive(current, s.paths)
		if better >= 0 {
			// Try next path. Note that this will keep cycling if we get down notifications
			s.current = better
			fmt.Println("failover:", s.current, len(s.paths))
		}
	}
}

func (s *DefaultSelector) Close() error {
	return nil
}

type PingingSelector struct {
	// Interval for pinging. Must be positive.
	Interval time.Duration
	// Timeout for the individual pings. Must be positive and less than Interval.
	Timeout time.Duration

	mutex   sync.Mutex
	paths   []*Path
	current int
	local   scionAddr
	remote  scionAddr

	numActive    int64
	pingerCtx    context.Context
	pingerCancel context.CancelFunc
	pinger       *ping.Pinger
}

// SetActive enables active pinging on at most numActive paths.
func (s *PingingSelector) SetActive(numActive int) {
	s.ensureRunning()
	atomic.SwapInt64(&s.numActive, int64(numActive))
}

func (s *PingingSelector) Path() *Path {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.paths) == 0 {
		return nil
	}
	return s.paths[s.current]
}

func (s *PingingSelector) Initialize(local, remote UDPAddr, paths []*Path) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.local = local.scionAddr()
	s.remote = remote.scionAddr()
	s.paths = paths
	s.current = stats.LowestLatency(s.remote, s.paths)
}

func (s *PingingSelector) Refresh(paths []*Path) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.paths = paths
	s.current = stats.LowestLatency(s.remote, s.paths)
}

func (s *PingingSelector) PathDown(pf PathFingerprint, pi PathInterface) {
	s.reselectPath()
}

func (s *PingingSelector) reselectPath() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.current = stats.LowestLatency(s.remote, s.paths)
}

func (s *PingingSelector) ensureRunning() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.local.IA == s.remote.IA {
		return
	}
	if s.pinger != nil {
		return
	}
	s.pingerCtx, s.pingerCancel = context.WithCancel(context.Background())
	local := s.local.snetUDPAddr()
	pinger, err := ping.NewPinger(s.pingerCtx, host().dispatcher, local)
	if err != nil {
		return
	}
	s.pinger = pinger
	go s.pinger.Drain(s.pingerCtx)
	go s.run()
}

func (s *PingingSelector) run() {
	pingTicker := time.NewTicker(s.Interval)
	pingTimeout := time.NewTimer(0)
	if !pingTimeout.Stop() {
		<-pingTimeout.C // drain initial timer event
	}

	var sequenceNo uint16
	replyPending := make(map[PathFingerprint]struct{})

	for {
		select {
		case <-s.pingerCtx.Done():
			return
		case <-pingTicker.C:
			numActive := int(atomic.LoadInt64(&s.numActive))
			if numActive > len(s.paths) {
				numActive = len(s.paths)
			}
			if numActive == 0 {
				continue
			}

			activePaths := s.paths[:numActive]
			for _, p := range activePaths {
				replyPending[p.Fingerprint] = struct{}{}
			}
			sequenceNo++
			s.sendPings(activePaths, sequenceNo)
			resetTimer(pingTimeout, s.Timeout)
		case r := <-s.pinger.Replies:
			s.handlePingReply(r, replyPending, sequenceNo)
			if len(replyPending) == 0 {
				pingTimeout.Stop()
				s.reselectPath()
			}
		case <-pingTimeout.C:
			if len(replyPending) == 0 {
				continue // already handled above
			}
			for pf := range replyPending {
				stats.RecordLatency(s.remote, pf, s.Timeout)
				delete(replyPending, pf)
			}
			s.reselectPath()
		}
	}
}

func (s *PingingSelector) sendPings(paths []*Path, sequenceNo uint16) {
	for _, p := range paths {
		remote := s.remote.snetUDPAddr()
		remote.Path = p.ForwardingPath.dataplanePath
		remote.NextHop = net.UDPAddrFromAddrPort(p.ForwardingPath.underlay)
		err := s.pinger.Send(s.pingerCtx, remote, sequenceNo, 16)
		if err != nil {
			new_panic(err)
		}
	}
}

func (s *PingingSelector) handlePingReply(reply ping.Reply,
	expectedReplies map[PathFingerprint]struct{},
	expectedSequenceNo uint16) {
	if reply.Error != nil {
		// handle NotifyPathDown.
		// The Pinger is not using the normal scmp handler in raw.go, so we have to
		// reimplement this here.
		pf, err := reversePathFingerprint(reply.Path)
		if err != nil {
			return
		}
		switch e := reply.Error.(type) { //nolint:errorlint
		case ping.InternalConnectivityDownError:
			pi := PathInterface{
				IA:   IA(e.IA),
				IfID: IfID(e.Egress),
			}
			stats.NotifyPathDown(pf, pi)
		case ping.ExternalInterfaceDownError:
			pi := PathInterface{
				IA:   IA(e.IA),
				IfID: IfID(e.Interface),
			}
			stats.NotifyPathDown(pf, pi)
		}
		return
	}

	if reply.Source.Host.Type() != addr.HostTypeIP {
		return // ignore replies from non-IP addresses
	}
	src := scionAddr{
		IA: IA(reply.Source.IA),
		IP: reply.Source.Host.IP(),
	}
	if src != s.remote || reply.Reply.SeqNumber != expectedSequenceNo {
		return
	}
	pf, err := reversePathFingerprint(reply.Path)
	if err != nil {
		return
	}
	if _, expected := expectedReplies[pf]; !expected {
		return
	}
	stats.RecordLatency(s.remote, pf, reply.RTT())
	delete(expectedReplies, pf)
}

func (s *PingingSelector) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.pinger == nil {
		return nil
	}
	s.pingerCancel()
	return s.pinger.Close()
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

type RRReplySelector struct {
	mtx     sync.RWMutex
	hctx    *hostContext
	remotes map[UDPAddr]remoteEntry
	idx     int
}

type SmartReplySelector struct {
	rrs *RRReplySelector
	cid int
}

func NewRRReplySelector() *RRReplySelector {
	return &RRReplySelector{
		hctx:    host(),
		remotes: make(map[UDPAddr]remoteEntry),
		idx:     0,
	}
}

func (s *RRReplySelector) Initialize(local UDPAddr) {
	//fmt.Println(len(s.remotes[local].paths))
}

func (s *RRReplySelector) Record(remote UDPAddr, path *Path) {
	if path == nil {
		return
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	r, ok := s.remotes[remote]
	if ok && len(r.paths) > 0 {
		return
	}

	r.seen = time.Now()
	paths, err := s.hctx.queryPaths(context.Background(), remote.IA)
	if err != nil {
		return
	}

	r.paths = paths
	s.remotes[remote] = r
}

func (s *RRReplySelector) PathDown(PathFingerprint, PathInterface) {
	// TODO failover.
}

func (s *RRReplySelector) Close() error {
	return nil
}

func (s *RRReplySelector) Path(remote UDPAddr) *Path {
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

	  remotePaths := make(map[new_pan.UDPAddr]remoteEntry)
	  fmt.Println(len(remotePaths))

	  for addr, entry := range remotePaths {
	  	entry.paths = filterPaths(entry.paths, content_id)
	  	remotePaths[addr] = entry
	  }
	*/

	return &SmartReplySelector{
		rrs: &RRReplySelector{
			remotes: make(map[UDPAddr]remoteEntry),
			hctx:    host(),
			idx:     0,
		},
		cid: content_id,
	}
}

func (s *SmartReplySelector) Initialize(local UDPAddr) {
	s.rrs.Initialize(local)
}

func (s *SmartReplySelector) Record(remote UDPAddr, path *Path) {
	if path == nil {
		return
	}

	s.rrs.mtx.Lock()
	defer s.rrs.mtx.Unlock()

	r, ok := s.rrs.remotes[remote]
	if ok && len(r.paths) > 0 {
		return
	}

	r.seen = time.Now()
	paths, err := s.rrs.hctx.queryPaths(context.Background(), remote.IA)
	if err != nil {
		return
	}

	r.paths = paths
	fmt.Printf("Inserted %d path(s) into the record!", len(r.paths))
	r.paths = filterPaths(r.paths, s.cid)
	s.rrs.remotes[remote] = r
}

func (s *SmartReplySelector) PathDown(pf PathFingerprint, pi PathInterface) {
	s.rrs.PathDown(pf, pi)
	// TODO failover.
}

func (s *SmartReplySelector) Close() error {
	return nil
}

func (s *SmartReplySelector) Path(remote UDPAddr) *Path {
	return s.rrs.Path(remote)
}
