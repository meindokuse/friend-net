package config

// Service names and Kafka topics
const (
	// AppName - service name
	AppName = "auth-service"

	// AccountsEventsTopic - Kafka topic for account events
	AccountsEventsTopic = "accounts.events"
)

// OAuth provider constants
const (
	OAuthProviderGoogle = "google"
	OAuthProviderGitHub = "github"
)
