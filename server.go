package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"

	internal "github.com/minghao912/octopus-signal/internal"
)

func h(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "hello\n")
}

func diagnostic(w http.ResponseWriter, req *http.Request, ids *map[string]internal.Channel) {
	d, err := json.Marshal(ids)
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

	// "Globals"
	// Create gorilla websocket upgrader
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	var ids = make(map[string]internal.Channel)

	//Handlers for API routes
	http.HandleFunc("/h", h)
	http.HandleFunc("/diagnostic", func(w http.ResponseWriter, req *http.Request) { diagnostic(w, req, &ids) })
	http.HandleFunc("/send", func(w http.ResponseWriter, req *http.Request) { internal.Send(w, req, upgrader, &ids) })
	http.HandleFunc("/receive", func(w http.ResponseWriter, req *http.Request) { internal.Receive(w, req, upgrader, &ids) })
	http.HandleFunc("/remove", func(w http.ResponseWriter, req *http.Request) { internal.Remove(w, req, &ids) })

	// Server config
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	log.Println("Server listening on port " + httpPort)
	http.ListenAndServe(":"+httpPort, nil)
}
