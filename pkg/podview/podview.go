package podview

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/ayush5588/PodView/api"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// ErrDeploymentNotFound ...
	ErrDeploymentNotFound = errors.New("No deployment found with the given name")
)

type podViewClient struct {
	client.Client
	deploymentName      string
	deploymentNamespace string
}

// NewPodViewClient creates a new ViewClient
func NewPodViewClient(c client.Client, deploymentName, deploymentNamespace string) podViewClient {
	return podViewClient{
		Client:              c,
		deploymentName:      deploymentName,
		deploymentNamespace: deploymentNamespace,
	}
}

func (p podViewClient) ValidateDeployment() (v1.Deployment, error) {
	depName := p.deploymentName

	parsedFieldSelector, _ := fields.ParseSelector(fmt.Sprintf("metadata.name: %s", depName))

	listOptions := &client.ListOptions{
		FieldSelector: parsedFieldSelector,
	}

	if p.deploymentNamespace != "" {
		listOptions.Namespace = p.deploymentNamespace
	}

	depList := &v1.DeploymentList{}
	err := p.Client.List(context.Background(), depList, listOptions)
	if err != nil {
		return v1.Deployment{}, err
	}

	deps := depList.Items

	var wg sync.WaitGroup
	wg.Add(len(deps))

	var depExist bool
	var dep v1.Deployment

	for _, d := range deps {
		go func(d v1.Deployment) {
			defer wg.Done()
			// If deployment found -> return
			if d.Name == depName {
				dep = d
				depExist = true
				return
			}
		}(d)
	}

	wg.Wait()

	if !depExist {
		return v1.Deployment{}, nil
	}

	return dep, nil

}

func (p podViewClient) GetPods() (api.PodList, error) {
	dep, err := p.ValidateDeployment()
	if err != nil {
		return api.PodList{}, err
	}

	if reflect.DeepEqual(dep, v1.Deployment{}) {
		return api.PodList{}, ErrDeploymentNotFound
	}
}
