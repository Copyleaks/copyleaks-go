# Changelog

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
