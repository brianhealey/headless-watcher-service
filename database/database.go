package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// TaskFlow represents a task automation configuration
type TaskFlow struct {
	ID               int       `json:"id"`
	DeviceEUI        string    `json:"device_eui"`
	Name             string    `json:"name"`
	Headline         string    `json:"headline"`
	TriggerCondition string    `json:"trigger_condition"`
	TargetObjects    []string  `json:"target_objects"`
	Actions          []string  `json:"actions"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// NotificationEvent represents an alarm/notification event
type NotificationEvent struct {
	ID            int       `json:"id"`
	RequestID     string    `json:"request_id"`
	DeviceEUI     string    `json:"device_eui"`
	Timestamp     int64     `json:"timestamp"`
	Text          string    `json:"text"`
	Img           string    `json:"img"`
	InferenceData string    `json:"inference_data"`
	SensorData    string    `json:"sensor_data"`
	CreatedAt     time.Time `json:"created_at"`
}

// Initialize opens the database connection and creates tables
func Initialize(dbPath string) error {
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Create tables
	if err := createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	log.Printf("Database initialized: %s", dbPath)
	return nil
}

// createTables creates the database schema
func createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS task_flows (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		device_eui TEXT NOT NULL,
		name TEXT NOT NULL,
		headline TEXT NOT NULL,
		trigger_condition TEXT NOT NULL,
		target_objects TEXT NOT NULL,
		actions TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS notification_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		request_id TEXT,
		device_eui TEXT NOT NULL,
		timestamp INTEGER,
		text TEXT,
		img TEXT,
		inference_data TEXT,
		sensor_data TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_task_flows_device ON task_flows(device_eui);
	CREATE INDEX IF NOT EXISTS idx_events_device ON notification_events(device_eui);
	CREATE INDEX IF NOT EXISTS idx_events_timestamp ON notification_events(timestamp);
	`

	_, err := db.Exec(schema)
	return err
}

// Close closes the database connection
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// SaveTaskFlow saves a task flow to the database
func SaveTaskFlow(taskFlow *TaskFlow) error {
	// Convert target objects and actions to JSON
	targetObjectsJSON, err := json.Marshal(taskFlow.TargetObjects)
	if err != nil {
		return fmt.Errorf("failed to marshal target objects: %w", err)
	}

	actionsJSON, err := json.Marshal(taskFlow.Actions)
	if err != nil {
		return fmt.Errorf("failed to marshal actions: %w", err)
	}

	query := `
	INSERT INTO task_flows (device_eui, name, headline, trigger_condition, target_objects, actions, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := db.Exec(query,
		taskFlow.DeviceEUI,
		taskFlow.Name,
		taskFlow.Headline,
		taskFlow.TriggerCondition,
		string(targetObjectsJSON),
		string(actionsJSON),
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to insert task flow: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	taskFlow.ID = int(id)
	taskFlow.CreatedAt = now
	taskFlow.UpdatedAt = now

	log.Printf("Saved task flow: ID=%d, Device=%s, Headline='%s'", taskFlow.ID, taskFlow.DeviceEUI, taskFlow.Headline)
	return nil
}

// GetTaskFlowsByDevice retrieves all task flows for a device
func GetTaskFlowsByDevice(deviceEUI string) ([]*TaskFlow, error) {
	query := `
	SELECT id, device_eui, name, headline, trigger_condition, target_objects, actions, created_at, updated_at
	FROM task_flows
	WHERE device_eui = ?
	ORDER BY created_at DESC
	`

	rows, err := db.Query(query, deviceEUI)
	if err != nil {
		return nil, fmt.Errorf("failed to query task flows: %w", err)
	}
	defer rows.Close()

	var taskFlows []*TaskFlow
	for rows.Next() {
		var tf TaskFlow
		var targetObjectsJSON, actionsJSON string

		err := rows.Scan(
			&tf.ID,
			&tf.DeviceEUI,
			&tf.Name,
			&tf.Headline,
			&tf.TriggerCondition,
			&targetObjectsJSON,
			&actionsJSON,
			&tf.CreatedAt,
			&tf.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task flow: %w", err)
		}

		// Parse JSON arrays
		if err := json.Unmarshal([]byte(targetObjectsJSON), &tf.TargetObjects); err != nil {
			log.Printf("WARNING: Failed to unmarshal target objects for task %d: %v", tf.ID, err)
			tf.TargetObjects = []string{}
		}

		if err := json.Unmarshal([]byte(actionsJSON), &tf.Actions); err != nil {
			log.Printf("WARNING: Failed to unmarshal actions for task %d: %v", tf.ID, err)
			tf.Actions = []string{}
		}

		taskFlows = append(taskFlows, &tf)
	}

	return taskFlows, nil
}

// GetTaskFlowByID retrieves a task flow by ID
func GetTaskFlowByID(id int) (*TaskFlow, error) {
	query := `
	SELECT id, device_eui, name, headline, trigger_condition, target_objects, actions, created_at, updated_at
	FROM task_flows
	WHERE id = ?
	`

	var tf TaskFlow
	var targetObjectsJSON, actionsJSON string

	err := db.QueryRow(query, id).Scan(
		&tf.ID,
		&tf.DeviceEUI,
		&tf.Name,
		&tf.Headline,
		&tf.TriggerCondition,
		&targetObjectsJSON,
		&actionsJSON,
		&tf.CreatedAt,
		&tf.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query task flow: %w", err)
	}

	// Parse JSON arrays
	if err := json.Unmarshal([]byte(targetObjectsJSON), &tf.TargetObjects); err != nil {
		log.Printf("WARNING: Failed to unmarshal target objects for task %d: %v", tf.ID, err)
		tf.TargetObjects = []string{}
	}

	if err := json.Unmarshal([]byte(actionsJSON), &tf.Actions); err != nil {
		log.Printf("WARNING: Failed to unmarshal actions for task %d: %v", tf.ID, err)
		tf.Actions = []string{}
	}

	return &tf, nil
}

// DeleteTaskFlow deletes a task flow by ID
func DeleteTaskFlow(id int) error {
	query := `DELETE FROM task_flows WHERE id = ?`
	result, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete task flow: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("task flow not found: %d", id)
	}

	log.Printf("Deleted task flow: ID=%d", id)
	return nil
}

// SaveNotificationEvent saves a notification event to the database
func SaveNotificationEvent(event *NotificationEvent) error {
	query := `
	INSERT INTO notification_events (request_id, device_eui, timestamp, text, img, inference_data, sensor_data, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	result, err := db.Exec(query,
		event.RequestID,
		event.DeviceEUI,
		event.Timestamp,
		event.Text,
		event.Img,
		event.InferenceData,
		event.SensorData,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to insert notification event: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	event.ID = int(id)
	event.CreatedAt = now

	log.Printf("Saved notification event: ID=%d, Device=%s", event.ID, event.DeviceEUI)
	return nil
}

// GetNotificationEventsByDevice retrieves notification events for a device
func GetNotificationEventsByDevice(deviceEUI string, limit int) ([]*NotificationEvent, error) {
	query := `
	SELECT id, request_id, device_eui, timestamp, text, img, inference_data, sensor_data, created_at
	FROM notification_events
	WHERE device_eui = ?
	ORDER BY timestamp DESC
	LIMIT ?
	`

	rows, err := db.Query(query, deviceEUI, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query notification events: %w", err)
	}
	defer rows.Close()

	var events []*NotificationEvent
	for rows.Next() {
		var event NotificationEvent
		err := rows.Scan(
			&event.ID,
			&event.RequestID,
			&event.DeviceEUI,
			&event.Timestamp,
			&event.Text,
			&event.Img,
			&event.InferenceData,
			&event.SensorData,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification event: %w", err)
		}
		events = append(events, &event)
	}

	return events, nil
}
