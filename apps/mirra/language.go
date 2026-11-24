package main

import (
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"
	"github.com/yldm-ai/mirra/apps/mirra/locales"
)

// keyLanguage keeps the language chosen in the settings; without it the app follows the system's.
const keyLanguage = "language"

// rememberLanguage shows the language chosen last time.
func rememberLanguage(cx *core.Scope) {
	if lang, ok := cx.Storage().GetItem(keyLanguage); ok && lang != "" {
		cx.SetLocale(lang)
	}
}

// languageRow chooses the language the app shows: the system's, or one of its catalogs', each named in itself.
func (s *studio) languageRow(cx *core.Scope) core.Node {
	t := locales.For(cx)
	store := cx.Storage()
	chosen, _ := store.GetItem(keyLanguage)
	lang := core.NewSignal(cx, chosen)
	options := []ui.Option[string]{{Value: "", Label: t.LanguageSystem(), TestID: "language-system"}}
	for _, l := range locales.Catalog.Languages() {
		options = append(options, ui.Option[string]{Value: l, Label: locales.In(l).LanguageName(), TestID: "language-" + l})
	}
	picker := ui.Select[string]{Label: t.Language(), Options: options, Value: lang.Get, Presentation: s.selectPresentation(), TestID: "language", Style: core.Style{Width: core.Pt(180)},
		OnChange: func(v string) {
			lang.Set(v)
			cx.SetLocale(v)
			if v == "" {
				store.RemoveItem(keyLanguage)
				return
			}
			if err := store.SetItem(keyLanguage, v); err != nil {
				s.toastError(locales.For(cx).SaveFailed(), err)
			}
		}}
	return ui.FormRow{Title: t.Language(), Control: picker, TestID: "language-row"}
}
