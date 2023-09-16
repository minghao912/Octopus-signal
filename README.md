# octopus-signal
Web server for Octopus. Uses websockets to keep track of generated codes and to pass messages between users.

Deployed via Docker on port 8088.

## Process

1. Sender initiates the channel by sending `INIT` to the `/send` endpoint; server returns a six-digit code for frontend use
2. If the user wants to send a file, metadata is sent to the `/send` endpoint with the format `[CODE]: FILE,[FILENAME],[FILESIZE]`
3. Sender sends file (base64 encoded) or text data to the `/send` endpoint with the format `[CODE]: [DATA]`
4. Recipient indicates to the server which user it is supposed to connect to by sending `[CODE]: INIT` to the `/receive` endpoint
5. Frontend initiates data transfer by streaming data to the server in chunks of 1KB or less. Server immediately forwards each message to the recipient.
6. When channel is no longer needed, the six-digit code is sent to the `/remove?code=[CODE]` endpoint, which closes all associated websocket
    connections and deletes the map entry in memory
