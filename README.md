**Build and Run**:
1. Build the frontend: `cd ui && npm run build`
2. Run the server: `go run main.go` (or `go build && OPENAI_API_KEY=<OPENAI_API_KEY> ./agent`)

The server will serve both the API and the frontend on port 8080.

**Development** (optional, for hot reload):
- Backend: `go run main.go`
- Frontend: `cd ui && npm run dev` (runs on port 5173)
