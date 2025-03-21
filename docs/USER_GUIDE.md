# Panopticon Scanner - User Guide

## Introduction

Panopticon Scanner is a comprehensive network scanning and monitoring solution designed for Linux environments. This tool helps system administrators and security professionals discover, monitor, and analyze devices on their networks, with a focus on efficiency and thorough data collection.

## Installation

### System Requirements

- **Operating Systems**: Ubuntu 20.04+, Debian 11+, Fedora 34+
- **Disk Space**: Minimum 500MB for application, 5GB+ recommended for database and logs
- **Memory**: Minimum 2GB RAM
- **Dependencies**: Automatically installed during setup

### Installation Steps

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/panopticon-scanner.git
   ```

2. Run the installation script:
   ```bash
   cd panopticon-scanner
   ./install.sh
   ```

3. The installer will:
   - Install required dependencies
   - Set up the database
   - Configure system services
   - Create necessary directories

## Getting Started

### Starting the Application

1. **Backend Service**:
   ```bash
   systemctl start panopticond
   ```

2. **Frontend Application**:
   ```bash
   panopticon-ui
   ```

### Initial Configuration

1. Open the Panopticon UI application
2. Navigate to Settings → Configuration
3. Configure:
   - Network scan ranges
   - Scan frequency
   - Detection parameters
   - Report settings

## Main Features

### Network Scanning

Panopticon Scanner provides comprehensive network scanning capabilities:

- **Discovery Scans**: Identify all devices on specified network ranges
- **Port Scanning**: Detect open ports and services
- **OS Fingerprinting**: Identify operating systems
- **Service Detection**: Recognize running services
- **Scheduled Scans**: Configure automatic scanning at regular intervals

### Device Management

- View all discovered devices in a single dashboard
- Filter and sort by various attributes
- Track device history and changes
- Export device lists in multiple formats

### Reporting

Generate detailed reports for:
- Network inventory
- Open port analysis
- Change detection
- Service discovery
- Compliance status

Reports can be exported as PDF, HTML, or CSV.

### Alerts and Notifications

Configure alerts for:
- New device detection
- Changed services
- Suspicious open ports
- Compliance violations

Notifications can be sent via:
- Application UI
- System notifications
- Email (if configured)

## Advanced Features

### Custom Scan Templates

Create custom scan templates for specific use cases:
1. Navigate to Settings → Scan Templates
2. Select "New Template"
3. Configure scan parameters
4. Save the template

### Database Management

- **Backups**: Configure automatic weekly backups
- **Optimization**: Schedule database maintenance
- **Data Retention**: Configure retention policies

### Performance Tuning

Adjust performance settings:
- Scan concurrency
- Database optimization
- Resource limits

## Troubleshooting

### Common Issues

1. **Database Errors**
   - Check database integrity with: `panopticond --check-db`
   - Verify disk space availability

2. **Scanning Issues**
   - Ensure proper network connectivity
   - Verify firewall rules allow scanning
   - Check logs for specific error messages

3. **UI Not Responding**
   - Restart the UI application
   - Check backend service status
   - Verify sufficient system resources

### Log Files

- **Application Logs**: `/var/log/panopticon/app.log`
- **Scan Logs**: `/var/log/panopticon/scan.log`
- **UI Logs**: `~/.local/share/panopticon/ui.log`

## Appendix

### Command-Line Reference

```
panopticond [options]
  --config=FILE     Specify configuration file
  --scan-now        Run immediate scan
  --check-db        Run database integrity check
  --version         Show version information
  --help            Display this help message
```

```
panopticon-ui [options]
  --debug           Start in debug mode
  --disable-gpu     Disable GPU acceleration
  --help            Display this help message
```

### Configuration File Reference

Default location: `/etc/panopticon/config.yaml`

Key configuration options:
```yaml
scan:
  frequency: "0 * * * *"  # Cron format, default: hourly
  ranges:
    - "192.168.1.0/24"
    - "10.0.0.0/8"
  timeout: 300  # Seconds
  concurrency: 5

database:
  path: "/var/lib/panopticon/db.sqlite"
  backup:
    enabled: true
    frequency: "0 0 * * 0"  # Weekly on Sunday
    keep: 4  # Keep last 4 backups

logging:
  level: "info"  # debug, info, error
  max_size: 2048  # MB
  rotation: 7  # Days
```