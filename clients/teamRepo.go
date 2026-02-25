package clients

import (
	"context"

	"github.com/mdblp/crew/client/dto"
)

// Interface so that we can mock CrewClient for tests
type CrewRepo interface {

	// returns the map of teams for the user authenticated by the given token
	TeamsForUser(ctx context.Context, authToken string) ([]dto.Team, error)

	// return the team matching the team id provided (if found).
	GetTeam(ctx context.Context, authToken string, teamId string) (*dto.Team, error)

	// Deprecated: Add a member to a given team
	AddTeamMember(authToken string, member dto.Member) (*dto.Member, error)
	// Add a member to a given team
	AddTeamMemberWithContext(ctx context.Context, authToken string, member dto.Member) (*dto.Member, error)

	// Deprecated: Update a team member
	UpdateTeamMember(authToken string, member dto.Member) (*dto.Member, error)
	// Update a team member
	UpdateTeamMemberWithContext(ctx context.Context, authToken string, member dto.Member) (*dto.Member, error)

	// Deprecated: Remove a member from a team
	RemoveTeamMember(authToken string, teamId string, memberId string) error
	// Remove a member from a team
	RemoveTeamMemberWithContext(ctx context.Context, authToken string, teamId string, memberId string) error

	// Patients operations
	// Get Patients for a team
	GetTeamPatients(ctx context.Context, authToken string, teamId string) ([]dto.Patient, error)

	// Deprecated: Add a patient in a team
	AddPatient(authToken string, patient dto.Patient) (*dto.Patient, error)
	// Add a patient in a team
	AddPatientWithContext(ctx context.Context, authToken string, patient dto.Patient) (*dto.Patient, error)

	// Deprecated: Update a patient in a team
	UpdatePatient(authToken string, patient dto.Patient) (*dto.Patient, error)
	// Update a patient in a team
	UpdatePatientWithContext(ctx context.Context, authToken string, patient dto.Patient) (*dto.Patient, error)

	// Remove a patient from a team
	RemovePatient(authToken string, teamId string, memberId string) error

	// Direct shares

	// Deprecated: patientID  -- the Tidepool-assigned patientID
	// viewerID  -- the Tidepool-assigned viewerID
	SetPermissions(authToken string, patientID string, viewerID string) error
	// patientID  -- the Tidepool-assigned patientID
	// viewerID  -- the Tidepool-assigned viewerID
	SetPermissionsWithContext(ctx context.Context, authToken string, patientID string, viewerID string) error

	// return dataGetTeamPatients shares for the user authenticated by the given token
	GetDirectShares(ctx context.Context, authToken string) ([]dto.DataShare, error)
}
