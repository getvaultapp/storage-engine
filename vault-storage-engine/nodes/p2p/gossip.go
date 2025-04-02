// p2p/gossip.go
package p2p

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type Peer struct {
	NodeID  string `json:"node_id"`
	Address string `json:"address"`
}

var peerList []Peer

// RegisterPeer registers a peer with the local gossip node.
func RegisterPeer(p Peer) {
	peerList = append(peerList, p)
}

// ListPeers returns the current list of peers.
func ListPeers() []Peer {
	return peerList
}

// StartGossip periodically logs the list of peers.
func StartGossip() {
	go func() {
		for {
			time.Sleep(10 * time.Second)
			data, _ := json.Marshal(peerList)
			log.Println("Gossip peers:", string(data))
		}
	}()
}

// HTTP handlers for peer gossip.
func GossipRegisterHandler(w http.ResponseWriter, r *http.Request) {
	var p Peer
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	RegisterPeer(p)
	w.WriteHeader(http.StatusOK)
}

func GossipListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(peerList)
}

// StartGossipServer starts an HTTP server for gossip endpoints.
func StartGossipServer(port string) {
	http.HandleFunc("/gossip/register", GossipRegisterHandler)
	http.HandleFunc("/gossip/peers", GossipListHandler)
	StartGossip()
	log.Println("Gossip server started on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
