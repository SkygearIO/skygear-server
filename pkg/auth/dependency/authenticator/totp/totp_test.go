package totp_test

import (
	"testing"
	"time"

	"github.com/skygeario/skygear-server/pkg/auth/dependency/authenticator/totp"
	. "github.com/smartystreets/goconvey/convey"
)

func TestTOTP(t *testing.T) {
	Convey("TOTP", t, func() {
		// nolint: gosec
		fixtureSecret := "GJQFQHET4FX7U5EWSXU36MM36X46TJ7E"
		fixtureTime := time.Date(2019, 6, 1, 0, 0, 0, 0, time.UTC)

		Convey("GenerateSecret", func() {
			secret, err := totp.GenerateSecret()
			So(err, ShouldBeNil)
			So(secret, ShouldNotBeEmpty)
			// The secret is of 160 bits
			// Base32 groups 5 bits into 1 character.
			// So the length should be 160/5 = 32.
			So(len(secret), ShouldEqual, 32)
		})

		Convey("GenerateCode", func() {
			code, err := totp.GenerateCode(fixtureSecret, fixtureTime)
			So(err, ShouldBeNil)
			// Should be 6 digits
			So(len(code), ShouldEqual, 6)
			So(code, ShouldEqual, "833848")
		})

		Convey("ValidateCode", func() {
			Convey("Within the same period", func() {
				code, err := totp.GenerateCode(fixtureSecret, fixtureTime)
				So(err, ShouldBeNil)

				valid := totp.ValidateCode(fixtureSecret, code, fixtureTime)
				So(valid, ShouldBeTrue)
			})

			Convey("-1 period", func() {
				code, err := totp.GenerateCode(fixtureSecret, fixtureTime)
				So(err, ShouldBeNil)

				t1 := fixtureTime.Add(-30 * time.Second)
				t1Code, err := totp.GenerateCode(fixtureSecret, t1)
				So(err, ShouldBeNil)
				So(t1Code, ShouldNotEqual, code)
				So(t1Code, ShouldEqual, "817861")
				valid := totp.ValidateCode(fixtureSecret, t1Code, fixtureTime)
				So(valid, ShouldBeTrue)
			})

			Convey("+1 period", func() {
				code, err := totp.GenerateCode(fixtureSecret, fixtureTime)
				So(err, ShouldBeNil)

				t2 := fixtureTime.Add(30 * time.Second)
				t2Code, err := totp.GenerateCode(fixtureSecret, t2)
				So(err, ShouldBeNil)
				So(t2Code, ShouldNotEqual, code)
				So(t2Code, ShouldEqual, "503766")
				valid := totp.ValidateCode(fixtureSecret, t2Code, fixtureTime)
				So(valid, ShouldBeTrue)
			})

			Convey("Invalid code", func() {
				valid := totp.ValidateCode(fixtureSecret, "123456", fixtureTime)
				So(valid, ShouldBeFalse)
			})

			Convey("Expired code", func() {
				code, err := totp.GenerateCode(fixtureSecret, fixtureTime)
				So(err, ShouldBeNil)

				t1 := fixtureTime.Add(-60 * time.Second)
				t1Code, err := totp.GenerateCode(fixtureSecret, t1)
				So(err, ShouldBeNil)
				So(t1Code, ShouldNotEqual, code)
				So(t1Code, ShouldEqual, "369494")
				valid := totp.ValidateCode(fixtureSecret, t1Code, fixtureTime)
				So(valid, ShouldBeFalse)
			})
		})
	})
}
