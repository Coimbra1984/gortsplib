package main

import (
	"github.com/Coimbra1984/gortsplib"
	"github.com/Coimbra1984/gortsplib/pkg/base"
	"github.com/Coimbra1984/gortsplib/pkg/rtph264"
	"github.com/pion/rtp"
)

// This example shows how to
// 1. connect to a RTSP server and read all tracks on a path
// 2. check if there's a H264 track
// 3. save the content of the H264 track into a file in MPEG-TS format

func main() {
	c := gortsplib.Client{}

	// parse URL
	u, err := base.ParseURL("rtsp://localhost:8554/mystream")
	if err != nil {
		panic(err)
	}

	// connect to the server
	err = c.Start(u.Scheme, u.Host)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	// find published tracks
	tracks, baseURL, _, err := c.Describe(u)
	if err != nil {
		panic(err)
	}

	// find the H264 track
	h264TrackID, h264track := func() (int, *gortsplib.TrackH264) {
		for i, track := range tracks {
			if h264track, ok := track.(*gortsplib.TrackH264); ok {
				return i, h264track
			}
		}
		return -1, nil
	}()
	if h264TrackID < 0 {
		panic("H264 track not found")
	}

	// setup RTP->H264 decoder
	rtpDec := &rtph264.Decoder{}
	rtpDec.Init()

	// setup H264->MPEGTS encoder
	enc, err := newMPEGTSEncoder(h264track.SPS(), h264track.PPS())
	if err != nil {
		panic(err)
	}

	// called when a RTP packet arrives
	c.OnPacketRTP = func(trackID int, pkt *rtp.Packet) {
		if trackID != h264TrackID {
			return
		}

		// decode H264 NALUs from the RTP packet
		nalus, pts, err := rtpDec.DecodeUntilMarker(pkt)
		if err != nil {
			return
		}

		// encode H264 NALUs into MPEG-TS
		err = enc.encode(nalus, pts)
		if err != nil {
			return
		}
	}

	// start reading tracks
	err = c.SetupAndPlay(tracks, baseURL)
	if err != nil {
		panic(err)
	}

	// wait until a fatal error
	panic(c.Wait())
}
