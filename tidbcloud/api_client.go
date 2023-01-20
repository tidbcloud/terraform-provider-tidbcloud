package tidbcloud

import (
	"fmt"
	"github.com/c4pt0r/go-tidbcloud-sdk-v1/client/backup"
	"github.com/c4pt0r/go-tidbcloud-sdk-v1/client/restore"
	"net/http"
	"net/url"
	"os"

	apiClient "github.com/c4pt0r/go-tidbcloud-sdk-v1/client"
	"github.com/c4pt0r/go-tidbcloud-sdk-v1/client/cluster"
	"github.com/c4pt0r/go-tidbcloud-sdk-v1/client/project"
	httpTransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/icholy/digest"
)

const (
	DefaultApiUrl = "https://api.tidbcloud.com"
	userAgent     = "User-Agent"
)

type TiDBCloudClient interface {
	CreateCluster(params *cluster.CreateClusterParams, opts ...cluster.ClientOption) (*cluster.CreateClusterOK, error)

	UpdateCluster(params *cluster.UpdateClusterParams, opts ...cluster.ClientOption) (*cluster.UpdateClusterOK, error)

	DeleteCluster(params *cluster.DeleteClusterParams, opts ...cluster.ClientOption) (*cluster.DeleteClusterOK, error)

	GetCluster(params *cluster.GetClusterParams, opts ...cluster.ClientOption) (*cluster.GetClusterOK, error)

	ListClustersOfProject(params *cluster.ListClustersOfProjectParams, opts ...cluster.ClientOption) (*cluster.ListClustersOfProjectOK, error)

	ListProviderRegions(params *cluster.ListProviderRegionsParams, opts ...cluster.ClientOption) (*cluster.ListProviderRegionsOK, error)

	ListProjects(params *project.ListProjectsParams, opts ...project.ClientOption) (*project.ListProjectsOK, error)

	CreateBackup(params *backup.CreateBackupParams, opts ...backup.ClientOption) (*backup.CreateBackupOK, error)

	DeleteBackup(params *backup.DeleteBackupParams, opts ...backup.ClientOption) (*backup.DeleteBackupOK, error)

	GetBackupOfCluster(params *backup.GetBackupOfClusterParams, opts ...backup.ClientOption) (*backup.GetBackupOfClusterOK, error)

	ListBackUpOfCluster(params *backup.ListBackUpOfClusterParams, opts ...backup.ClientOption) (*backup.ListBackUpOfClusterOK, error)

	CreateRestoreTask(params *restore.CreateRestoreTaskParams, opts ...restore.ClientOption) (*restore.CreateRestoreTaskOK, error)

	GetRestoreTask(params *restore.GetRestoreTaskParams, opts ...restore.ClientOption) (*restore.GetRestoreTaskOK, error)

	ListRestoreTasks(params *restore.ListRestoreTasksParams, opts ...restore.ClientOption) (*restore.ListRestoreTasksOK, error)

	PreSignedUrlUpload(url *string, uploadFile *os.File, size int64) error
}

type ClientDelegate struct {
	c *apiClient.GoTidbcloud
}

func NewClientDelegate(publicKey string, privateKey string, apiUrl string, ver string) (*ClientDelegate, error) {
	c, err := NewApiClient(publicKey, privateKey, apiUrl, ver)
	if err != nil {
		return nil, err
	}
	return &ClientDelegate{
		c: c,
	}, nil
}

func (d *ClientDelegate) CreateCluster(params *cluster.CreateClusterParams, opts ...cluster.ClientOption) (*cluster.CreateClusterOK, error) {
	return d.c.Cluster.CreateCluster(params, opts...)
}

func (d *ClientDelegate) UpdateCluster(params *cluster.UpdateClusterParams, opts ...cluster.ClientOption) (*cluster.UpdateClusterOK, error) {
	return d.c.Cluster.UpdateCluster(params, opts...)
}

func (d *ClientDelegate) DeleteCluster(params *cluster.DeleteClusterParams, opts ...cluster.ClientOption) (*cluster.DeleteClusterOK, error) {
	return d.c.Cluster.DeleteCluster(params, opts...)
}

func (d *ClientDelegate) GetCluster(params *cluster.GetClusterParams, opts ...cluster.ClientOption) (*cluster.GetClusterOK, error) {
	return d.c.Cluster.GetCluster(params, opts...)
}

func (d *ClientDelegate) ListProviderRegions(params *cluster.ListProviderRegionsParams, opts ...cluster.ClientOption) (*cluster.ListProviderRegionsOK, error) {
	return d.c.Cluster.ListProviderRegions(params, opts...)
}

func (d *ClientDelegate) ListClustersOfProject(params *cluster.ListClustersOfProjectParams, opts ...cluster.ClientOption) (*cluster.ListClustersOfProjectOK, error) {
	return d.c.Cluster.ListClustersOfProject(params, opts...)
}

func (d *ClientDelegate) ListProjects(params *project.ListProjectsParams, opts ...project.ClientOption) (*project.ListProjectsOK, error) {
	return d.c.Project.ListProjects(params, opts...)
}

func (d *ClientDelegate) CreateBackup(params *backup.CreateBackupParams, opts ...backup.ClientOption) (*backup.CreateBackupOK, error) {
	return d.c.Backup.CreateBackup(params, opts...)
}

func (d *ClientDelegate) DeleteBackup(params *backup.DeleteBackupParams, opts ...backup.ClientOption) (*backup.DeleteBackupOK, error) {
	return d.c.Backup.DeleteBackup(params, opts...)
}

func (d *ClientDelegate) GetBackupOfCluster(params *backup.GetBackupOfClusterParams, opts ...backup.ClientOption) (*backup.GetBackupOfClusterOK, error) {
	return d.c.Backup.GetBackupOfCluster(params, opts...)
}

func (d *ClientDelegate) ListBackUpOfCluster(params *backup.ListBackUpOfClusterParams, opts ...backup.ClientOption) (*backup.ListBackUpOfClusterOK, error) {
	return d.c.Backup.ListBackUpOfCluster(params, opts...)
}

func (d *ClientDelegate) CreateRestoreTask(params *restore.CreateRestoreTaskParams, opts ...restore.ClientOption) (*restore.CreateRestoreTaskOK, error) {
	return d.c.Restore.CreateRestoreTask(params, opts...)
}

func (d *ClientDelegate) GetRestoreTask(params *restore.GetRestoreTaskParams, opts ...restore.ClientOption) (*restore.GetRestoreTaskOK, error) {
	return d.c.Restore.GetRestoreTask(params, opts...)
}

func (d *ClientDelegate) ListRestoreTasks(params *restore.ListRestoreTasksParams, opts ...restore.ClientOption) (*restore.ListRestoreTasksOK, error) {
	return d.c.Restore.ListRestoreTasks(params, opts...)
}

func (d *ClientDelegate) PreSignedUrlUpload(url *string, uploadFile *os.File, size int64) error {
	request, err := http.NewRequest("PUT", *url, uploadFile)
	if err != nil {
		return err
	}
	request.ContentLength = size

	putRes, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer putRes.Body.Close()

	if putRes.StatusCode != http.StatusOK {
		return fmt.Errorf("upload file failed : %s, %s", putRes.Status, putRes.Body)
	}

	return nil
}

func NewApiClient(publicKey string, privateKey string, apiUrl string, ver string) (*apiClient.GoTidbcloud, error) {
	httpclient := &http.Client{
		Transport: NewTransportWithAgent(&digest.Transport{
			Username: publicKey,
			Password: privateKey,
		}, fmt.Sprintf("terraform-provider-tidbcloud/%s", ver)),
	}

	// Parse the URL
	u, err := url.ParseRequestURI(apiUrl)
	if err != nil {
		return nil, err
	}

	transport := httpTransport.NewWithClient(u.Host, u.Path, []string{u.Scheme}, httpclient)
	return apiClient.New(transport, strfmt.Default), nil
}

// NewTransportWithAgent returns a new http.RoundTripper that add the User-Agent header,
// according to https://github.com/go-swagger/go-swagger/issues/1563.
func NewTransportWithAgent(inner http.RoundTripper, userAgent string) http.RoundTripper {
	return &UserAgentTransport{
		inner: inner,
		Agent: userAgent,
	}
}

type UserAgentTransport struct {
	inner http.RoundTripper
	Agent string
}

func (ug *UserAgentTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set(userAgent, ug.Agent)
	return ug.inner.RoundTrip(r)
}
