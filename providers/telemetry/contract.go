package telemetry

import (
	"context"
	"fmt"
	"strings"
	"time"

	"zbz/universal"
)

// TelemetryContract represents a typed telemetry collection within the telemetry service
// Embeds universal.DataAccess[T] for compile-time guarantee and universal compatibility
type TelemetryContract[T any] interface {
	universal.DataAccess[T] // Embedded interface provides automatic compile-time guarantee
	
	// Telemetry-specific extensions beyond universal interface
	EmitMetric(ctx context.Context, metric T) error
	EmitTrace(ctx context.Context, trace T) error
	EmitLog(ctx context.Context, log T) error
	Query(ctx context.Context, queryName string, params map[string]any) ([]T, error)
	QueryOne(ctx context.Context, queryName string, params map[string]any, dest *T) error
	
	// Aggregation operations
	Aggregate(ctx context.Context, aggregation AggregationQuery) (AggregationResults, error)
	
	// Health and metadata
	Health() (ProviderHealth, error)
	CollectionName() string
	Provider() string
}

// zTelemetryContract is the concrete implementation of TelemetryContract[T]
// Uses 'z' prefix following existing zDatabase pattern (z = self)
type zTelemetryContract[T any] struct {
	collectionName string             // Collection name: "metrics", "traces", "logs"
	providerKey    string             // Provider key: "default", "prometheus", "jaeger"
	service        *zTelemetryService // Reference to singleton service
	collectionType TelemetryType      // Type of telemetry data
}

// TelemetryType defines the type of telemetry collection
type TelemetryType string

const (
	TelemetryTypeMetric TelemetryType = "metric"
	TelemetryTypeTrace  TelemetryType = "trace"
	TelemetryTypeLog    TelemetryType = "log"
)

// Universal interface implementation (universal.DataAccess[T])

// Get retrieves a telemetry entry by ResourceURI
func (z *zTelemetryContract[T]) Get(ctx context.Context, resource universal.ResourceURI) (T, error) {
	var result T
	
	// Parse ResourceURI: "telemetry://metrics/cpu_usage" -> collection: "metrics", id: "cpu_usage"
	if resource.Service() != "telemetry" {
		return result, fmt.Errorf("invalid service '%s' for telemetry operation", resource.Service())
	}
	
	collectionName := resource.ResourcePath()
	if collectionName == "" {
		collectionName = z.collectionName
	}
	
	id := resource.Identifier()
	if id == "" {
		return result, fmt.Errorf("resource URI must specify an identifier for Get operation")
	}
	
	// Execute query via provider
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return result, err
	}
	
	// Convert to appropriate query based on telemetry type
	switch z.collectionType {
	case TelemetryTypeMetric:
		query := MetricQuery{
			MetricName: id,
			StartTime:  time.Now().Add(-1 * time.Hour), // Default to last hour
			EndTime:    time.Now(),
		}
		results, err := provider.QueryMetrics(ctx, query)
		if err != nil {
			return result, err
		}
		return z.convertMetricResultsToT(results)
		
	case TelemetryTypeLog:
		query := LogQuery{
			StartTime: time.Now().Add(-1 * time.Hour),
			EndTime:   time.Now(),
			Fields:    map[string]string{"id": id},
			Limit:     1,
		}
		results, err := provider.QueryLogs(ctx, query)
		if err != nil {
			return result, err
		}
		if len(results.Entries) > 0 {
			return z.convertLogEntryToT(results.Entries[0])
		}
		return result, fmt.Errorf("log entry not found")
		
	default:
		return result, fmt.Errorf("unsupported telemetry type for Get operation: %s", z.collectionType)
	}
}

// Set creates or updates a telemetry entry by ResourceURI
func (z *zTelemetryContract[T]) Set(ctx context.Context, resource universal.ResourceURI, data T) error {
	// Parse ResourceURI to extract collection and identifier
	collectionName := resource.ResourcePath()
	if collectionName == "" {
		collectionName = z.collectionName
	}
	
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return err
	}
	
	// Emit data based on telemetry type
	switch z.collectionType {
	case TelemetryTypeMetric:
		metric, err := z.convertTToMetric(data)
		if err != nil {
			return err
		}
		return provider.EmitMetric(ctx, metric)
		
	case TelemetryTypeLog:
		logEntry, err := z.convertTToLogEntry(data)
		if err != nil {
			return err
		}
		return provider.EmitLog(ctx, logEntry)
		
	default:
		return fmt.Errorf("unsupported telemetry type for Set operation: %s", z.collectionType)
	}
}

// Delete removes a telemetry entry by ResourceURI (if supported by provider)
func (z *zTelemetryContract[T]) Delete(ctx context.Context, resource universal.ResourceURI) error {
	// Most telemetry providers don't support deletion
	// This is typically used for managing configurations or alerts
	return fmt.Errorf("delete operation not supported for telemetry type: %s", z.collectionType)
}

// List retrieves multiple telemetry entries matching a ResourceURI pattern
func (z *zTelemetryContract[T]) List(ctx context.Context, pattern universal.ResourceURI) ([]T, error) {
	collectionName := pattern.ResourcePath()
	if collectionName == "" {
		collectionName = z.collectionName
	}
	
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return nil, err
	}
	
	// Build query from identifier pattern
	identifier := pattern.Identifier()
	
	switch z.collectionType {
	case TelemetryTypeMetric:
		query := MetricQuery{
			StartTime: time.Now().Add(-1 * time.Hour), // Default to last hour
			EndTime:   time.Now(),
		}
		
		if identifier != "" && identifier != "*" {
			// Parse filter patterns like "name:cpu_usage" or "label:service:api"
			if strings.Contains(identifier, ":") {
				parts := strings.SplitN(identifier, ":", 2)
				if parts[0] == "name" {
					query.MetricName = parts[1]
				} else if parts[0] == "label" {
					labelParts := strings.SplitN(parts[1], ":", 2)
					if len(labelParts) == 2 {
						query.Labels = map[string]string{labelParts[0]: labelParts[1]}
					}
				}
			}
		}
		
		results, err := provider.QueryMetrics(ctx, query)
		if err != nil {
			return nil, err
		}
		return z.convertMetricResultsToTSlice(results)
		
	case TelemetryTypeLog:
		query := LogQuery{
			StartTime: time.Now().Add(-1 * time.Hour),
			EndTime:   time.Now(),
			Limit:     1000, // Default limit
		}
		
		if identifier != "" && identifier != "*" {
			if strings.Contains(identifier, ":") {
				parts := strings.SplitN(identifier, ":", 2)
				if parts[0] == "level" {
					query.Level = parts[1]
				} else {
					query.Fields = map[string]string{parts[0]: parts[1]}
				}
			}
		}
		
		results, err := provider.QueryLogs(ctx, query)
		if err != nil {
			return nil, err
		}
		return z.convertLogEntriesToTSlice(results.Entries)
		
	default:
		return nil, fmt.Errorf("unsupported telemetry type for List operation: %s", z.collectionType)
	}
}

// Exists checks if a telemetry entry exists by ResourceURI
func (z *zTelemetryContract[T]) Exists(ctx context.Context, resource universal.ResourceURI) (bool, error) {
	_, err := z.Get(ctx, resource)
	if err != nil {
		return false, nil // Assume not found rather than error
	}
	return true, nil
}

// Count returns the number of telemetry entries matching a ResourceURI pattern
func (z *zTelemetryContract[T]) Count(ctx context.Context, pattern universal.ResourceURI) (int64, error) {
	results, err := z.List(ctx, pattern)
	if err != nil {
		return 0, err
	}
	return int64(len(results)), nil
}

// Execute performs complex telemetry operations via OperationURI
func (z *zTelemetryContract[T]) Execute(ctx context.Context, operation universal.OperationURI, params any) (any, error) {
	return z.service.Execute(operation, params)
}

// ExecuteMany performs multiple telemetry operations
func (z *zTelemetryContract[T]) ExecuteMany(ctx context.Context, operations []universal.Operation) ([]any, error) {
	var results []any
	for _, op := range operations {
		opURI, err := universal.ParseOperationURI(op.Type)
		if err != nil {
			return nil, fmt.Errorf("invalid operation URI '%s': %w", op.Type, err)
		}
		
		result, err := z.Execute(ctx, opURI, op.Params)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// Subscribe watches for telemetry changes matching a ResourceURI pattern
func (z *zTelemetryContract[T]) Subscribe(ctx context.Context, pattern universal.ResourceURI, callback universal.ChangeCallback[T]) (universal.SubscriptionID, error) {
	// Telemetry subscriptions are typically for real-time metrics/logs
	// This would integrate with provider-specific streaming APIs
	return "", fmt.Errorf("telemetry subscriptions not implemented yet")
}

// Unsubscribe removes a telemetry subscription
func (z *zTelemetryContract[T]) Unsubscribe(ctx context.Context, id universal.SubscriptionID) error {
	return fmt.Errorf("telemetry subscriptions not implemented yet")
}

// Name returns the data access name
func (z *zTelemetryContract[T]) Name() string {
	return fmt.Sprintf("telemetry-%s-%s", z.collectionType, z.collectionName)
}

// Type returns the data access type
func (z *zTelemetryContract[T]) Type() string {
	return "telemetry"
}

// Telemetry-specific methods (beyond universal interface)

// EmitMetric emits a single metric
func (z *zTelemetryContract[T]) EmitMetric(ctx context.Context, metric T) error {
	if z.collectionType != TelemetryTypeMetric {
		return fmt.Errorf("EmitMetric only available for metric collections")
	}
	
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return err
	}
	
	convertedMetric, err := z.convertTToMetric(metric)
	if err != nil {
		return err
	}
	
	return provider.EmitMetric(ctx, convertedMetric)
}

// EmitTrace emits a single trace
func (z *zTelemetryContract[T]) EmitTrace(ctx context.Context, trace T) error {
	if z.collectionType != TelemetryTypeTrace {
		return fmt.Errorf("EmitTrace only available for trace collections")
	}
	
	// Trace emission logic would go here
	return fmt.Errorf("trace emission not implemented yet")
}

// EmitLog emits a single log entry
func (z *zTelemetryContract[T]) EmitLog(ctx context.Context, log T) error {
	if z.collectionType != TelemetryTypeLog {
		return fmt.Errorf("EmitLog only available for log collections")
	}
	
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return err
	}
	
	convertedLog, err := z.convertTToLogEntry(log)
	if err != nil {
		return err
	}
	
	return provider.EmitLog(ctx, convertedLog)
}

// Query executes a named telemetry query
func (z *zTelemetryContract[T]) Query(ctx context.Context, queryName string, params map[string]any) ([]T, error) {
	// This would use query templates similar to database queries
	// For now, delegate to provider-specific querying
	return nil, fmt.Errorf("named telemetry queries not implemented yet")
}

// QueryOne executes a named telemetry query and returns a single result
func (z *zTelemetryContract[T]) QueryOne(ctx context.Context, queryName string, params map[string]any, dest *T) error {
	results, err := z.Query(ctx, queryName, params)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		return fmt.Errorf("no results found")
	}
	*dest = results[0]
	return nil
}

// Aggregate performs aggregation operations on telemetry data
func (z *zTelemetryContract[T]) Aggregate(ctx context.Context, aggregation AggregationQuery) (AggregationResults, error) {
	// Aggregation logic would go here
	return AggregationResults{}, fmt.Errorf("aggregation not implemented yet")
}

// Health returns the health status of the telemetry provider
func (z *zTelemetryContract[T]) Health() (ProviderHealth, error) {
	provider, err := z.service.getProvider(z.providerKey)
	if err != nil {
		return ProviderHealth{}, err
	}
	
	return provider.Health(context.Background())
}

// CollectionName returns the collection name
func (z *zTelemetryContract[T]) CollectionName() string {
	return z.collectionName
}

// Provider returns the provider key
func (z *zTelemetryContract[T]) Provider() string {
	return z.providerKey
}

// Helper methods for type conversion

func (z *zTelemetryContract[T]) convertTToMetric(data T) (Metric, error) {
	// This would use reflection or type assertions to convert T to Metric
	// For now, assume T is already a Metric or compatible type
	if metric, ok := any(data).(Metric); ok {
		return metric, nil
	}
	return Metric{}, fmt.Errorf("cannot convert type %T to Metric", data)
}

func (z *zTelemetryContract[T]) convertTToLogEntry(data T) (LogEntry, error) {
	if logEntry, ok := any(data).(LogEntry); ok {
		return logEntry, nil
	}
	return LogEntry{}, fmt.Errorf("cannot convert type %T to LogEntry", data)
}

func (z *zTelemetryContract[T]) convertMetricResultsToT(results MetricResults) (T, error) {
	var zero T
	if t, ok := any(results).(T); ok {
		return t, nil
	}
	return zero, fmt.Errorf("cannot convert MetricResults to type %T", zero)
}

func (z *zTelemetryContract[T]) convertMetricResultsToTSlice(results MetricResults) ([]T, error) {
	// Convert metric results to slice of T
	var slice []T
	for _, dataPoint := range results.DataPoints {
		if t, ok := any(dataPoint).(T); ok {
			slice = append(slice, t)
		}
	}
	return slice, nil
}

func (z *zTelemetryContract[T]) convertLogEntryToT(entry LogEntry) (T, error) {
	var zero T
	if t, ok := any(entry).(T); ok {
		return t, nil
	}
	return zero, fmt.Errorf("cannot convert LogEntry to type %T", zero)
}

func (z *zTelemetryContract[T]) convertLogEntriesToTSlice(entries []LogEntry) ([]T, error) {
	var slice []T
	for _, entry := range entries {
		if t, ok := any(entry).(T); ok {
			slice = append(slice, t)
		}
	}
	return slice, nil
}

// AggregationQuery represents a telemetry aggregation query
type AggregationQuery struct {
	MetricName   string            `json:"metric_name"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Aggregations []string          `json:"aggregations"` // "sum", "avg", "max", "min", "count"
	GroupBy      []string          `json:"group_by"`     // Label names to group by
	Labels       map[string]string `json:"labels,omitempty"`
}

// AggregationResults represents the results of an aggregation query
type AggregationResults struct {
	MetricName string              `json:"metric_name"`
	Groups     []AggregationGroup  `json:"groups"`
	Metadata   map[string]any      `json:"metadata,omitempty"`
}

// AggregationGroup represents a single group in aggregation results
type AggregationGroup struct {
	Labels map[string]string          `json:"labels"`
	Values map[string]AggregationValue `json:"values"`
}

// AggregationValue represents the result of a single aggregation
type AggregationValue struct {
	Aggregation string    `json:"aggregation"`
	Value       float64   `json:"value"`
	Timestamp   time.Time `json:"timestamp,omitempty"`
}