package copyleaks

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"
)

func TestSubmitContentIsEncodedLikeBase64(t *testing.T) {
	content := []byte("Some <text> & more, with \"quotes\".\n")
	b64 := base64.StdEncoding.EncodeToString(content)
	props := ScanProperties{Webhooks: SubmitWebhooks{Status: "https://example.com/webhook/{STATUS}"}, Sandbox: true, DeveloperPayload: "<a & b>"}
	want := `{"base64":"` + b64 + `","filename":"a.txt","properties":{"webhooks":{"status":"https://example.com/webhook/{STATUS}"},"sandbox":true,"developerPayload":"<a & b>"}}`

	for name, req := range map[string]*FileSubmission{
		"Content": {Content: content, Filename: "a.txt", Properties: props},
		"Base64":  {Base64: b64, Filename: "a.txt", Properties: props},
	} {
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t, http.StatusCreated, "")
			if err := env.client(t).SubmitFile(context.Background(), testToken(), "scan-1", req); err != nil {
				t.Fatal(err)
			}
			got := env.next(t)
			if string(got.body) != want {
				t.Errorf("body\n got: %s\nwant: %s", got.body, want)
			}
			if got.header.Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type = %q", got.header.Get("Content-Type"))
			}
		})
	}
	// The request is not changed.
	req := &FileSubmission{Content: content, Filename: "a.txt", Properties: props}
	env := newTestEnv(t, http.StatusCreated, "")
	if err := env.client(t).SubmitFile(context.Background(), testToken(), "scan-1", req); err != nil {
		t.Fatal(err)
	}
	if req.Base64 != "" || !bytes.Equal(req.Content, content) {
		t.Errorf("SubmitFile changed the request: %+v", req)
	}
}

func TestSubmitOCRContent(t *testing.T) {
	env := newTestEnv(t, http.StatusCreated, "")
	req := &OCRSubmission{Content: []byte{0x89, 'P', 'N', 'G'}, Filename: "a.png", LangCode: "en", Properties: validSubmitProps}
	if err := env.client(t).SubmitOCR(context.Background(), testToken(), "scan-1", req); err != nil {
		t.Fatal(err)
	}
	got := env.next(t)
	if got.method != http.MethodPut || got.path != "/v3/scans/submit/ocr/scan-1" {
		t.Errorf("request = %s %s", got.method, got.path)
	}
	assertJSONEqual(t, got.body, `{"base64":"iVBORw==","filename":"a.png","langCode":"en","properties":{"webhooks":{"status":"https://example.com/webhook/{STATUS}"},"sandbox":true}}`)
}

func TestUploadNeedsBase64OrContent(t *testing.T) {
	ctx := context.Background()
	calls := map[string]func(c *Client, b64 string, content []byte) error{
		"SubmitFile": func(c *Client, b64 string, content []byte) error {
			return c.SubmitFile(ctx, testToken(), "scan-1", &FileSubmission{Base64: b64, Content: content, Filename: "a.txt", Properties: validSubmitProps})
		},
		"SubmitOCR": func(c *Client, b64 string, content []byte) error {
			return c.SubmitOCR(ctx, testToken(), "scan-1", &OCRSubmission{Base64: b64, Content: content, Filename: "a.png", LangCode: "en", Properties: validSubmitProps})
		},
	}
	for name, call := range calls {
		for _, tt := range []struct {
			b64     string
			content []byte
		}{{"", nil}, {"", []byte{}}, {"aGk=", []byte("hi")}} {
			env := newTestEnv(t, http.StatusCreated, "")
			if err := call(env.client(t), tt.b64, tt.content); !errors.Is(err, ErrInvalidArgument) {
				t.Errorf("%s(%q, %q): err = %v, want ErrInvalidArgument", name, tt.b64, tt.content, err)
			}
			env.none(t)
		}
		// An empty, non-nil Content with Base64 uses Base64.
		env := newTestEnv(t, http.StatusCreated, "")
		if err := call(env.client(t), "aGk=", []byte{}); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if body := env.next(t).body; !bytes.HasPrefix(body, []byte(`{"base64":"aGk="`)) {
			t.Errorf("%s: body %s", name, body)
		}
	}
}

func TestBase64BodyQuotesTextThatIsNotBase64(t *testing.T) {
	env := newTestEnv(t, http.StatusCreated, "")
	req := &FileSubmission{Base64: "not \"base64\" \\ é\n", Filename: "a.txt", Properties: validSubmitProps}
	if err := env.client(t).SubmitFile(context.Background(), testToken(), "scan-1", req); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(env.next(t).body, &got); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if got["base64"] != req.Base64 || got["filename"] != "a.txt" {
		t.Errorf("body = %v", got)
	}
}

func TestRequestBodiesDoNotEscapeHTML(t *testing.T) {
	text := lionsText + " <b>bold</b> & more"
	env := newTestEnv(t, http.StatusOK, "{}")
	if _, err := env.client(t).DetectAIText(context.Background(), testToken(), "scan-1", &AITextDetectionRequest{Text: text}); err != nil {
		t.Fatal(err)
	}
	body := string(env.next(t).body)
	if !strings.Contains(body, "<b>bold</b> & more") || strings.Contains(body, `\u003c`) {
		t.Errorf("body escapes HTML: %s", body)
	}
}

// discardTransport reads and discards each request body and answers 201 without a
// body, without the allocations of a real connection.
type discardTransport struct{}

func (discardTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Body != nil {
		_, _ = io.Copy(io.Discard, r.Body)
		r.Body.Close()
	}
	return &http.Response{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader("")), Header: http.Header{}, Request: r}, nil
}

// allocated returns the bytes that f allocates on the heap.
func allocated(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	f()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

// largeFile returns n bytes of a file to upload.
func largeFile(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i * 7)
	}
	return b
}

func TestLargeUploadMemory(t *testing.T) {
	const size = 16 << 20
	file := largeFile(size)
	b64 := base64.StdEncoding.EncodeToString(file)
	c := mustClient(t, WithHTTPClient(&http.Client{Transport: discardTransport{}}))
	ctx := context.Background()
	tok := testToken()

	tests := []struct {
		name string
		// budget is the most that the call may allocate, as a multiple of the file size.
		budget float64
		call   func() error
	}{
		{"SubmitFile with Content", 1.45, func() error {
			return c.SubmitFile(ctx, tok, "scan-1", &FileSubmission{Content: file, Filename: "a.pdf", Properties: validSubmitProps})
		}},
		{"SubmitOCR with Content", 1.45, func() error {
			return c.SubmitOCR(ctx, tok, "scan-1", &OCRSubmission{Content: file, Filename: "a.png", LangCode: "en", Properties: validSubmitProps})
		}},
		{"SubmitFile with Base64", 1.45, func() error {
			return c.SubmitFile(ctx, tok, "scan-1", &FileSubmission{Base64: b64, Filename: "a.pdf", Properties: validSubmitProps})
		}},
		{"DetectAIImage with Base64", 1.45, func() error {
			_, err := c.DetectAIImage(ctx, tok, "scan-1", &AIImageDetectionRequest{Base64: b64, Filename: "a.png", Model: AIImageModel1Ultra})
			return ignoreDecode(err)
		}},
		{"DetectAIImageUpload", 1.1, func() error {
			_, err := c.DetectAIImageUpload(ctx, tok, "scan-1", &AIImageUploadRequest{Image: file, Filename: "a.png", Model: AIImageModel1Ultra})
			return ignoreDecode(err)
		}},
		{"CheckImagePlagiarism", 1.1, func() error {
			_, err := c.CheckImagePlagiarism(ctx, tok, "scan-1", &ImagePlagiarismRequest{Image: file, Filename: "a.png"})
			return ignoreDecode(err)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			n := allocated(func() { err = tt.call() })
			if err != nil {
				t.Fatal(err)
			}
			ratio := float64(n) / float64(size)
			t.Logf("allocated %.1f MiB for a %d MiB file: %.2fx", float64(n)/(1<<20), size>>20, ratio)
			if ratio > tt.budget {
				t.Errorf("allocated %.2fx the file size, want at most %.2fx", ratio, tt.budget)
			}
		})
	}

	// For comparison, encoding the request with encoding/json, as 1.0.0 did.
	n := allocated(func() {
		_, _ = json.Marshal(&FileSubmission{Base64: b64, Filename: "a.pdf", Properties: validSubmitProps})
	})
	t.Logf("json.Marshal of the same request allocates %.2fx the file size", float64(n)/float64(size))
}

// ignoreDecode drops the decode error of a call whose test transport answers without a
// body.
func ignoreDecode(err error) error {
	if err != nil && strings.Contains(err.Error(), "copyleaks: decode") {
		return nil
	}
	return err
}

func BenchmarkSubmitFile(b *testing.B) {
	const size = 8 << 20
	file := largeFile(size)
	b64 := base64.StdEncoding.EncodeToString(file)
	c, err := NewClient(WithHTTPClient(&http.Client{Transport: discardTransport{}}))
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	tok := testToken()
	b.Run("Content", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(size)
		for i := 0; i < b.N; i++ {
			if err := c.SubmitFile(ctx, tok, "scan-1", &FileSubmission{Content: file, Filename: "a.pdf", Properties: validSubmitProps}); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("Base64", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(size)
		for i := 0; i < b.N; i++ {
			if err := c.SubmitFile(ctx, tok, "scan-1", &FileSubmission{Base64: b64, Filename: "a.pdf", Properties: validSubmitProps}); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("json.Marshal (1.0.0)", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(size)
		for i := 0; i < b.N; i++ {
			if _, err := json.Marshal(&FileSubmission{Base64: b64, Filename: "a.pdf", Properties: validSubmitProps}); err != nil {
				b.Fatal(err)
			}
		}
	})
}
