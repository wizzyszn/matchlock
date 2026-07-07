package db

import (
	"context"

	"gorm.io/gorm"
)

// Probe wraps a GORM handle for readiness checks.
type Probe struct {
	DB *gorm.DB
}

func (p Probe) Ping(ctx context.Context) error {
	if p.DB == nil {
		return nil
	}
	return Ping(ctx, p.DB)
}