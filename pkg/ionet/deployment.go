package ionet

import (
	"fmt"

	"github.com/55gY/new-api-lite/common"

	"github.com/samber/lo"
)

// DeployContainer deploys a new container with the specified configuration
func (c *Client) DeployContainer(req *DeploymentRequest) (*DeploymentResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("deployment request cannot be nil")
	}

	// Validate required fields
	if req.ResourcePrivateName == "" {
		return nil, fmt.Errorf("resource_private_name is required")
	}
	if len(req.LocationIDs) == 0 {
		return nil, fmt.Errorf("location_ids is required")
	}
	if req.HardwareID <= 0 {
		return nil, fmt.Errorf("hardware_id is required")
	}
	if req.RegistryConfig.ImageURL == "" {
		return nil, fmt.Errorf("registry_config.image_url is required")
	}
	if req.GPUsPerContainer < 1 {
		return nil, fmt.Errorf("gpus_per_container must be at least 1")
	}
	if req.DurationHours < 1 {
		return nil, fmt.Errorf("duration_hours must be at least 1")
	}
	if req.ContainerConfig.ReplicaCount < 1 {
		return nil, fmt.Errorf("container_config.replica_count must be at least 1")
	}

	resp, err := c.makeRequest("POST", "/deploy", req)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy container: %w", err)
	}

	// API returns direct format:
	// {"status": "string", "deployment_id": "..."}
	var deployResp DeploymentResponse
	if err := common.Unmarshal(resp.Body, &deployResp); err != nil {
		return nil, fmt.Errorf("failed to parse deployment response: %w", err)
	}

	return &deployResp, nil
}

// ListDeployments retrieves a list of deployments with optional filtering
func (c *Client) ListDeployments(opts *ListDeploymentsOptions) (*DeploymentList, error) {
	params := make(map[string]interface{})

	if opts != nil {
		params["status"] = opts.Status
		params["location_id"] = opts.LocationID
		params["page"] = opts.Page
		params["page_size"] = opts.PageSize
		params["sort_by"] = opts.SortBy
		params["sort_order"] = opts.SortOrder
	}

	endpoint := "/deployments" + buildQueryParams(params)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	var deploymentList DeploymentList
	if err := decodeData(resp.Body, &deploymentList); err != nil {
		return nil, fmt.Errorf("failed to parse deployments list: %w", err)
	}

	deploymentList.Deployments = lo.Map(deploymentList.Deployments, func(deployment Deployment, _ int) Deployment {
		deployment.GPUCount = deployment.HardwareQuantity
		deployment.Replicas = deployment.HardwareQuantity // Assuming 1:1 mapping for now
		return deployment
	})

	return &deploymentList, nil
}

// GetDeployment retrieves detailed information about a specific deployment
func (c *Client) GetDeployment(deploymentID string) (*DeploymentDetail, error) {
	if deploymentID == "" {
		return nil, fmt.Errorf("deployment ID cannot be empty")
	}

	endpoint := fmt.Sprintf("/deployment/%s", deploymentID)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment details: %w", err)
	}

	var deploymentDetail DeploymentDetail
	if err := decodeDataWithFlexibleTimes(resp.Body, &deploymentDetail); err != nil {
		return nil, fmt.Errorf("failed to parse deployment details: %w", err)
	}

	return &deploymentDetail, nil
}

// UpdateDeployment updates the configuration of an existing deployment
func (c *Client) UpdateDeployment(deploymentID string, req *UpdateDeploymentRequest) (*UpdateDeploymentResponse, error) {
	if deploymentID == "" {
		return nil, fmt.Errorf("deployment ID cannot be empty")
	}
	if req == nil {
		return nil, fmt.Errorf("update request cannot be nil")
	}

	endpoint := fmt.Sprintf("/deployment/%s", deploymentID)

	resp, err := c.makeRequest("PATCH", endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update deployment: %w", err)
	}

	// API returns direct format:
	// {"status": "string", "deployment_id": "..."}
	var updateResp UpdateDeploymentResponse
	if err := common.Unmarshal(resp.Body, &updateResp); err != nil {
		return nil, fmt.Errorf("failed to parse update deployment response: %w", err)
	}

	return &updateResp, nil
}

// ExtendDeployment extends the duration of an existing deployment
func (c *Client) ExtendDeployment(deploymentID string, req *ExtendDurationRequest) (*DeploymentDetail, error) {
	if deploymentID == "" {
		return nil, fmt.Errorf("deployment ID cannot be empty")
	}
	if req == nil {
		return nil, fmt.Errorf("extend request cannot be nil")
	}
	if req.DurationHours < 1 {
		return nil, fmt.Errorf("duration_hours must be at least 1")
	}

	endpoint := fmt.Sprintf("/deployment/%s/extend", deploymentID)

	resp, err := c.makeRequest("POST", endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("failed to extend deployment: %w", err)
	}

	var deploymentDetail DeploymentDetail
	if err := decodeDataWithFlexibleTimes(resp.Body, &deploymentDetail); err != nil {
		return nil, fmt.Errorf("failed to parse extended deployment details: %w", err)
	}

	return &deploymentDetail, nil
}

// DeleteDeployment deletes an active deployment
func (c *Client) DeleteDeployment(deploymentID string) (*UpdateDeploymentResponse, error) {
	if deploymentID == "" {
		return nil, fmt.Errorf("deployment ID cannot be empty")
	}

	endpoint := fmt.Sprintf("/deployment/%s", deploymentID)

	resp, err := c.makeRequest("DELETE", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete deployment: %w", err)
	}

	// API returns direct format:
	// {"status": "string", "deployment_id": "..."}
	var deleteResp UpdateDeploymentResponse
	if err := common.Unmarshal(resp.Body, &deleteResp); err != nil {
		return nil, fmt.Errorf("failed to parse delete deployment response: %w", err)
	}

	return &deleteResp, nil
}

// CheckClusterNameAvailability checks if a cluster name is available
func (c *Client) CheckClusterNameAvailability(clusterName string) (bool, error) {
	if clusterName == "" {
		return false, fmt.Errorf("cluster name cannot be empty")
	}

	params := map[string]interface{}{
		"cluster_name": clusterName,
	}

	endpoint := "/clusters/check_cluster_name_availability" + buildQueryParams(params)

	resp, err := c.makeRequest("GET", endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to check cluster name availability: %w", err)
	}

	var availabilityResp bool
	if err := common.Unmarshal(resp.Body, &availabilityResp); err != nil {
		return false, fmt.Errorf("failed to parse cluster name availability response: %w", err)
	}

	return availabilityResp, nil
}

// UpdateClusterName updates the name of an existing cluster/deployment
func (c *Client) UpdateClusterName(clusterID string, req *UpdateClusterNameRequest) (*UpdateClusterNameResponse, error) {
	if clusterID == "" {
		return nil, fmt.Errorf("cluster ID cannot be empty")
	}
	if req == nil {
		return nil, fmt.Errorf("update cluster name request cannot be nil")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("cluster name cannot be empty")
	}

	endpoint := fmt.Sprintf("/clusters/%s/update-name", clusterID)

	resp, err := c.makeRequest("PUT", endpoint, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update cluster name: %w", err)
	}

	// Parse the response directly without data wrapper based on API docs
	var updateResp UpdateClusterNameResponse
	if err := common.Unmarshal(resp.Body, &updateResp); err != nil {
		return nil, fmt.Errorf("failed to parse update cluster name response: %w", err)
	}

	return &updateResp, nil
}
