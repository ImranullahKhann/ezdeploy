// Package middleware owns HTTP cross-cutting concerns.
//
// Available middleware:
//   - RequireAuth: validates session cookies and injects user context
//   - CORS: handles Cross-Origin Resource Sharing for browser clients
package middleware
