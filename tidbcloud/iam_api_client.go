package tidbcloud

import (
	"context"
	"net/http"

	"github.com/icholy/digest"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/iam"
)

const (
	DefaultIAMEndpoint = "https://iam.tidbapi.com"
)

type TiDBCloudIAMClient interface {
	ListSQLUsers(ctx context.Context, clusterID string, pageSize *int32, pageToken *string) (*iam.ApiListSqlUsersRsp, error)
	CreateSQLUser(ctx context.Context, clusterID string, body *iam.ApiCreateSqlUserReq) (*iam.ApiSqlUser, error)
	GetSQLUser(ctx context.Context, clusterID string, userName string) (*iam.ApiSqlUser, error)
	DeleteSQLUser(ctx context.Context, clusterID string, userName string) (*iam.ApiBasicResp, error)
	UpdateSQLUser(ctx context.Context, clusterID string, userName string, body *iam.ApiUpdateSqlUserReq) (*iam.ApiSqlUser, error)
}

func NewIAMApiClient(rt http.RoundTripper, iamEndpoint string, userAgent string) (*iam.APIClient, error) {
	httpClient := &http.Client{
		Transport: rt,
	}

	// v1beta1 api (iam)
	if iamEndpoint == "" {
		iamEndpoint = DefaultIAMEndpoint
	}
	iamURL, err := validateApiUrl(iamEndpoint)
	if err != nil {
		return nil, err
	}

	iamCfg := iam.NewConfiguration()
	iamCfg.HTTPClient = httpClient
	iamCfg.Host = iamURL.Host
	iamCfg.UserAgent = userAgent
	return iam.NewAPIClient(iamCfg), nil
}

type IAMClientDelegate struct {
	ic *iam.APIClient
}

func NewIAMClientDelegate(publicKey string, privateKey string, iamEndpoint string, userAgent string) (*IAMClientDelegate, error) {
	transport := NewTransportWithAgent(&digest.Transport{
		Username: publicKey,
		Password: privateKey,
	}, userAgent)

	ic, err := NewIAMApiClient(transport, iamEndpoint, userAgent)
	if err != nil {
		return nil, err
	}
	return &IAMClientDelegate{
		ic: ic,
	}, nil
}

func (d *IAMClientDelegate) ListSQLUsers(ctx context.Context, clusterID string, pageSize *int32, pageToken *string) (*iam.ApiListSqlUsersRsp, error) {
	r := d.ic.AccountAPI.V1beta1ClustersClusterIdSqlUsersGet(ctx, clusterID)
	if pageSize != nil {
		r = r.PageSize(*pageSize)
	}
	if pageToken != nil {
		r = r.PageToken(*pageToken)
	}
	resp, h, err := r.Execute()
	return resp, parseError(err, h)
}

func (d *IAMClientDelegate) CreateSQLUser(ctx context.Context, clusterId string, body *iam.ApiCreateSqlUserReq) (*iam.ApiSqlUser, error) {
	r := d.ic.AccountAPI.V1beta1ClustersClusterIdSqlUsersPost(ctx, clusterId)
	if body != nil {
		r = r.SqlUser(*body)
	}
	res, h, err := r.Execute()
	return res, parseError(err, h)
}

func (d *IAMClientDelegate) GetSQLUser(ctx context.Context, clusterID string, userName string) (*iam.ApiSqlUser, error) {
	r := d.ic.AccountAPI.V1beta1ClustersClusterIdSqlUsersUserNameGet(ctx, clusterID, userName)
	res, h, err := r.Execute()
	return res, parseError(err, h)
}

func (d *IAMClientDelegate) DeleteSQLUser(ctx context.Context, clusterID string, userName string) (*iam.ApiBasicResp, error) {
	r := d.ic.AccountAPI.V1beta1ClustersClusterIdSqlUsersUserNameDelete(ctx, clusterID, userName)
	res, h, err := r.Execute()
	return res, parseError(err, h)
}

func (d *IAMClientDelegate) UpdateSQLUser(ctx context.Context, clusterID string, userName string, body *iam.ApiUpdateSqlUserReq) (*iam.ApiSqlUser, error) {
	r := d.ic.AccountAPI.V1beta1ClustersClusterIdSqlUsersUserNamePatch(ctx, clusterID, userName)
	if body != nil {
		r = r.SqlUser(*body)
	}
	res, h, err := r.Execute()
	return res, parseError(err, h)
}
