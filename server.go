package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// Create gorilla websocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func h(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "hello\n")
}

// Map each code to a "channel"
// Contains sender's websocket connection, recipient's websocket connection
// as well as sender's signalling data and recipient's signalling data
type channel struct {
	SenderWSConn    *websocket.Conn `json:"senderWSConn"`
	RecipientWSConn *websocket.Conn `json:"recipientWSConn"`
	FileData        fileData        `json:"fileData"`
	ChunksReceived  uint32          `json:"chunksReceived"`
}

type fileData struct {
	FileName string `json:"fileName"`
	FileSize uint32 `json:"fileSize"`
}

var ids = make(map[string]channel)

func send(w http.ResponseWriter, req *http.Request) {
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
			for _, exist := ids[id]; exist; _, exist = ids[id] { // Keep generating new id until it is unique
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
			entry, _ := ids[id]
			entry.SenderWSConn = conn
			entry.FileData = fileData{}
			entry.ChunksReceived = 0
			ids[id] = entry

			continue
		}

		// Otherwise, message contains data, so pass to receiver
		incomingMessageParts := strings.SplitN(string(incomingMessage), ": ", 2)
		incomingCode := incomingMessageParts[0]
		incomingData := incomingMessageParts[1]

		log.Printf("[%s]: SEND: Received request: %s\n", incomingCode, incomingData)

		// If client code is not in dict (invalid), send rejection
		if entry, ok := ids[incomingCode]; !ok {
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
				ids[incomingCode] = entry
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
			ids[incomingCode] = entry

			err = conn.WriteMessage(websocket.TextMessage, []byte("OK"))
			if err != nil {
				log.Println(err)
				break
			}

			log.Printf("[%s]: SEND: Sent to recipient", incomingCode)
		}
	}
}

func receive(w http.ResponseWriter, req *http.Request) {
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

		// Try seeing if client code is in dict
		_, ok := ids[incomingCode]

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

			channel, _ := ids[incomingCode]

			if channel.SenderWSConn == nil {
				log.Printf("[%s]: Error: No sender associated with code %s\n", incomingCode, incomingCode)
				conn.WriteMessage(websocket.TextMessage, []byte("ERROR: No sender associated with this code"))

				continue
			}

			channel.RecipientWSConn = conn
			ids[incomingCode] = channel

			conn.WriteMessage(websocket.TextMessage, []byte("OK"))
			channel.SenderWSConn.WriteMessage(websocket.TextMessage, []byte("Connection received"))
			log.Printf("[%s]: RECEIVE: Recipient activated", incomingCode)
		}
	}
}

func remove(w http.ResponseWriter, req *http.Request) {
	if req.Method != "DELETE" {
		log.Println("Remove received " + req.Method + " request, but only DELETE requests allowed")

		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Only DELETE requests allowed"))
		return
	}

	// For CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientCode := req.URL.Query().Get("code")
	if clientCode == "" {
		log.Println("Remove did not receive a code parameter")

		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Request is missing the code parameter"))
		return
	}

	log.Printf("Received new delete request with sender ID %s\n", string(clientCode))

	// Try seeing if client code is in dict
	_, ok := ids[string(clientCode)]

	// If invalid code send response
	if !ok {
		log.Println("Advisory: Invalid code sent")

		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid code"))
	} else {
		// Close previous connections
		channel, _ := ids[string(clientCode)]
		if channel.SenderWSConn != nil {
			channel.SenderWSConn.Close()
		}
		if channel.RecipientWSConn != nil {
			channel.RecipientWSConn.Close()
		}

		// Delete map entry
		delete(ids, string(clientCode))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Successfully deleted code " + clientCode))

		log.Printf("[%s]: REMOVE: Success closing sockets and deleting map", string(clientCode))
	}
}

func diagnostic(w http.ResponseWriter, req *http.Request) {
	d, err := json.Marshal(&ids)
	if err != nil {
		fmt.Printf("Oh no!\n")
	}

	// Send data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(d)
}

func main() {
	log.Println("Server started")

	//Handlers for API routes
	http.HandleFunc("/h", h)
	http.HandleFunc("/diagnostic", diagnostic)
	http.HandleFunc("/send", send)
	http.HandleFunc("/receive", receive)
	http.HandleFunc("/remove", remove)

	// Server config
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	log.Println("Server listening on port " + httpPort)
	http.ListenAndServe(":"+httpPort, nil)
}
