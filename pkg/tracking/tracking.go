// Package tracking provides privacy-focused user tracking functions.
package tracking

import (
	"context"
	"crypto/rand"
	"net/http"
	"strings"

	"github.com/GGP1/adak/internal/logger"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// inspired by github.com/emvi/pirsch

// Tracker is the interface that wraps user tracking methods.
type Tracker interface {
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context) ([]Hit, error)
	Hit(ctx context.Context, r *http.Request) error
	Searcher
}

// Searcher is the interface that wraps hits search methods.
type Searcher interface {
	Search(ctx context.Context, value string) ([]Hit, error)
	SearchByField(ctx context.Context, field, value string) ([]Hit, error)
}

// Hitter provides methods to hit requests and store them in a data store.
type Hitter struct {
	DB   *sqlx.DB
	salt string
}

// NewService returns a new user tracker.
func NewService(db *sqlx.DB) Tracker {
	salt := make([]byte, 8)
	if _, err := rand.Read(salt); err != nil {
		logger.Fatalf("failed reading salt: %v", err)
	}

	return &Hitter{
		DB:   db,
		salt: string(salt),
	}
}

// Delete takes away the hit with the id specified from the database.
func (h *Hitter) Delete(ctx context.Context, id string) error {
	_, err := h.DB.ExecContext(ctx, "DELETE FROM hits WHERE id=$1", id)
	if err != nil {
		return errors.Wrap(err, "couldn't delete the hit")
	}

	return nil
}

// Get lists all the hits stored in the database.
func (h *Hitter) Get(ctx context.Context) ([]Hit, error) {
	var hits []Hit

	if err := h.DB.SelectContext(ctx, &hits, "SELECT * FROM hits"); err != nil {
		logger.Debugf("failed listing hits: %v", err)
		return nil, errors.Wrap(err, "couldn't find the hits")
	}

	return hits, nil
}

// Hit stores the given request.
// The request might be ignored if it meets certain conditions.
func (h *Hitter) Hit(ctx context.Context, r *http.Request) error {
	q := `INSERT INTO hits
	(id, footprint, path, url, language, user_agent, referer, date)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	if !ignoreHit(r) {
		hit, err := HitRequest(r, h.salt)
		if err != nil {
			return err
		}

		_, err = h.DB.ExecContext(ctx, q, hit.ID, hit.Footprint, hit.Path, hit.URL,
			hit.Language, hit.UserAgent, hit.Referer, hit.Date)
		if err != nil {
			logger.Debugf("failed creating hit: %v", err)
			return errors.Wrap(err, "couldn't save the hit")
		}
	}

	return nil
}

// Search looks for a value and returns a slice of the hits that contain that value.
func (h *Hitter) Search(ctx context.Context, query string) ([]Hit, error) {
	if strings.ContainsAny(query, ";-\\|@#~€¬<>_()[]}{¡^'") {
		return nil, errors.New("invalid search")
	}

	hits := []Hit{}
	q := `SELECT * FROM hits WHERE to_tsvector(
	id || ' ' || footprint || ' ' || 
	path || ' ' || url || ' ' || 
	language || ' ' || referer || ' ' || 
	user_agent || ' ' || date
	) @@ plainto_tsquery($1)`

	if err := h.DB.SelectContext(ctx, &hits, q, query); err != nil {
		return nil, errors.New("no hits found")
	}

	return hits, nil
}

// SearchByField looks for a value and returns a slice of the hits that contain that value.
func (h *Hitter) SearchByField(ctx context.Context, field, value string) ([]Hit, error) {
	var hits []Hit

	q := `SELECT * FROM hits WHERE CONTAINS($1, '$2')`

	if err := h.DB.SelectContext(ctx, &hits, q, field, value); err != nil {
		return nil, errors.New("no hits found")
	}

	return hits, nil
}
