package copyleaks

import "fmt"

// ScanProperties configures a Plagiarism Checker scan. Only Webhooks.Status is required.
// Every other field is optional and is left out of the request when it has its zero
// value, so the server default applies. Pointer fields are pointers because their zero
// value differs from the server default; set them with Ptr.
//
// See https://docs.copyleaks.com/reference/actions/authenticity/submit-file.
type ScanProperties struct {
	// Webhooks sets the URLs that Copyleaks calls with the scan status (required).
	Webhooks SubmitWebhooks `json:"webhooks"`
	// Sandbox submits a free test scan that returns mock results.
	Sandbox bool `json:"sandbox,omitempty"`
	// Action is what the submit does. The default, SubmitActionScan, starts the scan.
	Action SubmitAction `json:"action,omitempty"`
	// IncludeHTML also generates the report in HTML format, when possible.
	IncludeHTML bool `json:"includeHtml,omitempty"`
	// DeveloperPayload is sent back in the webhooks of the scan. Up to 512 characters.
	DeveloperPayload string `json:"developerPayload,omitempty"`
	// Expiration is how many hours the scan is kept after it completes, from 1 to 2880.
	// The server default is 2880.
	Expiration int `json:"expiration,omitempty"`
	// Author identifies the author of the content.
	Author *Author `json:"author,omitempty"`
	// Filters fine-tunes which results are included.
	Filters *Filters `json:"filters,omitempty"`
	// Scanning sets the sources that the content is compared against.
	Scanning *Scanning `json:"scanning,omitempty"`
	// Indexing sets the Private Cloud Hubs that the content is indexed in.
	Indexing *Indexing `json:"indexing,omitempty"`
	// Exclude sets the parts of the content that are not scanned.
	Exclude *Exclude `json:"exclude,omitempty"`
	// PDF requests and customizes a PDF report.
	PDF *PDF `json:"pdf,omitempty"`
	// DisplayLanguage is the language of the scan's PDF report: "en" (the server default),
	// "es", "pt", "fr", "de" or "it". The codes are lowercase and case-sensitive. It is
	// stored with the scan, so it also applies to a PDF report requested later.
	DisplayLanguage string `json:"displayLanguage,omitempty"`
	// SensitivityLevel trades speed for coverage, from 1 (fastest) to 5 (most thorough).
	// The server default is 3.
	SensitivityLevel int `json:"sensitivityLevel,omitempty"`
	// CheatDetection checks the content for text manipulation and reports it as scan
	// alerts in the completed webhook.
	CheatDetection bool `json:"cheatDetection,omitempty"`
	// ScanMethodAlgorithm chooses between maximum coverage (the default) and maximum results.
	ScanMethodAlgorithm ScanMethodAlgorithm `json:"scanMethodAlgorithm,omitempty"`
	// AIGeneratedText adds AI Text Detection to the scan. The result arrives as a
	// suspected-ai-text alert in the completed webhook; see CompletedWebhook.AIDetectionResult.
	AIGeneratedText *AIGeneratedText `json:"aiGeneratedText,omitempty"`
	// AISourceMatch identifies the online sources that are suspected of containing
	// AI-generated text. It only applies to documents detected as English.
	AISourceMatch *AISourceMatch `json:"aiSourceMatch,omitempty"`
	// CustomMetadata is attached to the document in a Private Cloud Hub and added to the
	// results that match it.
	CustomMetadata []CustomMetadata `json:"customMetadata,omitempty"`
	// ScanTimeZone is the time zone of the scan time shown in the PDF report, as a
	// case-sensitive IANA time zone name, such as "America/New_York". The server default
	// is the time zone of the account's country, or UTC.
	ScanTimeZone string `json:"scanTimeZone,omitempty"`
	// Course identifies the course of the content, for analytics.
	Course *Course `json:"course,omitempty"`
	// Assignment identifies the assignment of the content, for analytics.
	Assignment *Assignment `json:"assignment,omitempty"`
	// Institution identifies the institution of the content, for analytics.
	Institution *Institution `json:"institution,omitempty"`
	// SensitiveDataProtection masks sensitive data in the scanned document with #
	// characters. It is available on plans of 2,500 pages or more.
	SensitiveDataProtection *SensitiveDataProtection `json:"sensitiveDataProtection,omitempty"`
	// WritingFeedback adds the Grammar Checker to the scan.
	WritingFeedback *ScanWritingFeedback `json:"writingFeedback,omitempty"`
	// Overview adds a Gen-AI Overview of the scan results. Export it with
	// ExportRequest.Overview.
	Overview *ScanOverview `json:"overview,omitempty"`
	// InternalAISourceMatch identifies sources in the Copyleaks internal AI database that
	// contain AI-generated text. The matches are reported in Results.InternalAIData.
	InternalAISourceMatch *InternalAISourceMatch `json:"internalAiSourceMatch,omitempty"`
	// References turns on the Citation Checker, which validates the references of the
	// text. The summary is reported in CompletedWebhook.ReferencesValidation.
	References *References `json:"references,omitempty"`
}

// SubmitWebhooks sets the webhook URLs of a scan.
type SubmitWebhooks struct {
	// Status is called when the scan status changes (required). Its {STATUS} placeholder
	// is replaced with completed, error, creditschecked or indexed.
	Status string `json:"status"`
	// StatusHeaders are extra headers for the status webhook, as [name, value] pairs.
	StatusHeaders [][]string `json:"statusHeaders,omitempty"`
	// NewResult is called each time a result is found while the scan runs.
	NewResult string `json:"newResult,omitempty"`
	// NewResultHeaders are extra headers for the new-result webhook, as [name, value] pairs.
	NewResultHeaders [][]string `json:"newResultHeaders,omitempty"`
}

// Author identifies the author of the scanned content.
type Author struct {
	// ID is a unique identifier of the author.
	ID string `json:"id,omitempty"`
}

// Course identifies the course of the scanned content, for analytics.
type Course struct {
	// ID is a unique identifier of the course.
	ID string `json:"id,omitempty"`
}

// Assignment identifies the assignment of the scanned content, for analytics.
type Assignment struct {
	// ID is a unique identifier of the assignment.
	ID string `json:"id,omitempty"`
}

// Institution identifies the institution of the scanned content, for analytics.
type Institution struct {
	// ID is a unique identifier of the institution.
	ID string `json:"id,omitempty"`
}

// Filters fine-tunes which results are included in a scan report.
type Filters struct {
	// IdenticalEnabled matches exact words. The server default is true.
	IdenticalEnabled *bool `json:"identicalEnabled,omitempty"`
	// MinorChangesEnabled matches nearly identical words, such as slow and slowly.
	// The server default is true.
	MinorChangesEnabled *bool `json:"minorChangesEnabled,omitempty"`
	// RelatedMeaningEnabled matches paraphrased content. The server default is true.
	RelatedMeaningEnabled *bool `json:"relatedMeaningEnabled,omitempty"`
	// MinCopiedWords only keeps results with at least this many copied words.
	MinCopiedWords int `json:"minCopiedWords,omitempty"`
	// SafeSearch blocks explicit adult content from the results.
	SafeSearch bool `json:"safeSearch,omitempty"`
	// Domains are domains to include in or exclude from the results; see DomainsMode.
	Domains []string `json:"domains,omitempty"`
	// DomainsMode sets whether Domains are included or excluded. The server default is
	// DomainsModeExclude.
	DomainsMode *DomainsMode `json:"domainsMode,omitempty"`
	// AllowSameDomain allows results from the domain of the submitted URL.
	AllowSameDomain bool `json:"allowSameDomain,omitempty"`
}

// Scanning sets the sources that a scan compares the content against.
type Scanning struct {
	// Internet compares the content with online sources. The server default is true.
	Internet *bool `json:"internet,omitempty"`
	// Exclude leaves matching submissions out of the results.
	Exclude *ScanningExclude `json:"exclude,omitempty"`
	// Include keeps only matching submissions in the results.
	Include *ScanningInclude `json:"include,omitempty"`
	// Repositories are the Private Cloud Hubs to compare the content against.
	Repositories []ScanningRepository `json:"repositories,omitempty"`
	// CopyleaksDB compares the content against the Shared Data Hub.
	CopyleaksDB *CopyleaksDB `json:"copyleaksDb,omitempty"`
	// CrossLanguages also compares the content against sources in other languages.
	CrossLanguages *CrossLanguages `json:"crossLanguages,omitempty"`
}

// ScanningExclude leaves submissions out of the results.
type ScanningExclude struct {
	// IDPattern excludes the submissions whose scan ID matches the pattern, 1 to 36
	// characters. A * matches any number of characters and a . matches exactly one
	// non-whitespace character.
	IDPattern string `json:"idPattern,omitempty"`
	// BacklinksDomains excludes the results that link back to these domains: up to 10
	// domains, each 1 to 256 characters.
	BacklinksDomains []string `json:"backlinksDomains,omitempty"`
	// Text excludes the results that contain these phrases: up to 5 phrases, each 1 to 30
	// characters.
	Text []string `json:"text,omitempty"`
}

// ScanningInclude keeps only some submissions in the results.
type ScanningInclude struct {
	// IDPattern keeps only the Shared Data Hub and Private Cloud Hub results whose scan ID
	// matches the pattern. It uses the wildcards of ScanningExclude.IDPattern.
	IDPattern string `json:"idPattern,omitempty"`
}

// ScanningRepository is a Private Cloud Hub to compare the content against.
type ScanningRepository struct {
	// ID is the repository ID of the Private Cloud Hub.
	ID string `json:"id"`
	// IncludeMySubmissions compares against your own submissions in the hub.
	IncludeMySubmissions bool `json:"includeMySubmissions,omitempty"`
	// IncludeOthersSubmissions compares against other users' submissions in the hub.
	IncludeOthersSubmissions bool `json:"includeOthersSubmissions,omitempty"`
}

// CopyleaksDB sets how the content is compared against the Shared Data Hub.
type CopyleaksDB struct {
	// IncludeMySubmissions compares against content that you added to the Shared Data Hub.
	IncludeMySubmissions bool `json:"includeMySubmissions,omitempty"`
	// IncludeOthersSubmissions compares against content that other users added to the
	// Shared Data Hub.
	IncludeOthersSubmissions bool `json:"includeOthersSubmissions,omitempty"`
}

// CrossLanguages sets the extra languages of a cross-language scan.
type CrossLanguages struct {
	// Languages are the languages to compare the content against, besides its own.
	Languages []CrossLanguage `json:"languages"`
}

// CrossLanguage is one language of a cross-language scan.
type CrossLanguage struct {
	// Code is the language code, such as "fr".
	Code string `json:"code"`
}

// Indexing sets where the submitted content is indexed.
type Indexing struct {
	// Repositories are the Private Cloud Hubs to index the content in.
	Repositories []IndexingRepository `json:"repositories,omitempty"`
	// CopyleaksDB adds the content to the Copyleaks Shared Data Hub.
	CopyleaksDB bool `json:"copyleaksDb,omitempty"`
}

// IndexingRepository is a Private Cloud Hub to index the content in.
type IndexingRepository struct {
	// ID is the repository ID of the Private Cloud Hub.
	ID string `json:"id"`
	// MaskingPolicy sets who sees the content of this document in their results.
	MaskingPolicy *MaskingPolicy `json:"maskingPolicy,omitempty"`
}

// Exclude sets the parts of the content that a scan does not compare.
type Exclude struct {
	// Quotes excludes quoted text.
	Quotes bool `json:"quotes,omitempty"`
	// Citations excludes citations.
	Citations bool `json:"citations,omitempty"`
	// References excludes referenced text.
	References bool `json:"references,omitempty"`
	// TableOfContents excludes the table of contents.
	TableOfContents bool `json:"tableOfContents,omitempty"`
	// Titles excludes the titles of an HTML document.
	Titles bool `json:"titles,omitempty"`
	// HTMLTemplate excludes the text that repeats across the pages of a site, such as the
	// header and the footer.
	HTMLTemplate bool `json:"htmlTemplate,omitempty"`
	// DocumentTemplateIDs excludes the template text found in up to three other scans.
	DocumentTemplateIDs []string `json:"documentTemplateIds,omitempty"`
	// Code sets what is excluded from source code files.
	Code *ExcludeCode `json:"code,omitempty"`
}

// ExcludeCode sets what is excluded from source code files.
type ExcludeCode struct {
	// Comments excludes code comments.
	Comments bool `json:"comments,omitempty"`
}

// PDF requests and customizes the PDF report of a scan.
type PDF struct {
	// Create generates a PDF report.
	Create bool `json:"create,omitempty"`
	// Title is the title of the report, up to 256 characters.
	Title string `json:"title,omitempty"`
	// LargeLogo is a base64-encoded PNG logo of up to 100 KB.
	LargeLogo string `json:"largeLogo,omitempty"`
	// RTL aligns the text of the report from right to left.
	RTL bool `json:"rtl,omitempty"`
	// Version is the version of the report layout. ReportVersion overrides it.
	//
	// Deprecated: The API reference deprecates pdf.version. Use ReportVersion.
	Version PDFVersion `json:"version,omitempty"`
	// ReportVersion is the version of the report layout: PDFReportVersionV1,
	// PDFReportVersionV2, PDFReportVersionV3 or PDFReportVersionLatest. The values are
	// lowercase and case-sensitive. It overrides Version when both are set. When neither
	// is set, accounts created before 2025-05-15 get v1 and newer accounts get the latest.
	ReportVersion string `json:"reportVersion,omitempty"`
	// Colors customizes the colors of the report.
	Colors *PDFColors `json:"colors,omitempty"`
}

// PDFColors customizes the colors of a PDF report. Colors are in HEX format, such as
// "#FE5758".
type PDFColors struct {
	// MainStrip is the color of the main strip.
	MainStrip string `json:"mainStrip,omitempty"`
	// Titles is the color of the titles.
	Titles string `json:"titles,omitempty"`
	// Identical is the highlight color of identical matches.
	Identical string `json:"identical,omitempty"`
	// MinorChanges is the highlight color of minor changes.
	MinorChanges string `json:"minorChanges,omitempty"`
	// RelatedMeaning is the highlight color of paraphrased matches.
	RelatedMeaning string `json:"relatedMeaning,omitempty"`
	// AIDetection is the highlight color of AI-generated text.
	AIDetection string `json:"aiDetection,omitempty"`
	// WritingFeedback is the highlight color of Grammar Checker corrections.
	WritingFeedback string `json:"writingFeedback,omitempty"`
}

// AIGeneratedText adds AI Text Detection to a Plagiarism Checker scan.
type AIGeneratedText struct {
	// Detect detects whether the text was written by AI.
	Detect bool `json:"detect"`
	// Sensitivity controls the detection, from 1 to 3; higher values also catch AI text
	// that was edited more heavily. Zero keeps the server default, 2.
	Sensitivity int `json:"sensitivity,omitempty"`
	// Explain adds AI Logic, an explanation of the detection, to the result. It is only
	// returned for some languages.
	Explain *AIGeneratedTextExplain `json:"explain,omitempty"`
}

// AIGeneratedTextExplain turns AI Logic on in a Plagiarism Checker scan.
type AIGeneratedTextExplain struct {
	// Enable turns AI Logic on.
	Enable bool `json:"enable"`
}

// SensitiveDataProtection masks kinds of sensitive data in the scanned document with #
// characters. Each field masks one kind of data.
type SensitiveDataProtection struct {
	// DriversLicense masks driver's license numbers.
	DriversLicense bool `json:"driversLicense,omitempty"`
	// Credentials masks credentials, such as passwords, authentication tokens and cloud
	// keys.
	Credentials bool `json:"credentials,omitempty"`
	// Passport masks passport numbers.
	Passport bool `json:"passport,omitempty"`
	// Network masks network identifiers, such as IP and MAC addresses.
	Network bool `json:"network,omitempty"`
	// URL masks URLs.
	URL bool `json:"url,omitempty"`
	// EmailAddress masks email addresses.
	EmailAddress bool `json:"emailAddress,omitempty"`
	// CreditCard masks credit card numbers.
	CreditCard bool `json:"creditCard,omitempty"`
	// PhoneNumber masks phone numbers.
	PhoneNumber bool `json:"phoneNumber,omitempty"`
}

// ScanWritingFeedback adds the Grammar Checker to a Plagiarism Checker scan.
type ScanWritingFeedback struct {
	// Enable turns the Grammar Checker on.
	Enable bool `json:"enable"`
	// Score sets the weight of each correction category in the overall writing score.
	Score *GrammarScoreWeights `json:"score,omitempty"`
}

// ScanOverview adds a Gen-AI Overview, a summary of the key insights of the scan.
type ScanOverview struct {
	// Enable turns the overview on.
	Enable bool `json:"enable"`
	// IgnoreAIDetection leaves the AI Text Detection results out of the overview.
	IgnoreAIDetection bool `json:"ignoreAIDetection,omitempty"`
	// IgnorePlagiarismDetection leaves the plagiarism results out of the overview.
	IgnorePlagiarismDetection bool `json:"ignorePlagiarismDetection,omitempty"`
	// IgnoreWritingFeedback leaves the Grammar Checker results out of the overview.
	IgnoreWritingFeedback bool `json:"ignoreWritingFeedback,omitempty"`
	// IgnoreAuthorData leaves the author's earlier scans out of the overview.
	IgnoreAuthorData bool `json:"ignoreAuthorData,omitempty"`
}

// InternalAISourceMatch identifies sources in the Copyleaks internal AI database that
// contain AI-generated text. See ScanProperties.InternalAISourceMatch.
type InternalAISourceMatch struct {
	// Enable turns Internal AI Source Match on. The server default is false.
	Enable bool `json:"enable"`
}

// References configures the Citation Checker. See ScanProperties.References.
type References struct {
	// Validate detects, parses and validates every reference in the text.
	Validate bool `json:"validate"`
}

// AISourceMatch identifies the online sources that are suspected of containing
// AI-generated text. See ScanProperties.AISourceMatch.
type AISourceMatch struct {
	// Enable turns AI Source Match on. The server default is false.
	Enable bool `json:"enable"`
}

// CustomMetadata is a key-value pair attached to a document in a Private Cloud Hub.
type CustomMetadata struct {
	// Key is the name of the property.
	Key string `json:"key"`
	// Value is the value of the property.
	Value string `json:"value"`
}

// SubmitAction is what a submit does. See ScanProperties.Action.
type SubmitAction int

const (
	// SubmitActionScan starts the scan right away. It is the default.
	SubmitActionScan SubmitAction = 0
	// SubmitActionCheckCredits only reports the cost of the scan, in a credits-checked
	// webhook. Start the scan later with Client.Start.
	SubmitActionCheckCredits SubmitAction = 1
	// SubmitActionIndex only indexes the content, for free, in the Shared Data Hub or a
	// Private Cloud Hub.
	SubmitActionIndex SubmitAction = 2
)

// ScanMethodAlgorithm is the goal of the scan algorithm. See ScanProperties.ScanMethodAlgorithm.
type ScanMethodAlgorithm int

const (
	// ScanMethodMaximumCoverage prioritizes a higher similarity score. It is the default.
	ScanMethodMaximumCoverage ScanMethodAlgorithm = 0
	// ScanMethodMaximumResults prioritizes finding more sources.
	ScanMethodMaximumResults ScanMethodAlgorithm = 1
)

// DomainsMode sets whether Filters.Domains are included or excluded.
type DomainsMode int

const (
	// DomainsModeInclude only keeps results from Filters.Domains.
	DomainsModeInclude DomainsMode = 0
	// DomainsModeExclude drops results from Filters.Domains. It is the server default.
	DomainsModeExclude DomainsMode = 1
)

// MaskingPolicy sets who sees the content of a document indexed in a Private Cloud Hub
// when it is found as a result.
type MaskingPolicy int

const (
	// MaskingPolicyNoMask shows the content to everyone.
	MaskingPolicyNoMask MaskingPolicy = 0
	// MaskingPolicyMaskOtherUsersFiles masks the content from other users.
	MaskingPolicyMaskOtherUsersFiles MaskingPolicy = 1
	// MaskingPolicyMaskAllFiles masks the content from everyone.
	MaskingPolicyMaskAllFiles MaskingPolicy = 2
)

// PDFVersion is the layout version of a PDF report. See PDF.Version.
//
// Deprecated: It is only used by PDF.Version, which is deprecated. Use PDF.ReportVersion
// and the PDFReportVersion constants.
type PDFVersion int

const (
	// PDFVersion1 is the first report layout. It is the server default for accounts
	// created before 2025-05-15; see PDF.ReportVersion.
	PDFVersion1 PDFVersion = 1
	// PDFVersion2 is the second report layout, which includes AI Text Detection results
	// and cheat detection alerts.
	PDFVersion2 PDFVersion = 2
	// PDFVersion3 is the third report layout.
	PDFVersion3 PDFVersion = 3
)

// The values of PDF.ReportVersion.
const (
	// PDFReportVersionV1 is the first report layout.
	PDFReportVersionV1 = "v1"
	// PDFReportVersionV2 is the second report layout, which includes AI Text Detection
	// results and cheat detection alerts.
	PDFReportVersionV2 = "v2"
	// PDFReportVersionV3 is the third report layout.
	PDFReportVersionV3 = "v3"
	// PDFReportVersionLatest is the latest report layout, currently v3.
	PDFReportVersionLatest = "latest"
)

// Ptr returns a pointer to v. Use it to set the optional pointer fields of the request
// types, such as Filters.IdenticalEnabled: copyleaks.Ptr(false).
func Ptr[T any](v T) *T {
	return &v
}

// maxAnalyticsIDLength is the longest ID of a Course, an Assignment or an Institution.
const maxAnalyticsIDLength = 64

// validate checks the limits of the scanning settings.
func (s *Scanning) validate() error {
	if s == nil {
		return nil
	}
	if e := s.Exclude; e != nil {
		if utf16Len(e.IDPattern) > 36 {
			return invalid("Properties.Scanning.Exclude.IDPattern must be 1 to 36 characters")
		}
		if err := checkStrings("Properties.Scanning.Exclude.BacklinksDomains", e.BacklinksDomains, 10, 256); err != nil {
			return err
		}
		if err := checkStrings("Properties.Scanning.Exclude.Text", e.Text, 5, 30); err != nil {
			return err
		}
	}
	if s.Include != nil && utf16Len(s.Include.IDPattern) > 36 {
		return invalid("Properties.Scanning.Include.IDPattern must be 1 to 36 characters")
	}
	return nil
}

// checkStrings checks that list has at most maxItems strings, each 1 to maxLength
// characters. name names the list in the error.
func checkStrings(name string, list []string, maxItems, maxLength int) error {
	if len(list) > maxItems {
		return invalid(fmt.Sprintf("%s must list at most %d items", name, maxItems))
	}
	for _, v := range list {
		if n := utf16Len(v); n < 1 || n > maxLength {
			return invalid(fmt.Sprintf("each of %s must be 1 to %d characters", name, maxLength))
		}
	}
	return nil
}

// validate checks the scan properties that the server requires.
func (p *ScanProperties) validate() error {
	if p.Webhooks.Status == "" {
		return invalid("Properties.Webhooks.Status is required")
	}
	if utf16Len(p.DeveloperPayload) > 512 {
		return invalid("Properties.DeveloperPayload must be at most 512 characters")
	}
	if p.SensitivityLevel != 0 && (p.SensitivityLevel < 1 || p.SensitivityLevel > 5) {
		return invalid("Properties.SensitivityLevel must be 1 to 5")
	}
	if err := p.Scanning.validate(); err != nil {
		return err
	}
	var ids []string
	if p.Course != nil {
		ids = append(ids, p.Course.ID)
	}
	if p.Assignment != nil {
		ids = append(ids, p.Assignment.ID)
	}
	if p.Institution != nil {
		ids = append(ids, p.Institution.ID)
	}
	for _, id := range ids {
		if utf16Len(id) > maxAnalyticsIDLength {
			return invalid(fmt.Sprintf("Properties.Course.ID, Assignment.ID and Institution.ID must be at most %d characters", maxAnalyticsIDLength))
		}
	}
	if a := p.AIGeneratedText; a != nil && a.Sensitivity != 0 && (a.Sensitivity < 1 || a.Sensitivity > 3) {
		return invalid("Properties.AIGeneratedText.Sensitivity must be 1 to 3")
	}
	switch p.DisplayLanguage {
	case "", "en", "es", "pt", "fr", "de", "it":
	default:
		return invalid(`Properties.DisplayLanguage must be "en", "es", "pt", "fr", "de" or "it"`)
	}
	if p.PDF != nil {
		switch p.PDF.ReportVersion {
		case "", PDFReportVersionV1, PDFReportVersionV2, PDFReportVersionV3, PDFReportVersionLatest:
		default:
			return invalid(`Properties.PDF.ReportVersion must be "v1", "v2", "v3" or "latest"`)
		}
	}
	return nil
}
