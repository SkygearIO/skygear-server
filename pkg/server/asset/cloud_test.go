package asset

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCloudStoreCreation(t *testing.T) {
	testServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			method := r.Method
			authHeader := r.Header.Get("Authorization")

			if method == http.MethodGet &&
				path == "/token/testapp" &&
				authHeader == "Bearer correct-auth-token" {

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
          "value": "correct-signer-token",
          "extra": "correct-signer-extra",
          "expired_at": "2016-08-25T10:19:27Z"
        }`))
			} else {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad Request"))
			}
		}),
	)

	Convey("Cloud Store Creation", t, func() {
		Convey("Success on normal flow (public)", func() {
			_store, err := NewCloudStore(
				"testapp",
				testServer.URL,
				"correct-auth-token",
				"http://localhost:12345/public",
				"http://localhost:12345/private",
				true,
				15*60,
				5*60,
			)
			store := _store.(*cloudStore)
			defer store.signer.refreshTicker.Stop()

			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			So(store.appName, ShouldEqual, "testapp")
			So(store.host, ShouldEqual, testServer.URL)
			So(store.authToken, ShouldEqual, "correct-auth-token")
			So(store.urlPrefix, ShouldEqual, "http://localhost:12345/public")
			So(store.public, ShouldBeTrue)

			time.Sleep(100 * time.Millisecond)
			So(store.signer.token, ShouldEqual, "correct-signer-token")
			So(store.signer.extra, ShouldEqual, "correct-signer-extra")
			So(store.signer.expiredAt.Unix(), ShouldEqual, 1472120367)
		})

		Convey("Success on normal flow (private)", func() {
			_store, err := NewCloudStore(
				"testapp",
				testServer.URL,
				"correct-auth-token",
				"http://localhost:12345/public",
				"http://localhost:12345/private",
				false,
				15*60,
				5*60,
			)
			store := _store.(*cloudStore)
			defer store.signer.refreshTicker.Stop()

			So(err, ShouldBeNil)
			So(store, ShouldNotBeNil)
			So(store.appName, ShouldEqual, "testapp")
			So(store.host, ShouldEqual, testServer.URL)
			So(store.authToken, ShouldEqual, "correct-auth-token")
			So(store.urlPrefix, ShouldEqual, "http://localhost:12345/private")
			So(store.public, ShouldBeFalse)

			time.Sleep(100 * time.Millisecond)
			So(store.signer.token, ShouldEqual, "correct-signer-token")
			So(store.signer.extra, ShouldEqual, "correct-signer-extra")
			So(store.signer.expiredAt.Unix(), ShouldEqual, 1472120367)
		})

		Convey("Fail when no app name", func() {
			store, err := NewCloudStore(
				"",
				testServer.URL,
				"correct-auth-token",
				"http://localhost:12345/public",
				"http://localhost:12345/private",
				true,
				15*60,
				5*60,
			)

			So(err, ShouldNotBeNil)
			So(store, ShouldBeNil)
		})

		Convey("Fail when no host", func() {
			store, err := NewCloudStore(
				"testapp",
				"",
				"correct-auth-token",
				"http://localhost:12345/public",
				"http://localhost:12345/private",
				true,
				15*60,
				5*60,
			)

			So(err, ShouldNotBeNil)
			So(store, ShouldBeNil)
		})

		Convey("Fail when no auth token", func() {
			store, err := NewCloudStore(
				"testapp",
				testServer.URL,
				"",
				"http://localhost:12345/public",
				"http://localhost:12345/private",
				true,
				15*60,
				5*60,
			)

			So(err, ShouldNotBeNil)
			So(store, ShouldBeNil)
		})

		Convey("Fail when no public url when needed", func() {
			store, err := NewCloudStore(
				"testapp",
				testServer.URL,
				"correct-auth-token",
				"",
				"http://localhost:12345/private",
				true,
				15*60,
				5*60,
			)

			So(err, ShouldNotBeNil)
			So(store, ShouldBeNil)
		})

		Convey("Fail when no private url when needed", func() {
			store, err := NewCloudStore(
				"testapp",
				testServer.URL,
				"correct-auth-token",
				"http://localhost:12345/public",
				"",
				false,
				15*60,
				5*60,
			)

			So(err, ShouldNotBeNil)
			So(store, ShouldBeNil)
		})
	})
}

func TestCloudStoreGetSignedURL(t *testing.T) {
	testServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			method := r.Method
			authHeader := r.Header.Get("Authorization")

			if method == http.MethodGet &&
				path == "/token/testapp" &&
				authHeader == "Bearer correct-auth-token" {

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
        "value": "correct-signer-token",
        "extra": "correct-signer-extra",
        "expired_at": "2016-08-25T10:19:27Z"
      }`))
			} else {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad Request"))
			}
		}),
	)

	Convey("Cloud Store Get Signed URL", t, func() {
		Convey("Success on public store", func() {
			_publicStore, _ := NewCloudStore(
				"testapp",
				testServer.URL,
				"correct-auth-token",
				"http://localhost:12345/public",
				"http://localhost:12345/private",
				true,
				15*60,
				5*60,
			)
			publicStore := _publicStore.(*cloudStore)
			defer publicStore.signer.refreshTicker.Stop()

			time.Sleep(100 * time.Millisecond)
			signedURL, err := publicStore.SignedURL("file001")
			So(err, ShouldBeNil)
			So(
				signedURL,
				ShouldEqual,
				"http://localhost:12345/public/testapp/file001",
			)
		})

		Convey("Success on private store", func() {
			_publicStore, _ := NewCloudStore(
				"testapp",
				testServer.URL,
				"correct-auth-token",
				"http://localhost:12345/public",
				"http://localhost:12345/private",
				false,
				15*60,
				5*60,
			)
			publicStore := _publicStore.(*cloudStore)
			defer publicStore.signer.refreshTicker.Stop()

			time.Sleep(100 * time.Millisecond)
			signedURL, err := publicStore.SignedURL("file001")
			So(err, ShouldBeNil)

			parsed, err := url.Parse(signedURL)
			So(err, ShouldBeNil)
			So(parsed, ShouldNotBeNil)
			So(parsed.Host, ShouldEqual, "localhost:12345")
			So(parsed.Path, ShouldEqual, "/private/testapp/file001")

			expiredAtString := parsed.Query().Get("expired_at")
			So(expiredAtString, ShouldNotBeEmpty)

			targetExpiredAtUnix := time.Now().Add(15 * time.Minute).Unix()
			expiredAtUnix, err := strconv.ParseInt(expiredAtString, 10, 64)
			So(err, ShouldBeNil)
			So(
				expiredAtUnix,
				ShouldBeBetween,
				targetExpiredAtUnix-100,
				targetExpiredAtUnix+100,
			)

			signature := parsed.Query().Get("signature")
			So(signature, ShouldNotBeEmpty)
		})
	})
}

func TestCloudStoreGeneratePostFileRequest(t *testing.T) {
	testServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			method := r.Method
			authHeader := r.Header.Get("Authorization")

			if method == http.MethodPut &&
				path == "/asset/testapp/file001" &&
				authHeader == "Bearer correct-auth-token" {

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"action": "http://skygear.dev/files/file001",
					"extra-fields": {
						"X-Extra-Field-1": "extra-value-1",
						"X-Extra-Field-2": "extra-value-2"
					}
				}`))
			} else {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Bad Request"))
			}
		}),
	)

	Convey("Generate Post File Request", t, func() {
		Convey("Success on normal flow", func() {
			store := &cloudStore{
				appName:   "testapp",
				host:      testServer.URL,
				authToken: "correct-auth-token",
				urlPrefix: "http://localhost:12345/public",
				public:    true,
			}
			postRequest, err := store.GeneratePostFileRequest("file001")

			So(err, ShouldBeNil)
			So(postRequest, ShouldNotBeNil)

			So(postRequest.Action, ShouldEqual, "http://skygear.dev/files/file001")
			So(postRequest.ExtraFields["X-Extra-Field-1"], ShouldEqual, "extra-value-1")
			So(postRequest.ExtraFields["X-Extra-Field-2"], ShouldEqual, "extra-value-2")
		})
	})
}
