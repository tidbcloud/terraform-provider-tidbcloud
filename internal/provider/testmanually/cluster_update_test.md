# Test cluster update

It is hard for us to test update cluster with accepting test for it takes too long for the ready of the dedicated cluster.

Here are the steps to test cluster update manually:

1. write config file
```
terraform {
  required_providers {
    tidbcloud = {
      source = ""  // update it
    }
  }
}

provider "tidbcloud" {
  username = ""  //  update it
  password = ""  // update it
}

resource "tidbcloud_cluster" "cluster1" {
  project_id     = ""     // update it
  name           = "cluster1"
  cluster_type   = "DEDICATED"
  cloud_provider = "AWS"
  region         = "us-east-1"
  config = {
    root_password = ""  // update it
    components = {
      tidb = {
        node_size : "8C16G"
        node_quantity : 1
      }
      tikv = {
        node_size : "8C32G"
        storage_size_gib : 500,
        node_quantity : 3
      }
    }
  }
}
```

2. execute `terraform apply --auto-approve`, it should be success
3. wait the cluster status turned to available
4. change the config file: add tiflash
```
resource "tidbcloud_cluster" "cluster1" {
  project_id     = ""     // update it
  name           = "cluster1"
  cluster_type   = "DEDICATED"
  cloud_provider = "AWS"
  region         = "us-east-1"
  config = {
    root_password = ""  // update it
    components = {
      tidb = {
        node_size : "8C16G"
        node_quantity : 1
      }
      tikv = {
        node_size : "8C32G"
        storage_size_gib : 500,
        node_quantity : 3
      }
      tiflash = {
        node_size : "8C64G"
        storage_size_gib : 500,
        node_quantity : 1
      }
    }
  }
}
```
5. execute `terraform apply --auto-approve`
6. wait the cluster status turned to available
7. change the config file: add tiflash's storage_size_gib
```
resource "tidbcloud_cluster" "cluster1" {
  project_id     = ""     // update it
  name           = "cluster1"
  cluster_type   = "DEDICATED"
  cloud_provider = "AWS"
  region         = "us-east-1"
  config = {
    root_password = ""  // update it
    components = {
      tidb = {
        node_size : "8C16G"
        node_quantity : 1
      }
      tikv = {
        node_size : "8C32G"
        storage_size_gib : 500,
        node_quantity : 3
      }
      tiflash = {
        node_size : "8C64G"
        storage_size_gib : 1000,
        node_quantity : 1
      }
    }
  }
}
```
8. execute `terraform apply --auto-approve`, and it should return error like `tiflash node_size or storage_size_gib can't be changed`
9. change the config file: scale tiflash with the increasing of node_quantity
```
resource "tidbcloud_cluster" "cluster1" {
  project_id     = ""     // update it
  name           = "cluster1"
  cluster_type   = "DEDICATED"
  cloud_provider = "AWS"
  region         = "us-east-1"
  config = {
    root_password = ""  // update it
    components = {
      tidb = {
        node_size : "8C16G"
        node_quantity : 1
      }
      tikv = {
        node_size : "8C32G"
        storage_size_gib : 500,
        node_quantity : 3
      }
      tiflash = {
        node_size : "8C64G"
        storage_size_gib : 500,
        node_quantity : 2
      }
    }
  }
}
```
10. execute `terraform apply --auto-approve`, it should be success
11. delete the cluster with `terraform destroy --auto-approve`
