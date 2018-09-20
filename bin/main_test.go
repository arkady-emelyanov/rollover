package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/arkady-emelyanov/rollover/config"
	"github.com/google/go-cmp/cmp"
	"gopkg.in/olivere/elastic.v6"
)

func getHTTPResponse(t *testing.T, uri string, body []byte) ([]byte, int) {
	mappedFile := strings.Replace(uri, "/", "_", -1)
	jsonFile := fmt.Sprintf("./_testdata/%s.json", mappedFile)

	code := 200
	if _, err := os.Stat(jsonFile); err != nil {
		code = 500
	}

	if len(body) > 0 {
		bodyFile := fmt.Sprintf("./_testdata/%s_body.json", mappedFile)
		reqBody := make(map[string]interface{})
		if err := json.Unmarshal(body, &reqBody); err != nil {
			panic(err)
		}

		rb, _ := ioutil.ReadFile(bodyFile)
		resBody := make(map[string]interface{})
		if err := json.Unmarshal(rb, &resBody); err != nil {
			panic(err)
		}

		fmt.Println(reqBody, resBody)
		if !cmp.Equal(reqBody, resBody) {
			t.Fatal("should be equal")
		}
	}

	b, _ := ioutil.ReadFile(jsonFile)
	return b, code
}

func getHTTPServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBody, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()

		resBody, code := getHTTPResponse(t, r.RequestURI, reqBody)
		w.WriteHeader(code)
		w.Header().Set("Content-Type", "application/json")
		w.Write(resBody)
	}))
}

func getClientAndServer(t *testing.T) (*elastic.Client, *httptest.Server) {
	s := getHTTPServer(t)
	c, err := elastic.NewClient(
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
		elastic.SetURL(s.URL),
	)
	if err != nil {
		t.Fatal(err)
	}
	return c, s
}

func TestRollover(t *testing.T) {
	c, s := getClientAndServer(t)
	defer s.Close()
	defer c.Stop()

	// checking rollover
	st := &rolloverSt{
		client: c,
		ctx:    context.Background(),
		rule: config.RolloverAlias{
			Alias:   "test-index-01",
			NewName: "test-index-02",
			Conditions: config.RolloverConditions{
				MaxAge:  "7h",
				MaxDocs: 1000,
			},
		},
	}

	res := doRollover(st)
	if res == nil {
		t.Fatal("shouldn't be nil", st.err)
	}
}

func TestMakeReadOnly(t *testing.T) {
	c, s := getClientAndServer(t)
	defer s.Close()
	defer c.Stop()

	// checking rollover
	st := &rolloverSt{
		client:       c,
		ctx:          context.Background(),
		oldIndexName: "test-index-01",
		newIndexName: "test-index-02",
	}

	res := doMakeReadOnly(st)
	if res == nil {
		t.Fatal("shouldn't be nil", st.err)
	}
}
