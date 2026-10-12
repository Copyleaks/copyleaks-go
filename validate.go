package copyleaks

import (
	"fmt"
	"regexp"
	"strings"
)

// Limits that the server enforces and the SDK checks before sending a request.
// Lengths are counted in UTF-16 code units, as the server counts them.
const (
	minScanIDLength = 3
	maxScanIDLength = 36
	minAITextLength = 255
	maxAITextLength = 100000
)

var (
	// submitScanIDPattern is the character set that the Plagiarism Checker submit
	// endpoints accept in a scan ID.
	submitScanIDPattern = regexp.MustCompile(`^[a-z0-9!@$&=_()';:.,~-]+$`)
	// exportIDPattern is the character set that Export accepts in the scan ID and the
	// export ID.
	exportIDPattern = regexp.MustCompile("^[a-z0-9!@$^&+%=_(){}<>'\";:/.,~`|\\\\-]+$")
	// googlePattern matches the sequences that the submit and export endpoints reject in
	// an ID: g, then one or more o or 0, then gle (google, gogle, g00gle, ...).
	googlePattern = regexp.MustCompile(`g[o0]+gle`)
)

// aiImageExtensions are the file extensions that AI Image Detection accepts.
var aiImageExtensions = []string{".png", ".bmp", ".jpg", ".jpeg", ".webp", ".heic", ".heif", ".avif"}

// validateID checks the 3 to 36 character length that every endpoint enforces on scan
// and export IDs. name names the argument in the error.
func validateID(name, id string) error {
	if n := utf16Len(id); n < minScanIDLength || n > maxScanIDLength {
		return invalid(fmt.Sprintf("%s must be %d to %d characters, got %d", name, minScanIDLength, maxScanIDLength, n))
	}
	return nil
}

// validateSubmitScanID applies the stricter scan ID rules of the Plagiarism Checker
// submit endpoints: the length, the character set [a-z0-9!@$&=_()';:.,~-], and no
// g[o0]+gle sequence.
func validateSubmitScanID(id string) error {
	return validateCharsetID("scan ID", id, submitScanIDPattern, "[a-z0-9!@$&=_()';:.,~-]")
}

// validateExportID applies the rules of Export to its scan ID and export ID: the length,
// the character set [a-z0-9!@$^&+%=_(){}<>'";:/.,~`|\-], and no g[o0]+gle sequence.
func validateExportID(name, id string) error {
	return validateCharsetID(name, id, exportIDPattern, "[a-z0-9!@$^&+%=_(){}<>'\";:/.,~`|\\-]")
}

// validateCharsetID checks the length of an ID, that it only has characters that match
// pattern, which charset describes, and that it has no g[o0]+gle sequence.
func validateCharsetID(name, id string, pattern *regexp.Regexp, charset string) error {
	if err := validateID(name, id); err != nil {
		return err
	}
	if !pattern.MatchString(id) {
		return invalid(fmt.Sprintf("%s %q may only contain the characters %s", name, id, charset))
	}
	if googlePattern.MatchString(id) {
		return invalid(fmt.Sprintf("%s %q must not contain g, then o or 0, then gle (as in google)", name, id))
	}
	return nil
}

// hasAIImageExtension reports whether filename ends with a supported image extension.
func hasAIImageExtension(filename string) bool {
	lower := strings.ToLower(filename)
	for _, ext := range aiImageExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// utf16Len returns the length of s in UTF-16 code units.
func utf16Len(s string) int {
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
