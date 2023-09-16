package internal

import "github.com/gorilla/websocket"

// Map each code to a "channel"
// Contains sender's websocket connection, recipient's websocket connection
// as well as sender's signalling data and recipient's signalling data
type Channel struct {
	SenderWSConn    *websocket.Conn `json:"senderWSConn"`
	RecipientWSConn *websocket.Conn `json:"recipientWSConn"`
	FileData        FileData        `json:"fileData"`
	ChunksReceived  uint32          `json:"chunksReceived"`
}
