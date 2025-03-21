# Panopticon Scanner - Developer Guide

## Introduction

This guide provides comprehensive information for developers who want to contribute to the Panopticon Scanner project. It covers development environment setup, code organization, and contribution guidelines.

## Development Environment Setup

### Prerequisites

- Go 1.18+ for backend development
- Node.js 16+ for frontend development
- Git for version control
- Linux-based OS (Ubuntu 20.04+, Debian 11+, Fedora 34+)

### Local Development Setup

1. **Clone the repository**:
   ```bash
   git clone https://github.com/yourusername/panopticon-scanner.git
   cd panopticon-scanner
   ```

2. **Set up the backend**:
   ```bash
   go mod download
   cd cmd/panopticond
   go build .
   ```

3. **Set up the frontend**:
   ```bash
   cd ui
   npm install
   ```

### Running for Development

1. **Backend**:
   ```bash
   cd cmd/panopticond
   go run main.go
   ```

2. **Frontend**:
   ```bash
   cd ui
   npm start
   ```

## Project Structure

```
panopticon-scanner/
├── cmd/                    # Command-line applications
│   ├── panopticon-ui/      # UI application entry point
│   └── panopticond/        # Backend daemon entry point
├── configs/                # Configuration files
├── data/                   # Data storage
│   ├── backups/            # Database backups
│   ├── logs/               # Log files
│   ├── reports/            # Generated reports
│   └── scans/              # Scan results
├── docs/                   # Documentation
├── internal/               # Internal packages
│   ├── api/                # API handlers
│   ├── config/             # Configuration management
│   ├── database/           # Database operations
│   ├── models/             # Data models
│   └── scanner/            # Scanning functionality
├── pkg/                    # Public packages
├── scripts/                # Utility scripts
├── tests/                  # Integration and end-to-end tests
└── ui/                     # Frontend application
```

## Code Organization

### Backend (Go)

The backend is organized into several key packages:

1. **cmd/panopticond**: The main entry point for the backend daemon
2. **internal/api**: RESTful API handlers
3. **internal/config**: Configuration loading and validation
4. **internal/database**: Database operations and management
5. **internal/models**: Data models and structures
6. **internal/scanner**: Network scanning implementation

### Frontend (Electron/React)

The frontend is an Electron application with a React-based UI:

1. **ui/src/components**: Reusable UI components
2. **ui/src/pages**: Page layouts
3. **ui/src/services**: API client services
4. **ui/src/store**: State management
5. **ui/public**: Static assets

## Coding Standards

### Go Code Style

- Follow the [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use `gofmt` for consistent formatting
- Organize imports in three blocks: standard library, third-party, and local packages
- Document all exported functions, types, and constants
- Use meaningful variable names
- Handle errors explicitly
- Use `zerolog` for structured logging

### JavaScript/TypeScript Style

- Follow the ESLint configuration
- Use TypeScript for type safety
- Prefer functional components with hooks
- Document component props
- Use meaningful variable names
- Handle errors explicitly
- Keep components small and focused

## Testing

### Backend Testing

- Unit tests should be written for all packages
- Test files should be named `*_test.go`
- Integration tests are in the `tests/integration` directory
- Run tests with `go test ./...`

### Frontend Testing

- Unit tests should be written for all components
- Test files should be named `*.test.tsx` or `*.test.ts`
- Run tests with `npm test`

## Database Schema

The application uses SQLite with the following main tables:

1. **Devices**: Discovered network devices
2. **Scans**: Scan metadata
3. **ScanResults**: Results associated with scans
4. **Ports**: Open port information
5. **Services**: Detected service information

See the `internal/models/models.go` file for the complete schema definition.

## API Documentation

The REST API is versioned and accessible at `/api/v1`:

### Device Endpoints

- `GET /api/v1/devices`: List all devices
- `GET /api/v1/devices/:id`: Get device details
- `POST /api/v1/devices`: Create a new device (usually done by the scanner)
- `PUT /api/v1/devices/:id`: Update device information
- `DELETE /api/v1/devices/:id`: Remove a device

### Scan Endpoints

- `GET /api/v1/scans`: List all scans
- `GET /api/v1/scans/:id`: Get scan details
- `POST /api/v1/scans`: Create a new scan
- `PUT /api/v1/scans/:id`: Update scan status
- `DELETE /api/v1/scans/:id`: Remove a scan

### Status Endpoints

- `GET /api/v1/status`: Get system status
- `GET /api/v1/status/health`: Simple health check
- `GET /api/v1/status/database`: Get database statistics

## Submitting Contributions

1. **Fork the repository**
2. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. **Make your changes**
4. **Run tests**:
   ```bash
   go test ./...
   cd ui && npm test
   ```
5. **Commit your changes**:
   ```bash
   git commit -m "Add feature: your feature description"
   ```
6. **Push to your fork**:
   ```bash
   git push origin feature/your-feature-name
   ```
7. **Submit a pull request**

## Release Process

1. Update version numbers in:
   - `internal/config/version.go`
   - `ui/package.json`
2. Create and merge a release PR
3. Tag the release:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```
4. CI will build release artifacts

## Troubleshooting Development Issues

### Common Backend Issues

1. **Database connection problems**:
   - Check that SQLite is properly installed
   - Verify file permissions on the database file

2. **API not starting**:
   - Check for port conflicts
   - Verify configuration file

### Common Frontend Issues

1. **Node module issues**:
   - Try `npm clean-install`
   - Verify Node.js version

2. **Build failures**:
   - Check for TypeScript errors
   - Verify all dependencies are installed

## Performance Profiling

For performance profiling, use the built-in tools:

### Backend Profiling

```bash
# CPU profiling
go tool pprof http://localhost:8081/debug/pprof/profile?seconds=30

# Memory profiling
go tool pprof http://localhost:8081/debug/pprof/heap
```

### Frontend Profiling

Use Chrome DevTools with the React DevTools extension for frontend profiling.

## Security Considerations

- Never commit secrets or credentials
- Validate all user inputs
- Use parameterized SQL queries
- Follow the principle of least privilege
- Regularly update dependencies

## Additional Resources

- [Go Documentation](https://golang.org/doc/)
- [React Documentation](https://reactjs.org/docs/getting-started.html)
- [Electron Documentation](https://www.electronjs.org/docs)
- [SQLite Documentation](https://www.sqlite.org/docs.html)
- [Project Wiki](https://github.com/yourusername/panopticon-scanner/wiki)