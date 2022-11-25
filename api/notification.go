package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mdblp/go-common/v2/clients/status"
	"github.com/mdblp/hydrophone/models"
	"github.com/mdblp/shoreline/schema"
	log "github.com/sirupsen/logrus"
)

const (
	//Status message we return from the service
	statusExistingInviteMessage  = "There is already an existing invite"
	statusExistingMemberMessage  = "The user is already an existing member"
	statusInviteNotFoundMessage  = "No matching invite was found"
	statusInviteCanceledMessage  = "Invite has been canceled"
	statusInviteNotActiveMessage = "Invite already canceled"
	statusForbiddenMessage       = "Forbidden to perform requested operation"
	statusExpiredMessage         = "Invite has expired"
)

type (
	PrescriptionBody struct {
		Id            string `json:"id"`
		Code          string `json:"code"`
		PatientEmail  string `json:"patientEmail"`
		PatientId     string `json:"patientId"`
		PrescriptorId string `json:"prescriptorId"`
		Product       string `json:"product"`
	}

	Address struct {
		Line1   string `json:"line1" binding:"required,excludesall=<>!?,max=50" bson:"line1,omitempty"`
		Line2   string `json:"line2" binding:"omitempty,excludesall=<>!?,max=50" bson:"line2,omitempty"`
		Zip     string `json:"zip" binding:"required,excludesall=<>!?,max=20" bson:"zip,omitempty"`
		City    string `json:"city" binding:"required,excludesall=<>!?,max=50" bson:"city,omitempty"`
		Country string `json:"country" binding:"required,excludesall=<>!?,max=30" bson:"country,omitempty"`
	}

	InviteTeam struct {
		Email       string `json:"email"`
		InvitorId   string `json:invitorId`
		TeamId      string
		TeamName    string
		TeamPhone   string
		TeamCode    string
		TeamAddress Address
	}

	InviteBody struct {
		Email     string `json:"email"`
		InvitorId string `json:invitorId`
		Team      string `json:"team"`
	}
	//Invite details for generating a new patient monitoring invite
	inviteMonitoringBody struct {
		MonitoringEnd   time.Time `json:"monitoringEnd"`
		ReferringDoctor *string   `json:"referringDoctor,omitempty"`
	}
)

var (
	STATUS_WRONG_NOTIFICATION_TOPIC = "wrong notification topic"
	STATUS_WRONG_APP_PRESCRIPTION   = "missing information in prescription body"
)

func handlerGood(s string) string {
	escapedString := strings.Replace(s, "\n", "", -1)
	escapedString = strings.Replace(escapedString, "\r", "", -1)
	return escapedString
}

func checkInviteBody(ib *InviteBody) *InviteBody {
	var out = &InviteBody{
		Email: handlerGood(ib.Email),
		Team:  handlerGood(ib.Team),
	}
	return out
}

func (a *Api) checkForExistingNotif(ctx context.Context, destEmail, invitorID, token string, t models.Type, res http.ResponseWriter) (bool, *schema.UserData) {

	//already has invite from this user?
	invites, _ := a.Store.FindConfirmations(
		ctx,
		&models.Confirmation{CreatorId: invitorID, Email: destEmail, Type: t},
		[]models.Status{models.StatusPending},
		[]models.Type{},
	)

	if len(invites) > 0 {

		//rule is we cannot send if the invite is not yet expired
		if !invites[0].IsExpired() {
			log.Println(statusExistingInviteMessage)
			log.Println("last invite not yet expired")
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusConflict, statusExistingInviteMessage)}
			a.sendModelAsResWithStatus(res, statusErr, http.StatusConflict)
			return true, nil
		}
	}
	return false, nil
}

// @Summary Send a notification by email
// @Description Create a generic notification, send an email using the template matching the topic provided in the notification url
// @Description this route is only accessible for token servers
// @ID hydrophone-api-SendNotification
// @Accept  json
// @Produce  json
// @Param topic path string true "topic label"
// @Success 200 {array} models.Confirmation
// @Failure 400 {object} status.Status "usereid was not provided"
// @Failure 401 {object} status.Status "Authorization token is missing or does not provided sufficient privileges"
// @Failure 403 {object} status.Status "Not authorized to perform this action, probably you are not a server"
// @Failure 500 {object} status.Status "Internal error"
// @Router /notifications/{topic} [post]
// @security TidepoolAuth
func (a *Api) CreateNotification(res http.ResponseWriter, req *http.Request, vars map[string]string) {
	// Only servers can send notifs
	token := a.token(res, req)
	if token == nil {
		return
	}
	if !token.IsServer {
		a.sendModelAsResWithStatus(
			res,
			&status.StatusError{Status: status.NewStatus(http.StatusForbidden, STATUS_UNAUTHORIZED)},
			http.StatusForbidden,
		)
		return
	}
	// Get topic
	topic := vars["topic"]
	var notif *models.Confirmation
	var emailContent map[string]string

	switch topic {
	case "submit_app_prescription":
		notif, emailContent = a.createAppPrescription(res, req)
	case "invite_direct_share":
		notif, emailContent = a.inviteDirectShare(res, req)
	case "invite_patient_medical_team":
		notif, emailContent = a.inviteMedicalTeam(res, req, true)
	case "invite_hcp_medical_team":
		notif, emailContent = a.inviteMedicalTeam(res, req, false)
	case "promote_medical_team_admin":
		notif, emailContent = a.promoteTeamAdmin(res, req)
	case "leave_medical_team":
		notif, emailContent = a.leaveMedicalTeam(res, req)
	default:
		{
			a.sendModelAsResWithStatus(
				res,
				&status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_WRONG_NOTIFICATION_TOPIC)},
				http.StatusBadRequest,
			)
			return
		}
	}
	if notif == nil {
		return
	}
	a.processNotification(res, req, emailContent, notif)
}

// Send a notification email based on the given email content and confirmation model
func (a *Api) processNotification(res http.ResponseWriter, req *http.Request, content map[string]string, invite *models.Confirmation) {
	var inviteeLanguage = "en"
	creatorMetaData, err := a.seagull.GetCollections(req.Context(), invite.CreatorId, []string{"preferences", "profile"}, a.sl.TokenProvide())
	if err != nil {
		a.sendError(res, http.StatusInternalServerError, STATUS_ERR_FINDING_USR, "send invitation: error getting invitor user preferences: ", err.Error())
		return
	}
	invitedUsr := a.findExistingUser(invite.Email, a.sl.TokenProvide())
	if invitedUsr != nil {
		invite.UserId = invitedUsr.UserID
		// let's get the invitee user preferences
		inviteeLanguage = a.getUserLanguage(invite.UserId, req, res)
	} else {
		// fallback to the creator language
		invitorPreferences := creatorMetaData.Preferences
		if invitorPreferences != nil && invitorPreferences.DisplayLanguageCode != "" {
			inviteeLanguage = invitorPreferences.DisplayLanguageCode
		}
	}

	if !a.addOrUpdateConfirmation(req.Context(), invite, res) {
		return
	}
	a.logAudit(req, "notif created")
	if creatorMetaData.Profile == nil {
		a.sendError(
			res,
			http.StatusInternalServerError,
			STATUS_ERR_FINDING_USR,
			"send notification: error getting sender user profile: ", invite.CreatorId,
		)
		return
	}
	fullName := creatorMetaData.Profile.FullName

	// if invitee is already a user (ie already has an account), he won't go to signup but login instead
	if invite.UserId == "" || content["WebPath"] == "" {
		content["WebPath"] = "login"
	}
	content["FromUser"] = fullName
	content["Email"] = invite.Email
	content["Duration"] = invite.GetReadableDuration()

	invite.Creator.Profile = &models.Profile{FullName: creatorMetaData.Profile.FullName}

	if a.createAndSendNotification(req, invite, content, inviteeLanguage) {
		a.logAudit(req, "invite sent")
	} else {
		a.logAudit(req, "invite failed to be sent")
		log.Print("Something happened generating an invite email")
		res.WriteHeader(http.StatusUnprocessableEntity)
		return
	}

	a.sendModelAsResWithStatus(res, invite, http.StatusOK)
}

// @Summary Get list of received notifications for logged-in user
// @Description  Get list of received notifications that have been sent to this user but not yet acted upon.
// @ID hydrophone-api-GetReceivedNotifications
// @Accept  json
// @Produce  json
// @Param userid path string true "user id"
// @Success 200 {array} models.Confirmation
// @Failure 400 {object} status.Status "usereid was not provided"
// @Failure 401 {object} status.Status "Authorization token is missing or does not provided sufficient privileges"
// @Failure 403 {object} status.Status "Authorization token is invalid"
// @Failure 500 {object} status.Status "Error while extracting the data"
// @Router /notifications/{userid} [get]
// @security TidepoolAuth
func (a *Api) GetReceivedNotifications(res http.ResponseWriter, req *http.Request, vars map[string]string) {
	if token := a.token(res, req); token != nil {
		recipientId := vars["userid"]

		if recipientId == "" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		// Non-server tokens only legit when for same userid
		if !token.IsServer && recipientId != token.UserId {
			log.Printf("GetReceivedInvitations %s ", STATUS_UNAUTHORIZED)
			a.sendModelAsResWithStatus(res, status.StatusError{status.NewStatus(http.StatusUnauthorized, STATUS_UNAUTHORIZED)}, http.StatusUnauthorized)
			return
		}

		// TODO: invert the control: Auth0 should update ids in hydrophone when a new user is created
		// This would avoid multiple calls to auth0 and shoreline.
		recipient := a.findExistingUser(recipientId, a.sl.TokenProvide())

		types := []models.Type{models.TypeNotification}

		status := []models.Status{
			models.StatusPending,
		}
		//find all outstanding notifs were this user is the invite//
		found, err := a.Store.FindConfirmations(
			req.Context(),
			&models.Confirmation{Email: recipient.Emails[0]},
			status,
			types,
		)
		if err != nil {
			log.Printf("GetReceivedNotifications: error [%v] when finding pending notifs ", err)
		}

		if notifs := a.checkFoundConfirmations(req.Context(), res, found, err); notifs != nil {
			if len(notifs) > 0 {
				a.ensureIdSet(req.Context(), recipientId, notifs)
				log.Printf("GetReceivedNotifications: found and have checked [%d] invites ", len(notifs))
				a.logAudit(req, "get received notifs")
			}
			a.sendModelAsResWithStatus(res, notifs, http.StatusOK)
		}
	}
}

// Notififs specific logics

//Prepare email content and confirm object for topic "create prescription"
func (a *Api) createAppPrescription(res http.ResponseWriter, req *http.Request) (*models.Confirmation, map[string]string) {
	presc := &PrescriptionBody{}
	if err := json.NewDecoder(req.Body).Decode(presc); err != nil {
		log.Printf("CreateAppPrescription: error decoding presc to create [%v]", err)
		statusErr := &status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_DECODING_NOTIFICATION)}
		a.sendModelAsResWithStatus(res, statusErr, http.StatusBadRequest)
		return nil, nil
	}

	if presc.PatientEmail == "" || presc.Id == "" || presc.PrescriptorId == "" {
		a.sendModelAsResWithStatus(
			res,
			&status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_WRONG_APP_PRESCRIPTION)},
			http.StatusBadRequest,
		)
		return nil, nil
	}
	notif, _ := models.NewConfirmationWithContext(models.TypeDevicePrescription, models.TemplateNameAppPrescription, presc.PrescriptorId, presc)

	// if the invitee is already a Tidepool user, we can use his preferences
	notif.Email = presc.PatientEmail

	emailContent := map[string]string{
		"Product": presc.Product,
		// Prescription code may not be required
		"PrescriptionCode": presc.Code,
		"WebPath":          "prescriptions",
	}
	return notif, emailContent
}

//Prepare email content and confirm object for topic "add direct share"
func (a *Api) inviteDirectShare(res http.ResponseWriter, req *http.Request) (*models.Confirmation, map[string]string) {
	unescapedBody := &InviteBody{}
	if err := json.NewDecoder(req.Body).Decode(unescapedBody); err != nil {
		log.Printf("addDirectShare: error decoding body to create [%v]", err)
		statusErr := &status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_DECODING_NOTIFICATION)}
		a.sendModelAsResWithStatus(res, statusErr, http.StatusBadRequest)
		return nil, nil
	}
	invite := checkInviteBody(unescapedBody)

	if invite.Email == "" || invite.InvitorId == "" {
		a.sendModelAsResWithStatus(
			res,
			&status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_WRONG_APP_PRESCRIPTION)},
			http.StatusBadRequest,
		)
		return nil, nil
	}
	notif, _ := models.NewConfirmation(models.TypeDataShareInvite, models.TemplateNameCareteamInvite, invite.InvitorId)

	// if the invitee is already a Tidepool user, we can use his preferences
	notif.Email = invite.Email

	var webPath = "signup"
	if notif.UserId != "" {
		webPath = "login"
	}

	emailContent := map[string]string{
		"WebPath": webPath,
	}
	return notif, emailContent
}

//Prepare email content and confirm object for topic "add direct share"
func (a *Api) inviteMedicalTeam(res http.ResponseWriter, req *http.Request, isPatient bool) (*models.Confirmation, map[string]string) {
	invite := &InviteTeam{}
	if err := json.NewDecoder(req.Body).Decode(invite); err != nil {
		log.Printf("addDirectShare: error decoding body to create [%v]", err)
		statusErr := &status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_DECODING_NOTIFICATION)}
		a.sendModelAsResWithStatus(res, statusErr, http.StatusBadRequest)
		return nil, nil
	}

	if invite.Email == "" || invite.InvitorId == "" {
		a.sendModelAsResWithStatus(
			res,
			&status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_WRONG_APP_PRESCRIPTION)},
			http.StatusBadRequest,
		)
		return nil, nil
	}
	var notif *models.Confirmation
	if isPatient {
		notif, _ = models.NewConfirmation(models.TypeMedicalTeamInvite, models.TemplateNameMedicalteamInvite, invite.InvitorId)
	} else {
		notif, _ = models.NewConfirmation(models.TypeMedicalTeamInvite, models.TemplateNameMedicalteamInvite, invite.InvitorId)
	}

	// if the invitee is already a Tidepool user, we can use his preferences
	notif.Email = invite.Email

	var webPath = "signup"
	if notif.UserId != "" {
		webPath = "login"
	}

	emailContent := map[string]string{
		"MedicalteamName":          invite.TeamName,
		"MedicalteamAddress":       formatAddress(invite.TeamAddress),
		"MedicalteamPhone":         invite.TeamPhone,
		"MedicalteamIentification": invite.TeamCode,
		"WebPath":                  webPath,
	}
	return notif, emailContent
}

func (a *Api) leaveMedicalTeam(res http.ResponseWriter, req *http.Request) (*models.Confirmation, map[string]string) {
	invite := &InviteTeam{}
	if err := json.NewDecoder(req.Body).Decode(invite); err != nil {
		log.Printf("addDirectShare: error decoding body to create [%v]", err)
		statusErr := &status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_DECODING_NOTIFICATION)}
		a.sendModelAsResWithStatus(res, statusErr, http.StatusBadRequest)
		return nil, nil
	}

	if invite.Email == "" || invite.InvitorId == "" {
		a.sendModelAsResWithStatus(
			res,
			&status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_WRONG_APP_PRESCRIPTION)},
			http.StatusBadRequest,
		)
		return nil, nil
	}
	notif, _ := models.NewConfirmation(models.TypeMedicalTeamInvite, models.TemplateNameMedicalteamInvite, invite.InvitorId)

	// if the invitee is already a Tidepool user, we can use his preferences
	notif.Email = invite.Email

	emailContent := map[string]string{
		"MedicalteamName": invite.TeamName,
	}
	return notif, emailContent
}

//Prepare email content and confirm object for topic "add direct share"
func (a *Api) invitePatientMonitoringTeam(res http.ResponseWriter, req *http.Request) (*models.Confirmation, map[string]string) {
	invite := &InviteTeam{}
	if err := json.NewDecoder(req.Body).Decode(invite); err != nil {
		log.Printf("invitePatientMonitoringTeam: error decoding body to create [%v]", err)
		statusErr := &status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_DECODING_NOTIFICATION)}
		a.sendModelAsResWithStatus(res, statusErr, http.StatusBadRequest)
		return nil, nil
	}

	if invite.Email == "" || invite.InvitorId == "" {
		a.sendModelAsResWithStatus(
			res,
			&status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_WRONG_APP_PRESCRIPTION)},
			http.StatusBadRequest,
		)
		return nil, nil
	}
	notif, _ := models.NewConfirmation(models.TypeMedicalTeamMonitoringInvite, models.TemplateNameMedicalteamMonitoringInvite, invite.InvitorId)

	// if the invitee is already a Tidepool user, we can use his preferences
	notif.Email = invite.Email

	var webPath = "signup"
	if notif.UserId != "" {
		webPath = "login"
	}

	emailContent := map[string]string{
		"MedicalteamName":          invite.TeamName,
		"MedicalteamAddress":       formatAddress(invite.TeamAddress),
		"MedicalteamPhone":         invite.TeamPhone,
		"MedicalteamIentification": invite.TeamCode,
		"WebPath":                  webPath,
	}
	return notif, emailContent
}

//Prepare email content and confirm object for topic "add direct share"
func (a *Api) promoteTeamAdmin(res http.ResponseWriter, req *http.Request) (*models.Confirmation, map[string]string) {
	unescapedBody := &InviteBody{}
	if err := json.NewDecoder(req.Body).Decode(unescapedBody); err != nil {
		log.Printf("addDirectShare: error decoding body to create [%v]", err)
		statusErr := &status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_DECODING_NOTIFICATION)}
		a.sendModelAsResWithStatus(res, statusErr, http.StatusBadRequest)
		return nil, nil
	}
	invite := checkInviteBody(unescapedBody)

	if invite.Email == "" || invite.InvitorId == "" {
		a.sendModelAsResWithStatus(
			res,
			&status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_WRONG_APP_PRESCRIPTION)},
			http.StatusBadRequest,
		)
		return nil, nil
	}
	notif, _ := models.NewConfirmation(models.TypeMedicalTeamDoAdmin, models.TemplateNameMedicalteamDoAdmin, invite.InvitorId)

	// if the invitee is already a Tidepool user, we can use his preferences
	notif.Email = invite.Email

	emailContent := map[string]string{
		"MedicalteamName": invite.Team,
	}
	return notif, emailContent
}

func formatAddress(addr Address) string {
	if addr.Line2 != "" {
		return fmt.Sprintf("%s %s, %s %s, %s", addr.Line1, addr.Line2, addr.Zip, addr.City, addr.Country)
	} else {
		return fmt.Sprintf("%s, %s %s, %s", addr.Line1, addr.Zip, addr.City, addr.Country)
	}
}
