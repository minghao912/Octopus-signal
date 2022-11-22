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
	upgrader.CheckOrigin = func(req *http.Request) bool { return true }	// Allow all origins
	conn, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Println(err)
		fmt.Fprintf(w, "ERROR: Could not upgrade to web socket\n"+err.Error())
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

		log.Printf("Sent new code %s to id %s\n", id, string(senderID))
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

	defer conn.Close()

	for {
		// Read the client's code
		_, clientCode, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}

		log.Printf("Received new receive request with sender ID %s\n", string(clientCode))

		// Try seeing if client code is in dict
		senderID, ok := ids[string(clientCode)]

		// If client code is not in dict (invalid), send rejection
		if !ok {
			log.Println("Advisory: Invalid code sent")

			err = conn.WriteMessage(websocket.TextMessage, []byte("ERROR: The requested code is invalid"))
			if err != nil {
				log.Println(err)
				break
			}
		} else { // Else, send back the sender's ID
			log.Printf("Corresponding sender ID found: %s", senderID)

			err = conn.WriteMessage(websocket.TextMessage, []byte(senderID))
			if err != nil {
				log.Println(err)
				break
			}
		}
	}
}

func remove(w http.ResponseWriter, req *http.Request) {
	// Upgrade http request to web socket
	upgrader.CheckOrigin = func(req *http.Request) bool { return true } // Allow all origins
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

		log.Printf("Received new delete request with sender ID %s\n", string(clientCode))

		// Try seeing if client code is in dict
		_, ok := ids[string(clientCode)]

		// If invalid code send response
		if !ok {
			log.Println("Advisory: Invalid code sent")

			err = conn.WriteMessage(websocket.TextMessage, []byte("ERROR: The requested code is invalid"))
			if err != nil {
				log.Println(err)
				break
			}
		} else { // Otherwise delete map entry
			delete(ids, string(clientCode))

			err = conn.WriteMessage(websocket.TextMessage, []byte("Success: Deleted "+string(clientCode)))
			if err != nil {
				log.Println(err)
				break
			}
		}
	}
}

func main() {
	log.Println("Server started")

	//Handlers for API routes
	http.HandleFunc("/h", h)
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
