// Package subtitles owns durable subtitle generation; dialogue stays in temporary checkpoints.
package subtitles

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const ASRModel = "qwen-audio-3.1-asr-flash-filetrans"
const ASRModel3 = "qwen3-asr-flash-filetrans"
const bailianBase = "https://dashscope.aliyuncs.com/api/v1"

type Cue struct {
	ID          int    `json:"id"`
	Begin       int64  `json:"beginMs"`
	End         int64  `json:"endMs"`
	Text        string `json:"text"`
	Translation string `json:"translation,omitempty"`
}
type Provider interface {
	Upload(context.Context, string, string, ...string) (string, error)
	Submit(context.Context, string, string, ...string) (string, error)
	Result(context.Context, string, string) ([]Cue, bool, error)
}
type RecognitionTaskFailed struct{ Status string }

func (e *RecognitionTaskFailed) Error() string { return "recognition task " + e.Status }

type Bailian struct {
	Client  *http.Client
	BaseURL string
}

// TestPolicy validates the saved key's permission for this ASR model without uploading audio.
func (p *Bailian) TestPolicy(ctx context.Context, key string, models ...string) error {
	model, err := recognitionModel(models)
	if err != nil {
		return err
	}
	var result struct {
		Data struct {
			Host string `json:"upload_host"`
		} `json:"data"`
	}
	if key == "" {
		return fmt.Errorf("Bailian API key is not configured")
	}
	if e := p.json(ctx, "GET", p.base()+"/uploads?action=getPolicy&model="+model, key, nil, &result, false); e != nil {
		return e
	}
	if !supplierURL(result.Data.Host) {
		return fmt.Errorf("upload policy unavailable for ASR model")
	}
	return nil
}

func (p *Bailian) base() string {
	if p.BaseURL != "" {
		return p.BaseURL
	}
	return bailianBase
}
func supplierURL(raw string) bool {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return false
	}
	h := strings.ToLower(u.Hostname())
	return strings.HasSuffix(h, ".aliyuncs.com") || strings.HasSuffix(h, ".alibabacloud.com")
}

// Qwen3 may return an HTTP OSS result URL. Upgrade only the scheme before
// applying the existing supplier checks; preserve the signed path and query.
func transcriptURL(raw string) (string, bool) {
	if strings.HasPrefix(raw, "http://") {
		raw = "https://" + strings.TrimPrefix(raw, "http://")
	}
	return raw, supplierURL(raw)
}
func (p *Bailian) client() *http.Client {
	c := http.Client{Timeout: 3 * time.Minute}
	if p.Client != nil {
		c = *p.Client
	}
	c.CheckRedirect = func(r *http.Request, via []*http.Request) error {
		if len(via) > 3 || !supplierURL(r.URL.String()) {
			return fmt.Errorf("subtitle provider redirect rejected")
		}
		return nil
	}
	return &c
}
func (p *Bailian) json(ctx context.Context, method, raw, key string, body any, out any, async bool) error {
	var reader io.Reader
	if body != nil {
		b, e := json.Marshal(body)
		if e != nil {
			return e
		}
		reader = bytes.NewReader(b)
	}
	req, e := http.NewRequestWithContext(ctx, method, raw, reader)
	if e != nil {
		return fmt.Errorf("invalid subtitle provider request")
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	// OSS signed GET URLs include content headers in their signature. A JSON
	// Content-Type on a bodyless result download invalidates that signature.
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Curated-Subtitles/1.0")
	if async {
		req.Header.Set("X-DashScope-Async", "enable")
		req.Header.Set("X-DashScope-OssResourceResolve", "enable")
	}
	resp, e := p.client().Do(req)
	if e != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("subtitle provider connection failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var failure struct {
			Code string `json:"code"`
		}
		_ = json.NewDecoder(io.LimitReader(resp.Body, 32768)).Decode(&failure)
		code := failure.Code
		if len(code) > 100 || strings.ContainsAny(code, "\r\n:/@? ") {
			code = ""
		}
		return fmt.Errorf("subtitle provider HTTP %d (%s)", resp.StatusCode, code)
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, 16*1024*1024+1))
	if e != nil {
		return e
	}
	if len(b) > 16*1024*1024 {
		return fmt.Errorf("subtitle response too large")
	}
	return json.Unmarshal(b, out)
}
func (p *Bailian) Upload(ctx context.Context, key, path string, models ...string) (string, error) {
	model, err := recognitionModel(models)
	if err != nil {
		return "", err
	}
	var policy struct {
		Data struct {
			Dir       string `json:"upload_dir"`
			Access    string `json:"oss_access_key_id"`
			Signature string `json:"signature"`
			Policy    string `json:"policy"`
			ACL       string `json:"x_oss_object_acl"`
			Forbid    string `json:"x_oss_forbid_overwrite"`
			Host      string `json:"upload_host"`
		} `json:"data"`
	}
	if e := p.json(ctx, "GET", p.base()+"/uploads?action=getPolicy&model="+model, key, nil, &policy, false); e != nil {
		return "", e
	}
	d := policy.Data
	if !supplierURL(d.Host) || d.Dir == "" {
		return "", fmt.Errorf("invalid upload policy")
	}
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	object := strings.TrimSuffix(d.Dir, "/") + "/" + newID() + "-" + filepath.Base(path)
	rd, wr := io.Pipe()
	mw := multipart.NewWriter(wr)
	req, e := http.NewRequestWithContext(ctx, "POST", d.Host, rd)
	if e != nil {
		return "", e
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	done := make(chan error, 1)
	go func() {
		var err error
		defer func() { _ = wr.CloseWithError(err); done <- err }()
		for k, v := range map[string]string{"OSSAccessKeyId": d.Access, "Signature": d.Signature, "policy": d.Policy, "x-oss-object-acl": d.ACL, "x-oss-forbid-overwrite": d.Forbid, "key": object, "success_action_status": "200"} {
			if err = mw.WriteField(k, v); err != nil {
				return
			}
		}
		var part io.Writer
		part, err = mw.CreateFormFile("file", filepath.Base(path))
		if err != nil {
			return
		}
		_, err = io.Copy(part, f)
		if err == nil {
			err = mw.Close()
		}
	}()
	resp, e := p.client().Do(req)
	_ = rd.Close()
	writeErr := <-done
	if e != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("audio upload connection failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("audio upload HTTP %d", resp.StatusCode)
	}
	if writeErr != nil {
		return "", writeErr
	}
	return "oss://" + object, nil
}
func (p *Bailian) Submit(ctx context.Context, key, uploaded string, models ...string) (string, error) {
	model, err := recognitionModel(models)
	if err != nil {
		return "", err
	}
	var result struct {
		Output struct {
			ID string `json:"task_id"`
		} `json:"output"`
	}
	body := map[string]any{"model": model, "input": map[string]any{"file_urls": []string{uploaded}}, "parameters": map[string]any{"channel_id": []int{0}, "language_hints": []string{"ja"}}}
	if model == ASRModel3 {
		body["input"] = map[string]any{"file_url": uploaded}
		body["parameters"] = map[string]any{"channel_id": []int{0}, "language": "ja", "enable_words": false}
	}
	e := p.json(ctx, "POST", p.base()+"/services/audio/asr/transcription", key, body, &result, true)
	if e != nil {
		return "", e
	}
	if result.Output.ID == "" {
		return "", fmt.Errorf("recognition submission result unknown")
	}
	return result.Output.ID, nil
}
func (p *Bailian) Result(ctx context.Context, key, id string) ([]Cue, bool, error) {
	var result struct {
		Output struct {
			Status string `json:"task_status"`
			Result struct {
				URL string `json:"transcription_url"`
			} `json:"result"`
			Results []struct {
				Status string `json:"subtask_status"`
				URL    string `json:"transcription_url"`
			} `json:"results"`
		} `json:"output"`
	}
	if e := p.json(ctx, "GET", p.base()+"/tasks/"+url.PathEscape(id), key, nil, &result, false); e != nil {
		return nil, false, fmt.Errorf("recognition status: %w", e)
	}
	switch result.Output.Status {
	case "PENDING", "RUNNING":
		return nil, false, nil
	case "SUCCEEDED":
	case "FAILED", "CANCELED":
		return nil, false, &RecognitionTaskFailed{Status: result.Output.Status}
	default:
		return nil, false, fmt.Errorf("recognition task %s", result.Output.Status)
	}
	if result.Output.Result.URL != "" {
		result.Output.Results = append(result.Output.Results, struct {
			Status string `json:"subtask_status"`
			URL    string `json:"transcription_url"`
		}{Status: "SUCCEEDED", URL: result.Output.Result.URL})
	}
	cues := []Cue{}
	for _, r := range result.Output.Results {
		resultURL, validURL := transcriptURL(r.URL)
		if r.Status != "SUCCEEDED" || !validURL {
			return nil, false, fmt.Errorf("recognition result unavailable")
		}
		var transcript struct {
			Transcripts []struct {
				Sentences []struct {
					Begin int64  `json:"begin_time"`
					End   int64  `json:"end_time"`
					Text  string `json:"text"`
				} `json:"sentences"`
			} `json:"transcripts"`
		}
		if e := p.json(ctx, "GET", resultURL, "", nil, &transcript, false); e != nil {
			return nil, false, fmt.Errorf("transcript download: %w", e)
		}
		for _, t := range transcript.Transcripts {
			for _, s := range t.Sentences {
				if s.End > s.Begin && s.Begin >= 0 && strings.TrimSpace(s.Text) != "" {
					cues = append(cues, Cue{Begin: s.Begin, End: s.End, Text: strings.TrimSpace(s.Text)})
				}
			}
		}
	}
	return cues, true, nil
}

func recognitionModel(models []string) (string, error) {
	model := ASRModel
	if len(models) > 0 && models[0] != "" {
		model = models[0]
	}
	if model != ASRModel && model != ASRModel3 {
		return "", fmt.Errorf("unsupported recognition model")
	}
	return model, nil
}
