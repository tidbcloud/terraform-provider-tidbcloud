# Running the acceptance test

*Note:* Acceptance tests may create real resources, and often cost money to run.

## Requirements
- Go: The most recent stable version.
- Terraform CLI: Version 0.12.26 or later.

## Auth
You need to set API Key with environment before all the tests
```
export TiDBCLOUD_USERNAME=${public_key}
export TiDBCLOUD_PASSWORD=${private_key}
```

## Test With Project
The tests need project are put into the /internal/provide/testwithproject path.


> some tests like cluster_resource_test may cause cost, make sure you have enough balance
> 
Here are the steps to test them: 

1. create a new project for test in tidb cloud (You can also use the default project, but it is not recommended)

2. set projectId with environment
```
export TiDBCLOUD_PROJECTID=${your_project_id}
```
3. test
```
TF_ACC=1 go test -v ./internal/provider/testwithproject
```

## Test With Cluster
The tests need pre-created TiDB cluster are put into the /internal/provider/testwithcluster path

Here are the steps to test them:

1. Create a dedicated cluster and wait for it is ready. You can create it with tidb cloud or terraform

2. set projectId and clusterId with environment
```
export TiDBCLOUD_PROJECTID=${your_project_id}
export TiDBCLOUD_CLUSTERID=${your_cluster_id}
```

3. test
```
TF_ACC=1 go test -v ./internal/provider/testwithcluster
```


## Test Manually
The tests can't be tested directly are put into the /internal/provider/testmanually path

You need to test them manually following the code annotation in every test.