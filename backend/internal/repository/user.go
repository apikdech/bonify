package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/receipt-manager/backend/internal/model"
)

// UserRepo provides database operations for users
type UserRepo struct {
	db *pgxpool.Pool
}

// NewUserRepo creates a new user repository
func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

// Create inserts a new user and returns the created user with ID and timestamps
func (r *UserRepo) Create(ctx context.Context, user *model.User) (*model.User, error) {
	query := `
		INSERT INTO users (name, email, password_hash, telegram_id, discord_id, role, llm_provider, llm_model, home_currency)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`

	var id uuid.UUID
	var createdAt, updatedAt time.Time

	err := r.db.QueryRow(ctx, query,
		user.Name,
		user.Email,
		user.PasswordHash,
		user.TelegramID,
		user.DiscordID,
		user.Role,
		user.LLMProvider,
		user.LLMModel,
		user.HomeCurrency,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	user.ID = id
	user.CreatedAt = createdAt
	user.UpdatedAt = updatedAt

	return user, nil
}

// GetByEmail retrieves a user by email, including password hash
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, name, email, password_hash, telegram_id, discord_id, role, llm_provider, llm_model, home_currency, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &model.User{}
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.TelegramID,
		&user.DiscordID,
		&user.Role,
		&user.LLMProvider,
		&user.LLMModel,
		&user.HomeCurrency,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetByID retrieves a user by ID
func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, name, email, password_hash, telegram_id, discord_id, role, llm_provider, llm_model, home_currency, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &model.User{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.TelegramID,
		&user.DiscordID,
		&user.Role,
		&user.LLMProvider,
		&user.LLMModel,
		&user.HomeCurrency,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return user, nil
}

// GetByTelegramID retrieves a user by Telegram ID
func (r *UserRepo) GetByTelegramID(ctx context.Context, telegramID string) (*model.User, error) {
	query := `
		SELECT id, name, email, password_hash, telegram_id, discord_id, role, llm_provider, llm_model, home_currency, created_at, updated_at
		FROM users
		WHERE telegram_id = $1
	`

	user := &model.User{}
	err := r.db.QueryRow(ctx, query, telegramID).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.TelegramID,
		&user.DiscordID,
		&user.Role,
		&user.LLMProvider,
		&user.LLMModel,
		&user.HomeCurrency,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by telegram ID: %w", err)
	}

	return user, nil
}

// GetByDiscordID retrieves a user by Discord ID
func (r *UserRepo) GetByDiscordID(ctx context.Context, discordID string) (*model.User, error) {
	query := `
		SELECT id, name, email, password_hash, telegram_id, discord_id, role, llm_provider, llm_model, home_currency, created_at, updated_at
		FROM users
		WHERE discord_id = $1
	`

	user := &model.User{}
	err := r.db.QueryRow(ctx, query, discordID).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.TelegramID,
		&user.DiscordID,
		&user.Role,
		&user.LLMProvider,
		&user.LLMModel,
		&user.HomeCurrency,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by discord ID: %w", err)
	}

	return user, nil
}

// UpdatePassword updates a user's password hash
func (r *UserRepo) UpdatePassword(ctx context.Context, userID uuid.UUID, newHash string) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = now()
		WHERE id = $2
	`

	_, err := r.db.Exec(ctx, query, newHash, userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// UpdateBotIDs updates a user's Telegram and Discord IDs
func (r *UserRepo) UpdateBotIDs(ctx context.Context, userID uuid.UUID, telegramID, discordID *string) error {
	query := `
		UPDATE users
		SET telegram_id = $1, discord_id = $2, updated_at = now()
		WHERE id = $3
	`

	_, err := r.db.Exec(ctx, query, telegramID, discordID, userID)
	if err != nil {
		return fmt.Errorf("failed to update bot IDs: %w", err)
	}

	return nil
}

// List retrieves all users (for admin use)
func (r *UserRepo) List(ctx context.Context) ([]*model.User, error) {
	query := `
		SELECT id, name, email, password_hash, telegram_id, discord_id, role, llm_provider, llm_model, home_currency, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		user := &model.User{}
		err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.PasswordHash,
			&user.TelegramID,
			&user.DiscordID,
			&user.Role,
			&user.LLMProvider,
			&user.LLMModel,
			&user.HomeCurrency,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}
