package gke

// ResourceConfig defines resource configuration for n8n instances
type ResourceConfig struct {
	MainCPURequest      string
	MainMemoryRequest   string
	WorkerCPURequest    string
	WorkerMemoryRequest string
	PostgresStorageSize string
	N8NStorageSize      string
}
