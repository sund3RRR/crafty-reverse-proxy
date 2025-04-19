// Package crafty provides a client for interacting with the Crafty API,
// a web-based control panel for managing Minecraft servers.
package crafty

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sund3RRR/crafty-reverse-proxy/config"
)

// Crafty is a client for the Crafty API. It provides methods to start and stop Minecraft servers by port.
type Crafty struct {
	apiURL   string
	username string
	password string
	client   *http.Client
}

// New creates a new Crafty API client using the provided configuration.
func New(cfg config.Config) *Crafty {
	return &Crafty{
		apiURL:   cfg.APIURL,
		username: cfg.Username,
		password: cfg.Password,
		client:   &http.Client{},
	}
}

// StartMcServer starts a Minecraft server that is configured to listen on the specified port.
// It authenticates with the Crafty API, fetches the list of servers, and sends a start command to the matching one.
func (c *Crafty) StartMcServer(port int) error {
	bearer, err := c.getBearer()
	if err != nil {
		return fmt.Errorf("%w, %v", ErrAuthorizationFailed, err)
	}

	serverList, err := c.getServers(bearer)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToGetServers, err)
	}

	for _, server := range serverList.Data {
		if server.Port == port {
			return c.sendStartServerRequest(server, bearer)
		}
	}

	return ErrNoSuchServer
}

// StopMcServer stops a Minecraft server that is configured to listen on the specified port.
// It authenticates with the Crafty API, fetches the list of servers, and sends a stop command to the matching one.
func (c *Crafty) StopMcServer(port int) error {
	bearer, err := c.getBearer()
	if err != nil {
		return fmt.Errorf("%w, %v", ErrAuthorizationFailed, err)
	}

	serverList, err := c.getServers(bearer)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFailedToGetServers, err)
	}

	for _, server := range serverList.Data {
		if server.Port == port {
			return c.sendStopServerRequest(server, bearer)
		}
	}

	return ErrNoSuchServer
}

// sendStartServerRequest sends a start command for the specified server using its ID.
// Requires a valid bearer token for authentication.
func (c *Crafty) sendStartServerRequest(server Server, bearer string) error {
	startServerURL := c.apiURL + "/api/v2/servers/" + server.ServerID + "/action/start_server"
	request, err := http.NewRequest(http.MethodPost, startServerURL, nil)
	if err != nil {
		return err
	}

	request.Header.Add("Authorization", bearer)
	_, err = c.client.Do(request)
	if err != nil {
		return fmt.Errorf("%w, id %s, port %d: %v", ErrFailedToStartServer, server.ServerID, server.Port, err)
	}

	return nil
}

// sendStopServerRequest sends a stop command for the specified server using its ID.
// Requires a valid bearer token for authentication.
func (c *Crafty) sendStopServerRequest(server Server, bearer string) error {
	stopServerURL := c.apiURL + "/api/v2/servers/" + server.ServerID + "/action/stop_server"
	request, err := http.NewRequest(http.MethodPost, stopServerURL, nil)
	if err != nil {
		return err
	}

	request.Header.Add("Authorization", bearer)
	_, err = c.client.Do(request)
	if err != nil {
		return fmt.Errorf("%w, id %s, port %d: %v", ErrFailedToStopServer, server.ServerID, server.Port, err)
	}

	return nil
}

// getBearer authenticates with the Crafty API and returns a bearer token to be used for authorized requests.
func (c *Crafty) getBearer() (string, error) {
	loginBody := LoginPayload{
		Username: c.username,
		Password: c.password,
	}

	jsonData, err := json.Marshal(loginBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(c.apiURL+"/api/v2/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrHTTPRequestFailed, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFailedToReadBody, err)
	}

	var response LoginResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Bearer %s", response.Data.Token), nil
}

// getServers retrieves a list of all servers available in the Crafty panel.
// Requires a valid bearer token for authentication.
func (c *Crafty) getServers(bearer string) (ServerList, error) {
	request, _ := http.NewRequest(http.MethodGet, c.apiURL+"/api/v2/servers", nil)
	request.Header.Add("Authorization", bearer)

	response, err := c.client.Do(request)
	if err != nil {
		return ServerList{}, fmt.Errorf("%w: %v", ErrHTTPRequestFailed, err)
	}
	defer response.Body.Close()

	serversListBody, err := io.ReadAll(response.Body)
	if err != nil {
		return ServerList{}, fmt.Errorf("%w: %v", ErrFailedToReadBody, err)
	}

	var serverList ServerList
	err = json.Unmarshal(serversListBody, &serverList)
	if err != nil {
		return ServerList{}, err
	}

	return serverList, nil
}
