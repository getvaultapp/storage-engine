package main

// package p2p

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"syscall"
	"time"

	"go.opentelemetry.io/otel"
)

var (
	peerList   []Peer
	peerLock   sync.RWMutex
	httpClient = &http.Client{Timeout: 5 * time.Second}
)

// Enhanced peer information
type Peer struct {
	NodeID         string            `json:"node_id"`
	Address        string            `json:"address"`
	NodeType       string            `json:"node_type"`
	Capabilities   map[string]string `json:"capabilities"`
	LastHeartbeat  time.Time         `json:"last_heartbeat"`
	AvailableSpace int64             `json:"available_space,omitempty"` // Only for storage nodes
}

// Enhanced gossip mechanism
func StartHealthCheck(w http.ResponseWriter, r *http.Request) {
	go func() {
		for {
			time.Sleep(30 * time.Second)

			// Report health to discovery service
			nodeInfo := map[string]interface{}{
				"node_id":   os.Getenv("NODE_ID"),
				"node_type": os.Getenv("NODE_TYPE"),
				"address":   fmt.Sprintf("https://localhost:%s", os.Getenv("NODE_PORT")),
				"time":      time.Now().Format(time.RFC3339),
			}

			// If we're a storage node, include available space
			if os.Getenv("NODE_TYPE") == "storage" {
				var stat syscall.Statfs_t
				basePath := os.Getenv("SHARD_STORE_BASE_PATH")
				if basePath == "" {
					basePath = "./data"
				}

				if err := syscall.Statfs(basePath, &stat); err == nil {
					available := stat.Bavail * uint64(stat.Bsize)
					nodeInfo["available_space"] = available
				}
			}

			// Send to discovery service
			jsonData, _ := json.Marshal(nodeInfo)
			http.Post("http://localhost:8000/register", "application/json", bytes.NewReader(jsonData))
		}
	}()
}

func RegisterPeer(p Peer) {
	peerLock.Lock()
	defer peerLock.Unlock()
	for _, existing := range peerList {
		if existing.NodeID == p.NodeID {
			return
		}
	}
	peerList = append(peerList, p)
}

func ListPeers() []Peer {
	peerLock.RLock()
	defer peerLock.RUnlock()
	return peerList
}

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
			target := peerList[rand.Intn(len(peerList))]
			peerLock.RUnlock()

			url := target.Address + "/gossip/peers"
			req, _ := http.NewRequest("GET", url, nil)
			ctx, span := tracer.Start(req.Context(), "GossipPull")
			req = req.WithContext(ctx)
			resp, err := httpClient.Do(req)
			span.End()
			if err != nil {
				log.Printf("Gossip pull failed from %s: %v", target.NodeID, err)
				continue
			}
			var remotePeers []Peer
			json.NewDecoder(resp.Body).Decode(&remotePeers)
			resp.Body.Close()

			peerLock.Lock()
			for _, rp := range remotePeers {
				found := false
				for _, p := range peerList {
					if p.NodeID == rp.NodeID {
						found = true
						break
					}
				}
				if !found && len(peerList) < 50 {
					peerList = append(peerList, rp)
				}
			}
			peerLock.Unlock()

			data, _ := json.Marshal(ListPeers())
			log.Println("Gossip peers:", string(data))
		}
	}()
}

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

func StartGossipServer(port string, tlsConfig *tls.Config) {
	http.HandleFunc("/gossip/health", StartHealthCheck)
	http.HandleFunc("/gossip/register", GossipRegisterHandler)
	http.HandleFunc("/gossip/peers", GossipListHandler)
	StartGossip()
	log.Println("Gossip server started on port", port)
	log.Fatal(http.ListenAndServeTLS(":"+port, "/home/tnxl/storage-engine/vault-storage-engine/nodes/certs/server.crt", "/home/tnxl/storage-engine/vault-storage-engine/nodes/certs/server.key", nil))
}

func main() {
	// Load TLS Certificates
	certPath := "/home/tnxl/storage-engine/vault-storage-engine/nodes/certs/"
	cert, err := tls.LoadX509KeyPair(certPath+"server.crt", certPath+"server.key")
	if err != nil {
		log.Fatalf("Failed to load server certificates: %v", err)
	}

	caCert, err := os.ReadFile(certPath + "ca.crt")
	if err != nil {
		log.Fatalf("Failed to load CA certificate: %v", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}

	// Get Node ID and Port
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		log.Fatal("NODE_ID environment variable not set")
	}
	port := os.Getenv("NODE_PORT")
	if port == "" {
		port = "7000" // Default port
	}

	// Start Gossip Server
	fmt.Printf("Starting Gossip node %s on port %s...\n", nodeID, port)
	StartGossipServer(port, tlsConfig)
}
