package time

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestToMilliseconds(t *testing.T) {
	Convey("ToMilliseconds", t, func() {
		d, err := time.ParseDuration("300ms")
		So(err, ShouldBeNil)
		So(ToMilliseconds(d), ShouldEqual, 300)
	})
}
