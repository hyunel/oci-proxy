# OCI-Proxy

A lightweight caching proxy for OCI registries, designed for network-restricted environments with unreliable or blocked access to foreign registries.

## Features

- **Multi-Registry Support**: Access multiple registries through a single proxy endpoint
- **Smart Caching**: LRU cache with automatic eviction to maximize storage efficiency
- **Upstream Proxy**: HTTP, HTTPS, and SOCKS5 proxy support for restricted networks
- **Automatic Authentication**: Handles registry authentication transparently

## Installation

### Build from Source

```bash
go build -o oci-proxy ./cmd/oci-proxy
```

### Docker

```bash
docker run -p 80:80 -v $(pwd)/config.yaml:/app/config.yaml ghcr.io/hyunel/oci-proxy:latest
```

Or build from source:

```bash
docker build -t oci-proxy .
docker run -p 80:80 -v $(pwd)/config.yaml:/app/config.yaml oci-proxy
```

## Configuration

See [config.yaml](config.yaml) for a complete configuration example.

### Configuration Options

#### Global Settings

- `port`: Port to listen on (default: 80)
- `log_level`: Logging level (`debug`, `info`, `warn`, `error`)
- `whitelist_mode`: If true, only configured registries are allowed
- `default_registry`: Registry to use when image name has no registry prefix
- `base_url`: Base URL for the proxy (used in responses)

#### Authentication

- `auth.username`: Username for proxy access control
- `auth.password`: Password for proxy access control

#### Registry Settings

- `auth.username`: Registry username
- `auth.password`: Registry password or token
- `cache_dir`: Directory for cached blobs
- `cache_max_size`: Maximum cache size (e.g., `1g`, `500m`, `1024k`)
- `upstream_proxy`: Upstream proxy URL (http, https, or socks5)
- `follow_redirects`: Follow HTTP redirects (default: true)
- `insecure`: Allow HTTP connections (default: false)

## Usage

### Start the Proxy

```bash
./oci-proxy -c config.yaml
```

Deploy the proxy on a server with reliable internet access (e.g., `proxy.example.com`).

### Pull Images Through the Proxy

**Using Web Interface**: Open `http://proxy.example.com` in your browser, enter the image name, and copy the generated command.

The web interface generates commands like:

```bash
docker pull proxy.example.com/ubuntu:latest && \
docker tag proxy.example.com/ubuntu:latest ubuntu:latest && \
docker rmi proxy.example.com/ubuntu:latest
```

## API Endpoints

- `GET /_/health`: Health check endpoint
- `GET /_/stats`: Cache statistics (requires authentication)
- `GET /v2/*`: OCI registry API proxy

### Statistics Response

```json
{
  "registry-1.docker.io": {
    "Hits": 42,
    "Misses": 8,
    "Evictions": 2,
    "Items": 15,
    "CurrentSize": 524288000,
    "MaxSize": 1073741824
  }
}
```

## Cache Behavior

- **Caching Strategy**: Only blob content is cached (manifests are not cached to ensure freshness)
- **Verification**: All cached blobs are verified using SHA256 digests
- **Eviction**: LRU eviction when cache size exceeds `cache_max_size`
- **Persistence**: Cache state is persisted to disk and restored on restart
- **Concurrency**: Thread-safe cache operations with minimal lock contention

## License

WTFPL
