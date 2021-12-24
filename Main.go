//WebRTC-Broadcast © Albert Bregonia 2021
package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v3"
)

var (
	wsUpgrader = websocket.Upgrader{
		ReadBufferSize:  512,
		WriteBufferSize: 512,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}
	config = webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{URLs: []string{`stun:stun.l.google.com:19302`}}},
	}
	whiteboard *webrtc.TrackLocalStaticRTP
	api        *webrtc.API
)

//Run setup and host webserver on https://localhost/
func main() {
	RTCSetup()
	http.Handle(`/`, http.FileServer(http.Dir(`frontend`)))
	http.HandleFunc(`/connect`, SignalingServer)
	log.Println(`Server Initialized`)
	log.Fatal(http.ListenAndServeTLS(`:443`, `server.crt`, `server.key`, nil))
}

//register codecs to receive
func RTCSetup() {
	mediaEngine := &webrtc.MediaEngine{}
	if e := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{ //support VP8 video
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeVP8,
			ClockRate:    90000,
			Channels:     0,
			SDPFmtpLine:  "",
			RTCPFeedback: nil,
		},
		PayloadType: 96,
	}, webrtc.RTPCodecTypeVideo); e != nil {
		panic(e)
	}
	registry := &interceptor.Registry{}
	if e := webrtc.RegisterDefaultInterceptors(mediaEngine, registry); e != nil {
		log.Fatal(e)
	}
	api = webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine), webrtc.WithInterceptorRegistry(registry))
	whiteboard, _ = webrtc.NewTrackLocalStaticRTP( //register shared video track to broadcast an HTML canvas video stream
		webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeVP8,
			ClockRate:    90000,
			Channels:     0,
			SDPFmtpLine:  "",
			RTCPFeedback: nil,
		},
		`whiteboard`,
		`whiteboard`,
	)
}

//SignalingSocket is a thread safe WebSocket used only for establishing WebRTC connections
type SignalingSocket struct {
	*websocket.Conn
	sync.Mutex
}

//SendSignal is a thread safe wrapper for the `websocket.WriteJSON()` function
func (signaler *SignalingSocket) SendSignal(v interface{}) error {
	signaler.Lock()
	defer signaler.Unlock()
	return signaler.WriteJSON(v)
}

//Signals to be written on a SignalingSocket in order to establish WebRTC connections
type Signal struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}
