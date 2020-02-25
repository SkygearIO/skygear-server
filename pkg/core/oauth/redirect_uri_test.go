package oauth

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestValidateRedirectURI(t *testing.T) {
	Convey("Test ValidateRedirectURI", t, func() {
		f := ValidateRedirectURI

		So(f(nil, ""), ShouldBeError, "missing redirect URI")

		cases := []struct {
			urls        []string
			redirectURI string
			valid       bool
		}{
			{nil, "a", false},
			{[]string{}, "a", false},
			{[]string{"b"}, "a", false},
			{[]string{"/a"}, "/a/b", false},
			{[]string{"/a/c"}, "/a/b", false},
			{[]string{"/A/B"}, "/a/b", false},

			{[]string{"a"}, "a", true},

			// no query nor fragment
			{[]string{"/a"}, "/a", true},

			// Ignore query in redirectURI
			{[]string{"/a"}, "/a?q=1", true},
			// Does not ignore query in allowedCallbackURLs, leading to impossible match.
			{[]string{"/a?q=1"}, "/a?q=1", false},

			// Ignore fragment in redirectURI
			{[]string{"/a"}, "/a#f", true},
			// Does not ignore fragment in allowedCallbackURLs, leading to impossible match.
			{[]string{"/a#f"}, "/a#f", false},

			// Ignore trailing slash
			{[]string{"http://example.com/"}, "http://example.com", true},
			{[]string{"http://example.com/"}, "http://example.com/?q=1", true},
			{[]string{"http://example.com/"}, "http://example.com/#f", true},
			{[]string{"http://example.com/"}, "http://example.com/?q=1#f", true},

			{[]string{"http://example.com"}, "http://example.com/", true},
			{[]string{"http://example.com"}, "http://example.com/?q=1", true},
			{[]string{"http://example.com"}, "http://example.com/#f", true},
			{[]string{"http://example.com"}, "http://example.com/?q=1#f", true},

			{[]string{"http://example.com/a"}, "http://example.com/a", true},
			{[]string{"http://example.com/a"}, "http://example.com/a?q=1", true},
			{[]string{"http://example.com/a"}, "http://example.com/a#f", true},
			{[]string{"http://example.com/a"}, "http://example.com/a?q=1#f", true},

			{[]string{"http://example.com/a"}, "http://example.com/ab", false},
			{[]string{"http://example.com/a"}, "http://example.com/ab?q=1", false},
			{[]string{"http://example.com/a"}, "http://example.com/ab#f", false},
			{[]string{"http://example.com/a"}, "http://example.com/ab?q=1#f", false},

			{[]string{"http://example.com/a/"}, "http://example.com/a", true},
			{[]string{"http://example.com/a/"}, "http://example.com/a?q=1", true},
			{[]string{"http://example.com/a/"}, "http://example.com/a#f", true},
			{[]string{"http://example.com/a/"}, "http://example.com/a?q=1#f", true},
			{[]string{"http://example.com/a/"}, "http://example.com/a/", true},
			{[]string{"http://example.com/a/"}, "http://example.com/a/?q=1", true},
			{[]string{"http://example.com/a/"}, "http://example.com/a/#f", true},
			{[]string{"http://example.com/a/"}, "http://example.com/a/?q=1#f", true},
		}

		for _, c := range cases {
			So(f(c.urls, c.redirectURI) == nil, ShouldEqual, c.valid)
		}
	})
}
