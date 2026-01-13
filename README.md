# Go-Forwarder

A high-performance reverse proxy written in Go, designed to forward HTTP/HTTPS requests to backend services through a local proxy (like Proxyman) for traffic inspection without requiring certificate installation on client devices.

## Features

- **Protocol Support**: HTTP/1.1, HTTP/2, and WebSocket
- **Flexible Routing**: Rule-based routing with powerful matchers
- **Hot-Reload**: Configuration changes without restart
- **Proxy Integration**: Seamless integration with local proxies (Proxyman, Charles, etc.)
- **HTTPS Tunneling**: CONNECT method support for HTTPS traffic
- **Structured Logging**: JSON and text output with multiple log levels

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/simman/go-forwarder.git
cd go-forwarder

# Build the application
make build

# Install to system
sudo make install
```

### Using Go Install

```bash
go install github.com/simman/go-forwarder/cmd/forwarder@latest
```

## Quick Start

1. Copy the example configuration:

```bash
cp configs/config.example.yaml configs/config.yaml
```

2. Edit the configuration file to match your needs:

```yaml
server:
  addr: ":22222"

logging:
  level: info
  format: json

default_proxy: "http://127.0.0.1:9091"

services:
  - name: app-traffic
    handler:
      type: http
    listener:
      type: tcp
    forwarder:
      nodes:
        - name: my-api
          addr: api.example.com:443
          filter:
            host: api.example.com
```

3. Start the forwarder:

```bash
./bin/forwarder -config configs/config.yaml
```

4. Configure your client to use the proxy:

```bash
# Set HTTP proxy
export http_proxy=http://localhost:22222
export https_proxy=http://localhost:22222

# Or configure in your mobile device network settings
# HTTP Proxy: <your-machine-ip>
# Port: 22222
```

## Configuration

### Matcher Rule Syntax

The matcher rule syntax provides flexible request matching:

| Matcher | Syntax | Description |
|---------|--------|-------------|
| Host | `Host{example.com}` | Match request host |
| Host (wildcard) | `Host{*.example.com}` | Match subdomain wildcard |
| Path | `Path{/exact/path}` | Exact path match |
| PathPrefix | `PathPrefix{/api}` | Path prefix match |
| Method | `Method{GET}` or `Method{GET,POST}` | HTTP method match |
| Header | `Header{X-Key=value}` | Header key-value match |
| HeaderRegex | `HeaderRegex{X-Key=pattern.*}` | Header regex match |
| Query | `Query{key=value}` | Query parameter match |

**Operators:**
- `&&` - AND (both conditions must match)
- `||` - OR (either condition must match)
- `!` - NOT (negate condition)
- Parentheses `()` for grouping

**Examples:**

```yaml
# Simple host matching
filter:
  host: api.example.com

# Complex routing rules
matcher:
  rule: Host{api.example.com} && PathPrefix{/v1}

matcher:
  rule: Host{example.com} && (Method{GET} || Method{POST})

matcher:
  rule: Host{api.example.com} && !Path{/health}

matcher:
  rule: Host{*.example.com} && Header{X-Client-Type=mobile}
```

### Configuration Options

#### Server Configuration

```yaml
server:
  addr: ":22222"           # Listen address
  read_timeout: 30s        # Read timeout
  write_timeout: 30s       # Write timeout
  idle_timeout: 120s       # Idle connection timeout
```

#### Logging Configuration

```yaml
logging:
  level: info              # debug, info, warn, error
  format: json             # json, text
  output: stdout           # stdout, stderr, or file path
```

#### Service Configuration

```yaml
services:
  - name: service-name
    handler:
      type: http           # http, tcp
      metadata:
        sniffing: true
        max_body_size: 10mb
    listener:
      type: tcp
    forwarder:
      nodes:
        - name: node-name
          addr: backend.com:443
          filter:            # Simple filter OR
            host: backend.com
          matcher:           # Complex matcher
            rule: Host{backend.com} && PathPrefix{/api}
          proxy: "http://127.0.0.1:9091"  # Optional proxy override
```

## Architecture

```
┌─────────────┐
│ Mobile App  │
└──────┬──────┘
       │ HTTP/HTTPS
       ▼
┌─────────────────┐
│ Go-Forwarder    │
│   :22222        │
└──────┬──────────┘
       │ Match Rules
       ▼
┌─────────────────┐
│ Proxyman        │
│   :9091         │
└──────┬──────────┘
       │ Forward
       ▼
┌─────────────────┐
│ Backend Server  │
└─────────────────┘
```

## Traffic Flow

1. Client sends request to go-forwarder (`:22222`)
2. Go-forwarder matches request against configured rules
3. If matched, forwards to backend through configured proxy (Proxyman)
4. If not matched, returns error response
5. Response flows back through the chain to client

## Hot-Reload

Go-forwarder supports hot-reload of configuration without restart. Simply modify the configuration file, and the changes will be automatically applied.

```bash
# Edit configuration
vim configs/config.yaml

# Changes are automatically detected and applied
# Check logs for reload confirmation
```

## Development

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run linter
make lint
```

### Project Structure

```
go-forwarder/
├── cmd/
│   └── forwarder/          # Application entry point
├── internal/
│   ├── config/             # Configuration management
│   ├── server/             # HTTP server and handlers
│   ├── router/             # Routing engine and matchers
│   └── forwarder/          # Request forwarding
├── pkg/
│   └── logger/             # Logging utilities
├── configs/                # Configuration files
└── scripts/                # Build and installation scripts
```

## Use Cases

### Mobile App Development

Forward mobile app traffic through Proxyman for debugging:

```yaml
services:
  - name: mobile-app
    forwarder:
      nodes:
        - name: api-server
          addr: api.myapp.com:443
          filter:
            host: api.myapp.com
          proxy: "http://127.0.0.1:9091"
```

Configure mobile device:
- HTTP Proxy: `<your-mac-ip>`
- Port: `22222`

### API Development

Route specific API endpoints to different backends:

```yaml
services:
  - name: api-routing
    forwarder:
      nodes:
        - name: api-v1
          addr: api-v1.example.com:443
          matcher:
            rule: Host{api.example.com} && PathPrefix{/v1}
            
        - name: api-v2
          addr: api-v2.example.com:443
          matcher:
            rule: Host{api.example.com} && PathPrefix{/v2}
```

### Conditional Routing

Route based on headers, methods, or query parameters:

```yaml
nodes:
  - name: mobile-backend
    addr: mobile.api.com:443
    matcher:
      rule: Header{X-Client-Type=mobile}
      
  - name: web-backend
    addr: web.api.com:443
    matcher:
      rule: Header{X-Client-Type=web}
```

## Troubleshooting

### Enable Debug Logging

```yaml
logging:
  level: debug
  format: text
  output: stdout
```

### Check Route Matching

When a request doesn't match any route, go-forwarder returns a JSON error:

```json
{
  "error": "no matching route found",
  "host": "example.com",
  "path": "/api/test",
  "method": "GET"
}
```

### Verify Proxy Connection

Ensure your proxy (Proxyman) is running and accessible:

```bash
curl -x http://127.0.0.1:9091 https://example.com
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License

## Acknowledgments

- Inspired by [gost](https://github.com/ginuerzh/gost) configuration format
- Uses [Traefik](https://traefik.io/)-style matcher syntax
- Built with [zerolog](https://github.com/rs/zerolog) for structured logging

## Support

For issues, questions, or suggestions, please open an issue on GitHub.
