# MongoDB Syncer

A Go application that orchestrates MongoDB synchronization using the mongosync tool with REST API monitoring and progress tracking.

## Architecture

The application is structured into several packages:

- `pkg/config`: Configuration management from environment variables
- `pkg/downloader`: Handles mongosync binary downloading and extraction
- `pkg/api`: REST API client for mongosync operations with progress monitoring
- `pkg/mongosync`: Process management for the mongosync binary

## Usage

### Environment Variables

Set the following environment variables:

```bash
export MONGOSYNC_SOURCE="mongodb://source-cluster-connection-string"
export MONGOSYNC_TARGET="mongodb://target-cluster-connection-string"
```

### Running the Application

```bash
go run main.go
```

Or build and run:

```bash
go build -o mongosyncer
./mongosyncer
```

## Workflow

1. **Binary Check**: Ensures mongosync binary is available (downloads if needed)
2. **Process Start**: Starts the mongosync process with the configured source/target
3. **Sync Initiation**: Sends POST request to `/api/v1/start` to begin synchronization
4. **Progress Monitoring**: Polls `/api/v1/progress` every 5 seconds until `canCommit` is true
5. **Commit**: Sends POST request to `/api/v1/commit` to finalize the sync
6. **Completion**: Waits for the mongosync process to complete

## API Endpoints Used

- `POST /api/v1/start` - Initiates the sync process
- `GET /api/v1/progress` - Checks sync progress and canCommit status
- `POST /api/v1/commit` - Commits the synchronization

## Docker Support

The application includes a Dockerfile for containerized deployments.
