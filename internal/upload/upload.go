// Package upload builds upload request bodies (JSON with base64, or multipart/form-data), each in one buffer,
// so that a large upload costs about one copy of the file.
package upload

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/textproto"
	"path"
	"strings"
)

// base64Prefix starts the JSON of a request whose first field, `json:"base64"`, is empty.
const base64Prefix = `{"base64":""`

// Base64Body returns head, the JSON of a request whose first field, base64, is "", with b64 (or the base64 of
// raw, when set) as that field. One exactly sized buffer avoids the copies that encoding/json would make.
func Base64Body(head []byte, b64 string, raw []byte) ([]byte, error) {
	if !bytes.HasPrefix(head, []byte(base64Prefix)) {
		return nil, errors.New("the request does not start with its base64 field")
	}
	rest := head[len(base64Prefix):]
	useRaw := len(raw) > 0
	if !useRaw && !jsonSafe(b64) {
		// Not base64 text: let encoding/json quote it. The server rejects it anyway.
		return append(append([]byte(`{"base64":`), quote(b64)...), rest...), nil
	}
	n := len(b64)
	if useRaw {
		n = base64.StdEncoding.EncodedLen(len(raw))
	}
	const open = `{"base64":"`
	body := make([]byte, 0, len(open)+n+1+len(rest))
	body = append(body, open...)
	if useRaw {
		body = body[:len(body)+n]
		base64.StdEncoding.Encode(body[len(open):], raw)
	} else {
		body = append(body, b64...)
	}
	body = append(body, '"')
	return append(body, rest...), nil
}

// jsonSafe reports whether s needs no JSON escaping (printable ASCII without " or \), as base64 text.
func jsonSafe(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i]; c < 0x20 || c > 0x7e || c == '"' || c == '\\' {
			return false
		}
	}
	return true
}

// quote returns s as a JSON string without escaping <, > and &, like the rest of the request.
func quote(s string) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s) // A string always encodes.
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n"))
}

// Field is a text field of a multipart/form-data body.
type Field struct {
	Name  string
	Value string
}

// MultipartBody returns a multipart/form-data body, with content in fileField followed by fields, and its
// Content-Type.
func MultipartBody(fileField, filename string, content []byte, fields []Field) ([]byte, string, error) {
	size := len(content) + 1024
	for _, f := range fields {
		size += len(f.Name) + len(f.Value) + 128
	}
	buf := bytes.NewBuffer(make([]byte, 0, size))
	w := multipart.NewWriter(buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="`+escapeQuotes(fileField)+`"; filename="`+escapeQuotes(filename)+`"`)
	h.Set("Content-Type", imageContentType(filename))
	part, err := w.CreatePart(h)
	if err != nil {
		return nil, "", err
	}
	if _, err := part.Write(content); err != nil {
		return nil, "", err
	}
	for _, f := range fields {
		if err := w.WriteField(f.Name, f.Value); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), w.FormDataContentType(), nil
}

// quoteEscaper escapes what ends or breaks a quoted Content-Disposition parameter, as mime/multipart does.
var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"", "\r", "", "\n", "")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

// imageContentType returns the media type of an image file, by its extension.
func imageContentType(filename string) string {
	switch strings.ToLower(path.Ext(filename)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/x-icon"
	case ".heic":
		return "image/heic"
	case ".heif":
		return "image/heif"
	case ".avif":
		return "image/avif"
	case ".tif", ".tiff":
		return "image/tiff"
	}
	return "application/octet-stream"
}
