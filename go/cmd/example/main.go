package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"fmt"

	"github.com/openshift-online/maestro/pkg/api/openapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/watch"

	workv1 "open-cluster-management.io/api/work/v1"
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
		WithWorkClientWatcherStore(maestro.NewRESTFullAPIWatcherStore(maestroAPIClient, sourceID)).
		NewSourceClientHolder(ctx)
	if err != nil {
		log.Fatal(err)
	}

	watcher, err := workClient.ManifestWorks(clusterName).Watch(ctx, metav1.ListOptions{})

	go func() {
		ch := watcher.ResultChan()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-ch:
				if !ok {
					return
				}
				switch event.Type {
				case watch.Added:
					fmt.Printf("the work %v is added", event.Object)
				case watch.Modified:
					fmt.Printf("the work %v is modified", event.Object)
				case watch.Deleted:
					fmt.Printf("the work %v is deleted", event.Object)
				}
			}
		}
	}()

	// use workClient to get/create/patch/list works
	_, err = workClient.ManifestWorks(clusterName).Create(ctx, newManifestWork("test"), metav1.CreateOptions{})
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()
}

func newManifestWork(name string) *workv1.ManifestWork {
	return &workv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{
				Manifests: []workv1.Manifest{
					newManifest("test"),
				},
			},
		},
	}
}

func newManifest(name string) workv1.Manifest {
	obj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"namespace": "test",
				"name":      name,
			},
			"data": "test",
		},
	}
	objectStr, _ := obj.MarshalJSON()
	manifest := workv1.Manifest{}
	manifest.Raw = objectStr
	return manifest
}
