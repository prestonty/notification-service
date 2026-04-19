package event

// Type identifies a domain event that can trigger notifications
type Type string
const (
	PRMerged		Type = "PR_MERGED"
	DeploySucceeded Type = "DEPLOY_SUCCEEDED"
	DeployFailed	Type = "DEPLOY_FAILED"
	OrderReady		Type = "ORDER_READY"
)
