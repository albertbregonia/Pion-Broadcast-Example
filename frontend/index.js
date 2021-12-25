//WebRTC-Broadcast © Albert Bregonia 2021

const broadcast = document.getElementById(`broadcast`),
      whiteboard = document.getElementById(`whiteboard`);

// set up connections

function formatSignal(event, data) {
    return JSON.stringify({ 
        event: event, 
        data: JSON.stringify(data)
    });
}

const ws = new WebSocket(`wss://${location.hostname}:${location.port}/connect`); //create a websocket for WebRTC signaling 
ws.onopen = () => console.log(`Connected`);
ws.onclose = ws.onerror = ({reason}) => alert(`Disconnected ${reason}`);

const rtc = new RTCPeerConnection({iceServers: [{urls: `stun:stun.l.google.com:19302`}]}); //create a WebRTC instance
rtc.onicecandidate = ({candidate}) => candidate && ws.send(formatSignal(`ice`, candidate)); //if the ice candidate is not null, send it to the peer
rtc.oniceconnectionstatechange = () => rtc.iceConnectionState == `failed` && rtc.restartIce();
rtc.ontrack = ({streams}) => { console.log(streams); broadcast.srcObject = streams[0]; };

ws.onmessage = async ({data}) => { //signal handler
    const signal = JSON.parse(data),
          content = JSON.parse(signal.data);
    switch(signal.event) {
        case `offer-request`:
            console.log(`got offer-request!`);
            const whiteboardStream = whiteboard.captureStream(60);
            for(const track of whiteboardStream.getTracks())
                rtc.addTrack(track, whiteboardStream); //add whiteboard stream to offer
            whiteboardSetup();
            const offer = await rtc.createOffer();
            await rtc.setLocalDescription(offer);
            ws.send(formatSignal(`offer`, offer)); //send offer
            console.log(`sent offer!`, offer);
            break;
        case `offer`:
            console.log(`got offer!`, content);
            await rtc.setRemoteDescription(content); //accept offer
            const answer = await rtc.createAnswer();
            await rtc.setLocalDescription(answer);
            ws.send(formatSignal(`answer`, answer)); //send answer
            console.log(`sent answer!`, answer);
            break;
        case `answer`:
            console.log(`got answer!`, content);
            await rtc.setRemoteDescription(content); //accept answer
            break;
        case `ice`:
            console.log(`got ice!`, content);
            rtc.addIceCandidate(content); //add ice candidates
            break;
        default:
            console.log(`Invalid message:`, content);
    }
};

// set up drawing on the whiteboard

function whiteboardSetup() {
    whiteboard.brush = whiteboard.getContext(`2d`);
    whiteboard.brush.fillStyle = `white`;
    whiteboard.brush.fillRect(0, 0, whiteboard.width, whiteboard.height);
    whiteboard.brush.lineWidth = 5;
    whiteboard.brush.lineCap = `round`;
    whiteboard.addEventListener(`mousedown`, startDrawing);
    whiteboard.addEventListener(`mouseup`, stopDrawing);
    whiteboard.addEventListener(`mouseleave`, stopDrawing);
    whiteboard.addEventListener(`mousemove`, drawHandler);
    whiteboard.addEventListener(`touchmove`, drawHandler);
    whiteboard.addEventListener(`touchstart`, startDrawing);
    whiteboard.addEventListener(`touchend`, stopDrawing);
    whiteboard.addEventListener(`touchcancel`, stopDrawing);
}

function startDrawing(e) {
    whiteboard.isDrawing = true;
    drawHandler(e);
}

function stopDrawing() {
    whiteboard.isDrawing = false;
    whiteboard.brush.beginPath();
}

function drawHandler(e) {
    e.preventDefault();
    if(!whiteboard.isDrawing)
        return;
    let x = e.clientX - whiteboard.offsetLeft,
        y = e.clientY - whiteboard.offsetTop,
        clientX = e.clientX, 
        clientY = e.clientY;
    if (e.touches && e.touches.length == 1) {
        let touch = e.touches[0];
        x = touch.pageX - whiteboard.offsetLeft;
        y = touch.pageY - whiteboard.offsetTop;
        clientX = touch.pageX;
        clientY = touch.pageY;
    }
    whiteboard.brush.lineTo(x, y);
    whiteboard.brush.stroke();
    whiteboard.brush.beginPath();
    whiteboard.brush.moveTo(x, y);
}