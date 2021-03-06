package handlers

import (
	"bytes"
	"encoding/json"
	"github.com/jcmturner/mfaserver/config"
	"github.com/jcmturner/mfaserver/testtools"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestEnrolStatus(t *testing.T) {
	//Set up mock LDAP server
	l := testtools.NewLDAPServer(t)
	defer l.Stop()
	//Set up mock Vault instance
	ln, addr, appID, userID := testtools.RunMockVault(t)
	defer ln.Close()

	//Set up the MFA config
	c := config.NewConfig()
	c.WithVaultAppIdWrite(appID).WithVaultUserId(userID).WithVaultEndPoint(addr)
	c.WithLDAPConnection("ldap://"+l.Listener.Addr().String(), "", "{username}")
	c.MFAServer.Loggers.Debug = log.New(os.Stdout, "MFA Debug: ", log.Ldate|log.Ltime|log.Lshortfile)
	c.MFAServer.Loggers.Info = log.New(os.Stdout, "MFA Info: ", log.Ldate|log.Ltime|log.Lshortfile)
	c.MFAServer.Loggers.Warning = log.New(os.Stdout, "MFA Warn: ", log.Ldate|log.Ltime|log.Lshortfile)
	c.MFAServer.Loggers.Error = log.New(os.Stderr, "MFA Error: ", log.Ldate|log.Ltime|log.Lshortfile)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { Enrol(w, r, c) }))
	defer s.Close()

	var tests = []struct {
		Json     string
		HttpCode int
	}{
		{`{"domain": "testdom", "username": "validuser", "password": "validpassword", "issuer": "testapp"}`, http.StatusCreated},
		// Try again to test the 2nd time we are forbidden to enrol
		{`{"domain": "testdom", "username": "validuser", "password": "validpassword", "issuer": "testapp"}`, http.StatusForbidden},
		{`{"domain": "testdom", "username": "validuser", "password": "invalidpassword", "issuer": "testapp"}`, http.StatusUnauthorized},
		{`{"domain": "testdom", "username": "invaliduser", "password": "validpassword", "issuer": "testapp"}`, http.StatusUnauthorized},
		{`{"domain": "testdom", "password": "validpassword", "issuer": "testapp"}`, http.StatusBadRequest},
		{`{"domain": "testdom", "username": "validuser", "issuer": "testapp"}`, http.StatusBadRequest},
		{`{"domain": "testdom", "username": "validuser", "password": "validpassword"}`, http.StatusBadRequest},
		{`{"username": "validuser", "password": "validpassword", "issuer": "testapp"}`, http.StatusBadRequest},
		{`"domain": "testdom", "username": "validuser", "password": "validpassword", "issuer": "testapp"}`, http.StatusBadRequest},
	}
	for _, test := range tests {
		r, err := http.NewRequest("POST", s.URL+"/enrol", bytes.NewBuffer([]byte(test.Json)))
		if err != nil {
			t.Errorf("Error returned from creating request: %v", err)
		}
		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			t.Errorf("Error returned from sending request: %v", err)
		}
		if resp.StatusCode != test.HttpCode {
			t.Errorf("Expected code %v, got %v for post data %v", test.HttpCode, resp.StatusCode, test.Json)
		}
		//Check the JSON response was correct format
		if resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var dec *json.Decoder
			var j enrolResponseData
			dec = json.NewDecoder(resp.Body)
			err = dec.Decode(&j)
			if err != nil {
				body, _ := ioutil.ReadAll(r.Body)
				t.Errorf("Failed to marshal the response into the JSON object. Response: %s", body)
			}
		}
	}
}

func TestEnrolQRCode(t *testing.T) {
	//Set up mock LDAP server
	l := testtools.NewLDAPServer(t)
	defer l.Stop()
	//Set up mock Vault instance
	ln, addr, appID, userID := testtools.RunMockVault(t)
	defer ln.Close()

	//Set up the MFA config
	c := config.NewConfig()
	c.WithVaultAppIdWrite(appID).WithVaultUserId(userID).WithVaultEndPoint(addr)
	c.WithLDAPConnection("ldap://"+l.Listener.Addr().String(), "", "{username}")
	c.MFAServer.Loggers.Debug = log.New(os.Stdout, "MFA Debug: ", log.Ldate|log.Ltime|log.Lshortfile)
	c.MFAServer.Loggers.Info = log.New(os.Stdout, "MFA Info: ", log.Ldate|log.Ltime|log.Lshortfile)
	c.MFAServer.Loggers.Warning = log.New(os.Stdout, "MFA Warn: ", log.Ldate|log.Ltime|log.Lshortfile)
	c.MFAServer.Loggers.Error = log.New(os.Stderr, "MFA Error: ", log.Ldate|log.Ltime|log.Lshortfile)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { Enrol(w, r, c) }))
	defer s.Close()

	r, _ := http.NewRequest("POST", s.URL+"/enrol", bytes.NewBuffer([]byte(`{"domain": "testdom", "username": "validuser", "password": "validpassword", "issuer": "testapp"}`)))
	r.Header.Set("Accept-Encoding", "image/png")
	resp, err := http.DefaultClient.Do(r)
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected code %v, got %v for QR test", http.StatusCreated, resp.StatusCode)
	}
	if err != nil {
		t.Errorf("Error returned from sending request: %v", err)
	}
	_, err = png.Decode(resp.Body)
	if err != nil {
		t.Errorf("Could not decode QR code response into a png object: %v", err)
	}
}
