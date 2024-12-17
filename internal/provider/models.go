package provider

import "encoding/json"

// Cluster represents a Pomerium Zero cluster
type Cluster struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	NamespaceID         string `json:"namespaceId"`
	Domain              string `json:"domain"`
	FQDN                string `json:"fqdn"`
	AutoDetectIPAddress string `json:"autoDetectIpAddress"`
	CreatedAt           string `json:"createdAt"`
	UpdatedAt           string `json:"updatedAt"`
}

// Policy represents a Pomerium Zero policy
type Policy struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Enforced    bool            `json:"enforced"`
	Explanation string          `json:"explanation"`
	NamespaceID string          `json:"namespaceId"`
	PPL         json.RawMessage `json:"ppl"`
	Remediation string          `json:"remediation"`
	Routes      []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"routes"`
}
