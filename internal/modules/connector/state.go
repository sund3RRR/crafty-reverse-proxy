package connector

// state represents the internal state of the server connection state machine.
type state = int32

// Possible values for the state machine.
//
// These constants define the lifecycle stages of the Minecraft server.
// The Connector uses these to decide when to start or stop the server,
// handle incoming connections, and manage player activity.
const (
	// stateOff indicates the server is completely shut down.
	stateOff state = iota

	// stateStartingUp indicates the server is in the process of starting.
	stateStartingUp

	// stateRunning indicates the server is up and accepting connections.
	stateRunning

	// stateEmpty indicates the server is running but has no active players.
	stateEmpty
)

// String returns the human-readable name of a given state.
func String(state state) string {
	switch state {
	case stateOff:
		return "Off"
	case stateStartingUp:
		return "StartingUp"
	case stateRunning:
		return "Running"
	case stateEmpty:
		return "Empty"
	default:
		return "unknown"
	}
}
