package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"time"

	"fmt"

	"github.com/openshift-online/maestro/pkg/api/openapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

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
				URL:         "https://127.0.0.1:30080",
				Description: "current domain",
			},
		},
		OperationServers: map[string]openapi.ServerConfigurations{},
		HTTPClient: &http.Client{
			Transport: &http.Transport{TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			}},
			Timeout: 10 * time.Second,
		},
	})

	grpcOptions := grpc.NewGRPCOptions()
	grpcOptions.URL = "127.0.0.1:8090"

	sourceID := "cs-example"
	clusterName := "15efbd18-2848-430f-9a70-5c0408ea262e"

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

	// use workClient to list/get/create/patch/delete works
	workName := "client-b" + rand.String(5)
	_, err = workClient.ManifestWorks(clusterName).Create(ctx, maestro.NewManifestWork(workName), metav1.CreateOptions{})
	if err != nil {
		log.Fatal(err)
	}

	work, err := workClient.ManifestWorks(clusterName).Get(ctx, workName, metav1.GetOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("the work %s (uid=%s) is created\n", workName, work.UID)

	// util the work has status
	for {
		work, err := workClient.ManifestWorks(clusterName).Get(ctx, workName, metav1.GetOptions{})
		if err != nil {
			log.Fatal(err)
		}

		if len(work.Status.Conditions) != 0 {
			break
		}

		<-time.After(time.Second)
	}

	newWork := work.DeepCopy()
	newWork.Spec.Workload.Manifests = []workv1.Manifest{
		maestro.NewManifest(workName),
	}
	patchData, err := maestro.ToWorkPatch(work, newWork)
	if err != nil {
		log.Fatal(err)
	}
	_, err = workClient.ManifestWorks(clusterName).Patch(ctx, workName, types.MergePatchType, patchData, metav1.PatchOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("the work %s (uid=%s) is updated\n", workName, work.UID)

	err = workClient.ManifestWorks(clusterName).Delete(ctx, workName, metav1.DeleteOptions{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("the work %s (uid=%s) is deleting\n", workName, work.UID)

	<-ctx.Done()
}
