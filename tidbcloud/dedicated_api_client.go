package tidbcloud

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/icholy/digest"
	"github.com/juju/errors"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

const (
	DefaultDedicatedEndpoint = "https://dedicated.tidbapi.com"
)

type TiDBCloudDedicatedClient interface {
	ListRegions(ctx context.Context, cloudProvider string, projectId string) ([]dedicated.Commonv1beta1Region, error)
	GetRegion(ctx context.Context, regionId string) (*dedicated.Commonv1beta1Region, error)
	ListCloudProviders(ctx context.Context, projectId string) ([]dedicated.V1beta1RegionCloudProvider, error)
	CreateCluster(ctx context.Context, body *dedicated.TidbCloudOpenApidedicatedv1beta1Cluster) (*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error)
	GetCluster(ctx context.Context, clusterId string) (*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error)
	ListClusters(ctx context.Context, projectId string, pageSize *int32, pageToken *string) (*dedicated.TidbCloudOpenApidedicatedv1beta1ListClustersResponse, error)
	DeleteCluster(ctx context.Context, clusterId string) (*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error)
	UpdateCluster(ctx context.Context, clusterId string, body *dedicated.TheUpdatedClusterConfiguration) (*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error)
	PauseCluster(ctx context.Context, clusterId string) (*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error)
	ResumeCluster(ctx context.Context, clusterId string) (*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error)
	ChangeClusterRootPassword(ctx context.Context, clusterId string, body *dedicated.V1beta1ClusterServiceResetRootPasswordBody) error
	CreateTiDBNodeGroup(ctx context.Context, clusterId string, body *dedicated.TidbNodeGroupServiceCreateTidbNodeGroupRequest) (*dedicated.Dedicatedv1beta1TidbNodeGroup, error)
	DeleteTiDBNodeGroup(ctx context.Context, clusterId string, nodeGroupId string) error
	UpdateTiDBNodeGroup(ctx context.Context, clusterId string, nodeGroupId string, body *dedicated.TidbNodeGroupServiceUpdateTidbNodeGroupRequest) (*dedicated.Dedicatedv1beta1TidbNodeGroup, error)
	GetTiDBNodeGroup(ctx context.Context, clusterId string, nodeGroupId string) (*dedicated.Dedicatedv1beta1TidbNodeGroup, error)
	ListTiDBNodeGroups(ctx context.Context, clusterId string, pageSize *int32, pageToken *string) (*dedicated.Dedicatedv1beta1ListTidbNodeGroupsResponse, error)
	CreatePrivateEndpointConnection(ctx context.Context, clusterId string, nodeGroupId string, body *dedicated.PrivateEndpointConnectionServiceCreatePrivateEndpointConnectionRequest) (*dedicated.Dedicatedv1beta1PrivateEndpointConnection, error)
	DeletePrivateEndpointConnection(ctx context.Context, clusterId string, nodeGroupId string, privateEndpointConnectionId string) error
	GetPrivateEndpointConnection(ctx context.Context, clusterId string, nodeGroupId string, privateEndpointConnectionId string) (*dedicated.Dedicatedv1beta1PrivateEndpointConnection, error)
	ListPrivateEndpointConnections(ctx context.Context, clusterId string, nodeGroupId string, pageSize *int32, pageToken *string) (*dedicated.Dedicatedv1beta1ListPrivateEndpointConnectionsResponse, error)
	GetPrivateLinkService(ctx context.Context, clusterId string, nodeGroupId string) (*dedicated.Dedicatedv1beta1PrivateLinkService, error)
	CreateNetworkContainer(ctx context.Context, body *dedicated.V1beta1NetworkContainer) (*dedicated.V1beta1NetworkContainer, error)
	DeleteNetworkContainer(ctx context.Context, networkContainerId string) error
	GetNetworkContainer(ctx context.Context, networkContainerId string) (*dedicated.V1beta1NetworkContainer, error)
	ListNetworkContainers(ctx context.Context, projectId string, pageSize *int32, pageToken *string) (*dedicated.V1beta1ListNetworkContainersResponse, error)
	CreateVPCPeering(ctx context.Context, body *dedicated.Dedicatedv1beta1VpcPeering) (*dedicated.Dedicatedv1beta1VpcPeering, error)
	DeleteVPCPeering(ctx context.Context, vpcPeeringId string) error
	GetVPCPeering(ctx context.Context, vpcPeeringId string) (*dedicated.Dedicatedv1beta1VpcPeering, error)
	ListVPCPeerings(ctx context.Context, projectId string, cloudProvider string, pageSize *int32, pageToken *string) (*dedicated.Dedicatedv1beta1ListVpcPeeringsResponse, error)
	UpdatePublicEndpoint(ctx context.Context, clusterId string, nodeGroupId string, body *dedicated.TidbNodeGroupServiceUpdatePublicEndpointSettingRequest) (*dedicated.V1beta1PublicEndpointSetting, error)
	GetPublicEndpoint(ctx context.Context, clusterId string, nodeGroupId string) (*dedicated.V1beta1PublicEndpointSetting, error)
}

func NewDedicatedApiClient(rt http.RoundTripper, dedicatedEndpoint string, userAgent string) (*dedicated.APIClient, error) {
	httpClient := &http.Client{
		Transport: rt,
	}

	// v1beta1 api (dedicated)
	if dedicatedEndpoint == "" {
		dedicatedEndpoint = DefaultDedicatedEndpoint
	}
	dedicatedURL, err := validateApiUrl(dedicatedEndpoint)
	if err != nil {
		return nil, err
	}

	dedicatedCfg := dedicated.NewConfiguration()
	dedicatedCfg.HTTPClient = httpClient
	dedicatedCfg.Host = dedicatedURL.Host
	dedicatedCfg.UserAgent = userAgent
	return dedicated.NewAPIClient(dedicatedCfg), nil
}

type DedicatedClientDelegate struct {
	dc *dedicated.APIClient
}

func NewDedicatedClientDelegate(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (TiDBCloudDedicatedClient, error) {
	transport := NewTransportWithAgent(&digest.Transport{
		Username: publicKey,
		Password: privateKey,
	}, userAgent)

	dc, err := NewDedicatedApiClient(transport, dedicatedEndpoint, userAgent)
	if err != nil {
		return nil, err
	}
	return &DedicatedClientDelegate{
		dc: dc,
	}, nil
}

func (d *DedicatedClientDelegate) ListRegions(ctx context.Context, cloudProvider string, projectId string) ([]dedicated.Commonv1beta1Region, error) {
	req := d.dc.RegionServiceAPI.RegionServiceListRegions(ctx).PageSize(100)
	if cloudProvider != "" {
		req = req.CloudProvider(cloudProvider)
	}
	if projectId != "" {
		req = req.ProjectId(projectId)
	}

	resp, h, err := req.Execute()
	return resp.Regions, parseError(err, h)
}

func (d *DedicatedClientDelegate) GetRegion(ctx context.Context, regionId string) (*dedicated.Commonv1beta1Region, error) {
	resp, h, err := d.dc.RegionServiceAPI.RegionServiceGetRegion(ctx, regionId).Execute()
	return resp, parseError(err, h)
}

func (d *DedicatedClientDelegate) ListCloudProviders(ctx context.Context, projectId string) ([]dedicated.V1beta1RegionCloudProvider, error) {
	req := d.dc.RegionServiceAPI.RegionServiceShowCloudProviders(ctx)
	if projectId != "" {
		req = req.ProjectId(projectId)
	}

	resp, h, err := req.Execute()
	return resp.CloudProviders, parseError(err, h)
}

func (d *DedicatedClientDelegate) CreateCluster(ctx context.Context, body *dedicated.TidbCloudOpenApidedicatedv1beta1Cluster) (*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error) {
	r := d.dc.ClusterServiceAPI.ClusterServiceCreateCluster(ctx)
	if body != nil {
		r = r.Cluster(*body)
	}
	c, h, err := r.Execute()
	return c, parseError(err, h)
}

func (d *DedicatedClientDelegate) GetCluster(ctx context.Context, clusterId string) (*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error) {
	resp, h, err := d.dc.ClusterServiceAPI.ClusterServiceGetCluster(ctx, clusterId).Execute()
	return resp, parseError(err, h)
}

func (d *DedicatedClientDelegate) ListClusters(ctx context.Context, projectId string, pageSize *int32, pageToken *string) (*dedicated.TidbCloudOpenApidedicatedv1beta1ListClustersResponse, error) {
	r := d.dc.ClusterServiceAPI.ClusterServiceListClusters(ctx)
	if projectId != "" {
		r = r.ProjectId(projectId)
	}
	if pageSize != nil {
		r = r.PageSize(*pageSize)
	}
	if pageToken != nil {
		r = r.PageToken(*pageToken)
	}
	resp, h, err := r.Execute()
	return resp, parseError(err, h)
}

func (d *DedicatedClientDelegate) DeleteCluster(ctx context.Context, clusterId string) (*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error) {
	resp, h, err := d.dc.ClusterServiceAPI.ClusterServiceDeleteCluster(ctx, clusterId).Execute()
	return resp, parseError(err, h)
}

func (d *DedicatedClientDelegate) UpdateCluster(ctx context.Context, clusterId string, body *dedicated.TheUpdatedClusterConfiguration) (*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error) {
	r := d.dc.ClusterServiceAPI.ClusterServiceUpdateCluster(ctx, clusterId)
	if body != nil {
		r = r.Cluster(*body)
	}
	c, h, err := r.Execute()
	return c, parseError(err, h)
}

func (d *DedicatedClientDelegate) PauseCluster(ctx context.Context, clusterId string) (*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error) {
	resp, h, err := d.dc.ClusterServiceAPI.ClusterServicePauseCluster(ctx, clusterId).Execute()
	return &resp.Cluster, parseError(err, h)
}

func (d *DedicatedClientDelegate) ResumeCluster(ctx context.Context, clusterId string) (*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error) {
	resp, h, err := d.dc.ClusterServiceAPI.ClusterServiceResumeCluster(ctx, clusterId).Execute()
	return &resp.Cluster, parseError(err, h)
}

func (d *DedicatedClientDelegate) ChangeClusterRootPassword(ctx context.Context, clusterId string, body *dedicated.V1beta1ClusterServiceResetRootPasswordBody) error {
	r := d.dc.ClusterServiceAPI.ClusterServiceResetRootPassword(ctx, clusterId)
	if body != nil {
		r = r.Body(*body)
	}
	_, h, err := r.Execute()
	return parseError(err, h)
}

func (d *DedicatedClientDelegate) CreateTiDBNodeGroup(ctx context.Context, clusterId string, body *dedicated.TidbNodeGroupServiceCreateTidbNodeGroupRequest) (*dedicated.Dedicatedv1beta1TidbNodeGroup, error) {
	r := d.dc.TidbNodeGroupServiceAPI.TidbNodeGroupServiceCreateTidbNodeGroup(ctx, clusterId)
	if body != nil {
		r = r.TidbNodeGroup(*body)
	}
	c, h, err := r.Execute()
	return c, parseError(err, h)
}

func (d *DedicatedClientDelegate) DeleteTiDBNodeGroup(ctx context.Context, clusterId string, nodeGroupId string) error {
	_, h, err := d.dc.TidbNodeGroupServiceAPI.TidbNodeGroupServiceDeleteTidbNodeGroup(ctx, clusterId, nodeGroupId).Execute()
	return parseError(err, h)
}

func (d *DedicatedClientDelegate) UpdateTiDBNodeGroup(ctx context.Context, clusterId string, nodeGroupId string, body *dedicated.TidbNodeGroupServiceUpdateTidbNodeGroupRequest) (*dedicated.Dedicatedv1beta1TidbNodeGroup, error) {
	r := d.dc.TidbNodeGroupServiceAPI.TidbNodeGroupServiceUpdateTidbNodeGroup(ctx, clusterId, nodeGroupId)
	if body != nil {
		r = r.TidbNodeGroup(*body)
	}
	c, h, err := r.Execute()
	return c, parseError(err, h)
}

func (d *DedicatedClientDelegate) GetTiDBNodeGroup(ctx context.Context, clusterId string, nodeGroupId string) (*dedicated.Dedicatedv1beta1TidbNodeGroup, error) {
	resp, h, err := d.dc.TidbNodeGroupServiceAPI.TidbNodeGroupServiceGetTidbNodeGroup(ctx, clusterId, nodeGroupId).Execute()
	return resp, parseError(err, h)
}

func (d *DedicatedClientDelegate) ListTiDBNodeGroups(ctx context.Context, clusterId string, pageSize *int32, pageToken *string) (*dedicated.Dedicatedv1beta1ListTidbNodeGroupsResponse, error) {
	r := d.dc.TidbNodeGroupServiceAPI.TidbNodeGroupServiceListTidbNodeGroups(ctx, clusterId)
	if pageSize != nil {
		r = r.PageSize(*pageSize)
	}
	if pageToken != nil {
		r = r.PageToken(*pageToken)
	}
	resp, h, err := r.Execute()
	return resp, parseError(err, h)
}

func (d *DedicatedClientDelegate) CreatePrivateEndpointConnection(ctx context.Context, clusterId string, nodeGroupId string, body *dedicated.PrivateEndpointConnectionServiceCreatePrivateEndpointConnectionRequest) (*dedicated.Dedicatedv1beta1PrivateEndpointConnection, error) {
	r := d.dc.PrivateEndpointConnectionServiceAPI.PrivateEndpointConnectionServiceCreatePrivateEndpointConnection(ctx, clusterId, nodeGroupId)
	if body != nil {
		r = r.PrivateEndpointConnection(*body)
	}
	c, h, err := r.Execute()
	return c, parseError(err, h)
}

func (d *DedicatedClientDelegate) DeletePrivateEndpointConnection(ctx context.Context, clusterId string, nodeGroupId string, privateEndpointConnectionId string) error {
	_, h, err := d.dc.PrivateEndpointConnectionServiceAPI.PrivateEndpointConnectionServiceDeletePrivateEndpointConnection(ctx, clusterId, nodeGroupId, privateEndpointConnectionId).Execute()
	return parseError(err, h)
}

func (d *DedicatedClientDelegate) GetPrivateEndpointConnection(ctx context.Context, clusterId string, nodeGroupId string, privateEndpointConnectionId string) (*dedicated.Dedicatedv1beta1PrivateEndpointConnection, error) {
	resp, h, err := d.dc.PrivateEndpointConnectionServiceAPI.PrivateEndpointConnectionServiceGetPrivateEndpointConnection(ctx, clusterId, nodeGroupId, privateEndpointConnectionId).Execute()
	return resp, parseError(err, h)
}

func (d *DedicatedClientDelegate) ListPrivateEndpointConnections(ctx context.Context, clusterId string, nodeGroupId string, pageSize *int32, pageToken *string) (*dedicated.Dedicatedv1beta1ListPrivateEndpointConnectionsResponse, error) {
	r := d.dc.PrivateEndpointConnectionServiceAPI.PrivateEndpointConnectionServiceListPrivateEndpointConnections(ctx, clusterId, nodeGroupId)
	if pageSize != nil {
		r = r.PageSize(*pageSize)
	}
	if pageToken != nil {
		r = r.PageToken(*pageToken)
	}
	resp, h, err := r.Execute()
	return resp, parseError(err, h)
}

func (d *DedicatedClientDelegate) GetPrivateLinkService(ctx context.Context, clusterId string, nodeGroupId string) (*dedicated.Dedicatedv1beta1PrivateLinkService, error) {
	resp, h, err := d.dc.PrivateEndpointConnectionServiceAPI.PrivateEndpointConnectionServiceGetPrivateLinkService(ctx, clusterId, nodeGroupId).Execute()
	return resp, parseError(err, h)
}

func (d *DedicatedClientDelegate) CreateNetworkContainer(ctx context.Context, body *dedicated.V1beta1NetworkContainer) (*dedicated.V1beta1NetworkContainer, error) {
	r := d.dc.NetworkContainerServiceAPI.NetworkContainerServiceCreateNetworkContainer(ctx)
	if body != nil {
		r = r.NetworkContainer(*body)
	}
	c, h, err := r.Execute()
	return c, parseError(err, h)
}

func (d *DedicatedClientDelegate) DeleteNetworkContainer(ctx context.Context, networkContainerId string) error {
	_, h, err := d.dc.NetworkContainerServiceAPI.NetworkContainerServiceDeleteNetworkContainer(ctx, networkContainerId).Execute()
	return parseError(err, h)
}

func (d *DedicatedClientDelegate) GetNetworkContainer(ctx context.Context, networkContainerId string) (*dedicated.V1beta1NetworkContainer, error) {
	resp, h, err := d.dc.NetworkContainerServiceAPI.NetworkContainerServiceGetNetworkContainer(ctx, networkContainerId).Execute()
	return resp, parseError(err, h)
}

func (d *DedicatedClientDelegate) ListNetworkContainers(ctx context.Context, projectId string, pageSize *int32, pageToken *string) (*dedicated.V1beta1ListNetworkContainersResponse, error) {
	r := d.dc.NetworkContainerServiceAPI.NetworkContainerServiceListNetworkContainers(ctx)
	if projectId != "" {
		r = r.ProjectId(projectId)
	}
	if pageSize != nil {
		r = r.PageSize(*pageSize)
	}
	if pageToken != nil {
		r = r.PageToken(*pageToken)
	}
	resp, h, err := r.Execute()
	return resp, parseError(err, h)
}

func (d *DedicatedClientDelegate) CreateVPCPeering(ctx context.Context, body *dedicated.Dedicatedv1beta1VpcPeering) (*dedicated.Dedicatedv1beta1VpcPeering, error) {
	r := d.dc.NetworkContainerServiceAPI.NetworkContainerServiceCreateVpcPeering(ctx)
	if body != nil {
		r = r.VpcPeering(*body)
	}
	c, h, err := r.Execute()
	return c, parseError(err, h)
}

func (d *DedicatedClientDelegate) DeleteVPCPeering(ctx context.Context, vpcPeeringId string) error {
	_, h, err := d.dc.NetworkContainerServiceAPI.NetworkContainerServiceDeleteVpcPeering(ctx, vpcPeeringId).Execute()
	return parseError(err, h)
}

func (d *DedicatedClientDelegate) GetVPCPeering(ctx context.Context, vpcPeeringId string) (*dedicated.Dedicatedv1beta1VpcPeering, error) {
	resp, h, err := d.dc.NetworkContainerServiceAPI.NetworkContainerServiceGetVpcPeering(ctx, vpcPeeringId).Execute()
	return resp, parseError(err, h)
}

func (d *DedicatedClientDelegate) ListVPCPeerings(ctx context.Context, projectId string, cloudProvider string, pageSize *int32, pageToken *string) (*dedicated.Dedicatedv1beta1ListVpcPeeringsResponse, error) {
	r := d.dc.NetworkContainerServiceAPI.NetworkContainerServiceListVpcPeerings(ctx)
	if projectId != "" {
		r = r.ProjectId(projectId)
	}
	if cloudProvider != "" {
		r = r.CloudProvider(dedicated.PrivateEndpointConnectionServiceListPrivateEndpointConnectionsCloudProviderParameter(cloudProvider))
	}
	if pageSize != nil {
		r = r.PageSize(*pageSize)
	}
	if pageToken != nil {
		r = r.PageToken(*pageToken)
	}
	resp, h, err := r.Execute()
	return resp, parseError(err, h)
}

func (d *DedicatedClientDelegate) UpdatePublicEndpoint(ctx context.Context, clusterId string, nodeGroupId string, body *dedicated.TidbNodeGroupServiceUpdatePublicEndpointSettingRequest) (*dedicated.V1beta1PublicEndpointSetting, error) {
	r := d.dc.TidbNodeGroupServiceAPI.TidbNodeGroupServiceUpdatePublicEndpointSetting(ctx, clusterId, nodeGroupId)
	if body != nil {
		r = r.PublicEndpointSetting(*body)
	}
	c, h, err := r.Execute()
	return c, parseError(err, h)
}

func (d *DedicatedClientDelegate) GetPublicEndpoint(ctx context.Context, clusterId string, nodeGroupId string) (*dedicated.V1beta1PublicEndpointSetting, error) {
	resp, h, err := d.dc.TidbNodeGroupServiceAPI.TidbNodeGroupServiceGetPublicEndpointSetting(ctx, clusterId, nodeGroupId).Execute()
	return resp, parseError(err, h)
}

func parseError(err error, resp *http.Response) error {
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()
	if err == nil {
		return nil
	}
	if resp == nil {
		return err
	}
	body, err1 := io.ReadAll(resp.Body)
	if err1 != nil {
		return err
	}
	path := "<path>"
	if resp.Request != nil {
		path = fmt.Sprintf("[%s %s]", resp.Request.Method, resp.Request.URL.Path)
	}
	traceId := "<trace_id>"
	if resp.Header.Get("X-Debug-Trace-Id") != "" {
		traceId = resp.Header.Get("X-Debug-Trace-Id")
	}
	return fmt.Errorf("%s[%s][%s] %s", path, err.Error(), traceId, body)
}

func validateApiUrl(value string) (*url.URL, error) {
	u, err := url.ParseRequestURI(value)
	if err != nil {
		return nil, errors.Annotate(err, "api url should format as <schema>://<host>")
	}
	return u, nil
}
