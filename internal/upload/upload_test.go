package upload

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"strings"
	"testing"
)

func TestBase64Body(t *testing.T) {
	const head = `{"base64":"","filename":"a.txt"}`
	tests := []struct {
		name string
		b64  string
		raw  []byte
		want string
	}{
		{"base64", "aGk=", nil, `{"base64":"aGk=","filename":"a.txt"}`},
		{"raw", "", []byte("hi"), `{"base64":"aGk=","filename":"a.txt"}`},
		{"empty raw uses base64", "aGk=", []byte{}, `{"base64":"aGk=","filename":"a.txt"}`},
		{"empty", "", nil, head},
		{"not base64", "not \"base64\" \\ <é>\n", nil, `{"base64":"not \"base64\" \\ <é>\n","filename":"a.txt"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Base64Body([]byte(head), tt.b64, tt.raw)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("body\n got: %s\nwant: %s", got, tt.want)
			}
		})
	}
}

func TestBase64BodyUsesOneBufferOfTheExactSize(t *testing.T) {
	head := []byte(`{"base64":"","filename":"a.txt"}`)
	for name, call := range map[string]func() ([]byte, error){
		"base64": func() ([]byte, error) { return Base64Body(head, strings.Repeat("aGk=", 1000), nil) },
		"raw":    func() ([]byte, error) { return Base64Body(head, "", bytes.Repeat([]byte("hi"), 1000)) },
	} {
		got, err := call()
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != cap(got) {
			t.Errorf("%s: len = %d, cap = %d, want one buffer of the exact size", name, len(got), cap(got))
		}
	}
}

func TestBase64BodyNeedsTheBase64FieldFirst(t *testing.T) {
	for _, head := range []string{`{"filename":"a.txt","base64":""}`, `{"base64":"aGk=","filename":"a.txt"}`, ``} {
		if _, err := Base64Body([]byte(head), "aGk=", nil); err == nil {
			t.Errorf("Base64Body(%s): err = nil, want an error", head)
		}
	}
}

func TestMultipartBody(t *testing.T) {
	file := []byte{0x89, 'P', 'N', 'G', 0}
	body, contentType, err := MultipartBody("image", "my \"best\"\r\nphoto.PNG", file, []Field{{Name: "filename", Value: "a.png"}, {Name: "sandbox", Value: "true"}})
	if err != nil {
		t.Fatal(err)
	}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != "multipart/form-data" || params["boundary"] == "" {
		t.Fatalf("Content-Type = %q, want multipart/form-data", contentType)
	}
	r := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	var names []string
	for {
		part, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("bad multipart body: %v", err)
		}
		b, _ := io.ReadAll(part)
		names = append(names, part.FormName())
		switch part.FormName() {
		case "image":
			if part.FileName() != `my "best"photo.PNG` || part.Header.Get("Content-Type") != "image/png" || !bytes.Equal(b, file) {
				t.Errorf("file part: filename %q, Content-Type %q, content %q", part.FileName(), part.Header.Get("Content-Type"), b)
			}
		case "filename":
			if string(b) != "a.png" {
				t.Errorf("filename = %q", b)
			}
		case "sandbox":
			if string(b) != "true" {
				t.Errorf("sandbox = %q", b)
			}
		}
	}
	if len(names) != 3 || names[0] != "image" || names[1] != "filename" || names[2] != "sandbox" {
		t.Errorf("parts = %q, want image, filename, sandbox", names)
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
