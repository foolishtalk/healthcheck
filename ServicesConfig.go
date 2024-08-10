package main

type ServiceConfig struct {
	Wecom_hook_url string   `json:"wecom_hook_url"`
	URLs           []string `json:"urls"`
}
