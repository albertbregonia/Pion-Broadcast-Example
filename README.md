# Pion WebRTC Broadcast Example
This repository serves as an example/template for creating a WebRTC based Web Application.

This is the format I usually follow and I will use this repository as a boilerplate to constantly `git clone`
when testing new ideas for projects that use [pion/webrtc](https://github.com/pion/webrtc) in Golang.
A lot of this setup is usually very tedious and  that is why I have organized everything into a manner 
that will be easily debuggable and allow me to get to production faster.

## What it does:
This example creates a Web Server on port `443` (therefore, `https`) and upon a successful connection to the frontend,
the `index.js` file will create a WebSocket connection to `https://localhost/connect` and the necessary event handlers.

Upon successful connection to the backend, the HTTP handler `SignalingServer` will establish the handshake for the WebSocket
and the WebRTC peer connection. `SignalingServer` will do this operation in the following order:
1. Accept the WebSocket connection
2. Create a WebRTC peer connection
3. Add the dummy video track for broadcasting
4. Creates the necessary event handlers
5. Signals the frontend through the WebSocket to send its offer
6. Replies to the received offer
7. Sends ICE candidates
8. Waits for packets in the `OnTrack` function to write to the broadcast track

The frontend will then initialize the event handlers for drawing on the canvas and the video feed
from drawing on the canvas should be distributed to all peers connected to the server through the dummy track

## Dependencies
- [pion/webrtc](https://github.com/pion/webrtc)
- [gorilla/websocket](https://github.com/gorilla/websocket)
