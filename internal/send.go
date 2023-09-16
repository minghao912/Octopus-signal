package internal

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func Send(
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
		fmt.Fprintf(w, "ERROR: Could not upgrade to web socket\n"+err.Error())
		return
	}

	for {
		// Read the client's message (sender ID)
		_, incomingMessage, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}

		// If it's a new request generate IDs etc.
		if strings.HasPrefix(string(incomingMessage), "INIT") {
			log.Printf("Received new send request")

			// Generate random number 0 - 999999
			s := rand.NewSource(time.Now().UnixNano())

			num := rand.New(s).Intn(999999)
			id := fmt.Sprintf("%06d", num)
			for _, exist := (*ids)[id]; exist; _, exist = (*ids)[id] { // Keep generating new id until it is unique
				num = rand.New(s).Intn(999999)
				id = fmt.Sprintf("%06d", num)
				log.Println("here")
			}

			log.Printf("Generated new code %s\n", id)

			// Send back the generated code
			err = conn.WriteMessage(websocket.TextMessage, []byte(id))
			if err != nil {
				log.Println(err)
				break
			}

			log.Printf("Sent new code %s\n", id)

			// Add connection to channel data
			entry, _ := (*ids)[id]
			entry.SenderWSConn = conn
			entry.FileData = FileData{}
			entry.ChunksReceived = 0
			(*ids)[id] = entry

			fmt.Println("SEND")
			fmt.Println(*ids)

			continue
		}

		// Otherwise, message contains data, so pass to receiver
		incomingMessageParts := strings.SplitN(string(incomingMessage), ": ", 2)
		incomingCode := incomingMessageParts[0]
		incomingData := incomingMessageParts[1]

		log.Printf("[%s]: SEND: Received request: %s\n", incomingCode, incomingData)

		// If client code is not in dict (invalid), send rejection
		if entry, ok := (*ids)[incomingCode]; !ok {
			log.Printf("Advisory: Invalid code %s sent\n", incomingCode)

			err = conn.WriteMessage(websocket.TextMessage, []byte("ERROR: The requested code is invalid"))
			if err != nil {
				log.Println(err)
				break
			}

			continue
		} else {
			// If "incomingData" section is "FILE", then this message contains file data
			// in format [CODE]: FILE,filename,fileSizeBytes
			if strings.HasPrefix(incomingData, "FILE") {
				dat := strings.Split(incomingData, ",")

				size, err := strconv.Atoi(dat[2])
				if err != nil {
					log.Printf("[%s]: Error: Could not convert file size to int\n", incomingCode)
					conn.WriteMessage(websocket.TextMessage, []byte("ERROR: Could not convert file size to int"))
				}

				fd := entry.FileData
				fd.FileName = dat[1]
				fd.FileSize = uint32(size)

				entry.FileData = fd
				(*ids)[incomingCode] = entry
			}

			// Check if there is a recipient associated with the code
			if entry.RecipientWSConn == nil {
				log.Printf("[%s]: Error: No recipient associated with code %s\n", incomingCode, incomingCode)
				conn.WriteMessage(websocket.TextMessage, []byte("ERROR: No recipient associated with this code"))

				continue
			}

			entry.RecipientWSConn.WriteMessage(websocket.TextMessage, []byte(incomingData))

			// Increase chunk count
			entry.ChunksReceived = entry.ChunksReceived + 1
			(*ids)[incomingCode] = entry

			err = conn.WriteMessage(websocket.TextMessage, []byte("OK"))
			if err != nil {
				log.Println(err)
				break
			}

			log.Printf("[%s]: SEND: Sent to recipient", incomingCode)
		}
	}
}
