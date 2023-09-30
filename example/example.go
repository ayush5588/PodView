package main

import (
	"fmt"

	"github.com/ayush5588/PodView/pkg/podview"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func main() {
	// For testing purpose, I am reading kubeconfig and preparing the k8s client from it.
	// You can directly call the NewPodViewClient and pass it the k8s client.
	kc := "/home/ayush5588/kubeconfig.yaml"

	config, err := clientcmd.BuildConfigFromFlags("", kc)
	if err != nil {
		panic(err)
	}

	var scheme *runtime.Scheme

	c, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		fmt.Println(err)
	}

	/*
		Following arguments are passed to the NewPodViewClient method:
		 	1. client
		 	2. Deployment name
		 	3. Deployment namespace (optional)
	*/
	nc := podview.NewPodViewClient(c, "kube-state-metrics", "")

	pods, err = nc.GetPods()
	if err != nil {
		panic(err)

	}

	fmt.Println(pods.Pods)

	podsWithPendingStatus, err := nc.GetPodsWithStatus("Pending")
	if err != nil {
		panic(err)
	}

	fmt.Println("Pods belonging to the given deployment & having status as Pending: ")
	
	fmt.Println(podsWithPendingStatus.Pods)

}
