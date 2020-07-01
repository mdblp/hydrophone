package models

import (
	"strings"
	"testing"
)

const USERID = "1234-555"

type Extras struct {
	Blah  string `json:"blah"`
	Email string `json:"email"`
}

var contextData = &Extras{Blah: "stuff", Email: "test@user.org"}

func contains(source string, compared string) bool {
	for _, char := range source {
		if !strings.Contains(compared, string(char)) {
			return false
		}
	}
	return true
}

func Test_NewConfirmation(t *testing.T) {

	confirmation, _ := NewConfirmation(TypePasswordReset, TemplateNamePasswordReset, USERID)

	if confirmation.Status != StatusPending {
		t.Fatalf("Status should be [%s] but is [%s]", StatusPending, confirmation.Status)
	}

	if confirmation.Key == "" {
		t.Fatal("There should be a generated key")
	}

	if confirmation.Created.IsZero() {
		t.Fatal("The created time should be set")
	}

	if confirmation.Modified.IsZero() == false {
		t.Fatal("The modified time should NOT be set")
	}

	if confirmation.Type != TypePasswordReset {
		t.Fatalf("The type should be [%s] but is [%s]", TypePasswordReset, confirmation.Type)
	}

	if confirmation.TemplateName != TemplateNamePasswordReset {
		t.Fatalf("The template type should be [%s] but is [%s]", TemplateNamePasswordReset, confirmation.TemplateName)
	}

	if confirmation.UserId != "" {
		t.Logf("expected '' actual [%s]", confirmation.UserId)
		t.Fail()
	}

	if confirmation.CreatorId != USERID {
		t.Logf("expected [%s] actual [%s]", USERID, confirmation.CreatorId)
		t.Fail()
	}

	if confirmation.Creator.Profile != nil {
		t.Logf("expected `nil` actual [%v]", confirmation.Creator.Profile)
		t.Fail()
	}

	if confirmation.Creator.UserId != "" {
		t.Logf("expected `` actual [%s]", confirmation.Creator.UserId)
		t.Fail()
	}

	confirmation.UpdateStatus(StatusCompleted)

	if confirmation.Status != StatusCompleted {
		t.Fatalf("Status should be [%s] but is [%s]", StatusCompleted, confirmation.Status)
	}

	if confirmation.Modified.IsZero() != false {
		t.Fatal("The modified time should have been set")
	}

}

func Test_NewPatientPasswordResetConfirmation(t *testing.T) {

	confirmation, _ := NewConfirmation(TypePatientPasswordReset, TemplateNamePatientPasswordReset, USERID)

	if confirmation.Status != StatusPending {
		t.Fatalf("Status should be [%s] but is [%s]", StatusPending, confirmation.Status)
	}

	if confirmation.Key == "" {
		t.Fatal("There should be a generated key")
	}

	if confirmation.ShortKey == "" {
		t.Fatal("There should be a generated short key")
	}

	if confirmation.Created.IsZero() {
		t.Fatal("The created time should be set")
	}

	if confirmation.Modified.IsZero() == false {
		t.Fatal("The modified time should NOT be set")
	}

	if confirmation.Type != TypePatientPasswordReset {
		t.Fatalf("The type should be [%s] but is [%s]", TypePatientPasswordReset, confirmation.Type)
	}

	if confirmation.TemplateName != TemplateNamePatientPasswordReset {
		t.Fatalf("The template type should be [%s] but is [%s]", TemplateNamePatientPasswordReset, confirmation.TemplateName)
	}

	if confirmation.UserId != "" {
		t.Logf("expected '' actual [%s]", confirmation.UserId)
		t.Fail()
	}

	if confirmation.CreatorId != USERID {
		t.Logf("expected [%s] actual [%s]", USERID, confirmation.CreatorId)
		t.Fail()
	}

	if confirmation.Creator.Profile != nil {
		t.Logf("expected `nil` actual [%v]", confirmation.Creator.Profile)
		t.Fail()
	}

	if confirmation.Creator.UserId != "" {
		t.Logf("expected `` actual [%s]", confirmation.Creator.UserId)
		t.Fail()
	}

	confirmation.UpdateStatus(StatusCompleted)

	if confirmation.Status != StatusCompleted {
		t.Fatalf("Status should be [%s] but is [%s]", StatusCompleted, confirmation.Status)
	}

	if confirmation.Modified.IsZero() != false {
		t.Fatal("The modified time should have been set")
	}

}

func Test_NewConfirmationWithContext(t *testing.T) {

	confirmation, _ := NewConfirmationWithContext(TypePasswordReset, TemplateNamePasswordReset, USERID, contextData)

	myExtras := &Extras{}

	confirmation.DecodeContext(&myExtras)

	if myExtras.Blah == "" {
		t.Fatalf("context not decoded [%v]", myExtras)
	}

	if myExtras.Email == "" {
		t.Fatalf("context not decoded [%v]", myExtras)
	}

	//and all tests should pass for a new confirmation too
	Test_NewConfirmation(t)
}

func Test_Confirmation_AddContext(t *testing.T) {

	confirmation, _ := NewConfirmation(TypePasswordReset, TemplateNamePasswordReset, USERID)

	confirmation.AddContext(contextData)

	myExtras := &Extras{}

	confirmation.DecodeContext(&myExtras)

	if myExtras.Blah == "" {
		t.Fatalf("context not decoded [%v]", myExtras)
	}

	if myExtras.Email == "" {
		t.Fatalf("context not decoded [%v]", myExtras)
	}

}

func TestConfirmationKey(t *testing.T) {

	key, _ := generateKey()

	if key == "" {
		t.Fatal("There should be a generated key")
	}

	if len(key) != 32 {
		t.Fatal("The generated key should be 32 chars: ", len(key))
	}
}

func TestConfirmationShortKey(t *testing.T) {

	keyLength := 8
	key := generateShortKey(keyLength)

	if key == "" {
		t.Fatal("There should be a generated key")
	}

	if len(key) != keyLength {
		t.Fatal("The generated key should be 8 chars: ", len(key))
	}

	if !contains(key, letterBytes) {
		t.Fatal("The key should only contain authorized characters")
	}

}
