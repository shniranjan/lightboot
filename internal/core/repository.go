package core

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shniranjan/lightboot/internal/model"
)

// ISORepository provides CRUD operations for ISO records.
type ISORepository struct {
	db *sql.DB
}

// NewISORepository creates a new ISORepository.
func NewISORepository(db *sql.DB) *ISORepository {
	return &ISORepository{db: db}
}

// InsertISO inserts a new ISO record and sets its ID and timestamps.
func (r *ISORepository) InsertISO(iso *model.ISO) error {
	now := time.Now().UTC()
	iso.CreatedAt = now
	iso.UpdatedAt = now
	iso.LastScanned = now

	bootModesJSON, _ := json.Marshal(iso.BootModes)

	result, err := r.db.Exec(`
		INSERT INTO isos (name, source_path, size, sha256, architecture, boot_modes,
			distro, version, boot_profile, cached_path, status, last_scanned, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		iso.Name,
		iso.SourcePath,
		iso.Size,
		iso.SHA256,
		iso.Arch,
		string(bootModesJSON),
		iso.Distro,
		iso.Version,
		iso.BootProfile,
		iso.CachedPath,
		string(iso.Status),
		iso.LastScanned,
		iso.CreatedAt,
		iso.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert ISO: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get last insert ID: %w", err)
	}
	iso.ID = id

	return nil
}

// GetISOBySHA256 returns the ISO with the given SHA256 hash, or nil if not found.
func (r *ISORepository) GetISOBySHA256(sha256 string) (*model.ISO, error) {
	row := r.db.QueryRow(`
		SELECT id, name, source_path, size, sha256, architecture, boot_modes,
			distro, version, boot_profile, cached_path, status, last_scanned,
			created_at, updated_at
		FROM isos WHERE sha256 = ?
	`, sha256)

	iso, err := scanISO(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return iso, err
}

// GetISOBySourcePath returns the ISO with the given source path, or nil if not found.
func (r *ISORepository) GetISOBySourcePath(path string) (*model.ISO, error) {
	row := r.db.QueryRow(`
		SELECT id, name, source_path, size, sha256, architecture, boot_modes,
			distro, version, boot_profile, cached_path, status, last_scanned,
			created_at, updated_at
		FROM isos WHERE source_path = ?
	`, path)

	iso, err := scanISO(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return iso, err
}

// GetAllISOs returns all ISO records ordered by name.
func (r *ISORepository) GetAllISOs() ([]model.ISO, error) {
	rows, err := r.db.Query(`
		SELECT id, name, source_path, size, sha256, architecture, boot_modes,
			distro, version, boot_profile, cached_path, status, last_scanned,
			created_at, updated_at
		FROM isos ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("query all ISOs: %w", err)
	}
	defer rows.Close()

	var isos []model.ISO
	for rows.Next() {
		iso, err := scanISORows(rows)
		if err != nil {
			return nil, err
		}
		isos = append(isos, *iso)
	}

	return isos, rows.Err()
}

// UpdateISO updates an existing ISO record.
func (r *ISORepository) UpdateISO(iso *model.ISO) error {
	iso.UpdatedAt = time.Now().UTC()

	bootModesJSON, _ := json.Marshal(iso.BootModes)

	_, err := r.db.Exec(`
		UPDATE isos SET
			name = ?, source_path = ?, size = ?, sha256 = ?, architecture = ?,
			boot_modes = ?, distro = ?, version = ?, boot_profile = ?,
			cached_path = ?, status = ?, last_scanned = ?, updated_at = ?
		WHERE id = ?
	`,
		iso.Name,
		iso.SourcePath,
		iso.Size,
		iso.SHA256,
		iso.Arch,
		string(bootModesJSON),
		iso.Distro,
		iso.Version,
		iso.BootProfile,
		iso.CachedPath,
		string(iso.Status),
		iso.LastScanned,
		iso.UpdatedAt,
		iso.ID,
	)
	if err != nil {
		return fmt.Errorf("update ISO: %w", err)
	}

	return nil
}

// DeleteISO removes an ISO record by ID.
func (r *ISORepository) DeleteISO(id int64) error {
	_, err := r.db.Exec(`DELETE FROM isos WHERE id = ?`, id)
	return err
}

// GetReadyISOs returns all ISOs with status "ready".
func (r *ISORepository) GetReadyISOs() ([]model.ISO, error) {
	rows, err := r.db.Query(`
		SELECT id, name, source_path, size, sha256, architecture, boot_modes,
			distro, version, boot_profile, cached_path, status, last_scanned,
			created_at, updated_at
		FROM isos WHERE status = 'ready' ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("query ready ISOs: %w", err)
	}
	defer rows.Close()

	var isos []model.ISO
	for rows.Next() {
		iso, err := scanISORows(rows)
		if err != nil {
			return nil, err
		}
		isos = append(isos, *iso)
	}

	return isos, rows.Err()
}

// GetISOByID returns a single ISO by its primary key.
func (r *ISORepository) GetISOByID(id int64) (*model.ISO, error) {
	row := r.db.QueryRow(`
		SELECT id, name, source_path, size, sha256, architecture, boot_modes,
			distro, version, boot_profile, cached_path, status, last_scanned,
			created_at, updated_at
		FROM isos WHERE id = ?
	`, id)

	iso, err := scanISO(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return iso, err
}

// scanISO scans a single row into an ISO struct.
func scanISO(row *sql.Row) (*model.ISO, error) {
	var iso model.ISO
	var bootModesJSON string

	err := row.Scan(
		&iso.ID,
		&iso.Name,
		&iso.SourcePath,
		&iso.Size,
		&iso.SHA256,
		&iso.Arch,
		&bootModesJSON,
		&iso.Distro,
		&iso.Version,
		&iso.BootProfile,
		&iso.CachedPath,
		&iso.Status,
		&iso.LastScanned,
		&iso.CreatedAt,
		&iso.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if bootModesJSON != "" {
		var modes []model.BootMode
		json.Unmarshal([]byte(bootModesJSON), &modes)
		iso.BootModes = modes
	}

	return &iso, nil
}

// scanISORows scans a row from a Rows result.
func scanISORows(rows *sql.Rows) (*model.ISO, error) {
	var iso model.ISO
	var bootModesJSON string

	err := rows.Scan(
		&iso.ID,
		&iso.Name,
		&iso.SourcePath,
		&iso.Size,
		&iso.SHA256,
		&iso.Arch,
		&bootModesJSON,
		&iso.Distro,
		&iso.Version,
		&iso.BootProfile,
		&iso.CachedPath,
		&iso.Status,
		&iso.LastScanned,
		&iso.CreatedAt,
		&iso.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if bootModesJSON != "" {
		var modes []model.BootMode
		json.Unmarshal([]byte(bootModesJSON), &modes)
		iso.BootModes = modes
	}

	return &iso, nil
}
