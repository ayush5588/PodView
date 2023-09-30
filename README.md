# PodView
PodView can be used to list pods belonging to a deployment.

See [examples](https://github.com/ayush5588/PodView/tree/main/example) folder to understand how to use the package.

Currently 2 methods are supported by this package to GET the pods:
1. **GetPods()** - Given the deployment name and namespace (optional), return all the pods belonging to the deployment.

2. **GetPodsWithStatus(status)** - Same as above method. Only difference is that this also takes in an argument called status (i.e. Running, Failed, Pending) and returns pods of the deployment having that status.
