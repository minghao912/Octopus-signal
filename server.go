package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
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

// Global dict of codes and IDs
var ids = make(map[string]string)

func send(w http.ResponseWriter, req *http.Request) {
	// Upgrade http request to web socket
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Println(err)
		return
	}

	defer conn.Close()

	for {
		// Read the client's message (sender ID)
		_, senderID, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}

		log.Printf("Received new send request with sender ID %s\n", string(senderID))

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

		// Store sender ID in map along with generated code
		ids[id] = string(senderID)

		// Send back the generated codes
		err = conn.WriteMessage(websocket.TextMessage, []byte(id))
		if err != nil {
			log.Println(err)
			break
		}
	}
}

func receive(w http.ResponseWriter, req *http.Request) {
	// Upgrade http request to web socket
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Println(err)
		return
	}

	defer conn.Close()

	for {
		// Read the client's code
		_, clientCode, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}

		// Try seeing if client code is in dict
		senderID, ok := ids[string(clientCode)]

		// If client code is not in dict (invalid), send rejection
		if !ok {
			err = conn.WriteMessage(websocket.TextMessage, []byte("Error: The requested code is invalid"))
			if err != nil {
				log.Println(err)
				break
			}
		} else { // Else, send back the sender's ID
			err = conn.WriteMessage(websocket.TextMessage, []byte(senderID))
			if err != nil {
				log.Println(err)
				break
			}
		}
	}
}

func main() {
	//Handlers for API routes
	http.HandleFunc("/h", h)
	http.HandleFunc("/send", send)
	http.HandleFunc("/receive", receive)

	// Server config
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	http.ListenAndServe(":"+httpPort, nil)
}
