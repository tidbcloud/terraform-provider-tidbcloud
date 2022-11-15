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
