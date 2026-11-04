package hydrophone

import (
	"github.com/mdblp/hydrophone/models"
	"github.com/stretchr/testify/mock"
)

type HydrophoneMockClient struct {
	mock.Mock
}

func NewMock() *HydrophoneMockClient {
	return &HydrophoneMockClient{}
}

func (client *HydrophoneMockClient) GetPendingSignup(userId string, authToken string) (*models.Confirmation, error) {
	args := client.Called(userId, authToken)
	return args.Get(0).(*models.Confirmation), args.Error(1)
}

func (client *HydrophoneMockClient) CancelSignup(confirm models.Confirmation, authToken string) error {
	client.Called(confirm, authToken)
	return nil
}
