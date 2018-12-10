package quic

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"time"
)

// Config is a quic specyfic configuration
type Config struct {
	// tlsCfg server tls config
	tlsCfg *tls.Config
	// dialer creates quic session between two peers
	dialer dialer
	//handshakeTimeout quic handshakeTimeout
	handshakeTimeout time.Duration
}

const defaultHandshakeTimeout = 2000 * time.Millisecond

// NewInsecureTestConfig creates config for testing prupose,
// node with this quic configuration won't verify server's
// certificate chain and host name
func NewInsecureTestConfig() Config {
	return Config{
		tlsCfg:           generateTestTLSConfig(),
		dialer:           newInsecureQuicDialer(defaultHandshakeTimeout),
		handshakeTimeout: defaultHandshakeTimeout,
	}
}

// NewConfig creates quic configuration
func NewConfig(pathToTLSCert string, pathToTLSKey string, handshakeTimeout time.Duration, serverName string) Config {
	return Config{
		tlsCfg:           generateTLSConfig(pathToTLSCert, pathToTLSKey),
		dialer:           newQuicDialer(handshakeTimeout, serverName),
		handshakeTimeout: handshakeTimeout,
	}
}

func generateTestTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)} //, DNSNames: []string{"localhost"}}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{Certificates: []tls.Certificate{tlsCert}}
}

func generateTLSConfig(pathToCert string, pathToKey string) *tls.Config {
	tlsCert, err := tls.LoadX509KeyPair(pathToCert, pathToKey)
	if err != nil {
		panic(err)
	}
	return &tls.Config{Certificates: []tls.Certificate{tlsCert}}
}
