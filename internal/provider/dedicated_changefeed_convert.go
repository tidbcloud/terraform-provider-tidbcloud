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

// --- read-time reconciliation ----------------------------------------------
//
// The helpers below merge the API view of the downstream configuration onto
// the prior Terraform state on every read (not only on import), so changes
// made outside Terraform surface as drift. The merge is field-wise:
//
//   - a field the configuration manages (non-null in prior state) is
//     refreshed from the API, so an out-of-band change plans back to the
//     configured value;
//   - a field the configuration leaves unset (null in prior state) stays
//     null, so server-side defaults do not surface as perpetual diffs;
//   - input-only credentials (never returned by the API) are retained from
//     prior state;
//   - a nested block that is absent from prior state stays absent, and a
//     block the API stops returning keeps its prior value rather than
//     inventing a removal.
//
// Collections of nested blocks are merged element-wise when the lengths
// match; a length change is real structural drift and takes the API view.

type nullableValue interface{ IsNull() bool }

func mergeField[T nullableValue](prior, api T) T {
	if prior.IsNull() {
		return prior
	}
	return api
}

func mergeTableConfig(prior, api *changefeedTableConfigModel) *changefeedTableConfigModel {
	if prior == nil || api == nil {
		return prior
	}
	out := &changefeedTableConfigModel{
		FilterRules:   mergeField(prior.FilterRules, api.FilterRules),
		Mode:          mergeField(prior.Mode, api.Mode),
		CaseSensitive: mergeField(prior.CaseSensitive, api.CaseSensitive),
	}
	if len(prior.EventFilters) == len(api.EventFilters) {
		for i := range api.EventFilters {
			p, a := prior.EventFilters[i], api.EventFilters[i]
			out.EventFilters = append(out.EventFilters, changefeedEventFilterModel{
				TableMatchers:                   mergeField(p.TableMatchers, a.TableMatchers),
				IgnoredEvents:                   mergeField(p.IgnoredEvents, a.IgnoredEvents),
				IgnoredSqlStatements:            mergeField(p.IgnoredSqlStatements, a.IgnoredSqlStatements),
				IgnoredInsertValueExpression:    mergeField(p.IgnoredInsertValueExpression, a.IgnoredInsertValueExpression),
				IgnoredUpdateOldValueExpression: mergeField(p.IgnoredUpdateOldValueExpression, a.IgnoredUpdateOldValueExpression),
				IgnoredUpdateNewValueExpression: mergeField(p.IgnoredUpdateNewValueExpression, a.IgnoredUpdateNewValueExpression),
				IgnoredDeleteValueExpression:    mergeField(p.IgnoredDeleteValueExpression, a.IgnoredDeleteValueExpression),
			})
		}
	} else {
		out.EventFilters = api.EventFilters
	}
	return out
}

func mergeKafka(prior, api *changefeedKafkaModel) *changefeedKafkaModel {
	if prior == nil || api == nil {
		return prior
	}
	out := &changefeedKafkaModel{}
	if prior.Broker == nil || api.Broker == nil {
		out.Broker = prior.Broker
	} else {
		out.Broker = &changefeedKafkaBrokerModel{
			Version:            mergeField(prior.Broker.Version, api.Broker.Version),
			BrokerEndpoints:    mergeField(prior.Broker.BrokerEndpoints, api.Broker.BrokerEndpoints),
			UseTls:             mergeField(prior.Broker.UseTls, api.Broker.UseTls),
			InsecureSkipVerify: mergeField(prior.Broker.InsecureSkipVerify, api.Broker.InsecureSkipVerify),
			Compression:        mergeField(prior.Broker.Compression, api.Broker.Compression),
		}
	}
	if prior.Authentication == nil || api.Authentication == nil {
		out.Authentication = prior.Authentication
	} else {
		out.Authentication = &changefeedKafkaAuthenticationModel{
			AuthType: mergeField(prior.Authentication.AuthType, api.Authentication.AuthType),
			Username: mergeField(prior.Authentication.Username, api.Authentication.Username),
			// Password is input-only and never returned by the API.
			Password: prior.Authentication.Password,
		}
	}
	if prior.DataFormat == nil || api.DataFormat == nil {
		out.DataFormat = prior.DataFormat
	} else {
		df := &changefeedKafkaDataFormatModel{
			Protocol:             mergeField(prior.DataFormat.Protocol, api.DataFormat.Protocol),
			EnableTidbExtension:  mergeField(prior.DataFormat.EnableTidbExtension, api.DataFormat.EnableTidbExtension),
			OutputRawChangeEvent: mergeField(prior.DataFormat.OutputRawChangeEvent, api.DataFormat.OutputRawChangeEvent),
		}
		if prior.DataFormat.DebeziumConfig == nil || api.DataFormat.DebeziumConfig == nil {
			df.DebeziumConfig = prior.DataFormat.DebeziumConfig
		} else {
			df.DebeziumConfig = &changefeedKafkaDebeziumConfigModel{
				OutputOldValue: mergeField(prior.DataFormat.DebeziumConfig.OutputOldValue, api.DataFormat.DebeziumConfig.OutputOldValue),
				DisableSchema:  mergeField(prior.DataFormat.DebeziumConfig.DisableSchema, api.DataFormat.DebeziumConfig.DisableSchema),
			}
		}
		out.DataFormat = df
	}
	if prior.TopicPartitionConfig == nil || api.TopicPartitionConfig == nil {
		out.TopicPartitionConfig = prior.TopicPartitionConfig
	} else {
		tpc := &changefeedKafkaTopicPartitionConfigModel{
			DispatchType:      mergeField(prior.TopicPartitionConfig.DispatchType, api.TopicPartitionConfig.DispatchType),
			DefaultTopic:      mergeField(prior.TopicPartitionConfig.DefaultTopic, api.TopicPartitionConfig.DefaultTopic),
			TopicPrefix:       mergeField(prior.TopicPartitionConfig.TopicPrefix, api.TopicPartitionConfig.TopicPrefix),
			Separator:         mergeField(prior.TopicPartitionConfig.Separator, api.TopicPartitionConfig.Separator),
			TopicSuffix:       mergeField(prior.TopicPartitionConfig.TopicSuffix, api.TopicPartitionConfig.TopicSuffix),
			ReplicationFactor: mergeField(prior.TopicPartitionConfig.ReplicationFactor, api.TopicPartitionConfig.ReplicationFactor),
			PartitionNum:      mergeField(prior.TopicPartitionConfig.PartitionNum, api.TopicPartitionConfig.PartitionNum),
		}
		if len(prior.TopicPartitionConfig.PartitionDispatchers) == len(api.TopicPartitionConfig.PartitionDispatchers) {
			for i := range api.TopicPartitionConfig.PartitionDispatchers {
				p, a := prior.TopicPartitionConfig.PartitionDispatchers[i], api.TopicPartitionConfig.PartitionDispatchers[i]
				tpc.PartitionDispatchers = append(tpc.PartitionDispatchers, changefeedKafkaPartitionDispatcherModel{
					PartitionType: mergeField(p.PartitionType, a.PartitionType),
					Matcher:       mergeField(p.Matcher, a.Matcher),
					IndexName:     mergeField(p.IndexName, a.IndexName),
					Columns:       mergeField(p.Columns, a.Columns),
				})
			}
		} else {
			tpc.PartitionDispatchers = api.TopicPartitionConfig.PartitionDispatchers
		}
		out.TopicPartitionConfig = tpc
	}
	if len(prior.ColumnSelectors) == len(api.ColumnSelectors) {
		for i := range api.ColumnSelectors {
			p, a := prior.ColumnSelectors[i], api.ColumnSelectors[i]
			out.ColumnSelectors = append(out.ColumnSelectors, changefeedKafkaColumnSelectorModel{
				Matcher: mergeField(p.Matcher, a.Matcher),
				Columns: mergeField(p.Columns, a.Columns),
			})
		}
	} else {
		out.ColumnSelectors = api.ColumnSelectors
	}
	return out
}

func mergeMysql(prior, api *changefeedMysqlModel) *changefeedMysqlModel {
	if prior == nil || api == nil {
		return prior
	}
	out := &changefeedMysqlModel{}
	if prior.Connection == nil || api.Connection == nil {
		out.Connection = prior.Connection
	} else {
		out.Connection = &changefeedMysqlConnectionModel{
			Endpoint: mergeField(prior.Connection.Endpoint, api.Connection.Endpoint),
			Username: mergeField(prior.Connection.Username, api.Connection.Username),
			// Password is input-only and never returned by the API.
			Password: prior.Connection.Password,
		}
	}
	return out
}
