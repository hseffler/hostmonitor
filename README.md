# Host Monitor in Go

A Go implementation of a host monitor that sends Telegram notifications when targets go down or come back up.

## Features

- Monitors multiple targets via TCP connect (port 80) - no root privileges required
- Configurable failure tolerance (default: 5 failures)
- Configurable check interval (default: 30 seconds)
- Telegram notifications for down/up events
- **Advanced logging with logrus**: Structured logging with multiple log levels
- **Verbose debug mode**: Detailed troubleshooting information with `-verbose` flag
- Thread-safe with mutex protection
- Pure Go implementation with minimal dependencies

## Installation

### Prerequisites

- Go 1.16+
- Telegram bot token and chat ID

### Build from source

```bash
git clone https://github.com/yourusername/host-monitor.git
cd host-monitor
make build
```

### Manual Installation

1. Copy the binary to your desired location:

```bash
sudo cp build/hostmonitor /usr/local/bin/hostmonitor
sudo chmod +x /usr/local/bin/hostmonitor
```

2. Copy and edit the example configuration:

```bash
cp hostmonitor.yaml.example hostmonitor.yaml
nano hostmonitor.yaml
```

3. Edit the configuration file to add your Telegram bot token and chat ID, and configure your targets.

### System-wide Installation (Optional)

For a system-wide installation:

```bash
sudo mkdir -p /opt/hostmonitor
sudo cp hostmonitor.yaml.example /opt/hostmonitor/hostmonitor.yaml
sudo nano /opt/hostmonitor/hostmonitor.yaml
```

Then run with:
```bash
hostmonitor -config /opt/hostmonitor/hostmonitor.yaml
```

## Configuration

Edit the YAML config file at `/opt/hostmonitor/hostmonitor.yaml`:

```yaml
# Host Monitor Configuration (YAML format)

# Telegram bot configuration
telegram:
  token: "YOUR_TELEGRAM_BOT_TOKEN"
  chat_id: "YOUR_CHAT_ID"

# Monitoring settings
settings:
  tolerance: 5                    # Number of failures before notification
  interval: 30                    # Seconds between checks

# Targets to monitor (hostnames or IP addresses)
targets:
  - "google.com"
  - "example.com"
  - "8.8.8.8"
  - "1.1.1.1"
```

An example configuration file `hostmonitor.yaml.example` is provided with the default settings.

## Usage

```bash
hostmonitor -config /path/to/config.yaml
```

### Options

- `-config string`: Path to YAML config file (default: "hostmonitor.yaml" in current directory)
- `-verbose`: Enable verbose debug logging for troubleshooting

### Example

```bash
# Using default config location (current directory)
hostmonitor

# Using custom config file
hostmonitor -config /etc/hostmonitor/config.yaml

# For testing with the example config
hostmonitor -config hostmonitor.yaml.example

# Enable verbose debug logging
hostmonitor -verbose
hostmonitor -config custom.yaml -verbose
```

## Running as a service

To run as a systemd service:

1. Create a systemd service file at `/etc/systemd/system/hostmonitor.service`:

```bash
sudo nano /etc/systemd/system/hostmonitor.service
```

2. Add the following content:

```ini
[Unit]
Description=Host Monitor Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=/opt/hostmonitor/hostmonitor
Restart=always
RestartSec=30

[Install]
WantedBy=multi-user.target
```

3. Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable hostmonitor
sudo systemctl start hostmonitor
```

4. Check the service status:

```bash
sudo systemctl status hostmonitor
```

5. View logs:

```bash
sudo journalctl -u hostmonitor -f
```

## Logging

Logs are written to journald (via stdout) when running as a systemd service. Use `journalctl -u hostmonitor -f` to view logs.

## License

MIT License - see LICENSE file for details.

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.
