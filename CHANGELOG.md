## [0.2.0] 2023-01-16

**Feature**
* Support import resource in [#52](https://github.com/tidbcloud/terraform-provider-tidbcloud/pull/52)

**Enhancement**
* Replace open api sdk by in [#50](https://github.com/tidbcloud/terraform-provider-tidbcloud/pull/50)
* Update terraform-plugin-framework tp 1.1.1 by in [#51](https://github.com/tidbcloud/terraform-provider-tidbcloud/pull/51)
* Refactor test by in [#55](https://github.com/tidbcloud/terraform-provider-tidbcloud/pull/55)

**Docs**
* Unify user guide by in [#44](https://github.com/tidbcloud/terraform-provider-tidbcloud/pull/44)
* Add workflow for terraform by in [#56](https://github.com/tidbcloud/terraform-provider-tidbcloud/pull/56)

**Deps**
* Bump github.com/hashicorp/terraform-plugin-sdk/v2 from 2.21.0 to 2.24.1 by in [#27](https://github.com/tidbcloud/terraform-provider-tidbcloud/pull/27)
* Bump goreleaser/goreleaser-action from 3 to 4 by in [#39](https://github.com/tidbcloud/terraform-provider-tidbcloud/pull/39)
* Bump github.com/hashicorp/terraform-plugin-go from 0.14.1 to 0.14.3 in [#42](https://github.com/tidbcloud/terraform-provider-tidbcloud/pull/42)

## [0.1.1] 2022-11-15

**Docs**
- Fix docs forgeting to change cluster_spec data source to cluster_specs data source.

**Deps**
- Bump github.com/hashicorp/terraform-plugin-go from 0.14.0 to 0.14.1.

## [0.1.0] 2022-11-15

**Compatibility Changes**
- backup datasource is renamed to backups datasource.
- project datasource is renamed to projects datasource.
- restore datasource is renamed to restores datasource.
- cluster_spec datasource is renamed to cluster_specs datasource.

**Features**
- Support clusters datasource.

**Changes**
- Supplement the status information in cluster resource.

## [0.0.4] 2022-10-29

Docs:
- Change Developer Tier to Serverless Tier in response to [TiDB Cloud release](https://docs.pingcap.com/tidbcloud/release-notes-2022#october-28-2022).

## [0.0.3] 2022-10-14

Changes:
- Credentials in provider change from username/password to public_key/private_key.
- Include terraform provider information in client requests.

Docs:
- Fix restore data source example.
- Update local development commands in README.

## [0.0.2] 2022-09-26

Docs:
- Add the use of `terraform init` in README.md.
- Add the README.md link in provider example.

## 0.0.1

Features:
- Support datasource: project, cluster spec, restore, backup.
- Support resources: cluster, restore, backup.
