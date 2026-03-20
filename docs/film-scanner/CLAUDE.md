# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Film Scanner is a CLI tool that scans video files in a directory, extracts video IDs (番号) from filenames, searches for metadata using the metatube-sdk-go library, and generates NFO files for media servers (Kodi/Jellyfin/Plex).

## Commands

```bash
# Build the application
go build -o film-scanner .

# Run tests (metatube-sdk-go submodule)
go test ./metatube-sdk-go/...

# Run with default settings (current directory)
./film-scanner

# Scan a specific directory
./film-scanner -dir /path/to/videos

# Use a specific provider
./film-scanner -dir /path/to/videos -provider javbus

# Download cover images
./film-scanner -dir /path/to/videos -download

# Specify output directory
./film-scanner -dir /path/to/videos -output /path/to/output

# Custom video extensions
./film-scanner -dir /path/to/videos -exts mp4,mkv,avi

# Enable verbose output
./film-scanner -dir /path/to/videos -v 1

# View help
./film-scanner --help
```

## Architecture

- **main.go**: CLI application entry point
  - `findVideoFiles()`: Recursively finds video files by extension
  - `extractNumber()`: Uses regex patterns to extract video IDs from filenames (supports FC2, heyzo, tokyo-hot, 1pondo, caribbeancom, etc.)
  - `searchMovie()`: Queries metatube-sdk-go engine for movie metadata
  - `saveNFOFile()`: Generates Kodi-compatible NFO XML files
  - `downloadCover()`: Downloads poster images with anti-hotlinking headers
  - `generateNFO()`: Creates NFO XML content with title, plot, actors, genres, runtime, rating, studio, etc.

- **metatube-sdk-go/**: Vendored SDK dependency (git submodule)
  - Provides metadata fetching from 20+ adult video providers
  - Uses in-memory database by default via `engine.Default()`
  - Supports SQLite/PostgreSQL for persistent storage

## Video ID Patterns

The tool recognizes these patterns in filenames:
- `ABC-123`, `ABC_123` (letter-prefix + number)
- `abc123` (concatenated)
- `123abc` (number + letter suffix)
- `FC2-123`, `heyzo-123`, `tokyo-hot`, `1pondo`, `caribbeancom`, `sog-123`

## Key Files

- `main.go` (388 lines): Complete CLI application
- `go.mod` / `go.sum`: Dependencies
- `metatube-sdk-go/`: External SDK (go.mod is separate)
