package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/tidbcloud/terraform-provider-tidbcloud/tidbcloud"
)

// Converters between the Terraform models and the hand-written changefeed API
// models, shared by the changefeed resource and data sources.

func stringListModelToAPI(ctx context.Context, l types.List, diags *diag.Diagnostics) []string {
	if l.IsNull() || l.IsUnknown() {
		return nil
	}
	var out []string
	diags.Append(l.ElementsAs(ctx, &out, false)...)
	return out
}

func stringListAPIToModel(ctx context.Context, s []string) types.List {
	if s == nil {
		return types.ListNull(types.StringType)
	}
	l, _ := types.ListValueFrom(ctx, types.StringType, s)
	return l
}

func int64ListModelToAPI(ctx context.Context, l types.List, diags *diag.Diagnostics) []int64 {
	if l.IsNull() || l.IsUnknown() {
		return nil
	}
	var out []int64
	diags.Append(l.ElementsAs(ctx, &out, false)...)
	return out
}

func int64ListAPIToModel(ctx context.Context, s []int64) types.List {
	if s == nil {
		return types.ListNull(types.Int64Type)
	}
	l, _ := types.ListValueFrom(ctx, types.Int64Type, s)
	return l
}

func networkInfoModelToAPI(ctx context.Context, m *changefeedNetworkInfoModel, diags *diag.Diagnostics) *tidbcloud.ChangefeedNetworkInfo {
	if m == nil {
		return nil
	}
	return &tidbcloud.ChangefeedNetworkInfo{
		NetworkType:    m.NetworkType.ValueString(),
		SinkEndpointId: m.SinkEndpointId.ValueStringPointer(),
		Ports:          int64ListModelToAPI(ctx, m.Ports, diags),
	}
}

func networkInfoAPIToModel(ctx context.Context, n *tidbcloud.ChangefeedNetworkInfo) *changefeedNetworkInfoModel {
	if n == nil {
		return nil
	}
	return &changefeedNetworkInfoModel{
		NetworkType:    types.StringValue(n.NetworkType),
		SinkEndpointId: types.StringPointerValue(n.SinkEndpointId),
		Ports:          int64ListAPIToModel(ctx, n.Ports),
	}
}

func startPositionModelToAPI(m *changefeedStartPositionModel) *tidbcloud.ChangefeedStartPosition {
	if m == nil {
		return nil
	}
	return &tidbcloud.ChangefeedStartPosition{
		Mode:           m.Mode.ValueString(),
		StartTso:       m.StartTso.ValueStringPointer(),
		StartTimestamp: m.StartTimestamp.ValueStringPointer(),
	}
}

func startPositionAPIToModel(s *tidbcloud.ChangefeedStartPosition) *changefeedStartPositionModel {
	if s == nil {
		return nil
	}
	return &changefeedStartPositionModel{
		Mode:           types.StringValue(s.Mode),
		StartTso:       types.StringPointerValue(s.StartTso),
		StartTimestamp: types.StringPointerValue(s.StartTimestamp),
	}
}

func tableConfigModelToAPI(ctx context.Context, m *changefeedTableConfigModel, diags *diag.Diagnostics) *tidbcloud.ChangefeedTableConfig {
	if m == nil {
		return nil
	}
	out := &tidbcloud.ChangefeedTableConfig{
		FilterRules:   stringListModelToAPI(ctx, m.FilterRules, diags),
		Mode:          m.Mode.ValueStringPointer(),
		CaseSensitive: m.CaseSensitive.ValueBoolPointer(),
	}
	for _, f := range m.EventFilters {
		out.EventFilters = append(out.EventFilters, tidbcloud.ChangefeedEventFilter{
			TableMatchers:                   stringListModelToAPI(ctx, f.TableMatchers, diags),
			IgnoredEvents:                   stringListModelToAPI(ctx, f.IgnoredEvents, diags),
			IgnoredSqlStatements:            stringListModelToAPI(ctx, f.IgnoredSqlStatements, diags),
			IgnoredInsertValueExpression:    f.IgnoredInsertValueExpression.ValueStringPointer(),
			IgnoredUpdateOldValueExpression: f.IgnoredUpdateOldValueExpression.ValueStringPointer(),
			IgnoredUpdateNewValueExpression: f.IgnoredUpdateNewValueExpression.ValueStringPointer(),
			IgnoredDeleteValueExpression:    f.IgnoredDeleteValueExpression.ValueStringPointer(),
		})
	}
	return out
}

func tableConfigAPIToModel(ctx context.Context, t *tidbcloud.ChangefeedTableConfig) *changefeedTableConfigModel {
	if t == nil {
		return nil
	}
	out := &changefeedTableConfigModel{
		FilterRules:   stringListAPIToModel(ctx, t.FilterRules),
		Mode:          types.StringPointerValue(t.Mode),
		CaseSensitive: types.BoolPointerValue(t.CaseSensitive),
	}
	for _, f := range t.EventFilters {
		out.EventFilters = append(out.EventFilters, changefeedEventFilterModel{
			TableMatchers:                   stringListAPIToModel(ctx, f.TableMatchers),
			IgnoredEvents:                   stringListAPIToModel(ctx, f.IgnoredEvents),
			IgnoredSqlStatements:            stringListAPIToModel(ctx, f.IgnoredSqlStatements),
			IgnoredInsertValueExpression:    types.StringPointerValue(f.IgnoredInsertValueExpression),
			IgnoredUpdateOldValueExpression: types.StringPointerValue(f.IgnoredUpdateOldValueExpression),
			IgnoredUpdateNewValueExpression: types.StringPointerValue(f.IgnoredUpdateNewValueExpression),
			IgnoredDeleteValueExpression:    types.StringPointerValue(f.IgnoredDeleteValueExpression),
		})
	}
	return out
}

func kafkaModelToAPI(ctx context.Context, m *changefeedKafkaModel, diags *diag.Diagnostics) *tidbcloud.ChangefeedKafkaConfig {
	if m == nil {
		return nil
	}
	out := &tidbcloud.ChangefeedKafkaConfig{}
	if m.Broker != nil {
		out.Broker = &tidbcloud.ChangefeedKafkaBroker{
			Version:            m.Broker.Version.ValueString(),
			BrokerEndpoints:    m.Broker.BrokerEndpoints.ValueStringPointer(),
			UseTls:             m.Broker.UseTls.ValueBoolPointer(),
			InsecureSkipVerify: m.Broker.InsecureSkipVerify.ValueBoolPointer(),
			Compression:        m.Broker.Compression.ValueStringPointer(),
		}
	}
	if m.Authentication != nil {
		out.Authentication = &tidbcloud.ChangefeedKafkaAuthentication{
			AuthType: m.Authentication.AuthType.ValueString(),
			Username: m.Authentication.Username.ValueStringPointer(),
			Password: m.Authentication.Password.ValueStringPointer(),
		}
	}
	if m.DataFormat != nil {
		out.DataFormat = &tidbcloud.ChangefeedKafkaDataFormat{
			Protocol:             m.DataFormat.Protocol.ValueString(),
			EnableTidbExtension:  m.DataFormat.EnableTidbExtension.ValueBoolPointer(),
			OutputRawChangeEvent: m.DataFormat.OutputRawChangeEvent.ValueBoolPointer(),
		}
		if m.DataFormat.DebeziumConfig != nil {
			out.DataFormat.DebeziumConfig = &tidbcloud.ChangefeedKafkaDebeziumConfig{
				OutputOldValue: m.DataFormat.DebeziumConfig.OutputOldValue.ValueBoolPointer(),
				DisableSchema:  m.DataFormat.DebeziumConfig.DisableSchema.ValueBoolPointer(),
			}
		}
	}
	if m.TopicPartitionConfig != nil {
		out.TopicPartitionConfig = &tidbcloud.ChangefeedKafkaTopicPartitionConfig{
			DispatchType:      m.TopicPartitionConfig.DispatchType.ValueString(),
			DefaultTopic:      m.TopicPartitionConfig.DefaultTopic.ValueStringPointer(),
			TopicPrefix:       m.TopicPartitionConfig.TopicPrefix.ValueStringPointer(),
			Separator:         m.TopicPartitionConfig.Separator.ValueStringPointer(),
			TopicSuffix:       m.TopicPartitionConfig.TopicSuffix.ValueStringPointer(),
			ReplicationFactor: m.TopicPartitionConfig.ReplicationFactor.ValueInt64(),
			PartitionNum:      m.TopicPartitionConfig.PartitionNum.ValueInt64(),
		}
		for _, d := range m.TopicPartitionConfig.PartitionDispatchers {
			out.TopicPartitionConfig.PartitionDispatchers = append(out.TopicPartitionConfig.PartitionDispatchers, tidbcloud.ChangefeedKafkaPartitionDispatcher{
				PartitionType: d.PartitionType.ValueString(),
				Matcher:       stringListModelToAPI(ctx, d.Matcher, diags),
				IndexName:     d.IndexName.ValueStringPointer(),
				Columns:       stringListModelToAPI(ctx, d.Columns, diags),
			})
		}
	}
	for _, s := range m.ColumnSelectors {
		out.ColumnSelectors = append(out.ColumnSelectors, tidbcloud.ChangefeedKafkaColumnSelector{
			Matcher: stringListModelToAPI(ctx, s.Matcher, diags),
			Columns: stringListModelToAPI(ctx, s.Columns, diags),
		})
	}
	return out
}

func kafkaAPIToModel(ctx context.Context, k *tidbcloud.ChangefeedKafkaConfig) *changefeedKafkaModel {
	if k == nil {
		return nil
	}
	out := &changefeedKafkaModel{}
	if k.Broker != nil {
		out.Broker = &changefeedKafkaBrokerModel{
			Version:            types.StringValue(k.Broker.Version),
			BrokerEndpoints:    types.StringPointerValue(k.Broker.BrokerEndpoints),
			UseTls:             types.BoolPointerValue(k.Broker.UseTls),
			InsecureSkipVerify: types.BoolPointerValue(k.Broker.InsecureSkipVerify),
			Compression:        types.StringPointerValue(k.Broker.Compression),
		}
	}
	if k.Authentication != nil {
		out.Authentication = &changefeedKafkaAuthenticationModel{
			AuthType: types.StringValue(k.Authentication.AuthType),
			Username: types.StringPointerValue(k.Authentication.Username),
			// Password is input-only and never returned by the API.
			Password: types.StringNull(),
		}
	}
	if k.DataFormat != nil {
		out.DataFormat = &changefeedKafkaDataFormatModel{
			Protocol:             types.StringValue(k.DataFormat.Protocol),
			EnableTidbExtension:  types.BoolPointerValue(k.DataFormat.EnableTidbExtension),
			OutputRawChangeEvent: types.BoolPointerValue(k.DataFormat.OutputRawChangeEvent),
		}
		if k.DataFormat.DebeziumConfig != nil {
			out.DataFormat.DebeziumConfig = &changefeedKafkaDebeziumConfigModel{
				OutputOldValue: types.BoolPointerValue(k.DataFormat.DebeziumConfig.OutputOldValue),
				DisableSchema:  types.BoolPointerValue(k.DataFormat.DebeziumConfig.DisableSchema),
			}
		}
	}
	if k.TopicPartitionConfig != nil {
		out.TopicPartitionConfig = &changefeedKafkaTopicPartitionConfigModel{
			DispatchType:      types.StringValue(k.TopicPartitionConfig.DispatchType),
			DefaultTopic:      types.StringPointerValue(k.TopicPartitionConfig.DefaultTopic),
			TopicPrefix:       types.StringPointerValue(k.TopicPartitionConfig.TopicPrefix),
			Separator:         types.StringPointerValue(k.TopicPartitionConfig.Separator),
			TopicSuffix:       types.StringPointerValue(k.TopicPartitionConfig.TopicSuffix),
			ReplicationFactor: types.Int64Value(k.TopicPartitionConfig.ReplicationFactor),
			PartitionNum:      types.Int64Value(k.TopicPartitionConfig.PartitionNum),
		}
		for _, d := range k.TopicPartitionConfig.PartitionDispatchers {
			out.TopicPartitionConfig.PartitionDispatchers = append(out.TopicPartitionConfig.PartitionDispatchers, changefeedKafkaPartitionDispatcherModel{
				PartitionType: types.StringValue(d.PartitionType),
				Matcher:       stringListAPIToModel(ctx, d.Matcher),
				IndexName:     types.StringPointerValue(d.IndexName),
				Columns:       stringListAPIToModel(ctx, d.Columns),
			})
		}
	}
	for _, s := range k.ColumnSelectors {
		out.ColumnSelectors = append(out.ColumnSelectors, changefeedKafkaColumnSelectorModel{
			Matcher: stringListAPIToModel(ctx, s.Matcher),
			Columns: stringListAPIToModel(ctx, s.Columns),
		})
	}
	return out
}

func mysqlModelToAPI(m *changefeedMysqlModel) *tidbcloud.ChangefeedMySQLConfig {
	if m == nil {
		return nil
	}
	out := &tidbcloud.ChangefeedMySQLConfig{}
	if m.Connection != nil {
		out.Connection = &tidbcloud.ChangefeedMySQLConnection{
			Endpoint: m.Connection.Endpoint.ValueStringPointer(),
			Username: m.Connection.Username.ValueString(),
			Password: m.Connection.Password.ValueStringPointer(),
		}
	}
	return out
}

func mysqlAPIToModel(m *tidbcloud.ChangefeedMySQLConfig) *changefeedMysqlModel {
	if m == nil {
		return nil
	}
	out := &changefeedMysqlModel{}
	if m.Connection != nil {
		out.Connection = &changefeedMysqlConnectionModel{
			Endpoint: types.StringPointerValue(m.Connection.Endpoint),
			Username: types.StringValue(m.Connection.Username),
			// Password is input-only and never returned by the API.
			Password: types.StringNull(),
		}
	}
	return out
}
