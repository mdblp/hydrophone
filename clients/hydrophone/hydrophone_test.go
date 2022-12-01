package hydrophone

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func buildServer(t *testing.T, userID string, testToken string, confirmType string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		urlPath := req.URL.Path
		if strings.HasPrefix(urlPath, "/"+confirmType+"/") {
			if req.Method != "GET" && req.Method != "PUT" {
				t.Errorf("Incorrect HTTP Method [%s]", req.Method)
			} else if req.Header.Get("x-tidepool-session-token") != testToken && req.Header.Get("Authorization") != "Bearer "+testToken {
				res.WriteHeader(http.StatusUnauthorized)
			} else {
				userID = strings.TrimPrefix(urlPath, "/"+confirmType+"/")
				if req.Method == "GET" {
					switch userID {
					case "authorizedWithData":
						res.WriteHeader(http.StatusOK)
						if confirmType == "invite" {
							fmt.Fprint(res, `[{"key":"key1","type":"medicalteam_invitation"}, {"key":"key2","type":"medicalteam_do_admin"}]`)
						} else {
							fmt.Fprint(res, `[{"key":"key3","type":"signup_confirmation"}]`)
						}
					case "authorizedWithWrongData":
						res.WriteHeader(http.StatusOK)
						fmt.Fprint(res, `{"key":"key1"}`)
					case "authorizedWithoutData":
						res.WriteHeader(http.StatusNotFound)
					case "unrecognizedResponseCode":
						res.WriteHeader(http.StatusNoContent)
					case "error":
						res.WriteHeader(http.StatusInternalServerError)
					}
				} else {
					switch userID {
					case "error":
						res.WriteHeader(http.StatusInternalServerError)
					default:
						res.WriteHeader(http.StatusOK)
					}
				}

			}
		} else {
			t.Errorf("Unknown path[%s]", urlPath)
		}
	}))
}
