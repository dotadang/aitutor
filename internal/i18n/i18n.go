package i18n

import (
	"fmt"
	"strings"
	"sync"

	"github.com/naorpeled/aitutor/pkg/types"
)

type Language string

const (
	English             Language = "en"
	SimplifiedChinese   Language = "zh-CN"
	defaultLanguage     Language = English
)

var (
	mu              sync.RWMutex
	currentLanguage = defaultLanguage
)

func NormalizeLanguage(raw string) Language {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "zh", "zh-cn", "zh_hans", "zh-hans", "zh_cn":
		return SimplifiedChinese
	case "en", "en-us", "en_us", "":
		return English
	default:
		return defaultLanguage
	}
}

func SetLanguage(lang Language) {
	mu.Lock()
	currentLanguage = NormalizeLanguage(string(lang))
	mu.Unlock()
}

func CurrentLanguage() Language {
	mu.RLock()
	defer mu.RUnlock()
	return currentLanguage
}

func Languages() []Language {
	return []Language{English, SimplifiedChinese}
}

func LanguageName(lang Language) string {
	switch NormalizeLanguage(string(lang)) {
	case SimplifiedChinese:
		return "简体中文"
	default:
		return "English"
	}
}

func Text(text string) string {
	if CurrentLanguage() != SimplifiedChinese {
		return text
	}
	if translated, ok := zhCN[text]; ok {
		return translated
	}
	return text
}

func Textf(format string, args ...interface{}) string {
	return fmt.Sprintf(Text(format), args...)
}

func TierLabel(t types.Tier) string {
	return Text(t.String())
}

func AnswerVariants(answer string) []string {
	variants := []string{answer}
	translated := Text(answer)
	if translated != answer {
		variants = append(variants, translated)
	}
	switch answer {
	case "persona":
		variants = append(variants, "角色")
	case "junior":
		variants = append(variants, "初级")
	}
	return variants
}
