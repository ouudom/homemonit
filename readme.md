# HomeMonit

HomeMonit is a lightweight, single-server system and container monitoring dashboard fork of [Beszel](https://github.com/henrygd/beszel). 

It has been optimized and stripped down specifically to host on a single homeserver. It features a simplified layout, native Telegram thresholds alerts, WebSocket-only communication between the agent and hub, and consumes minimal resources.

## Features

- **Core Metrics**: Real-time CPU, Memory, Disk, and Network stats.
- **Docker/Podman Containers**: View container list and resource usage (CPU/Memory/Network charts per container).
- **Outbound-only WebSockets**: The agent establishes a secure WebSocket connection to the hub (no listening ports or SSH fallback needed on the agent).
- **Native Telegram Alerts**: Configure threshold alerts (CPU, Memory, Disk) and status alerts (Up/Down) sent directly to your Telegram chat.
- **Single Admin View**: Bypasses complex user/settings management, rendering your server's stats directly on the main page.
- **Ultra-lightweight**: No complex multi-user setups, OAuth logins, SMART, ZFS ARC, systemd services, or GPU monitoring.

## Getting Started

### 1. Deploy with Docker Compose

Create a `docker-compose.yml` file as follows:

```yaml
version: '3.8'

services:
  homemonit:
    image: ouudom/homemonit:latest
    container_name: homemonit
    restart: unless-stopped
    ports:
      - "8090:8090"
    environment:
      - PORT=8090
      - TELEGRAM_BOT_TOKEN=your_bot_token_here
      - TELEGRAM_CHAT_ID=your_chat_id_here
    volumes:
      - homemonit_data:/pb_data

  homemonit-agent:
    image: ouudom/homemonit-agent:latest
    container_name: homemonit-agent
    restart: unless-stopped
    network_mode: host
    environment:
      - PORT=45876
      - KEY=your_agent_connection_token_here
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - /:/host:ro

volumes:
  homemonit_data:
```

### 2. Configuration Env Vars

- **Hub (`homemonit`)**:
  - `TELEGRAM_BOT_TOKEN`: The bot token obtained from `@BotFather`.
  - `TELEGRAM_CHAT_ID`: Your personal Telegram Chat ID (can be fetched using `@userinfobot`).
- **Agent (`homemonit-agent`)**:
  - `KEY`: The authentication token (must match the token stored in the `port` field of the system record).

## Building from Source

### Frontend
```bash
cd internal/site
npm install
npm run build
```

### Hub and Agent Go binaries
```bash
make
```

## Credits

- Forked from [Beszel](https://github.com/henrygd/beszel) by henrygd.
- Powered by [PocketBase](https://pocketbase.io/).
