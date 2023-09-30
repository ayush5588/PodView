package podview

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/ayush5588/PodView/api"
	v1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ErrDeploymentNotFound = errors.New("no deployment found with the given name")
	ErrReplicaSetNotFound = errors.New("no replicaSet found for the given deployment")
	ErrPodsNotFound       = errors.New("no pods found for the given deployment")
	ErrInvalidPodStatus   = errors.New("invalid pod status")
	ErrEmptyString        = errors.New("given string argument cannot be empty")
)

type podViewClient struct {
	client.Client
	deploymentName      string
	deploymentNamespace string
}

// NewViewClient creates a new ViewClient
func NewPodViewClient(c client.Client, deploymentName, deploymentNamespace string) podViewClient {
	return podViewClient{
		Client:              c,
		deploymentName:      deploymentName,
		deploymentNamespace: deploymentNamespace,
	}
}

// ValidateDeployment validates if a deployment exist with the given deployment name
func (p podViewClient) ValidateDeployment() (v1.Deployment, error) {
	depName := p.deploymentName

	// prepare the list based on the deployment name & namespace (if given)
	parsedFieldSelector, _ := fields.ParseSelector(fmt.Sprintf("metadata.name: %s", depName))

	deploymentListOptions := &client.ListOptions{
		FieldSelector: parsedFieldSelector,
	}

	if p.deploymentNamespace != "" {
		deploymentListOptions.Namespace = p.deploymentNamespace
	}

	depList := &v1.DeploymentList{}
	err := p.Client.List(context.Background(), depList, deploymentListOptions)
	if err != nil {
		return v1.Deployment{}, err
	}

	deps := depList.Items

	var wg sync.WaitGroup
	wg.Add(len(deps))

	var dep v1.Deployment

	for _, d := range deps {
		go func(d v1.Deployment) {
			defer wg.Done()
			// if deployment found - return
			if d.Name == depName {
				dep = d
				return
			}
		}(d)
	}

	// wait for all the goroutines to finish
	wg.Wait()

	return dep, nil

}

// GetReplicaSetInfo ...
func (p podViewClient) GetReplicaSetInfo(dep api.Deployment) (v1.ReplicaSet, error) {

	parsedFieldSelector, _ := fields.ParseSelector(fmt.Sprintf("spec.replicas: %d", &dep.Replicas))

	replicaSetListOptions := &client.ListOptions{
		FieldSelector: parsedFieldSelector,
		Namespace:     dep.Namespace,
	}

	rsList := &v1.ReplicaSetList{}

	if err := p.Client.List(context.Background(), rsList, replicaSetListOptions); err != nil {
		return v1.ReplicaSet{}, err
	}

	replicaSets := rsList.Items

	var wg sync.WaitGroup
	wg.Add(len(replicaSets))

	var rs v1.ReplicaSet

	for _, r := range replicaSets {
		go func(r v1.ReplicaSet) {
			defer wg.Done()
			ownersRef := r.OwnerReferences
			for _, o := range ownersRef {
				if o.Kind == "Deployment" && o.Name == dep.Name {
					rs = r
					return
				}
			}

		}(r)
	}

	wg.Wait()

	return rs, nil

}

func (p podViewClient) getPods(rs v1.ReplicaSet) (api.PodList, error) {
	podListOptions := client.ListOptions{
		Namespace: rs.Namespace,
	}

	podList := &coreV1.PodList{}

	if err := p.Client.List(context.Background(), podList, &podListOptions); err != nil {
		return api.PodList{}, err
	}

	var depPodList api.PodList
	depPodList.Pods = make([]api.Pod, 0)

	podListChan := make(chan api.Pod, 1)

	pods := podList.Items

	var wg sync.WaitGroup
	wg.Add(len(pods))

	for _, pod := range pods {
		go func(pod coreV1.Pod) {
			defer wg.Done()
			for _, o := range pod.OwnerReferences {
				if o.Kind == "ReplicaSet" && o.Name == rs.Name {
					podObj := api.Pod{
						Name:   pod.Name,
						Status: string(pod.Status.Phase),
					}
					if podObj.Status == "Failed" {
						podObj.Message = pod.Status.Message
					}
					podListChan <- podObj

				}
			}
		}(pod)
	}

	go func() {
		wg.Wait()
		close(podListChan)
	}()

	for pod := range podListChan {
		depPodList.Pods = append(depPodList.Pods, pod)
	}

	return depPodList, nil

}

// GetPods returns list of pods belonging to the given deployment
func (p podViewClient) GetPods() (api.PodList, error) {
	// First validate if the given deployment exist
	dep, err := p.ValidateDeployment()
	if err != nil {
		return api.PodList{}, err
	}

	if reflect.DeepEqual(dep, v1.Deployment{}) {
		return api.PodList{}, ErrDeploymentNotFound
	}

	// Get the replicaSet based on the ownerRef & the replica.
	// The active replicaSet will have replica == deployment.spec.replica and the non-active will have replica = 0

	ownerDeployment := api.Deployment{
		Name:     dep.Name,
		Replicas: *dep.Spec.Replicas,
	}

	if dep.Namespace != "" {
		ownerDeployment.Namespace = dep.Namespace
	} else {
		ownerDeployment.Namespace = "default"
	}

	rs, err := p.GetReplicaSetInfo(ownerDeployment)
	if err != nil {
		return api.PodList{}, err
	}

	if reflect.DeepEqual(rs, v1.ReplicaSet{}) {
		return api.PodList{}, ErrReplicaSetNotFound
	}

	pods, err := p.getPods(rs)
	if err != nil {
		return api.PodList{}, err
	}

	if reflect.DeepEqual(pods, api.PodList{}) {
		return api.PodList{}, ErrPodsNotFound
	}

	return pods, nil
}

// GetPodsWithStatus takes status of the pod as argument and returns list of pods belonging to the deployment with that status
func (p podViewClient) GetPodsWithStatus(status string) (api.PodList, error) {
	if status != "Running" && status != "Failed" && status != "Pending" {
		return api.PodList{}, ErrInvalidPodStatus
	}

	allPods, err := p.GetPods()
	if err != nil {
		return api.PodList{}, err
	}

	var wg sync.WaitGroup

	var pods api.PodList
	pods.Pods = make([]api.Pod, 0)

	podChan := make(chan api.Pod, 1)

	for _, pod := range allPods.Pods {
		wg.Add(1)
		go func(pod api.Pod) {
			defer wg.Done()
			if pod.Status == status {
				// Using channel to avoid data race condition
				podChan <- pod
			}
		}(pod)
	}

	go func() {
		wg.Wait()
		close(podChan)
	}()

	for pod := range podChan {
		pods.Pods = append(pods.Pods, pod)
	}

	return pods, nil
}
