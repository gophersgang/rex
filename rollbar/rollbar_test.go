package rollbar

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type CustomError struct {
	s string
}

func (e *CustomError) Error() string {
	return e.s
}

func testErrorStack(s string) {
	testErrorStack2(s)
}

func testErrorStack2(s string) {
	Error("error", errors.New(s))
}

func testErrorStackWithSkip(s string) {
	testErrorStackWithSkip2(s)
}

func testErrorStackWithSkip2(s string) {
	ErrorWithStackSkip("error", errors.New(s), 2)
}

func TestErrorClass(t *testing.T) {
	errors := map[string]error{
		"{508e076d}":          fmt.Errorf("Something is broken!"),
		"rollbar.CustomError": &CustomError{"Terrible mistakes were made."},
	}

	for expected, err := range errors {
		if GlobalClient.errorClass(err) != expected {
			t.Error("Got:", GlobalClient.errorClass(err), "Expected:", expected)
		}
	}
}

func TestEverything(t *testing.T) {
	GlobalConfig.Token = os.Getenv("TOKEN")
	GlobalConfig.Environment = "test"

	Error("critical", errors.New("Normal critical error"))
	Error("error", &CustomError{"This is a custom error"})

	testErrorStack("This error should have a nice stacktrace")
	testErrorStackWithSkip("This error should have a skipped stacktrace")

	done := make(chan bool)
	go func() {
		testErrorStack("I'm in a goroutine")
		done <- true
	}()
	<-done

	Message("error", "This is an error message")
	Message("info", "And this is an info message")

	// If you don't see the message sent on line 65 in Rollbar, that means this
	// is broken:
	Wait()
}

func TestErrorRequest(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://foo.com/somethere?param1=true", nil)
	r.RemoteAddr = "1.1.1.1:123"

	object := GlobalClient.errorRequest(r)

	if object["url"] != "http://foo.com/somethere?param1=true" {
		t.Errorf("wrong url, got %v", object["url"])
	}

	if object["method"] != "GET" {
		t.Errorf("wrong method, got %v", object["method"])
	}

	if object["query_string"] != "param1=true" {
		t.Errorf("wrong id, got %v", object["query_string"])
	}
}

func TestFilterParams(t *testing.T) {
	values := map[string][]string{
		"password":     {"one"},
		"ok":           {"one"},
		"access_token": {"one"},
	}

	clean := GlobalClient.filterParams(values)
	if clean["password"][0] != FILTERED {
		t.Error("should filter password parameter")
	}

	if clean["ok"][0] == FILTERED {
		t.Error("should keep ok parameter")
	}

	if clean["access_token"][0] != FILTERED {
		t.Error("should filter access_token parameter")
	}
}

func TestFlattenValues(t *testing.T) {
	values := map[string][]string{
		"a": {"one"},
		"b": {"one", "two"},
	}

	flattened := GlobalClient.flattenValues(values)
	if flattened["a"].(string) != "one" {
		t.Error("should flatten single parameter to string")
	}

	if len(flattened["b"].([]string)) != 2 {
		t.Error("should leave multiple parametres as []string")
	}
}

func TestCustomField(t *testing.T) {
	body := GlobalClient.buildError(ERR, errors.New("test-custom"), BuildStack(0), &Field{
		Name: "custom",
		Data: map[string]string{
			"NAME1": "VALUE1",
		},
	})

	dataField, ok := body["data"]
	if !ok {
		t.Error("should have field 'data'")
	}

	data, ok := dataField.(map[string]interface{})
	if !ok {
		t.Error("should be of type map[string]interface{}")
	}

	custom, ok := data["custom"]
	if !ok {
		t.Error("should have field 'custom'")
	}

	customMap, ok := custom.(map[string]string)
	if !ok {
		t.Error("should be a map[string]string")
	}

	val, ok := customMap["NAME1"]
	if !ok {
		t.Error("should have a key 'NAME1'")
	}

	if val != "VALUE1" {
		t.Error("should be VALUE1")
	}
}

func TestNewClientFromGlobal(t *testing.T) {
	Convey("overrides global config values correctly", t, func() {
		c := NewClientFromGlobal(&Config{Token: "1df23546vnf"})
		So(c.config.Token, ShouldNotEqual, GlobalClient.config.Token)
		So(c.config.Endpoint, ShouldEqual, GlobalClient.config.Endpoint)
	})
}
