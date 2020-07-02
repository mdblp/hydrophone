package api

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	otp "github.com/tidepool-org/hydrophone/utils/otp"

	"github.com/tidepool-org/go-common/clients/portal"
	"github.com/tidepool-org/go-common/clients/shoreline"
	"github.com/tidepool-org/go-common/clients/status"
	"github.com/tidepool-org/hydrophone/models"
)

const (
	statusPinResetNoID     = "Required userid is missing"
	statusPinResetErr      = "Error sending PIN Reset"
	statusPinResetNoServer = "This API cannot be requested with server token"
	statusUserDoesNotExist = "This user does not exist"
	timeStep               = 1800 // time interval for the OTP = 30 minutes
	digits                 = 9    // nb digits for the OTP
	startTime              = 0    // start time for the OTP = EPOCH
)

// SendPinReset handles the pin reset http route
// @Summary Send an OTP for PIN Reset to a patient
// @Description  It sends an email that contains a time-based One-time password for PIN Reset
// @ID hydrophone-api-sendPinReset
// @Accept  json
// @Produce  json
// @Param userid path string true "user id"
// @Success 200 {string} string "OK"
// @Failure 400 {object} status.Status "userId was not provided"
// @Failure 403 {object} status.Status "only authorized for patients, not clinicians nor server token"
// @Failure 422 {object} status.Status "Error when sending the email (probably caused by the mailing service)"
// @Failure 500 {object} status.Status "Internal error while processing the request, detailed error returned in the body"
// @Router /send/pin_reset/{userid} [post]
func (a *Api) SendPinReset(res http.ResponseWriter, req *http.Request, vars map[string]string) {

	// by default, language is EN. It will be overriden if prefered language is found later
	var userLanguage = "en"
	var newOTP *models.Confirmation
	var usrDetails *shoreline.UserData
	var err error

	var token = req.Header.Get(TP_SESSION_TOKEN)

	td := a.sl.CheckToken(token)
	if td == nil || td.IsServer {
		a.sendError(res, http.StatusForbidden, statusPinResetNoServer)
		return
	}

	userID := vars["userid"]
	if userID == "" {
		log.Printf("sendPinReset - %s", statusPinResetNoID)
		a.sendModelAsResWithStatus(res, status.NewStatus(http.StatusBadRequest, statusPinResetNoID), http.StatusBadRequest)
		return
	}

	if usrDetails, err = a.sl.GetUser(userID, a.sl.TokenProvide()); err != nil {
		log.Printf("sendPinReset - %s err[%s]", STATUS_ERR_FINDING_USR, err.Error())
		a.sendModelAsResWithStatus(res, STATUS_ERR_FINDING_USR, http.StatusInternalServerError)
		return
	}

	if usrDetails == nil {
		log.Printf("sendPinReset - %s err[%s]", statusUserDoesNotExist, userID)
		a.sendModelAsResWithStatus(res, statusUserDoesNotExist, http.StatusBadRequest)
		return
	}

	if usrDetails.IsClinic() {
		log.Printf("sendPinReset - Clinician account [%s] cannot receive PIN Reset message", usrDetails.UserID)
		a.sendModelAsResWithStatus(res, STATUS_ERR_CLINICAL_USR, http.StatusForbidden)
		return
	}

	// send PIN Reset OTP to patient
	// the secret for the TOTP is the concatenation of userID + IMEI + userID
	// first get the IMEI of the patient's handset
	var patientConfig *portal.PatientConfig

	if patientConfig, err = a.portal.GetPatientConfig(token); err != nil {
		a.sendError(res, http.StatusInternalServerError, statusPinResetErr, "error getting patient config: ", err.Error())
		return
	}

	if patientConfig.Device.IMEI == "" {
		a.sendError(res, http.StatusInternalServerError, statusPinResetErr, "error getting patient config")
		return
	}

	// Prepare TOTP generator
	// here we want TOTP with time step of 30 minutes
	// So it will be valid between 1 second and 30 minutes
	// We assume the validator (i.e. the DBLG handset will check validation against the last 2 time steps)
	// So the patient will have, in reality, between 30 minutes and 1 hour to receive the TOTP and enter it in the handset
	var gen = otp.TOTPGenerator{
		TimeStep:  timeStep,
		StartTime: startTime,
		Secret:    userID + patientConfig.Device.IMEI + userID,
		Digit:     digits,
	}

	var totp otp.TOTP = gen.Now()
	var re = regexp.MustCompile(`^(...)(...)(...)$`)

	// let's get the user preferences
	userPreferences := &models.Preferences{}
	if err := a.seagull.GetCollection(userID, "preferences", a.sl.TokenProvide(), userPreferences); err != nil {
		a.sendError(res, http.StatusInternalServerError, statusPinResetErr, "error getting user preferences: ", err.Error())
		return
	}

	// does the user have a preferred language?
	if userPreferences.DisplayLanguage != "" {
		userLanguage = userPreferences.DisplayLanguage
	}

	var templateName = models.TemplateNamePatientPinReset

	// Support address configuration contains the mailto we want to strip out
	suppAddr := fmt.Sprintf("<a href=%s>%s</a>", a.Config.SupportURL, strings.Replace(a.Config.SupportURL, "mailto:", "", 1))

	emailContent := map[string]interface{}{
		"Email":        usrDetails.Emails[0],
		"OTP":          re.ReplaceAllString(totp.OTP, `$1-$2-$3`),
		"SupportEmail": suppAddr,
	}

	// Create new confirmation with context data = totp
	newOTP, _ = models.NewConfirmationWithContext(models.TypePinReset, templateName, usrDetails.UserID, totp)
	newOTP.Email = usrDetails.Emails[0]

	// Save confirmation in DB
	if a.addOrUpdateConfirmation(newOTP, res) {

		if a.createAndSendNotification(req, newOTP, emailContent, userLanguage) {
			log.Printf("sendPinReset - OTP sent for %s", userID)
			a.logAudit(req, "pin reset OTP sent")
			res.WriteHeader(http.StatusOK)
			res.Write([]byte("OK"))
		} else {
			log.Print("sendPinReset - Something happened generating a Pin Reset email")
			res.WriteHeader(http.StatusUnprocessableEntity)
		}
	}
}
