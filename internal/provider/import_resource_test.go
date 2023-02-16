package provider

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	mockClient "github.com/tidbcloud/terraform-provider-tidbcloud/mock"
	importService "github.com/tidbcloud/terraform-provider-tidbcloud/pkg/import/client/import_service"
	importModel "github.com/tidbcloud/terraform-provider-tidbcloud/pkg/import/models"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
	"os"
	"testing"
)

// Please Fill pass the projectID and clusterID and fill in the file_name to run the acc test
func TestACCImportResourceLOCAL(t *testing.T) {
	config := fmt.Sprintf(`
		resource "tidbcloud_import" "local" {
		  project_id  = "%s"
		  cluster_id  = "%s"
		  type        = "LOCAL"
		  data_format = "CSV"
		  target_table = {
			schema = "test"
			table  = "import_test_%s"
		  }
		  file_name = "fake_file"
		}
		`, os.Getenv(TiDBCloudProjectID), os.Getenv(TiDBCloudClusterID), GenerateRandomString(3))
	testImportResourceLocal(t, config, false)
}

// Please Fill pass the projectID and clusterID and fill in the aws_role_arn and source url to run the acc test
func TestACCImportResourceS3(t *testing.T) {
	config := fmt.Sprintf(`
		resource "tidbcloud_import" "s3" {
		  project_id  = "%s"
		  cluster_id  = "%s"
		  type        = "S3"
		  data_format = "CSV"
          aws_role_arn = "fake_arn"
          source_url   = "fake_url"
		}
		`, os.Getenv(TiDBCloudProjectID), os.Getenv(TiDBCloudClusterID))
	testImportResourceS3(t, config, false)
}

func TestUTImportResourceLOCAL(t *testing.T) {
	if os.Getenv(TiDBCloudPublicKey) == "" {
		os.Setenv(TiDBCloudPublicKey, "fake")
	}
	if os.Getenv(TiDBCloudPrivateKey) == "" {
		os.Setenv(TiDBCloudPrivateKey, "fake")
	}
	if os.Getenv(TiDBCloudProjectID) == "" {
		os.Setenv(TiDBCloudProjectID, "fake")
	}

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudClient(ctrl)

	defer HookGlobal(&NewClient, func(publicKey string, privateKey string, apiUrl string, userAgent string) (tidbcloud.TiDBCloudClient, error) {
		return s, nil
	})()

	clusterId := "cluster-id"
	importId := "import-id"
	fileName := "fake.csv"
	file, _ := os.Create(fileName)
	defer file.Close()

	createImportOK := &importService.CreateImportOK{
		Payload: &importModel.OpenapiCreateImportResp{
			ID: &importId,
		},
	}

	getImportResp := importModel.OpenapiGetImportResp{}
	getImportResp.UnmarshalBinary([]byte(fmt.Sprintf(`{
    "cluster_id": "%s",
    "total_size": "20",
    "total_files": 0,
    "source_url": "",
    "completed_tables": 1,
    "pending_tables": 0,
    "created_at": "2023-01-31T05:27:50Z",
    "status": "COMPLETED",
    "completed_percent": 100,
    "current_tables": [],
    "data_format": "CSV",
    "message": "",
    "elapsed_time_seconds": 35,
    "id": "%s",
    "processed_source_data_size": "20",
    "total_tables_count": 1,
    "post_import_completed_percent": 100,
    "all_completed_tables": [
        {
            "table_name": "test.r",
            "result": "SUCCESS",
            "message": ""
        }
    ],
    "creation_details": {
        "project_id": "%s",
        "cluster_id": "%s",
        "type": "LOCAL",
        "data_format": "CSV",
        "csv_format": {
            "separator": ",",
            "delimiter": "\"",
            "header": true,
            "not_null": false,
            "null": "",
            "backslash_escape": true,
            "trim_last_separator": false
        },
        "source_url": "",
        "aws_role_arn": "",
        "file_name": "fake.csv",
        "target_table": {
            "database": "test",
            "table": "r"
        }
    }}`, clusterId, importId, os.Getenv(TiDBCloudProjectID), clusterId)))

	getImportOK := &importService.GetImportOK{
		Payload: &getImportResp,
	}

	cancelImportOK := &importService.CancelImportOK{}

	generateUploadURL := &importService.GenerateUploadURLOK{
		Payload: &importModel.OpenapiGenerateUploadURLResq{
			NewFileName: Ptr("fake_new_file_name"),
			UploadURL:   Ptr("fake_upload_url"),
		},
	}

	s.EXPECT().CreateImport(gomock.Any()).
		Return(createImportOK, nil)
	s.EXPECT().GetImport(gomock.Any()).
		Return(getImportOK, nil).MinTimes(1).MaxTimes(3)
	s.EXPECT().CancelImport(gomock.Any()).
		Return(cancelImportOK, nil)
	s.EXPECT().GenerateUploadURL(gomock.Any()).
		Return(generateUploadURL, nil)
	s.EXPECT().PreSignedUrlUpload(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	config := fmt.Sprintf(`
		resource "tidbcloud_import" "local" {
		  project_id  = "%s"
		  cluster_id  = "cluster-id"
		  type        = "LOCAL"
		  data_format = "CSV"
		  target_table = {
			database = "test"
			table  = "r"
		  }
		  file_name = "fake.csv"
		}
		`, os.Getenv(TiDBCloudProjectID))
	testImportResourceLocal(t, config, true)
}

func testImportResourceLocal(t *testing.T, config string, useMock bool) {
	resource.Test(t, resource.TestCase{
		IsUnitTest:               useMock,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read import resource
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("tidbcloud_import.local", "id"),
					resource.TestCheckResourceAttr("tidbcloud_import.local", "type", "LOCAL"),
					resource.TestCheckResourceAttrSet("tidbcloud_import.local", "file_name"),
					resource.TestCheckResourceAttrSet("tidbcloud_import.local", "new_file_name"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestUTImportResourceS3(t *testing.T) {
	if os.Getenv(TiDBCloudPublicKey) == "" {
		os.Setenv(TiDBCloudPublicKey, "fake")
	}
	if os.Getenv(TiDBCloudPrivateKey) == "" {
		os.Setenv(TiDBCloudPrivateKey, "fake")
	}
	if os.Getenv(TiDBCloudProjectID) == "" {
		os.Setenv(TiDBCloudProjectID, "fake")
	}

	ctrl := gomock.NewController(t)
	s := mockClient.NewMockTiDBCloudClient(ctrl)

	defer HookGlobal(&NewClient, func(publicKey string, privateKey string, apiUrl string, userAgent string) (tidbcloud.TiDBCloudClient, error) {
		return s, nil
	})()

	clusterId := "cluster-id"
	importId := "import-id"

	createImportOK := &importService.CreateImportOK{
		Payload: &importModel.OpenapiCreateImportResp{
			ID: &importId,
		},
	}

	getImportResp := importModel.OpenapiGetImportResp{}
	getImportResp.UnmarshalBinary([]byte(fmt.Sprintf(`{
    "cluster_id": "%s",
    "total_size": "20",
    "total_files": 0,
    "source_url": "fake_source_url",
    "completed_tables": 1,
    "pending_tables": 0,
    "created_at": "2023-01-31T05:27:50Z",
    "status": "COMPLETED",
    "completed_percent": 100,
    "current_tables": [],
    "data_format": "CSV",
    "message": "",
    "elapsed_time_seconds": 35,
    "id": "%s",
    "processed_source_data_size": "20",
    "total_tables_count": 1,
    "post_import_completed_percent": 100,
    "all_completed_tables": [],
    "creation_details": {
        "project_id": "%s",
        "cluster_id": "%s",
        "type": "S3",
        "data_format": "CSV",
        "csv_format": {
            "separator": ",",
            "delimiter": "\"",
            "header": true,
            "not_null": false,
            "null": "",
            "backslash_escape": true,
            "trim_last_separator": false
        },
        "source_url": "fake_url",
        "aws_role_arn": "fake_aws_role_arn",
        "file_name": "",
        "target_table": null
    }}`, clusterId, importId, os.Getenv(TiDBCloudProjectID), clusterId)))

	getImportOK := &importService.GetImportOK{
		Payload: &getImportResp,
	}

	cancelImportOK := &importService.CancelImportOK{}

	generateUploadURL := &importService.GenerateUploadURLOK{
		Payload: &importModel.OpenapiGenerateUploadURLResq{
			NewFileName: Ptr("fake_new_file_name"),
			UploadURL:   Ptr("fake_upload_url"),
		},
	}

	s.EXPECT().CreateImport(gomock.Any()).
		Return(createImportOK, nil).AnyTimes()
	s.EXPECT().GetImport(gomock.Any()).
		Return(getImportOK, nil).AnyTimes()
	s.EXPECT().CancelImport(gomock.Any()).
		Return(cancelImportOK, nil).AnyTimes()
	s.EXPECT().GenerateUploadURL(gomock.Any()).
		Return(generateUploadURL, nil).AnyTimes()
	s.EXPECT().PreSignedUrlUpload(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil).AnyTimes()

	config := fmt.Sprintf(`
		resource "tidbcloud_import" "s3" {
		  project_id  = "%s"
		  cluster_id  = "cluster-id"
		  type        = "S3"
		  data_format  = "Parquet"
		  aws_role_arn = "fake_arn"
		  source_url   = "fake_url"
		}
		`, os.Getenv(TiDBCloudProjectID))

	testImportResourceS3(t, config, true)
}

func testImportResourceS3(t *testing.T, config string, useMock bool) {
	resource.Test(t, resource.TestCase{
		IsUnitTest:               useMock,
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read import resource
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("tidbcloud_import.s3", "id"),
					resource.TestCheckResourceAttr("tidbcloud_import.s3", "type", "S3"),
					resource.TestCheckResourceAttrSet("tidbcloud_import.s3", "source_url"),
					resource.TestCheckResourceAttrSet("tidbcloud_import.s3", "aws_role_arn"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
