package maestro

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/openshift-online/maestro/pkg/api/openapi"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	workv1 "open-cluster-management.io/api/work/v1"

	"open-cluster-management.io/sdk-go/pkg/cloudevents/work/common"
	"open-cluster-management.io/sdk-go/pkg/cloudevents/work/utils"
)

// This will be provided in maestro repo as a lib
type RESTFullAPIWatcherStore struct {
	sync.RWMutex

	result chan watch.Event
	done   chan struct{}

	sourceID  string
	apiClient *openapi.APIClient
}

func NewRESTFullAPIWatcherStore(apiClient *openapi.APIClient, sourceID string) *RESTFullAPIWatcherStore {
	return &RESTFullAPIWatcherStore{
		result:    make(chan watch.Event),
		done:      make(chan struct{}),
		sourceID:  sourceID,
		apiClient: apiClient,
	}
}

// ResultChan implements watch interface.
func (m *RESTFullAPIWatcherStore) ResultChan() <-chan watch.Event {
	return m.result
}

// Stop implements watch interface.
func (m *RESTFullAPIWatcherStore) Stop() {
	// Call Close() exactly once by locking and setting a flag.
	m.Lock()
	defer m.Unlock()
	// closing a closed channel always panics, therefore check before closing
	select {
	case <-m.done:
		close(m.result)
	default:
		close(m.done)
	}
}

func (m *RESTFullAPIWatcherStore) Get(namespace, name string) (*workv1.ManifestWork, error) {
	id := utils.UID(m.sourceID, namespace, name)
	rb, resp, err := m.apiClient.DefaultApi.ApiMaestroV1ResourceBundlesIdGet(context.Background(), id).Execute()
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return nil, errors.NewNotFound(common.ManifestWorkGR, name)
		}

		return nil, err
	}

	return toManifestWork(rb)
}

// List the works from the cache with the list options
func (m *RESTFullAPIWatcherStore) List(opts metav1.ListOptions) ([]*workv1.ManifestWork, error) {
	// TODO add cluster name as filter
	clusterName := ""
	if len(opts.FieldSelector) != 0 {
		fieldSelector, err := fields.ParseSelector(opts.FieldSelector)
		if err != nil {
			return nil, err
		}

		for _, req := range fieldSelector.Requirements() {
			if req.Field == "metadata.namespace" {
				clusterName = req.Value
				break
			}
		}
	}
	apiRequest := m.apiClient.DefaultApi.ApiMaestroV1ResourceBundlesGet(context.Background())
	if clusterName != "" {
		apiRequest = apiRequest.Search(fmt.Sprintf("name = '%s'", clusterName))
	}

	rbs, _, err := apiRequest.Execute()
	if err != nil {
		return nil, err
	}

	works := []*workv1.ManifestWork{}
	for _, rb := range rbs.Items {
		work, err := toManifestWork(&rb)
		if err != nil {
			return nil, err
		}

		works = append(works, work)

	}
	return works, nil
}

func (m *RESTFullAPIWatcherStore) ListAll() ([]*workv1.ManifestWork, error) {
	works := []*workv1.ManifestWork{}
	rbs, _, err := m.apiClient.DefaultApi.ApiMaestroV1ResourceBundlesGet(context.Background()).Execute()
	if err != nil {
		return works, err
	}

	for _, rb := range rbs.Items {
		work, err := toManifestWork(&rb)
		if err != nil {
			return nil, err
		}

		works = append(works, work)

	}
	return works, nil
}

func (m *RESTFullAPIWatcherStore) Add(work *workv1.ManifestWork) error {
	m.result <- watch.Event{Type: watch.Added, Object: work}
	return nil
}

func (m *RESTFullAPIWatcherStore) Update(work *workv1.ManifestWork) error {
	m.result <- watch.Event{Type: watch.Modified, Object: work}
	return nil
}

func (m *RESTFullAPIWatcherStore) Delete(work *workv1.ManifestWork) error {
	m.result <- watch.Event{Type: watch.Deleted, Object: work}
	return nil
}

func (m *RESTFullAPIWatcherStore) HasInitiated() bool {
	return false
}

func toManifestWork(rb *openapi.ResourceBundle) (*workv1.ManifestWork, error) {
	manifests := []workv1.Manifest{}

	for _, manifest := range rb.Manifests {
		obj := unstructured.Unstructured{Object: manifest}
		raw, err := obj.MarshalJSON()
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, workv1.Manifest{
			RawExtension: runtime.RawExtension{Raw: raw},
		})
	}

	// TODO set deleteOption and ManifestConfigs

	return &workv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{
			UID:             types.UID(*rb.Id),
			Name:            *rb.Name,
			Namespace:       *rb.ConsumerName,
			ResourceVersion: "0",
		},
		Spec: workv1.ManifestWorkSpec{
			Workload: workv1.ManifestsTemplate{
				Manifests: manifests,
			},
		},
	}, nil
}
