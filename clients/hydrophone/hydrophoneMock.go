package hydrophone

import (
	"context"
	"github.com/stretchr/testify/mock"
)

type HydrophoneMockClient struct {
	mock.Mock
	MockedError    error
	MockedConfirms []Confirmation
}

func NewMock() *HydrophoneMockClient {
	return &HydrophoneMockClient{
		MockedError:    nil,
		MockedConfirms: []Confirmation{},
	}
}

func (client *HydrophoneMockClient) GetPendingInvitations(userID string, authToken string) ([]Confirmation, error) {
	if client.MockedError != nil {
		return nil, client.MockedError
	}
	return client.MockedConfirms, nil
}

//TODO: refactor methods above to use testify like bellow

func (client *HydrophoneMockClient) GetPendingSignup(userId string, authToken string) (*Confirmation, error) {
	args := client.Called(userId, authToken)
	return args.Get(0).(*Confirmation), args.Error(1)
}

func (client *HydrophoneMockClient) CancelSignup(confirm Confirmation, authToken string) error {
	client.Called(confirm, authToken)
	return nil
}

func (client *HydrophoneMockClient) SendNotification(topic string, notif interface{}, authToken string) error {
	client.Called(topic, notif, authToken)
	return nil
}

func (client *HydrophoneMockClient) InviteHcp(ctx context.Context, teamId string, inviteeEmail string, role string, authToken string) (*Confirmation, error) {
	args := client.Called(ctx, teamId, inviteeEmail, role, authToken)
	return args.Get(0).(*Confirmation), args.Error(1)
}

func (client *HydrophoneMockClient) GetSentInvitations(ctx context.Context, userID string, authToken string) ([]Confirmation, error) {
	args := client.Called(ctx, userID, authToken)
	return args.Get(0).([]Confirmation), args.Error(1)
}
