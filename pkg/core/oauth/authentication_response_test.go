package oauth

import (
	"net/url"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestNewAuthenticationResponse(t *testing.T) {
	Convey("NewAuthenticationResponse", t, func() {
		test := func(form url.Values, expected string) {
			actualURL, err := NewAuthenticationResponse(form, "code")
			So(err, ShouldBeNil)

			expectedURL, err := url.Parse(expected)
			So(err, ShouldBeNil)

			q1 := actualURL.Query()
			q2 := expectedURL.Query()

			So(q1, ShouldResemble, q2)
		}

		test(url.Values{
			"redirect_uri": []string{"http://example.com"},
		}, "http://example.com?code=code")

		test(url.Values{
			"redirect_uri": []string{"http://example.com"},
			"state":        []string{"state"},
		}, "http://example.com?code=code&state=state")

		test(url.Values{
			"redirect_uri": []string{"http://example.com?a=b"},
			"state":        []string{"state"},
		}, "http://example.com?a=b&code=code&state=state")
	})
}
