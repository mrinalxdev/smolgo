package avsyncer

import (
	"sync"
	"time"

	"github.com/pion/rtp"
)

// first is to setup the priority queue for rtp packets ordered by sequence number
// where all the heap interface requirement is put up

type pktItem struct {
	pkt *rtp.Packet
	idx int
}

type pktHeap []*pktItem


func (h pktHeap) Len() int{return len(h)}
func (h pktHeap) Less(i, j int) bool {return h[i].pkt.Header.SequenceNumber < h[j].pkt.Header.SequenceNumber}
func (h pktHeap) Swap(i, j int) {h[i], h[j] = h[j], h[i]; h[i].idx, h[j].idx = i, j}

func (h *pktHeap) Push(x interface{}) {
	*h = append(*h, x.(*pktItem))
	
	(*h)[len(*h)-1].idx = len(*h) - 1
	
}

func (h *pktHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

// jittbuffer with two variables one for handling both ~6s @ 50 fps video and ~12s @ 48 kHz audio
// the stretch last frame by 20% on loss, then there will a structure for next sequence which will expect 
// a nxtseq number and another variable for rtp timestamp of first packet stored, there will something which is known
// as PLC state ... hmmm kinda out of my syllabus 
// 
// okkay got this definition from reddit and few more wisdom from reddit --> [https://www.reddit.com/r/PLC/comments/2202bz/discussion_how_do_you_guys_prefer_to_program/]
// here is my quick skim up ... plc programming, is a state refers to a distinct condition or status that a system can be in at any given time
// forming the basis of finite state machine 


const (
	maxJitterFrames = 300
	plcStretch = 1.2
)

type JitterBuffer struct {
	mu sync.Mutex
	clock *Clock
	heap pktHeap
	nextSeq uint16
	baseTS uint32
	
	lastPayload []byte
	lastDur time.Duration
}


func NewJitterBuffer(clockRate int) *JitterBuffer {
	jb := &JitterBuffer{
		clock: NewClock(clockRate),
		heap : make(pktHeap, 0, maxJitterFrames),
	}
	
	heap.Init(&jb.heap)
	
	return jb
}

// so now we need a function to basically insert the a package which may trigger rtcp nack if gaps > 1
// a quick skim up through what the funtion will be doing
// intialise base timestamp
// detect loss
// and if missing packets ? generate NACK list : do nothing (for now)

func (jb *JitterBuffer) Push(pkt *rtp.Packet) (nack []uint16) {
	jb.mu.Lock()
	defer jb.mu.Unlock()
	
	seq := pkt.Header.SequenceNumber
	if jb.heap.Len() == 0 {
		jb.baseTS = pkt.Header.Timestamp
		jb.clock.SetBase(time.Now(), jb.baseTS)
		jb.nextSeq = seq + 1
	}
	
	
	if jb.heap.Len() > 0 {
		exp := jb.nextSeq
		// if seqDiff(seq, exp) > 1 {
			
		// }
	}
	
}

// just a smol helper function to signed 16 bit difference, wrap aware

func seqDiff(a, b uint16) int {
	d := int32(a) - int32(b)
	
	if d > 32768 {
		d -= 65536
	} else if d < -32768 {
		d += 65536
	}
	
	return int(d)
}


