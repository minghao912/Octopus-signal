# octopus-signal
Signaling server for Octopus. Uses websockets to keep track of sender IDs and file retreival codes.

Deployed via Docker on port 8080.

## Signal Process

1. Sender initiates the channel by sending `INIT` to the `/send` endpoint; server returns a six-digit code for user use
2. Sender sends WebRTC signalling data to the `/send` endpoint with the format `[CODE]: [DATA]`
3. Recipient indicates to the server which user it is supposed to connect to by sending `[CODE]: INIT` to the `/receive` endpoint
    * Immediately, the server responds with the signalling data of the sender, for use with the frontend
4. Recipient sends its signalling data to the `/receive` endpoint with the format `[CODE]: [DATA]`
    * Server automatically pushes this data to the sender
5. When channel is no longer needed, the six-digit code is sent to the `/remove` endpoint, which closes all associated websocket
    connections and deletes the map entry in memory
