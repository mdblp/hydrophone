package localize

import "fmt"

type PreviewLokalise struct {
}

// getLocalizedContentPart returns translated content part based on key and locale
func (l *PreviewLokalise) Localize(key string, locale string, data map[string]interface{}) (string, error) {
	return fmt.Sprintf("{.%s.}", key), nil
}
