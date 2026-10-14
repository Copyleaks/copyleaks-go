package copyleaks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// newEndpointCases covers the endpoints and request fields added in 1.1.0.
func newEndpointCases(t *testing.T) []endpointCase {
	tok := testToken()
	return []endpointCase{
		{
			name:   "SubmitAIImageAsync",
			method: http.MethodPut,
			path:   "/v1/ai-image-detector-async/scan-1/submit",
			auth:   true,
			body: `{"url":"https://example.com/image1.png","fileName":"image1.png","model":"ai-image-1-ultra","maskType":"heatmap",` +
				`"objectDetection":[{"object":"face"},{"object":"hand"}],"headers":[["X-Custom-Auth","my-token"]],` +
				`"webhooks":{"url":"https://example.com/webhook/{STATUS}","headers":[["Authorization","Bearer hook"]]},` +
				`"sandbox":true,"developerPayload":"order-1"}`,
			status: http.StatusCreated,
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.SubmitAIImageAsync(ctx, tok, "scan-1", &AIImageAsyncSubmission{
					URL:              "https://example.com/image1.png",
					FileName:         "image1.png",
					Model:            AIImageModel1Ultra,
					MaskType:         AIImageMaskHeatmap,
					ObjectDetection:  []AIImageObject{{Object: "face"}, {Object: "hand"}},
					Headers:          [][]string{{"X-Custom-Auth", "my-token"}},
					Webhooks:         ResultWebhook{URL: "https://example.com/webhook/{STATUS}", Headers: [][]string{{"Authorization", "Bearer hook"}}},
					Sandbox:          true,
					DeveloperPayload: "order-1",
				})
			},
		},
		{
			name:   "SubmitAIImageAsync minimal",
			method: http.MethodPut,
			path:   "/v1/ai-image-detector-async/scan-1/submit",
			auth:   true,
			body:   `{"url":"https://example.com/a.png","fileName":"a.png","model":"ai-image-1-ultra","webhooks":{"url":"https://example.com/hook"}}`,
			status: http.StatusCreated,
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.SubmitAIImageAsync(ctx, tok, "scan-1", &AIImageAsyncSubmission{
					URL: "https://example.com/a.png", FileName: "a.png", Model: AIImageModel1Ultra, Webhooks: ResultWebhook{URL: "https://example.com/hook"},
				})
			},
		},
		{
			name:   "SubmitAIVideo",
			method: http.MethodPost,
			path:   "/v1/ai-video-detector/my-video-scan-1/submit",
			auth:   true,
			body: `{"url":"https://example.com/my-video.mp4","headers":[["X-Custom-Auth","my-token"]],"verb":"POST",` +
				`"filename":"my-video.mp4","model":"ai-video-1-pro","webhooks":{"url":"https://example.com/hook","headers":[["k","v"]]},` +
				`"sandbox":true,"developerPayload":"order-id-12345"}`,
			status: http.StatusCreated,
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.SubmitAIVideo(ctx, tok, "my-video-scan-1", &AIVideoSubmission{
					URL:              "https://example.com/my-video.mp4",
					Headers:          [][]string{{"X-Custom-Auth", "my-token"}},
					Verb:             http.MethodPost,
					Filename:         "my-video.mp4",
					Model:            AIVideoModel1Pro,
					Webhooks:         ResultWebhook{URL: "https://example.com/hook", Headers: [][]string{{"k", "v"}}},
					Sandbox:          true,
					DeveloperPayload: "order-id-12345",
				})
			},
		},
		{
			name:   "SubmitAIVideo minimal",
			method: http.MethodPost,
			path:   "/v1/ai-video-detector/scan-1/submit",
			auth:   true,
			body:   `{"url":"https://example.com/v.mp4","filename":"v.mp4","model":"ai-video-1-pro","webhooks":{"url":"https://example.com/hook"}}`,
			status: http.StatusCreated,
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.SubmitAIVideo(ctx, tok, "scan-1", &AIVideoSubmission{
					URL: "https://example.com/v.mp4", Filename: "v.mp4", Model: AIVideoModel1Pro, Webhooks: ResultWebhook{URL: "https://example.com/hook"},
				})
			},
		},
		{
			name:     "AITextCreditsBalance",
			method:   http.MethodGet,
			path:     "/v2/writer-detector/credits",
			auth:     true,
			status:   http.StatusOK,
			response: `{"credits":382}`,
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.AITextCreditsBalance(ctx, tok)
			},
			check: func(t *testing.T, got any) {
				if got.(*AITextCreditsBalance).Credits != 382 {
					t.Errorf("got %+v", got)
				}
			},
		},
		{
			name:     "AllowedCrossLanguages",
			method:   http.MethodGet,
			path:     "/v3/miscellaneous/allowed-cross-languages",
			status:   http.StatusOK,
			response: fixture(t, "allowed_cross_languages.json"),
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.AllowedCrossLanguages(ctx)
			},
			check: func(t *testing.T, got any) {
				res := got.(*AllowedCrossLanguages)
				if res.DocumentLanguages["english"] != "en" || len(res.ResultLanguages) != 4 {
					t.Errorf("got %+v", res)
				}
				if codes := res.DocumentLanguages.Codes(); !reflect.DeepEqual(codes, []string{"da", "en", "es"}) {
					t.Errorf("Codes() = %v", codes)
				}
			},
		},
		{
			name:     "SupportedAITextFileTypes",
			method:   http.MethodGet,
			path:     "/v3/miscellaneous/supported-ai-text-detection-file-types",
			status:   http.StatusOK,
			response: `{"supportedAiFileTypes":["pdf","docx","LaTeX"]}`,
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.SupportedAITextFileTypes(ctx)
			},
			check: func(t *testing.T, got any) {
				if want := []string{"pdf", "docx", "LaTeX"}; !reflect.DeepEqual(got.(*SupportedAITextFileTypes).FileTypes, want) {
					t.Errorf("got %+v", got)
				}
			},
		},
		{
			name:     "RepositoryInfo",
			method:   http.MethodGet,
			path:     "/v3/repositories/repository/my-repo-123/info",
			auth:     true,
			status:   http.StatusOK,
			response: fixture(t, "repository_info.json"),
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.RepositoryInfo(ctx, tok, "my-repo-123")
			},
			check: func(t *testing.T, got any) {
				want := &RepositoryInfo{
					ID: "private-data-hub-id", Name: "Private Data Hub Name", Description: "Your Description",
					Permission: RepositoryRoleSuperAdmin, Status: RepositoryStatusRunning, MaxCredits: 1000, CurrentCredits: 900,
					MaskingPolicy: MaskingPolicyMaskOtherUsersFiles, CreationTime: "2024-09-09T10:43:52",
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("got %+v, want %+v", got, want)
				}
			},
		},
		{
			name:     "RepositoryInfo with nulls",
			method:   http.MethodGet,
			path:     "/v3/repositories/repository/repo_1/info",
			auth:     true,
			status:   http.StatusOK,
			response: `{"id":"repo_1","description":null,"maxCredits":null,"currentCredits":null,"creationTime":null,"permission":3,"status":4}`,
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.RepositoryInfo(ctx, tok, "repo_1")
			},
			check: func(t *testing.T, got any) {
				want := &RepositoryInfo{ID: "repo_1", Permission: RepositoryRoleAdmin, Status: RepositoryStatusPaymentRequired}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("got %+v, want %+v", got, want)
				}
			},
		},
		{
			name:   "DetectAIImage mask and objects",
			method: http.MethodPost,
			path:   "/v1/ai-image-detector/scan-1/check",
			auth:   true,
			body: `{"base64":"iVBORw0KGgo=","filename":"image.png","model":"ai-image-1-ultra","sandbox":true,"maskType":"heatmap",` +
				`"objectDetection":[{"object":"face"},{"object":"hand"}]}`,
			status:   http.StatusOK,
			response: fixture(t, "ai_image_heatmap_response.json"),
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.DetectAIImage(ctx, tok, "scan-1", &AIImageDetectionRequest{
					Base64: "iVBORw0KGgo=", Filename: "image.png", Model: AIImageModel1Ultra, Sandbox: true,
					MaskType: AIImageMaskHeatmap, ObjectDetection: &[]AIImageObject{{Object: "face"}, {Object: "hand"}},
				})
			},
			check: func(t *testing.T, got any) {
				checkHeatmapResponse(t, got.(*AIImageDetectionResponse))
			},
		},
		{
			name:     "DetectAIImage without a mask",
			method:   http.MethodPost,
			path:     "/v1/ai-image-detector/scan-1/check",
			auth:     true,
			body:     `{"base64":"iVBORw0KGgo=","filename":"image.png","model":"ai-image-1-ultra","maskType":"none"}`,
			status:   http.StatusOK,
			response: `{"model":"ai-image-1-ultra","summary":{"human":1,"ai":0},"isAiDetected":false}`,
			call: func(ctx context.Context, c *Client) (any, error) {
				return c.DetectAIImage(ctx, tok, "scan-1", &AIImageDetectionRequest{Base64: "iVBORw0KGgo=", Filename: "image.png", Model: AIImageModel1Ultra, MaskType: AIImageMaskNone})
			},
			check: func(t *testing.T, got any) {
				if res := got.(*AIImageDetectionResponse); res.Result != nil || res.DetectedObjects != nil || res.Summary.Human != 1 {
					t.Errorf("got %+v", res)
				}
			},
		},
		{
			name:   "SubmitURL verb and headers",
			method: http.MethodPut,
			path:   "/v3/scans/submit/url/scan-1",
			auth:   true,
			body: `{"url":"https://example.com/form","properties":{"webhooks":{"status":"https://example.com/webhook/{STATUS}"},"sandbox":true},` +
				`"verb":"POST","headers":[["Set-Cookie","session=1"],["X-Key","v"]]}`,
			status: http.StatusCreated,
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.SubmitURL(ctx, tok, "scan-1", &URLSubmission{
					URL: "https://example.com/form", Properties: validSubmitProps, Verb: http.MethodPost,
					Headers: [][]string{{"Set-Cookie", "session=1"}, {"X-Key", "v"}},
				})
			},
		},
		{
			name:   "Export new targets",
			method: http.MethodPost,
			path:   "/v3/downloads/scan-1/export/export-1",
			auth:   true,
			body: `{"completionWebhook":"https://example.com/export/done","completionWebhookHeaders":[["Authorization","Bearer x"]],` +
				`"aiDetection":{"endpoint":"https://example.com/ai","verb":"PUT","headers":[["k","v"]]},` +
				`"writingFeedback":{"endpoint":"https://example.com/wf","verb":"PUT"},` +
				`"overview":{"endpoint":"https://example.com/ov","verb":"POST"}}`,
			status: http.StatusNoContent,
			call: func(ctx context.Context, c *Client) (any, error) {
				return nil, c.Export(ctx, tok, "scan-1", "export-1", &ExportRequest{
					CompletionWebhook:        "https://example.com/export/done",
					CompletionWebhookHeaders: [][]string{{"Authorization", "Bearer x"}},
					AIDetection:              &ExportTarget{Endpoint: "https://example.com/ai", Verb: "PUT", Headers: [][]string{{"k", "v"}}},
					WritingFeedback:          &ExportTarget{Endpoint: "https://example.com/wf", Verb: "PUT"},
					Overview:                 &ExportTarget{Endpoint: "https://example.com/ov", Verb: "POST"},
				})
			},
		},
	}
}

func TestNewEndpointRequestsAndResponses(t *testing.T) {
	for _, tc := range newEndpointCases(t) {
		t.Run(tc.name, func(t *testing.T) { runEndpointCase(t, tc) })
	}
}

func checkHeatmapResponse(t *testing.T, res *AIImageDetectionResponse) {
	t.Helper()
	if !reflect.DeepEqual(res.Result.Labels, []int{1, 2, 4, 3}) || !reflect.DeepEqual(res.Result.Starts, []int{0, 512, 1536, 2560}) {
		t.Errorf("result: %+v", res.Result)
	}
	want := []AIImageDetectedObject{
		{Object: "face", Mask: &AIImageDetectionResult{Starts: []int{1024, 3072}, Lengths: []int{512, 256}}},
		{Object: "hand", Mask: &AIImageDetectionResult{Starts: []int{8192}, Lengths: []int{128}}},
	}
	if res.DetectedObjects == nil || !reflect.DeepEqual(*res.DetectedObjects, want) {
		t.Errorf("detectedObjects: %+v", res.DetectedObjects)
	}
}

func TestSubmitNewScanProperties(t *testing.T) {
	props := ScanProperties{
		Webhooks:                SubmitWebhooks{Status: "https://example.com/webhook/{STATUS}"},
		Sandbox:                 true,
		ScanTimeZone:            "America/New_York",
		Course:                  &Course{ID: "course-1"},
		Assignment:              &Assignment{ID: "assignment-1"},
		Institution:             &Institution{ID: "institution-1"},
		SensitiveDataProtection: &SensitiveDataProtection{DriversLicense: true, Credentials: true, Passport: true, Network: true, URL: true, EmailAddress: true, CreditCard: true, PhoneNumber: true},
		WritingFeedback:         &ScanWritingFeedback{Enable: true, Score: &GrammarScoreWeights{GrammarScoreWeight: Ptr(0.5), WordChoiceScoreWeight: Ptr(0.0)}},
		Overview:                &ScanOverview{Enable: true, IgnoreAIDetection: true, IgnorePlagiarismDetection: true, IgnoreWritingFeedback: true, IgnoreAuthorData: true},
		InternalAISourceMatch:   &InternalAISourceMatch{Enable: true},
		References:              &References{Validate: true},
		AIGeneratedText:         &AIGeneratedText{Detect: true, Sensitivity: 3, Explain: &AIGeneratedTextExplain{Enable: true}},
		Scanning: &Scanning{
			Exclude: &ScanningExclude{IDPattern: "abc*", BacklinksDomains: &[]string{"example.com"}, Text: &[]string{"lorem ipsum"}},
			Include: &ScanningInclude{IDPattern: "course-1-*"},
		},
		Indexing: &Indexing{CopyleaksDB: Ptr(true)},
		PDF:      &PDF{Create: true, Colors: &PDFColors{AIDetection: "#FE5758", WritingFeedback: "#FF9A02"}},
	}
	want := `{"webhooks":{"status":"https://example.com/webhook/{STATUS}"},"sandbox":true,` +
		`"scanning":{"exclude":{"idPattern":"abc*","backlinksDomains":["example.com"],"text":["lorem ipsum"]},"include":{"idPattern":"course-1-*"}},` +
		`"indexing":{"copyleaksDb":true},` +
		`"pdf":{"create":true,"colors":{"aiDetection":"#FE5758","writingFeedback":"#FF9A02"}},` +
		`"aiGeneratedText":{"detect":true,"sensitivity":3,"explain":{"enable":true}},` +
		`"scanTimeZone":"America/New_York","course":{"id":"course-1"},"assignment":{"id":"assignment-1"},"institution":{"id":"institution-1"},` +
		`"sensitiveDataProtection":{"driversLicense":true,"credentials":true,"passport":true,"network":true,"url":true,"emailAddress":true,"creditCard":true,"phoneNumber":true},` +
		`"writingFeedback":{"enable":true,"score":{"grammarScoreWeight":0.5,"wordChoiceScoreWeight":0}},` +
		`"overview":{"enable":true,"ignoreAIDetection":true,"ignorePlagiarismDetection":true,"ignoreWritingFeedback":true,"ignoreAuthorData":true},` +
		`"internalAiSourceMatch":{"enable":true},"references":{"validate":true}}`
	env := newTestEnv(t, http.StatusCreated, "")
	req := &FileSubmission{Base64: "aGk=", Filename: "a.txt", Properties: props}
	if err := env.client(t).SubmitFile(context.Background(), testToken(), "scan-1", req); err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, env.next(t).body, `{"base64":"aGk=","filename":"a.txt","properties":`+want+`}`)

	// Explicitly disabled features are still sent, so that they override account defaults.
	env = newTestEnv(t, http.StatusCreated, "")
	props = ScanProperties{
		Webhooks:              SubmitWebhooks{Status: "https://example.com/webhook/{STATUS}"},
		WritingFeedback:       &ScanWritingFeedback{},
		Overview:              &ScanOverview{},
		InternalAISourceMatch: &InternalAISourceMatch{},
		References:            &References{},
		AIGeneratedText:       &AIGeneratedText{Explain: &AIGeneratedTextExplain{}},
		Indexing:              &Indexing{CopyleaksDB: Ptr(false)},
	}
	if err := env.client(t).SubmitURL(context.Background(), testToken(), "scan-1", &URLSubmission{URL: "https://example.com", Properties: props}); err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, env.next(t).body, `{"url":"https://example.com","properties":{"webhooks":{"status":"https://example.com/webhook/{STATUS}"},`+
		`"aiGeneratedText":{"detect":false,"explain":{"enable":false}},"writingFeedback":{"enable":false},"overview":{"enable":false},`+
		`"internalAiSourceMatch":{"enable":false},"references":{"validate":false},"indexing":{"copyleaksDb":false}}}`)
}

// multipartRequest is a multipart/form-data request that a test server received.
type multipartRequest struct {
	fields   map[string]string
	file     []byte
	fileName string
	fileType string
}

// parseMultipart decodes the multipart/form-data body of req, with its file field
// fileField, and fails the test when the body is not multipart/form-data.
func parseMultipart(t *testing.T, req captured, fileField string) multipartRequest {
	t.Helper()
	mediaType, params, err := mime.ParseMediaType(req.header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" || params["boundary"] == "" {
		t.Fatalf("Content-Type = %q, want multipart/form-data", req.header.Get("Content-Type"))
	}
	got := multipartRequest{fields: map[string]string{}}
	r := multipart.NewReader(bytes.NewReader(req.body), params["boundary"])
	for {
		part, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("bad multipart body: %v", err)
		}
		b, _ := io.ReadAll(part)
		if part.FormName() == fileField {
			got.file, got.fileName, got.fileType = b, part.FileName(), part.Header.Get("Content-Type")
			continue
		}
		if _, dup := got.fields[part.FormName()]; dup {
			t.Errorf("field %q sent twice", part.FormName())
		}
		got.fields[part.FormName()] = string(b)
	}
	return got
}

func TestCheckImagePlagiarism(t *testing.T) {
	image := []byte{0xff, 0xd8, 0xff, 0x00, '"', '\r', '\n', 0x01}
	tests := []struct {
		name   string
		req    *ImagePlagiarismRequest
		fields map[string]string
		ctype  string
	}{
		{"all fields", &ImagePlagiarismRequest{Image: image, Filename: "my-photo.jpg", Sandbox: true, DeveloperPayload: "my \"custom\" data"},
			map[string]string{"filename": "my-photo.jpg", "sandbox": "true", "developerPayload": "my \"custom\" data"}, "image/jpeg"},
		{"required fields", &ImagePlagiarismRequest{Image: image, Filename: "IMAGE.ICO"},
			map[string]string{"filename": "IMAGE.ICO"}, "image/x-icon"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t, http.StatusOK, fixture(t, "image_plagiarism_response.json"))
			res, err := env.client(t).CheckImagePlagiarism(context.Background(), testToken(), "my-scan-1", tt.req)
			if err != nil {
				t.Fatal(err)
			}
			req := env.next(t)
			if req.method != http.MethodPost || req.path != "/v1/image-plagiarism-detector/my-scan-1/check" {
				t.Errorf("request = %s %s", req.method, req.path)
			}
			if req.header.Get("Authorization") != "Bearer "+testAccessToken || req.header.Get("User-Agent") != "copyleaks-go-sdk/"+Version {
				t.Errorf("headers: %v", req.header)
			}
			form := parseMultipart(t, req, "image")
			if !bytes.Equal(form.file, image) || form.fileName != tt.req.Filename || form.fileType != tt.ctype {
				t.Errorf("image part: %q %q %q", form.file, form.fileName, form.fileType)
			}
			if !reflect.DeepEqual(form.fields, tt.fields) {
				t.Errorf("fields = %v, want %v", form.fields, tt.fields)
			}
			want := &ImagePlagiarismResponse{
				DeveloperPayload: "my-custom-data",
				ScannedImage:     &ScannedImage{ScanID: "my-scan-1", ExpectedCredits: 1, ActualCredits: 1, CreationTime: "2026-05-24T10:00:00Z", Width: 1920, Height: 1080, Filename: "my-photo.jpg"},
				Matches: &ImagePlagiarismMatches{
					Internet: []ImagePlagiarismMatch{
						{URL: "https://example.com/images/photo.jpg", MatchType: ImageMatchFull, WebPages: []ImagePlagiarismWebPage{{URL: "https://example.com/blog/my-post"}, {URL: "https://example.org/news/article"}}},
						{URL: "https://example.com/thumbs/photo-thumb.jpg", MatchType: ImageMatchPartial, WebPages: []ImagePlagiarismWebPage{{URL: "https://example.org/gallery"}}},
						{URL: "https://example.org/gallery/photo-sm.jpg", MatchType: ImageMatchPartial},
					},
					Score: &ImagePlagiarismScore{TotalMatches: 3, FullMatches: 1, PartialMatches: 2},
				},
			}
			if !reflect.DeepEqual(res, want) {
				t.Errorf("response = %+v", res)
			}
		})
	}
}

func TestDetectAIImageUpload(t *testing.T) {
	image := []byte{0x89, 'P', 'N', 'G', 0x00, 0xff}
	tests := []struct {
		name   string
		req    *AIImageUploadRequest
		fields map[string]string
	}{
		{"all fields", &AIImageUploadRequest{
			Image: image, Filename: "test-image.png", Model: AIImageModel1Ultra, Sandbox: true, MaskType: AIImageMaskHeatmap,
			ObjectDetection: []AIImageObject{{Object: "face"}, {Object: "hand"}},
		}, map[string]string{
			"filename": "test-image.png", "model": "ai-image-1-ultra", "sandbox": "true", "maskType": "heatmap",
			"objectDetection": `[{"object":"face"},{"object":"hand"}]`,
		}},
		{"required fields", &AIImageUploadRequest{Image: image, Filename: "test-image.png", Model: AIImageModel1Ultra},
			map[string]string{"filename": "test-image.png", "model": "ai-image-1-ultra"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t, http.StatusOK, fixture(t, "ai_image_heatmap_response.json"))
			res, err := env.client(t).DetectAIImageUpload(context.Background(), testToken(), "scan-1", tt.req)
			if err != nil {
				t.Fatal(err)
			}
			req := env.next(t)
			if req.method != http.MethodPost || req.path != "/v1/ai-image-detector/scan-1/check" || req.header.Get("Authorization") != "Bearer "+testAccessToken {
				t.Errorf("request = %s %s %v", req.method, req.path, req.header)
			}
			form := parseMultipart(t, req, "image")
			if !bytes.Equal(form.file, image) || form.fileName != "test-image.png" || form.fileType != "image/png" {
				t.Errorf("image part: %q %q %q", form.file, form.fileName, form.fileType)
			}
			if !reflect.DeepEqual(form.fields, tt.fields) {
				t.Errorf("fields = %v, want %v", form.fields, tt.fields)
			}
			checkHeatmapResponse(t, res)
		})
	}
}

func TestNewEndpointValidation(t *testing.T) {
	tok := testToken()
	ctx := context.Background()
	hook := ResultWebhook{URL: "https://example.com/hook"}
	video := func(mod func(r *AIVideoSubmission)) func(c *Client) error {
		return func(c *Client) error {
			r := &AIVideoSubmission{URL: "https://example.com/v.mp4", Filename: "v.mp4", Model: AIVideoModel1Pro, Webhooks: hook}
			mod(r)
			return c.SubmitAIVideo(ctx, tok, "scan-1", r)
		}
	}
	async := func(mod func(r *AIImageAsyncSubmission)) func(c *Client) error {
		return func(c *Client) error {
			r := &AIImageAsyncSubmission{URL: "https://example.com/a.png", FileName: "a.png", Model: AIImageModel1Ultra, Webhooks: hook}
			mod(r)
			return c.SubmitAIImageAsync(ctx, tok, "scan-1", r)
		}
	}
	image := func(mod func(r *AIImageDetectionRequest)) func(c *Client) error {
		return func(c *Client) error {
			r := &AIImageDetectionRequest{Base64: "aGk=", Filename: "a.png", Model: AIImageModel1Ultra}
			mod(r)
			_, err := c.DetectAIImage(ctx, tok, "scan-1", r)
			return err
		}
	}
	upload := func(mod func(r *AIImageUploadRequest)) func(c *Client) error {
		return func(c *Client) error {
			r := &AIImageUploadRequest{Image: []byte{1}, Filename: "a.png", Model: AIImageModel1Ultra}
			mod(r)
			_, err := c.DetectAIImageUpload(ctx, tok, "scan-1", r)
			return err
		}
	}
	plagiarism := func(mod func(r *ImagePlagiarismRequest)) func(c *Client) error {
		return func(c *Client) error {
			r := &ImagePlagiarismRequest{Image: []byte{1}, Filename: "a.gif"}
			mod(r)
			_, err := c.CheckImagePlagiarism(ctx, tok, "scan-1", r)
			return err
		}
	}
	objects := func(names ...string) []AIImageObject {
		var list []AIImageObject
		for _, n := range names {
			list = append(list, AIImageObject{Object: n})
		}
		return list
	}
	long := func(n int) string { return strings.Repeat("a", n) }
	tests := []struct {
		name  string
		call  func(c *Client) error
		valid bool
	}{
		{"video ok", video(func(r *AIVideoSubmission) {}), true},
		{"video verb lower case", video(func(r *AIVideoSubmission) { r.Verb = "put" }), true},
		{"video verb DELETE", video(func(r *AIVideoSubmission) { r.Verb = "DELETE" }), false},
		{"video .MXF", video(func(r *AIVideoSubmission) { r.Filename = "CLIP.MXF" }), true},
		{"video .gif", video(func(r *AIVideoSubmission) { r.Filename = "clip.gif" }), false},
		{"video 254 characters", video(func(r *AIVideoSubmission) { r.Filename = long(250) + ".mp4" }), true},
		{"video 255 characters", video(func(r *AIVideoSubmission) { r.Filename = long(251) + ".mp4" }), false},
		{"video payload 511", video(func(r *AIVideoSubmission) { r.DeveloperPayload = long(511) }), true},
		{"video payload 512", video(func(r *AIVideoSubmission) { r.DeveloperPayload = long(512) }), false},
		{"video no URL", video(func(r *AIVideoSubmission) { r.URL = "" }), false},
		{"video no model", video(func(r *AIVideoSubmission) { r.Model = "" }), false},
		{"video no webhook", video(func(r *AIVideoSubmission) { r.Webhooks.URL = "" }), false},
		{"video nil", func(c *Client) error { return c.SubmitAIVideo(ctx, tok, "scan-1", nil) }, false},
		{"video short scan ID", func(c *Client) error {
			return c.SubmitAIVideo(ctx, tok, "ab", &AIVideoSubmission{URL: "u", Filename: "v.mp4", Model: "m", Webhooks: hook})
		}, false},
		{"async ok", async(func(r *AIImageAsyncSubmission) {}), true},
		{"async .avif", async(func(r *AIImageAsyncSubmission) { r.FileName = "a.avif" }), true},
		{"async .tiff", async(func(r *AIImageAsyncSubmission) { r.FileName = "a.tiff" }), false},
		{"async any mask type", async(func(r *AIImageAsyncSubmission) { r.MaskType = "future-mask" }), true},
		{"async 3 objects", async(func(r *AIImageAsyncSubmission) { r.ObjectDetection = objects("a", "b", long(32)) }), true},
		{"async 4 objects", async(func(r *AIImageAsyncSubmission) { r.ObjectDetection = objects("a", "b", "c", "d") }), false},
		{"async object of 33", async(func(r *AIImageAsyncSubmission) { r.ObjectDetection = objects(long(33)) }), false},
		{"async empty object", async(func(r *AIImageAsyncSubmission) { r.ObjectDetection = objects("") }), false},
		{"async payload 512", async(func(r *AIImageAsyncSubmission) { r.DeveloperPayload = long(512) }), false},
		{"async no webhook", async(func(r *AIImageAsyncSubmission) { r.Webhooks.URL = "" }), false},
		{"image heatmap", image(func(r *AIImageDetectionRequest) { r.MaskType = AIImageMaskHeatmap }), true},
		{"image mask Heatmap", image(func(r *AIImageDetectionRequest) { r.MaskType = "Heatmap" }), false},
		{"image 4 objects", image(func(r *AIImageDetectionRequest) { r.ObjectDetection = Ptr(objects("a", "b", "c", "d")) }), false},
		{"upload ok", upload(func(r *AIImageUploadRequest) {}), true},
		{"upload no image", upload(func(r *AIImageUploadRequest) { r.Image = nil }), false},
		{"upload no model", upload(func(r *AIImageUploadRequest) { r.Model = "" }), false},
		{"upload .gif", upload(func(r *AIImageUploadRequest) { r.Filename = "a.gif" }), false},
		{"upload mask blurry", upload(func(r *AIImageUploadRequest) { r.MaskType = "blurry" }), false},
		{"upload object of 33", upload(func(r *AIImageUploadRequest) { r.ObjectDetection = objects(long(33)) }), false},
		{"plagiarism ok", plagiarism(func(r *ImagePlagiarismRequest) {}), true},
		{"plagiarism .WEBP", plagiarism(func(r *ImagePlagiarismRequest) { r.Filename = "A.WEBP" }), true},
		{"plagiarism .heic", plagiarism(func(r *ImagePlagiarismRequest) { r.Filename = "a.heic" }), false},
		{"plagiarism 255 characters", plagiarism(func(r *ImagePlagiarismRequest) { r.Filename = long(251) + ".png" }), true},
		{"plagiarism 256 characters", plagiarism(func(r *ImagePlagiarismRequest) { r.Filename = long(252) + ".png" }), false},
		{"plagiarism no image", plagiarism(func(r *ImagePlagiarismRequest) { r.Image = nil }), false},
		{"plagiarism nil", func(c *Client) error { _, err := c.CheckImagePlagiarism(ctx, tok, "scan-1", nil); return err }, false},
		{"repository no ID", func(c *Client) error { _, err := c.RepositoryInfo(ctx, tok, ""); return err }, false},
		{"URL verb post", func(c *Client) error {
			return c.SubmitURL(ctx, tok, "scan-1", &URLSubmission{URL: "https://example.com", Verb: "post", Properties: validSubmitProps})
		}, true},
		{"URL verb PATCH", func(c *Client) error {
			return c.SubmitURL(ctx, tok, "scan-1", &URLSubmission{URL: "https://example.com", Verb: "PATCH", Properties: validSubmitProps})
		}, false},
		{"export AIDetection without verb", func(c *Client) error {
			return c.Export(ctx, tok, "scan-1", "export-1", &ExportRequest{CompletionWebhook: "https://example.com", AIDetection: &ExportTarget{Endpoint: "https://example.com"}})
		}, false},
		{"export Overview without endpoint", func(c *Client) error {
			return c.Export(ctx, tok, "scan-1", "export-1", &ExportRequest{CompletionWebhook: "https://example.com", Overview: &ExportTarget{Verb: "PUT"}})
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t, http.StatusOK, "{}")
			err := tt.call(env.client(t))
			if tt.valid {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				env.next(t)
				return
			}
			if !errors.Is(err, ErrInvalidArgument) {
				t.Fatalf("err = %v, want ErrInvalidArgument", err)
			}
			env.none(t)
		})
	}
}

func TestNewScanPropertiesValidation(t *testing.T) {
	list := func(n, length int) []string {
		l := make([]string, n)
		for i := range l {
			l[i] = strings.Repeat("a", length)
		}
		return l
	}
	tests := []struct {
		name  string
		mod   func(p *ScanProperties)
		valid bool
	}{
		{"sensitivity 1", func(p *ScanProperties) { p.AIGeneratedText = &AIGeneratedText{Detect: true, Sensitivity: 1} }, true},
		{"sensitivity 3", func(p *ScanProperties) { p.AIGeneratedText = &AIGeneratedText{Detect: true, Sensitivity: 3} }, true},
		{"sensitivity 4", func(p *ScanProperties) { p.AIGeneratedText = &AIGeneratedText{Detect: true, Sensitivity: 4} }, false},
		{"sensitivity -1", func(p *ScanProperties) { p.AIGeneratedText = &AIGeneratedText{Detect: true, Sensitivity: -1} }, false},
		{"10 backlink domains", func(p *ScanProperties) {
			p.Scanning = &Scanning{Exclude: &ScanningExclude{BacklinksDomains: Ptr(list(10, 256))}}
		}, true},
		{"11 backlink domains", func(p *ScanProperties) {
			p.Scanning = &Scanning{Exclude: &ScanningExclude{BacklinksDomains: Ptr(list(11, 5))}}
		}, false},
		{"backlink domain of 257", func(p *ScanProperties) {
			p.Scanning = &Scanning{Exclude: &ScanningExclude{BacklinksDomains: Ptr(list(1, 257))}}
		}, false},
		{"empty backlink domain", func(p *ScanProperties) {
			p.Scanning = &Scanning{Exclude: &ScanningExclude{BacklinksDomains: Ptr(list(1, 0))}}
		}, false},
		{"5 texts of 30", func(p *ScanProperties) { p.Scanning = &Scanning{Exclude: &ScanningExclude{Text: Ptr(list(5, 30))}} }, true},
		{"6 texts", func(p *ScanProperties) { p.Scanning = &Scanning{Exclude: &ScanningExclude{Text: Ptr(list(6, 3))}} }, false},
		{"text of 31", func(p *ScanProperties) { p.Scanning = &Scanning{Exclude: &ScanningExclude{Text: Ptr(list(1, 31))}} }, false},
		{"include pattern of 36", func(p *ScanProperties) {
			p.Scanning = &Scanning{Include: &ScanningInclude{IDPattern: strings.Repeat("a", 36)}}
		}, true},
		{"include pattern of 37", func(p *ScanProperties) {
			p.Scanning = &Scanning{Include: &ScanningInclude{IDPattern: strings.Repeat("a", 37)}}
		}, false},
		{"course ID of 64", func(p *ScanProperties) { p.Course = &Course{ID: strings.Repeat("a", 64)} }, true},
		{"course ID of 65", func(p *ScanProperties) { p.Course = &Course{ID: strings.Repeat("a", 65)} }, false},
		{"assignment ID of 65", func(p *ScanProperties) { p.Assignment = &Assignment{ID: strings.Repeat("a", 65)} }, false},
		{"institution ID of 65", func(p *ScanProperties) { p.Institution = &Institution{ID: strings.Repeat("a", 65)} }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props := validSubmitProps
			tt.mod(&props)
			env := newTestEnv(t, http.StatusCreated, "")
			err := env.client(t).SubmitFile(context.Background(), testToken(), "scan-1", &FileSubmission{Base64: "aGk=", Filename: "a.txt", Properties: props})
			if tt.valid != (err == nil) {
				t.Fatalf("err = %v, want valid = %v", err, tt.valid)
			}
			if !tt.valid {
				if !errors.Is(err, ErrInvalidArgument) {
					t.Errorf("err = %v, want ErrInvalidArgument", err)
				}
				env.none(t)
			}
		})
	}
}

func TestParseCompletedWebhookNewFields(t *testing.T) {
	w, err := ParseCompletedWebhook([]byte(fixture(t, "webhook_completed_features.json")))
	if err != nil {
		t.Fatal(err)
	}
	doc := w.ScannedDocument
	if doc.ExpectedCredits != 1 || doc.DetectedLanguage != "en" {
		t.Errorf("scannedDocument: %+v", doc)
	}
	wantEnabled := &ScanEnabledFeatures{PlagiarismDetection: true, WritingFeedback: true, PDFReport: true, Overview: true, ReferencesValidation: true, InternalAISourceMatch: true}
	if !reflect.DeepEqual(doc.Enabled, wantEnabled) {
		t.Errorf("enabled = %+v", doc.Enabled)
	}
	wantAI := []InternalAIResult{{
		ID: "a1b2c3", Title: "AI essay", Introduction: "Sample excerpt", MatchedWords: 40, IdenticalWords: 30, SimilarWords: 6, ParaphrasedWords: 4, TotalWords: 120,
		Tags: []ResultTag{{Code: "internal-ai-source-match", Title: "AI-Generated", Description: "This result contains AI-generated text"}},
	}}
	if !reflect.DeepEqual(w.Results.InternalAIData, wantAI) {
		t.Errorf("internalAIData = %+v", w.Results.InternalAIData)
	}
	wantRefs := &ReferencesValidation{Summary: &ReferencesValidationSummary{Total: 5, Academic: 3, AcademicValidated: 2, NonAcademicValidated: 1}}
	if !reflect.DeepEqual(w.ReferencesValidation, wantRefs) {
		t.Errorf("referencesValidation = %+v", w.ReferencesValidation)
	}
	wf := w.WritingFeedback
	if wf == nil || wf.TextStatistics.SentenceCount != 5 || wf.TextStatistics.SpeakingTimeSeconds != 29.5 || wf.Score.OverallScore != 94 ||
		wf.Score.GrammarCorrectionsScore != 93 || wf.Readability.ReadabilityLevelText != "5th Grader" {
		t.Errorf("writingFeedback = %+v", wf)
	}

	// The example of the API reference.
	ref, err := ParseCompletedWebhook([]byte(fixture(t, "webhook_completed.json")))
	if err != nil || ref.ReferencesValidation == nil || *ref.ReferencesValidation.Summary != (ReferencesValidationSummary{}) ||
		len(ref.Results.InternalAIData) != 1 || ref.Results.InternalAIData[0].ID != "string" {
		t.Errorf("reference example: %+v, %v", ref, err)
	}
	// A payload without the new fields leaves them empty.
	bare, err := ParseCompletedWebhook([]byte(`{"status":0,"scannedDocument":{"scanId":"s"},"results":{"internet":[]}}`))
	if err != nil || bare.ReferencesValidation != nil || bare.WritingFeedback != nil || bare.ScannedDocument.Enabled != nil || bare.Results.InternalAIData != nil {
		t.Errorf("bare payload: %+v, %v", bare, err)
	}
}

func TestParseNewResultWebhookInternalAIData(t *testing.T) {
	w, err := ParseNewResultWebhook([]byte(`{"score":{"aggregatedScore":3},"internalAIData":[{"id":"x","matchedWords":7,"totalWords":20}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if want := []InternalAIResult{{ID: "x", MatchedWords: 7, TotalWords: 20}}; !reflect.DeepEqual(w.InternalAIData, want) {
		t.Errorf("internalAIData = %+v", w.InternalAIData)
	}
}

func TestParseAIVideoWebhook(t *testing.T) {
	w, err := ParseAIVideoWebhook([]byte(fixture(t, "ai_video_webhook.json")))
	if err != nil {
		t.Fatal(err)
	}
	want := &AIVideoDetectionResponse{
		Model: "ai-video-1-pro",
		AudioResult: &AIVideoTrackResult{Starts: []int{13000, 45000, 47000}, Lengths: []int{14000, 1000, 8700},
			Exclude: &AIVideoRanges{Starts: []int{0, 3250, 5400, 7600, 10500}, Lengths: []int{2950, 1500, 1200, 650, 1050}}},
		VisualResult: &AIVideoTrackResult{Starts: []int{11566, 29433}, Lengths: []int{6134, 26267}, Exclude: &AIVideoRanges{Starts: []int{}, Lengths: []int{}}},
		Summary:      &AIVideoDetectionSummary{AudioAIRatio: 0.4902, VisualAIRatio: 0.5817, OverallAIRatio: 0.7487},
		IsAIDetected: true,
		VideoInfo: &AIVideoInfo{Duration: 55.7, Metadata: &AIVideoMetadata{
			IssuedTime: "2026-03-17T13:14:57+00:00", IssuedBy: "OpenAI", AppOrDeviceUsed: "Sora", ContentSummary: "Created using generative AI",
		}},
		ScannedVideo: &ScannedVideo{ScanID: "my-video-scan-1", ActualCredits: 1, ExpectedCredits: 1, CreationTime: "2026-05-05T12:37:50Z"},
	}
	if !reflect.DeepEqual(w, want) {
		t.Errorf("got %+v", w)
	}
	if _, err := ParseAIVideoWebhook([]byte(`{"summary":`)); err == nil || !strings.HasPrefix(err.Error(), "copyleaks: parse AI video webhook: ") {
		t.Errorf("err = %v", err)
	}
}

func TestParseAIImageWebhook(t *testing.T) {
	w, err := ParseAIImageWebhook([]byte(fixture(t, "ai_image_heatmap_response.json")))
	if err != nil {
		t.Fatal(err)
	}
	checkHeatmapResponse(t, w)
	w, err = ParseAIImageWebhook([]byte(`{"model":"ai-image-1-ultra","summary":{"ai":1,"human":0},"isAiDetected":true,"developerPayload":"order-1"}`))
	if err != nil || w.DeveloperPayload != "order-1" || !w.IsAIDetected || w.ScannedDocument != nil {
		t.Errorf("got %+v, %v", w, err)
	}
	if _, err := ParseAIImageWebhook([]byte(`[`)); err == nil || !strings.HasPrefix(err.Error(), "copyleaks: parse AI image webhook: ") {
		t.Errorf("err = %v", err)
	}
}

func TestParseAsyncFailureWebhooks(t *testing.T) {
	body := []byte(`{"error":{"type":"invalid_request_error","id":"file-too-small","code":400,` +
		`"message":"The file is too small.","url":"https://docs.copyleaks.com/errors",` +
		`"details":[{"param":"url","message":"The file is too small."}]},"developerPayload":"order-1"}`)
	want := &WebhookError{
		Type: "invalid_request_error", ID: "file-too-small", Code: 400, Message: "The file is too small.",
		URL: "https://docs.copyleaks.com/errors", Details: []WebhookErrorDetail{{Param: "url", Message: "The file is too small."}},
	}
	video, err := ParseAIVideoWebhook(body)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(video, &AIVideoDetectionResponse{Error: want, DeveloperPayload: "order-1"}) {
		t.Errorf("video: got %+v, error %+v", video, video.Error)
	}
	image, err := ParseAIImageWebhook(body)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(image, &AIImageDetectionResponse{Error: want, DeveloperPayload: "order-1"}) {
		t.Errorf("image: got %+v, error %+v", image, image.Error)
	}
	// Without the optional fields, and without a developer payload.
	body = []byte(`{"error":{"type":"api_error","id":"internal","code":500,"message":"Something went wrong."}}`)
	want = &WebhookError{Type: "api_error", ID: "internal", Code: 500, Message: "Something went wrong."}
	if video, err = ParseAIVideoWebhook(body); err != nil || !reflect.DeepEqual(video.Error, want) || video.DeveloperPayload != "" {
		t.Errorf("video: got %+v, %v", video, err)
	}
	if image, err = ParseAIImageWebhook(body); err != nil || !reflect.DeepEqual(image.Error, want) || image.Summary != nil {
		t.Errorf("image: got %+v, %v", image, err)
	}
	// A successful result has no Error.
	if video, err = ParseAIVideoWebhook([]byte(fixture(t, "ai_video_webhook.json"))); err != nil || video.Error != nil {
		t.Errorf("video success: Error = %+v, %v", video.Error, err)
	}
	if image, err = ParseAIImageWebhook([]byte(fixture(t, "ai_image_heatmap_response.json"))); err != nil || image.Error != nil {
		t.Errorf("image success: Error = %+v, %v", image.Error, err)
	}
}

func TestParseExportedFiles(t *testing.T) {
	res, err := ParseExportedResult([]byte(fixture(t, "exported_result.json")))
	if err != nil {
		t.Fatal(err)
	}
	if *res.Statistics != (ExportedResultStatistics{Identical: 12, MinorChanges: 3, RelatedMeaning: 1}) || res.Text.Value != "Hello world!" ||
		!reflect.DeepEqual(res.Text.Pages.StartPosition, []int{0}) {
		t.Errorf("result: %+v %+v", res.Statistics, res.Text)
	}
	ident := res.Text.Comparison.Identical
	if !reflect.DeepEqual(ident.Source.Chars, &TextRanges{Starts: []int{0}, Lengths: []int{5}}) || !reflect.DeepEqual(ident.Suspected.Words, &TextRanges{Starts: []int{1}, Lengths: []int{1}}) {
		t.Errorf("text comparison: %+v %+v", ident.Source, ident.Suspected)
	}
	if res.Text.Comparison.RelatedMeaning.Suspected.Chars.Starts[0] != 8 || len(res.Text.Comparison.MinorChanges.Source.Chars.Starts) != 0 {
		t.Errorf("text comparison: %+v", res.Text.Comparison)
	}
	if h := res.HTML; h.Value == "" || !reflect.DeepEqual(h.Comparison.Identical.GroupID, []int{0}) || h.Comparison.Identical.Source.Chars.Starts[0] != 16 || h.Comparison.MinorChanges != nil {
		t.Errorf("html: %+v", h)
	}

	cv, err := ParseExportedCrawledVersion([]byte(fixture(t, "exported_crawled_version.json")))
	if err != nil {
		t.Fatal(err)
	}
	if *cv.Metadata != (CrawledVersionMetadata{Words: 30, Excluded: 2}) || !reflect.DeepEqual(cv.Text.Exclude, &ExcludedRanges{Starts: []int{0}, Lengths: []int{14}, Reasons: []int{3}}) ||
		!reflect.DeepEqual(cv.HTML.Exclude, &ExcludedRanges{Starts: []int{16}, Lengths: []int{14}, Reasons: []int{3}, GroupIDs: []int{1}}) || cv.Text.Pages.StartPosition[0] != 0 {
		t.Errorf("crawled version: %+v %+v %+v", cv.Metadata, cv.Text, cv.HTML)
	}
	refs := cv.ReferencesValidation
	if *refs.Summary != (ReferencesValidationSummary{Total: 2, Academic: 1, AcademicValidated: 1}) || len(refs.Results) != 2 {
		t.Fatalf("references: %+v", refs)
	}
	academic := refs.Results[0]
	if academic.Type != "academic" || !reflect.DeepEqual(academic.HTMLRanges, &HTMLRanges{Starts: []int{240}, Lengths: []int{131}, GroupIDs: []int{0}}) ||
		academic.Parsed.Year != 2017 || len(academic.Parsed.Authors) != 3 || academic.Suggestions[0].DOI != "10.48550/arXiv.1706.03762" ||
		academic.Suggestions[0].ModifiedYear != 2023 || !*academic.Suggestions[0].Signals.Authors {
		t.Errorf("academic reference: %+v", academic)
	}
	other := refs.Results[1]
	signals := other.Suggestions[0].Signals
	if other.Type != "nonAcademic" || !reflect.DeepEqual(other.HTMLRanges, &HTMLRanges{}) || other.Parsed.URL == "" ||
		!*signals.Title || *signals.Year || signals.Authors != nil {
		t.Errorf("non-academic reference: %+v %+v", other, signals)
	}

	ai, err := ParseExportedAIDetection([]byte(fixture(t, "ai_text_response.json")))
	if err != nil || ai.ModelVersion != "v11.0" || ai.Summary.AI != 1 || ai.Explain == nil {
		t.Errorf("AI detection: %+v, %v", ai, err)
	}
	wf, err := ParseExportedWritingFeedback([]byte(fixture(t, "grammar_response.json")))
	if err != nil || wf.Score.Corrections.OverallScore != 79 || len(wf.Corrections.Text.Chars.Types) != 3 {
		t.Errorf("writing feedback: %+v, %v", wf, err)
	}
	ov, err := ParseExportedOverview([]byte(fixture(t, "exported_overview.json")))
	if err != nil || !strings.HasPrefix(ov.Overview, "### Current Plagiarism Detection") || ov.ModelVersion != "v1" {
		t.Errorf("overview: %+v, %v", ov, err)
	}

	for name, parse := range map[string]func([]byte) error{
		"result":           func(b []byte) error { _, err := ParseExportedResult(b); return err },
		"crawled version":  func(b []byte) error { _, err := ParseExportedCrawledVersion(b); return err },
		"AI detection":     func(b []byte) error { _, err := ParseExportedAIDetection(b); return err },
		"writing feedback": func(b []byte) error { _, err := ParseExportedWritingFeedback(b); return err },
		"overview":         func(b []byte) error { _, err := ParseExportedOverview(b); return err },
	} {
		if err := parse([]byte(`{"x":`)); err == nil || err.Error() == "" || !strings.HasPrefix(err.Error(), fmt.Sprintf("copyleaks: parse exported %s: ", name)) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if _, err := ParseExportedCrawledVersion([]byte(`{"referencesValidation":{"results":[{"htmlRanges":[1}]}}`)); err == nil {
		t.Error("malformed htmlRanges: err = nil")
	}
}

func TestLanguageCodesAcceptsAList(t *testing.T) {
	var res AllowedCrossLanguages
	if err := jsonUnmarshal(`{"documentLanguages":["en","fr","en"],"resultLanguages":null}`, &res); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.DocumentLanguages, LanguageCodes{"en": "en", "fr": "fr"}) || res.ResultLanguages != nil {
		t.Errorf("got %+v", res)
	}
	if codes := res.DocumentLanguages.Codes(); !reflect.DeepEqual(codes, []string{"en", "fr"}) {
		t.Errorf("Codes() = %v", codes)
	}
	if codes := (LanguageCodes{"english": "en", "anglais": "en"}).Codes(); !reflect.DeepEqual(codes, []string{"en"}) {
		t.Errorf("Codes() with a duplicate code = %v", codes)
	}
	if err := jsonUnmarshal(`{"documentLanguages":5}`, &res); err == nil {
		t.Error("a number must not decode")
	}
	if err := jsonUnmarshal(`{"documentLanguages":[5]}`, &res); err == nil {
		t.Error("a list of numbers must not decode")
	}
}

func TestAIDetectionResultSingleDecode(t *testing.T) {
	for data, wantErr := range map[string]bool{
		" \n\t{\"summary\":{\"ai\":0.5}}": false,
		"[1,":                             true,
		"nul":                             true,
		"\f{}":                            true,
	} {
		w, err := ParseCompletedWebhook(completedWithAlerts(t, aiAlert(data)))
		if err != nil {
			t.Fatal(err)
		}
		res, err := w.AIDetectionResult()
		if wantErr != (err != nil) {
			t.Errorf("%q: err = %v, want error = %v", data, err, wantErr)
		}
		if !wantErr && (res == nil || res.Summary.AI != 0.5) {
			t.Errorf("%q: result %+v", data, res)
		}
	}
}

func jsonUnmarshal(s string, v any) error {
	return json.Unmarshal([]byte(s), v)
}

func TestNewEndpointErrors(t *testing.T) {
	expired := &AuthToken{AccessToken: "x"}
	calls := map[string]func(c *Client, tok *AuthToken) error{
		"CheckImagePlagiarism": func(c *Client, tok *AuthToken) error {
			_, err := c.CheckImagePlagiarism(context.Background(), tok, "scan-1", &ImagePlagiarismRequest{Image: []byte{1}, Filename: "a.png"})
			return err
		},
		"DetectAIImageUpload": func(c *Client, tok *AuthToken) error {
			_, err := c.DetectAIImageUpload(context.Background(), tok, "scan-1", &AIImageUploadRequest{Image: []byte{1}, Filename: "a.png", Model: AIImageModel1Ultra})
			return err
		},
		"SubmitAIImageAsync": func(c *Client, tok *AuthToken) error {
			return c.SubmitAIImageAsync(context.Background(), tok, "scan-1", &AIImageAsyncSubmission{URL: "u", FileName: "a.png", Model: "m", Webhooks: ResultWebhook{URL: "w"}})
		},
		"SubmitAIVideo": func(c *Client, tok *AuthToken) error {
			return c.SubmitAIVideo(context.Background(), tok, "scan-1", &AIVideoSubmission{URL: "u", Filename: "a.mp4", Model: "m", Webhooks: ResultWebhook{URL: "w"}})
		},
		"AITextCreditsBalance": func(c *Client, tok *AuthToken) error {
			_, err := c.AITextCreditsBalance(context.Background(), tok)
			return err
		},
		"RepositoryInfo": func(c *Client, tok *AuthToken) error {
			_, err := c.RepositoryInfo(context.Background(), tok, "repo_1")
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t, http.StatusInternalServerError, `{"error":"boom"}`)
			c := env.client(t)
			if err := call(c, expired); !errors.Is(err, ErrTokenExpired) {
				t.Errorf("expired token: err = %v", err)
			}
			env.none(t)
			var apiErr *APIError
			if err := call(c, testToken()); !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusInternalServerError {
				t.Errorf("HTTP 500: err = %v", err)
			}
		})
	}
	for name, call := range map[string]func(c *Client) error{
		"AllowedCrossLanguages":    func(c *Client) error { _, err := c.AllowedCrossLanguages(context.Background()); return err },
		"SupportedAITextFileTypes": func(c *Client) error { _, err := c.SupportedAITextFileTypes(context.Background()); return err },
		"SupportedFileTypes":       func(c *Client) error { _, err := c.SupportedFileTypes(context.Background()); return err },
	} {
		env := newTestEnv(t, http.StatusServiceUnavailable, "")
		if err := call(env.client(t)); !errors.Is(err, ErrUnderMaintenance) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestImageContentType(t *testing.T) {
	for filename, want := range map[string]string{
		"a.png": "image/png", "a.JPG": "image/jpeg", "a.jpeg": "image/jpeg", "a.gif": "image/gif", "a.bmp": "image/bmp",
		"a.webp": "image/webp", "a.ico": "image/x-icon", "a.heic": "image/heic", "a.heif": "image/heif", "a.avif": "image/avif",
		"a.tif": "image/tiff", "a.tiff": "image/tiff", "a.raw": "application/octet-stream", "noext": "application/octet-stream",
	} {
		if got := imageContentType(filename); got != want {
			t.Errorf("imageContentType(%q) = %q, want %q", filename, got, want)
		}
	}
}

func TestMultipartFilenameIsEscaped(t *testing.T) {
	env := newTestEnv(t, http.StatusOK, "{}")
	req := &ImagePlagiarismRequest{Image: []byte{1, 2}, Filename: "my \"best\"\r\nphoto.png"}
	if _, err := env.client(t).CheckImagePlagiarism(context.Background(), testToken(), "scan-1", req); err != nil {
		t.Fatal(err)
	}
	form := parseMultipart(t, env.next(t), "image")
	if form.fileName != `my "best"photo.png` || form.fields["filename"] != req.Filename || !bytes.Equal(form.file, []byte{1, 2}) {
		t.Errorf("form = %+v", form)
	}
}
