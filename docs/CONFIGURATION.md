# Panopticon Scanner - Configuration Guide

## Overview

Panopticon Scanner uses YAML configuration files to control its behavior. This document describes all available configuration options, their meanings, and recommended values.

## Configuration File Location

The default configuration file is located at:

```
/etc/panopticon/config.yaml
```

You can specify an alternative configuration file when starting the application:

```bash
panopticond --config=/path/to/custom-config.yaml
```

## Configuration Sections

The configuration file is organized into logical sections:

1. **General** - Application-wide settings
2. **Network** - Network scanning parameters
3. **Database** - Database connection and optimization settings
4. **Logging** - Logging configuration
5. **API** - API server settings
6. **UI** - User interface settings
7. **Reports** - Report generation settings
8. **Security** - Security-related settings
9. **Notifications** - Alert and notification settings
10. **Advanced** - Advanced performance tuning options

## Sample Configuration

Below is a sample configuration file with commonly used settings:

```yaml
# Main configuration file for Panopticon Scanner

general:
  application_name: "Panopticon Scanner"
  environment: "production"  # production, development, testing
  temp_dir: "/tmp/panopticon"
  data_dir: "/var/lib/panopticon"

network:
  scan:
    frequency: "0 * * * *"  # Cron format, default: hourly
    ranges:
      - "192.168.1.0/24"
      - "10.0.0.0/8"
    exclude_ranges:
      - "10.0.0.5/32"  # Exclude specific IP
    ports: "1-1024,3389,8080,8443"  # Ports to scan
    timeout: 300  # Scan timeout in seconds
    concurrency: 5  # Number of concurrent scans
    rate: 5000  # Packets per second
    disable_ping: true  # Scan hosts that don't respond to ping
    os_detection: true  # Enable OS detection
    service_detection: true  # Enable service detection
  templates:
    quick:
      ports: "22,80,443,3389"
      timeout: 60
      rate: 10000
    thorough:
      ports: "1-65535"
      timeout: 600
      rate: 2000
      os_detection: true
      service_detection: true
    stealth:
      ports: "21-23,25,53,80,443,8080,8443"
      rate: 100
      disable_ping: true

database:
  type: "sqlite"
  path: "/var/lib/panopticon/db.sqlite"
  optimize_frequency: "0 0 * * *"  # Daily at midnight
  backup:
    enabled: true
    frequency: "0 0 * * 0"  # Weekly on Sunday
    path: "/var/lib/panopticon/backups"
    keep: 4  # Keep last 4 backups
  retention:
    enabled: true
    duration: "730d"  # 2 years retention

logging:
  level: "info"  # debug, info, warning, error
  format: "json"
  output: "file"  # file, stdout, both
  file_path: "/var/log/panopticon/app.log"
  max_size: 2048  # 2GB maximum log size in MB
  rotation: 7  # Rotate logs every 7 days
  syslog:
    enabled: false
    facility: "local0"

api:
  host: "127.0.0.1"
  port: 8080
  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:3000"
  rate_limit:
    enabled: true
    requests_per_minute: 60
  timeout: 30  # Request timeout in seconds
  auth:
    enabled: true
    jwt_secret: ""  # Set via environment variable PANOPTICON_JWT_SECRET
    token_expiration: 86400  # 24 hours in seconds

ui:
  theme: "dark"  # dark, light, system
  auto_refresh: 300  # Auto-refresh interval in seconds
  date_format: "YYYY-MM-DD HH:mm:ss"
  accessibility:
    high_contrast: false
    large_font: false
    colorblind_mode: false

reports:
  output_dir: "/var/lib/panopticon/reports"
  default_format: "pdf"  # pdf, html, csv
  logo_path: "/etc/panopticon/logo.png"
  retention: 30  # Days to keep generated reports
  templates_dir: "/etc/panopticon/report-templates"

security:
  encryption_key: ""  # Set via environment variable PANOPTICON_ENCRYPTION_KEY
  allow_local_auth: true
  failed_login_delay: 3  # Seconds to wait after failed login
  failed_login_max: 5  # Lock account after this many failed attempts
  session_timeout: 3600  # Inactive session timeout in seconds

notifications:
  enabled: true
  types:
    new_device: true
    service_change: true
    scan_complete: false
    system_error: true
  methods:
    ui: true
    email:
      enabled: false
      smtp_host: "smtp.example.com"
      smtp_port: 587
      smtp_user: ""
      smtp_password: ""
      from_address: "panopticon@example.com"
      to_addresses:
        - "admin@example.com"
    webhook:
      enabled: false
      url: "https://example.com/webhook"
      headers:
        Authorization: "Bearer your_token"

advanced:
  performance:
    database_cache_size: 2048  # Cache size in MB
    scan_buffer_size: 1024  # Buffer size in KB
    max_memory_usage: 4096  # Maximum memory usage in MB
    worker_threads: 4  # Number of worker threads
  timeouts:
    database: 30  # Database operation timeout in seconds
    network: 60  # Network operation timeout in seconds
  debug:
    enabled: false
    profiling: false
    verbose_logging: false
```

## Configuration Reference

### General Settings

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `general.application_name` | Application name displayed in UI and logs | "Panopticon Scanner" | No |
| `general.environment` | Deployment environment | "production" | No |
| `general.temp_dir` | Temporary file storage directory | "/tmp/panopticon" | No |
| `general.data_dir` | Main data storage directory | "/var/lib/panopticon" | Yes |

### Network Settings

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `network.scan.frequency` | Cron expression for scan schedule | "0 * * * *" | No |
| `network.scan.ranges` | List of CIDR ranges to scan | - | Yes |
| `network.scan.exclude_ranges` | CIDR ranges to exclude from scanning | - | No |
| `network.scan.ports` | Ports to scan | "1-1024" | No |
| `network.scan.timeout` | Scan timeout in seconds | 300 | No |
| `network.scan.concurrency` | Number of concurrent scans | 5 | No |
| `network.scan.rate` | Packets per second | 5000 | No |
| `network.scan.disable_ping` | Scan hosts that don't respond to ping | true | No |
| `network.scan.os_detection` | Enable OS detection | true | No |
| `network.scan.service_detection` | Enable service detection | true | No |

### Database Settings

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `database.type` | Database type (only sqlite supported) | "sqlite" | No |
| `database.path` | Path to SQLite database file | "/var/lib/panopticon/db.sqlite" | Yes |
| `database.optimize_frequency` | Cron expression for optimization schedule | "0 0 * * *" | No |
| `database.backup.enabled` | Enable automated backups | true | No |
| `database.backup.frequency` | Cron expression for backup schedule | "0 0 * * 0" | No |
| `database.backup.path` | Directory for storing backups | "/var/lib/panopticon/backups" | Yes if backups enabled |
| `database.backup.keep` | Number of backups to retain | 4 | No |
| `database.retention.enabled` | Enable data retention policy | true | No |
| `database.retention.duration` | Data retention period | "730d" (2 years) | Yes if retention enabled |

### Logging Settings

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `logging.level` | Logging level | "info" | No |
| `logging.format` | Log format | "json" | No |
| `logging.output` | Where to send logs | "file" | No |
| `logging.file_path` | Path to log file | "/var/log/panopticon/app.log" | Yes if output=file |
| `logging.max_size` | Maximum log size in MB | 2048 | No |
| `logging.rotation` | Log rotation period in days | 7 | No |
| `logging.syslog.enabled` | Enable syslog integration | false | No |
| `logging.syslog.facility` | Syslog facility to use | "local0" | No |

### API Settings

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `api.host` | Host to bind API server | "127.0.0.1" | No |
| `api.port` | Port for API server | 8080 | No |
| `api.cors.enabled` | Enable CORS | true | No |
| `api.cors.allowed_origins` | Origins allowed for CORS | - | Yes if CORS enabled |
| `api.rate_limit.enabled` | Enable API rate limiting | true | No |
| `api.rate_limit.requests_per_minute` | Rate limit threshold | 60 | No |
| `api.timeout` | API request timeout in seconds | 30 | No |
| `api.auth.enabled` | Enable API authentication | true | No |
| `api.auth.jwt_secret` | Secret for JWT tokens | - | Yes if auth enabled |
| `api.auth.token_expiration` | Token validity period in seconds | 86400 | No |

### UI Settings

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `ui.theme` | Default UI theme | "dark" | No |
| `ui.auto_refresh` | Auto-refresh interval in seconds | 300 | No |
| `ui.date_format` | Date/time format | "YYYY-MM-DD HH:mm:ss" | No |
| `ui.accessibility.high_contrast` | Enable high contrast mode | false | No |
| `ui.accessibility.large_font` | Enable large font size | false | No |
| `ui.accessibility.colorblind_mode` | Enable colorblind-friendly mode | false | No |

### Report Settings

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `reports.output_dir` | Directory for saving reports | "/var/lib/panopticon/reports" | No |
| `reports.default_format` | Default report format | "pdf" | No |
| `reports.logo_path` | Path to logo file for reports | - | No |
| `reports.retention` | Days to keep generated reports | 30 | No |
| `reports.templates_dir` | Directory containing report templates | "/etc/panopticon/report-templates" | No |

### Security Settings

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `security.encryption_key` | Key for encrypting sensitive data | - | Yes |
| `security.allow_local_auth` | Allow local authentication | true | No |
| `security.failed_login_delay` | Seconds to wait after failed login | 3 | No |
| `security.failed_login_max` | Maximum failed login attempts | 5 | No |
| `security.session_timeout` | Session timeout in seconds | 3600 | No |

### Notification Settings

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `notifications.enabled` | Enable notifications | true | No |
| `notifications.types.new_device` | Notify on new device discovery | true | No |
| `notifications.types.service_change` | Notify on service changes | true | No |
| `notifications.types.scan_complete` | Notify when scans complete | false | No |
| `notifications.types.system_error` | Notify on system errors | true | No |
| `notifications.methods.ui` | Enable UI notifications | true | No |
| `notifications.methods.email.enabled` | Enable email notifications | false | No |
| `notifications.methods.email.smtp_host` | SMTP server hostname | - | Yes if email enabled |
| `notifications.methods.email.smtp_port` | SMTP server port | 587 | No |
| `notifications.methods.email.smtp_user` | SMTP username | - | No |
| `notifications.methods.email.smtp_password` | SMTP password | - | No |
| `notifications.methods.email.from_address` | Sender email address | - | Yes if email enabled |
| `notifications.methods.email.to_addresses` | Recipient email addresses | - | Yes if email enabled |
| `notifications.methods.webhook.enabled` | Enable webhook notifications | false | No |
| `notifications.methods.webhook.url` | Webhook URL | - | Yes if webhook enabled |
| `notifications.methods.webhook.headers` | Webhook HTTP headers | - | No |

### Advanced Settings

| Option | Description | Default | Required |
|--------|-------------|---------|----------|
| `advanced.performance.database_cache_size` | Database cache size in MB | 2048 | No |
| `advanced.performance.scan_buffer_size` | Network scan buffer size in KB | 1024 | No |
| `advanced.performance.max_memory_usage` | Maximum memory usage in MB | 4096 | No |
| `advanced.performance.worker_threads` | Number of worker threads | 4 | No |
| `advanced.timeouts.database` | Database operation timeout in seconds | 30 | No |
| `advanced.timeouts.network` | Network operation timeout in seconds | 60 | No |
| `advanced.debug.enabled` | Enable debug mode | false | No |
| `advanced.debug.profiling` | Enable performance profiling | false | No |
| `advanced.debug.verbose_logging` | Enable verbose logging | false | No |

## Environment Variables

Sensitive configuration values can be set via environment variables instead of being stored in the configuration file:

| Environment Variable | Configuration Option |
|----------------------|----------------------|
| `PANOPTICON_JWT_SECRET` | `api.auth.jwt_secret` |
| `PANOPTICON_ENCRYPTION_KEY` | `security.encryption_key` |
| `PANOPTICON_SMTP_PASSWORD` | `notifications.methods.email.smtp_password` |
| `PANOPTICON_DB_PATH` | `database.path` |

## Hot Reload

Panopticon Scanner supports hot reloading of configuration changes. When you modify the configuration file, the changes will be automatically applied without restarting the application, except for:

- Database connection parameters
- API server host/port
- Authentication settings

These settings require a restart to take effect.

## Configuration Management

You can manage configuration through the UI:

1. Navigate to Settings → Configuration
2. Modify the settings as needed
3. Click Save to apply changes

The UI will validate your changes before applying them to prevent configuration errors.

## Troubleshooting

### Common Configuration Issues

1. **Database Connection Issues**
   - Ensure the database path is correct and the directory exists
   - Verify file permissions allow the application to read/write to the database

2. **Scanning Issues**
   - Check that the network ranges are correctly formatted (CIDR notation)
   - Ensure port specifications are valid (e.g., "80,443" or "1-1024")
   - Verify that rate limits are reasonable for your network environment

3. **Permission Issues**
   - Ensure log directories and data directories have appropriate permissions
   - For system integration, verify the application has the necessary privileges

### Configuration Validation

You can validate your configuration without starting the application:

```bash
panopticond --validate-config=/path/to/config.yaml
```

This will check your configuration file for errors and provide feedback.