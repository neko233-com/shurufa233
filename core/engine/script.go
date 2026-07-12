package engine

// convertScriptText remains the single candidate/commit rendering boundary so
// dictionary import code does not leak presentation policy into callers. The
// production IME is simplified-Chinese only, therefore text is intentionally
// returned unchanged.
func convertScriptText(text string, _ string) string {
	return text
}
