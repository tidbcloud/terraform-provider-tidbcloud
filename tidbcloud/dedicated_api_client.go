package tidbcloud

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/icholy/digest"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
)

const (
	DefaultDedicatedEndpoint = "https://dedicated.tidbapi.com"
)

type TiDBCloudDedicatedClient interface {
	ListRegions(ctx context.Context, cloudProvider string, projectId string) ([]dedicated.Commonv1beta1Region, error)
	GetRegion(ctx context.Context, regionId string) (*dedicated.Commonv1beta1Region, error)
	ListCloudProviders(ctx context.Context, projectId string) ([]dedicated.V1beta1RegionCloudProvider, error)
}

func NewDedicatedApiClient(rt http.RoundTripper, dedicatedEndpoint string, userAgent string) (*dedicated.APIClient, error) {
	httpClient := &http.Client{
		Transport: rt,
	}

	// v1beta1 api (dedicated)
	if dedicatedEndpoint == "" {
		dedicatedEndpoint = DefaultDedicatedEndpoint
	}
	dedicatedURL, err := url.ParseRequestURI(dedicatedEndpoint)
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

func NewDedicatedClientDelegate(publicKey string, privateKey string, dedicatedEndpoint string, userAgent string) (*DedicatedClientDelegate, error) {
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

func parseError(err error, resp *http.Response) error {
	if resp != nil {
		defer resp.Body.Close()
	}
	if err == nil {
		return nil
	}
	if resp == nil {
		return err
	}
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return err
	}
	path := "<path>"
	if resp.Request != nil {
		path = fmt.Sprintf("[%s %s]", resp.Request.Method, resp.Request.URL.Path)
	}
	traceId := resp.Header.Get("X-Debug-Trace-Id")
	if traceId == "" {
		traceId = "<trace_id>"
	}
	return fmt.Errorf("%s[%s][%s] %s", path, err.Error(), traceId, body)
}
