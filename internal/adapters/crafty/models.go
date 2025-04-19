package crafty

// LoginResponse represents the response returned by the Crafty API upon successful authentication.
type LoginResponse struct {
	Status string `json:"status"`
	Data   struct {
		Token  string `json:"token"`   // Bearer token used for authenticated requests
		UserID string `json:"user_id"` // ID of the authenticated user
	} `json:"data"`
}

// LoginPayload represents the payload used to authenticate with the Crafty API.
type LoginPayload struct {
	Username string `json:"username"` // Username for authentication
	Password string `json:"password"` // Password for authentication
}

// Server represents a Minecraft server instance managed by the Crafty panel.
type Server struct {
	ServerID string `json:"server_id"`   // Unique ID of the server
	Port     int    `json:"server_port"` // Port the server is listening on
}

// ServerList represents the response structure containing a list of servers from the Crafty API.
type ServerList struct {
	Data []Server `json:"data"` // List of servers
}

// Settings represents a partial response from the Crafty API containing server-related configuration settings.
type Settings struct {
	Servers struct {
		ProxyPort  int `json:"proxy_port"`  // Port used by the proxy (e.g., BungeeCord, Velocity)
		ServerIP   int `json:"server_ip"`   // IP address of the server (usually stored as int for internal use)
		ServerPort int `json:"server_port"` // Port used by the server
	} `json:"servers"`
}
