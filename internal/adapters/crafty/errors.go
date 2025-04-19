package crafty

import "errors"

var (
	// ErrHTTPRequestFailed is returned when an HTTP request to the Crafty API fails.
	ErrHTTPRequestFailed = errors.New("failed to send HTTP request")

	// ErrFailedToReadBody is returned when the response body from the Crafty API cannot be read.
	ErrFailedToReadBody = errors.New("failed to read response body")

	// ErrFailedToGetServers is returned when the server list could not be retrieved from the Crafty API.
	ErrFailedToGetServers = errors.New("failed to get servers")

	// ErrFailedToStartServer is returned when a server start request fails.
	ErrFailedToStartServer = errors.New("failed to start Minecraft server")

	// ErrFailedToStopServer is returned when a server stop request fails.
	ErrFailedToStopServer = errors.New("failed to stop Minecraft server")

	// ErrAuthorizationFailed is returned when authentication with the Crafty API fails.
	ErrAuthorizationFailed = errors.New("authorization failed")

	// ErrNoSuchServer is returned when no Minecraft server with the specified port is found.
	ErrNoSuchServer = errors.New("no such server")
)
