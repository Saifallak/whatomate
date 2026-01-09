<a href="https://zerodha.tech"><img src="https://zerodha.tech/static/images/github-badge.svg" align="right" alt="Zerodha Tech Badge" /></a>
# Whatomate

A modern WhatsApp Business Platform built with Go (Fastglue) and Vue.js (shadcn-vue).

## Features

- **Multi-tenant Architecture**: Support multiple organizations with isolated data
- **Role-Based Access Control**: Three roles (Admin, Manager, Agent) with granular permissions
- **WhatsApp Cloud API Integration**: Connect with Meta's WhatsApp Business API
- **Real-time Chat**: Live messaging with WebSocket support
- **Template Management**: Create and manage message templates
- **Bulk Messaging**: Send campaigns to multiple contacts with retry support for failed messages
- **Chatbot Automation**:
  - Keyword-based auto-replies
  - Conversation flows with branching logic and skip conditions
  - AI-powered responses (OpenAI, Anthropic, Google)
  - Agent transfer support
- **Canned Responses**: Pre-defined quick replies for agents
  - Organization-wide shared responses
  - Category-based organization (Greetings, Support, Sales, etc.)
  - Slash command support (type `/shortcut` in chat)
  - Dynamic placeholders (`{{contact_name}}`, `{{phone_number}}`)
- **Analytics Dashboard**: Track messages, engagement, and campaign performance

## Screenshots

<details>
<summary>Click to view screenshots</summary>

### Dashboard
![Dashboard](docs/public/images/01-dashboard.png)

### Chatbot Settings
![Chatbot Settings](docs/public/images/02-chatbot-settings.png)

### Keyword Rules
![Keyword Rules](docs/public/images/03-keyword-rules.png)
![Keyword Rule Editor](docs/public/images/04-keyword-rule-editor.png)

### AI Contexts
![AI Contexts](docs/public/images/05-ai-contexts.png)
![AI Context Editor](docs/public/images/06-ai-context-editor.png)

### Conversation Flows
![Conversation Flows](docs/public/images/07-conversation-flows.png)
![Conversation Flow Builder](docs/public/images/08-conversation-flow-builder.png)

### WhatsApp Flows
![WhatsApp Flows](docs/public/images/09-whatsapp-flows.png)
![WhatsApp Flow Builder](docs/public/images/10-whatsapp-flow-builder.png)

### Templates
![Templates](docs/public/images/11-templates.png)
![Template Editor](docs/public/images/12-template-editor.png)

### Campaigns
![Campaigns](docs/public/images/13-campaigns.png)
![Campaign Details](docs/public/images/14-campaign-details.png)

### Settings
![Settings](docs/public/images/15-settings.png)
![Account Settings](docs/public/images/16-account-settings.png)

</details>

## Tech Stack

### Backend
- **Go 1.21+** with [Fastglue](https://github.com/zerodha/fastglue) (fasthttp-based HTTP framework)
- **PostgreSQL** for data storage with GORM v2
- **Redis** for caching, pub/sub, and job queues (Redis Streams)
- **JWT** for authentication
- **Worker Service** for reliable campaign processing

### Frontend
- **Vue 3** with Composition API and TypeScript
- **Vite** for build tooling
- **shadcn-vue** / Radix Vue for UI components
- **TailwindCSS** for styling
- **Pinia** for state management
- **Vue Query** for server state

## Project Structure

```
whatomate/
├── cmd/
│   └── whatomate/        # Single binary with server/worker subcommands
├── internal/
│   ├── config/           # Configuration management
│   ├── database/         # Database connections
│   ├── handlers/         # HTTP handlers
│   ├── middleware/       # HTTP middleware
│   ├── models/           # Data models
│   ├── queue/            # Redis Streams job queue
│   ├── services/         # Business logic
│   └── worker/           # Worker service for job processing
├── docker/               # Docker configuration
├── frontend/             # Vue.js frontend
│   ├── src/
│   │   ├── components/   # UI components
│   │   ├── views/        # Page views
│   │   ├── stores/       # Pinia stores
│   │   ├── services/     # API services
│   │   └── lib/          # Utilities
│   └── ...
├── config.example.toml   # Example configuration
├── Makefile              # Build commands
└── README.md
```

## Getting Started

### Quick Start (Docker)

The fastest way to get started:

```bash
# Clone the repository
git clone https://github.com/shridarpatil/whatomate.git
cd whatomate

# Start all services (PostgreSQL, Redis, Server)
make docker-up

# Access the application
open http://localhost:8080
```

**Default login:** `admin@admin.com` / `admin`

### Manual Installation

#### Prerequisites

- Go 1.21+
- Node.js 18+
- PostgreSQL 14+
- Redis 6+

#### Step 1: Clone and Configure

```bash
git clone https://github.com/shridarpatil/whatomate.git
cd whatomate
cp config.example.toml config.toml
```

#### Step 2: Setup PostgreSQL

```bash
# Create database
createdb whatomate

# Or using psql
psql -c "CREATE DATABASE whatomate;"

# Or with custom user
psql -c "CREATE USER whatomate WITH PASSWORD 'whatomate123';"
psql -c "CREATE DATABASE whatomate OWNER whatomate;"
```

Update `config.toml` with your database credentials:

```toml
[database]
host = "localhost"
port = 5432
user = "whatomate"
password = "whatomate123"
name = "whatomate"
```

#### Step 3: Setup Redis

```bash
# macOS
brew install redis && brew services start redis

# Ubuntu/Debian
sudo apt install redis-server && sudo systemctl start redis

# Docker (alternative)
docker run -d -p 6379:6379 redis:alpine
```

#### Step 4: Start Backend

```bash
# Install Go dependencies
go mod download

# Run migrations and start server
make run-migrate
```

#### Step 5: Start Frontend (separate terminal)

```bash
cd frontend
npm install
npm run dev
```

#### Step 6: Access Application

- **Frontend:** http://localhost:3000
- **API:** http://localhost:8080
- **Login:** `admin@admin.com` / `admin`

> **Important:** Change the admin password after first login.

### Docker Commands

```bash
make docker-up       # Start all services
make docker-down     # Stop all services
make docker-logs     # View logs
make docker-build    # Rebuild images

# Scale workers for high-volume campaigns
cd docker && docker compose up -d --scale worker=3
```

### Troubleshooting

| Issue | Solution |
|-------|----------|
| `connection refused` (database) | Ensure PostgreSQL is running: `pg_isready` |
| `connection refused` (redis) | Ensure Redis is running: `redis-cli ping` |
| `database does not exist` | Create it: `createdb whatomate` |
| `permission denied` | Check database user/password in `config.toml` |
| `port already in use` | Change port in `config.toml` or stop conflicting service |
| Frontend can't reach API | Ensure backend is running on port 8080 |

## Building

### Development Build

```bash
make build    # Backend only (no frontend embedded)
```

For development, run backend and frontend separately:
- Backend: `make run` or `./whatomate server`
- Frontend: `cd frontend && npm run dev`

### Production Build

```bash
make build-prod    # Single binary with embedded frontend
```

This creates a self-contained binary that serves both API and frontend. No separate frontend server needed.

| Command | Frontend | Use Case |
|---------|----------|----------|
| `make build` | Not embedded | Development (run frontend separately) |
| `make build-prod` | Embedded | Production (single binary deployment) |

## CLI Usage

Whatomate uses a single binary with subcommands:

```bash
./whatomate <command> [options]
```

### Commands

| Command | Description |
|---------|-------------|
| `server` | Start the API server (with optional embedded workers) |
| `worker` | Start background workers only (no API server) |
| `version` | Show version information |
| `help` | Show help message |

### Server Options

```bash
./whatomate server [options]

Options:
  -config string    Path to config file (default "config.toml")
  -migrate          Run database migrations on startup
  -workers int      Number of embedded workers, 0 to disable (default 1)
```

### Worker Options

```bash
./whatomate worker [options]

Options:
  -config string    Path to config file (default "config.toml")
  -workers int      Number of workers to run (default 1)
```

### Deployment Examples

```bash
# All-in-one (API + workers)
./whatomate server

# API only (no workers)
./whatomate server -workers=0

# Separate API and workers (distributed deployment)
./whatomate server -workers=0    # On API server
./whatomate worker -workers=4    # On worker server(s)
```

## Configuration

Copy `config.example.toml` to `config.toml` and update the values:

```toml
[app]
name = "Whatomate"
environment = "development"
debug = true

[server]
host = "0.0.0.0"
port = 8080

[database]
host = "localhost"
port = 5432
user = "whatomate"
password = "your-password"
name = "whatomate"
ssl_mode = "disable"

[redis]
host = "localhost"
port = 6379
password = ""
db = 0

[jwt]
secret = "your-jwt-secret"
access_expiry_mins = 15
refresh_expiry_days = 7

[storage]
type = "local"       # local or s3
local_path = "./uploads"
```

> **Note:** WhatsApp credentials and AI API keys are configured via the UI and stored in the database.

## Worker Service

The worker service handles bulk campaign message processing using Redis Streams for reliable job queuing.

### Architecture

```
┌─────────────┐     Redis Streams      ┌─────────────┐
│   Server    │ ─────────────────────► │  Worker 1   │
│  (enqueue)  │   whatomate:campaigns  ├─────────────┤
└─────────────┘         │              │  Worker 2   │
                        └─────────────►│  Worker N   │
                                       └─────────────┘
```

### Features

- **Reliable Processing**: Jobs persist in Redis until acknowledged
- **Horizontal Scaling**: Add more workers to increase throughput
- **Graceful Shutdown**: Workers complete current job before stopping
- **Automatic Recovery**: Stale jobs are reclaimed on worker startup

### Running Workers

**Embedded Mode** (default): The server runs workers internally.

```bash
# Default: 1 worker
./whatomate server

# Run with 3 embedded workers
./whatomate server -workers=3

# Disable embedded workers (use standalone workers only)
./whatomate server -workers=0
```

**Standalone Mode**: Run workers as separate processes.

```bash
# Run standalone worker (1 worker)
./whatomate worker

# Run with multiple workers
./whatomate worker -workers=5

# Or with Docker Compose
docker compose up -d --scale worker=3
```

### Scaling Workers Without Restart

Workers can be added dynamically without restarting the server. Since all workers consume from the same Redis Stream consumer group, new workers immediately start processing queued jobs.

```bash
# Server running with 1 embedded worker
./whatomate server -workers=1

# In another terminal, add 5 more workers
./whatomate worker -workers=5

# Add even more workers if needed
./whatomate worker -workers=10
```

This is useful for:
- **Burst processing**: Scale up workers during high-volume campaigns
- **Zero-downtime scaling**: Add capacity without interrupting the server
- **Resource optimization**: Run workers on different machines

## API Endpoints

### Authentication
- `POST /api/auth/register` - Register new user
- `POST /api/auth/login` - Login
- `POST /api/auth/refresh` - Refresh token
- `POST /api/auth/logout` - Logout

### WhatsApp Accounts
- `GET /api/accounts` - List accounts
- `POST /api/accounts` - Create account
- `PUT /api/accounts/:id` - Update account
- `DELETE /api/accounts/:id` - Delete account

### Users (Admin only)
- `GET /api/users` - List users
- `POST /api/users` - Create user
- `GET /api/users/:id` - Get user
- `PUT /api/users/:id` - Update user
- `DELETE /api/users/:id` - Delete user

### Contacts
- `GET /api/contacts` - List contacts (agents see only assigned)
- `POST /api/contacts` - Create contact
- `PUT /api/contacts/:id/assign` - Assign contact to agent
- `GET /api/contacts/:id/messages` - Get messages
- `POST /api/contacts/:id/messages` - Send message

### Templates
- `GET /api/templates` - List templates
- `POST /api/templates/sync` - Sync from Meta

### Campaigns
- `GET /api/campaigns` - List campaigns
- `POST /api/campaigns` - Create campaign
- `GET /api/campaigns/:id` - Get campaign details
- `PUT /api/campaigns/:id` - Update campaign
- `DELETE /api/campaigns/:id` - Delete campaign
- `POST /api/campaigns/:id/start` - Start campaign (queues for processing)
- `POST /api/campaigns/:id/pause` - Pause campaign
- `POST /api/campaigns/:id/cancel` - Cancel campaign
- `POST /api/campaigns/:id/retry-failed` - Retry failed messages
- `GET /api/campaigns/:id/stats` - Get campaign statistics
- `GET /api/campaigns/:id/recipients` - List recipients
- `POST /api/campaigns/:id/recipients/import` - Import recipients

### WhatsApp Flows
- `GET /api/flows` - List flows
- `POST /api/flows` - Create flow
- `GET /api/flows/:id` - Get flow
- `PUT /api/flows/:id` - Update flow
- `DELETE /api/flows/:id` - Delete flow
- `POST /api/flows/:id/save-to-meta` - Save/update flow on Meta
- `POST /api/flows/:id/publish` - Publish flow on Meta
- `POST /api/flows/:id/deprecate` - Deprecate flow
- `POST /api/flows/sync` - Sync flows from Meta

### Chatbot
- `GET /api/chatbot/settings` - Get settings
- `PUT /api/chatbot/settings` - Update settings
- `GET /api/chatbot/keywords` - List keyword rules
- `GET /api/chatbot/flows` - List flows
- `GET /api/chatbot/ai-contexts` - List AI contexts

### Canned Responses
- `GET /api/canned-responses` - List canned responses
- `POST /api/canned-responses` - Create canned response
- `GET /api/canned-responses/:id` - Get canned response
- `PUT /api/canned-responses/:id` - Update canned response
- `DELETE /api/canned-responses/:id` - Delete canned response
- `POST /api/canned-responses/:id/use` - Track usage

### Webhooks
- `GET /api/webhook` - Webhook verification
- `POST /api/webhook` - Receive messages

## Role-Based Access Control

The platform supports three user roles with different permission levels:

| Feature | Admin | Manager | Agent |
|---------|-------|---------|-------|
| User Management | Full | None | None |
| Account Settings | Full | Full | None |
| Contacts | Full | Full | Assigned only |
| Messages | Full | Full | Assigned only |
| Templates | Full | Full | None |
| Flows | Full | Full | None |
| Campaigns | Full | Full | None |
| Chatbot Settings | Full | Full | None |
| Canned Responses | Full | Full | Use only |
| Analytics | Full | Full | None |

- **Admin**: Full access to all features including user management
- **Manager**: Full access except cannot manage users
- **Agent**: Can only chat with contacts assigned to them

## Conversation Flows

Conversation flows allow you to create multi-step automated conversations that collect information from users.

### Skip Conditions

Each step can have an optional **skip condition** that determines whether to skip the step based on previously collected data. If the condition evaluates to `true`, the step is skipped and the flow proceeds to the next step.

#### Syntax

| Operator | Description | Example |
|----------|-------------|---------|
| `==` | Equals | `status == 'confirmed'` |
| `!=` | Not equals | `phone != ''` (not empty) |
| `>` | Greater than | `amount > 1000` |
| `<` | Less than | `age < 18` |
| `>=` | Greater or equal | `count >= 5` |
| `<=` | Less or equal | `score <= 100` |
| `AND` | All conditions must be true | `name != '' AND phone != ''` |
| `OR` | Any condition can be true | `status == 'vip' OR amount > 1000` |
| `()` | Grouping | `(status == 'vip' OR amount > 100) AND name != ''` |

#### Examples

```
# Skip if phone already collected
phone != ''

# Skip if both name and email are provided
name != '' AND email != ''

# Skip for VIP users or high-value orders
status == 'vip' OR amount > 1000

# Complex condition with grouping
(status == 'vip' OR amount > 100) AND name != ''
```

#### Button Responses

When a user clicks a button, two variables are stored:
- `{store_as}` - The button ID (e.g., `btn_1`)
- `{store_as}_title` - The button text (e.g., `Yes`)

To check the button text in skip conditions, use the `_title` suffix:
```
# Check button text
choice_title == 'Yes'

# Check button ID
choice == 'btn_yes'

# Check if any button was clicked (not empty)
choice != ''
```

## WhatsApp Setup

1. Create a Meta Developer account at [developers.facebook.com](https://developers.facebook.com)
2. Create a new app and add the WhatsApp product
3. Get your Phone Number ID and Business Account ID
4. Generate a permanent access token
5. Configure the webhook URL to point to `/api/webhook`
6. Set the webhook verify token in your configuration

## License

See [LICENSE](LICENSE) for details. Free to use and distribute.
