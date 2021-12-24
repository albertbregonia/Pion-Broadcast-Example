//WebRTC-Broadcast © Albert Bregonia 2021
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"
	"sync"

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
	defer peer.Close()
	if _, e := peer.AddTransceiverFromTrack(whiteboard, //add whiteboard video
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionSendonly}); e != nil {
		panic(e)
	}
	if _, e := peer.AddTrack(whiteboard); e != nil {
		panic(e)
	}

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
		log.Println(`got track!`)
		for {
			packet, _, e := track.ReadRTP()
			if e != nil {
				panic(e)
			}
			if e := whiteboard.WriteRTP(packet); e != nil {
				panic(e)
			}
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
