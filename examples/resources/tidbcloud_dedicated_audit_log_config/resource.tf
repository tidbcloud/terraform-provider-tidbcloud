variable "cluster_id" {
  type     = string
  nullable = false
}

variable "bucket_uri" {
  type     = string
  nullable = false
}

variable "bucket_region_id" {
  type     = string
  nullable = false
}

variable "aws_role_arn" {
  type     = string
  nullable = true
}

variable "azure_sas_token" {
  type     = string
  nullable = true
}

resource "tidbcloud_dedicated_audit_log_config" "example" {
  enabled          = false
  cluster_id       = var.cluster_id
  bucket_uri       = var.bucket_uri
  bucket_region_id = var.bucket_region_id
  aws_role_arn     = var.aws_role_arn
  azure_sas_token  = var.azure_sas_token
}