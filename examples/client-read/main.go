package main

import (
	"log"

	"github.com/Coimbra1984/gortsplib"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

// This example shows how to connect to a RTSP server
// and read all tracks on a path.

func main() {
	c := gortsplib.Client{
		// called when a RTP packet arrives
		OnPacketRTP: func(trackID int, pkt *rtp.Packet) {
			log.Printf("RTP packet from track %d, payload type %d\n", trackID, pkt.Header.PayloadType)
		},
		// called when a RTCP packet arrives
		OnPacketRTCP: func(trackID int, pkt rtcp.Packet) {
			log.Printf("RTCP packet from track %d, type %T\n", trackID, pkt)
		},
	}

	// connect to the server and start reading all tracks
	panic(c.StartReadingAndWait("rtsp://localhost:8554/mystream"))
}
