package copyleaks

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
	// CustomMetadata is attached to the document in a Private Cloud Hub and added to the
	// results that match it.
	CustomMetadata []CustomMetadata `json:"customMetadata,omitempty"`
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
	// Version is the version of the report layout.
	Version PDFVersion `json:"version,omitempty"`
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
}

// AIGeneratedText adds AI Text Detection to a Plagiarism Checker scan.
type AIGeneratedText struct {
	// Detect detects whether the text was written by AI.
	Detect bool `json:"detect"`
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
type PDFVersion int

const (
	// PDFVersion1 is the first report layout. It is the server default.
	PDFVersion1 PDFVersion = 1
	// PDFVersion2 is the second report layout, which includes AI Text Detection results
	// and cheat detection alerts.
	PDFVersion2 PDFVersion = 2
	// PDFVersion3 is the third report layout.
	PDFVersion3 PDFVersion = 3
)

// Ptr returns a pointer to v. Use it to set the optional pointer fields of the request
// types, such as Filters.IdenticalEnabled: copyleaks.Ptr(false).
func Ptr[T any](v T) *T {
	return &v
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
	if p.Scanning != nil && p.Scanning.Exclude != nil && utf16Len(p.Scanning.Exclude.IDPattern) > 36 {
		return invalid("Properties.Scanning.Exclude.IDPattern must be 1 to 36 characters")
	}
	return nil
}
