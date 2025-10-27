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
	OnAck(bytes int, metrices AlgorithmMetrices)
	OnLoss(metrics AlgorithmMetrices)
	GetWindowSize()
}

type AlgorithmMetrices struct {
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


