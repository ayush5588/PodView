package api

// PodList ...
type PodList struct {
	Pods []Pod `json:"pods"`
}

// Pod ...
type Pod struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}
