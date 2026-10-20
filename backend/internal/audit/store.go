package audit

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Record writes one audit entry. actorID is nil for actions taken before anyone is
// logged in, such as the first-run setup.
func (s *Store) Record(ctx context.Context, actorID *int64, action, entityType string, entityID int64, details any) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO audit_log (actor_id, action, entity_type, entity_id, details)
		VALUES ($1, $2, $3, $4, $5)`, actorID, action, entityType, entityID, detailsJSON)
	return err
}
