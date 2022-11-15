package tidbcloud

import (
	"fmt"
)

type TiDBCloudClient struct {
}

func NewTiDBCloudClient(publicKey, privateKey, providerVersion string) (*TiDBCloudClient, error) {
	initClient(publicKey, privateKey, providerVersion)
	c := TiDBCloudClient{}
	return &c, nil
}

// GetSpecifications returns all the available specifications
func (c *TiDBCloudClient) GetSpecifications() (*GetSpecificationsResp, error) {
	var (
		url    = fmt.Sprintf("%s/api/v1beta/clusters/provider/regions", host)
		result GetSpecificationsResp
	)

	_, err := doGET(url, nil, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetAllProjects  returns all the projects
func (c *TiDBCloudClient) GetAllProjects(page, pageSize int64) (*GetAllProjectsResp, error) {
	var (
		url    = fmt.Sprintf("%s/api/v1beta/projects?page=%d&page_size=%d", host, page, pageSize)
		result GetAllProjectsResp
	)

	_, err := doGET(url, nil, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetClusters returns all the clusters
func (c *TiDBCloudClient) GetClusters(projectId string, page, pageSize int64) (*GetAllClustersResp, error) {
	var (
		url    = fmt.Sprintf("%s/api/v1beta/projects/%s/clusters?page=%d&page_size=%d", host, projectId, page, pageSize)
		result GetAllClustersResp
	)

	_, err := doGET(url, nil, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateCluster create a cluster in the given project
func (c *TiDBCloudClient) CreateCluster(projectID string, clusterReq *CreateClusterReq) (*CreateClusterResp, error) {
	var (
		url    = fmt.Sprintf("%s/api/v1beta/projects/%s/clusters", host, projectID)
		result CreateClusterResp
	)

	_, err := doPOST(url, clusterReq, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetClusterById return detail status of given cluster
func (c *TiDBCloudClient) GetClusterById(projectID string, clusterID string) (*GetClusterResp, error) {
	var (
		url    = fmt.Sprintf("%s/api/v1beta/projects/%s/clusters/%s", host, projectID, clusterID)
		result GetClusterResp
	)

	_, err := doGET(url, nil, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteClusterById delete a cluster by the given ID
func (c *TiDBCloudClient) DeleteClusterById(projectID, clusterID string) error {
	url := fmt.Sprintf("%s/api/v1beta/projects/%s/clusters/%s", host, projectID, clusterID)
	_, err := doDELETE(url, nil, nil)
	if err != nil {
		return err
	}

	return nil
}

// UpdateClusterById can only scale out and set pause now
func (c *TiDBCloudClient) UpdateClusterById(projectID, clusterID string, updateClusterReq UpdateClusterReq) error {
	url := fmt.Sprintf("%s/api/v1beta/projects/%s/clusters/%s", host, projectID, clusterID)
	_, err := doPATCH(url, updateClusterReq, nil)
	if err != nil {
		return err
	}
	return nil
}

// CreateBackup can create a backup for the cluster
func (c *TiDBCloudClient) CreateBackup(projectID, clusterID string, req CreateBackupReq) (*CreateBackupResp, error) {
	var (
		url    = fmt.Sprintf("%s/api/v1beta/projects/%s/clusters/%s/backups", host, projectID, clusterID)
		result CreateBackupResp
	)

	_, err := doPOST(url, req, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetBackupById show the detail of the bakcup
func (c *TiDBCloudClient) GetBackupById(projectID, clusterID, backupID string) (*GetBackupResp, error) {
	var (
		url    = fmt.Sprintf("%s/api/v1beta/projects/%s/clusters/%s/backups/%s", host, projectID, clusterID, backupID)
		result GetBackupResp
	)

	_, err := doGET(url, nil, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteBackupById delete a backup
func (c *TiDBCloudClient) DeleteBackupById(projectID, clusterID, backupID string) error {
	url := fmt.Sprintf("%s/api/v1beta/projects/%s/clusters/%s/backups/%s", host, projectID, clusterID, backupID)
	_, err := doDELETE(url, nil, nil)
	if err != nil {
		return err
	}

	return nil
}

// GetBackups get all the backups
func (c *TiDBCloudClient) GetBackups(projectID, clusterID string, page, pageSize int64) (*GetBackupsResp, error) {
	var (
		url    = fmt.Sprintf("%s/api/v1beta/projects/%s/clusters/%s/backups?page=%d&page_size=%d", host, projectID, clusterID, page, pageSize)
		result GetBackupsResp
	)

	_, err := doGET(url, nil, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateRestoreTask create a restore task from a backup
func (c *TiDBCloudClient) CreateRestoreTask(projectID string, req CreateRestoreTaskReq) (*CreateRestoreTaskResp, error) {
	var (
		url    = fmt.Sprintf("%s/api/v1beta/projects/%s/restores", host, projectID)
		result CreateRestoreTaskResp
	)

	_, err := doPOST(url, req, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetRestoreTask show the details of the restore task
func (c *TiDBCloudClient) GetRestoreTask(projectID, restoreId string) (*GetRestoreTaskResp, error) {
	var (
		url    = fmt.Sprintf("%s/api/v1beta/projects/%s/restores/%s", host, projectID, restoreId)
		result GetRestoreTaskResp
	)

	_, err := doGET(url, nil, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetRestoreTasks get All the restore tasks
func (c *TiDBCloudClient) GetRestoreTasks(projectID string, page, pageSize int64) (*GetRestoreTasksResp, error) {
	var (
		url    = fmt.Sprintf("%s/api/v1beta/projects/%s/restores?page=%d&page_size=%d", host, projectID, page, pageSize)
		result GetRestoreTasksResp
	)

	_, err := doGET(url, nil, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
