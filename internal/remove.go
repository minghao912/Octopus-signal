package internal

import (
	"log"
	"net/http"
)

func Remove(w http.ResponseWriter, req *http.Request, ids *map[string]Channel) {
	// For CORS
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE, OPTIONS")

	if req.Method != "DELETE" && req.Method != "OPTIONS" {
		log.Println("Remove received " + req.Method + " request, but only DELETE and OPTIONS requests allowed")

		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Only DELETE and OPTIONS requests allowed"))
		return
	}

	if req.Method == "OPTIONS" {
		w.WriteHeader((http.StatusAccepted))
		w.Write([]byte("Preflight OK"))
		return
	}

	clientCode := req.URL.Query().Get("code")
	if clientCode == "" {
		log.Println("Remove did not receive a code parameter")

		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Request is missing the code parameter"))
		return
	}

	log.Printf("Received new delete request with sender ID %s\n", string(clientCode))

	// Try seeing if client code is in dict
	_, ok := (*ids)[string(clientCode)]

	// If invalid code send response
	if !ok {
		log.Println("Advisory: Invalid code sent")

		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid code"))
	} else {
		// Close previous connections
		channel, _ := (*ids)[string(clientCode)]
		if channel.SenderWSConn != nil {
			channel.SenderWSConn.Close()
		}
		if channel.RecipientWSConn != nil {
			channel.RecipientWSConn.Close()
		}

		// Delete map entry
		delete(*ids, string(clientCode))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Successfully deleted code " + clientCode))

		log.Printf("[%s]: REMOVE: Success closing sockets and deleting map", string(clientCode))
	}
}
