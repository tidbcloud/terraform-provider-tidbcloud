


# Internal OpenAPIs for TiDB Cloud
The TiDB Cloud API uses HTTP Digest Authentication. It protects your private key from being sent over the network.The API key contains a public key and a private key, which act as the username and password required in the HTTP Digest Authentication. The private key only displays upon the key creation.
  

## Informations

### Version

alpha

## Tags

  ### <span id="tag-import-service"></span>ImportService

## Content negotiation

### URI Schemes
  * https

### Consumes
  * application/json

### Produces
  * application/json

## All endpoints

###  import_service

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| DELETE | /api/internal/projects/{project_id}/clusters/{cluster_id}/imports/{id} | [cancel import](#cancel-import) | Cancel an import job. |
| POST | /api/internal/projects/{project_id}/clusters/{cluster_id}/imports | [create import](#create-import) | Create an import job. |
| POST | /api/internal/projects/{project_id}/clusters/{cluster_id}/upload_url | [generate upload URL](#generate-upload-url) | Generate S3 url to upload file. |
| GET | /api/internal/projects/{project_id}/clusters/{cluster_id}/imports/{id} | [get import](#get-import) | Get an import job. |
| GET | /api/internal/projects/{project_id}/clusters/{cluster_id}/imports | [list imports](#list-imports) | List all import jobs in the cluster. |
  


## Paths

### <span id="cancel-import"></span> Cancel an import job. (*CancelImport*)

```
DELETE /api/internal/projects/{project_id}/clusters/{cluster_id}/imports/{id}
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| cluster_id | `path` | uint64 (formatted string) | `string` |  | ✓ |  | The ID of the cluster. |
| id | `path` | uint64 (formatted string) | `string` |  | ✓ |  | The ID of the import job. |
| project_id | `path` | uint64 (formatted string) | `string` |  | ✓ |  | The ID of the project. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#cancel-import-200) | OK | A successful response. |  | [schema](#cancel-import-200-schema) |
| [default](#cancel-import-default) | | An unexpected error response. |  | [schema](#cancel-import-default-schema) |

#### Responses


##### <span id="cancel-import-200"></span> 200 - A successful response.
Status: OK

###### <span id="cancel-import-200-schema"></span> Schema
   
  

any

##### <span id="cancel-import-default"></span> Default Response
An unexpected error response.

###### <span id="cancel-import-default-schema"></span> Schema

  

[GooglerpcStatus](#googlerpc-status)

### <span id="create-import"></span> Create an import job. (*CreateImport*)

```
POST /api/internal/projects/{project_id}/clusters/{cluster_id}/imports
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| cluster_id | `path` | uint64 (formatted string) | `string` |  | ✓ |  | The ID of the cluster. |
| project_id | `path` | uint64 (formatted string) | `string` |  | ✓ |  | The ID of the project. |
| body | `body` | [CreateImportBody](#create-import-body) | `CreateImportBody` | | ✓ | |  |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#create-import-200) | OK | A successful response. |  | [schema](#create-import-200-schema) |
| [default](#create-import-default) | | An unexpected error response. |  | [schema](#create-import-default-schema) |

#### Responses


##### <span id="create-import-200"></span> 200 - A successful response.
Status: OK

###### <span id="create-import-200-schema"></span> Schema
   
  

[OpenapiCreateImportResp](#openapi-create-import-resp)

##### <span id="create-import-default"></span> Default Response
An unexpected error response.

###### <span id="create-import-default-schema"></span> Schema

  

[GooglerpcStatus](#googlerpc-status)

###### Inlined models

**<span id="create-import-body"></span> CreateImportBody**


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| aws_role_arn | string| `string` |  | | The arn of AWS IAM role. |  |
| csv_format | [OpenapiCustomCSVFormat](#openapi-custom-c-s-v-format)| `models.OpenapiCustomCSVFormat` |  | | The CSV configuration. |  |
| data_format | [OpenapiDataFormat](#openapi-data-format)| `models.OpenapiDataFormat` | ✓ | | The format of data to import. |  |
| file_name | string| `string` |  | | The file name returned by generating upload url. |  |
| source_url | string| `string` |  | | The full s3 path that contains data to import. |  |
| target_table | [OpenapiTable](#openapi-table)| `models.OpenapiTable` |  | | The target db and table to import data. |  |
| type | [CreateImportReqImportType](#create-import-req-import-type)| `models.CreateImportReqImportType` | ✓ | | The type of data source. |  |



### <span id="generate-upload-url"></span> Generate S3 url to upload file. (*GenerateUploadURL*)

```
POST /api/internal/projects/{project_id}/clusters/{cluster_id}/upload_url
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| cluster_id | `path` | uint64 (formatted string) | `string` |  | ✓ |  | The ID of the cluster. |
| project_id | `path` | uint64 (formatted string) | `string` |  | ✓ |  | The ID of the project. |
| body | `body` | [GenerateUploadURLBody](#generate-upload-url-body) | `GenerateUploadURLBody` | | ✓ | |  |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#generate-upload-url-200) | OK | A successful response. |  | [schema](#generate-upload-url-200-schema) |
| [default](#generate-upload-url-default) | | An unexpected error response. |  | [schema](#generate-upload-url-default-schema) |

#### Responses


##### <span id="generate-upload-url-200"></span> 200 - A successful response.
Status: OK

###### <span id="generate-upload-url-200-schema"></span> Schema
   
  

[OpenapiGenerateUploadURLResq](#openapi-generate-upload-url-resq)

##### <span id="generate-upload-url-default"></span> Default Response
An unexpected error response.

###### <span id="generate-upload-url-default-schema"></span> Schema

  

[GooglerpcStatus](#googlerpc-status)

###### Inlined models

**<span id="generate-upload-url-body"></span> GenerateUploadURLBody**


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| content_length | int64 (formatted string)| `string` | ✓ | |  |  |
| file_name | string| `string` | ✓ | |  |  |



### <span id="get-import"></span> Get an import job. (*GetImport*)

```
GET /api/internal/projects/{project_id}/clusters/{cluster_id}/imports/{id}
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| cluster_id | `path` | uint64 (formatted string) | `string` |  | ✓ |  | The ID of the cluster. |
| id | `path` | uint64 (formatted string) | `string` |  | ✓ |  | The ID of the import job. |
| project_id | `path` | uint64 (formatted string) | `string` |  | ✓ |  | The ID of the project. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#get-import-200) | OK | A successful response. |  | [schema](#get-import-200-schema) |
| [default](#get-import-default) | | An unexpected error response. |  | [schema](#get-import-default-schema) |

#### Responses


##### <span id="get-import-200"></span> 200 - A successful response.
Status: OK

###### <span id="get-import-200-schema"></span> Schema
   
  

[OpenapiGetImportResp](#openapi-get-import-resp)

##### <span id="get-import-default"></span> Default Response
An unexpected error response.

###### <span id="get-import-default-schema"></span> Schema

  

[GooglerpcStatus](#googlerpc-status)

### <span id="list-imports"></span> List all import jobs in the cluster. (*ListImports*)

```
GET /api/internal/projects/{project_id}/clusters/{cluster_id}/imports
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| cluster_id | `path` | uint64 (formatted string) | `string` |  | ✓ |  | The ID of the cluster. |
| project_id | `path` | uint64 (formatted string) | `string` |  | ✓ |  | The ID of the project. |
| page | `query` | int32 (formatted integer) | `int32` |  |  | `1` | The number of pages. |
| page_size | `query` | int32 (formatted integer) | `int32` |  |  | `10` | The size of a page. |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#list-imports-200) | OK | A successful response. |  | [schema](#list-imports-200-schema) |
| [default](#list-imports-default) | | An unexpected error response. |  | [schema](#list-imports-default-schema) |

#### Responses


##### <span id="list-imports-200"></span> 200 - A successful response.
Status: OK

###### <span id="list-imports-200-schema"></span> Schema
   
  

[OpenapiListImportsResp](#openapi-list-imports-resp)

##### <span id="list-imports-default"></span> Default Response
An unexpected error response.

###### <span id="list-imports-default-schema"></span> Schema

  

[GooglerpcStatus](#googlerpc-status)

## Models

### <span id="create-import-req-import-type"></span> CreateImportReqImportType


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| CreateImportReqImportType | string| string | |  |  |



### <span id="import-table-completion-info-result"></span> ImportTableCompletionInfoResult


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| ImportTableCompletionInfoResult | string| string | |  |  |



### <span id="googlerpc-status"></span> googlerpcStatus


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| code | int32 (formatted integer)| `int32` |  | |  |  |
| details | [][ProtobufAny](#protobuf-any)| `[]*ProtobufAny` |  | |  |  |
| message | string| `string` |  | |  |  |



### <span id="openapi-create-import-req"></span> openapiCreateImportReq


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| aws_role_arn | string| `string` |  | | The arn of AWS IAM role. |  |
| cluster_id | uint64 (formatted string)| `string` | ✓ | | The ID of the cluster. | `1` |
| csv_format | [OpenapiCustomCSVFormat](#openapi-custom-c-s-v-format)| `OpenapiCustomCSVFormat` |  | | The CSV configuration. |  |
| data_format | [OpenapiDataFormat](#openapi-data-format)| `OpenapiDataFormat` | ✓ | | The format of data to import. |  |
| file_name | string| `string` |  | | The file name returned by generating upload url. |  |
| project_id | uint64 (formatted string)| `string` | ✓ | | The ID of the project. | `1` |
| source_url | string| `string` |  | | The full s3 path that contains data to import. |  |
| target_table | [OpenapiTable](#openapi-table)| `OpenapiTable` |  | | The target db and table to import data. |  |
| type | [CreateImportReqImportType](#create-import-req-import-type)| `CreateImportReqImportType` | ✓ | | The type of data source. |  |



### <span id="openapi-create-import-resp"></span> openapiCreateImportResp


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| id | uint64 (formatted string)| `string` | ✓ | | The ID of the import job. | `1` |



### <span id="openapi-current-table"></span> openapiCurrentTable


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| completed_percent | int64 (formatted integer)| `int64` | ✓ | | The process in percent of importing the table. |  |
| name | string| `string` | ✓ | | The name of the table. |  |
| size | uint64 (formatted string)| `string` | ✓ | | The data size of the table. |  |



### <span id="openapi-custom-c-s-v-format"></span> openapiCustomCSVFormat


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| backslash_escape | boolean| `bool` |  | |  |  |
| delimiter | string| `string` |  | |  |  |
| header | boolean| `bool` |  | |  |  |
| not_null | boolean| `bool` |  | |  |  |
| null | string| `string` |  | |  |  |
| separator | string| `string` |  | |  |  |
| trim_last_separator | boolean| `bool` |  | |  |  |



### <span id="openapi-data-format"></span> openapiDataFormat


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| openapiDataFormat | string| string | |  |  |



### <span id="openapi-generate-upload-url-resq"></span> openapiGenerateUploadURLResq


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| new_file_name | string| `string` | ✓ | |  |  |
| upload_url | string| `string` | ✓ | |  |  |



### <span id="openapi-get-import-resp"></span> openapiGetImportResp


> ImportItem is the information of import job.
  





**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| all_completed_tables | [][OpenapiImportTableCompletionInfo](#openapi-import-table-completion-info)| `[]*OpenapiImportTableCompletionInfo` |  | | Completion information of the tables imported. |  |
| cluster_id | uint64 (formatted string)| `string` | ✓ | | The ID of the cluster. | `1` |
| completed_percent | int64 (formatted integer)| `int64` | ✓ | | The process in percent of the import job, but doesn't include the post-processing progress. |  |
| completed_tables | int64 (formatted integer)| `int64` | ✓ | | The number of completed tables. |  |
| created_at | date-time (formatted string)| `strfmt.DateTime` | ✓ | | The creation timestamp of the import job. |  |
| creation_details | [OpenapiCreateImportReq](#openapi-create-import-req)| `OpenapiCreateImportReq` |  | | The creation details of the import job. |  |
| current_tables | [][OpenapiCurrentTable](#openapi-current-table)| `[]*OpenapiCurrentTable` | ✓ | | The current tables are being imported. |  |
| data_format | [OpenapiDataFormat](#openapi-data-format)| `OpenapiDataFormat` | ✓ | | The format of data to import. |  |
| elapsed_time_seconds | int64 (formatted integer)| `int64` | ✓ | | The elapsed time of the import job in seconds. |  |
| id | uint64 (formatted string)| `string` |  | | The ID of the import job. | `1` |
| message | string| `string` | ✓ | | The message. |  |
| pending_tables | int64 (formatted integer)| `int64` | ✓ | | The number of pending tables. |  |
| post_import_completed_percent | int64 (formatted integer)| `int64` |  | | The post-process in percent of the import job. |  |
| processed_source_data_size | uint64 (formatted string)| `string` |  | | The size of source data processed. |  |
| source_url | string| `string` |  | | The full s3 path that contains data to import. |  |
| status | [OpenapiGetImportRespStatus](#openapi-get-import-resp-status)| `OpenapiGetImportRespStatus` | ✓ | | The status of the import job. |  |
| total_files | int64 (formatted integer)| `int64` | ✓ | | The total number of files of the data imported. |  |
| total_size | uint64 (formatted string)| `string` | ✓ | | The total size of the data imported. |  |
| total_tables_count | int64 (formatted integer)| `int64` |  | | The total number of tables. |  |



### <span id="openapi-get-import-resp-status"></span> openapiGetImportRespStatus


  

| Name | Type | Go type | Default | Description | Example |
|------|------|---------| ------- |-------------|---------|
| openapiGetImportRespStatus | string| string | |  |  |



### <span id="openapi-import-table-completion-info"></span> openapiImportTableCompletionInfo


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| message | string| `string` |  | | The message. |  |
| result | [ImportTableCompletionInfoResult](#import-table-completion-info-result)| `ImportTableCompletionInfoResult` | ✓ | | The result status of importing the table. |  |
| table_name | string| `string` | ✓ | | The name of the table. |  |



### <span id="openapi-list-imports-resp"></span> openapiListImportsResp


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| imports | [][OpenapiGetImportResp](#openapi-get-import-resp)| `[]*OpenapiGetImportResp` | ✓ | | The items of import jobs in the cluster. |  |
| total | int64 (formatted string)| `string` | ✓ | | The total number of import jobs in the cluster. |  |



### <span id="openapi-table"></span> openapiTable


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| schema | string| `string` |  | |  |  |
| table | string| `string` |  | |  |  |



### <span id="protobuf-any"></span> protobufAny


  



**Properties**

| Name | Type | Go type | Required | Default | Description | Example |
|------|------|---------|:--------:| ------- |-------------|---------|
| @type | string| `string` |  | |  |  |


