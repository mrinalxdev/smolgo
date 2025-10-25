package avsyncer

import (
	"errors"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

var (
    ErrNoData   = errors.New("no data available")
    ErrUnderrun = errors.New("buffer underrun")
)


type SyncLayer struct {
    audio *JitterBuffer
    video *JitterBuffer
    mu    sync.Mutex
}


func NewSyncLayer() *SyncLayer {
    return &SyncLayer{
        audio: NewJitterBuffer(48000),
        video: NewJitterBuffer(90000),
    }
}

// IngestRTP routes a packet to the proper media buffer and returns
// any RTCP NACKs that should be sent upstream.
func (s *SyncLayer) IngestRTP(pkt *rtp.Packet) []rtcp.Packet {
    var jb *JitterBuffer
    switch pkt.Header.PayloadType {
    case 96:
        jb = s.audio
    case 97:
        jb = s.video
    default:
        return nil
    }
    nackSeqs := jb.Push(pkt)
    if len(nackSeqs) > 0 {
        return []rtcp.Packet{BuildNACK(pkt.Header.SSRC, nackSeqs)}
    }
    return nil
}
 

func (s *SyncLayer) Sync(now time.Time) (isAudio bool, payload []byte, pts time.Duration, err error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    aPay, aPts, aErr := s.audio.Pop(now)
    vPay, vPts, vErr := s.video.Pop(now)


    if aErr == nil && (vErr != nil || aPts < vPts) {
        return true, aPay, aPts, nil
    }
    if vErr == nil {
        return false, vPay, vPts, nil
    }

    if aErr == ErrUnderrun && s.audio.lastPayload != nil {
        return true, s.audio.plcFrame(), s.audio.lastDur, nil
    }
    if vErr == ErrUnderrun && s.video.lastPayload != nil {
        return false, s.video.plcFrame(), s.video.lastDur, nil
    }
    return false, nil, 0, ErrNoData
}