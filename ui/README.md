# Agent Chat UI

A ChatGPT-like React interface for interacting with the Agent API.

## Features

- **Sidebar Navigation**: List of all conversations with ability to select, create, and delete
- **Chat Interface**: Clean message display with user and assistant messages
- **Real-time Messaging**: Send messages and receive responses from the agent
- **Conversation Management**: Create new conversations and manage existing ones

## Getting Started

### Prerequisites

- Node.js and npm installed
- Agent API server running on `http://localhost:8080` (or configure via environment variable)

### Installation

```bash
npm install
```

### Development

```bash
npm run dev
```

The UI will be available at `http://localhost:5173` (or the port Vite assigns).

### Build for Production

```bash
npm run build
```

### Configuration

You can configure the API URL by setting the `VITE_API_URL` environment variable:

```bash
VITE_API_URL=http://localhost:8080 npm run dev
```

## API Integration

The UI uses the following API endpoints:

- `POST /api/chat` - Send a message
- `GET /api/conversations` - List all conversations
- `GET /api/conversations/{id}` - Get a specific conversation

## Note on Delete Functionality

Currently, the delete functionality is client-side only. To persist deletions, you'll need to add a `DELETE /api/conversations/{id}` endpoint to the backend.
