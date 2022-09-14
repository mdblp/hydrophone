package api

import (
	"encoding/json"
	"net/http"

	"github.com/mdblp/go-common/clients/status"
	"github.com/mdblp/hydrophone/models"
	log "github.com/sirupsen/logrus"
)

var (
	STATUS_WRONG_NOTIFICATION_TOPIC = "wrong notification topic"
	STATUS_WRONG_APP_PRESCRIPTION   = "missing information in prescription body"
)

func (a *Api) CreateNotification(res http.ResponseWriter, req *http.Request, vars map[string]string) {
	// Only servers can send notifs
	token := a.token(res, req)
	if token == nil {
		return
	}
	if !token.IsServer {
		a.sendModelAsResWithStatus(
			res,
			&status.StatusError{Status: status.NewStatus(http.StatusUnauthorized, STATUS_UNAUTHORIZED)},
			http.StatusUnauthorized,
		)
		return
	}
	// Get topic
	topic := vars["topic"]
	var notif *models.Confirmation
	var emailContent map[string]string

	switch topic {
	case "submit_app_prescription":
		{
			notif, emailContent = a.createAppPrescription(res, req)
		}
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
	a.processNotification(res, req, emailContent, notif)
}

func (a *Api) createAppPrescription(res http.ResponseWriter, req *http.Request) (*models.Confirmation, map[string]string) {
	presc := &models.PrescriptionBody{}
	if err := json.NewDecoder(req.Body).Decode(presc); err != nil {
		log.Printf("CreateAppPrescription: error decoding presc to create [%v]", err)
		statusErr := &status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_DECODING_CONFIRMATION)}
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
	notif, _ := models.NewConfirmationWithContext(models.TypeNotification, models.TemplateNameAppPrescription, presc.PrescriptorId, presc)

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

func (a *Api) processNotification(res http.ResponseWriter, req *http.Request, content map[string]string, invite *models.Confirmation) {
	var inviteeLanguage = GetUserChosenLanguage(req)

	invitedUsr := a.findExistingUser(invite.Email, a.sl.TokenProvide())
	//None exist so lets create the invite
	if invitedUsr != nil {
		invite.UserId = invitedUsr.UserID

		// let's get the invitee user preferences
		inviteePreferences := &models.Preferences{}
		if err := a.seagull.GetCollection(invite.UserId, "preferences", a.sl.TokenProvide(), inviteePreferences); err != nil {
			a.sendError(res, http.StatusInternalServerError, STATUS_ERR_FINDING_USR, "send invitation: error getting invitee user preferences: ", err.Error())
			return
		}
		// does the invitee have a preferred language?
		if inviteePreferences.DisplayLanguage != "" {
			inviteeLanguage = inviteePreferences.DisplayLanguage
		}
	} else {
		invitorPreferences := &models.Preferences{}
		if err := a.seagull.GetCollection(invite.CreatorId, "preferences", a.sl.TokenProvide(), invitorPreferences); err != nil {
			a.sendError(res, http.StatusInternalServerError, STATUS_ERR_FINDING_USR, "send invitation: error getting invitor user preferences: ", err.Error())
			return
		}
		// does the invitee have a preferred language?
		if invitorPreferences.DisplayLanguage != "" {
			inviteeLanguage = invitorPreferences.DisplayLanguage
		}
	}

	if !a.addOrUpdateConfirmation(req.Context(), invite, res) {
		return
	}
	a.logAudit(req, "notif created")

	if err := a.addProfile(invite); err != nil {
		log.Println("SendInvite: ", err.Error())
	} else {
		fullName := invite.Creator.Profile.FullName

		// if invitee is already a user (ie already has an account), he won't go to signup but login instead
		if invite.UserId == "" || content["WebPath"] == "" {
			content["WebPath"] = "login"
		}
		content["Invitor"] = fullName
		content["Email"] = invite.Email
		content["Duration"] = invite.GetReadableDuration()

		if a.createAndSendNotification(req, invite, content, inviteeLanguage) {
			a.logAudit(req, "invite sent")
		} else {
			a.logAudit(req, "invite failed to be sent")
			log.Print("Something happened generating an invite email")
			res.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
	}

	a.sendModelAsResWithStatus(res, invite, http.StatusOK)

}
