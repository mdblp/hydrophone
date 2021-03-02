package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	commonClients "github.com/tidepool-org/go-common/clients"
	"github.com/tidepool-org/go-common/clients/shoreline"
	"github.com/tidepool-org/go-common/clients/status"
	"github.com/tidepool-org/hydrophone/models"
)

const (
	//Status message we return from the service
	statusExistingInviteMessage  = "There is already an existing invite"
	statusExistingMemberMessage  = "The user is already an existing member"
	statusInviteNotFoundMessage  = "No matching invite was found"
	statusInviteCanceledMessage  = "Invite has been canceled"
	statusInviteNotActiveMessage = "Invite already canceled"
	statusForbiddenMessage       = "Forbidden to perform requested operation"
)

type (
	//Invite details for generating a new invite
	inviteBody struct {
		Email       string                    `json:"email"`
		Permissions commonClients.Permissions `json:"permissions"`
		TeamID      string                    `json:"teamId"`
		IsAdmin     string                    `json:"isAdmin"`
	}
)

//Checks do they have an existing invite or are they already a team member
//Or are they an existing user and already in the group?
func (a *Api) checkForDuplicateInvite(ctx context.Context, inviteeEmail, invitorID, token string, res http.ResponseWriter) (bool, *shoreline.UserData) {

	//already has invite from this user?
	invites, _ := a.Store.FindConfirmations(
		ctx,
		&models.Confirmation{CreatorId: invitorID, Email: inviteeEmail, Type: models.TypeCareteamInvite},
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

	//already in the group?
	invitedUsr := a.findExistingUser(inviteeEmail, a.sl.TokenProvide())

	if invitedUsr != nil && invitedUsr.UserID != "" {
		if shares, err := a.perms.GetDirectShares(token); err != nil {
			log.Printf("error checking if user is in group [%v]", err)
		} else if len(shares) > 0 {
			for _, share := range shares {
				if share.ViewerId == invitedUsr.UserID {
					//already sharing data with this user:
					log.Println(statusExistingMemberMessage)
					statusErr := &status.StatusError{Status: status.NewStatus(http.StatusConflict, statusExistingMemberMessage)}
					a.sendModelAsResWithStatus(res, statusErr, http.StatusConflict)
					return true, invitedUsr
				}
			}

		}
		return false, invitedUsr
	}
	return false, nil
}

//Checks do they have an existing invite or are they already a team member
//Or are they an existing user and already in the group?
func (a *Api) checkForDuplicateTeamInvite(ctx context.Context, inviteeEmail, invitorID, teamID, token string, invite models.Type, res http.ResponseWriter) (bool, *shoreline.UserData) {

	//already has invite from this user?
	invites, _ := a.Store.FindConfirmations(
		ctx,
		&models.Confirmation{
			CreatorId: invitorID,
			Email:     inviteeEmail,
			TeamID:    teamID,
			Type:      invite},
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

	invitedUsr := a.findExistingUser(inviteeEmail, a.sl.TokenProvide())
	// TODO
	// call the teams service to check if the user is already a member
	if invitedUsr != nil {
		isMember, err := a.isTeamMember(invitedUsr.UserID, teamID)
		if isMember || err != nil {
			log.Printf("checkForDuplicateTeamInvite: invited [%s] user is already a member of [%s]", inviteeEmail, teamID)
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusConflict, statusExistingMemberMessage)}
			a.sendModelAsResWithStatus(res, statusErr, http.StatusConflict)
			return true, invitedUsr
		}
	}
	return false, nil
}

func (a *Api) getUserPreferences(userid string, res http.ResponseWriter) *models.Preferences {
	// let's get the invitee user preferences
	inviteePreferences := &models.Preferences{}
	if err := a.seagull.GetCollection(userid, "preferences", a.sl.TokenProvide(), inviteePreferences); err != nil {
		a.sendError(
			res,
			http.StatusInternalServerError,
			STATUS_ERR_FINDING_USR,
			"send invitation: error getting invitee user preferences: ",
			err.Error())
	}
	return inviteePreferences
}

func (a *Api) getUserLanguage(userid string, res http.ResponseWriter) string {
	// let's get the invitee user preferences
	language := "en"
	inviteePreferences := a.getUserPreferences(userid, res)
	// does the invitee have a preferred language?
	if inviteePreferences.DisplayLanguage != "" {
		language = inviteePreferences.DisplayLanguage
	}
	return language
}

// @Summary Get list of received invitations for logged-in user
// @Description  Get list of received invitations that have been sent to this user but not yet acted upon.
// @ID hydrophone-api-GetReceivedInvitations
// @Accept  json
// @Produce  json
// @Param userid path string true "user id"
// @Success 200 {string} string "OK"
// @Failure 400 {object} status.Status "usereid was not provided"
// @Failure 401 {object} status.Status "Authorization token is missing or does not provided sufficient privileges"
// @Failure 403 {object} status.Status "Authorization token is invalid"
// @Failure 404 {object} status.Status "no invitations found for this user"
// @Failure 500 {object} status.Status "Error while extracting the data"
// @Router /invitations/{userid} [get]
// @security TidepoolAuth
func (a *Api) GetReceivedInvitations(res http.ResponseWriter, req *http.Request, vars map[string]string) {
	if token := a.token(res, req); token != nil {
		inviteeID := vars["userid"]

		if inviteeID == "" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}
		// Non-server tokens only legit when for same userid
		if !token.IsServer && inviteeID != token.UserID {
			log.Printf("GetReceivedInvitations %s ", STATUS_UNAUTHORIZED)
			a.sendModelAsResWithStatus(res, status.StatusError{status.NewStatus(http.StatusUnauthorized, STATUS_UNAUTHORIZED)}, http.StatusUnauthorized)
			return
		}

		invitedUsr := a.findExistingUser(inviteeID, req.Header.Get(TP_SESSION_TOKEN))

		types := []models.Type{
			models.TypeCareteamInvite,
			models.TypeMedicalTeamInvite,
		}
		status := []models.Status{
			models.StatusPending,
		}
		//find all oustanding invites were this user is the invite//
		found, err := a.Store.FindConfirmations(
			req.Context(),
			&models.Confirmation{Email: invitedUsr.Emails[0]},
			status,
			types,
		)

		//log.Printf("GetReceivedInvitations: found [%d] pending invite(s)", len(found))
		if err != nil {
			log.Printf("GetReceivedInvitations: error [%v] when finding pending invites ", err)
		}

		if invites := a.checkFoundConfirmations(res, found, err); invites != nil {
			a.ensureIdSet(req.Context(), inviteeID, invites)
			log.Printf("GetReceivedInvitations: found and have checked [%d] invites ", len(invites))
			a.logAudit(req, "get received invites")
			a.sendModelAsResWithStatus(res, invites, http.StatusOK)
			return
		}
	}
	return
}

// @Summary Get the still-pending invitations for a group you own or are an admin of
// @Description  Get list of invitations you have sent that have not been accepted.
// @Description  There is no way to tell if an invitation has been ignored.
// @ID hydrophone-api-GetSentInvitations
// @Accept  json
// @Produce  json
// @Param userid path string true "user id"
// @Success 200 {array} models.Confirmation
// @Failure 400 {object} status.Status "usereid was not provided"
// @Failure 401 {object} status.Status "Authorization token is missing or does not provided sufficient privileges"
// @Failure 403 {object} status.Status "Authorization token is invalid"
// @Failure 404 {object} status.Status "no invitations found for this user"
// @Failure 500 {object} status.Status "Error while extracting the data"
// @Router /invite/{userid} [get]
// @security TidepoolAuth
func (a *Api) GetSentInvitations(res http.ResponseWriter, req *http.Request, vars map[string]string) {
	if token := a.token(res, req); token != nil {

		invitorID := vars["userid"]

		if invitorID == "" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if !a.isAuthorizedUser(token, invitorID) {
			a.sendError(res, http.StatusUnauthorized, STATUS_UNAUTHORIZED)
			return
		}

		//find all invites I have sent that are pending or declined
		found, err := a.Store.FindConfirmations(
			req.Context(),
			&models.Confirmation{CreatorId: invitorID, Type: models.TypeCareteamInvite},
			[]models.Status{models.StatusPending, models.StatusDeclined},
			[]models.Type{models.TypeCareteamInvite, models.TypeMedicalTeamInvite, models.TypeMedicalTeamDoAdmin, models.TypeMedicalTeamRemove},
		)
		if invitations := a.checkFoundConfirmations(res, found, err); invitations != nil {
			a.logAudit(req, "get sent invites")
			a.sendModelAsResWithStatus(res, invitations, http.StatusOK)
			return
		}
	}
	return
}

//Accept the given invite
//
// http.StatusOK when accepted
// http.StatusBadRequest when the incoming data is incomplete or incorrect
// http.StatusForbidden when mismatch of user ID's, type or status
// @Summary Accept the given invite
// @Description  This would be PUT by the web page at the link in the invite email. No authentication is required.
// @ID hydrophone-api-acceptInvite
// @Accept  json
// @Produce  json
// @Param userid path string true "invitee id"
// @Param invitedby path string true "invitor id"
// @Param invitation body models.Confirmation true "invitation details"
// @Success 200 {string} string "OK"
// @Failure 400 {object} status.Status "inviteeid, invitorid or/and the payload is missing or malformed"
// @Failure 401 {object} status.Status "Authorization token is missing or does not provided sufficient privileges"
// @Failure 403 {object} status.Status "Operation is forbiden. Either the authorization token is invalid or this invite cannot be accepted"
// @Failure 404 {object} status.Status "invitation not found"
// @Failure 500 {object} status.Status "Error (internal) while processing the data"
// @Router /accept/invite/{userid}/{invitedby} [put]
// @security TidepoolAuth
func (a *Api) AcceptInvite(res http.ResponseWriter, req *http.Request, vars map[string]string) {
	if token := a.token(res, req); token != nil {

		inviteeID := vars["userid"]
		invitorID := vars["invitedby"]

		if inviteeID == "" || invitorID == "" {
			log.Printf("AcceptInvite inviteeID %s or invitorID %s not set", inviteeID, invitorID)
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		// Non-server tokens only legit when for same userid
		if !token.IsServer && inviteeID != token.UserID {
			log.Println("AcceptInvite ", STATUS_UNAUTHORIZED)
			a.sendModelAsResWithStatus(
				res,
				status.StatusError{Status: status.NewStatus(http.StatusUnauthorized, STATUS_UNAUTHORIZED)},
				http.StatusUnauthorized,
			)
			return
		}

		accept := &models.Confirmation{}
		if err := json.NewDecoder(req.Body).Decode(accept); err != nil {
			log.Printf("AcceptInvite error decoding invite data: %v\n", err)
			a.sendModelAsResWithStatus(
				res,
				&status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_DECODING_CONFIRMATION)},
				http.StatusBadRequest,
			)
			return
		}

		if accept.Key == "" {
			log.Println("AcceptInvite has no confirmation key set")
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		conf, err := a.findExistingConfirmation(req.Context(), accept, res)
		if err != nil {
			log.Printf("AcceptInvite error while finding confirmation [%s]\n", err.Error())
			a.sendModelAsResWithStatus(res, err, http.StatusInternalServerError)
			return
		}
		if conf == nil {
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusNotFound, statusInviteNotFoundMessage)}
			log.Println("AcceptInvite ", statusErr.Error())
			a.sendModelAsResWithStatus(res, statusErr, http.StatusNotFound)
			return
		}

		validationErrors := []error{}

		conf.ValidateStatus(models.StatusPending, &validationErrors).
			ValidateType(models.TypeCareteamInvite, &validationErrors).
			ValidateUserID(inviteeID, &validationErrors).
			ValidateCreatorID(invitorID, &validationErrors)

		if len(validationErrors) > 0 {
			for _, validationError := range validationErrors {
				log.Println("AcceptInvite forbidden as there was a expectation mismatch", validationError)
			}
			a.sendModelAsResWithStatus(
				res,
				&status.StatusError{Status: status.NewStatus(http.StatusForbidden, statusForbiddenMessage)},
				http.StatusForbidden,
			)
			return
		}

		if err := a.perms.SetPermissions(a.sl.TokenProvide(), invitorID, inviteeID); err != nil {
			log.Printf("AcceptInvite error setting permissions [%v]\n", err)
			a.sendModelAsResWithStatus(
				res,
				&status.StatusError{Status: status.NewStatus(http.StatusInternalServerError, STATUS_ERR_DECODING_CONFIRMATION)},
				http.StatusInternalServerError,
			)
			return
		}
		log.Printf("AcceptInvite: permissions were set for [%v -> %v] after an invite was accepted", invitorID, inviteeID)
		conf.UpdateStatus(models.StatusCompleted)
		if !a.addOrUpdateConfirmation(req.Context(), conf, res) {
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusInternalServerError, STATUS_ERR_SAVING_CONFIRMATION)}
			log.Println("AcceptInvite ", statusErr.Error())
			a.sendModelAsResWithStatus(res, statusErr, http.StatusInternalServerError)
			return
		}
		a.logAudit(req, "acceptinvite")
		res.WriteHeader(http.StatusOK)
		res.Write([]byte(STATUS_OK))
		return
	}
}

// @Summary Cancel an invite
// @Description Cancel an invite that has been sent to an email address
// @ID hydrophone-api-cancelInvite
// @Accept  json
// @Produce  json
// @Param userid path string true "invitor id"
// @Param invited_address path string true "invited email address"
// @Success 200 {string} string "OK"
// @Failure 400 {object} status.Status "invited_address and/or invitorid is missing or incorrect"
// @Failure 401 {object} status.Status "Authorization token is missing or does not provided sufficient privileges"
// @Failure 403 {object} status.Status "Authorization token is invalid"
// @Failure 404 {object} status.Status "invitation not found"
// @Failure 500 {object} status.Status "Error (internal) while processing the data"
// @Router /{userid}/invited/{invited_address} [put]
// @security TidepoolAuth
func (a *Api) CancelInvite(res http.ResponseWriter, req *http.Request, vars map[string]string) {
	if token := a.token(res, req); token != nil {

		invitorID := vars["userid"]
		email := vars["invited_address"]

		if invitorID == "" || email == "" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if !a.isAuthorizedUser(token, invitorID) {
			a.sendError(res, http.StatusUnauthorized, STATUS_UNAUTHORIZED)
			return
		}

		invite := &models.Confirmation{
			Email:     email,
			CreatorId: invitorID,
			Creator:   models.Creator{},
			Type:      models.TypeCareteamInvite,
		}

		if conf, err := a.findExistingConfirmation(req.Context(), invite, res); err != nil {
			log.Printf("CancelInvite: finding [%s]", err.Error())
			a.sendModelAsResWithStatus(res, err, http.StatusInternalServerError)
		} else if conf != nil {
			//cancel the invite
			conf.UpdateStatus(models.StatusCanceled)

			if a.addOrUpdateConfirmation(req.Context(), conf, res) {
				a.logAudit(req, "cancelled invite")
				res.WriteHeader(http.StatusOK)
				return
			}
		}
		statusErr := &status.StatusError{Status: status.NewStatus(http.StatusNotFound, statusInviteNotFoundMessage)}
		log.Printf("CancelInvite: [%s]", statusErr.Error())
		a.sendModelAsResWithStatus(res, statusErr, http.StatusNotFound)
		return
	}
	return
}

// @Summary Dismiss an invite
// @Description Invitee can dismiss an invite
// @ID hydrophone-api-dismissInvite
// @Accept  json
// @Produce  json
// @Param userid path string true "invitor id"
// @Param invitedby path string true "invited id"
// @Param invitation body models.Confirmation true "invitation details"
// @Success 200 {string} string "OK"
// @Failure 400 {object} status.Status "inviteeid, invitorid or/and the payload is missing or malformed"
// @Failure 401 {object} status.Status "Authorization token is missing or does not provided sufficient privileges"
// @Failure 403 {object} status.Status "Authorization token is invalid"
// @Failure 404 {object} status.Status "invitation not found"
// @Failure 500 {object} status.Status "Error (internal) while processing the data"
// @Router /dismiss/invite/{userid}/{invitedby} [put]
// @security TidepoolAuth
func (a *Api) DismissInvite(res http.ResponseWriter, req *http.Request, vars map[string]string) {
	if token := a.token(res, req); token != nil {

		inviteeID := vars["userid"]
		invitorID := vars["invitedby"]

		if inviteeID == "" || invitorID == "" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		// Non-server tokens only legit when for same userid
		if !token.IsServer && inviteeID != token.UserID {
			log.Printf("DismissInvite %s ", STATUS_UNAUTHORIZED)
			a.sendModelAsResWithStatus(res, status.StatusError{status.NewStatus(http.StatusUnauthorized, STATUS_UNAUTHORIZED)}, http.StatusUnauthorized)
			return
		}

		dismiss := &models.Confirmation{}
		if err := json.NewDecoder(req.Body).Decode(dismiss); err != nil {
			log.Printf("DismissInvite: error decoding invite to dismiss [%v]", err)
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_DECODING_CONFIRMATION)}
			a.sendModelAsResWithStatus(res, statusErr, http.StatusBadRequest)
			return
		}

		if dismiss.Key == "" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if conf, err := a.findExistingConfirmation(req.Context(), dismiss, res); err != nil {
			log.Printf("DismissInvite: finding [%s]", err.Error())
			a.sendModelAsResWithStatus(res, err, http.StatusInternalServerError)
			return
		} else if conf != nil {

			conf.UpdateStatus(models.StatusDeclined)

			if a.addOrUpdateConfirmation(req.Context(), conf, res) {
				a.logAudit(req, "dismissinvite")
				res.WriteHeader(http.StatusOK)
				return
			}
		}
		statusErr := &status.StatusError{Status: status.NewStatus(http.StatusNotFound, statusInviteNotFoundMessage)}
		log.Printf("DismissInvite: [%s]", statusErr.Error())
		a.sendModelAsResWithStatus(res, statusErr, http.StatusNotFound)
		return
	}
	return
}

// @Summary Dismiss a team invite
// @Description Invitee can dismiss a team invite
// @ID hydrophone-api-dismissTeamInvite
// @Accept  json
// @Produce  json
// @Param teamid path string true "Team ID"
// @Success 200 {string} string "OK"
// @NotModified 304 {string} "not modified"
// @Failure 400 {object} status.Status "inviteeid or/and the payload is missing or malformed"
// @Failure 401 {object} status.Status "Authorization token is missing or does not provided sufficient privileges"
// @Failure 403 {object} status.Status "Authorization token is invalid"
// @Failure 404 {object} status.Status "invitation not found"
// @Failure 500 {object} status.Status "Error (internal) while processing the data"
// @Router /dismiss/team-invite/{teamid} [put]
// @security TidepoolAuth
func (a *Api) DismissTeamInvite(res http.ResponseWriter, req *http.Request, vars map[string]string) {
	if token := a.token(res, req); token != nil {

		teamID := vars["teamid"]
		inviteeID := token.UserID

		if teamID == "" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		dismiss := &models.Confirmation{}
		if err := json.NewDecoder(req.Body).Decode(dismiss); err != nil {
			log.Printf("DismissInvite: error decoding invite to dismiss [%v]", err)
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_DECODING_CONFIRMATION)}
			a.sendModelAsResWithStatus(res, statusErr, http.StatusBadRequest)
			return
		}

		// key of the request
		if dismiss.Key == "" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if isAdmin, err := a.isTeamAdmin(inviteeID, teamID); isAdmin && err == nil {
			// as team admin you can act on behalf of members
			// for any invitation for the given team
			dismiss.TeamID = teamID
		} else if !token.IsServer {
			// non server tokens, set the userid and teamId
			dismiss.TeamID = teamID
			dismiss.UserId = inviteeID
		}

		if conf, err := a.findExistingConfirmation(req.Context(), dismiss, res); err != nil {
			log.Printf("DismissInvite: finding [%s]", err.Error())
			a.sendModelAsResWithStatus(res, err, http.StatusInternalServerError)
			return
		} else if conf != nil {

			if conf.Status != models.StatusDeclined && conf.Status != models.StatusCanceled {
				conf.UpdateStatus(models.StatusDeclined)

				if a.addOrUpdateConfirmation(req.Context(), conf, res) {
					log.Printf("dismiss invite [%s] for [%s]", dismiss.Key, dismiss.TeamID)
					a.logAudit(req, "dismissinvite ")
					res.WriteHeader(http.StatusOK)
					return
				}
			}
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusNotModified, statusInviteNotActiveMessage)}
			log.Printf("DismissInvite: [%s]", statusErr.Error())
			a.sendModelAsResWithStatus(res, statusErr, statusErr.Code)
			return
		}
		statusErr := &status.StatusError{Status: status.NewStatus(http.StatusNotFound, statusInviteNotFoundMessage)}
		log.Printf("DismissInvite: [%s]", statusErr.Error())
		a.sendModelAsResWithStatus(res, statusErr, http.StatusNotFound)
		return
	}
	return
}

// @Summary Send a invite to join a patient's team
// @Description  Send a invite to new or existing users to join the patient's team
// @ID hydrophone-api-SendInvite
// @Accept  json
// @Produce  json
// @Param userid path string true "invitor user id"
// @Success 200 {object} models.Confirmation "invite details"
// @Failure 400 {object} status.Status "userId was not provided or the payload is missing/malformed"
// @Failure 401 {object} status.Status "Authorization token is missing or does not provided sufficient privileges"
// @Failure 403 {object} status.Status "Authorization token is invalid"
// @Failure 409 {object} status.Status "user already has a pending or declined invite OR user is already part of the team"
// @Failure 422 {object} status.Status "Error when sending the email (probably caused by the mailling service"
// @Failure 500 {object} status.Status "Internal error while processing the invite, detailled error returned in the body"
// @Router /send/invite/{userid} [post]
// @security TidepoolAuth
func (a *Api) SendInvite(res http.ResponseWriter, req *http.Request, vars map[string]string) {
	// By default, the invitee language will be "en" for Englih (as we don't know which language suits him)
	// In case the invitee is a known user, the language will be overriden in a later step
	var inviteeLanguage = "en"
	if token := a.token(res, req); token != nil {

		invitorID := vars["userid"]

		if invitorID == "" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if !a.isAuthorizedUser(token, invitorID) {
			a.sendError(res, http.StatusUnauthorized, STATUS_UNAUTHORIZED)
			return
		}

		defer req.Body.Close()
		var ib = &inviteBody{}
		if err := json.NewDecoder(req.Body).Decode(ib); err != nil {
			log.Printf("SendInvite: error decoding invite to detail %v\n", err)
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_DECODING_CONFIRMATION)}
			a.sendModelAsResWithStatus(res, statusErr, http.StatusBadRequest)
			return
		}

		if ib.Email == "" || ib.Permissions == nil {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		if existingInvite, invitedUsr := a.checkForDuplicateInvite(req.Context(), ib.Email, invitorID, req.Header.Get(TP_SESSION_TOKEN), res); existingInvite == true {
			log.Printf("SendInvite: invited [%s] user already has or had an invite", ib.Email)
			return
		} else {
			//None exist so lets create the invite
			invite, _ := models.NewConfirmationWithContext(models.TypeCareteamInvite, models.TemplateNameCareteamInvite, invitorID, ib.Permissions)

			// if the invitee is already a Tidepool user, we can use his preferences
			invite.Email = ib.Email
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
			}

			if a.addOrUpdateConfirmation(req.Context(), invite, res) {
				a.logAudit(req, "invite created")

				if err := a.addProfile(invite); err != nil {
					log.Println("SendInvite: ", err.Error())
				} else {

					fullName := invite.Creator.Profile.FullName

					if invite.Creator.Profile.Patient.IsOtherPerson {
						fullName = invite.Creator.Profile.Patient.FullName
					}

					var webPath = "signup/clinician"

					// if invitee is already a user (ie already has an account), he won't go to signup but login instead
					if invite.UserId != "" {
						webPath = "login"
					}

					emailContent := map[string]interface{}{
						"PatientName": fullName,
						"Email":       invite.Email,
						"WebPath":     webPath,
					}

					if a.createAndSendNotification(req, invite, emailContent, inviteeLanguage) {
						a.logAudit(req, "invite sent")
					} else {
						a.logAudit(req, "invite failed to be sent")
						log.Print("Something happened generating an invite email")
						res.WriteHeader(http.StatusUnprocessableEntity)
						return
					}
				}

				a.sendModelAsResWithStatus(res, invite, http.StatusOK)
				return
			}
		}

	}
	return
}

// @Summary Send invitation to a hcp joining a medical team
// @Description  create a notification for the invitee and send him an email with the invitation
// @ID hydrophone-api-SendTeamInvite
// @Accept  json
// @Produce  json
// @Success 200 {object} models.Confirmation "invite details"
// @Failure 400 {object} status.Status "userId, teamId and isAdmin were not provided or the payload is missing/malformed"
// @Failure 401 {object} status.Status "Authorization token is missing or does not provide sufficient privileges"
// @Failure 403 {object} status.Status "Authorization token is invalid"
// @Failure 409 {object} status.Status "user already has a pending or declined invite OR user is already part of the team"
// @Failure 422 {object} status.Status "Error when sending the email (probably caused by the mailling service"
// @Failure 500 {object} status.Status "Internal error while processing the invite, detailled error returned in the body"
// @Router /confirm/send/team-invite [post]
// @security TidepoolAuth
func (a *Api) SendTeamInvite(res http.ResponseWriter, req *http.Request, vars map[string]string) {
	// By default, the invitee language will be "en" for Englih (as we don't know which language suits him)
	// In case the invitee is a known user, the language will be overriden in a later step
	var inviteeLanguage = "en"
	if token := a.token(res, req); token != nil {

		invitorID := token.UserID
		if invitorID == "" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		defer req.Body.Close()
		var ib = &inviteBody{}
		if err := json.NewDecoder(req.Body).Decode(ib); err != nil {
			log.Printf("SendInvite: error decoding invite to detail %v\n", err)
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_DECODING_INVITE)}
			a.sendModelAsResWithStatus(res, statusErr, statusErr.Code)
			return
		}

		if ib.Email == "" || ib.TeamID == "" || ib.IsAdmin == "" {
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_MISSING_DATA_INVITE)}
			a.sendModelAsResWithStatus(res, statusErr, statusErr.Code)
			return
		}

		if auth, err := a.tokenUserIsTeamAdmin(token, ib.TeamID); !auth || err != nil {
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusUnauthorized, STATUS_NOT_ADMIN)}
			a.sendModelAsResWithStatus(res, statusErr, statusErr.Code)
			return
		}

		// TODO get the team Name from teams service
		teamName := "Name of the team"
		teamAddress := "Address of the team"
		teamIdentification := "Shared ID of the team"

		var sessionToken = req.Header.Get(TP_SESSION_TOKEN)
		// check duplicate invite and if user is already a member
		if existingInvite, invitedUsr := a.checkForDuplicateTeamInvite(req.Context(), ib.Email, invitorID, ib.TeamID, sessionToken, models.TypeMedicalTeamInvite, res); existingInvite == true {
			return
		} else {
			//None exist so lets create the invite
			invite, _ := models.NewConfirmationWithContext(
				models.TypeMedicalTeamInvite,
				models.TemplateNameMedicalteamInvite,
				invitorID,
				ib.Permissions)

			// if the invitee is already a user, we can use his preferences
			invite.TeamID = ib.TeamID
			invite.Email = ib.Email
			invite.IsAdmin = ib.IsAdmin
			if invitedUsr != nil {
				invite.UserId = invitedUsr.UserID
				inviteeLanguage = a.getUserLanguage(invite.UserId, res)
			}

			if a.addOrUpdateConfirmation(req.Context(), invite, res) {
				a.logAudit(req, "invite created")

				if err := a.addProfile(invite); err != nil {
					log.Println("SendInvite: ", err.Error())
				} else {

					fullName := invite.Creator.Profile.FullName

					// TODO
					// validate the web path
					var webPath = "hcp/signup"

					// if invitee is already a user (ie already has an account), he won't go to signup but login instead
					if invite.UserId != "" {
						webPath = ""
					}

					emailContent := map[string]interface{}{
						"MedicalteamName":          teamName,
						"MedicalteamAddress":       teamAddress,
						"MedicalteamIentification": teamIdentification,
						"CreatorName":              fullName,
						"Email":                    invite.Email,
						"WebPath":                  webPath,
					}

					if a.createAndSendNotification(req, invite, emailContent, inviteeLanguage) {
						a.logAudit(req, "invite sent")
					} else {
						a.logAudit(req, "invite failed to be sent")
						log.Print("Something happened generating an invite email")
						res.WriteHeader(http.StatusUnprocessableEntity)
						return
					}
				}

				a.sendModelAsResWithStatus(res, invite, http.StatusOK)
				return
			}
		}

	}
	return
}

// @Summary Send notification to an hcp that becomes admin
// @Description  Send an email to th enew admin
// @ID hydrophone-api-UpdateTeamInvite
// @Accept  json
// @Produce  json
// @Success 200 {object} models.Confirmation "invite details"
// @Failure 400 {object} status.Status "userId, teamId and isAdmin were not provided or the payload is missing/malformed"
// @Failure 401 {object} status.Status "Authorization token is missing or does not provide sufficient privileges"
// @Failure 403 {object} status.Status "Authorization token is invalid"
// @Failure 409 {object} status.Status "No notification and email sent; User is already an admin"
// @Failure 422 {object} status.Status "Error when sending the email (probably caused by the mailling service"
// @Failure 500 {object} status.Status "Internal error while processing the invite, detailled error returned in the body"// @Router /confirm/send/team-invite [put]
// @security TidepoolAuth
func (a *Api) UpdateTeamInvite(res http.ResponseWriter, req *http.Request, vars map[string]string) {
	// By default, the invitee language will be "en" for Englih (as we don't know which language suits him)
	// In case the invitee is a known user, the language will be overriden in a later step
	var inviteeLanguage = "en"
	if token := a.token(res, req); token != nil {

		invitorID := token.UserID
		if invitorID == "" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		defer req.Body.Close()
		var ib = &inviteBody{}
		if err := json.NewDecoder(req.Body).Decode(ib); err != nil {
			log.Printf("UpdateInvite: error decoding invite to detail %v\n", err)
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_DECODING_INVITE)}
			a.sendModelAsResWithStatus(res, statusErr, statusErr.Code)
			return
		}

		if ib.Email == "" || ib.TeamID == "" || ib.IsAdmin == "" {
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_MISSING_DATA_INVITE)}
			a.sendModelAsResWithStatus(res, statusErr, statusErr.Code)
			return
		}

		// TODO get the team Name from teams service
		teamName := "Name of the team"

		// TODO validate the token.userid is admin of the given team
		if auth, err := a.tokenUserIsTeamAdmin(token, ib.TeamID); !auth || err != nil {
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusUnauthorized, STATUS_NOT_ADMIN)}
			a.sendModelAsResWithStatus(res, statusErr, statusErr.Code)
			return
		}

		// var sessionToken = req.Header.Get(TP_SESSION_TOKEN)
		var isAdmin = strings.ToLower(ib.IsAdmin) == "true"
		if isAdmin {
			invitedUsr := a.findExistingUser(ib.Email, a.sl.TokenProvide())
			if invitedUsr != nil && invitedUsr.UserID != "" {
				// make teams call
				if admin, err := a.isTeamAdmin(invitedUsr.UserID, ib.TeamID); admin == true || err != nil {
					statusErr := &status.StatusError{Status: status.NewStatus(http.StatusConflict, STATUS_ALREADY_ADMIN)}
					a.sendModelAsResWithStatus(res, statusErr, statusErr.Code)
					return
				}
			}
			invite, _ := models.NewConfirmationWithContext(
				models.TypeMedicalTeamDoAdmin,
				models.TemplateNameMedicalteamDoAdmin,
				invitorID,
				ib.Permissions)

			// if the invitee is already a user, we can use his preferences
			invite.TeamID = ib.TeamID
			invite.Email = ib.Email
			invite.IsAdmin = ib.IsAdmin
			invite.Status = models.StatusCompleted
			if invitedUsr != nil {
				invite.UserId = invitedUsr.UserID
				// does the invitee have a preferred language?
				inviteeLanguage = a.getUserLanguage(invite.UserId, res)
			}

			if a.addOrUpdateConfirmation(req.Context(), invite, res) {
				a.logAudit(req, "invite created")

				if err := a.addProfile(invite); err != nil {
					log.Println("SendInvite: ", err.Error())
				} else {

					emailContent := map[string]interface{}{
						"MedicalteamName": teamName,
						"Email":           invite.Email,
						"Language":        inviteeLanguage,
					}

					if a.createAndSendNotification(req, invite, emailContent, inviteeLanguage) {
						a.logAudit(req, "invite sent")
					} else {
						a.logAudit(req, "invite failed to be sent")
						log.Print("Something happened generating an invite email")
						res.WriteHeader(http.StatusUnprocessableEntity)
						return
					}
				}
				a.sendModelAsResWithStatus(res, invite, http.StatusOK)
				return
			}
		}
	}
	return
}

// @Summary Delete hcp from a medical team
// @Description create a notification for the hcp and send him an email to inform him he has been removed from the medical team
// @ID hydrophone-api-DeleteTeamMember
// @Accept  json
// @Produce  json
// @Success 200 {object} models.Confirmation "delete member"
// @Failure 400 {object} status.Status "userId, teamId and isAdmin were not provided or the payload is missing/malformed"
// @Failure 401 {object} status.Status "Authorization token is missing or does not provide sufficient privileges"
// @Failure 403 {object} status.Status "Authorization token is invalid"
// @Failure 409 {object} status.Status "No notification and email sent; User is not a member"
// @Failure 422 {object} status.Status "Error when sending the email (probably caused by the mailling service"
// @Failure 500 {object} status.Status "Internal error while processing the invite, detailled error returned in the body"
// @Router /confirm/send/team-invite [delete]
// @security TidepoolAuth
func (a *Api) DeleteTeamMember(res http.ResponseWriter, req *http.Request, vars map[string]string) {
	// By default, the invitee language will be "en" for Englih (as we don't know which language suits him)
	// In case the invitee is a known user, the language will be overriden in a later step
	var inviteeLanguage = "en"
	if token := a.token(res, req); token != nil {

		invitorID := token.UserID
		if invitorID == "" {
			res.WriteHeader(http.StatusBadRequest)
			return
		}

		defer req.Body.Close()
		var ib = &inviteBody{}
		if err := json.NewDecoder(req.Body).Decode(ib); err != nil {
			log.Printf("UpdateInvite: error decoding invite to detail %v\n", err)
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_DECODING_INVITE)}
			a.sendModelAsResWithStatus(res, statusErr, statusErr.Code)
			return
		}

		if ib.Email == "" || ib.TeamID == "" || ib.IsAdmin == "" {
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusBadRequest, STATUS_ERR_MISSING_DATA_INVITE)}
			a.sendModelAsResWithStatus(res, statusErr, statusErr.Code)
			return
		}

		// TODO get the team Name from teams service
		teamName := "Name of the team"

		// TODO validate the token.userid is admin of the given team
		if auth, err := a.tokenUserIsTeamAdmin(token, ib.TeamID); !auth || err != nil {
			statusErr := &status.StatusError{Status: status.NewStatus(http.StatusUnauthorized, STATUS_NOT_ADMIN)}
			a.sendModelAsResWithStatus(res, statusErr, statusErr.Code)
			return
		}

		invitedUsr := a.findExistingUser(ib.Email, a.sl.TokenProvide())
		if invitedUsr != nil && invitedUsr.UserID != "" {
			// make teams call
			if member, err := a.isTeamMember(invitedUsr.UserID, ib.TeamID); !member || err != nil {
				statusErr := &status.StatusError{Status: status.NewStatus(http.StatusConflict, STATUS_NOT_MEMBER)}
				a.sendModelAsResWithStatus(res, statusErr, statusErr.Code)
				return
			}
		}
		invite, _ := models.NewConfirmationWithContext(
			models.TypeMedicalTeamRemove,
			models.TemplateNameMedicalteamRemove,
			invitorID,
			ib.Permissions)

		// let's use the user preferences
		invite.TeamID = ib.TeamID
		invite.Email = ib.Email
		invite.IsAdmin = ib.IsAdmin
		if invitedUsr != nil {
			invite.UserId = invitedUsr.UserID
			// does the invitee have a preferred language?
			inviteeLanguage = a.getUserLanguage(invite.UserId, res)
		}

		if a.addOrUpdateConfirmation(req.Context(), invite, res) {
			a.logAudit(req, "invite created")

			if err := a.addProfile(invite); err != nil {
				log.Println("SendInvite: ", err.Error())
			} else {

				emailContent := map[string]interface{}{
					"MedicalteamName": teamName,
					"Email":           invite.Email,
					"Language":        inviteeLanguage,
				}

				if a.createAndSendNotification(req, invite, emailContent, inviteeLanguage) {
					a.logAudit(req, "invite sent")
				} else {
					a.logAudit(req, "invite failed to be sent")
					log.Print("Something happened generating an invite email")
					res.WriteHeader(http.StatusUnprocessableEntity)
					return
				}
			}

			a.sendModelAsResWithStatus(res, invite, http.StatusOK)
			return
		}

		return
	}
}
