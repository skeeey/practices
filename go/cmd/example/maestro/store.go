package maestro

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/openshift-online/maestro/pkg/api/openapi"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	workv1 "open-cluster-management.io/api/work/v1"

	cetypes "open-cluster-management.io/sdk-go/pkg/cloudevents/generic/types"
	"open-cluster-management.io/sdk-go/pkg/cloudevents/work/common"
	"open-cluster-management.io/sdk-go/pkg/cloudevents/work/payload"
	"open-cluster-management.io/sdk-go/pkg/cloudevents/work/utils"
)

type watchEvent struct {
	Key types.UID
}

// This will be provided in maestro repo as a lib
type RESTFullAPIWatcherStore struct {
	sync.RWMutex

	result chan watch.Event
	done   chan struct{}

	sourceID  string
	apiClient *openapi.APIClient

	// A queue to save the work client send events, if run a client without a watcher,
	// it will block the client, this queue helps to resolve this blocking.
	// Only save the latest event for a work.
	eventQueue cache.Queue
}

func NewRESTFullAPIWatcherStore(ctx context.Context, apiClient *openapi.APIClient, sourceID string) *RESTFullAPIWatcherStore {
	s := &RESTFullAPIWatcherStore{
		result: make(chan watch.Event),
		done:   make(chan struct{}),

		sourceID:  sourceID,
		apiClient: apiClient,

		eventQueue: cache.NewFIFO(func(obj interface{}) (string, error) {
			evt, ok := obj.(*watchEvent)
			if !ok {
				return "", fmt.Errorf("unknown object type %T", obj)
			}

			return string(evt.Key), nil
		}),
	}

	go wait.Until(s.processLoop, time.Second, ctx.Done())
	return s
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

func (m *RESTFullAPIWatcherStore) HandleReceivedWork(action cetypes.ResourceAction, work *workv1.ManifestWork) error {
	switch action {
	case cetypes.StatusModified:
		m.eventQueue.Add(&watchEvent{Key: work.UID})
	default:
		return fmt.Errorf("unsupported resource action %s", action)
	}
	return nil
}

func (m *RESTFullAPIWatcherStore) Get(namespace, name string) (*workv1.ManifestWork, error) {
	id := utils.UID(m.sourceID, namespace, name)
	return m.getWorkByID(id)
}

// List the works from the cache with the list options
func (m *RESTFullAPIWatcherStore) List(opts metav1.ListOptions) ([]*workv1.ManifestWork, error) {
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
		apiRequest = apiRequest.Search(fmt.Sprintf("consumer_name = '%s'", clusterName))
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
	return nil
}

func (m *RESTFullAPIWatcherStore) Update(work *workv1.ManifestWork) error {
	return nil
}

func (m *RESTFullAPIWatcherStore) Delete(work *workv1.ManifestWork) error {
	return nil
}

func (m *RESTFullAPIWatcherStore) HasInitiated() bool {
	return true
}

func (m *RESTFullAPIWatcherStore) getWorkByID(id string) (*workv1.ManifestWork, error) {
	rb, resp, err := m.apiClient.DefaultApi.ApiMaestroV1ResourceBundlesIdGet(context.Background(), id).Execute()
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, errors.NewNotFound(common.ManifestWorkGR, id)
		}

		return nil, err
	}

	return toManifestWork(rb)
}

// processLoop drains the work event queue and send the event to the watch channel.
func (m *RESTFullAPIWatcherStore) processLoop() {
	for {
		// this will be blocked until the event queue has events
		obj, err := m.eventQueue.Pop(func(interface{}, bool) error {
			// do nothing
			return nil
		})
		if err != nil {
			if err == cache.ErrFIFOClosed {
				return
			}

			// this is the safe way to re-enqueue.
			if err := m.eventQueue.AddIfNotPresent(obj); err != nil {
				return
			}
		}

		evt, ok := obj.(*watchEvent)
		if !ok {
			return
		}

		// TODO returning a work with spec instead of this operation
		work, err := m.getWorkByID(string(evt.Key))

		if errors.IsNotFound(err) {
			// TODO also return the work's name and namespace
			// the work has been deleted, return a work only with its uid
			// this will be blocked until this event is consumed
			m.result <- watch.Event{
				Type: watch.Deleted,
				Object: &workv1.ManifestWork{
					ObjectMeta: metav1.ObjectMeta{
						UID: evt.Key,
					},
				},
			}
			return
		}

		if err != nil {
			fmt.Printf("failed to get the work %s, %v\n", evt.Key, err)
			return
		}
		// this will be blocked until this event is consumed
		m.result <- watch.Event{Type: watch.Modified, Object: work}
	}
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
	work := &workv1.ManifestWork{
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
	}

	if rb.Status != nil {
		status, err := json.Marshal(rb.Status)
		if err != nil {
			return nil, err
		}
		manifestStatus := &payload.ManifestBundleStatus{}
		if err := json.Unmarshal(status, manifestStatus); err != nil {
			return nil, err
		}

		work.Status = workv1.ManifestWorkStatus{
			Conditions: manifestStatus.Conditions,
			ResourceStatus: workv1.ManifestResourceStatus{
				Manifests: manifestStatus.ResourceStatus,
			},
		}
	}

	return work, nil
}
