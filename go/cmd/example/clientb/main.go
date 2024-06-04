package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"fmt"

	"github.com/openshift-online/maestro/pkg/api/openapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	"open-cluster-management.io/sdk-go/pkg/cloudevents/generic/options/grpc"
	"open-cluster-management.io/sdk-go/pkg/cloudevents/work"
	"open-cluster-management.io/sdk-go/pkg/cloudevents/work/source/codec"

	"skeeey/go-test/cmd/example/maestro"
	"skeeey/go-test/signal"
)

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

	maestroAPIClient := openapi.NewAPIClient(&openapi.Configuration{
		DefaultHeader: make(map[string]string),
		UserAgent:     "OpenAPI-Generator/1.0.0/go",
		Debug:         false,
		Servers: openapi.ServerConfigurations{
			{
				URL:         "127.0.0.1:8000",
				Description: "current domain",
			},
		},
		OperationServers: map[string]openapi.ServerConfigurations{},
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	})

	grpcOptions := grpc.NewGRPCOptions()
	grpcOptions.URL = "127.0.0.1:8090"

	sourceID := "cs-example"
	clusterName := "cluster1"

	workClient, err := work.NewClientHolderBuilder(grpcOptions).
		WithClientID(fmt.Sprintf("%s-%s", sourceID, rand.String(5))).
		WithSourceID(sourceID).
		WithCodecs(codec.NewManifestBundleCodec()).
		WithWorkClientWatcherStore(maestro.NewRESTFullAPIWatcherStore(ctx, maestroAPIClient, sourceID)).
		WithResyncEnabled(false).
		NewSourceClientHolder(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// use workClient to get/create/patch/list works
	_, err = workClient.ManifestWorks(clusterName).Create(ctx, maestro.NewManifestWork("test"), metav1.CreateOptions{})
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
}
