# Terraform TiDBCloud Provider

[![License](https://img.shields.io/github/license/pingcap/tispark.svg)](https://github.com/pingcap/tispark/blob/master/LICENSE)

This is the repository for the terraform-provider-tidbcloud, which allows one to use Terraform with TiDBCloud. Learn more about [TiDBCloud](https://en.pingcap.com/tidb-cloud/)

For general information about Terraform, visit the [official website](https://www.terraform.io) and the [GitHub project page](https://github.com/hashicorp/terraform).

## TOC

- [Requirements](#requirements)
- [Support](#support)
- [Using the provider](#using-the-provider)
    * [Set up](#set-up)
    * [Create an API key](#create-an-api-key)
    * [Get TiDBCloud provider](#get-tidbcloud-provider)
    * [Get projectId with project Data Source](#get-projectid-with-project-data-source)
    * [Get cluster spec info with cluster-spec Data Source](#get-cluster-spec-info-with-cluster-spec-data-source)
    * [Create a dedicated cluster with cluster resource](#create-a-dedicated-cluster-with-cluster-resource)
    * [Change the dedicated cluster](#change-the-dedicated-cluster)
    * [Create a backup with backup resource](#create-a-backup-with-backup-resource)
    * [Create a restore task with restore resource](#create-a-restore-task-with-restore-resource)
    * [destroy the dedicated cluster](#destroy-the-dedicated-cluster)
- [Developing the Provider](#developing-the-provider)
    * [Environment](#environment)
    * [Building the provider from source](#building-the-provider-from-source)
    * [Generate or update documentation in docs file](#generate-or-update-documentation-in-docs-file)
    * [Running the acceptance test](#running-the-acceptance-test)
    * [Debug the provider](#debug-the-provider)
- [Follow us](#follow-us)
    * [Twitter](#twitter)
- [License](#license)


## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.18 (if you want to build the provider plugin)

## Support

Resources
- [cluster](./docs/resources/cluster.md)
- [backup](./docs/resources/backup.md) (not support update)
- [restore](./docs/resources/restore.md) (not support update and delete)

DataSource
- [project](./docs/data-sources/project.md)
- [cluster spec](./docs/data-sources/cluster_spec.md)
- [restore](./docs/data-sources/restore.md)
- [backcup](./docs/data-sources/backup.md)


## Using the provider

Documentation about the provider use and the corresponding specific configuration options can be found on the [official's website](https://www.terraform.io/language/providers).

Here We just give an example to show how to use the TiDBCloud provider. 

In this example, you will create and manage a dedicated cluster, create a backup for it and restore from the backup.

### Set up

TiDBCloud provider has released to terraform registry. All you need to do is install to terraform(>=1.0).

For Mac user, you can install it with Homebrew. About other installation method, see [official doc](https://learn.hashicorp.com/tutorials/terraform/install-cli?in=terraform/aws-get-started)

First, install the HashiCorp tap, a repository of all our Homebrew packages.
```shell
brew install hashicorp/tap/terraform
```
Now, install Terraform with hashicorp/tap/terraform.
```shell
brew install hashicorp/tap/terraform
```

### Create an API key

The TiDB Cloud API uses HTTP Digest Authentication. It protects your private key from being sent over the network. 

However, it does not support manage API key now. So you need to create the API key in the console.

1. Click the account name in the upper-right corner of the TiDB Cloud console.
2. Click Organization Settings. The organization settings page is displayed.
3. Click the API Keys tab and then click Create API Key.
4. Enter a description for your API key. The role of the API key is always Owner currently.
5. Click Next. Copy and save the public key and the private key.
6. Make sure that you have copied and saved the private key in a secure location. The private key only displays upon the creation. After leaving this page, you will not be able to get the full private key again.
7. Click Done.

### Get TiDBCloud provider

Create a main.tf file to get the TiDBCloud provider like:

```
terraform {
  required_providers {
    tidbcloud = {
      source = "hashicorp/tidbcloud"
    }
  }
}

provider "tidbcloud" {
  username = ""
  password = ""
}
```

You need to fill the username and password with API key's public key and private key. 

Or you can pass them with the environment
```
export TiDBCLOUD_USERNAME = ${public_key}
export TiDBCLOUD_PASSWORD = ${private_key}
```

### Get projectId with project Data Source

### Get cluster spec info with cluster-spec Data Source

### Create a dedicated cluster with cluster resource

### Change the dedicated cluster

### Create a backup with backup resource

### Create a restore task with restore resource

### destroy the dedicated cluster


## Developing the Provider

### Environment

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

### Building the provider from source

1. Clone the repository
```shell
git clone git@github.com:tidbcloud/terraform-provider-tidbcloud
```
2. Enter the repository directory
```shell
cd terraform-provider-tidbcloud
```
3. Build the provider using the Go `install` command. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.
```shell
go install
```

### Generate or update documentation in docs file

run `go generate`

### Running the acceptance test

see [here](./internal/README.md) for more detail.

### Debug the provider

I will introduce how to debug with Terraform CLI development overrides. About other ways to debug the provider. See [terraform official doc](https://www.terraform.io/plugin/debugging) for more detail

Development overrides is a method of using a specified local filesystem Terraform provider binary with Terraform CLI, such as one locally built with updated code, rather than a released binary.

1. create `.terraformrc` in your operating system user directory
```
provider_installation {

  dev_overrides {
      "hashicorp.com/edu/tidbcloud" = "/usr/local/go/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

2. run go install in the terraform-provider-tidbcloud, you will find the `terraform-provider-tidbcloud` will be installed under the `/usr/local/go/bin` 
```
go install
```

3. Terraform CLI commands, such as terraform apply, will now use the specified provider binary if you follow the below config:
```
terraform {
  required_providers {
    tidbcloud = {
      source = "hashicorp/tidbcloud"
    }
  }
}
```

## Follow us

Twitter [@PingCAP](https://twitter.com/PingCAP)


## License

TiSpark is under the Apache 2.0 license. See the [LICENSE](./LICENSE) file for details.