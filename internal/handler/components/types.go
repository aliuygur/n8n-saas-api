package components

// Instance represents an n8n instance
type Instance struct {
	ID          string
	InstanceURL string
	Status      string
	CreatedAt   string
}

func (i *Instance) GetInstanceURL() string {
	return i.InstanceURL
}
