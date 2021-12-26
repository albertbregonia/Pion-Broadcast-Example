//WebRTC-Broadcast © Albert Bregonia 2021
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

// Main handler for creating/managing a WebSocket/WebRTC PeerConnection

func ErrorHandler() {
	if e := recover(); e != nil {
		debug.PrintStack()
		log.Println(e)
	}
}

func SignalingServer(w http.ResponseWriter, r *http.Request) {
	defer ErrorHandler()
	//create a thread safe websocket for signaling with JavaScript
	ws, e := wsUpgrader.Upgrade(w, r, nil)
	if e != nil {
		panic(e)
	}
	signaler := SignalingSocket{ws, sync.Mutex{}}
	defer signaler.Close()

	//create the WebRTC peer connection that will broadcast the remote stream to everyone
	peer, e := api.NewPeerConnection(config)
	if e != nil {
		panic(e)
	}
	if _, e := peer.AddTrack(whiteboard); e != nil {
		panic(e)
	}
	defer peer.Close()

	peer.OnConnectionStateChange(
		func(pcs webrtc.PeerConnectionState) { log.Println(`peer connection state:`, pcs.String()) })
	peer.OnICEConnectionStateChange(
		func(is webrtc.ICEConnectionState) { log.Println(`ice connection state:`, is.String()) })
	peer.OnICECandidate(func(ice *webrtc.ICECandidate) {
		defer ErrorHandler()
		if ice == nil {
			return
		}
		iceJS, e := json.Marshal(ice.ToJSON())
		if e != nil {
			panic(e)
		}
		signaler.SendSignal(Signal{`ice`, string(iceJS)})
	})

	peer.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		defer ErrorHandler()
		var lastTimestamp uint32 = 0
		keyframePeriod := 500 * time.Millisecond
		lastKeyframe := time.Now().Add(-keyframePeriod)
		for {
			packet, _, e := track.ReadRTP()
			if e != nil {
				panic(e)
			}
			if time.Since(lastKeyframe) >= keyframePeriod {
				if e := peer.WriteRTCP([]rtcp.Packet{
					&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}}); e != nil {
					panic(e)
				}
			}
			oldTimestamp := packet.Timestamp //old time stamp relative to the incoming track
			if lastTimestamp == 0 {
				packet.Timestamp = 0
			} else { //packet timestamps have been modified to be the change in time since the last frame
				packet.Timestamp -= lastTimestamp
			}
			lastTimestamp = oldTimestamp
			whiteboardPackets <- packet
		}
	})

	if e := signaler.SendSignal(Signal{`offer-request`, `{}`}); e != nil {
		panic(e)
	}
	signal := Signal{}
	for {
		if e := signaler.ReadJSON(&signal); e != nil {
			return
		}
		switch signal.Event {
		case `ice`:
			candidate := webrtc.ICECandidateInit{}
			if e := json.Unmarshal([]byte(signal.Data), &candidate); e != nil {
				panic(e)
			}
			if e := peer.AddICECandidate(candidate); e != nil {
				panic(e)
			}
		case `answer`:
			answer := webrtc.SessionDescription{}
			if e := json.Unmarshal([]byte(signal.Data), &answer); e != nil {
				panic(e)
			}
			if e := peer.SetRemoteDescription(answer); e != nil {
				panic(e)
			}
		case `offer`:
			offer := webrtc.SessionDescription{}
			if e := json.Unmarshal([]byte(signal.Data), &offer); e != nil {
				panic(e)
			}
			if e := peer.SetRemoteDescription(offer); e != nil {
				panic(e)
			}
			answer, e := peer.CreateAnswer(nil)
			if e != nil {
				panic(e)
			}
			if e := peer.SetLocalDescription(answer); e != nil {
				panic(e)
			}
			answerJS, e := json.Marshal(answer)
			if e != nil {
				panic(e)
			}
			if e := signaler.SendSignal(Signal{`answer`, string(answerJS)}); e != nil {
				panic(e)
			}
		}
	}
}
