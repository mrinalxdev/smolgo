package avsyncer

import (
	"container/heap"
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
		if seqDiff(seq, exp) > 1 {
			miss := uint16(seqDiff(seq,exp))
			for i := uint16(1); i < miss; i ++ {
				nack = append(nack, (exp + i - 1) &0xFFFF)
			}
		}
	}
	
	
	item := &pktItem{pkt : pkt}
	
	heap.Push(&jb.heap, item)
	jb.updateNextSeq()
	return nack
	
}

func (jb *JitterBuffer) Pop(now time.Time) (payload []byte, pts time.Duration, err error) {
	jb.mu.Lock()
	defer jb.mu.Unlock()

	if jb.heap.Len() == 0 {
		if jb.lastPayload == nil {
			return nil, 0, ErrNoData
		}
		return jb.plcFrame(), jb.lastDur, nil
	}

	const jitterMargin = 40 * time.Millisecond
	head := jb.heap[0].pkt
	headPts := jb.clock.RTPToDuration(head.Header.Timestamp)

	playWall := jb.clock.baseWall.Add(headPts)

	if now.Sub(playWall) > jitterMargin {
		heap.Pop(&jb.heap)
		jb.updateNextSeq()
		return jb.Pop(now) // recurse once
	}

	if playWall.Sub(now) > jitterMargin {
		if jb.lastPayload == nil {
			return nil, 0, ErrUnderrun
		}
		return jb.plcFrame(), jb.lastDur, nil
	}

	// emit real packet
	heap.Pop(&jb.heap)
	jb.updateNextSeq()

	payload = head.Payload
	pts = headPts

	// remember for PLC
	jb.lastPayload = append([]byte(nil), payload...)

	// duration of this frame = (timestamp 1e9 * (ts - baseTS) ) / rate
	deltaTS := int64(head.Header.Timestamp) - int64(jb.baseTS)
	nanos := deltaTS * 1_000_000_000 / int64(jb.clock.rate)
	jb.lastDur = time.Duration(nanos) * time.Nanosecond

	return payload, pts, nil
}

// func (jb *JitterBuffer) Pop(now time.Time) (payload []byte, pts time.Duration, err error) {
//     jb.mu.Lock()
//     defer jb.mu.Unlock()

//     if jb.heap.Len() == 0 {
//         if jb.lastPayload == nil {
//             return nil, 0, ErrNoData
//         }
//         return jb.plcFrame(), jb.lastDur, nil
//     }

//     const jitterMargin = 40 * time.Millisecond
//     head := jb.heap[0].pkt
//     headPts := jb.clock.RTPToDuration(head.Header.Timestamp)

//     playWall := jb.clock.baseWall.Add(headPts)

//     if now.Sub(playWall) > jitterMargin {
//         heap.Pop(&jb.heap)
//         jb.updateNextSeq()
//         return jb.Pop(now)
//     }

//     if playWall.Sub(now) > jitterMargin {
//         if jb.lastPayload == nil {
//             return nil, 0, ErrUnderrun
//         }
//         return jb.plcFrame(), jb.lastDur, nil
//     }

//     heap.Pop(&jb.heap)
//     jb.updateNextSeq()

//     payload = head.Payload
//     pts = headPts
 
//     jb.lastPayload = append([]byte(nil), payload...)
//     // jb.lastDur = time.Duration(int64(head.Header.Timestamp-jb.baseTS)*1e9/jb.clock.rate) * time.Nanosecond
//     if jb.baseTS == head.Header.Timestamp {
//         jb.lastDur = 0
//     }

//     return payload, pts, nil
// }



// func (jb *JitterBuffer) Pop(now time.Time) (payload []byte, pts time.Duration, err error){
// 	jb.mu.Lock()
// 	defer jb.mu.Unlock()
	
// 	if jb.heap.Len() == 0 {
// 		if jb.lastPayload == nil {
// 			return nil, 0, ErroNoData
// 		}
		
// 		return jb.plcFrame(), jb.lastDur, nil
// 	}
	
	
// 	const jitterMargin = 40 * time.Millisecond
// 	head := jb.heap[0].pkt
	
// 	headPts := jb.clock.RTPToDuration(head.Header.Timestamp)
	
// 	playWall := jb.clock.baseWall.Add(headPts)
	
	
// 	if now.Sub(playWall) > jitterMargin {
// 		head.Pop(&jb.heap)
// 		jb.updateNextSeq()
		
// 		return jb.Pop(now)
// 	}
	
	
// 	if playWall.Sub(now) > jitterMargin {
// 		if jb.lastPayload == nil {
// 			return nil, 0, ErrUnderrun
// 		}
		
// 		return jb.plcFrame(), jb.lastDur, nil
// 	}
// }



func (jb *JitterBuffer) plcFrame() []byte {
    orig := jb.lastPayload
    if len(orig) == 0 {
        return nil
    }
    stretch := int(float64(len(orig)) * plcStretch)
    plc := make([]byte, stretch)
    copy(plc, orig)

    tail := orig[len(orig)/2:]
    for i := len(orig); i < stretch; i++ {
        plc[i] = tail[(i-len(orig))%len(tail)]
    }
    for i := len(orig); i < stretch; i++ {
        factor := float32(stretch-i) / float32(stretch-len(orig))
        plc[i] = uint8(float32(plc[i]) * factor)
    }
    return plc
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


func (jb *JitterBuffer) updateNextSeq() {
	if jb.heap.Len() == 0 {
		return
	}
	
	
	var highest uint16
	
	for i := 0; i < jb.heap.Len(); i++ {
		cur := jb.heap[i].pkt.Header.SequenceNumber
		
		if i == 0 {
			highest = cur
		} else if seqDiff(cur, highest) != i + 1 {
			break
		}
		
		highest = cur
	}
	
	jb.nextSeq = (highest + 1) & 0xFFFF
}