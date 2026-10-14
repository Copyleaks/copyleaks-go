package copyleaks

import (
	"bytes"
	"encoding/base64"
	"errors"
	"mime/multipart"
	"net/textproto"
	"path"
	"strings"
)

// base64Prefix is how the JSON of a request type whose first field is
// `json:"base64"` starts when that field is empty.
const base64Prefix = `{"base64":""`

// base64Body returns the JSON body of an upload request. v is the request, with its
// Base64 field, which must be its first field, set to "". The body holds b64, or, when
// raw is not empty, the base64 encoding of raw, as the base64 field.
//
// It builds the body in one buffer of the exact size, so that a large upload costs about
// one copy of its base64 text, instead of the several copies that encoding the whole
// request with encoding/json makes.
func base64Body(v any, b64 string, raw []byte) ([]byte, error) {
	rest, err := encodeJSON(v)
	if err != nil {
		return nil, err
	}
	if !bytes.HasPrefix(rest, []byte(base64Prefix)) {
		return nil, errors.New("the request does not start with its base64 field")
	}
	rest = rest[len(base64Prefix):]
	useRaw := len(raw) > 0
	if !useRaw && !jsonSafe(b64) {
		// Not base64 text: let encoding/json quote it. The server rejects it anyway.
		quoted, err := encodeJSON(b64)
		if err != nil {
			return nil, err
		}
		return append(append([]byte(`{"base64":`), quoted...), rest...), nil
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

// jsonSafe reports whether s can be put in a JSON string as it is: printable ASCII
// without quotes and backslashes, which is true of base64 text.
func jsonSafe(s string) bool {
	for i := 0; i < len(s); i++ {
		if c := s[i]; c < 0x20 || c > 0x7e || c == '"' || c == '\\' {
			return false
		}
	}
	return true
}

// formField is a text field of a multipart/form-data body.
type formField struct {
	name  string
	value string
}

// multipartBody returns a multipart/form-data body with the file content in the field
// fileField, followed by fields, and the Content-Type header that goes with it. The body
// is built in one buffer that is sized for the file, so that the upload costs about one
// copy of the file.
func multipartBody(fileField, filename string, content []byte, fields []formField) ([]byte, string, error) {
	size := len(content) + 1024
	for _, f := range fields {
		size += len(f.name) + len(f.value) + 128
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
		if err := w.WriteField(f.name, f.value); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), w.FormDataContentType(), nil
}

// quoteEscaper escapes the characters that end or break a quoted Content-Disposition
// parameter, as mime/multipart does.
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
