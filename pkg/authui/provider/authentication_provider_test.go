package provider

import (
	"net/url"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDeriveLoginID(t *testing.T) {
	Convey("DeriveLoginID", t, func() {
		test := func(expected string, typ string, xLoginID string, callingCode string, nationalNumber string) {
			form := url.Values{}
			form.Set("x_login_id_input_type", typ)
			form.Set("x_login_id", xLoginID)
			form.Set("x_calling_code", callingCode)
			form.Set("x_national_number", nationalNumber)

			actual := DeriveLoginID(form)
			So(actual, ShouldEqual, expected)
		}

		test("user@example.com", "text", "user@example.com", "", "")
		test("+85222334455", "phone", "", "852", "2233 4455")
	})
}
