variable "kafka_password" {
  type      = string
  sensitive = true
}

variable "mysql_password" {
  type      = string
  sensitive = true
}

# Replicate changes to Apache Kafka.
resource "tidbcloud_dedicated_changefeed" "kafka_example" {
  cluster_id           = "your_cluster_id"
  name                 = "changefeed-to-kafka"
  replication_capacity = "4rcu"
  downstream_type      = "KAFKA"

  network_info = {
    network_type = "NETWORK_TYPE_PUBLIC"
  }

  start_position = {
    mode = "FROM_NOW"
  }

  table_config = {
    filter_rules = ["mydb.*"]
  }

  kafka = {
    broker = {
      version          = "KAFKA_VERSION_3XX"
      broker_endpoints = "broker1:9092,broker2:9092"
    }
    authentication = {
      auth_type = "SASL_SCRAM_SHA_256"
      username  = "kafka_user"
      password  = var.kafka_password
    }
    data_format = {
      protocol = "PROTOCOL_CANAL_JSON"
    }
    topic_partition_config = {
      dispatch_type      = "DISPATCH_TYPE_ONE_TOPIC"
      default_topic      = "tidb-cdc"
      replication_factor = 3
      partition_num      = 6
    }
  }
}

# Replicate changes to a MySQL-compatible database.
resource "tidbcloud_dedicated_changefeed" "mysql_example" {
  cluster_id           = "your_cluster_id"
  name                 = "changefeed-to-mysql"
  replication_capacity = "4rcu"
  downstream_type      = "MYSQL"

  network_info = {
    network_type = "NETWORK_TYPE_PUBLIC"
  }

  start_position = {
    mode      = "FROM_TSO"
    start_tso = "457111558064308225"
  }

  mysql = {
    connection = {
      endpoint = "mysql.example.com:3306"
      username = "root"
      password = var.mysql_password
    }
  }
}
