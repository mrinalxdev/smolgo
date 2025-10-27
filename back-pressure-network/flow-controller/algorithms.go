package flowcontroller

import "time"

/*
 * So the algo is going to have a structure of algometrics in which it
 * will be providing the data to congestion algorithm. then a bbralgo which implements
 * bbr like congestion control with bandwidth estimation window ...
 *
 * More over the interesting part is the congestionalgorithm will be defining the interface for congestion
 * algo
 */

type CongestionAlgorithm interface {
	OnAck(bytes int, metrices AlgorithmMetrics)
	OnLoss(metrics AlgorithmMetrics)
	GetWindowSize()
}

type AlgorithmMetrics struct {
	WindowSize float64
	CurrentRTT time.Duration
	BaseRTT time.Duration
	PacketLoss float64
	BytesInFlight int64
	State string
}

type BBRAlgorithm struct {
	bwWindow [10]float64
	bwIndex int
	minRTT time.Duration
	maxBandwidth float64
	cycleCount int
}

func NewBBRAlgorithm() *BBRAlgorithm {
	return &BBRAlgorithm {
		minRTT : time.Second,
		bwWindow: [10]float64{},
		bwIndex: 0,
	}
}


/*
 * So for the bbra algo here is a small skim up for the following
 * - update min rtt
 * - calculate instantaneous bandwidth (like we can go bytes per second)
 * - during the calculation we can perform two things which
 * - update bandwidth window
 * - update max bandwidth
 */
 
 func (b *BBRAlgorithm) OnAck(bytes int, metrics AlgorithmMetrics) {
	if metrics.CurrentRTT > 0 && (metrics.CurrentRTT < b.minRTT || b.minRTT == 0) {
		b.minRTT = metrics.CurrentRTT
	}
	if metrics.CurrentRTT > 0 {
		instantBW := float64(bytes) / metrics.CurrentRTT.Seconds()
		b.bwWindow[b.bwIndex] = instantBW
		b.bwIndex = (b.bwIndex + 1) % len(b.bwWindow)
		b.updateMaxBandwidth()
	}
 }

 func (b *BBRAlgorithm) updateMaxBandwidth() {
	var maxBW float64
	for _, bw := range b.bwWindow {
		if bw > maxBW {
			maxBW = bw
		}
	}
	
	// Smooth the max bandwidth estimate
	if b.maxBandwidth == 0 {
		b.maxBandwidth = maxBW
	} else {
		b.maxBandwidth = 0.9*b.maxBandwidth + 0.1*maxBW
	}
 }