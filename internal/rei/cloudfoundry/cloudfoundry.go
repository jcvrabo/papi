package cloudfoundry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rabobank/papi/pkg/rei"
)

type Config struct {
	APIURL       string `json:"api_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type CloudFoundry struct {
	config     Config
	httpClient *http.Client
}

func New(connectionConfig map[string]interface{}) (rei.RuntimeEnvironment, error) {
	cfg := Config{}
	if v, ok := connectionConfig["api_url"].(string); ok {
		cfg.APIURL = v
	} else {
		return nil, fmt.Errorf("api_url is required")
	}
	if v, ok := connectionConfig["client_id"].(string); ok {
		cfg.ClientID = v
	}
	if v, ok := connectionConfig["client_secret"].(string); ok {
		cfg.ClientSecret = v
	}

	return &CloudFoundry{
		config:     cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (cf *CloudFoundry) CreateNamespace(ctx context.Context, req rei.CreateNamespaceRequest) (rei.CreateNamespaceResult, error) {
	orgName := req.CompositeName
	spaceName := "default"
	if team, ok := req.NameComponents["team"]; ok {
		orgName = team
	}
	if project, ok := req.NameComponents["project"]; ok {
		spaceName = project
	}

	return rei.CreateNamespaceResult{
		ExtensionData: map[string]string{
			"cf_org_name":   orgName,
			"cf_space_name": spaceName,
			"cf_org_guid":   fmt.Sprintf("org-%s", req.NamespaceID[:8]),
			"cf_space_guid": fmt.Sprintf("space-%s", req.NamespaceID[:8]),
		},
	}, nil
}

func (cf *CloudFoundry) DeleteNamespace(ctx context.Context, req rei.DeleteNamespaceRequest) error {
	return nil
}

func (cf *CloudFoundry) HealthCheck(ctx context.Context) (rei.HealthStatus, error) {
	resp, err := cf.httpClient.Get(cf.config.APIURL + "/v3/info")
	if err != nil {
		return rei.HealthStatus{Status: rei.StatusUnavailable, Message: err.Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return rei.HealthStatus{Status: rei.StatusHealthy}, nil
	}
	return rei.HealthStatus{Status: rei.StatusDegraded, Message: fmt.Sprintf("status %d", resp.StatusCode)}, nil
}

func init() {
	rei.Registry["cloudfoundry"] = New
}
