//WebRTC-Broadcast © Albert Bregonia 2021
package main

import (
	"errors"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/interceptor"
	"github.com/pion/rtp"
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
	whiteboard        *webrtc.TrackLocalStaticRTP
	whiteboardPackets = make(chan *rtp.Packet)
	api               *webrtc.API
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
			SDPFmtpLine:  ``,
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
			SDPFmtpLine:  ``,
			RTCPFeedback: nil,
		},
		`whiteboard`,
		`whiteboard`,
	)
	go func() {
		//whiteboard goroutine to handle incoming packets and give packets a valid sequence number
		//so that packets are going to be sent in as if we were reading frames from a file
		var currentTimestamp uint32 = 0
		for sequenceNumber := uint16(0); ; sequenceNumber++ {
			packet := <-whiteboardPackets
			if sequenceNumber > 10 { //take in 10 sample frames
				if avg := currentTimestamp / uint32(sequenceNumber); packet.Timestamp > avg {
					packet.Timestamp = avg // if the time elapsed since a packet's last frame is large, just set it to the average to avoid video delay
				}
			}
			currentTimestamp += packet.Timestamp //adjust the simulated timestamp
			packet.Timestamp = currentTimestamp
			packet.SequenceNumber = sequenceNumber
			if e := whiteboard.WriteRTP(packet); e != nil {
				if errors.Is(e, io.ErrClosedPipe) {
					return
				}
				panic(e)
			}
		}
	}()
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
