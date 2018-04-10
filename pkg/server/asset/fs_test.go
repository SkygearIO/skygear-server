package asset

import (
	"net/url"
	"strconv"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestFileStore(t *testing.T) {

	Convey("FS Asset Store", t, func() {
		fsStore := &fileStore{
			"data/asset",
			"http://skygear.dev/files",
			"asset_secret",
			false,
			time.Minute * time.Duration(15),
			time.Minute * time.Duration(5),
		}
		Convey("Sign the Parse Signature correctly", func() {
			s, err := fsStore.SignedURL("index.html")
			So(err, ShouldBeNil)
			parsedURL, urlErr := url.Parse(s)
			So(urlErr, ShouldBeNil)
			qs := parsedURL.Query()
			expiredAtUnix, expiredErr := strconv.ParseInt(qs.Get("expiredAt"), 10, 64)
			So(expiredErr, ShouldBeNil)
			expiredAt := time.Unix(expiredAtUnix, 0)
			valid, matchErr := fsStore.ParseSignature(
				qs.Get("signature"),
				"index.html",
				expiredAt,
			)
			So(matchErr, ShouldBeNil)
			So(valid, ShouldBeTrue)
		})

		Convey("Parse Signature correctly", func() {
			expiredAt := time.Unix(1481096834, 0)
			valid, matchErr := fsStore.ParseSignature(
				"R5kMq2neUkCGBjQD6zSv99PajRvI0EqMesuRHQS4hNA=",
				"index.html",
				expiredAt,
			)
			So(matchErr, ShouldBeNil)
			So(valid, ShouldBeTrue)
		})

		Convey("Reject incorrectly Signature correctly", func() {
			expiredAt := time.Unix(1481096834, 0)
			valid, matchErr := fsStore.ParseSignature(
				"limouren",
				"index.html",
				expiredAt,
			)
			So(matchErr, ShouldBeNil)
			So(valid, ShouldBeFalse)
		})

	})
}
