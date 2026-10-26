package blobstore

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/owncloud/reva/v2/pkg/storage/utils/decomposedfs/node"
)

// s3 answers every DELETE with the given status; 404 carries a NoSuchKey body like GCS does.
func s3(status int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead || r.URL.Query().Has("location") {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(`<LocationConstraint>us-east-1</LocationConstraint>`))
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(status)
		switch status {
		case http.StatusNotFound:
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>NoSuchKey</Code><Message>The specified key does not exist.</Message></Error>`))
		case http.StatusForbidden:
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>AccessDenied</Code><Message>denied</Message></Error>`))
		}
	}))
}

func del(t *testing.T, status int) error {
	srv := s3(status)
	defer srv.Close()
	bs, err := New(srv.URL, "us-east-1", "bucket", "key", "secret", Options{})
	if err != nil {
		t.Fatal(err)
	}
	n := &node.Node{SpaceID: "space", BlobID: "aabbccdd-1111-2222-3333-444455556666"}
	return bs.Delete(n)
}

func TestDeleteMissingIsSuccess(t *testing.T) {
	if err := del(t, http.StatusNotFound); err != nil {
		t.Fatalf("404 NoSuchKey should count as deleted, got %v", err)
	}
}

func TestDeleteOK(t *testing.T) {
	if err := del(t, http.StatusNoContent); err != nil {
		t.Fatalf("204 should succeed, got %v", err)
	}
}

func TestDeleteOtherErrorsStillFail(t *testing.T) {
	err := del(t, http.StatusForbidden)
	if err == nil || !strings.Contains(err.Error(), "could not delete object") {
		t.Fatalf("403 must still fail, got %v", err)
	}
}
