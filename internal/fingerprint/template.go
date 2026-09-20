package fingerprint

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FingerprintTemplate struct {
	StudentID       uuid.UUID  `json:"student_id"`
	TemplateData    []byte     `json:"-"`
	TemplateVersion int        `json:"template_version"`
	EnrolledAt      time.Time  `json:"enrolled_at"`
	EnrolledBy      uuid.UUID  `json:"enrolled_by"`
	DeviceID        uuid.UUID  `json:"device_id"`
	IsActive        bool       `json:"is_active"`
}

type TemplateStore struct {
	pool   *pgxpool.Pool
	crypto *Crypto
}

func NewTemplateStore(pool *pgxpool.Pool, crypto *Crypto) *TemplateStore {
	return &TemplateStore{pool: pool, crypto: crypto}
}

func (s *TemplateStore) Store(ctx context.Context, studentID uuid.UUID, templateB64 string, enrolledBy, deviceID uuid.UUID) error {
	raw, err := base64.StdEncoding.DecodeString(templateB64)
	if err != nil {
		return fmt.Errorf("decode base64 template: %w", err)
	}
	if len(raw) == 0 {
		return errors.New("empty template")
	}
	encrypted, err := s.crypto.EncryptTemplate(raw)
	if err != nil {
		return fmt.Errorf("encrypt template: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE fingerprint_templates SET is_active = false WHERE student_id = $1 AND is_active = true`, studentID); err != nil {
		return err
	}
	var version int
	err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(template_version), 0) + 1 FROM fingerprint_templates WHERE student_id = $1`, studentID).Scan(&version)
	if err != nil {
		return err
	}
	now := time.Now()
	const ins = `
		INSERT INTO fingerprint_templates (student_id, template_data, template_version, enrolled_at, enrolled_by, device_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, true)`
	if _, err := tx.Exec(ctx, ins, studentID, encrypted, version, now, enrolledBy, deviceID); err != nil {
		return fmt.Errorf("insert template: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *TemplateStore) GetActive(ctx context.Context, studentID uuid.UUID) (*FingerprintTemplate, error) {
	const q = `
		SELECT student_id, template_data, template_version, enrolled_at, enrolled_by, device_id, is_active
		FROM fingerprint_templates
		WHERE student_id = $1 AND is_active = true
		ORDER BY template_version DESC LIMIT 1`
	row := s.pool.QueryRow(ctx, q, studentID)
	t := &FingerprintTemplate{}
	if err := row.Scan(&t.StudentID, &t.TemplateData, &t.TemplateVersion, &t.EnrolledAt, &t.EnrolledBy, &t.DeviceID, &t.IsActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}
	return t, nil
}

func (s *TemplateStore) Export(ctx context.Context, studentID uuid.UUID) (templateB64 string, version int, enrolledAt time.Time, deviceID uuid.UUID, isActive bool, err error) {
	t, err := s.GetActive(ctx, studentID)
	if err != nil {
		return "", 0, time.Time{}, uuid.Nil, false, err
	}
	plain, err := s.crypto.DecryptTemplate(t.TemplateData)
	if err != nil {
		return "", 0, time.Time{}, uuid.Nil, false, fmt.Errorf("decrypt template: %w", err)
	}
	return base64.StdEncoding.EncodeToString(plain), t.TemplateVersion, t.EnrolledAt, t.DeviceID, t.IsActive, nil
}

func (s *TemplateStore) Delete(ctx context.Context, studentID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE fingerprint_templates SET is_active = false WHERE student_id = $1 AND is_active = true`, studentID)
	return err
}

func (s *TemplateStore) StoreStaff(ctx context.Context, staffID uuid.UUID, templateB64 string, enrolledBy, deviceID uuid.UUID) error {
	raw, err := base64.StdEncoding.DecodeString(templateB64)
	if err != nil {
		return fmt.Errorf("decode base64 template: %w", err)
	}
	if len(raw) == 0 {
		return errors.New("empty template")
	}
	encrypted, err := s.crypto.EncryptTemplate(raw)
	if err != nil {
		return fmt.Errorf("encrypt template: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE fingerprint_templates SET is_active = false WHERE staff_id = $1 AND is_active = true`, staffID); err != nil {
		return err
	}
	var version int
	err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(template_version), 0) + 1 FROM fingerprint_templates WHERE staff_id = $1`, staffID).Scan(&version)
	if err != nil {
		return err
	}
	now := time.Now()
	const ins = `
		INSERT INTO fingerprint_templates (staff_id, template_data, template_version, enrolled_at, enrolled_by, device_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, true)`
	if _, err := tx.Exec(ctx, ins, staffID, encrypted, version, now, enrolledBy, deviceID); err != nil {
		return fmt.Errorf("insert staff template: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *TemplateStore) GetActiveStaff(ctx context.Context, staffID uuid.UUID) (*FingerprintTemplate, error) {
	const q = `
		SELECT staff_id, template_data, template_version, enrolled_at, enrolled_by, device_id, is_active
		FROM fingerprint_templates
		WHERE staff_id = $1 AND is_active = true
		ORDER BY template_version DESC LIMIT 1`
	row := s.pool.QueryRow(ctx, q, staffID)
	t := &FingerprintTemplate{}
	var sid uuid.UUID
	if err := row.Scan(&sid, &t.TemplateData, &t.TemplateVersion, &t.EnrolledAt, &t.EnrolledBy, &t.DeviceID, &t.IsActive); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}
	t.StudentID = sid
	return t, nil
}

func (s *TemplateStore) ExportStaff(ctx context.Context, staffID uuid.UUID) (templateB64 string, version int, enrolledAt time.Time, deviceID uuid.UUID, isActive bool, err error) {
	t, err := s.GetActiveStaff(ctx, staffID)
	if err != nil {
		return "", 0, time.Time{}, uuid.Nil, false, err
	}
	plain, err := s.crypto.DecryptTemplate(t.TemplateData)
	if err != nil {
		return "", 0, time.Time{}, uuid.Nil, false, fmt.Errorf("decrypt template: %w", err)
	}
	return base64.StdEncoding.EncodeToString(plain), t.TemplateVersion, t.EnrolledAt, t.DeviceID, t.IsActive, nil
}

func (s *TemplateStore) DeleteStaff(ctx context.Context, staffID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE fingerprint_templates SET is_active = false WHERE staff_id = $1 AND is_active = true`, staffID)
	return err
}

func (s *TemplateStore) GetTemplateBlob(ctx context.Context, templateID uuid.UUID) ([]byte, error) {
	var ct []byte
	err := s.pool.QueryRow(ctx, `SELECT template_data FROM fingerprint_templates WHERE id = $1`, templateID).Scan(&ct)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTemplateNotFound
		}
		return nil, err
	}
	return s.crypto.DecryptTemplate(ct)
}
