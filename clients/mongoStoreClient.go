package clients

import (
	"./../models"
	"github.com/tidepool-org/go-common/clients/mongo"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"log"
)

const (
	CONFIRMATIONS_COLLECTION = "confirmations"
)

type MongoStoreClient struct {
	session        *mgo.Session
	confirmationsC *mgo.Collection
}

func NewMongoStoreClient(config *mongo.Config) *MongoStoreClient {

	mongoSession, err := mongo.Connect(config)
	if err != nil {
		log.Fatal(err)
	}

	return &MongoStoreClient{
		session:        mongoSession,
		confirmationsC: mongoSession.DB("").C(CONFIRMATIONS_COLLECTION),
	}
}

func (d MongoStoreClient) Close() {
	log.Println("Close the session")
	d.session.Close()
	return
}

func (d MongoStoreClient) Ping() error {
	// do we have a store session
	if err := d.session.Ping(); err != nil {
		return err
	}
	return nil
}

func (d MongoStoreClient) UpsertConfirmation(confirmation *models.Confirmation) error {

	// if the user already exists we update otherwise we add
	if _, err := d.confirmationsC.Upsert(bson.M{"key": confirmation.Key}, confirmation); err != nil {
		return err
	}
	return nil
}

func (d MongoStoreClient) FindConfirmation(confirmation *models.Confirmation) (result *models.Confirmation, err error) {

	if confirmation.Key != "" {
		if err = d.confirmationsC.Find(bson.M{"key": confirmation.Key}).One(&result); err != nil {
			return result, err
		}
	}

	return result, nil
}

func (d MongoStoreClient) FindConfirmations(userId, creatorId string, status models.Status) (results []*models.Confirmation, err error) {

	fieldsToMatch := []bson.M{}

	if userId != "" {
		fieldsToMatch = append(fieldsToMatch, bson.M{"userid": userId})
	}
	if creatorId != "" {
		fieldsToMatch = append(fieldsToMatch, bson.M{"username": creatorId})
	}
	if string(status) != "" {
		fieldsToMatch = append(fieldsToMatch, bson.M{"emails": status})
	}

	if err = d.confirmationsC.Find(bson.M{"$or": fieldsToMatch}).All(&results); err != nil {
		return results, err
	}
	return results, nil
}

func (d MongoStoreClient) RemoveConfirmation(confirmation *models.Confirmation) error {
	if err := d.confirmationsC.Remove(bson.M{"key": confirmation.Key}); err != nil {
		return err
	}
	return nil
}
