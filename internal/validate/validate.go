// Package validate holds the client-side checks of IDs, file names and uploads. Its errors only describe the
// problem; the caller wraps them to match copyleaks.ErrInvalidArgument.
package validate

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Scan and export ID length limits, in UTF-16 code units as the server counts them.
const (
	minScanIDLength = 3
	maxScanIDLength = 36
)

var (
	// submitScanIDPattern is the scan ID character set of the Plagiarism Checker submits.
	submitScanIDPattern = regexp.MustCompile(`^[a-z0-9!@$&=_()';:.,~-]+$`)
	// exportIDPattern is the character set of the scan ID and export ID of Export.
	exportIDPattern = regexp.MustCompile("^[a-z0-9!@$^&+%=_(){}<>'\";:/.,~`|\\\\-]+$")
	// googlePattern matches what submits and Export reject in an ID: google, gogle, g00gle, ...
	googlePattern = regexp.MustCompile(`g[o0]+gle`)
)

// aiImageExtensions are the file extensions that AI Image Detection accepts.
var aiImageExtensions = []string{".png", ".bmp", ".jpg", ".jpeg", ".webp", ".heic", ".heif", ".avif"}

// ID checks the 3 to 36 character length of every scan and export ID; name names it in the error.
func ID(name, id string) error {
	if n := UTF16Len(id); n < minScanIDLength || n > maxScanIDLength {
		return fmt.Errorf("%s must be %d to %d characters, got %d", name, minScanIDLength, maxScanIDLength, n)
	}
	return nil
}

// SubmitScanID applies the stricter scan ID rules of the Plagiarism Checker submits.
func SubmitScanID(id string) error {
	return charsetID("scan ID", id, submitScanIDPattern, "[a-z0-9!@$&=_()';:.,~-]")
}

// ExportID applies the rules of Export to its scan ID and export ID.
func ExportID(name, id string) error {
	return charsetID(name, id, exportIDPattern, "[a-z0-9!@$^&+%=_(){}<>'\";:/.,~`|\\-]")
}

// charsetID checks the length of an ID, its characters (pattern, described by charset), and googlePattern.
func charsetID(name, id string, pattern *regexp.Regexp, charset string) error {
	if err := ID(name, id); err != nil {
		return err
	}
	if !pattern.MatchString(id) {
		return fmt.Errorf("%s %q may only contain the characters %s", name, id, charset)
	}
	if googlePattern.MatchString(id) {
		return fmt.Errorf("%s %q must not contain g, then o or 0, then gle (as in google)", name, id)
	}
	return nil
}

// HasAIImageExtension reports whether filename ends with a supported image extension.
func HasAIImageExtension(filename string) bool {
	lower := strings.ToLower(filename)
	for _, ext := range aiImageExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// UTF16Len returns the length of s in UTF-16 code units.
func UTF16Len(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// UploadContent checks that exactly one of Base64 and Content of an upload request is set.
func UploadContent(b64 string, content []byte) error {
	switch {
	case b64 == "" && len(content) == 0:
		return errors.New("either Base64 or Content is required")
	case b64 != "" && len(content) > 0:
		return errors.New("set either Base64 or Content, not both")
	}
	return nil
}
