# Config

Backend configuration loading lives here.

## Environment Variables

- `BACKEND_PORT` - Port for the HTTP server (default: 8080)
- `DATABASE_URL` - PostgreSQL connection string (required)
- `LOG_LEVEL` - Logging level: debug, info, warn, error (default: info)
- `APP_ENV` - Application environment: development, production (default: development)
- `SESSION_SECRET` - Secret key for session cookies (required)
- `CORS_ORIGINS` - Comma-separated list of allowed CORS origins (default: http://localhost:5173)
- `SHUTDOWN_TIMEOUT_SECONDS` - Graceful shutdown timeout in seconds (default: 10)

## CORS Configuration

The `CORS_ORIGINS` variable accepts a comma-separated list of allowed origins:

```bash
# Single origin
CORS_ORIGINS=http://localhost:5173

# Multiple origins
CORS_ORIGINS=http://localhost:5173,http://localhost:3000,https://app.example.com

# Allow all origins (not recommended for production)
CORS_ORIGINS=*
```

The CORS middleware automatically handles:
- `Access-Control-Allow-Origin` header with the requesting origin
- `Access-Control-Allow-Credentials: true` for cookie support
- `Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, Authorization`
- Preflight OPTIONS requests

