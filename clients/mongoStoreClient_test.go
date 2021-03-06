package clients

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/mdblp/hydrophone/models"
	goComMgo "github.com/tidepool-org/go-common/clients/mongo"
)

var logger = log.New(os.Stdout, "mongo-test ", log.LstdFlags|log.LUTC|log.Lshortfile)

var testingConfig = &goComMgo.Config{
	Database:               "confirm_test",
	Timeout:                2 * time.Second,
	WaitConnectionInterval: 5 * time.Second,
	MaxConnectionAttempts:  0,
}

func TestMongoStoreConfirmationOperations(t *testing.T) {
	if _, exist := os.LookupEnv("TIDEPOOL_STORE_ADDRESSES"); exist {
		// if mongo connexion information is provided via env var
		testingConfig.FromEnv()
	}

	confirmation, _ := models.NewConfirmation(models.TypePasswordReset, models.TemplateNamePasswordReset, "123.456")
	confirmation.Email = "test@test.com"

	doesNotExist, _ := models.NewConfirmation(models.TypePasswordReset, models.TemplateNamePasswordReset, "123.456")
	mc, _ := NewStore(testingConfig, logger)
	mc.Start()
	mc.WaitUntilStarted()
	/*
	 * INIT THE TEST - we use a clean copy of the collection before we start
	 */

	mgoConfirmationsCollection(mc).Drop(context.TODO())
	ctx := context.Background()
	//The basics
	//+++++++++++++++++++++++++++
	if err := mc.UpsertConfirmation(ctx, confirmation); err != nil {
		t.Fatalf("we could not save the con - err [%v]", err)
	}

	if found, err := mc.FindConfirmation(ctx, confirmation); err == nil {
		if found == nil {
			t.Fatalf("the confirmation was not found")
		}
		if found.Key == "" {
			t.Fatalf("the confirmation string isn't included - err [%v]", found)
		}
	} else {
		t.Fatalf("no confirmation was returned when it should have been - err[%v]", err)
	}

	// Uppercase the email and try again (detect case sensitivity)
	confirmation.Email = "TEST@TEST.COM"
	if found, err := mc.FindConfirmation(ctx, confirmation); err == nil {
		if found == nil {
			t.Fatalf("the uppercase confirmation was not found")
		}
		if found.Key == "" {
			t.Fatalf("the confirmation string isn't included %v", found)
		}
	} else {
		t.Fatalf("no confirmation was returned when it should have been - err[%v]", err)
	}

	//when the conf doesn't exist
	if found, err := mc.FindConfirmation(ctx, doesNotExist); err == nil && found != nil {
		t.Fatalf("there should have been no confirmation found [%v]", found)
	} else if err != nil {
		t.Fatalf("and error was returned when it should not have been - err [%v]", err)
	}

	if err := mc.RemoveConfirmation(ctx, confirmation); err != nil {
		t.Fatalf("we could not remove the confirmation - err [%v]", err)
	}

	if confirmation, err := mc.FindConfirmation(ctx, confirmation); err == nil {
		if confirmation != nil {
			t.Fatalf("the confirmation has been removed so we shouldn't find it %v", confirmation)
		}
	}

	//Find with other statuses
	const fromUser, toUser, toEmail, toOtherEmail = "999.111", "312.123", "some@email.org", "some@other.org"
	c1, _ := models.NewConfirmation(models.TypeCareteamInvite, models.TemplateNameCareteamInvite, fromUser)
	c1.UserId = toUser
	c1.Email = toEmail
	c1.UpdateStatus(models.StatusDeclined)
	mc.UpsertConfirmation(ctx, c1)

	// Sleep some so the second confirmation created time is after the first confirmation created time
	time.Sleep(time.Second)

	c2, _ := models.NewConfirmation(models.TypeCareteamInvite, models.TemplateNameCareteamInvite, fromUser)
	c2.Email = toOtherEmail
	c2.UpdateStatus(models.StatusCompleted)
	mc.UpsertConfirmation(ctx, c2)

	searchForm := &models.Confirmation{CreatorId: fromUser}
	searchStatus := []models.Status{models.StatusDeclined, models.StatusCompleted}
	searchTypes := []models.Type{}

	if confirmations, err := mc.FindConfirmations(ctx, searchForm, searchStatus, searchTypes); err == nil {
		if len(confirmations) != 2 {
			t.Fatalf("we should have found 2 confirmations %v", confirmations)
		}

		t1 := confirmations[0].Created
		t2 := confirmations[1].Created

		if !t1.After(t2) {
			t.Fatalf("the newest confirmtion should be first %v", confirmations)
		}

		if confirmations[0].Email != toOtherEmail {
			t.Fatalf("email invalid: %s", confirmations[0].Email)
		}
		if confirmations[0].Status != models.StatusCompleted && confirmations[0].Status != models.StatusDeclined {
			t.Fatalf("status invalid: %s", confirmations[0].Status)
		}
		if confirmations[1].Email != toEmail {
			t.Fatalf("email invalid: %s", confirmations[1].Email)
		}
		if confirmations[1].Status != models.StatusCompleted && confirmations[1].Status != models.StatusDeclined {
			t.Fatalf("status invalid: %s", confirmations[1].Status)
		}
	}
	searchToOtherEmail := &models.Confirmation{CreatorId: fromUser, Email: toOtherEmail}
	searchStatus = []models.Status{models.StatusDeclined, models.StatusCompleted}
	searchTypes = []models.Type{}
	//only email address
	if confirmations, err := mc.FindConfirmations(ctx, searchToOtherEmail, searchStatus, searchTypes); err == nil {
		if len(confirmations) != 1 {
			t.Fatalf("we should have found 1 confirmations %v", confirmations)
		}
		if confirmations[0].Email != toOtherEmail {
			t.Fatalf("should be for email: %s", toOtherEmail)
		}
		if confirmations[0].Status != models.StatusCompleted && confirmations[0].Status != models.StatusDeclined {
			t.Fatalf("status invalid: %s", confirmations[0].Status)
		}
	}
	searchToEmail := &models.Confirmation{CreatorId: fromUser, Email: toEmail}
	searchStatus = []models.Status{models.StatusDeclined, models.StatusCompleted}
	searchTypes = []models.Type{}
	//with both userid and email address
	if confirmations, err := mc.FindConfirmations(ctx, searchToEmail, searchStatus, searchTypes); err == nil {
		if len(confirmations) != 1 {
			t.Fatalf("we should have found 1 confirmations %v", confirmations)
		}
		if confirmations[0].UserId != toUser {
			t.Fatalf("should be for user: %s", toUser)
		}
		if confirmations[0].Email != toEmail {
			t.Fatalf("should be for email: %s", toEmail)
		}
		if confirmations[0].Status != models.StatusCompleted && confirmations[0].Status != models.StatusDeclined {
			t.Fatalf("status invalid: %s", confirmations[0].Status)
		}
	}
}
