package models

type KubectlPluginRequest struct {
	Container string `json:"container"`
	Image     string `json:"image"`
	Command   string `json:"commad"`
}
