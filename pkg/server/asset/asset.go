// Copyright 2015-present Oursky Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package asset

import (
	"io"
	"time"

	"github.com/skygeario/skygear-server/pkg/server/logging"
)

var log = logging.LoggerEntry("asset")

// PostFileRequest models the POST request for upload asset file
type PostFileRequest struct {
	Action      string                 `json:"action"`
	ExtraFields map[string]interface{} `json:"extra-fields,omitempty"`
}

// Store specify the interfaces of an asset store
type Store interface {
	GetFileReader(name string) (io.ReadCloser, error)
	PutFileReader(name string, src io.Reader, length int64, contentType string) error
	GeneratePostFileRequest(name string) (*PostFileRequest, error)
}

// URLSigner signs a signature and returns a URL accessible to that asset.
type URLSigner interface {
	// SignedURL returns a url with access to the named file. If asset
	// store is private, the returned URL is a signed one, allowing access
	// to asset for a short period.
	SignedURL(name string) (string, error)
	IsSignatureRequired() bool
}

// SignatureParser parses a signed signature string
type SignatureParser interface {
	ParseSignature(signed string, name string, expiredAt time.Time) (valid bool, err error)
}

// getPresignIntervalStartTime returns the asset expiration interval start time,
// which is added to the expiry duration to calculate the asset expiration time.
func getPresignIntervalStartTime(now time.Time, interval time.Duration) time.Time {
	if int64(interval) == 0 {
		return now
	}
	return time.Unix((now.Unix()/int64(interval))*int64(interval), 0)
}

// getPresignExpireTime returns the asset expiration time that is consistent
// for the duration of the current presign interval.
func getPresignExpireTime(now time.Time, interval, expiry time.Duration) time.Time {
	return getPresignIntervalStartTime(now, interval).Add(expiry)
}
