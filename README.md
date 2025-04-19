# Crafty Reverse Proxy
A lightweight Go-based reverse proxy designed to manage the lifecycle of your Minecraft server automatically. It starts the server upon incoming connections and shuts it down after a period of inactivity, optimizing power consumption.

## Features
- Automatic Server Startup: Initiates the Minecraft server when a connection attempt is detected.
- Automatic Server Shutdown: Stops the server after 2 minutes of inactivity to conserve resources.
- Docker Integration: Seamlessly integrates with Docker Compose setups.
- Customizable Configuration: Easily adjust settings to fit your specific needs.

## Getting Started
Prerequisites
- Docker and Docker Compose installed on your system.
- A running instance of [Crafty Controller](https://craftycontrol.com/).

1. Create the `docker-compose.yaml`:
```yaml
networks:
  crafty-net:
    driver: bridge

services:
  crafty:
    container_name: crafty
    image: registry.gitlab.com/crafty-controller/crafty-4:latest
    restart: always
    ports:
      - "8000:8000"
      - "8800:8800"
      - "8443:8443"
    networks:
      - crafty-net
    volumes:
      - ./crafty/backups:/crafty/backups
      - ./crafty/logs:/crafty/logs
      - ./crafty/servers:/crafty/servers
      - ./crafty/config:/crafty/app/config
      - ./crafty/import:/crafty/import

  reverse-proxy:
    container_name: crafty-reverse-proxy
    image: ghcr.io/sund3rrr/crafty-reverse-proxy:latest
    ports:
      - "25565-25575:25565-25575"
    networks:
      - crafty-net
    volumes:
      - ./reverse-proxy/config.yaml:/craftyproxy/config/config.yaml
    depends_on:
      - crafty
    restart: unless-stopped
```

2. Configure your proxy in `reverse-proxy/config.yaml`:
```yaml
api_url: "http://crafty:8443" # Your's Crafty Controller URL
username: "admin"             # Crafty Controller admin panel username 
password: "password"          # Crafty Controller admin panel password
auto_shutdown: true           # Auto shutdown feature
timeout: "2m"                 # MC server Shutdown timeout 
log_level: "INFO"             # Log Level

addresses:                    # Set of addresses for handling and proxy
  - crafty_host:
      addr: "crafty"          # MC server address (hostname/IP)
      port: 25565             # MC server port
    listener:
      addr: "localhost"       # Proxy address (hostname/IP)
      port: 25565             # Proxy port
    protocol: "tcp"           # Procotol (tcp only, udp is not supported)
  - crafty_host:
      addr: "crafty"
      port: 25566
    listener:
      addr: "localhost"
      port: 25566
    protocol: "tcp"
```

3) Start the services:
```bash
docker-compose up
```

4) Connect to your MC server using proxy's host and port.

## Contributing

Contributions are welcome! Please fork the repository and submit a pull request for any enhancements or bug fixes.​

## License

This project is licensed under the Apache 2.0 License.​
