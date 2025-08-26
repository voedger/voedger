# Voedger

[![Go Version](https://img.shields.io/github/go-mod/go-version/voedger/voedger)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/voedger/voedger)](https://goreportcard.com/report/github.com/voedger/voedger)

Voedger is a high-performance, event-sourced application platform built with Go that implements CQRS (Command Query Responsibility Segregation) architecture. It provides a robust foundation for building scalable, distributed applications with strong consistency guarantees and real-time data processing capabilities.

## Table of Contents

- [Key Features](#key-features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Usage](#usage)
- [Tools](#tools)
- [Development](#development)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)

## Key Features

- **Event Sourcing & CQRS**: Built-in support for event sourcing with separate command and query processing
- **High Performance**: Optimized for high-throughput, low-latency applications
- **WebAssembly Extensions**: Support for WASM-based extensions and custom business logic
- **Projectors & Actualizers**: Real-time data projection and synchronization mechanisms
- **Multi-Storage Support**: Pluggable storage backends (memory, BoltDB, Cassandra, DynamoDB)
- **Distributed Architecture**: Support for multi-node clusters with automatic failover
- **Type-Safe Schema**: Strongly-typed schema definition with validation
- **REST API**: Automatic REST API generation from schema definitions
- **Real-time Monitoring**: Built-in monitoring and metrics collection
- **Package Management**: VPM (Voedger Package Manager) for extension management

## Architecture

Voedger implements a sophisticated architecture with the following core components:

- **Command Processor**: Handles write operations and business logic execution
- **Query Processor**: Manages read operations with optimized data access
- **Projectors**: Transform events into read models for efficient querying
- **Actualizers**: Synchronize projections with event streams (sync/async)
- **State Management**: Provides transactional access to different storage types
- **Extension Engine**: Executes WebAssembly-based business logic
- **Bus System**: Internal message passing and communication layer

## Quick Start

### Prerequisites

- Go 1.24 or higher
- Git

### Run Voedger Server

```bash
# Clone the repository
git clone https://github.com/voedger/voedger.git
cd voedger

# Run the server with in-memory storage
go run ./cmd/voedger --ihttp.Port 8888 --storage mem server
```

The server will start on `http://localhost:8888`. You can access:

- Static resources: `http://localhost:8888/static/sys/monitor/site/hello/`
- Monitoring dashboard: `http://localhost:8888/static/sys/monitor/site/main/`

### Create Your First Application

```bash
# Install VPM (Voedger Package Manager)
go install github.com/voedger/voedger/cmd/vpm@latest

# Initialize a new project
mkdir my-voedger-app && cd my-voedger-app
vpm mod init

# Compile the application
vpm compile

# Build the application
vpm build
```

## Installation

### From Source

```bash
git clone https://github.com/voedger/voedger.git
cd voedger
go build ./cmd/voedger
go build ./cmd/vpm
go build ./cmd/ctool
go build ./cmd/edger
```

### Install Tools

```bash
# Install all tools
go install github.com/voedger/voedger/cmd/voedger@latest
go install github.com/voedger/voedger/cmd/vpm@latest
go install github.com/voedger/voedger/cmd/ctool@latest
go install github.com/voedger/voedger/cmd/edger@latest
```

## Usage

### Server Configuration

```bash
# Run with custom configuration
./voedger --ihttp.Port 8080 --storage cas3 server

# Run with HTTPS and ACME
./voedger --ihttp.Port 443 --acme.domains example.com server
```

### Storage Options

- `mem` - In-memory storage (development)
- `bbolt` - BoltDB file storage
- `cas3` - Cassandra/ScyllaDB
- `dynamo` - AWS DynamoDB

## Tools

Voedger provides several command-line tools for different purposes:

### voedger
The main server application for running Voedger Community Edition.

```bash
# Start server
./voedger server

# Get help
./voedger --help
```

### vpm (Voedger Package Manager)
Package manager for Voedger applications, similar to npm or go modules.

```bash
# Initialize new project
vpm mod init

# Compile application
vpm compile

# Build application
vpm build

# Generate ORM
vpm orm

# Check compatibility
vpm compat baseline-folder
```

### ctool (Cluster Tool)
Deploy and manage Voedger clusters in production environments.

```bash
# Deploy single-node cluster
./ctool init n1 10.0.0.21

# Deploy 5-node cluster
./ctool init n5 host1 host2 host3 host4 host5 --ssh-key ./key.pem

# Upgrade cluster
./ctool upgrade --ssh-key ./key.pem

# Backup database
./ctool backup cron "0 2 * * *" --expire 30d --ssh-key ./key.pem
```

### edger
Edge node controller for managing distributed deployments.

```bash
# Start edger
./edger server

# Run edge operations
./edger run
```

## Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/voedger/voedger.git
cd voedger

# Build all components
go build ./cmd/voedger
go build ./cmd/vpm
go build ./cmd/ctool
go build ./cmd/edger

# Run tests
go test ./...
```

### Project Structure

```
voedger/
├── cmd/                    # Command-line tools
│   ├── voedger/           # Main server
│   ├── vpm/               # Package manager
│   ├── ctool/             # Cluster management
│   └── edger/             # Edge controller
├── pkg/                   # Core packages
│   ├── processors/        # Command/Query processors
│   ├── projectors/        # Event projectors
│   ├── state/             # State management
│   ├── storage/           # Storage backends
│   └── vvm/               # Voedger Virtual Machine
├── design/                # Architecture documentation
└── examples/              # Example applications
```

### Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

Please read our [contribution guidelines](CONTRIBUTING.md) for details on our code of conduct and development process.

## Documentation

- **[User Documentation](https://docs.voedger.io)** - Complete user guide and API reference
- **[Architecture & Internals](https://internals.voedger.io)** - Detailed technical documentation
- **[Knowledge Base](https://github.com/voedger/voedger-kb/blob/main/README.md)** - FAQs and troubleshooting
- **[Project Management](https://github.com/orgs/voedger/projects/11)** - Development roadmap and issues

### Additional Resources

- [Design Documents](./design/README.md) - Architecture and design decisions
- [Examples](./examples/) - Sample applications and use cases
- [API Reference](https://docs.voedger.io/api) - Complete API documentation

## Community

- **GitHub Discussions** - Ask questions and share ideas
- **Issues** - Report bugs and request features
- **Discord** - Join our community chat (coming soon)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Copyright

Copyright (c) 2021-present unTill Software Development Group B. V., unTill Pro, Ltd., Sigma-Soft, Ltd. and other contributors.

---

**Built with ❤️ by the Voedger team**

