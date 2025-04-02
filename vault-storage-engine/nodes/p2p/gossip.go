package p2p

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
)

type Peer struct {
	NodeID  string `json:"node_id"`
	Address string `json:"address"`
}

var (
	peerList   []Peer
	peerLock   sync.RWMutex
	httpClient = &http.Client{
		Timeout: 5 * time.Second,
	}
)

// RegisterPeer adds a peer to the local list.
func RegisterPeer(p Peer) {
	peerLock.Lock()
	defer peerLock.Unlock()
	peerList = append(peerList, p)
}

// ListPeers returns the current list.
func ListPeers() []Peer {
	peerLock.RLock()
	defer peerLock.RUnlock()
	return peerList
}

// StartGossip periodically pulls peer info from a random peer.
func StartGossip() {
	go func() {
		tracer := otel.Tracer("vault-p2p")
		for {
			time.Sleep(10 * time.Second)
			peerLock.RLock()
			if len(peerList) == 0 {
				peerLock.RUnlock()
				continue
			}
			// Choose a random peer.
			target := peerList[rand.Intn(len(peerList))]
			peerLock.RUnlock()

			// Pull their peer list.
			url := target.Address + "/gossip/peers"
			req, _ := http.NewRequest("GET", url, nil)
			// Propagate tracing context.
			ctx, span := tracer.Start(req.Context(), "GossipPull")
			req = req.WithContext(ctx)
			resp, err := httpClient.Do(req)
			if err != nil {
				span.End()
				log.Printf("Gossip pull failed from %s: %v", target.NodeID, err)
				continue
			}
			var remotePeers []Peer
			json.NewDecoder(resp.Body).Decode(&remotePeers)
			resp.Body.Close()
			span.End()

			peerLock.Lock()
			for _, rp := range remotePeers {
				// Simple de-duplication.
				found := false
				for _, p := range peerList {
					if p.NodeID == rp.NodeID {
						found = true
						break
					}
				}
				if !found {
					peerList = append(peerList, rp)
				}
			}
			peerLock.Unlock()

			data, _ := json.Marshal(ListPeers())
			log.Println("Gossip peers:", string(data))
		}
	}()
}

// HTTP handlers remain the same as before.
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
	json.NewEncoder(w).Encode(ListPeers())
}

// StartGossipServer starts an HTTPS server with mTLS for gossip.
func StartGossipServer(port string, tlsConfig *tls.Config) {
	http.HandleFunc("/gossip/register", GossipRegisterHandler)
	http.HandleFunc("/gossip/peers", GossipListHandler)
	StartGossip()
	log.Println("Gossip server started on port", port)
	log.Fatal(http.ListenAndServeTLS(":"+port, "certs/server.crt", "certs/server.key", nil))
}
