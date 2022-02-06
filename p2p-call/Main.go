package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	wsUpgrader = websocket.Upgrader{
		ReadBufferSize:  512,
		WriteBufferSize: 512,
	}
	users    = make(map[string]*SignalingSocket)
	usersMtx = sync.RWMutex{}
)

func main() {
	http.Handle(`/`, http.FileServer(http.Dir(`frontend`)))
	http.HandleFunc(`/register`, RegisterUser)
	http.HandleFunc(`/call`, ConnectUser)
	log.Println(`Server Initialized`)
	log.Fatal(http.ListenAndServeTLS(`:443`, `server.crt`, `server.key`, nil))
}

//SignalingSocket is a thread safe WebSocket used only for establishing WebRTC connections
type SignalingSocket struct {
	*websocket.Conn
	sync.Mutex
}

//SendSignal is a thread safe wrapper for the `websocket.WriteJSON()` function
func (signaler *SignalingSocket) SendSignal(s Signal) error {
	signaler.Lock()
	defer signaler.Unlock()
	return signaler.WriteJSON(s)
}

//Signals to be written on a SignalingSocket in order to establish WebRTC connections
type Signal struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

//RegisterUser creates a WebSocket connection for the user given a username and adds them to a global map of users
func RegisterUser(w http.ResponseWriter, r *http.Request) {
	if e := r.ParseForm(); e != nil {
		http.Error(w, e.Error(), http.StatusBadRequest)
		return
	}
	username := r.FormValue(`username`)
	if username == `` {
		http.Error(w, `error: username cannot be empty`, http.StatusBadRequest)
		return
	}
	usersMtx.RLock()
	signaler := users[username]
	usersMtx.RUnlock()
	if signaler != nil {
		http.Error(w, fmt.Sprintf(`error: the username: '%v' is already taken`, username), http.StatusBadRequest)
		return
	}
	ws, e := wsUpgrader.Upgrade(w, r, nil)
	if e != nil {
		http.Error(w, e.Error(), http.StatusBadRequest)
		return
	}
	usersMtx.Lock()
	users[username] = &SignalingSocket{ws, sync.Mutex{}}
	users[username].SetCloseHandler(func(_ int, _ string) error {
		usersMtx.Lock() //if the display websocket closes, remove it from pending
		defer usersMtx.Unlock()
		delete(users, username)
		return nil
	})
	usersMtx.Unlock()
}

//ConnectUser handles connecting two users together and then forwarding their messages to each other
func ConnectUser(w http.ResponseWriter, r *http.Request) {
	if e := r.ParseForm(); e != nil {
		http.Error(w, e.Error(), http.StatusBadRequest)
		return
	}
	caller, callee := r.FormValue(`caller`), r.FormValue(`callee`)
	if caller == `` || callee == `` {
		http.Error(w, `error: username of caller and callee cannot be empty`, http.StatusBadRequest)
		return
	}
	usersMtx.Lock()
	callerSignaler := users[caller]
	calleeSignaler := users[callee]
	delete(users, caller)
	delete(users, callee)
	usersMtx.Unlock()

	if callerSignaler == nil {
		http.Error(w, fmt.Sprintf(`error: cannot call as '%v', user not found`, caller), http.StatusNotFound)
		return
	} else if calleeSignaler == nil {
		http.Error(w, fmt.Sprintf(`error: cannot call'%v', user not found`, callee), http.StatusNotFound)
		return
	}

	defer func() {
		callerSignaler.Close()
		calleeSignaler.Close()
	}()
	callerSignaler.SendSignal(Signal{`offer-request`, `{}`}) //tell caller to send their video offer
	go Signaler(callerSignaler, calleeSignaler)              //send signals from the caller to the callee
	Signaler(calleeSignaler, callerSignaler)                 //send signals from the callee to the caller
}

//Signaler is a function that allows two frontends to communicate with each other using WebSockets
func Signaler(from, to *SignalingSocket) error {
	var signal Signal
	for {
		if e := from.ReadJSON(&signal); e != nil { //receive signals from front end
			return e
		}
		to.SendSignal(signal) //send signal to peer
	}
}
