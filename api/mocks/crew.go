package mocks

import (
	"context"
	"fmt"

	"github.com/stretchr/testify/mock"

	"github.com/mdblp/crew/client/dto"
)

type PermsCall struct {
	result interface{}
	err    error
}

// CrewMockClient The mocked interface to crew api.
type CrewMockClient struct {
	mock.Mock
	nextCall map[string]*PermsCall
}

// NewCrewMock create a new crew mock client
func NewCrewMock() *CrewMockClient {
	return &CrewMockClient{
		nextCall: make(map[string]*PermsCall),
	}
}

func (client *CrewMockClient) SetMockNextCall(key string, expectedResult interface{}, expectedError error) {
	client.nextCall[key] = &PermsCall{
		result: expectedResult,
		err:    expectedError,
	}
}

func (client *CrewMockClient) GetDirectShares(ctx context.Context, authToken string) ([]dto.DataShare, error) {
	permsResponse, ok := client.nextCall[authToken]
	if !ok {
		return nil, fmt.Errorf("Unknown response code[404] from service[http://crew/direct-share]")
	}
	return permsResponse.result.([]dto.DataShare), permsResponse.err
}

// GetDirectSharesWithContext provides a mock function with given fields: ctx, authToken
func (_m *CrewMockClient) GetDirectSharesWithContext(ctx context.Context, authToken string) ([]dto.DataShare, error) {
	ret := _m.Called(ctx, authToken)

	var r0 []dto.DataShare
	if rf, ok := ret.Get(0).(func(context.Context, string) []dto.DataShare); ok {
		r0 = rf(ctx, authToken)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]dto.DataShare)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, authToken)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// returns the map of teams
func (client *CrewMockClient) TeamsForUser(ctx context.Context, authToken string) ([]dto.Team, error) {
	crewResponse, ok := client.nextCall[authToken]
	if !ok {
		return nil, fmt.Errorf("Unknown response code[404] from service[http://crew/teams]")
	}
	if crewResponse.result != nil {
		return crewResponse.result.([]dto.Team), crewResponse.err
	}
	return nil, crewResponse.err
}

// TeamsForUserWithContext provides a mock function with given fields: ctx, authToken
func (_m *CrewMockClient) TeamsForUserWithContext(ctx context.Context, authToken string) ([]dto.Team, error) {
	ret := _m.Called(ctx, authToken)

	var r0 []dto.Team
	if rf, ok := ret.Get(0).(func(context.Context, string) []dto.Team); ok {
		r0 = rf(ctx, authToken)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]dto.Team)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, authToken)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (client *CrewMockClient) GetTeam(ctx context.Context, authToken string, teamId string) (*dto.Team, error) {
	crewResponse, ok := client.nextCall[authToken+teamId]
	if !ok {
		return nil, fmt.Errorf("Unknown response code[404] from service[http://crew/teams]")
	}
	if crewResponse.result != nil {
		return crewResponse.result.(*dto.Team), crewResponse.err
	}
	return nil, crewResponse.err
}

// GetTeamWithContext provides a mock function with given fields: ctx, authToken, teamId
func (_m *CrewMockClient) GetTeamWithContext(ctx context.Context, authToken string, teamId string) (*dto.Team, error) {
	ret := _m.Called(ctx, authToken, teamId)

	var r0 *dto.Team
	if rf, ok := ret.Get(0).(func(context.Context, string, string) *dto.Team); ok {
		r0 = rf(ctx, authToken, teamId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dto.Team)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, authToken, teamId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (client *CrewMockClient) SetPermissions(authToken string, patientID string, viewerID string) error {
	permsResponse, ok := client.nextCall[authToken]
	if !ok {
		return fmt.Errorf("Unknown response code[404] from service[http://crew/direct-share]")
	}
	return permsResponse.err
}

// SetPermissionsWithContext provides a mock function with given fields: ctx, authToken, patientID, viewerID
func (_m *CrewMockClient) SetPermissionsWithContext(ctx context.Context, authToken string, patientID string, viewerID string) error {
	ret := _m.Called(ctx, authToken, patientID, viewerID)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string) error); ok {
		r0 = rf(ctx, authToken, patientID, viewerID)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// returns the list of patients
func (client *CrewMockClient) PatientsForUser(authToken string) ([]dto.Patient, error) {
	crewResponse, ok := client.nextCall[authToken]
	if !ok {
		return nil, fmt.Errorf("Unknown response code[404] from service[http://crew/patients]")
	}
	if crewResponse.result != nil {
		return crewResponse.result.([]dto.Patient), crewResponse.err
	}
	return nil, crewResponse.err
}

// PatientsForUserWithContext provides a mock function with given fields: ctx, authToken
func (_m *CrewMockClient) PatientsForUserWithContext(ctx context.Context, authToken string) ([]dto.Patient, error) {
	ret := _m.Called(ctx, authToken)

	var r0 []dto.Patient
	if rf, ok := ret.Get(0).(func(context.Context, string) []dto.Patient); ok {
		r0 = rf(ctx, authToken)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]dto.Patient)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, authToken)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Add a member to a given team
func (client *CrewMockClient) AddTeamMember(authToken string, member dto.Member) (*dto.Member, error) {
	crewResponse, ok := client.nextCall[authToken+member.UserID]
	if !ok {
		return nil, fmt.Errorf("Unknown response code[404] from service[http://crew/teams/%s/members]", member.TeamID)
	}
	if crewResponse.result != nil {
		return crewResponse.result.(*dto.Member), crewResponse.err
	}
	return nil, crewResponse.err
}

// AddTeamMemberWithContext provides a mock function with given fields: ctx, authToken, member
func (_m *CrewMockClient) AddTeamMemberWithContext(ctx context.Context, authToken string, member dto.Member) (*dto.Member, error) {
	ret := _m.Called(ctx, authToken, member)

	var r0 *dto.Member
	if rf, ok := ret.Get(0).(func(context.Context, string, dto.Member) *dto.Member); ok {
		r0 = rf(ctx, authToken, member)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dto.Member)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, dto.Member) error); ok {
		r1 = rf(ctx, authToken, member)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Update a team member
func (client *CrewMockClient) UpdateTeamMember(authToken string, member dto.Member) (*dto.Member, error) {
	crewResponse, ok := client.nextCall[authToken+member.UserID]
	if !ok {
		return nil, fmt.Errorf("Unknown response code[404] from service[http://crew/teams/%s/members]", member.TeamID)
	}
	if crewResponse.result != nil {
		return crewResponse.result.(*dto.Member), crewResponse.err
	}
	return nil, crewResponse.err
}

// UpdateTeamMemberWithContext provides a mock function with given fields: ctx, authToken, member
func (_m *CrewMockClient) UpdateTeamMemberWithContext(ctx context.Context, authToken string, member dto.Member) (*dto.Member, error) {
	ret := _m.Called(ctx, authToken, member)

	var r0 *dto.Member
	if rf, ok := ret.Get(0).(func(context.Context, string, dto.Member) *dto.Member); ok {
		r0 = rf(ctx, authToken, member)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dto.Member)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, dto.Member) error); ok {
		r1 = rf(ctx, authToken, member)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Remove a member from a team
func (client *CrewMockClient) RemoveTeamMember(authToken string, teamId string, memberId string) error {
	crewResponse, ok := client.nextCall[authToken+memberId]
	if !ok {
		return fmt.Errorf("Unknown response code[404] from service[http://crew/teams/%s/members/%s]", teamId, memberId)
	}
	return crewResponse.err
}

// RemoveTeamMemberWithContext provides a mock function with given fields: ctx, authToken, teamId, memberId
func (_m *CrewMockClient) RemoveTeamMemberWithContext(ctx context.Context, authToken string, teamId string, memberId string) error {
	ret := _m.Called(ctx, authToken, teamId, memberId)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string) error); ok {
		r0 = rf(ctx, authToken, teamId, memberId)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (client *CrewMockClient) GetTeamPatients(ctx context.Context, authToken string, teamId string) ([]dto.Patient, error) {
	crewResponse, ok := client.nextCall["GetTeamPatients"+authToken+teamId]
	if !ok {
		return nil, fmt.Errorf("unknown response code[404] from service[http://crew/teams/%s/patients]", teamId)
	}
	if crewResponse.result != nil {
		return crewResponse.result.([]dto.Patient), crewResponse.err
	}
	return nil, crewResponse.err
}

// GetTeamPatientsWithContext provides a mock function with given fields: ctx, authToken, teamId
func (_m *CrewMockClient) GetTeamPatientsWithContext(ctx context.Context, authToken string, teamId string) ([]dto.Patient, error) {
	ret := _m.Called(ctx, authToken, teamId)

	var r0 []dto.Patient
	if rf, ok := ret.Get(0).(func(context.Context, string, string) []dto.Patient); ok {
		r0 = rf(ctx, authToken, teamId)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]dto.Patient)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, authToken, teamId)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Add or update a patient in a team
func (client *CrewMockClient) AddPatient(authToken string, patient dto.Patient) (*dto.Patient, error) {
	crewResponse, ok := client.nextCall["AddPatient"+authToken+patient.UserID]
	if !ok {
		return nil, fmt.Errorf("Unknown response code[404] from service[http://crew/teams/%s/patients]", patient.TeamID)
	}
	if crewResponse.result != nil {
		return crewResponse.result.(*dto.Patient), crewResponse.err
	}
	return nil, crewResponse.err
}

// AddPatientWithContext provides a mock function with given fields: ctx, authToken, patient
func (_m *CrewMockClient) AddPatientWithContext(ctx context.Context, authToken string, patient dto.Patient) (*dto.Patient, error) {
	ret := _m.Called(ctx, authToken, patient)

	var r0 *dto.Patient
	if rf, ok := ret.Get(0).(func(context.Context, string, dto.Patient) *dto.Patient); ok {
		r0 = rf(ctx, authToken, patient)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dto.Patient)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, dto.Patient) error); ok {
		r1 = rf(ctx, authToken, patient)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (client *CrewMockClient) UpdatePatient(authToken string, patient dto.Patient) (*dto.Patient, error) {
	crewResponse, ok := client.nextCall["UpdatePatient"+authToken+patient.UserID]
	if !ok {
		return nil, fmt.Errorf("Unknown response code[404] from service[http://crew/teams/%s/patients]", patient.TeamID)
	}
	if crewResponse.result != nil {
		return crewResponse.result.(*dto.Patient), crewResponse.err
	}
	return nil, crewResponse.err
}

// UpdatePatientWithContext provides a mock function with given fields: ctx, authToken, patient
func (_m *CrewMockClient) UpdatePatientWithContext(ctx context.Context, authToken string, patient dto.Patient) (*dto.Patient, error) {
	ret := _m.Called(ctx, authToken, patient)

	var r0 *dto.Patient
	if rf, ok := ret.Get(0).(func(context.Context, string, dto.Patient) *dto.Patient); ok {
		r0 = rf(ctx, authToken, patient)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*dto.Patient)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, dto.Patient) error); ok {
		r1 = rf(ctx, authToken, patient)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Remove a patient from a team
func (client *CrewMockClient) RemovePatient(authToken string, teamId string, patientId string) error {
	crewResponse, ok := client.nextCall[authToken+patientId]
	if !ok {
		return fmt.Errorf("Unknown response code[404] from service[http://crew/teams/%s/patients/%s]", teamId, patientId)
	}
	return crewResponse.err
}
