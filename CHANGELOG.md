# Changelog

## 1.1.0

This release adds to the API of 1.0.0. It removes, renames and changes no function, type or field.

### Added

- Image Plagiarism Detector: `CheckImagePlagiarism` searches the web for copies of an image.
- AI Video Detection: `SubmitAIVideo` submits a video URL, and `ParseAIVideoWebhook` reads the result.
- AI Image Detection in the background: `SubmitAIImageAsync` submits an image URL, and `ParseAIImageWebhook`
  reads the result.
- `DetectAIImageUpload` uploads the image as `multipart/form-data`, without base64.
- AI Image Detection options: `MaskType` (`AIImageMaskBinary`, `AIImageMaskHeatmap`, `AIImageMaskNone`) and
  `ObjectDetection`, with the response fields `Result.Labels` and `DetectedObjects`. In `AIImageDetectionRequest`
  and `AIImageDetectionResponse`, `ObjectDetection` and `DetectedObjects` are pointers to slices, such as
  `&[]copyleaks.AIImageObject{{Object: "face"}}`, so that both types stay comparable, as in 1.0.0.
- `AIImageDetectionResponse.Error` and `AIVideoDetectionResponse.Error` (`AsyncError`) tell why a check submitted
  with `SubmitAIImageAsync` or `SubmitAIVideo` failed. Check `Error` first in the webhook handler.
- Lookups: `AITextCreditsBalance`, `RepositoryInfo` (Private Cloud Hub), `AllowedCrossLanguages` and
  `SupportedAITextFileTypes`. `AllowedCrossLanguages` maps language names to codes, as the server sends them,
  and `LanguageCodes.Codes` lists the codes.
- Scan properties: `SensitiveDataProtection`, `WritingFeedback`, `Overview`, `InternalAISourceMatch`,
  `References` (Citation Checker), `AIGeneratedText.Sensitivity` and `AIGeneratedText.Explain`, `ScanTimeZone`,
  `Course`, `Assignment` and `Institution`, `Scanning.Include`, `Scanning.Exclude.BacklinksDomains` and
  `Scanning.Exclude.Text` (pointers to slices, such as `&[]string{"example.com"}`, so that `ScanningExclude` stays
  comparable), `Indexing.CopyleaksDB`, and the PDF colors `AIDetection` and `WritingFeedback`.
- `URLSubmission.Verb` and `URLSubmission.Headers`, to fetch pages that need a POST or authentication.
- Export: the `AIDetection`, `WritingFeedback` and `Overview` targets, and `CompletionWebhookHeaders`.
- Types and parsers for the files that Export uploads: `ParseExportedResult`, `ParseExportedCrawledVersion`,
  `ParseExportedAIDetection`, `ParseExportedWritingFeedback` and `ParseExportedOverview`.
- `WithRetry` turns on retries of HTTP 429, HTTP 503 and connection errors, such as a timeout or a refused, reset or
  dropped connection, with exponential backoff and the server's `Retry-After` delay. It does not retry an invalid
  TLS certificate or an unknown host. It only retries calls that are safe to send again, and does not retry
  `Export` after HTTP 503. Retries stay off by default.
- `APIError.RetryAfter` reports the `Retry-After` delay of a failed response.
- `FileSubmission.Content` and `OCRSubmission.Content` take the file as it is, so that you do not need to
  encode it to base64 yourself.
- `WithTimeout` sets the request timeout of the default HTTP client, and `WithMaxJSONResponseSize` raises the
  16 MiB limit of JSON responses.

### Changed

- Uploads use much less memory. The SDK builds the request body in a single buffer of the right size, so a
  large file costs about 1.4 times its size with `Content`, instead of 4 to 6 times. The multipart uploads cost
  about one copy of the file.
- Request bodies no longer escape `<`, `>` and `&` in text, which made text with HTML or code larger.
- Reading the AI Text Detection result of a scan alert decodes the data once instead of twice.
- `PDF.Version`, the `PDFVersion` type and `AITextDetectionResult.Probability` are marked as deprecated, as in
  the API reference. They still work.

### Fixed

- The completed and new-result webhooks no longer drop fields that the server sends: `Results.InternalAIData`,
  `CompletedWebhook.ReferencesValidation`, `CompletedWebhook.WritingFeedback`, and `ExpectedCredits`,
  `DetectedLanguage` and `Enabled` in the scanned document.
- `SupportedFileTypes` calls the documented path, `/v3/miscellaneous/supported-plagiarism-file-types`. The
  response is the same.

## 1.0.0

First release of the Copyleaks Go SDK. It covers:

- Login, and a local token expiry check (`VerifyAuthToken`).
- AI Text Detection (`DetectAIText`) and AI Image Detection (`DetectAIImage`).
- Plagiarism Checker: submit a file, an image (OCR) or a URL, start, delete, export, and resend webhooks.
- Scan properties `aiSourceMatch` (`AISourceMatch`), `displayLanguage` (`DisplayLanguage`) and
  `pdf.reportVersion` (`PDF.ReportVersion`, with the `PDFReportVersion...` constants). The display language and the report version are checked against the values the server accepts.
- Grammar Checker (`CheckGrammar`) and its correction types (`GrammarCorrectionTypes`).
- Text Moderation (`ModerateText`).
- Credit balance, usage history (CSV), release notes, supported file types and OCR languages.
- Webhook payload types and parsers for the completed, error, creditsChecked, indexed, new-result and
  export-completed webhooks, and the AI Text Detection result of the `suspected-ai-text` alert.
- Typed errors: `*APIError`, with `ErrUnauthorized`, `ErrRateLimited` and `ErrUnderMaintenance` for `errors.Is`.
- Safety limits: base URLs must use https (http only for localhost), and response bodies are capped
  (`ErrResponseTooLarge`: 16 MiB of JSON, 64 MiB of CSV).
