package models

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

type (
	Confirmation struct {
		Key       string          `json:"key" bson:"_id"`
		Type      Type            `json:"type" bson:"type"`
		Email     string          `json:"email" bson:"email"`
		CreatorId string          `json:"creatorId" bson:"creatorId"`
		Creator   Creator         `json:"creator" bson:"creator"`
		Context   json.RawMessage `json:"context" bson:"context,omitempty" swaggertype:"string" format:"base64"`
		Created   time.Time       `json:"created" bson:"created"`

		TemplateName  TemplateName `json:"-" bson:"templateName"`
		UserId        string       `json:"userId" bson:"userId"`
		Status        Status       `json:"status" bson:"status"`
		Modified      time.Time    `json:"-" bson:"modified"`
		ShortKey      string       `json:"shortKey" bson:"shortKey"`
		ResendCounter int64        `bson:"resendCounter"`
	}

	//basic details for the creator of the confirmation
	Creator struct {
		*Profile `json:"profile" bson:"-"`
		UserId   string `json:"userid" bson:"-"` //for compatability with blip
	}
	Patient struct {
		Birthday      string `json:"birthday"`
		DiagnosisDate string `json:"diagnosisDate"`
		IsOtherPerson bool   `json:"isOtherPerson"`
		FullName      string `json:"fullName"`
	}
	Preferences struct {
		DisplayLanguage string `json:"displayLanguageCode"`
	}
	Profile struct {
		FullName string  `json:"fullName"`
		Patient  Patient `json:"patient"`
	}

	//Enum type's
	Status string
	Type   string

	ConfirmationDuration struct {
		Unit  string
		Value time.Duration
	}
	TypeDurations map[Type]ConfirmationDuration
)

const (
	//Available Status's
	StatusPending   Status = "pending"
	StatusCompleted Status = "completed"
	//Available Type's
	TypePasswordReset        Type = "password_reset"
	TypePatientPasswordReset Type = "patient_password_reset"
	TypePatientPasswordInfo  Type = "patient_password_info"
	TypeSignUp               Type = "signup_confirmation"
	TypeNoAccount            Type = "no_account"
	TypeInformation          Type = "patient_information"
	TypePatientPinReset      Type = "patient_pin_reset"
	shortKeyLength                = 8
	letterBytes                   = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var (
	Timeouts TypeDurations = TypeDurations{
		TypePasswordReset:        ConfirmationDuration{"day", 7 * 24 * time.Hour},
		TypeSignUp:               ConfirmationDuration{"day", 31 * 24 * time.Hour},
		TypePatientPasswordReset: ConfirmationDuration{"hour", 1 * time.Hour},
	}
)

// New confirmation with just the basics
func NewConfirmation(theType Type, templateName TemplateName, creatorId string) (*Confirmation, error) {

	shortKey := ""
	status := StatusPending
	var err error = nil

	switch theType {
	case TypePatientPasswordReset:
		if shortKey, err = generateShortKey(shortKeyLength); err != nil {
			return nil, err
		}
	case TypePatientPasswordInfo:
		status = StatusCompleted
	default:
	}
	if key, err := generateKey(); err != nil {
		return nil, err
	} else {

		conf := &Confirmation{
			Key:          key,
			Type:         theType,
			TemplateName: templateName,
			CreatorId:    creatorId,
			Creator:      Creator{}, //set before sending back to client
			Status:       status,
			Created:      time.Now(),
			ShortKey:     shortKey,
		}

		return conf, nil
	}
}

// New confirmation that includes context data
func NewConfirmationWithContext(theType Type, templateName TemplateName, creatorId string, data interface{}) (*Confirmation, error) {
	if conf, err := NewConfirmation(theType, templateName, creatorId); err != nil {
		return nil, err
	} else {
		conf.AddContext(data)
		return conf, nil
	}
}

// Add context data
func (c *Confirmation) AddContext(data interface{}) {

	jsonData, _ := json.Marshal(data)
	c.Context = jsonData
}

// Set a new status and update the modified time
func (c *Confirmation) UpdateStatus(newStatus Status) {
	c.Status = newStatus
	c.Modified = time.Now()
}

func (c *Confirmation) IsExpired() bool {
	timeoutObj, ok := Timeouts[c.Type]
	if !ok {
		log.Printf("[%s] does not exist", c.Type)
		return false
	}
	return time.Now().After(c.Created.Add(timeoutObj.Value))
}

func (c *Confirmation) GetReadableDuration() string {
	timeoutObj, ok := c.getDuration()
	var value float64
	if ok {
		value = (timeoutObj.Value).Hours()
		if timeoutObj.Unit == "day" {
			value = value / 24
		}
	}
	return fmt.Sprintf("%.0f", value)
}

func GenerateRandomBytes(length int) ([]byte, error) {
	rb := make([]byte, length)
	if _, err := rand.Read(rb); err != nil {
		log.Println(err)
		return nil, err
	} else {
		return rb, nil
	}
}

func generateKey() (string, error) {
	length := 24 // change the length of the generated random string here
	rb, err := GenerateRandomBytes(length)
	return base64.URLEncoding.EncodeToString(rb), err
}

func generateShortKey(length int) (string, error) {
	bytes, err := GenerateRandomBytes(length)
	if err != nil {
		return "", err
	}
	for i, b := range bytes {
		bytes[i] = letterBytes[b%byte(len(letterBytes))]
	}
	return string(bytes), nil
}

func (c *Confirmation) getDuration() (ConfirmationDuration, bool) {
	var timeoutObj ConfirmationDuration
	var ok = false
	timeoutObj, ok = Timeouts[c.Type]
	if !ok {
		log.Printf("[%s] does not exist", c.Type)
		return timeoutObj, false
	}
	return timeoutObj, true
}
