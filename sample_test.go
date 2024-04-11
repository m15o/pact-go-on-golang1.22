package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/pact-foundation/pact-go/v2/consumer"
	"github.com/pact-foundation/pact-go/v2/log"
	"github.com/pact-foundation/pact-go/v2/matchers"
	"github.com/pact-foundation/pact-go/v2/models"
	"github.com/stretchr/testify/assert"
)

type S = matchers.S

var Like = matchers.Like
var Regex = matchers.Regex
var ArrayMinLike = matchers.ArrayMinLike

type Map = matchers.MapMatcher

var Decimal = matchers.Decimal
var Integer = matchers.Integer
var Equality = matchers.Equality
var Includes = matchers.Includes
var FromProviderState = matchers.FromProviderState
var ArrayContaining = matchers.ArrayContaining
var ArrayMinMaxLike = matchers.ArrayMinMaxLike
var DateTimeGenerated = matchers.DateTimeGenerated

func TestConsumerV4(t *testing.T) {
	log.SetLogLevel("TRACE")

	mockProvider, err := consumer.NewV4Pact(consumer.MockHTTPProviderConfig{
		Consumer: "PactGoV4Consumer",
		Provider: "V4Provider",
		Host:     "127.0.0.1",
		TLS:      true,
	})
	assert.NoError(t, err)

	// Set up our expected interactions.
	err = mockProvider.
		AddInteraction().
		Given("state 1").
		GivenWithParameter(models.ProviderState{
			Name: "User foo exists",
			Parameters: map[string]interface{}{
				"id": "foo",
			},
		}).
		UponReceiving("A request to do a foo").
		WithRequest("POST", "/foobar", func(b *consumer.V4RequestBuilder) {
			b.
				Header("Content-Type", S("application/json")).
				Header("Authorization", Like("Bearer 1234")).
				Query("baz", Regex("bar", "[a-z]+"), Regex("bat", "[a-z]+"), Regex("baz", "[a-z]+")).
				JSONBody(Map{
					"id":       Like(27),
					"name":     FromProviderState("${name}", "billy"),
					"lastName": Like("billy"),
					"datetime": DateTimeGenerated("2020-01-01T08:00:45", "yyyy-MM-dd'T'HH:mm:ss"),
				})

		}).
		WillRespondWith(200, func(b *consumer.V4ResponseBuilder) {
			b.
				Header("Content-Type", S("application/json")).
				JSONBody(Map{
					"datetime":       Regex("2020-01-01", "[0-9\\-]+"),
					"name":           S("Billy"),
					"lastName":       S("Sampson"),
					"superstring":    Includes("foo"),
					"id":             Integer(12),
					"accountBalance": Decimal(123.76),
					"itemsMinMax":    ArrayMinMaxLike(27, 3, 5),
					"itemsMin":       ArrayMinLike("thereshouldbe3ofthese", 3),
					"equality":       Equality("a thing"),
					"arrayContaining": ArrayContaining([]interface{}{
						Like("string"),
						Integer(1),
						Map{
							"foo": Like("bar"),
						},
					}),
				})
		}).
		ExecuteTest(t, test)
	assert.NoError(t, err)
}

var test = func() func(config consumer.MockServerConfig) error {
	return rawTest("baz=bat&baz=foo&baz=something")
}()

var rawTest = func(query string) func(config consumer.MockServerConfig) error {

	return func(config consumer.MockServerConfig) error {

		config.TLSConfig.InsecureSkipVerify = true
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: config.TLSConfig,
			},
		}
		req := &http.Request{
			Method: "POST",
			URL: &url.URL{
				Host:     fmt.Sprintf("%s:%d", "localhost", config.Port),
				Scheme:   "https",
				Path:     "/foobar",
				RawQuery: query,
			},
			Body:   io.NopCloser(strings.NewReader(`{"id": 27, "name":"billy", "lastName":"sampson", "datetime":"2021-01-01T08:00:45"}`)),
			Header: make(http.Header),
		}

		// NOTE: by default, request bodies are expected to be sent with a Content-Type
		// of application/json. If you don't explicitly set the content-type, you
		// will get a mismatch during Verification.
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer 1234")

		_, err := client.Do(req)

		return err
	}
}
