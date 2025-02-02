package main

import (
	"log"

	"github.com/Coimbra1984/gortsplib"
	"github.com/Coimbra1984/gortsplib/pkg/base"
	"github.com/Coimbra1984/gortsplib/pkg/rtph264"
	"github.com/pion/rtp"
)

// This example shows how to
// 1. connect to a RTSP server and read all tracks on a path
// 2. check if there's an H264 track
// 3. decode H264 into raw frames

// This example requires the ffmpeg libraries, that can be installed in this way:
// apt install -y libavformat-dev libswscale-dev gcc pkg-config

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

	// setup H264->raw frames decoder
	h264dec, err := newH264Decoder()
	if err != nil {
		panic(err)
	}
	defer h264dec.close()

	// if present, send SPS and PPS from the SDP to the decoder
	if h264track.SPS() != nil {
		h264dec.decode(h264track.SPS())
	}
	if h264track.PPS() != nil {
		h264dec.decode(h264track.PPS())
	}

	// called when a RTP packet arrives
	c.OnPacketRTP = func(trackID int, pkt *rtp.Packet) {
		if trackID != h264TrackID {
			return
		}

		// decode H264 NALUs from the RTP packet
		nalus, _, err := rtpDec.Decode(pkt)
		if err != nil {
			return
		}

		for _, nalu := range nalus {
			// decode raw frames from H264 NALUs
			img, err := h264dec.decode(nalu)
			if err != nil {
				panic(err)
			}

			// wait for a frame
			if img == nil {
				continue
			}

			log.Printf("decoded frame with size %v", img.Bounds().Max)
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
