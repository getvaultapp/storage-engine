package utils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"
	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Helper function to convert a slice to a map
func ConvertSliceToMap(slice []string) map[string]string {
	result := make(map[string]string)
	for i, v := range slice {
		key := fmt.Sprintf("key_%d", i) // Use a suitable key generation logic
		result[key] = v
	}
	return result
}

// ConvertViperToConfig converts a viper.Viper instance to a config.Config instance
func ConvertViperToConfig(v *viper.Viper) *config.Config {
	cfg := &config.Config{
		ServerAddress:      v.GetString("server_address"),
		ShardStoreBasePath: v.GetString("shard_store_base_path"),
		EncryptionKeyHex:   v.GetString("encryption_key"),
		Database:           v.GetString("database"),
	}

	// Decode the hex-encoded encryption key
	key, err := hex.DecodeString(cfg.EncryptionKeyHex)
	if err != nil {
		log.Fatalf("failed to decode encryption key: %v", err)
	}

	// Ensure the key length is valid for AES (16, 24, or 32 bytes)
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		log.Fatalf("invalid encryption key size: %d bytes", len(key))
	}

	cfg.EncryptionKey = key

	return cfg
}

// LoadTLSConfig loads server and client certificates for mTLS.
func LoadTLSConfig(certFile, keyFile, caFile string, requireClientCert bool) (*tls.Config, error) {
	// Load server's certificate and private key
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	}
	if requireClientCert {
		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig, nil
}

// InitTracer initializes an OpenTelemetry tracer that writes to stdout.
func InitTracer(serviceName string) func() {
	// For demo purposes, export to stdout.
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Fatalf("failed to initialize exporter: %v", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(nil),
	)
	otel.SetTracerProvider(tp)
	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("Error shutting down tracer provider: %v", err)
		}
	}
}
