package loginid

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestDefaultRealmChecker(t *testing.T) {
	Convey("DefaultRealmChecker", t, func() {
		Convey("no allowed realms", func() {
			c := &DefaultRealmChecker{}
			So(c.IsValid("default"), ShouldBeFalse)
		})
		Convey("default realm", func() {
			c := &DefaultRealmChecker{
				AllowedRealms: []string{DefaultRealm},
			}
			So(c.IsValid("default"), ShouldBeTrue)
		})
		Convey("custom realm", func() {
			c := &DefaultRealmChecker{
				AllowedRealms: []string{"a", "b"},
			}
			So(c.IsValid("default"), ShouldBeFalse)
			So(c.IsValid("a"), ShouldBeTrue)
			So(c.IsValid("b"), ShouldBeTrue)
		})
	})
}
