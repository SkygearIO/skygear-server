package loginid

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/skygeario/skygear-server/pkg/core/config"
)

func TestEmailTypeChecker(t *testing.T) {
	type Case struct {
		LoginID string
		Err     string
	}
	f := func(c Case, check TypeChecker) {
		err := check.Validate(c.LoginID)

		if c.Err == "" {
			So(err, ShouldBeNil)
		} else {
			So(err, ShouldBeError, c.Err)
		}
	}
	newTrue := func() *bool {
		b := true
		return &b
	}
	newFalse := func() *bool {
		b := false
		return &b
	}
	Convey("EmailTypeChecker", t, func() {
		Convey("default setting", func() {
			cases := []Case{
				{"Faseng@Example.com", ""},
				{"Faseng+Chima@example.com", ""},
				{"faseng.the.cat", "invalid login ID"},
				{"fasengthecat", "invalid login ID"},
				{"fasengthecat@", "invalid login ID"},
				{"@fasengthecat", "invalid login ID"},
				{"Faseng <faseng@example>", "invalid login ID"},
				{"faseng.ℌ𝒌@測試.香港", ""},
				{`"fase ng@cat"@example.com`, ""},
				{`"faseng@"@example.com`, ""},
			}

			check := &EmailTypeChecker{
				config: &config.LoginIDTypeEmailConfiguration{
					BlockPlusSign: newFalse(),
				},
			}

			for _, c := range cases {
				f(c, check)
			}
		})

		Convey("block plus sign", func() {
			cases := []Case{
				{"Faseng@Example.com", ""},
				{"Faseng+Chima@example.com", "invalid login ID"},
				{`"faseng@cat+123"@example.com`, "invalid login ID"},
			}

			checker := &EmailTypeChecker{
				config: &config.LoginIDTypeEmailConfiguration{
					BlockPlusSign: newTrue(),
				},
			}

			for _, c := range cases {
				f(c, checker)
			}
		})
	})
}

func TestUsernameTypeChecker(t *testing.T) {
	type Case struct {
		LoginID string
		Err     string
	}
	f := func(c Case, check TypeChecker) {
		err := check.Validate(c.LoginID)

		if c.Err == "" {
			So(err, ShouldBeNil)
		} else {
			So(err, ShouldBeError, c.Err)
		}
	}
	newTrue := func() *bool {
		b := true
		return &b
	}
	newFalse := func() *bool {
		b := false
		return &b
	}
	Convey("UsernameTypeChecker", t, func() {
		Convey("allow all", func() {
			cases := []Case{
				{"admin", ""},
				{"settings", ""},
				{"skygear", ""},
				{"花生thecat", ""},
				{"faseng", ""},

				// space is not allowed in Identifier class
				{"Test ID", "invalid login ID"},

				// confusable homoglyphs
				{"microsoft", ""},
				{"microsоft", "invalid login ID"},
				// byte array versions
				{string([]byte{109, 105, 99, 114, 111, 115, 111, 102, 116}), ""},
				{string([]byte{109, 105, 99, 114, 111, 115, 208, 190, 102, 116}), "invalid login ID"},
			}

			n := &UsernameTypeChecker{
				config: &config.LoginIDTypeUsernameConfiguration{
					BlockReservedUsernames: newFalse(),
					ExcludedKeywords:       []string{},
					ASCIIOnly:              newFalse(),
				},
			}

			for _, c := range cases {
				f(c, n)
			}
		})

		Convey("block keywords and non ascii", func() {
			cases := []Case{
				{"admin", "invalid login ID"},
				{"settings", "invalid login ID"},
				{"skygear", "invalid login ID"},
				{"skygearcloud", "invalid login ID"},
				{"myskygearapp", "invalid login ID"},
				{"花生thecat", "invalid login ID"},
				{"faseng", ""},
				{"faseng_chima-the.cat", ""},
			}

			reversedNameChecker, _ := NewReservedNameCheckerWithFile("../../../reserved_name.txt")
			n := &UsernameTypeChecker{
				config: &config.LoginIDTypeUsernameConfiguration{
					BlockReservedUsernames: newTrue(),
					ExcludedKeywords:       []string{"skygear"},
					ASCIIOnly:              newTrue(),
				},
				reservedNameChecker: reversedNameChecker,
			}

			for _, c := range cases {
				f(c, n)
			}
		})
	})
}

func TestPhoneTypeChecker(t *testing.T) {
	Convey("PhoneTypeChecker", t, func() {
		c := &PhoneTypeChecker{}
		So(c.Validate(""), ShouldNotBeNil)
		So(c.Validate("+85222334455"), ShouldBeNil)
	})
}

func TestNullTypeChecker(t *testing.T) {
	Convey("NullTypeChecker", t, func() {
		c := &NullTypeChecker{}
		So(c.Validate(""), ShouldBeNil)
		So(c.Validate("a"), ShouldBeNil)
		So(c.Validate("+85222334455"), ShouldBeNil)
		So(c.Validate("user@example.com"), ShouldBeNil)
	})
}
