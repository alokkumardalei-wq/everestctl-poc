package backend

import "time"

// Engine identifies a database engine supported by OpenEverest.
type Engine string

const (
	EnginePostgreSQL Engine = "postgresql"
	EngineMySQL      Engine = "mysql"
	EngineMongoDB    Engine = "mongodb"
)

// SupportedEngines returns the engines this POC understands.
func SupportedEngines() []Engine {
	return []Engine{EnginePostgreSQL, EngineMySQL, EngineMongoDB}
}

// Status is a coarse lifecycle state for a managed resource.
type Status string

const (
	StatusPending Status = "pending"
	StatusReady   Status = "ready"
	StatusError   Status = "error"
	StatusUnknown Status = "unknown"
)

// Database is a logical OpenEverest-managed database cluster.
type Database struct {
	Name      string    `json:"name" yaml:"name"`
	Namespace string    `json:"namespace" yaml:"namespace"`
	Engine    Engine    `json:"engine" yaml:"engine"`
	Version   string    `json:"version" yaml:"version"`
	Replicas  int       `json:"replicas" yaml:"replicas"`
	Cluster   string    `json:"cluster" yaml:"cluster"`
	Status    Status    `json:"status" yaml:"status"`
	CreatedAt time.Time `json:"createdAt" yaml:"createdAt"`
}

// Cluster is a registered Kubernetes cluster OpenEverest can target.
type Cluster struct {
	Name        string    `json:"name" yaml:"name"`
	Endpoint    string    `json:"endpoint" yaml:"endpoint"`
	Context     string    `json:"context" yaml:"context"`
	Version     string    `json:"version" yaml:"version"`
	Status      Status    `json:"status" yaml:"status"`
	RegisteredAt time.Time `json:"registeredAt" yaml:"registeredAt"`
}

// Plugin describes an OpenEverest plugin (e.g. backup, monitoring).
type Plugin struct {
	Name        string            `json:"name" yaml:"name"`
	Version     string            `json:"version" yaml:"version"`
	Description string            `json:"description" yaml:"description"`
	Installed   bool              `json:"installed" yaml:"installed"`
	Config      map[string]string `json:"config,omitempty" yaml:"config,omitempty"`
}

// LogLine is a single emitted log record.
type LogLine struct {
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`
	Level     string    `json:"level" yaml:"level"`
	Message   string    `json:"message" yaml:"message"`
}
