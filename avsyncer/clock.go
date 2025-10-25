package avsyncer

import "time"

type Clock struct {
	rate int
	baseWall time.Time
	baseRTP uint32
}

func NewClock(rate int) *Clock {
	return &Clock{rate : rate, baseWall : time.Now(), baseRTP: 0}
}

func (c *Clock) SetBase(wall time.Time, rtp uint32){
	c.baseWall, c.baseRTP = wall, rtp
	
}


/*
 * RTP timestamps are expressed in 'clock rate' ticks, in which the audio is usually 48 kHz, video 90kHz, we keep a reference point (wall time <--> RTP timestamp) and convert to time, duration
 * 
 * and there will be another function for rtp <--> duration which converts rtp timestamp to a presentation offset
 */
 

func (c *Clock) RTPToDuration(ts uint32) time.Duration{
	delta := int64(ts) - int64(c.baseRTP)
	
	if delta < -(1<<31) {
		delta += 1 << 32
	} else if delta > (1<<31) {
		delta -= 1 << 32
	}
	
	
	// return time.Duration(delta*1e9/c.rate) * time.Nanosecond
	return time.Duration(delta*1e9/c.rate) * time.Nanosecond
}