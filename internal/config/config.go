package config

import (
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.yaml.in/yaml/v3"
	"golang.org/x/text/language"
)

// Configuration management

var (
	LanguageBundle *i18n.Bundle
	once           sync.Once
)

//go:embed locale.*.yaml
var LocaleFS embed.FS

type Language string

var Languages = []Language{
	"en", "he", "ru", "ar",
}

var msgNotFoundError *i18n.MessageNotFoundErr

func Init() {
	LanguageBundle = i18n.NewBundle(language.English)
	LanguageBundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)
	for _, lang := range Languages {
		_, err := LanguageBundle.LoadMessageFileFS(LocaleFS, fmt.Sprintf("locale.%s.yaml", lang))
		if err != nil {
			slog.Error("failed loading translation", "error", err, "lang", lang)
			os.Exit(2)
		}

	}
}

func GetText(id string, lang Language) string {
	once.Do(Init)
	result, err := i18n.NewLocalizer(LanguageBundle, string(lang)).Localize(&i18n.LocalizeConfig{MessageID: id})
	if err != nil {
		slog.Error("GetText localizer error",
			"id", id,
			"lang", lang,
			"error", err,
		)
	}
	return result
}

func GetTextOptional(id string, lang Language, def string) string {
	once.Do(Init)
	localizer := i18n.NewLocalizer(LanguageBundle, string(lang))
	result, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: id})
	if err != nil {
		if errors.As(err, &msgNotFoundError) {
			return def
		}
		slog.Error("unexpected error localizing",
			"id", id,
			"lang", lang,
			"def", def,
			"error", err,
		)
	}
	return result

}
