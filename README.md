# Panopticon Scanner

[![Go Report Card](https://goreportcard.com/badge/github.com/legio314/panopticon-scanner)](https://goreportcard.com/report/github.com/legio314/panopticon-scanner)
[![License](https://img.shields.io/github/license/legio314/panopticon-scanner)](https://github.com/legio314/panopticon-scanner/blob/master/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/legio314/panopticon-scanner)](https://golang.org/)

Panopticon Scanner is a comprehensive network scanning and monitoring solution designed for Linux environments. It provides automated discovery, monitoring, and analysis of network devices with an intuitive user interface.

## Features

- **Automated Network Scanning**: Scheduled scans of your network using configurable parameters
- **Device Tracking**: Monitor devices, open ports, and running services
- **Change Detection**: Identify new devices and changes to existing ones
- **Reporting**: Generate detailed reports in PDF, HTML, and CSV formats
- **User-Friendly UI**: Modern Electron-based interface with filtering and visualization
- **Optimized Performance**: Efficient resource usage with background processing
- **Detailed Logging**: Structured logs for easy troubleshooting
- **Database Management**: Automatic optimization and backup of the SQLite database
- **Self-Diagnostics**: Built-in health checks and performance monitoring
- **Linux Integration**: Proper system service integration on Linux distributions

## Project Structure

- `cmd/`: Application entry points
  - `panopticond/`: Backend daemon
  - `panopticon-ui/`: Frontend launcher
- `internal/`: Private application code
  - `api/`: API handlers
  - `config/`: Configuration management
  - `database/`: Database operations
  - `models/`: Data models
  - `scanner/`: Scanning functionality
- `pkg/`: Public libraries
- `ui/`: Electron/React frontend
- `configs/`: Configuration files
- `scripts/`: Build and utility scripts
- `data/`: Application data (scans, logs, etc.)
- `docs/`: Documentation

## Installation

### System Requirements

- **Operating Systems**: Ubuntu 20.04+, Debian 11+, Fedora 34+
- **Disk Space**: Minimum 500MB for application, 5GB+ recommended for database and logs
- **Memory**: Minimum 2GB RAM

### Quick Install

```bash
# Clone the repository
git clone https://github.com/legio314/panopticon-scanner.git
cd panopticon-scanner

# Run the installation script
./install.sh
```

For detailed installation instructions, see the [Installation Guide](docs/USER_GUIDE.md#installation).

## Development Setup

### Getting Started

1. Start the backend:
   ```bash
   cd cmd/panopticond
   go run main.go
   ```

2. Start the frontend:
   ```bash
   cd ui
   npm start
   ```

### Building for Production

Build scripts are provided in the `scripts/` directory:

```bash
# Build the backend daemon
./scripts/build_backend.sh

# Build the frontend application
./scripts/build_frontend.sh

# Build everything and package for distribution
./scripts/build_all.sh
```

## Documentation

- [User Guide](docs/USER_GUIDE.md) - Comprehensive user documentation
- [API Documentation](docs/API.md) - REST API reference
- [Configuration Guide](docs/CONFIGURATION.md) - Detailed configuration options
- [Developer Guide](docs/DEVELOPER.md) - Information for contributors
- [Contributing Guidelines](docs/CONTRIBUTING.md) - How to contribute to the project

## Contributing

Contributions are welcome! Please read our [Contributing Guidelines](docs/CONTRIBUTING.md) before submitting pull requests.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
