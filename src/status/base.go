package status

type Status string

const (
	Created  Status = "created"
	Queued   Status = "queued"
	Running  Status = "running"
	Stopped  Status = "stopped"
	Active   Status = "active"
	Archived Status = "archived"
	Failed   Status = "failed"
	Pending  Status = "pending"
)
