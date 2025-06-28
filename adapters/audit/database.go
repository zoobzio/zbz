package audit

import (
	"time"

	"zbz/capitan"
	"zbz/database"
	"zbz/zlog"
)

// ConnectDatabaseToAudit creates audit logging for all database operations
func ConnectDatabaseToAudit() error {
	// Register hooks for all CRUD operations
	capitan.RegisterOutput[database.RecordCreatedData](
		database.RecordCreated,
		auditCreate,
	)

	capitan.RegisterOutput[database.RecordUpdatedData](
		database.RecordUpdated,
		auditUpdate,
	)

	capitan.RegisterOutput[database.RecordDeletedData](
		database.RecordDeleted,
		auditDelete,
	)

	zlog.Info("Connected database service to audit logging")
	return nil
}

// AuditEvent represents an audit log entry
type AuditEvent struct {
	Action    string    `json:"action"`
	Table     string    `json:"table"`
	RecordID  any       `json:"record_id"`
	Timestamp time.Time `json:"timestamp"`
	Changes   any       `json:"changes,omitempty"`
}

func auditCreate(data database.RecordCreatedData) error {
	auditEvent := AuditEvent{
		Action:    "CREATE",
		Table:     data.TableName,
		RecordID:  data.RecordID,
		Timestamp: time.Now(),
		Changes:   data.Data,
	}

	zlog.Info("Database audit event",
		zlog.String("action", auditEvent.Action),
		zlog.String("table", auditEvent.Table),
		zlog.Any("record_id", auditEvent.RecordID))

	return nil
}

func auditUpdate(data database.RecordUpdatedData) error {
	auditEvent := AuditEvent{
		Action:    "UPDATE",
		Table:     data.TableName,
		RecordID:  data.RecordID,
		Timestamp: time.Now(),
		Changes:   data.Changes,
	}

	zlog.Info("Database audit event",
		zlog.String("action", auditEvent.Action),
		zlog.String("table", auditEvent.Table),
		zlog.Any("record_id", auditEvent.RecordID))

	return nil
}

func auditDelete(data database.RecordDeletedData) error {
	auditEvent := AuditEvent{
		Action:    "DELETE",
		Table:     data.TableName,
		RecordID:  data.RecordID,
		Timestamp: time.Now(),
	}

	zlog.Info("Database audit event",
		zlog.String("action", auditEvent.Action),
		zlog.String("table", auditEvent.Table),
		zlog.Any("record_id", auditEvent.RecordID))

	return nil
}