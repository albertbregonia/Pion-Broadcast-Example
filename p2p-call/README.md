# Peer to Peer Call Example

A map is used to save websocket connections and usernames of connected users.

To use this example:
- Start the server with the makefile by running `make`
- Connect to [`https://localhost/`](https://localhost/) in the browser
- Have two people create unique usernames and hit `Register`
- Have one person call the other by entering the peer's username into the second dialog box and then hit `Call`

The peers will automatically send offers and answers once these steps are followed correctly and a `<video>` element will be created upon receiving a valid WebRTC video stream.