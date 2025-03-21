# Panopticon Scanner - API Documentation

## Overview

The Panopticon Scanner provides a RESTful API that allows you to interact with the scanner programmatically. This document describes the available endpoints, request/response formats, and authentication mechanisms.

## Base URL

All API endpoints are prefixed with `/api/v1`.

## Authentication

API requests require authentication using a JWT token. To obtain a token:

1. Make a POST request to `/api/v1/auth/login` with your credentials.
2. The server will respond with a JWT token.
3. Include this token in the `Authorization` header of all subsequent requests:
   ```
   Authorization: Bearer <your_token>
   ```

## Response Format

All API responses are formatted as JSON objects with the following structure:

```json
{
  "status": "success",
  "data": { ... },
  "error": null
}
```

In case of an error:
```json
{
  "status": "error",
  "data": null,
  "error": {
    "code": "error_code",
    "message": "Error description"
  }
}
```

## Common Status Codes

- `200 OK`: Request successful
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Authentication required or failed
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

## Endpoints

### Authentication

#### Login

```
POST /api/v1/auth/login
```

Request:
```json
{
  "username": "admin",
  "password": "your_password"
}
```

Response:
```json
{
  "status": "success",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2023-04-01T12:00:00Z"
  },
  "error": null
}
```

### Device Management

#### List Devices

```
GET /api/v1/devices
```

Query Parameters:
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 25, max: 100)
- `sort`: Field to sort by (default: "last_seen")
- `order`: Sort order ("asc" or "desc", default: "desc")
- `filter`: JSON-encoded filter criteria

Response:
```json
{
  "status": "success",
  "data": {
    "devices": [
      {
        "id": "1",
        "ip_address": "192.168.1.1",
        "mac_address": "00:11:22:33:44:55",
        "hostname": "router.local",
        "os": "Linux",
        "first_seen": "2023-01-01T00:00:00Z",
        "last_seen": "2023-04-01T00:00:00Z",
        "ports": [
          {
            "port": 80,
            "protocol": "tcp",
            "service": "http"
          },
          {
            "port": 443,
            "protocol": "tcp",
            "service": "https"
          }
        ]
      }
    ],
    "total": 150,
    "page": 1,
    "limit": 25
  },
  "error": null
}
```

#### Get Device

```
GET /api/v1/devices/:id
```

Response:
```json
{
  "status": "success",
  "data": {
    "id": "1",
    "ip_address": "192.168.1.1",
    "mac_address": "00:11:22:33:44:55",
    "hostname": "router.local",
    "os": "Linux",
    "first_seen": "2023-01-01T00:00:00Z",
    "last_seen": "2023-04-01T00:00:00Z",
    "ports": [
      {
        "port": 80,
        "protocol": "tcp",
        "service": "http"
      },
      {
        "port": 443,
        "protocol": "tcp",
        "service": "https"
      }
    ],
    "scans": [
      {
        "id": "101",
        "timestamp": "2023-03-01T00:00:00Z",
        "status": "completed"
      }
    ]
  },
  "error": null
}
```

#### Create Device

```
POST /api/v1/devices
```

Request:
```json
{
  "ip_address": "192.168.1.1",
  "mac_address": "00:11:22:33:44:55",
  "hostname": "router.local",
  "os": "Linux"
}
```

Response:
```json
{
  "status": "success",
  "data": {
    "id": "1",
    "ip_address": "192.168.1.1",
    "mac_address": "00:11:22:33:44:55",
    "hostname": "router.local",
    "os": "Linux",
    "first_seen": "2023-04-01T00:00:00Z",
    "last_seen": "2023-04-01T00:00:00Z"
  },
  "error": null
}
```

#### Update Device

```
PUT /api/v1/devices/:id
```

Request:
```json
{
  "hostname": "new-hostname.local",
  "os": "FreeBSD"
}
```

Response:
```json
{
  "status": "success",
  "data": {
    "id": "1",
    "ip_address": "192.168.1.1",
    "mac_address": "00:11:22:33:44:55",
    "hostname": "new-hostname.local",
    "os": "FreeBSD",
    "first_seen": "2023-01-01T00:00:00Z",
    "last_seen": "2023-04-01T00:00:00Z"
  },
  "error": null
}
```

#### Delete Device

```
DELETE /api/v1/devices/:id
```

Response:
```json
{
  "status": "success",
  "data": {
    "message": "Device successfully deleted"
  },
  "error": null
}
```

### Scan Management

#### List Scans

```
GET /api/v1/scans
```

Query Parameters:
- `page`: Page number (default: 1)
- `limit`: Items per page (default: 25, max: 100)
- `sort`: Field to sort by (default: "timestamp")
- `order`: Sort order ("asc" or "desc", default: "desc")
- `status`: Filter by status (e.g., "completed", "in_progress", "failed")

Response:
```json
{
  "status": "success",
  "data": {
    "scans": [
      {
        "id": "101",
        "timestamp": "2023-03-01T00:00:00Z",
        "status": "completed",
        "duration": 120,
        "devices_found": 45,
        "template": "default"
      }
    ],
    "total": 50,
    "page": 1,
    "limit": 25
  },
  "error": null
}
```

#### Get Scan

```
GET /api/v1/scans/:id
```

Response:
```json
{
  "status": "success",
  "data": {
    "id": "101",
    "timestamp": "2023-03-01T00:00:00Z",
    "status": "completed",
    "duration": 120,
    "devices_found": 45,
    "template": "default",
    "parameters": {
      "ranges": ["192.168.1.0/24"],
      "ports": "1-1000",
      "rate": "5000"
    },
    "results": [
      {
        "device_id": "1",
        "ip_address": "192.168.1.1",
        "ports": [80, 443]
      }
    ]
  },
  "error": null
}
```

#### Create Scan

```
POST /api/v1/scans
```

Request:
```json
{
  "template": "default",
  "parameters": {
    "ranges": ["192.168.1.0/24"],
    "ports": "1-1000",
    "rate": "5000"
  },
  "schedule": "now"
}
```

Response:
```json
{
  "status": "success",
  "data": {
    "id": "102",
    "timestamp": "2023-04-01T00:00:00Z",
    "status": "scheduled",
    "template": "default",
    "parameters": {
      "ranges": ["192.168.1.0/24"],
      "ports": "1-1000",
      "rate": "5000"
    }
  },
  "error": null
}
```

#### Update Scan

```
PUT /api/v1/scans/:id
```

Request:
```json
{
  "status": "cancelled"
}
```

Response:
```json
{
  "status": "success",
  "data": {
    "id": "102",
    "timestamp": "2023-04-01T00:00:00Z",
    "status": "cancelled",
    "template": "default"
  },
  "error": null
}
```

#### Delete Scan

```
DELETE /api/v1/scans/:id
```

Response:
```json
{
  "status": "success",
  "data": {
    "message": "Scan successfully deleted"
  },
  "error": null
}
```

### System Status

#### Get System Status

```
GET /api/v1/status
```

Response:
```json
{
  "status": "success",
  "data": {
    "version": "1.0.0",
    "uptime": 86400,
    "memory_usage": {
      "used": 1024,
      "total": 8192,
      "percent": 12.5
    },
    "cpu_usage": 5.2,
    "disk_usage": {
      "used": 10240,
      "total": 102400,
      "percent": 10.0
    },
    "active_scans": 1,
    "scan_queue": 0
  },
  "error": null
}
```

#### Health Check

```
GET /api/v1/status/health
```

Response:
```json
{
  "status": "success",
  "data": {
    "status": "healthy",
    "components": {
      "api": "up",
      "database": "up",
      "scanner": "up"
    }
  },
  "error": null
}
```

#### Database Status

```
GET /api/v1/status/database
```

Response:
```json
{
  "status": "success",
  "data": {
    "size": 51200,
    "tables": {
      "devices": 150,
      "scans": 50,
      "scan_results": 7500
    },
    "last_backup": "2023-03-28T00:00:00Z",
    "last_optimization": "2023-03-31T00:00:00Z",
    "integrity_check": "ok"
  },
  "error": null
}
```

### Reports

#### Generate Report

```
POST /api/v1/reports
```

Request:
```json
{
  "type": "network_inventory",
  "format": "pdf",
  "parameters": {
    "date_range": {
      "start": "2023-03-01T00:00:00Z",
      "end": "2023-04-01T00:00:00Z"
    },
    "filters": {
      "os": "Linux"
    }
  }
}
```

Response:
```json
{
  "status": "success",
  "data": {
    "id": "r123",
    "status": "generating",
    "estimated_completion": "2023-04-01T00:01:00Z"
  },
  "error": null
}
```

#### Get Report Status

```
GET /api/v1/reports/:id
```

Response:
```json
{
  "status": "success",
  "data": {
    "id": "r123",
    "status": "completed",
    "type": "network_inventory",
    "format": "pdf",
    "created_at": "2023-04-01T00:00:00Z",
    "completed_at": "2023-04-01T00:01:00Z",
    "download_url": "/api/v1/reports/r123/download",
    "expires_at": "2023-04-08T00:00:00Z"
  },
  "error": null
}
```

#### Download Report

```
GET /api/v1/reports/:id/download
```

Response:
- Binary file with appropriate Content-Type header

## Error Codes

- `auth_failed`: Authentication failure
- `invalid_params`: Invalid request parameters
- `resource_not_found`: Requested resource not found
- `operation_failed`: Operation could not be completed
- `permission_denied`: User lacks permission for this operation
- `scan_in_progress`: Cannot modify a scan that is in progress
- `db_error`: Database operation error
- `internal_error`: Internal server error

## Rate Limiting

API requests are subject to rate limiting:
- 60 requests per minute for authenticated users
- 10 requests per minute for unauthenticated users

When rate limited, you will receive a `429 Too Many Requests` response with headers:
- `X-RateLimit-Limit`: Total number of requests allowed per time window
- `X-RateLimit-Remaining`: Number of requests remaining in current window
- `X-RateLimit-Reset`: Unix timestamp when the rate limit resets

## Websocket API

For real-time updates, connect to the websocket endpoint:
```
ws://your-server/api/v1/ws
```

Authentication:
- Connect with a query parameter containing your JWT token:
  ```
  ws://your-server/api/v1/ws?token=your_jwt_token
  ```

Message Format:
```json
{
  "type": "event_type",
  "data": { ... }
}
```

Event Types:
- `scan_status`: Updates about scan status changes
- `device_found`: New device discovered
- `device_changed`: Device information updated
- `report_status`: Report generation status updates

## API Versioning

The API is versioned using the path prefix. The current version is v1.

When new API versions are released, the older versions will be maintained for a transition period to ensure backward compatibility.