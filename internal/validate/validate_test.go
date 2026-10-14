package validate

import (
	"strings"
	"testing"
)

func TestID(t *testing.T) {
	for _, tt := range []struct {
		id    string
		valid bool
	}{
		{"abc", true},
		{strings.Repeat("a", 36), true},
		{"my-google-scan", true},
		{"Scan_ID-With-Upper", true},
		{strings.Repeat("\U0001F600", 18), true},
		{"ab", false},
		{"", false},
		{strings.Repeat("a", 37), false},
		{strings.Repeat("\U0001F600", 19), false},
	} {
		err := ID("scan ID", tt.id)
		if tt.valid != (err == nil) {
			t.Errorf("ID(%q) = %v, want valid = %v", tt.id, err, tt.valid)
		}
		if err != nil && !strings.HasPrefix(err.Error(), "scan ID must be 3 to 36 characters") {
			t.Errorf("ID(%q) = %v, want an error about the scan ID length", tt.id, err)
		}
	}
}

func TestSubmitScanID(t *testing.T) {
	for _, tt := range []struct {
		id    string
		valid bool
	}{
		{"abc", true},
		{"scan-1!@$&=_()';:.,~", true},
		{strings.Repeat("a", 36), true},
		{"ggle-1", true},
		{"ab", false},
		{strings.Repeat("a", 37), false},
		{"Scan-1", false},
		{"scan 1", false},
		{"scan/1", false},
		{"scan#1", false},
		{"my-google-scan", false},
		{"gogle1", false},
		{"g00gle1", false},
		{"goo0ogle", false},
	} {
		err := SubmitScanID(tt.id)
		if tt.valid != (err == nil) {
			t.Errorf("SubmitScanID(%q) = %v, want valid = %v", tt.id, err, tt.valid)
		}
	}
}

func TestExportID(t *testing.T) {
	for _, tt := range []struct {
		id    string
		valid bool
	}{
		{"export-1", true},
		{"a^b+c%d{e}<f>|g~h", true},
		{"it's;ok:\"`\\", true},
		{"ab", false},
		{strings.Repeat("a", 37), false},
		{"Export-1", false},
		{"export 1", false},
		{"export#1", false},
		{"export[1]", false},
		{"my-google-export", false},
		{"g00gle-1", false},
	} {
		err := ExportID("export ID", tt.id)
		if tt.valid != (err == nil) {
			t.Errorf("ExportID(%q) = %v, want valid = %v", tt.id, err, tt.valid)
		}
		if err != nil && !strings.HasPrefix(err.Error(), "export ID ") {
			t.Errorf("ExportID(%q) = %v, want an error that names the export ID", tt.id, err)
		}
	}
}

func TestHasAIImageExtension(t *testing.T) {
	for filename, want := range map[string]bool{
		"a.png": true, "IMAGE.JPEG": true, "a.jpg": true, "a.bmp": true, "a.webp": true,
		"photo.heic": true, "photo.heif": true, "photo.avif": true, ".png": true,
		"image.gif": false, "image": false, "png": false, "a.png.txt": false,
	} {
		if got := HasAIImageExtension(filename); got != want {
			t.Errorf("HasAIImageExtension(%q) = %v, want %v", filename, got, want)
		}
	}
}

func TestUTF16Len(t *testing.T) {
	for s, want := range map[string]int{
		"": 0, "abc": 3, "é": 1, "日本": 2, "\U0001F600": 2, "a\U0001F600b": 4,
	} {
		if got := UTF16Len(s); got != want {
			t.Errorf("UTF16Len(%q) = %d, want %d", s, got, want)
		}
	}
}

func TestUploadContent(t *testing.T) {
	for _, tt := range []struct {
		b64     string
		content []byte
		valid   bool
	}{
		{"aGk=", nil, true},
		{"aGk=", []byte{}, true},
		{"", []byte("hi"), true},
		{"", nil, false},
		{"", []byte{}, false},
		{"aGk=", []byte("hi"), false},
	} {
		if err := UploadContent(tt.b64, tt.content); tt.valid != (err == nil) {
			t.Errorf("UploadContent(%q, %q) = %v, want valid = %v", tt.b64, tt.content, err, tt.valid)
		}
	}
}
