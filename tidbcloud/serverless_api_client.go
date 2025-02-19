package tidbcloud

import (
	"context"
	"net/http"

	"github.com/icholy/digest"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/br"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/branch"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/cluster"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/export"
	"github.com/tidbcloud/tidbcloud-cli/pkg/tidbcloud/v1beta1/serverless/imp"
)

const (
	DefaultServerlessEndpoint = "https://serverless.tidbapi.com"
)

type TiDBCloudServerlessClient interface {
	CreateCluster(ctx context.Context, req *cluster.TidbCloudOpenApiserverlessv1beta1Cluster) (*cluster.TidbCloudOpenApiserverlessv1beta1Cluster, error)
	DeleteCluster(ctx context.Context, clusterId string) (*cluster.TidbCloudOpenApiserverlessv1beta1Cluster, error)
	GetCluster(ctx context.Context, clusterId string, view cluster.ServerlessServiceGetClusterViewParameter) (*cluster.TidbCloudOpenApiserverlessv1beta1Cluster, error)
	ListClusters(ctx context.Context) ([]cluster.TidbCloudOpenApiserverlessv1beta1Cluster, error)
	PartialUpdateCluster(ctx context.Context, clusterId string, body *cluster.V1beta1ServerlessServicePartialUpdateClusterBody) (*cluster.TidbCloudOpenApiserverlessv1beta1Cluster, error)
	ListProviderRegions(ctx context.Context) (*cluster.TidbCloudOpenApiserverlessv1beta1ListRegionsResponse, error)
	CancelImport(ctx context.Context, clusterId string, id string) error
	CreateImport(ctx context.Context, clusterId string, body *imp.ImportServiceCreateImportBody) (*imp.Import, error)
	GetImport(ctx context.Context, clusterId string, id string) (*imp.Import, error)
	ListImports(ctx context.Context, clusterId string, pageSize *int32, pageToken, orderBy *string) (*imp.ListImportsResp, error)
	GetBranch(ctx context.Context, clusterId, branchId string, view branch.BranchServiceGetBranchViewParameter) (*branch.Branch, error)
	ListBranches(ctx context.Context, clusterId string, pageSize *int32, pageToken *string) (*branch.ListBranchesResponse, error)
	CreateBranch(ctx context.Context, clusterId string, body *branch.Branch) (*branch.Branch, error)
	DeleteBranch(ctx context.Context, clusterId string, branchId string) (*branch.Branch, error)
	ResetBranch(ctx context.Context, clusterId string, branchId string) (*branch.Branch, error)
	DeleteBackup(ctx context.Context, backupId string) (*br.V1beta1Backup, error)
	GetBackup(ctx context.Context, backupId string) (*br.V1beta1Backup, error)
	ListBackups(ctx context.Context, clusterId *string, pageSize *int32, pageToken *string) (*br.V1beta1ListBackupsResponse, error)
	Restore(ctx context.Context, body *br.V1beta1RestoreRequest) (*br.V1beta1RestoreResponse, error)
	StartUpload(ctx context.Context, clusterId string, fileName, targetDatabase, targetTable *string, partNumber *int32) (*imp.StartUploadResponse, error)
	CompleteUpload(ctx context.Context, clusterId string, uploadId *string, parts *[]imp.CompletePart) error
	CancelUpload(ctx context.Context, clusterId string, uploadId *string) error
	GetExport(ctx context.Context, clusterId string, exportId string) (*export.Export, error)
	CancelExport(ctx context.Context, clusterId string, exportId string) (*export.Export, error)
	CreateExport(ctx context.Context, clusterId string, body *export.ExportServiceCreateExportBody) (*export.Export, error)
	DeleteExport(ctx context.Context, clusterId string, exportId string) (*export.Export, error)
	ListExports(ctx context.Context, clusterId string, pageSize *int32, pageToken *string, orderBy *string) (*export.ListExportsResponse, error)
	ListExportFiles(ctx context.Context, clusterId string, exportId string, pageSize *int32, pageToken *string, isGenerateUrl bool) (*export.ListExportFilesResponse, error)
	DownloadExportFiles(ctx context.Context, clusterId string, exportId string, body *export.ExportServiceDownloadExportFilesBody) (*export.DownloadExportFilesResponse, error)
}

func NewServerlessApiClient(rt http.RoundTripper, serverlessEndpoint string, userAgent string) (*branch.APIClient, *cluster.APIClient, *br.APIClient, *imp.APIClient, *export.APIClient, error) {
	httpClient := &http.Client{
		Transport: rt,
	}

	// v1beta1 api (serverless)
	if serverlessEndpoint == "" {
		serverlessEndpoint = DefaultServerlessEndpoint
	}
	serverlessURL, err := validateApiUrl(serverlessEndpoint)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	clusterCfg := cluster.NewConfiguration()
	clusterCfg.HTTPClient = httpClient
	clusterCfg.Host = serverlessURL.Host
	clusterCfg.UserAgent = userAgent

	branchCfg := branch.NewConfiguration()
	branchCfg.HTTPClient = httpClient
	branchCfg.Host = serverlessURL.Host
	branchCfg.UserAgent = userAgent

	exportCfg := export.NewConfiguration()
	exportCfg.HTTPClient = httpClient
	exportCfg.Host = serverlessURL.Host
	exportCfg.UserAgent = userAgent

	importCfg := imp.NewConfiguration()
	importCfg.HTTPClient = httpClient
	importCfg.Host = serverlessURL.Host
	importCfg.UserAgent = userAgent

	backupRestoreCfg := br.NewConfiguration()
	backupRestoreCfg.HTTPClient = httpClient
	backupRestoreCfg.Host = serverlessURL.Host
	backupRestoreCfg.UserAgent = userAgent

	return branch.NewAPIClient(branchCfg), cluster.NewAPIClient(clusterCfg),
		br.NewAPIClient(backupRestoreCfg), imp.NewAPIClient(importCfg),
		export.NewAPIClient(exportCfg), nil
}

type ServerlessClientDelegate struct {
	bc  *branch.APIClient
	brc *br.APIClient
	sc  *cluster.APIClient
	sic *imp.APIClient
	ec  *export.APIClient
}

func NewServerlessClientDelegate(publicKey string, privateKey string, serverlessEndpoint string, userAgent string) (*ServerlessClientDelegate, error) {
	transport := NewTransportWithAgent(&digest.Transport{
		Username: publicKey,
		Password: privateKey,
	}, userAgent)

	bc, sc, brc, sic, ec, err := NewServerlessApiClient(transport, serverlessEndpoint, userAgent)
	if err != nil {
		return nil, err
	}
	return &ServerlessClientDelegate{
		bc:  bc,
		sc:  sc,
		brc: brc,
		ec:  ec,
		sic: sic,
	}, nil
}

func (d *ServerlessClientDelegate) CreateCluster(ctx context.Context, body *cluster.TidbCloudOpenApiserverlessv1beta1Cluster) (*cluster.TidbCloudOpenApiserverlessv1beta1Cluster, error) {
	r := d.sc.ServerlessServiceAPI.ServerlessServiceCreateCluster(ctx)
	if body != nil {
		r = r.Cluster(*body)
	}
	c, h, err := r.Execute()
	return c, parseError(err, h)
}

func (d *ServerlessClientDelegate) DeleteCluster(ctx context.Context, clusterId string) (*cluster.TidbCloudOpenApiserverlessv1beta1Cluster, error) {
	c, h, err := d.sc.ServerlessServiceAPI.ServerlessServiceDeleteCluster(ctx, clusterId).Execute()
	return c, parseError(err, h)
}

func (d *ServerlessClientDelegate) GetCluster(ctx context.Context, clusterId string, view cluster.ServerlessServiceGetClusterViewParameter) (*cluster.TidbCloudOpenApiserverlessv1beta1Cluster, error) {
	r := d.sc.ServerlessServiceAPI.ServerlessServiceGetCluster(ctx, clusterId)
	r = r.View(view)
	c, h, err := r.Execute()
	return c, parseError(err, h)
}

func (d *ServerlessClientDelegate) ListProviderRegions(ctx context.Context) (*cluster.TidbCloudOpenApiserverlessv1beta1ListRegionsResponse, error) {
	resp, h, err := d.sc.ServerlessServiceAPI.ServerlessServiceListRegions(ctx).Execute()
	return resp, parseError(err, h)
}

func (d *ServerlessClientDelegate) ListClusters(ctx context.Context) ([]cluster.TidbCloudOpenApiserverlessv1beta1Cluster, error) {
	r := d.sc.ServerlessServiceAPI.ServerlessServiceListClusters(ctx)
	resp, h, err := r.Execute()
	return resp.Clusters, parseError(err, h)
}

func (d *ServerlessClientDelegate) PartialUpdateCluster(ctx context.Context, clusterId string, body *cluster.V1beta1ServerlessServicePartialUpdateClusterBody) (*cluster.TidbCloudOpenApiserverlessv1beta1Cluster, error) {
	r := d.sc.ServerlessServiceAPI.ServerlessServicePartialUpdateCluster(ctx, clusterId)
	if body != nil {
		r = r.Body(*body)
	}
	c, h, err := r.Execute()
	return c, parseError(err, h)
}

func (d *ServerlessClientDelegate) CancelImport(ctx context.Context, clusterId string, id string) error {
	_, h, err := d.sic.ImportServiceAPI.ImportServiceCancelImport(ctx, clusterId, id).Execute()
	return parseError(err, h)
}

func (d *ServerlessClientDelegate) CreateImport(ctx context.Context, clusterId string, body *imp.ImportServiceCreateImportBody) (*imp.Import, error) {
	r := d.sic.ImportServiceAPI.ImportServiceCreateImport(ctx, clusterId)
	if body != nil {
		r = r.Body(*body)
	}
	i, h, err := r.Execute()
	return i, parseError(err, h)
}

func (d *ServerlessClientDelegate) GetImport(ctx context.Context, clusterId string, id string) (*imp.Import, error) {
	i, h, err := d.sic.ImportServiceAPI.ImportServiceGetImport(ctx, clusterId, id).Execute()
	return i, parseError(err, h)
}

func (d *ServerlessClientDelegate) ListImports(ctx context.Context, clusterId string, pageSize *int32, pageToken, orderBy *string) (*imp.ListImportsResp, error) {
	r := d.sic.ImportServiceAPI.ImportServiceListImports(ctx, clusterId)
	if pageSize != nil {
		r = r.PageSize(*pageSize)
	}
	if pageToken != nil {
		r = r.PageToken(*pageToken)
	}
	if orderBy != nil {
		r = r.OrderBy(*orderBy)
	}
	is, h, err := r.Execute()
	return is, parseError(err, h)
}

func (d *ServerlessClientDelegate) GetBranch(ctx context.Context, clusterId, branchId string, view branch.BranchServiceGetBranchViewParameter) (*branch.Branch, error) {
	r := d.bc.BranchServiceAPI.BranchServiceGetBranch(ctx, clusterId, branchId)
	r = r.View(view)
	b, h, err := r.Execute()
	return b, parseError(err, h)
}

func (d *ServerlessClientDelegate) ListBranches(ctx context.Context, clusterId string, pageSize *int32, pageToken *string) (*branch.ListBranchesResponse, error) {
	r := d.bc.BranchServiceAPI.BranchServiceListBranches(ctx, clusterId)
	if pageSize != nil {
		r = r.PageSize(*pageSize)
	}
	if pageToken != nil {
		r = r.PageToken(*pageToken)
	}
	bs, h, err := r.Execute()
	return bs, parseError(err, h)
}

func (d *ServerlessClientDelegate) CreateBranch(ctx context.Context, clusterId string, body *branch.Branch) (*branch.Branch, error) {
	r := d.bc.BranchServiceAPI.BranchServiceCreateBranch(ctx, clusterId)
	if body != nil {
		r = r.Branch(*body)
	}
	b, h, err := r.Execute()
	return b, parseError(err, h)
}

func (d *ServerlessClientDelegate) DeleteBranch(ctx context.Context, clusterId string, branchId string) (*branch.Branch, error) {
	b, h, err := d.bc.BranchServiceAPI.BranchServiceDeleteBranch(ctx, clusterId, branchId).Execute()
	return b, parseError(err, h)
}

func (d *ServerlessClientDelegate) ResetBranch(ctx context.Context, clusterId string, branchId string) (*branch.Branch, error) {
	b, h, err := d.bc.BranchServiceAPI.BranchServiceResetBranch(ctx, clusterId, branchId).Execute()
	return b, parseError(err, h)
}
func (d *ServerlessClientDelegate) DeleteBackup(ctx context.Context, backupId string) (*br.V1beta1Backup, error) {
	b, h, err := d.brc.BackupRestoreServiceAPI.BackupRestoreServiceDeleteBackup(ctx, backupId).Execute()
	return b, parseError(err, h)
}

func (d *ServerlessClientDelegate) GetBackup(ctx context.Context, backupId string) (*br.V1beta1Backup, error) {
	b, h, err := d.brc.BackupRestoreServiceAPI.BackupRestoreServiceGetBackup(ctx, backupId).Execute()
	return b, parseError(err, h)
}

func (d *ServerlessClientDelegate) ListBackups(ctx context.Context, clusterId *string, pageSize *int32, pageToken *string) (*br.V1beta1ListBackupsResponse, error) {
	r := d.brc.BackupRestoreServiceAPI.BackupRestoreServiceListBackups(ctx)
	if clusterId != nil {
		r = r.ClusterId(*clusterId)
	}
	if pageSize != nil {
		r = r.PageSize(*pageSize)
	}
	if pageToken != nil {
		r = r.PageToken(*pageToken)
	}
	bs, h, err := r.Execute()
	return bs, parseError(err, h)
}

func (d *ServerlessClientDelegate) Restore(ctx context.Context, body *br.V1beta1RestoreRequest) (*br.V1beta1RestoreResponse, error) {
	r := d.brc.BackupRestoreServiceAPI.BackupRestoreServiceRestore(ctx)
	if body != nil {
		r = r.Body(*body)
	}
	bs, h, err := r.Execute()
	return bs, parseError(err, h)
}

func (d *ServerlessClientDelegate) StartUpload(ctx context.Context, clusterId string, fileName, targetDatabase, targetTable *string, partNumber *int32) (*imp.StartUploadResponse, error) {
	r := d.sic.ImportServiceAPI.ImportServiceStartUpload(ctx, clusterId)
	if fileName != nil {
		r = r.FileName(*fileName)
	}
	if targetDatabase != nil {
		r = r.TargetDatabase(*targetDatabase)
	}
	if targetTable != nil {
		r = r.TargetTable(*targetTable)
	}
	if partNumber != nil {
		r = r.PartNumber(*partNumber)
	}
	res, h, err := r.Execute()
	return res, parseError(err, h)
}

func (d *ServerlessClientDelegate) CompleteUpload(ctx context.Context, clusterId string, uploadId *string, parts *[]imp.CompletePart) error {
	r := d.sic.ImportServiceAPI.ImportServiceCompleteUpload(ctx, clusterId)
	if uploadId != nil {
		r = r.UploadId(*uploadId)
	}
	if parts != nil {
		r = r.Parts(*parts)
	}
	_, h, err := r.Execute()
	return parseError(err, h)
}

func (d *ServerlessClientDelegate) CancelUpload(ctx context.Context, clusterId string, uploadId *string) error {
	r := d.sic.ImportServiceAPI.ImportServiceCancelUpload(ctx, clusterId)
	if uploadId != nil {
		r = r.UploadId(*uploadId)
	}
	_, h, err := r.Execute()
	return parseError(err, h)
}

func (d *ServerlessClientDelegate) GetExport(ctx context.Context, clusterId string, exportId string) (*export.Export, error) {
	res, h, err := d.ec.ExportServiceAPI.ExportServiceGetExport(ctx, clusterId, exportId).Execute()
	return res, parseError(err, h)
}

func (d *ServerlessClientDelegate) CancelExport(ctx context.Context, clusterId string, exportId string) (*export.Export, error) {
	res, h, err := d.ec.ExportServiceAPI.ExportServiceCancelExport(ctx, clusterId, exportId).Execute()
	return res, parseError(err, h)
}

func (d *ServerlessClientDelegate) CreateExport(ctx context.Context, clusterId string, body *export.ExportServiceCreateExportBody) (*export.Export, error) {
	r := d.ec.ExportServiceAPI.ExportServiceCreateExport(ctx, clusterId)
	if body != nil {
		r = r.Body(*body)
	}
	res, h, err := r.Execute()
	return res, parseError(err, h)
}

func (d *ServerlessClientDelegate) DeleteExport(ctx context.Context, clusterId string, exportId string) (*export.Export, error) {
	res, h, err := d.ec.ExportServiceAPI.ExportServiceDeleteExport(ctx, clusterId, exportId).Execute()
	return res, parseError(err, h)
}

func (d *ServerlessClientDelegate) ListExports(ctx context.Context, clusterId string, pageSize *int32, pageToken *string, orderBy *string) (*export.ListExportsResponse, error) {
	r := d.ec.ExportServiceAPI.ExportServiceListExports(ctx, clusterId)
	if pageSize != nil {
		r = r.PageSize(*pageSize)
	}
	if pageToken != nil {
		r = r.PageToken(*pageToken)
	}
	if orderBy != nil {
		r = r.OrderBy(*orderBy)
	}
	res, h, err := r.Execute()
	return res, parseError(err, h)
}

func (d *ServerlessClientDelegate) ListExportFiles(ctx context.Context, clusterId string, exportId string, pageSize *int32,
	pageToken *string, isGenerateUrl bool) (*export.ListExportFilesResponse, error) {
	r := d.ec.ExportServiceAPI.ExportServiceListExportFiles(ctx, clusterId, exportId)
	if pageSize != nil {
		r = r.PageSize(*pageSize)
	}
	if pageToken != nil {
		r = r.PageToken(*pageToken)
	}
	if isGenerateUrl {
		r = r.GenerateUrl(isGenerateUrl)
	}
	res, h, err := r.Execute()
	return res, parseError(err, h)
}

func (d *ServerlessClientDelegate) DownloadExportFiles(ctx context.Context, clusterId string, exportId string, body *export.ExportServiceDownloadExportFilesBody) (*export.DownloadExportFilesResponse, error) {
	r := d.ec.ExportServiceAPI.ExportServiceDownloadExportFiles(ctx, clusterId, exportId)
	if body != nil {
		r = r.Body(*body)
	}
	res, h, err := r.Execute()
	return res, parseError(err, h)
}
