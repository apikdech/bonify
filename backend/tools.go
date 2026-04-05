//go:build tools

package tools

// This file ensures all required dependencies are tracked in go.mod
// even if not yet imported in the main codebase.

import (
	_ "github.com/go-chi/chi/v5"
	_ "github.com/go-chi/cors"
	_ "github.com/golang-jwt/jwt/v5"
	_ "github.com/jackc/pgx/v5"
	_ "github.com/joho/godotenv"
	_ "github.com/minio/minio-go/v7"
	_ "github.com/mozilla-ai/any-llm-go"
	_ "github.com/redis/go-redis/v9"
	_ "go.temporal.io/sdk/client"
	_ "go.temporal.io/sdk/worker"
	_ "go.temporal.io/sdk/workflow"
	_ "golang.org/x/crypto/bcrypt"
)
