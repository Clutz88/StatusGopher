# StatusGopher

A lightweight Go-based website status monitoring tool that periodically checks the availability and latency of websites.

## Features

- **Periodic Monitoring**: Checks websites every minute using HEAD requests
- **Concurrent Workers**: Uses 3 parallel workers to check multiple sites simultaneously
- **SQLite Storage**: Persists site URLs and check results in a local SQLite database
- **Graceful Shutdown**: Handles SIGINT and SIGTERM signals for clean shutdown
- **Latency Tracking**: Records response time for each check
- **Error Logging**: Captures and stores any errors encountered during checks

## Requirements

- Go 1.25+
- SQLite (via modernc.org/sqlite driver)

## Installation

```bash
go build -o statusgopher .
```

## Usage

```bash
./statusgopher
```

The service will:
1. Initialize a SQLite database at `./data/gopher.db`
2. Add default sites (google.com, github.com, go.dev) if they don't exist
3. Start monitoring immediately and every minute thereafter
4. Print results to stdout and save them to the database
5. Wait for Ctrl+C to shut down gracefully

## Database Schema

### sites table
| Column   | Type    | Description          |
|----------|---------|----------------------|
| id       | INTEGER | Primary key          |
| url      | TEXT    | Website URL (unique) |
| added_at | DATETIME| When site was added  |

### checks table
| Column     | Type    | Description              |
|------------|---------|--------------------------|
| id         | INTEGER | Primary key              |
| site_id    | INTEGER | Foreign key to sites     |
| status_code| INTEGER | HTTP status code         |
| latency_ms | INTEGER | Response time in ms      |
| checked_at | DATETIME| When check occurred      |
| error_msg  | TEXT    | Error message if failed  |

## Architecture

- **main.go**: Entry point, manages scheduling and worker pool
- **internal/checker**: Performs HTTP HEAD requests
- **internal/models**: Data structures (Site, CheckResult)
- **internal/database**: SQLite operations

## Configuration

Edit `main.go` to modify:
- Initial sites list (line 24)
- Check interval (line 33, default: 1 minute)
- Number of workers (line 75, default: 3)
- HTTP timeout (checker.go, line 17, default: 10 seconds)

## License

MIT
