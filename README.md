# GOAT: Go CV Annotation Tool

GOAT is a computer vision annotation tool built with Go and React.

The repository is named `goat-cv` to make the project discoverable while keeping the product name short. GOAT stands for **Go CV Annotation Tool**, with the naming intentionally close to CVAT.

## Current Status

Phase 1 is implemented as a single-user, synchronous annotation workflow:

- Project and label management
- Image upload and serving
- BBox annotation on images
- Annotation save/load through the REST API
- Zoom, pan, rotate, and flip controls
- GOAT JSON export

## Tech Stack

- Backend: Go, Chi, SQLite, sqlc
- Frontend: React, Vite, TypeScript, Konva, Zustand, Tailwind CSS

## Development

Backend:

```sh
cd backend
go test ./...
go run ./cmd/server
```

Frontend:

```sh
cd frontend
npm install
npm test
npm run build
npm run dev
```

The frontend dev server proxies API requests to the backend on `localhost:8080`.

## Roadmap

Work is tracked as small branches and pull requests. See [docs/spec.md](docs/spec.md) for product scope and [docs/roadmap.md](docs/roadmap.md) for the planned PR sequence.
