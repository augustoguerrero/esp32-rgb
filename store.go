package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Animation represents a named LED animation stored in the database.
type Animation struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	FPS       int       `json:"fps"`
	Loop      bool      `json:"loop"`
	CreatedAt time.Time `json:"created_at"`
	Frames    []Frame   `json:"frames,omitempty"`
}

// Frame is one snapshot of all LEDs for a given duration.
type Frame struct {
	ID          int64  `json:"id"`
	AnimationID int64  `json:"animation_id"`
	Order       int    `json:"order"`
	DurationMs  int    `json:"duration_ms"`
	LEDs        []RGB  `json:"leds"` // len == cfg.NumLEDs
}

// RGB is a single LED colour.
type RGB struct {
	R, G, B uint8
}

// Store wraps the SQLite connection.
type Store struct {
	db      *sql.DB
	numLEDs int
}

// InitDB opens (or creates) the SQLite database and runs migrations.
func InitDB(path string, numLEDs int) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	s := &Store{db: db, numLEDs: numLEDs}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA foreign_keys=ON;

		CREATE TABLE IF NOT EXISTS animations (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT    NOT NULL,
			fps         INTEGER NOT NULL DEFAULT 30,
			loop        INTEGER NOT NULL DEFAULT 1,
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS animation_frames (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			animation_id INTEGER NOT NULL REFERENCES animations(id) ON DELETE CASCADE,
			frame_order  INTEGER NOT NULL,
			duration_ms  INTEGER NOT NULL DEFAULT 100,
			leds         BLOB    NOT NULL
		);
	`)
	return err
}

// Close closes the database connection.
func (s *Store) Close() error { return s.db.Close() }

// ListAnimations returns all animations (without frames).
func (s *Store) ListAnimations() ([]Animation, error) {
	rows, err := s.db.Query(
		`SELECT id, name, fps, loop, created_at FROM animations ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Animation
	for rows.Next() {
		var a Animation
		var loopInt int
		if err := rows.Scan(&a.ID, &a.Name, &a.FPS, &loopInt, &a.CreatedAt); err != nil {
			return nil, err
		}
		a.Loop = loopInt != 0
		list = append(list, a)
	}
	return list, rows.Err()
}

// GetAnimation returns a single animation with its frames loaded.
func (s *Store) GetAnimation(id int64) (*Animation, error) {
	var a Animation
	var loopInt int
	err := s.db.QueryRow(
		`SELECT id, name, fps, loop, created_at FROM animations WHERE id = ?`, id,
	).Scan(&a.ID, &a.Name, &a.FPS, &loopInt, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	a.Loop = loopInt != 0

	frames, err := s.GetFrames(id)
	if err != nil {
		return nil, err
	}
	a.Frames = frames
	return &a, nil
}

// CreateAnimation inserts a new animation and returns its ID.
func (s *Store) CreateAnimation(name string, fps int, loop bool) (int64, error) {
	loopInt := 0
	if loop {
		loopInt = 1
	}
	res, err := s.db.Exec(
		`INSERT INTO animations (name, fps, loop, created_at) VALUES (?, ?, ?, ?)`,
		name, fps, loopInt, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateAnimation changes the name/fps/loop of an existing animation.
func (s *Store) UpdateAnimation(id int64, name string, fps int, loop bool) error {
	loopInt := 0
	if loop {
		loopInt = 1
	}
	_, err := s.db.Exec(
		`UPDATE animations SET name=?, fps=?, loop=? WHERE id=?`,
		name, fps, loopInt, id,
	)
	return err
}

// DeleteAnimation removes an animation and all its frames (cascade).
func (s *Store) DeleteAnimation(id int64) error {
	_, err := s.db.Exec(`DELETE FROM animations WHERE id=?`, id)
	return err
}

// GetFrames returns all frames for an animation in order.
func (s *Store) GetFrames(animID int64) ([]Frame, error) {
	rows, err := s.db.Query(
		`SELECT id, animation_id, frame_order, duration_ms, leds
		 FROM animation_frames WHERE animation_id=? ORDER BY frame_order`, animID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var frames []Frame
	for rows.Next() {
		var f Frame
		var blob []byte
		if err := rows.Scan(&f.ID, &f.AnimationID, &f.Order, &f.DurationMs, &blob); err != nil {
			return nil, err
		}
		f.LEDs = blobToLEDs(blob, s.numLEDs)
		frames = append(frames, f)
	}
	return frames, rows.Err()
}

// SetFrames replaces all frames for an animation atomically.
func (s *Store) SetFrames(animID int64, frames []Frame) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM animation_frames WHERE animation_id=?`, animID); err != nil {
		return err
	}
	for i, f := range frames {
		blob := ledsToBlob(f.LEDs, s.numLEDs)
		if _, err := tx.Exec(
			`INSERT INTO animation_frames (animation_id, frame_order, duration_ms, leds) VALUES (?,?,?,?)`,
			animID, i, f.DurationMs, blob,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ledsToBlob encodes an []RGB into a raw byte slice (R,G,B per LED).
func ledsToBlob(leds []RGB, numLEDs int) []byte {
	b := make([]byte, numLEDs*3)
	for i := 0; i < numLEDs && i < len(leds); i++ {
		b[i*3] = leds[i].R
		b[i*3+1] = leds[i].G
		b[i*3+2] = leds[i].B
	}
	return b
}

// blobToLEDs decodes a raw byte slice into []RGB.
func blobToLEDs(b []byte, numLEDs int) []RGB {
	leds := make([]RGB, numLEDs)
	for i := 0; i < numLEDs && i*3+2 < len(b); i++ {
		leds[i] = RGB{b[i*3], b[i*3+1], b[i*3+2]}
	}
	return leds
}
