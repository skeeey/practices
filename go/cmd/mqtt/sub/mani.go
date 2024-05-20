package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"skeeey/go-test/util"
	"time"

	"github.com/eclipse/paho.golang/paho"
)

const brokerHost = "wl-mqtt-test.eastus-1.ts.eventgrid.azure.net:8883"

func main() {
	certPool, err := x509.SystemCertPool()
	if err != nil {
		log.Fatalf("Failed to connect to %s : %v", brokerHost, err)
	}

	tlsConfig := &tls.Config{
		RootCAs: certPool,
		GetClientCertificate: func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return util.CachingCertificateLoader("client1-authnID.pem", "client1-authnID.key")()
		},
	}

	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 30 * time.Second}, "tcp", brokerHost, tlsConfig)
	if err != nil {
		log.Fatalf("Failed to connect to %s : %v", brokerHost, err)
	}

	c := paho.NewClient(paho.ClientConfig{
		OnPublishReceived: []func(paho.PublishReceived) (bool, error){ // Noop handler
			func(pr paho.PublishReceived) (bool, error) {
				log.Printf("received %v", pr.Packet)
				return true, nil
			}},
		Conn: conn,
	})

	cp := &paho.Connect{
		KeepAlive:  30,
		ClientID:   "client1-authnID",
		CleanStart: true,
	}

	ca, err := c.Connect(context.Background(), cp)
	if err != nil {
		log.Fatalf("Failed to connect to %s : %v", brokerHost, err)
	}
	if ca.ReasonCode != 0 {
		log.Fatalf("Failed to connect to %s : %d - %s", brokerHost, ca.ReasonCode, ca.Properties.ReasonString)
	}

	fmt.Printf("Connected to %s\n", brokerHost)

	time.Sleep(10 * time.Second)

	//
}
