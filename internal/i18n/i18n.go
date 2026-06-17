// Package i18n provides localized message strings for the supported chat
// languages. Strings live in embedded locale.<lang>.yaml files.
package i18n

import (
	"embed"
	"errors"
	"fmt"
	"sync"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// Language is an ISO-639-1 code identifying a supported chat language.
type Language string

// Languages are all supported languages. English is the fallback.
var Languages = []Language{"en", "he", "ru", "ar"}

//go:embed locale.*.yaml
var localeFS embed.FS

var (
	bundle      *goi18n.Bundle
	loadOnce    sync.Once
	notFoundErr *goi18n.MessageNotFoundErr
)

func load() {
	bundle = goi18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	for _, lang := range Languages {
		name := fmt.Sprintf("locale.%s.yaml", lang)
		if _, err := bundle.LoadMessageFileFS(localeFS, name); err != nil {
			// Locale files are embedded, so a failure here is a build-time
			// programming error, not a runtime condition.
			panic(fmt.Sprintf("i18n: loading %s: %v", name, err))
		}
	}
}

func localizer(lang Language) *goi18n.Localizer {
	loadOnce.Do(load)
	return goi18n.NewLocalizer(bundle, string(lang))
}

// Text returns the localized string for id, or logs and returns an empty
// string if the id is missing.
func Text(id string, lang Language) string {
	result, err := localizer(lang).Localize(&goi18n.LocalizeConfig{MessageID: id})
	if err != nil {
		mlog.Error("i18n missing message", mlog.String("id", id), mlog.Any("lang", lang), mlog.Err(err))
	}
	return result
}

// TextOr returns the localized string for id, or def if the id is missing.
func TextOr(id string, lang Language, def string) string {
	result, err := localizer(lang).Localize(&goi18n.LocalizeConfig{MessageID: id})
	if err != nil {
		if errors.As(err, &notFoundErr) {
			return def
		}
		mlog.Error("i18n localize error", mlog.String("id", id), mlog.Any("lang", lang), mlog.Err(err))
	}
	return result
}
