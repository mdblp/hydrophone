package clients

import (
	"context"
	"os"
	"testing"
	"time"

	goComMgo "github.com/mdblp/go-db/v3/mongo"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/mdblp/hydrophone/v2/models"
)

var logger, _ = test.NewNullLogger()
var testingConfig = &goComMgo.Config{
	Database:               "confirm_test",
	Timeout:                2 * time.Second,
	WaitConnectionInterval: 5 * time.Second,
	MaxConnectionAttempts:  0,
}

func TestMongoStoreConfirmationOperations(t *testing.T) {
	if _, exist := os.LookupEnv("BLP_STORE_ADDRESSES"); exist {
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

}

func TestMongoStoreCountConfirmations(t *testing.T) {
	if _, exist := os.LookupEnv("BLP_STORE_ADDRESSES"); exist {
		// if mongo connexion information is provided via env var
		testingConfig.FromEnv()
	}

	mc, _ := NewStore(testingConfig, logger)
	mc.Start()
	mc.WaitUntilStarted()
	/*
	 * INIT THE TEST - we use a clean copy of the collection before we start
	 */

	mgoConfirmationsCollection(mc).Drop(context.TODO())
	ctx := context.Background()
	start := time.Now()
	countTests(ctx, t, mc, "creatorId", "", "")
	countTests(ctx, t, mc, "", "userId", "")
	countTests(ctx, t, mc, "", "", "test@test.com")

	// Testing error raising when creatorId & userId & email not set
	count, err := mc.CountLatestConfirmations(ctx, models.Confirmation{}, start)
	if err == nil {
		t.Fatal("Not passing creatorId or userId or Email should raise an error")
	}
	if count != 0 {
		t.Fatalf("Expected 0 confirmations got: %v", count)
	}
}

func countTests(ctx context.Context, t *testing.T, mongoClient *Client, creatorId string, userId string, email string) {
	now := time.Now()
	time.Sleep(time.Second)
	confirmation := prepareCountTest(ctx, t, mongoClient, models.TypePasswordReset, models.TemplateNamePasswordReset, creatorId, userId, email)
	countLatestConfirmationsTest(ctx, t, mongoClient, *confirmation, now, 1)

	// Testing second insertion (and different creation time)
	secondInsert := time.Now()
	time.Sleep(time.Second)
	prepareCountTest(ctx, t, mongoClient, models.TypePasswordReset, models.TemplateNamePasswordReset, creatorId, userId, email)
	countLatestConfirmationsTest(ctx, t, mongoClient, *confirmation, now, 2)
	countLatestConfirmationsTest(ctx, t, mongoClient, *confirmation, secondInsert, 1)

	// Testing different creatorId
	thirdInsert := time.Now()
	time.Sleep(time.Second)
	var (
		otherCreatorId string
		otherUserId    string
		otherEmail     string
	)
	if creatorId != "" {
		otherCreatorId = creatorId + ".1"
	}
	if userId != "" {
		otherUserId = userId + ".1"
	}
	if email != "" {
		otherEmail = email + ".1"
	}
	confirmation3 := prepareCountTest(ctx, t, mongoClient, models.TypePasswordReset, models.TemplateNamePasswordReset, otherCreatorId, otherUserId, otherEmail)
	countLatestConfirmationsTest(ctx, t, mongoClient, *confirmation, now, 2)
	countLatestConfirmationsTest(ctx, t, mongoClient, *confirmation3, thirdInsert, 1)

	// Testing different confirmation types
	confirmation4 := prepareCountTest(ctx, t, mongoClient, models.TypePatientPasswordReset, models.TemplateNamePatientPasswordReset, otherCreatorId, otherUserId, otherEmail)
	countLatestConfirmationsTest(ctx, t, mongoClient, *confirmation3, thirdInsert, 1)
	countLatestConfirmationsTest(ctx, t, mongoClient, *confirmation4, thirdInsert, 1)
}

func countLatestConfirmationsTest(ctx context.Context, t *testing.T, mongoClient *Client, confirm models.Confirmation, timeToCount time.Time, expectedCount int64) {
	count, err := mongoClient.CountLatestConfirmations(ctx, confirm, timeToCount)
	if err != nil {
		t.Fatalf("we could not count confirmations - err [%v]", err)
	}
	if count != expectedCount {
		t.Fatalf("Expected %v confirmations got: %v", expectedCount, count)
	}
}

func prepareCountTest(ctx context.Context, t *testing.T, mongoClient *Client, modelType models.Type, templateName models.TemplateName, creatorId string, userId string, email string) *models.Confirmation {
	confirmation, _ := models.NewConfirmation(modelType, templateName, creatorId)
	confirmation.UserId = userId
	confirmation.Email = email
	if err := mongoClient.UpsertConfirmation(ctx, confirmation); err != nil {
		t.Fatalf("we could not save the confirmation - err [%v]", err)
	}
	return confirmation
}
