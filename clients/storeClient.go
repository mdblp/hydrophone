package clients

import (
	"context"
	"time"

	goComMgo "github.com/mdblp/go-db/v3/mongo"
	"github.com/mdblp/hydrophone/models"
)

type StoreClient interface {
	goComMgo.Storage
	UpsertConfirmation(ctx context.Context, confirmation *models.Confirmation) error
	FindConfirmation(ctx context.Context, confirmation *models.Confirmation) (result *models.Confirmation, err error)
	CountLatestConfirmations(ctx context.Context, confirmation models.Confirmation, createdSince time.Time) (int64, error)
}
