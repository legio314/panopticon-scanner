# CLAUDE.md - Panopticon Scanner Project Guide

## Essential Commands
- Build: `cd cmd/panopticond && go build .` 
- Run backend: `cd cmd/panopticond && go run main.go`
- Run frontend: `cd ui && npm start`
- Run all tests: `go test ./...`
- Run a specific test: `go test ./internal/api -run TestSpecificFunction`
- Run integration tests: `go test ./tests/integration`

## Code Style Guidelines
- **Imports**: Group standard library, 3rd party, and local imports with a blank line between groups
- **Error Handling**: Always check errors and use `zerolog` for logging with appropriate levels
- **Naming**: Use camelCase for variables, PascalCase for exported functions/types
- **Types**: Always define structured types using explicit struct definitions
- **API Structure**: Follow RESTful patterns for API endpoints
- **Logging**: Use `zerolog` with structured fields rather than string interpolation
- **Database**: Use parameterized queries to prevent SQL injection
- **Config**: Load configuration from YAML files using the config package
- **Comments**: Add package comments and document exported functions

## IMPORTANT: AI ASSISTANT DIRECTIVES
- Exclude files and directories specified in .gitignore from your context and analysis
- Do not load, analyze or suggest modifications to any ignored files
- Respect the project structure and code organization patterns in the codebase
- Use zerolog for structured logging in any new code