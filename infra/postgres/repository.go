package postgres

import (
	"auction/domain"
	"context"
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type PgRepository struct {
	db *sql.DB
}

func NewPgRepository(host string, database string, user string, password string, port string) *PgRepository {
	db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, database))
	if err != nil {
		panic(err)
	}
	return &PgRepository{
		db: db,
	}
}

func (r *PgRepository) FindByID(id string) (*domain.User, error) {
	var user domain.User
	err := r.db.QueryRow("SELECT id, email, name, password, two_factor_enabled, two_factor_verified, COALESCE(two_factor_secret, ''), COALESCE(two_factor_recovery_codes, '') FROM users WHERE id = $1", id).Scan(&user.ID, &user.Email, &user.Name, &user.Password, &user.TwoFactorEnabled, &user.TwoFactorVerified, &user.TwoFactorSecret, &user.TwoFactorRecoveryCodes)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *PgRepository) FindByEmail(email string) (*domain.User, error) {
	var user domain.User
	err := r.db.QueryRow("SELECT id, email, name, password, two_factor_enabled, COALESCE(two_factor_secret, ''), two_factor_verified, COALESCE(two_factor_recovery_codes, '') password FROM users WHERE email = $1", email).Scan(&user.ID, &user.Email, &user.Name, &user.Password,  &user.TwoFactorEnabled, &user.TwoFactorSecret, &user.TwoFactorVerified, &user.TwoFactorRecoveryCodes)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *PgRepository) Close() error {
	return r.db.Close()
}

func (r *PgRepository) Create(ctx context.Context, email string, password string, name string) (string, error) {
	var user domain.User
	err := r.db.QueryRowContext(ctx, "INSERT INTO users (email, password, name) VALUES ($1, $2, $3) RETURNING id", email, password, name).Scan(&user.ID)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

func (r *PgRepository) Update(ctx context.Context, id string, email string, name string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE users SET email = $1, name = $2 WHERE id = $3", email, name, id)
	if err != nil {
		return err
	}
	return nil
}

func (r *PgRepository) EnableTwoFactor(ctx context.Context, id string, twoFactorSecret string) error {
	var user domain.User

	err := r.db.QueryRowContext(
		ctx,
		`UPDATE users
         SET two_factor_enabled = TRUE,
             two_factor_secret  = $1
         WHERE id = $2
         RETURNING id, name, email, password, two_factor_enabled, COALESCE(two_factor_secret, ''), COALESCE(two_factor_recovery_codes, '')`,
		twoFactorSecret,
		id,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.TwoFactorEnabled,
		&user.TwoFactorSecret,
		&user.TwoFactorRecoveryCodes,
	)

	return err
}

func (r *PgRepository) DisableTwoFactor(ctx context.Context, id string) error {
	var user domain.User

	err := r.db.QueryRowContext(
		ctx,
		`UPDATE users
         SET two_factor_enabled = FALSE,
			two_factor_secret = NULL,
			two_factor_verified = FALSE
         WHERE id = $1
         RETURNING id, name, email, password, two_factor_enabled, COALESCE(two_factor_secret, ''), two_factor_verified, COALESCE(two_factor_recovery_codes, '')`,
		id,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.TwoFactorEnabled,
		&user.TwoFactorSecret,
		&user.TwoFactorVerified,
		&user.TwoFactorRecoveryCodes,
	)

	return err
}

func (r *PgRepository) MarkTwoFactorVerified(ctx context.Context, id string) error {
	var user domain.User

	err := r.db.QueryRowContext(
		ctx,
		`UPDATE users
         SET two_factor_verified = true
         WHERE id = $1
         RETURNING id, name, email, password, two_factor_enabled, COALESCE(two_factor_secret, ''), two_factor_verified, COALESCE(two_factor_recovery_codes, '')`,
		id,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.TwoFactorEnabled,
		&user.TwoFactorSecret,
		&user.TwoFactorVerified,
		&user.TwoFactorRecoveryCodes,
	)

	return err
}

func (r *PgRepository) SetRecoveryCodes(ctx context.Context, id string, recoveryCodes string) error {
	var user domain.User

	err := r.db.QueryRowContext(
		ctx,
		`UPDATE users
         SET two_factor_recovery_codes = $1
         WHERE id = $2
         RETURNING id, name, email, password, two_factor_enabled, COALESCE(two_factor_secret, ''), two_factor_verified, COALESCE(two_factor_recovery_codes, '')`,
		 recoveryCodes,
		 id,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.TwoFactorEnabled,
		&user.TwoFactorSecret,
		&user.TwoFactorVerified,
		&user.TwoFactorRecoveryCodes,
	)

	return err
}