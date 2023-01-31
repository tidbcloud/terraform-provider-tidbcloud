terraform {
  required_providers {
    tidbcloud = {
      source = "tidbcloud/tidbcloud"
    }
  }
}

provider "tidbcloud" {
  public_key  = "fake_public_key"
  private_key = "fake_private_key"
}

resource "tidbcloud_import" "example_local" {
  project_id  = "fake_id"
  cluster_id  = "fake_id"
  type        = "LOCAL"
  data_format = "CSV"
  target_table = {
    schema = "test"
    table  = "t"
  }
  file_name = "fake_path"
}

resource "tidbcloud_import" "example_s3_csv" {
  project_id   = "fake_id"
  cluster_id   = "fake_id"
  type         = "S3"
  data_format  = "CSV"
  aws_role_arn = "fake_arn"
  source_url   = "fake_url"
}

resource "tidbcloud_import" "example_s3_parquet" {
  project_id   = "1369847559691367867"
  cluster_id   = "1373933076658240623"
  type         = "S3"
  data_format  = "Parquet"
  aws_role_arn = "fake_arn"
  source_url   = "fake_url"
}