package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"go.uber.org/zap"

	"curated-backend/internal/contracts"
	"curated-backend/internal/curatedexport"
	"curated-backend/internal/storage"
	"curated-backend/internal/version"
)

const maxCuratedExportFrames = 20

func dedupeCuratedExportIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func contentDispositionAttachment(filename, asciiFallback string) string {
	var ascii strings.Builder
	for _, r := range filename {
		if r < 0x80 && r != '"' && r != '\\' && r != '\r' && r != '\n' {
			ascii.WriteRune(r)
		} else {
			ascii.WriteByte('_')
		}
	}
	fb := ascii.String()
	if fb == "" || !utf8.ValidString(filename) {
		fb = asciiFallback
	}
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, fb, url.PathEscape(filename))
}

func (h *Handler) handlePostCuratedFramesExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAppError(w, http.StatusMethodNotAllowed, contracts.ErrorCodeBadRequest, "method not allowed")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "failed to read body")
		return
	}
	var req contracts.PostCuratedFramesExportBody
	if err := json.Unmarshal(body, &req); err != nil {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid json body")
		return
	}
	ids := dedupeCuratedExportIDs(req.IDs)
	if len(ids) == 0 {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "ids is required")
		return
	}
	if len(ids) > maxCuratedExportFrames {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, fmt.Sprintf("at most %d frames per export", maxCuratedExportFrames))
		return
	}

	format := strings.ToLower(strings.TrimSpace(req.Format))
	if format == "" {
		format = "webp"
	}
	if format != "webp" && format != "png" {
		writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, `format must be "webp" or "png"`)
		return
	}

	ctx := r.Context()
	rows, err := h.store.ListCuratedFramesForExport(ctx, ids)
	if err != nil {
		if errors.Is(err, storage.ErrCuratedFrameNotFound) {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, err.Error())
			return
		}
		h.logger.Error("curated frames export list", zap.Error(err))
		writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to load curated frames")
		return
	}

	actorContext := strings.TrimSpace(req.ActorName)
	usedNames := make(map[string]struct{})
	type outFile struct {
		name string
		data []byte
	}
	files := make([]outFile, 0, len(rows))
	exportedAt := time.Now().UTC().Format(time.RFC3339Nano)
	appVersion := version.Display()

	for _, row := range rows {
		if len(row.ImageBlob) == 0 {
			writeAppError(w, http.StatusNotFound, contracts.ErrorCodeNotFound, "curated frame has no image")
			return
		}
		actorForName, err := curatedexport.FilenameActor(row.Actors, actorContext)
		if err != nil {
			if errors.Is(err, curatedexport.ErrActorContextNotInFrame) {
				writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeCuratedExportActorMismatch, "actorName is not in this frame's actors")
				return
			}
			writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, err.Error())
			return
		}
		meta := curatedexport.FrameMetaJSON{
			Title:         row.Title,
			Code:          row.Code,
			Actors:        row.Actors,
			PositionSec:   row.PositionSec,
			CapturedAt:    row.CapturedAt,
			FrameID:       row.ID,
			MovieID:       row.MovieID,
			Tags:          row.Tags,
			SchemaVersion: 1,
			ExportedAt:    exportedAt,
			AppName:       "Curated",
			AppVersion:    appVersion,
		}
		var outBytes []byte
		var fname string
		switch format {
		case "png":
			pngBytes, err := curatedexport.EncodePNGWithCuratedMetaITxt(row.ImageBlob, meta)
			if err != nil {
				h.logger.Error("curated frame png itxt", zap.String("id", row.ID), zap.Error(err))
				writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid or unsupported frame image")
				return
			}
			outBytes = pngBytes
			fname = curatedexport.ExportPNGFilename(actorForName, row.Code, row.PositionSec, row.ID, usedNames)
		default:
			webpBytes, err := curatedexport.EncodePNGToWebP(row.ImageBlob, meta, 82)
			if err != nil {
				h.logger.Error("curated frame webp encode", zap.String("id", row.ID), zap.Error(err))
				writeAppError(w, http.StatusBadRequest, contracts.ErrorCodeBadRequest, "invalid or unsupported frame image")
				return
			}
			outBytes = webpBytes
			fname = curatedexport.ExportWebPFilename(actorForName, row.Code, row.PositionSec, row.ID, usedNames)
		}
		files = append(files, outFile{name: fname, data: outBytes})
	}

	if len(files) == 1 {
		ct := "image/webp"
		fallback := "curated-export.webp"
		if format == "png" {
			ct = "image/png"
			fallback = "curated-export.png"
		}
		w.Header().Set("Content-Type", ct)
		w.Header().Set("Content-Disposition", contentDispositionAttachment(files[0].name, fallback))
		_, _ = w.Write(files[0].data)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", contentDispositionAttachment("curated-frames-export.zip", "curated-frames-export.zip"))

	zw := zip.NewWriter(w)
	for _, f := range files {
		hdr := &zip.FileHeader{Name: f.name, Method: zip.Deflate}
		wr, err := zw.CreateHeader(hdr)
		if err != nil {
			h.logger.Error("zip create", zap.Error(err))
			writeAppError(w, http.StatusInternalServerError, contracts.ErrorCodeInternal, "failed to build zip")
			return
		}
		if _, err := io.Copy(wr, bytes.NewReader(f.data)); err != nil {
			h.logger.Error("zip write", zap.Error(err))
			return
		}
	}
	if err := zw.Close(); err != nil {
		h.logger.Error("zip close", zap.Error(err))
	}
}
