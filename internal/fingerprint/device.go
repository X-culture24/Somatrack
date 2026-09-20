package fingerprint

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Device struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Location    string    `json:"location"`
	DeviceType  string    `json:"device_type"`
	SecretHash  string    `json:"-"`
	LastSeenAt  *time.Time `json:"last_seen_at"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

type DeviceStore struct {
	pool *pgxpool.Pool
}

func NewDeviceStore(pool *pgxpool.Pool) *DeviceStore {
	return &DeviceStore{pool: pool}
}

func (s *DeviceStore) Register(ctx context.Context, name, location, deviceType, sharedSecret string) (*Device, error) {
	id := uuid.New()
	secretHash := HashDeviceSecret(sharedSecret)
	now := time.Now()
	const q = `
		INSERT INTO fingerprint_devices (id, name, location, device_type, secret_hash, is_active, created_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, true, $6, $6)
		RETURNING id, name, location, device_type, secret_hash, is_active, created_at, last_seen_at`
	row := s.pool.QueryRow(ctx, q, id, name, location, deviceType, secretHash, now)
	d := &Device{}
	var ls *time.Time
	if err := row.Scan(&d.ID, &d.Name, &d.Location, &d.DeviceType, &d.SecretHash, &d.IsActive, &d.CreatedAt, &ls); err != nil {
		return nil, fmt.Errorf("insert device: %w", err)
	}
	d.LastSeenAt = ls
	return d, nil
}

func (s *DeviceStore) GetByID(ctx context.Context, id uuid.UUID) (*Device, error) {
	const q = `
		SELECT id, name, location, device_type, secret_hash, is_active, created_at, last_seen_at
		FROM fingerprint_devices WHERE id = $1`
	row := s.pool.QueryRow(ctx, q, id)
	d := &Device{}
	var ls *time.Time
	if err := row.Scan(&d.ID, &d.Name, &d.Location, &d.DeviceType, &d.SecretHash, &d.IsActive, &d.CreatedAt, &ls); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDeviceNotFound
		}
		return nil, err
	}
	d.LastSeenAt = ls
	return d, nil
}

func (s *DeviceStore) Heartbeat(ctx context.Context, id uuid.UUID, ts time.Time) error {
	res, err := s.pool.Exec(ctx, `
		UPDATE fingerprint_devices SET last_seen_at = $2, is_active = true
		WHERE id = $1`, id, ts)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrDeviceNotFound
	}
	return nil
}

func (s *DeviceStore) SetInactive(ctx context.Context, id uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE fingerprint_devices SET is_active = false WHERE id = $1`, id)
	return err
}

var ErrDeviceNotFound = errors.New("device not found")
var ErrTemplateNotFound = errors.New("fingerprint template not found")
