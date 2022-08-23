package tidbcloud

type Project struct {
	Id              string `json:"id"`
	OrgId           string `json:"org_id"`
	Name            string `json:"name"`
	ClusterCount    int64  `json:"cluster_count"`
	UserCount       int64  `json:"user_count"`
	CreateTimestamp string `json:"create_timestamp"`
}

type ConnectionString struct {
	Standard   string `json:"standard"`
	VpcPeering string `json:"vpc_peering"`
}

type IPAccess struct {
	CIDR        string `json:"cidr"`
	Description string `json:"description"`
}

type ComponentTiDB struct {
	NodeSize     string `json:"node_size"`
	NodeQuantity int    `json:"node_quantity"`
}

type ComponentTiKV struct {
	NodeSize       string `json:"node_size"`
	StorageSizeGib int    `json:"storage_size_gib"`
	NodeQuantity   int    `json:"node_quantity"`
}

type ComponentTiFlash struct {
	NodeSize       string `json:"node_size"`
	StorageSizeGib int    `json:"storage_size_gib"`
	NodeQuantity   int    `json:"node_quantity"`
}

type Components struct {
	TiDB    ComponentTiDB     `json:"tidb"`
	TiKV    ComponentTiKV     `json:"tikv"`
	TiFlash *ComponentTiFlash `json:"tiflash,omitempty"`
}

type ClusterConfig struct {
	RootPassword string     `json:"root_password"`
	Port         int        `json:"port"`
	Components   Components `json:"components"`
	IPAccessList []IPAccess `json:"ip_access_list"`
}

type ClusterStatus struct {
	TidbVersion   string `json:"tidb_version"`
	ClusterStatus string `json:"cluster_status"`
}

type CreateClusterReq struct {
	Name          string        `json:"name"`
	ClusterType   string        `json:"cluster_type"`
	CloudProvider string        `json:"cloud_provider"`
	Region        string        `json:"region"`
	Config        ClusterConfig `json:"config"`
}

type CreateClusterResp struct {
	ClusterId uint64 `json:"id,string"`
	Message   string `json:"message"`
}

type UpdateClusterReq struct {
	Config UpdateClusterConfig `json:"config"`
}

type UpdateClusterConfig struct {
	Components Components `json:"components"`
}

type GetAllProjectsResp struct {
	Items []Project `json:"items"`
	Total int64     `json:"total"`
}

type GetClusterResp struct {
	Id                uint64           `json:"id,string"`
	ProjectId         uint64           `json:"project_id,string"`
	Name              string           `json:"name"`
	Port              int32            `json:"port"`
	TiDBVersion       string           `json:"tidb_version"`
	ClusterType       string           `json:"cluster_type"`
	CloudProvider     string           `json:"cloud_provider"`
	Region            string           `json:"region"`
	Status            ClusterStatus    `json:"status"`
	CreateTimestamp   string           `json:"create_timestamp"`
	Config            ClusterConfig    `json:"config"`
	ConnectionStrings ConnectionString `json:"connection_strings"`
}

type Specification struct {
	ClusterType   string `json:"cluster_type"`
	CloudProvider string `json:"cloud_provider"`
	Region        string `json:"region"`
	Tidb          []struct {
		NodeSize          string `json:"node_size"`
		NodeQuantityRange struct {
			Min  int `json:"min"`
			Step int `json:"step"`
		} `json:"node_quantity_range"`
	} `json:"tidb"`
	Tikv []struct {
		NodeSize          string `json:"node_size"`
		NodeQuantityRange struct {
			Min  int `json:"min"`
			Step int `json:"step"`
		} `json:"node_quantity_range"`
		StorageSizeGibRange struct {
			Min int `json:"min"`
			Max int `json:"max"`
		} `json:"storage_size_gib_range"`
	} `json:"tikv"`
	Tiflash []struct {
		NodeSize          string `json:"node_size"`
		NodeQuantityRange struct {
			Min  int `json:"min"`
			Step int `json:"step"`
		} `json:"node_quantity_range"`
		StorageSizeGibRange struct {
			Min int `json:"min"`
			Max int `json:"max"`
		} `json:"storage_size_gib_range"`
	} `json:"tiflash"`
}

type GetSpecificationsResp struct {
	Items []Specification `json:"items"`
}

type CreateBackupResp struct {
	BackupId string `json:"id"`
}

type CreateBackupReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type GetBackupResp struct {
	Id              string `json:"id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Type            string `json:"type"`
	Size            string `json:"size"`
	Status          string `json:"status"`
	CreateTimestamp string `json:"create_timestamp"`
}

type GetBackupsResp struct {
	Items []GetBackupResp `json:"items"`
	Total int64           `json:"total"`
}

type CreateRestoreTaskReq struct {
	BackupId string        `json:"backup_id"`
	Name     string        `json:"name"`
	Config   ClusterConfig `json:"config"`
}

type CreateRestoreTaskResp struct {
	Id        string `json:"id"`
	ClusterId string `json:"cluster_id"`
}

type GetRestoreTaskResp struct {
	Id              string      `json:"id"`
	CreateTimestamp string      `json:"create_timestamp"`
	BackupId        string      `json:"backup_id"`
	ClusterId       string      `json:"cluster_id"`
	Status          string      `json:"status"`
	Cluster         ClusterInfo `json:"cluster"`
	ErrorMessage    string      `json:"error_message"`
}

type ClusterInfo struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type GetRestoreTasksResp struct {
	Items []GetRestoreTaskResp `json:"items"`
	Total int64                `json:"total"`
}
