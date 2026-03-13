package viz

import "github.com/naorpeled/aitutor/internal/i18n"

func t(text string) string {
	return i18n.Text(text)
}

func tf(format string, args ...interface{}) string {
	return i18n.Textf(format, args...)
}
