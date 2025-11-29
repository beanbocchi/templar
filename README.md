
# Templar

Templar is a template management system with versioning and intelligent caching. It provides a RESTful API for managing template files with support for multiple storage backends, automatic caching, and background job processing.

## Features

- **Template Versioning**: Store and manage multiple versions of templates with unique version numbers

- **Intelligent Caching**: LRU-based cache layer with configurable size limits and automatic eviction

- **Background Processing**: Asynchronous upload jobs with real-time progress tracking

- **Job Tracking**: Monitor upload progress and job status through the API

## Architecture

Templar uses a layered architecture with clear separation of concerns:

- **Transport Layer**: HTTP handlers using Echo framework

- **Service Layer**: Business logic and orchestration

- **Storage Layer**: SQLite database for metadata

- **Object Store**: Pluggable storage backends (Local, Storj, Cache)

The system implements a multi-tier storage strategy:

1. **Cache Layer**: Fast local access with LRU eviction

2. **Local Storage**: Filesystem-based storage for quick access

3. **Storj Storage**: Decentralized cloud storage for durability

All storage operations are wrapped with synchronization to ensure thread safety.

## API Endpoints

Please check the postman collection
  
## Getting Started

### Prerequisites

- Go 1.25.0 or later

- SQLite (embedded, no separate installation needed)

### Installation

1. Clone the repository:

```bash

git  clone  https://github.com/beanbocchi/templar.git

cd  templar

```

2. Install dependencies:

```bash

go  mod  download

```

3. Configure the application by creating a config file. See `config/config.dev.yml` for an example.

4. Run database migrations (automatically handled on startup):

- Migrations are located in the `migrations/` directory

- The application will automatically run migrations on startup

5. Start the server:

```bash

go  run  main.go

```

The server will start on port 8080 by default.

### Configuration

Templar uses Viper for configuration management. Configuration can be provided via:

- YAML configuration files (default: `config/config.dev.yml` or `config/config.prod.yml`)

- Environment variables (prefixed with `APP_`)

- Command-line flags

Key configuration sections:

- **App**: Application name, job buffer size, JWT settings

- **Log**: Logging level, format, and source tracking

- **Objectstore**: Storage configuration for local, Storj, and cache layers

Example configuration structure: Please check the `config/config.default.yml` file

## Project Structure

```

templar/

├── config/ # Configuration management

├── internal/

│ ├── client/ # External client implementations

│ │ └── objectstore/ # Storage backends (local, storj, cache, sync)

│ ├── db/ # Database models and queries (sqlc generated)

│ ├── model/ # Domain models and errors

│ ├── service/ # Business logic layer

│ ├── transport/ # HTTP handlers and routing

│ └── utils/ # Utility functions (blake3, locker, progressr)

├── migrations/ # Database migration files

├── pkg/ # Shared packages (binder, response, validator)

├── queries/ # SQL queries for sqlc

└── main.go # Application entry point

```

## Storage Backends

### Local Storage

File-based storage on the local filesystem. Configured via `objectstore.local.root` and `objectstore.local.baseUrl`.

### Storj Storage

Decentralized cloud storage integration. Requires Storj access grant and bucket configuration.

### Cache Layer

Intelligent caching with LRU eviction policy. Automatically manages cache size and evicts least recently used items when the cache limit is reached.

## Development

### Running Tests

```bash

go  test  ./...

```

### Database Migrations

Migrations are managed using `golang-migrate`. Migration files are in the `migrations/` directory:

- `0001_init.up.sql`: Creates initial schema

- `0001_init.down.sql`: Drops schema

Migrations run automatically on application startup.

### Code Generation

The project uses `sqlc` for type-safe database queries. To regenerate database code:

```bash

sqlc  generate

```

## License

See LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
