package tidbcloud

import (
	"fmt"
)

type TiDBCloudClient struct {
}

func NewTiDBCloudClient(publicKey, privateKey string) (*TiDBCloudClient, error) {
	initClient(publicKey, privateKey)
	c := TiDBCloudClient{}
	return &c, nil
}

// getSpecifications returns all the available specifications
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

func (c *TiDBCloudClient) GetDedicatedSpec(specifications *GetSpecificationsResp) (*Specification, error) {
	for _, i := range specifications.Items {
		if i.ClusterType == "DEDICATED" {
			return &i, nil
		}
	}

	return nil, fmt.Errorf("No specification found")
}

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

// createDedicatedCluster create a cluster in the given project
func (c *TiDBCloudClient) CreateDedicatedCluster(projectID string, spec *Specification) (*CreateClusterResp, error) {
	var (
		url    = fmt.Sprintf("%s/api/v1beta/projects/%s/clusters", host, projectID)
		result CreateClusterResp
	)

	// We have check the boundary in main function
	tidbSpec := spec.Tidb[0]
	tikvSpec := spec.Tikv[0]

	payload := CreateClusterReq{
		Name:          "tidbcloud-sample-1", // NOTE change to your cluster name
		ClusterType:   spec.ClusterType,
		CloudProvider: spec.CloudProvider,
		Region:        spec.Region,
		Config: ClusterConfig{
			RootPassword: "your secret password", // NOTE change to your cluster password, we generate a random password here
			Port:         4000,
			Components: Components{
				TiDB: ComponentTiDB{
					NodeSize:     tidbSpec.NodeSize,
					NodeQuantity: tidbSpec.NodeQuantityRange.Min,
				},
				TiKV: ComponentTiKV{
					NodeSize:       tikvSpec.NodeSize,
					StorageSizeGib: tikvSpec.StorageSizeGibRange.Min,
					NodeQuantity:   tikvSpec.NodeQuantityRange.Min,
				},
			},
			IPAccessList: []IPAccess{
				{
					CIDR:        "0.0.0.0/0",
					Description: "Allow Access from Anywhere.",
				},
			},
		},
	}

	_, err := doPOST(url, payload, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// createCuster create a cluster in the given project
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

// getClusterByID return detail status of given cluster
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

// deleteClusterByID delete a cluster
func (c *TiDBCloudClient) DeleteClusterById(projectID, clusterID string) error {
	url := fmt.Sprintf("%s/api/v1beta/projects/%s/clusters/%s", host, projectID, clusterID)
	_, err := doDELETE(url, nil, nil)
	if err != nil {
		return err
	}

	return nil
}

// updateClusterByID delete a cluster
func (c *TiDBCloudClient) UpdateClusterById(projectID, clusterID string, components Components) error {
	url := fmt.Sprintf("%s/api/v1beta/projects/%s/clusters/%s", host, projectID, clusterID)
	payload := UpdateClusterReq{
		Config: UpdateClusterConfig{
			Components: components,
		},
	}
	resp, err := doPATCH(url, payload, nil)
	if err != nil {
		return err
	}
	print(resp)
	return nil
}

// create a backup
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

// getBackupByID
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

// deleteBackupByID
func (c *TiDBCloudClient) DeleteBackupById(projectID, clusterID, backupID string) error {
	url := fmt.Sprintf("%s/api/v1beta/projects/%s/clusters/%s/backups/%s", host, projectID, clusterID, backupID)
	_, err := doDELETE(url, nil, nil)
	if err != nil {
		return err
	}

	return nil
}

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
