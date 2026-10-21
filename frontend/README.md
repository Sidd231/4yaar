# 4Yaar Frontend

A private, invite-only communication platform built with React 19, TypeScript, Vite, Tailwind CSS v4, and native WebSockets and WebRTC. The UI adheres strictly to a minimalist Swiss-grid dark aesthetic with amber accents based on the Stitch UI Design System.

---

## 🚀 Features

- **Authentication & Security**:
  - JWT token pair handling (short-lived access tokens stored in memory, rotating refresh tokens stored via persistent storage abstraction).
  - Synchronized, single-flight automatic token refresh queuing on HTTP 401s.
  - Invite-only verification (`/register?token=<uuid>`) and role-based route protection (`ProtectedRoute` & `AdminRoute`).
- **Real-Time Group & Direct Messaging**:
  - Native WebSocket client (`ws://localhost:8080/ws/chat/:roomId?token=...`) with exponential backoff and jitter auto-reconnection.
  - Cursor-based message history pagination preserving scroll offsets.
  - Real-time message deduplication.
  - Channels creation and direct message list.
- **File Sharing (S3 Integration)**:
  - Multipart upload (`POST /files/upload`) with file size validation (<50MB) and upload progress tracking.
  - Attachment preview cards with on-demand presigned URL generation (`GET /files/:fileId`).
- **WebRTC Multi-Peer Video Calling**:
  - Multi-user mesh video calling over native browser `RTCPeerConnection` and signaling WebSockets (`ws://localhost:8080/ws/video/:roomId?token=...`).
  - Signaling state machine: `existing-peers`, `peer-joined`, `offer`, `answer`, `ice-candidate`, `peer-left`, `leave`.
  - ICE candidate buffering for unordered network packets.
  - Signature floating video overlay coexisting with live chat.
  - Mic mute/unmute, camera toggle, screen sharing, and minimization controls.
- **Admin Management Panel**:
  - Email invitation generation (`POST /invite/`).
  - Searchable user registry with debounced filtering (`GET /users/?search=`).

---

## 🛠️ Technology Stack

- **Framework**: React 19 + TypeScript
- **Bundler**: Vite 8
- **Styling**: Tailwind CSS v4 (configured with `@tailwindcss/vite` and design tokens)
- **Icons**: Lucide React
- **State Management**: Zustand
- **HTTP Client**: Axios with interceptors
- **Real-Time**: Native WebSocket & WebRTC

---

## 📦 Getting Started

### Prerequisites

- Node.js (v18+ recommended)
- **pnpm** (All package operations must use `pnpm`)
- 4Yaar Go Backend running on `http://localhost:8080`

### 1. Environment Configuration

Copy `.env.example` to `.env`:

```bash
cp .env.example .env
```

Default configuration:
```env
VITE_API_BASE_URL=http://localhost:8080/api/v1
VITE_WS_BASE_URL=ws://localhost:8080
```

### 2. Install Dependencies

```bash
pnpm install
```

### 3. Development Server

```bash
pnpm dev
```
The frontend dev server will start at `http://localhost:5173`.

### 4. Build & Typecheck

```bash
# Typecheck
pnpm typecheck

# Production build
pnpm build

# Preview build
pnpm preview
```

---

## 🌐 Backend Compatibility & Limitations Notes

1. **Direct Messages**: The backend supports room types `"group"` and `"direct"`. Both types are fully supported in the UI. For direct conversations, the backend currently accepts user-created direct rooms.
2. **WebRTC Mesh Scalability**: The current implementation utilizes full-mesh peer-to-peer WebRTC signaling as supported by the backend hub. For large-scale conferences (>6-8 participants), an SFU (Selective Forwarding Unit) such as Mediasoup or LiveKit is recommended in production.
3. **STUN/TURN in Production**: Development uses Google's public STUN servers. For production deployment across symmetric NATs and firewalls, a TURN server (e.g., `coturn`) should be configured.
