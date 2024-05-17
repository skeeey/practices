package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"

	"skeeey/go-test/signal"
	"skeeey/go-test/util"

	mochimqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
)

const mqttBrokerHost = "127.0.0.1:1883"
const mqttTLSBrokerHost = "127.0.0.1:8883"

func main() {
	shutdownCtx, cancel := context.WithCancel(context.TODO())
	shutdownHandler := signal.SetupSignalHandler()
	go func() {
		defer cancel()
		<-shutdownHandler
		fmt.Println("Received SIGTERM or SIGINT signal, shutting down controller.")
	}()

	ctx, terminate := context.WithCancel(shutdownCtx)
	defer terminate()

	certPairs, err := prepareCerts()
	if err != nil {
		log.Fatal(err)
	}

	certPool, err := util.AppendToCertPool(certPairs.CA)
	if err != nil {
		log.Fatal(err)
	}

	serverCert, err := tls.X509KeyPair(certPairs.ServerCert, certPairs.ServerKey)
	if err != nil {
		log.Fatal(err)
	}

	// start a MQTT broker
	mqttBroker := mochimqtt.New(&mochimqtt.Options{})
	// allow all connections.
	if err := mqttBroker.AddHook(new(auth.AllowHook), nil); err != nil {
		log.Fatal(err)
	}

	if err := mqttBroker.AddListener(listeners.NewTCP("mqtt-test-broker", mqttBrokerHost, nil)); err != nil {
		log.Fatal(err)
	}
	if err := mqttBroker.AddListener(listeners.NewTCP("mqtt-tls-test-broker", mqttTLSBrokerHost, &listeners.Config{
		TLSConfig: &tls.Config{
			ClientCAs:    certPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			Certificates: []tls.Certificate{serverCert},
		},
	})); err != nil {
		log.Fatal(err)
	}

	go func() {
		err := mqttBroker.Serve()
		if err != nil {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
}

func prepareCerts() (*util.CertPairs, error) {
	certPairs, err := util.NewCertPairs()
	if err != nil {
		return nil, err
	}

	return certPairs, nil
}
