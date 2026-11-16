# AI Health Coach

A comprehensive, cloud-native health tracking application built with Go, featuring a calendar-based interface, AI-powered health insights, and modern web technologies.

## 🌟 Features

### Core Functionality
- **Daily Health Journal**: Track food intake, water consumption, sleep patterns, exercise, and alcohol consumption
- **Calendar Interface**: Intuitive calendar view for easy navigation and data entry
- **AI-Powered Insights**: Get personalized health evaluations and recommendations using Ollama with markdown responses
- **Interactive Chat**: Follow-up questions with suggested quick queries and custom chat input
- **Data Visualization**: Interactive charts and dashboards to visualize health trends
- **Historical Analysis**: RAG (Retrieval Augmented Generation) powered historical comparisons

### Technical Features
- **Cloud-Native Architecture**: Deployed on Kubernetes with best practices
- **Simplified Authentication**: Development mode with Azure AD integration ready for production
- **Responsive Design**: Mobile-first web interface that works on all devices
- **Real-time Updates**: Dynamic data loading and updates
- **Secure by Design**: Authentication, authorization, and data encryption
- **Scalable Infrastructure**: Horizontal scaling with load balancing
- **Markdown AI Responses**: Rich formatted AI evaluations with score extraction

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Frontend      │    │   Go Backend     │    │   PostgreSQL    │
│   (HTML/JS/CSS) │◄──►│   (REST API)     │◄──►│   Database      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌──────────────────┐
                       │   Remote Ollama  │
                       │ (192.168.1.66)   │
                       │ llama3.2:latest  │
                       └──────────────────┘
```

### Technology Stack
- **Backend**: Go 1.21+ with Gorilla Mux
- **Database**: PostgreSQL 15 with migrations and custom JSON marshaling
- **AI**: Configurable Ollama or Anthropic provider
- **Authentication**: Simplified development mode (Azure AD ready for production)
- **Deployment**: Kubernetes with Kustomize
- **Containerization**: Podman multi-stage builds
- **Development**: Podman Compose for local development
- **Data Types**: Custom date/time handling for flexible JSON parsing

## 🚀 Quick Start

## 🆕 Recent Improvements

### Enhanced AI Chat Experience
- **Interactive Follow-up Chat**: Continue conversations with the AI health coach
- **Smart Question Pills**: Quick access to common health questions
- **Markdown Responses**: Rich formatted AI responses with health score extraction
- **Real-time Chat Interface**: Seamless conversation flow with loading states

### Improved Data Handling
- **Custom Date/Time Types**: Flexible JSON parsing for dates ("2025-08-03") and times ("23:30")
- **Remote Ollama Integration**: Connected to external Ollama server (192.168.1.66:11434)
- **Simplified Authentication**: Development-friendly auth with production Azure AD ready
- **Optimized Architecture**: Removed unused Redis dependency for cleaner deployment

### Technical Enhancements
- **Better Error Handling**: Improved error messages and debugging
- **Mobile Responsive**: Enhanced mobile experience for chat interface
- **Cache Management**: Smart browser cache handling for JavaScript/CSS updates
- **Robust Data Types**: CustomDate and CustomTime with database compatibility

## 🚀 Quick Start

### Prerequisites
- Go 1.21+
- Podman and a Compose provider such as `podman-compose`
- kubectl (for Kubernetes deployment)
- Git

### Local Development

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd health-coach
   ```

2. **Configure the AI provider**
   ```bash
   cp .env.example .env
   ```

   To use Anthropic while Ollama is unavailable, set these values in `.env`:
   ```dotenv
   AI_PROVIDER=anthropic
   ANTHROPIC_API_KEY=your-api-key
   ANTHROPIC_MODEL=claude-sonnet-5-5
   ```

   Keep `.env` local and never commit the API key.

3. **Start the development environment**
   ```bash
   make up
   ```

4. **Access the application**
   - Main application: http://localhost:8080
   - Nginx reverse proxy: http://localhost:8088
   - AI Chat interface: http://localhost:8080/chat
   - Database admin: http://localhost:8081 (Adminer) - use profile `dev-tools`
   - API health check: http://localhost:8080/api/health

5. **Stop the environment**
   ```bash
   make down
   ```

### Alternative Development Setup

If you prefer to run components separately:

```bash
# Start only the database (no Redis)
make dev

# In another terminal, run the Go application
make build
./bin/health-coach
```

## 📊 Usage

### Creating Journal Entries

1. **Navigate to the Calendar**: The home page shows a calendar interface
2. **Select a Date**: Click on any date to create or edit an entry
3. **Fill in Health Data**:
   - Food intake (free text)
   - Water consumption (liters)
   - Sleep times (HH:MM format) and quality (1-5 rating)
   - Exercise type, duration, and intensity
   - Alcohol consumption (units)
   - Additional notes

4. **Save Entry**: Click "Save Entry" to store your data
5. **Get AI Evaluation**: Use the chat interface for personalized health insights

### AI Health Evaluation

1. **Daily Evaluation**: Get insights about your current day's choices
2. **Historical Analysis**: Compare today's entry with your historical patterns
3. **Interactive Follow-up Chat**: Ask specific questions about your health
4. **Quick Question Pills**: Use suggested questions like:
   - 🍎 Food Focus - "What specific foods should I focus on today?"
   - 😴 Sleep Tips - "How can I improve my sleep quality?"
   - 💪 Exercise Plan - "What exercise would be best for me today?"
   - 💧 Hydration - "How can I increase my water intake?"
   - 🧘 Stress Relief - "What are some stress management techniques?"
   - 📈 Habit Building - "How can I maintain consistent habits?"
5. **Personalized Recommendations**: Receive tailored advice based on your data
6. **Markdown Responses**: Rich formatted AI responses with health scores

### Data Visualization

1. **Dashboard**: View trends and statistics over time
2. **Interactive Charts**: Explore water intake, sleep quality, and exercise patterns
3. **Summary Statistics**: See averages and totals for different time periods

## 🛠️ Development

### Project Structure

```
health-coach/
├── cmd/
│   └── server/              # Application entry point
├── pkg/
│   ├── api/                 # REST API handlers with markdown support
│   ├── auth/                # Authentication middleware (simplified)
│   ├── database/            # Database operations with custom types
│   ├── models/              # Data models with CustomDate/CustomTime
│   └── ollama/              # AI client for remote Ollama server
├── web/
│   ├── assets/              # CSS, JavaScript, images
│   └── templates/           # HTML templates
├── k8s/
│   ├── base/                # Base Kubernetes manifests
│   └── overlays/            # Environment-specific overlays
├── scripts/                 # Deployment and utility scripts
├── migrations/              # Database migrations
├── docker-compose.yaml      # Local development setup
├── Dockerfile              # Container build instructions
└── Makefile                # Development automation
```

### Available Make Commands

```bash
# Development
make build          # Build the Go binary
make test           # Run all tests
make dev            # Start development environment
make fmt            # Format code
make lint           # Run linting

# Docker
make docker-build   # Build Docker image
make up             # Start all services
make down           # Stop all services
make logs           # View service logs

# Kubernetes
make deploy         # Deploy to Kubernetes
make k8s-apply      # Apply manifests

# Utilities
make clean          # Clean build artifacts
make help           # Show all commands
```

### Testing

The project includes comprehensive testing:

```bash
# Run all tests
make test

# Run only unit tests
make test-unit

# Run integration tests
make test-integration

# Generate coverage report
make coverage
```

### Code Quality

```bash
# Format code
make fmt

# Run linting
make lint

# Security checks
make security

# All quality checks
make check
```

## 🚢 Deployment

### Kubernetes Deployment

The Kubernetes manifests deploy the app and connect to an existing PostgreSQL server. They do not create a database in the cluster. Before deploying, ensure the cluster can reach the PostgreSQL host configured as `DB_HOST` in `k8s/base/kustomization.yaml` (currently `postgres.lan`), and that the database/user already exist with permission to run migrations.

#### Development Deployment

Run these commands from the repository root. The current development overlay uses namespace `health-coach-development` and image `docker.io/kringen/health-coach:latest`.

1. **Build and push the image** if the current version is not already in the registry:

   ```sh
   podman build -f Dockerfile -t docker.io/kringen/health-coach:latest .
   podman push docker.io/kringen/health-coach:latest
   ```

2. **Create the namespace and Kubernetes Secrets** from the local, Git-ignored files:

   ```sh
   kubectl create namespace health-coach-development \
     --dry-run=client -o yaml | kubectl apply -f -

    kubectl create secret generic postgres-secret \
     --namespace health-coach-development \
       --from-file=password=k8s/secrets/postgres-password \
     --dry-run=client -o yaml | kubectl apply -f -

   kubectl create secret generic app-secrets \
     --namespace health-coach-development \
     --from-env-file=k8s/secrets/app-secrets.env \
     --dry-run=client -o yaml | kubectl apply -f -
   ```

   `app-secrets.env` must contain `google-client-id`, `google-client-secret`, `session-secret`, and `anthropic-api-key` because the current base configuration selects Anthropic.

3. **Deploy and verify rollout:**

   ```sh
   kubectl apply -k k8s/overlays/development
   kubectl rollout status deployment/health-coach --namespace health-coach-development
   kubectl get pods,services,ingress --namespace health-coach-development
   kubectl logs deployment/health-coach --namespace health-coach-development
   ```

The development overlay currently shares the `tracker.kringen.io` ingress hostname and OAuth callback with the base configuration. If development and production are in the same cluster, set a separate development hostname and matching `GOOGLE_REDIRECT_URL`, then register that callback with Google OAuth before applying.

### Production Configuration

1. **Update Environment Variables**: Configure production settings in `k8s/overlays/production/`
2. **Set Up SSL Certificates**: Configure TLS in the Ingress resource
3. **Configure Authentication**: Set up Azure AD and oauth2-proxy
4. **Database Configuration**: Set up persistent storage and backup

### Monitoring

The application includes health checks and metrics:

- Health endpoint: `/api/health`
- Kubernetes readiness and liveness probes
- Optional Prometheus metrics (with monitoring profile)

## 🔒 Security

### Authentication Flow

1. **User Access**: User attempts to access the application
2. **Development Mode**: Simplified authentication with default test user
3. **Automatic User Creation**: Users created automatically in database
4. **Production Ready**: Azure AD integration available for production deployment

### Security Features

- **HTTPS/TLS**: SSL termination at ingress (production)
- **Authentication**: Simplified for development, Azure AD ready for production
- **Authorization**: User-specific data access
- **Data Validation**: Input sanitization and validation with custom date/time parsing
- **Security Headers**: XSS protection, content type sniffing prevention
- **Rate Limiting**: API rate limiting and DDoS protection (production)

## 🧪 Testing Strategy

### Test Types

1. **Unit Tests**: Test individual functions and components
2. **Integration Tests**: Test database interactions and API endpoints
3. **Security Tests**: Vulnerability scanning and security analysis
4. **Load Tests**: Performance and scalability testing
5. **End-to-End Tests**: Complete user workflow testing

### Test Data

The application includes test fixtures and seed data for development:

```bash
# Seed development database
make db-seed

# Reset database (careful!)
make db-reset
```

## 📊 API Documentation

### Endpoints

#### Health Check
- `GET /api/health` - Application health status

#### Journal Entries
- `GET /api/journal/{date}` - Get journal entry for date
- `POST /api/journal` - Create/update journal entry
- `PUT /api/journal/{id}` - Update specific entry

#### AI Evaluation
- `POST /api/ollama/evaluate` - Request AI health evaluation (supports daily, historical, and followup types)

**Request Body Examples:**
```json
{
  "date": "2025-08-03",
  "evaluation_type": "daily"
}
```

```json
{
  "date": "2025-08-03", 
  "evaluation_type": "historical"
}
```

```json
{
  "date": "2025-08-03",
  "evaluation_type": "followup",
  "question": "What specific foods should I focus on today?"
}
```

**Response**: Markdown formatted health evaluation with embedded health scores

#### Data Analysis
- `GET /api/data/summary` - Get summary statistics
- `GET /api/data/trends` - Get trend data for visualization

### Authentication

All API endpoints use simplified authentication in development mode:

```bash
# Development mode (automatic)
curl -H "Content-Type: application/json" http://localhost:8080/api/health

# Production (when Azure AD is configured)
curl -H "Authorization: Bearer <token>" https://your-domain.com/api/health
```

## 🔧 Configuration

### Environment Variables

Key configuration options (see `.env.example` for complete list):

```bash
# Application
ENVIRONMENT=development
PORT=8080

# Database
DB_HOST=postgres
DB_NAME=healthtracker
DB_USER=healthtracker
DB_PASSWORD=healthtracker123
DB_SSLMODE=disable

# AI Service (Remote Ollama)
OLLAMA_BASE_URL=http://192.168.1.66:11434
OLLAMA_MODEL=llama3.2:latest

# Authentication (Optional - for production)
JWT_SECRET=<secure-random-string>
```

### Database Configuration

The application uses PostgreSQL with automatic migrations:

```sql
-- Example connection
psql postgresql://healthtracker:password@localhost:5432/healthtracker
```

## 🤝 Contributing

1. **Fork the Repository**: Create your own fork
2. **Create Feature Branch**: `git checkout -b feature/amazing-feature`
3. **Run Tests**: `make test`
4. **Commit Changes**: `git commit -m 'Add amazing feature'`
5. **Push Branch**: `git push origin feature/amazing-feature`
6. **Open Pull Request**: Submit for review

### Code Style

- Follow Go formatting: `make fmt`
- Run linting: `make lint`
- Write tests for new features
- Update documentation

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **Ollama**: For providing the AI inference engine
- **PostgreSQL**: For the robust database system
- **Kubernetes**: For the orchestration platform
- **Go Community**: For the excellent ecosystem

## 📞 Support

For support and questions:

- **Issues**: Create an issue on GitHub
- **Documentation**: Check the docs/ directory
- **Email**: support@yourdomain.com

## 🔧 Troubleshooting

### Common Issues

1. **Chat not showing follow-up options**
   - Ensure you've completed an initial health evaluation
   - Check browser console for JavaScript errors
   - Try hard refresh (Ctrl+F5) to clear cache

2. **Date parsing errors**
   - Use YYYY-MM-DD format for dates (e.g., "2025-08-03")
   - Sleep times should be in HH:MM format (e.g., "23:30")

3. **Ollama connection issues**
   - Verify remote Ollama server is running at 192.168.1.66:11434
   - Check network connectivity
   - Confirm llama3.2:latest model is available

4. **Docker container issues**
   - Use `make down && make up` for a clean restart
   - Remove orphaned containers with `--remove-orphans` flag

### Debug Commands

```bash
# Check service status
make status

# View application logs
make logs-app

# Test API health
curl http://localhost:8080/api/health

# Test Ollama connectivity (from container)
curl http://192.168.1.66:11434/api/tags
```

---

**Built with ❤️ using Go and modern cloud-native technologies**
