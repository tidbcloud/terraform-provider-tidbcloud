package tidbcloud

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/icholy/digest"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/dedicated"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/iam"
)

const (
	DefaultDedicatedEndpoint = "https://dedicated.tidbapi.com"
	DefaultIAMEndpoint       = "https://iam.tidbapi.com"
)

type TiDBCloudDedicatedClient interface {
	// CreateCluster(ctx context.Context, body *dedicated.TidbCloudOpenApidedicatedv1beta1Cluster) (*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error)
	ListRegions(ctx context.Context) (*dedicated.TidbCloudOpenApidedicatedv1beta1ListRegionsResponse, error)
	GetRegion(ctx context.Context, regionId string) (*dedicated.Commonv1beta1Region, error)
}

func NewDedicatedApiClient(rt http.RoundTripper, dedicatedEndpoint string, iamEndpoint string, userAgent string) (*dedicated.APIClient, *iam.APIClient, error) {
	httpClient := &http.Client{
		Transport: rt,
	}

	iamURL, err := url.ParseRequestURI(iamEndpoint)
	if err != nil {
		return nil, nil, err
	}

	// v1beta1 api (dedicated)
	dedicatedURL, err := url.ParseRequestURI(dedicatedEndpoint)
	if err != nil {
		return nil, nil, err
	}

	iamCfg := iam.NewConfiguration()
	iamCfg.HTTPClient = httpClient
	iamCfg.Host = iamURL.Host
	iamCfg.UserAgent = userAgent

	dedicatedCfg := dedicated.NewConfiguration()
	dedicatedCfg.HTTPClient = httpClient
	dedicatedCfg.Host = dedicatedURL.Host
	dedicatedCfg.UserAgent = userAgent
	return dedicated.NewAPIClient(dedicatedCfg), iam.NewAPIClient(iamCfg), nil
}

type DedicatedClientDelegate struct {
	ic *iam.APIClient
	dc *dedicated.APIClient
}

func NewDedicatedClientDelegate(publicKey string, privateKey string, dedicatedEndpoint string, iamEndpoint string, userAgent string) (*DedicatedClientDelegate, error) {
	transport := NewTransportWithAgent(&digest.Transport{
		Username: publicKey,
		Password: privateKey,
	}, userAgent)

	dc, ic, err := NewDedicatedApiClient(transport, dedicatedEndpoint, iamEndpoint, userAgent)
	if err != nil {
		return nil, err
	}
	return &DedicatedClientDelegate{
		dc: dc,
		ic: ic,
	}, nil
}

func (d *DedicatedClientDelegate) ListRegions(ctx context.Context) (*dedicated.TidbCloudOpenApidedicatedv1beta1ListRegionsResponse, error) {
	tflog.Debug(ctx, fmt.Sprintf("dc.cfg: %v", *d.dc.GetConfig()))
	resp, h, err := d.dc.RegionServiceAPI.RegionServiceListRegions(ctx).Execute()
	tflog.Trace(ctx, fmt.Sprintf("ListRegions: %v, h: %v, err: %v", resp, h, err))
	return resp, parseError(err, h)
}

func (d *DedicatedClientDelegate) GetRegion(ctx context.Context, regionId string) (*dedicated.Commonv1beta1Region, error) {
	resp, h, err := d.dc.RegionServiceAPI.RegionServiceGetRegion(ctx, regionId).Execute()
	return resp, parseError(err, h)
}

// func (d *DedicatedClientDelegate) CreateCluster(ctx context.Context, body *dedicated.TidbCloudOpenApidedicatedv1beta1Cluster) (*dedicated.TidbCloudOpenApidedicatedv1beta1Cluster, error) {
// 	r := d.dc.ClusterServiceAPI.ClusterServiceCreateCluster(ctx)
// 	if body != nil {
// 		r = r.Cluster(*body)
// 	}
// 	c, h, err := r.Execute()
// 	return c, parseError(err, h)
// }

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