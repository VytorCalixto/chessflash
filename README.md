# ChessFlash

ChessFlash is a chess analysis and training application that imports games from Chess.com, analyzes positions using Stockfish, and creates flashcards for spaced repetition learning. The application helps you identify mistakes, blunders, and missed opportunities in your games, then trains you on those positions using spaced repetition.

## Features

- Import games from Chess.com profiles
- Automatic position analysis using Stockfish engine
- Spaced repetition flashcards for training on mistakes and missed opportunities
- Opening performance statistics and analytics
- Web-based interface for reviewing games and flashcards
- SQLite database for data persistence

## Prerequisites

- Docker and Docker Compose installed on your system
- For local development: Go 1.24+ and Stockfish binary (if building without Docker)
  - You need to obtain the Stockfish binary separately. Download it from [Stockfish's official website](https://stockfishchess.org/download/) or install it via your system's package manager
  - The Stockfish binary must be available in your PATH, or you can specify its location using the `STOCKFISH_PATH` environment variable

## Quick Start with Docker

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd chess
   ```

2. Start the application:
   ```bash
   docker-compose up -d
   ```

3. Access the application:
   Open your browser and navigate to `http://localhost:8080`

4. Stop the application:
   ```bash
   docker-compose down
   ```

## Configuration

The application can be configured using environment variables or a `.env` file placed in the mounted volume at `/data/.env` inside the container.

### Environment Variables

- `ADDR` - Server address (default: `:8080`)
- `DB_PATH` - Database path (default: `file:/data/chessflash.db` in Docker)
- `STOCKFISH_PATH` - Path to Stockfish binary (default: `/usr/local/bin/stockfish` in Docker)
- `STOCKFISH_DEPTH` - Analysis depth (default: `18`)
- `STOCKFISH_MAX_TIME` - Max time per position in milliseconds, 0 = disabled (default: `0`)
- `LOG_LEVEL` - Logging level: `DEBUG`, `INFO`, `WARN`, `ERROR` (default: `INFO`)
- `ANALYSIS_WORKER_COUNT` - Number of analysis workers (default: `2`)
- `ANALYSIS_QUEUE_SIZE` - Analysis queue size (default: `64`)
- `IMPORT_WORKER_COUNT` - Number of import workers (default: `2`)
- `IMPORT_QUEUE_SIZE` - Import queue size (default: `32`)
- `ARCHIVE_LIMIT` - Archive limit (default: `0`)
- `MAX_CONCURRENT_ARCHIVE` - Max concurrent archives (default: `10`)

### Customizing Configuration

You can customize the configuration in two ways:

1. **Using a `.env` file**: Create a `.env` file in the mounted volume directory. The application will automatically load variables from this file.

2. **Using docker-compose.yml**: Add environment variables to the `environment` section in `docker-compose.yml`:
   ```yaml
   environment:
     - DB_PATH=file:/data/chessflash.db
     - STOCKFISH_PATH=/usr/local/bin/stockfish
     - STOCKFISH_DEPTH=20
     - LOG_LEVEL=DEBUG
   ```

## Volume Management

The Docker setup uses a named volume `chessflash_data` that is mounted to `/data` inside the container. This volume contains:

- `chessflash.db` - SQLite database file with all your games, positions, and flashcards
- `chessflash.db-shm` - SQLite shared memory file (auto-created)
- `chessflash.db-wal` - SQLite write-ahead log (auto-created)
- `.env` - Optional environment configuration file

### Backing Up Your Data

Since the database is stored in a Docker volume, you can backup your data by copying files from the volume:

```bash
# Find the volume location
docker volume inspect chess_chessflash_data

# Copy the database file (replace <volume-path> with actual path)
docker run --rm -v chess_chessflash_data:/data -v $(pwd):/backup alpine tar czf /backup/chessflash-backup.tar.gz -C /data .
```

### Restoring Data

To restore from a backup:

```bash
# Copy files back to the volume
docker run --rm -v chess_chessflash_data:/data -v $(pwd):/backup alpine sh -c "cd /data && tar xzf /backup/chessflash-backup.tar.gz"
```

### Moving Data Between Servers

Since ChessFlash uses SQLite, you can easily move your database between servers:

1. Stop the container on the source server
2. Copy the database files from the volume (as shown in backup section)
3. Transfer the backup file to the new server
4. Restore the files to the volume on the new server (as shown in restore section)
5. Start the container on the new server

The database file contains all your games, positions, flashcards, and review history, so your entire state will be preserved.

## Development Setup

For local development without Docker:

1. Install Go 1.24+

2. Obtain and configure the Stockfish binary:
   - Download Stockfish from [Stockfish's official website](https://stockfishchess.org/download/) or install via package manager:
     - Ubuntu/Debian: `sudo apt-get install stockfish`
     - macOS: `brew install stockfish`
     - Or download a pre-built binary for your platform
   - Make the binary executable: `chmod +x /path/to/stockfish`
   - You can either:
     - Add it to your PATH, or
     - Set the `STOCKFISH_PATH` environment variable to point to the binary location
     - Example: `export STOCKFISH_PATH=/usr/local/bin/stockfish`

3. Install dependencies:
   ```bash
   go mod download
   ```

4. Run the application:
   ```bash
   make run
   # or
   go run ./cmd/server
   ```

4. The application will use `chessflash.db` in the current directory by default.

## Building Manually

To build the Docker image manually:

```bash
docker build -t chessflash:latest .
```

### Using a Custom Stockfish Binary

By default, the Dockerfile expects the Stockfish binary to be located at `stockfish-ubuntu-x86-64-avx2/stockfish/stockfish-ubuntu-x86-64-avx2` relative to the project root. If you have a Stockfish binary in a different location, you can specify it using the `STOCKFISH_BINARY_PATH` build argument:

**Using docker build:**
```bash
docker build --build-arg STOCKFISH_BINARY_PATH=path/to/your/stockfish -t chessflash:latest .
```

**Using docker-compose:**
Add the build argument to your `docker-compose.yml`:
```yaml
services:
  chessflash:
    build:
      context: .
      dockerfile: Dockerfile
      args:
        STOCKFISH_BINARY_PATH: path/to/your/stockfish
```

The path should be relative to the project root (where the Dockerfile is located). The binary will be copied to `/usr/local/bin/stockfish` inside the container.

To run the container manually:

```bash
docker run -d \
  --name chessflash \
  -p 8080:8080 \
  -v chessflash_data:/data \
  -e DB_PATH=file:/data/chessflash.db \
  -e STOCKFISH_PATH=/usr/local/bin/stockfish \
  chessflash:latest
```

## Project Structure

- `cmd/server/` - Main application entry point
- `internal/` - Internal packages (API, services, repositories, etc.)
- `web/` - Web templates and static assets
- `internal/db/migrations/` - Database migration files

Note: The Stockfish binary is not included in this repository. You need to obtain it separately as described in the Prerequisites section.

## License

This project is licensed under the terms of the GNU Affero General Public License v3.0. See the [LICENSE](./LICENSE) file for details.
