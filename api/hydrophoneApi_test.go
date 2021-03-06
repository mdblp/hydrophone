package api

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gorilla/mux"

	crewClient "github.com/mdblp/crew/client"
	"github.com/mdblp/hydrophone/clients"
	"github.com/mdblp/hydrophone/localize"
	"github.com/mdblp/hydrophone/models"
	"github.com/mdblp/shoreline/clients/shoreline"
	"github.com/mdblp/shoreline/schema"
	"github.com/mdblp/shoreline/token"
	commonClients "github.com/tidepool-org/go-common/clients"
	"github.com/tidepool-org/go-common/clients/portal"
	"github.com/tidepool-org/go-common/clients/version"
)

const (
	make_store_fail           = true
	make_store_return_nothing = true

	testing_token = "a.fake.token.to.use.in.tests"

	testing_token_uid1 = "a.fake.token.for.uid.1"
	testing_uid1       = "UID123"

	testing_token_uid2 = "a.fake.token.for.uid.2"
	testing_uid2       = "UID999"

	testing_token_hcp       = "a.fake.token.for.hcp"
	testing_token_caregiver = "a.fake.token.for.caregiver"

	testing_uid3 = "UID002"
	testing_uid4 = "UID004"
)

var (
	NO_PARAMS = map[string]string{}

	FAKE_CONFIG = Config{
		ServerSecret:      "shhh! don't tell",
		I18nTemplatesPath: "../templates",
	}
	/*
	 * basics setup
	 */
	rtr           = mux.NewRouter()
	mockNotifier  = clients.NewMockNotifier()
	mockShoreline = shoreline.NewMock(testing_token)
	mockPerms     = crewClient.NewMock()

	mockSeagull = commonClients.NewSeagullMock()

	mockPortal = portal.NewMock()

	mockTemplates = models.Templates{}

	/*
	 * stores
	 */
	mockStore      = clients.NewMockStoreClient(false, false)
	mockStoreEmpty = clients.NewMockStoreClient(make_store_return_nothing, false)
	mockStoreFails = clients.NewMockStoreClient(false, make_store_fail)

	/*
	 * users permissons scenarios
	 */
	mock_uid1Shoreline = newtestingShorelingMock(testing_uid1)
	mock_uid2Shoreline = newtestingShorelingMock(testing_uid2)

	responsableHydrophone = InitApi(FAKE_CONFIG, mockStore, mockNotifier, mockShoreline, mockPerms, mockSeagull, mockPortal, mockTemplates)

	mockLocalizer, _ = localize.NewI18nLocalizer("../templates/locales")
)

// In an effort to mock shoreline so that we can return the token we wish
type testingShorelingMock struct {
	userid string
}

func newtestingShorelingMock(userid string) *testingShorelingMock {
	return &testingShorelingMock{userid: userid}
}

func (m *testingShorelingMock) Start() error { return nil }
func (m *testingShorelingMock) Close()       { return }
func (m *testingShorelingMock) Login(username, password string) (*schema.UserData, string, error) {
	return &schema.UserData{UserID: m.userid, Emails: []string{m.userid + "@email.org"}, Username: m.userid + "@email.org"}, "", nil
}
func (m *testingShorelingMock) Signup(username, password, email string) (*schema.UserData, error) {
	return &schema.UserData{UserID: m.userid, Emails: []string{m.userid + "@email.org"}, Username: m.userid + "@email.org"}, nil
}
func (m *testingShorelingMock) TokenProvide() string { return testing_token }
func (m *testingShorelingMock) GetUser(userID, token string) (*schema.UserData, error) {
	if userID == "me2@myemail.com" {
		return &schema.UserData{UserID: testing_uid3, Emails: []string{userID}, Username: userID}, nil
	}
	if userID == "patient.team@myemail.com" {
		return &schema.UserData{UserID: testing_uid4, Emails: []string{userID}, Username: userID, Roles: []string{"patient"}}, nil
	}
	if userID == "caregiver@myemail.com" {
		return &schema.UserData{UserID: testing_uid1, Emails: []string{userID}, Username: userID, Roles: []string{"caregiver"}}, nil
	}
	if userID == "hcpMember@myemail.com" {
		return &schema.UserData{UserID: testing_uid2, Emails: []string{userID}, Username: userID, Roles: []string{"hcp"}}, nil
	}
	if userID == "doesnotexist@myemail.com" {
		return nil, nil
	}

	return &schema.UserData{UserID: m.userid, Emails: []string{m.userid + "@email.org"}, Username: m.userid + "@email.org", Roles: []string{"hcp"}}, nil
}
func (m *testingShorelingMock) UpdateUser(userID string, userUpdate schema.UserUpdate, token string) error {
	return nil
}
func (m *testingShorelingMock) CheckToken(chkToken string) *token.TokenData {
	if chkToken == testing_token_hcp {
		return &token.TokenData{UserId: m.userid, IsServer: false, Role: "hcp"}
	}
	if chkToken == testing_token_caregiver {
		return &token.TokenData{UserId: m.userid, IsServer: false, Role: "caregiver"}
	}
	return &token.TokenData{UserId: m.userid, IsServer: false, Role: "patient"}
}

type (
	//common test structure
	toTest struct {
		desc          string
		skip          bool
		returnNone    bool
		doBad         bool
		method        string
		url           string
		body          testJSONObject
		token         string
		respCode      int
		response      testJSONObject
		emailSubject  string
		customHeaders map[string]string
	}
	// These two types make it easier to define blobs of json inline.
	// We don't use the types defined by the API because we want to
	// be able to test with partial data structures.
	// testJSONObject is a generic json object
	testJSONObject map[string]interface{}

	// and ja is a generic json array
	ja []interface{}
)

func TestGetStatus_StatusOk(t *testing.T) {

	version.ReleaseNumber = "1.2.3"
	version.FullCommit = "e0c73b95646559e9a3696d41711e918398d557fb"

	request, _ := http.NewRequest("GET", "/status", nil)
	response := httptest.NewRecorder()

	hydrophone := InitApi(FAKE_CONFIG, mockStore, mockNotifier, mockShoreline, mockPerms, mockSeagull, mockPortal, mockTemplates)
	hydrophone.SetHandlers("", rtr)

	hydrophone.GetStatus(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("Resp given [%d] expected [%d] ", response.Code, http.StatusOK)
	}

	body, _ := ioutil.ReadAll(response.Body)

	if string(body) != `{"status":{"code":200,"reason":"OK"},"version":"1.2.3+e0c73b95646559e9a3696d41711e918398d557fb"}` {
		t.Fatalf("Message given [%s] expected [%s] ", string(body), "OK")
	}

}

func TestGetStatus_StatusInternalServerError(t *testing.T) {

	version.ReleaseNumber = "1.2.3"
	version.FullCommit = "e0c73b95646559e9a3696d41711e918398d557fb"

	request, _ := http.NewRequest("GET", "/status", nil)
	response := httptest.NewRecorder()

	hydrophoneFails := InitApi(FAKE_CONFIG, mockStoreFails, mockNotifier, mockShoreline, mockPerms, mockSeagull, mockPortal, mockTemplates)
	hydrophoneFails.SetHandlers("", rtr)

	hydrophoneFails.GetStatus(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("Resp given [%d] expected [%d] ", response.Code, http.StatusInternalServerError)
	}

	body, _ := ioutil.ReadAll(response.Body)

	if string(body) != `{"status":{"code":500,"reason":"Session failure"},"version":"1.2.3+e0c73b95646559e9a3696d41711e918398d557fb"}` {
		t.Fatalf("Message given [%s] expected [%s] ", string(body), "Session failure")
	}
}

func (i *testJSONObject) deepCompare(j *testJSONObject) string {
	for k := range *i {
		if reflect.DeepEqual((*i)[k], (*j)[k]) == false {
			return fmt.Sprintf("`%s` expected `%v` actual `%v` ", k, (*j)[k], (*i)[k])
		}
	}
	return ""
}

func Test_isAuthorizedUser_Server(t *testing.T) {
	tokenData := &token.TokenData{UserId: "abcdef1234", IsServer: true}
	res := responsableHydrophone.isAuthorizedUser(tokenData, "some_server")
	if res != true {
		t.Fatalf("Test_isAuthorizedUser_Server should have returned true")
	}
}

func Test_isAuthorizedUser_Owner(t *testing.T) {
	tokenData := &token.TokenData{UserId: "abcdef1234", IsServer: false}
	res := responsableHydrophone.isAuthorizedUser(tokenData, "abcdef1234")
	if res != true {
		t.Fatalf("Test_isAuthorizedUser_Owner should have returned true")
	}
}

func Test_isAuthorizedUser_UnAuthorized(t *testing.T) {
	tokenData := &token.TokenData{UserId: "abcdef1234", IsServer: false}
	res := responsableHydrophone.isAuthorizedUser(tokenData, "abcdef1238")
	if res == true {
		t.Fatalf("Test_isAuthorizedUser_UnAuthorized should have returned false")
	}
}
