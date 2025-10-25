package avsyncer

import "github.com/pion/rtcp"

func BuildNACK(ssrc uint32, missing []uint16) rtcp.Packet{
	var blks []rtcp.NackPair
	
	if len(missing) > 0 {
		blk := rtcp.NackPair{PacketID: missing[0]}
		bitmask := uint16(0)
		
		for _, m := range missing [1:] {
			diff := int(m) - int(blk.PacketID)
			
			if diff == 1 {
				bitmask |= 1 << uint(len(missing)-2)
			} else {
				blks = append(blks, blk)
				blk = rtcp.NackPair{PacketID: m}
				bitmask = 0
			}
		}
		
		blks = append(blks, blk)
	}
	
	return &rtcp.TransportLayerNack{
		SenderSSRC : ssrc,
		MediaSSRC: ssrc,
		Nacks : blks,
	}
}