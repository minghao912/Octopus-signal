package internal

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

func Receive(
	w http.ResponseWriter,
	req *http.Request,
	upgrader websocket.Upgrader,
	ids *map[string]Channel,
) {
	// Upgrade http request to web socket
	upgrader.CheckOrigin = func(req *http.Request) bool { return true } // Allow all origins
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Println(err)
		return
	}

	for {
		// Read the client's code
		_, incomingMessage, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}

		log.Printf("Received new receive request\n")

		// Split incoming data
		incomingMessageParts := strings.SplitN(string(incomingMessage), ": ", 2)
		incomingCode := incomingMessageParts[0]
		incomingSignalData := incomingMessageParts[1]
		log.Printf("Code was %s\n", incomingCode)

		// Try seeing if client code is in dict
		_, ok := (*ids)[incomingCode]

		fmt.Println("RECEIVE")
		fmt.Println(*ids)

		// If client code is not in dict (invalid), send rejection
		if !ok {
			log.Println("Advisory: Invalid code sent")

			err = conn.WriteMessage(websocket.TextMessage, []byte("ERROR: The requested code is invalid"))
			if err != nil {
				log.Println(err)
				break
			}

			continue
		}

		// If the "signal data" is the string INIT, this is the first message;
		// send back the sender's signal data
		if strings.HasPrefix(incomingSignalData, "INIT") {
			log.Printf("[%s]: RECEIVE: INIT request detected", incomingCode)

			channel, _ := (*ids)[incomingCode]

			if channel.SenderWSConn == nil {
				log.Printf("[%s]: Error: No sender associated with code %s\n", incomingCode, incomingCode)
				conn.WriteMessage(websocket.TextMessage, []byte("ERROR: No sender associated with this code"))

				continue
			}

			channel.RecipientWSConn = conn
			(*ids)[incomingCode] = channel

			conn.WriteMessage(websocket.TextMessage, []byte("OK"))
			channel.SenderWSConn.WriteMessage(websocket.TextMessage, []byte("Connection received"))
			log.Printf("[%s]: RECEIVE: Recipient activated", incomingCode)
		}
	}
}
