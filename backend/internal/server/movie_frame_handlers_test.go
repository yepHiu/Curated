package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSourceFrameRejectsInvalidPosition(t *testing.T) {
	h := &Handler{}
	for _, body := range []string{`{"positionSec":-1}`, `{"positionSec":1e999}`, `{"positionSec":999999999}`, `broken`} {
		r := httptest.NewRequest(http.MethodPost, "/api/library/movies/test/frame", strings.NewReader(body))
		w := httptest.NewRecorder()
		h.handleExtractMovieFrame(w, r)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d", body, w.Code)
		}
	}
}

func TestFrameOutputMemoryBound(t *testing.T) {
	b := boundedFrameBuffer{limit: 4}
	if _, err := b.Write([]byte("1234")); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Write([]byte("5")); err == nil {
		t.Fatal("oversize output accepted")
	}
	if b.Len() != 4 {
		t.Fatalf("buffer grew to %d", b.Len())
	}
}
