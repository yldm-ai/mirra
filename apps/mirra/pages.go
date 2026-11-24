package main

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"github.com/nsgo-dev/nsgo/updater"
	"github.com/yldm-ai/mirra/apps/mirra/locales"
)

// ---- A shot ----

// shotEyebrow says what the picture on show is: a new try, by number, or a work in the lookbook.
func shotEyebrow(t locales.Messages, sh *shot) string {
	if sh.Saved != nil {
		return t.EyebrowLookbookNo(workNo(sh.Saved.No))
	}
	return t.EyebrowNewLook(sh.Attempt)
}

// swapOutfits are the outfits one tap puts on the same face and pose: the favorites, the user's clothes, then the gallery's in the chip on show, without the one being worn.
func (s *studio) swapOutfits() []Item {
	var out []Item
	seen := map[string]bool{}
	if sh := s.shot.Get(); sh != nil && sh.Req.Costume != nil {
		seen[sh.Req.Costume.ID] = true
	}
	add := func(items []Item) {
		for _, it := range items {
			if len(out) == 8 {
				return
			}
			if !seen[it.ID] {
				seen[it.ID] = true
				out = append(out, it)
			}
		}
	}
	add(s.favorites.Get())
	add(s.uploads.Get())
	add(loaded(s.catalogs[KindCostume]).InGroup(s.filter[KindCostume].Get(), s.adult[KindCostume].Get()))
	return out
}

func (s *studio) swapRow(cx *core.Scope) core.Node {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	return core.View{Style: core.Style{Gap: 10}, Children: []core.Node{
		caption(cx, t.SwapOutfit),
		core.Scroll{Horizontal: true, HideIndicator: true, Style: core.Style{Height: core.Pt(84), MinHeight: core.Pt(84), FlexShrink: 0}, Children: []core.Node{
			core.Dynamic(func(cx *core.Scope) core.Node {
				var nodes []core.Node
				for i, it := range s.swapOutfits() {
					it := it
					nodes = append(nodes, core.View{Role: core.RoleButton, Label: t.SwapTo(itemName(t, &it)), TestID: "swap-" + string(rune('0'+i)),
						OnPress: func(core.PressEvent) { s.regenerate(&it) },
						StyleFunc: func() core.Style {
							return core.Style{Width: core.Pt(60), Height: core.Pt(80), CornerRadius: 12, Overflow: core.Hidden, Background: colors.Inset(),
								BorderWidth: 1, BorderColor: colors.Border()}
						},
						Children: []core.Node{core.Image{Source: s.src(it.Thumb), ResizeMode: core.Contain, Style: core.Style{Width: core.Pct(100), Height: core.Pct(100)}}},
					})
				}
				return core.View{Style: core.Style{Direction: core.Row, Gap: 8}, Children: nodes}
			}),
		}},
	}}
}

// usedRefs are the face and pose a shot was made from, small, over the picture's corner.
func (s *studio) usedRefs(cx *core.Scope, sh *shot) core.Node {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	var nodes []core.Node
	if sh.Req.Face != nil {
		nodes = append(nodes, core.View{Label: t.UsedFace(), Role: core.RoleImage, Children: []core.Node{s.thumb(cx, KindFace, func() *Item { return sh.Req.Face }, 48, 48, true)}})
	}
	if sh.Req.Pose != nil {
		nodes = append(nodes, core.View{Label: t.UsedPose(), Role: core.RoleImage, Children: []core.Node{s.thumb(cx, KindPose, func() *Item { return sh.Req.Pose }, 48, 64, false)}})
	}
	return core.View{StyleFunc: func() core.Style {
		return core.Style{Direction: core.Row, Gap: 8, AlignItems: core.AlignEnd, Padding: core.All(4), CornerRadius: 16, Background: colors.Background().Faded(0.6)}
	}, Children: nodes}
}

// saveButton keeps the shot in Works, or says it is there.
func (s *studio) saveButton(cx *core.Scope, sh *shot) core.Node {
	t := locales.For(cx)
	if sh.Saved != nil {
		return ui.Button{Text: t.InWorks(), Icon: lucide.Check, Variant: ui.ButtonSecondary, Flex: 1, TestID: "saved", Disabled: func() bool { return true },
			Style: core.Style{CornerRadius: 999}}
	}
	return ui.Button{Text: t.SaveToWorks(), Flex: 1, TestID: "save-to-works", Style: core.Style{CornerRadius: 999}, OnPress: func() { s.saveShot(false) }}
}

func (s *studio) favoriteButton(cx *core.Scope, sh *shot) core.Node {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	fav := sh.Saved != nil && sh.Saved.Favorite
	label := t.Favorite()
	if fav {
		label = t.Unfavorite()
	}
	return ui.IconButton{Icon: lucide.Heart, Label: label, Variant: ui.IconButtonRaised, Selected: func() bool { return fav }, TestID: "favorite",
		Tint: func() core.Color {
			if fav {
				return colors.Danger()
			}
			return colors.Text()
		}, OnPress: s.toggleFavorite}
}

// shotPage is the phone's result: the picture, its actions over it, then its title, the outfits to swap to, Generate again and Save.
func (s *studio) shotPage(cx *core.Scope) core.Node {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	return core.Dynamic(func(cx *core.Scope) core.Node {
		sh := s.shot.Get()
		if sh == nil {
			return nil
		}
		picture := core.View{Style: fill, Children: []core.Node{
			core.Image{Source: s.src(sh.Path), ResizeMode: core.Contain, TestID: "shot-image", Label: lookTitle(t, sh.Req), Style: core.Style{Width: core.Pct(100), Height: core.Pct(100)}},
			core.View{Style: core.Style{Position: core.Absolute, Top: core.Pt(12), Left: core.Pt(16), Right: core.Pt(16), Direction: core.Row, Gap: 10}, Children: []core.Node{
				ui.IconButton{Icon: lucide.X, Label: t.Close(), Variant: ui.IconButtonRaised, TestID: "close-shot", OnPress: s.closeShot},
				core.View{Style: core.Style{FlexGrow: 1}},
				ui.IconButton{Icon: lucide.Download, Label: t.Export(), Variant: ui.IconButtonRaised, TestID: "export", OnPress: func() { s.saveFile(cx, sh.Path) }},
				ui.IconButton{Icon: lucide.Trash, Label: t.Delete(), Variant: ui.IconButtonRaised, TestID: "delete", OnPress: s.discardShot},
				s.favoriteButton(cx, sh),
			}},
			core.View{Style: core.Style{Position: core.Absolute, Right: core.Pt(16), Bottom: core.Pt(12)}, Children: []core.Node{s.usedRefs(cx, sh)}},
		}}
		return core.View{TestID: "shot", StyleFunc: func() core.Style { return fill.Merge(core.Style{Background: colors.Background()}) }, Children: []core.Node{
			picture,
			core.View{Style: core.Style{Padding: core.Symmetric(14, 20), Gap: 14, PaddingEnv: core.EnvEdges{Bottom: core.EnvSafeArea}}, Children: []core.Node{
				core.View{Style: core.Style{Gap: 4}, Children: []core.Node{
					eyebrow(cx, func() string { return shotEyebrow(t, sh) }),
					display(func() string { return lookTitle(t, sh.Req) }, 26),
				}},
				s.swapRow(cx),
				core.View{Style: core.Style{Direction: core.Row, Gap: 10}, Children: []core.Node{
					ui.Button{Text: t.Regenerate(), Icon: lucide.RefreshCw, Variant: ui.ButtonOutline, Flex: 1, TestID: "regenerate", Style: core.Style{CornerRadius: 999},
						OnPress: func() { s.regenerate(nil) }},
					s.saveButton(cx, sh),
				}},
			}},
		}}
	})
}

// ---- Works ----

// worksPage is the lookbook: every saved picture, newest first, or only the favorites.
func (s *studio) worksPage(cx *core.Scope, columns int) core.Node {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	shown := core.NewMemoFunc(cx, func() []work {
		ws := s.works.Get()
		if !s.favoritesOn.Get() {
			return ws
		}
		var out []work
		for _, w := range ws {
			if w.Favorite {
				out = append(out, w)
			}
		}
		return out
	}, nil)
	at := func(i int) (work, bool) {
		ws := shown.Get()
		if i < 0 || i >= len(ws) {
			return work{}, false
		}
		return ws[i], true
	}
	favCount := func() int {
		n := 0
		for _, w := range s.works.Get() {
			if w.Favorite {
				n++
			}
		}
		return n
	}
	filter := ui.Tabs[bool]{Label: t.WorksFilter(), Value: s.favoritesOn.Get, OnChange: s.favoritesOn.Set, TestID: "works-filter", Options: []ui.Option[bool]{
		{Value: false, Label: t.All(), TestID: "works-all", Content: core.Text{ContentFunc: func() string { return t.AllCount(len(s.works.Get())) }}},
		{Value: true, Label: t.Favorites(), TestID: "works-favorites", Content: core.Text{ContentFunc: func() string { return t.FavoritesCount(favCount()) }}},
	}}
	empty := func(cx *core.Scope) core.Node {
		title, body := t.NoWorksTitle(), t.NoWorksBody()
		if s.favoritesOn.Get() {
			title, body = t.NoFavoriteWorksTitle(), ""
		}
		children := []core.Node{core.Text{Content: title, Style: core.Style{FontFamily: displayFont, FontSize: 22, FontWeight: core.FontWeightBold}}}
		if body != "" {
			children = append(children, core.Text{Content: body, StyleFunc: func() core.Style {
				return core.Style{FontSize: 14, Color: colors.Muted(), TextAlign: core.TextAlignCenter}
			}}, ui.Button{Text: t.FirstLook(), TestID: "first-look", Style: core.Style{CornerRadius: 999, Margin: core.Edges{Top: 8}},
				OnPress: func() { s.section.Set(sectionStudio) }})
		}
		return core.View{TestID: "no-works", Style: fill.Merge(core.Style{AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 8, Padding: core.All(24)}), Children: children}
	}
	grid := core.List{Style: fill, NumColumns: columns, Gap: 12, ColumnGap: 12, EstimatedSize: 300,
		Count: func() int { return len(shown.Get()) },
		Key: func(i int) any {
			w, _ := at(i)
			return w.Path
		},
		Row: func(cx *core.Scope, row core.ListRow) core.Node {
			current := func() work {
				w, _ := at(row.Index())
				return w
			}
			return core.View{Role: core.RoleButton, LabelFunc: func() string { return t.OpenWork(workNo(current().No)) },
				OnPress: func(core.PressEvent) {
					if w, ok := at(row.Index()); ok {
						s.openWork(w)
					}
				},
				StyleFunc: func() core.Style {
					return core.Style{CornerRadius: 16, Overflow: core.Hidden, Background: colors.Surface(), BorderWidth: 1, BorderColor: colors.Border()}
				},
				Children: []core.Node{
					core.Image{ResizeMode: core.Cover, Style: core.Style{Width: core.Pct(100), AspectRatio: 2.0 / 3.0},
						SourceFunc: func() core.ImageSource { return s.src(current().Path) }},
					core.View{Style: core.Style{Padding: core.Symmetric(8, 10), Gap: 2}, Children: []core.Node{
						core.Text{ContentFunc: func() string { return t.EyebrowLookbookNo(workNo(current().No)) }, StyleFunc: func() core.Style {
							return core.Style{FontSize: 10, LetterSpacing: 1.5, Color: colors.Accent()}
						}},
						core.Text{ContentFunc: func() string {
							w := current()
							name := lookTitle(t, Request{Costume: w.Costume, Pose: w.Pose})
							if w.Favorite {
								name = "♥ " + name
							}
							return name
						}, MaxLines: 1, Style: core.Style{FontSize: 14, FontWeight: core.FontWeightMedium}},
					}},
				},
			}
		},
	}
	children := []core.Node{
		core.View{Style: core.Style{Gap: 4}, Children: []core.Node{eyebrow(cx, t.EyebrowLookbook), display(t.TabWorks, 30)}},
		filter,
	}
	if !s.pics.Lasting() {
		children = append(children, s.pageOnlyNote(cx))
	}
	return core.View{TestID: "works", Style: fill.Merge(core.Style{Gap: 14}), Children: append(children,
		core.Show{When: func() bool { return len(shown.Get()) == 0 }, Then: empty, Else: func(*core.Scope) core.Node { return grid }},
	)}
}

// pageOnlyNote says that in a browser the pictures last as long as the page, so they are exported to be kept.
func (s *studio) pageOnlyNote(cx *core.Scope) core.Node {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	return core.Text{Content: t.PageOnly(), TestID: "page-only", StyleFunc: func() core.Style {
		return core.Style{FontSize: 13, Color: colors.Accent()}
	}}
}

// ---- AI service ----

func (s *studio) resetConnection() {
	s.conn.Set(connUntested)
	s.connErr.Set("")
}

// listModels fetches the image models the chosen provider's key can use, for the model field to offer; nothing for a provider that can't list them, without a key a cloud service needs, or while it is already listing.
func (s *studio) listModels() {
	p, set := s.current()
	cfg, client := s.config(), s.Client
	if !canListModels(p) || set.listing.Get() || strings.TrimSpace(cfg.BaseURL) == "" || !cfg.Local && strings.TrimSpace(cfg.APIKey) == "" {
		return
	}
	set.listing.Set(true)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		ids, err := ListModels(ctx, client, cfg)
		s.cx.Post(func() {
			set.listing.Set(false)
			if err != nil {
				set.modelsErr.Set(explain(locales.For(s.cx), err))
				return
			}
			set.modelsErr.Set("")
			set.models.Set(ids)
		})
	}()
}

// forgetModels drops the listed models, which a new key or address may not offer.
func (s *studio) forgetModels() {
	_, set := s.current()
	set.models.Set(nil)
	set.modelsErr.Set("")
}

func (s *studio) testConnection() {
	cfg, client := s.config(), s.Client
	s.conn.Set(connTesting)
	s.connErr.Set("")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		err := CheckConnection(ctx, client, cfg)
		s.cx.Post(func() {
			if err != nil {
				s.conn.Set(connFailed)
				s.connErr.Set(explain(locales.For(s.cx), err))
				return
			}
			s.conn.Set(connOK)
			s.listModels()
		})
	}()
}

// modelPicker chooses the model among those the key can use, searchable, with a button that lists them again. The chosen model and the preset's always show, listed or not; choosing the preset's keeps the setting following the preset. Its error shows under its card.
func (s *studio) modelPicker(cx *core.Scope, p provider, set providerSettings, setModel func(string)) core.Node {
	t := locales.For(cx)
	value := func() string { return or(set.model.Get(), p.Model) }
	options := func() []ui.Option[string] {
		ids := slices.Clone(set.models.Get())
		for _, id := range []string{p.Model, value()} {
			if id != "" && !slices.Contains(ids, id) {
				ids = append([]string{id}, ids...)
			}
		}
		out := make([]ui.Option[string], len(ids))
		for i, id := range ids {
			out[i] = ui.Option[string]{Value: id, Label: id}
		}
		return out
	}
	picker := ui.Select[string]{Label: t.Model(), OptionsFunc: options, Value: value, Searchable: true, SearchPlaceholder: t.SearchModels(), Empty: t.NoModelMatches(),
		TestID: "model", Style: core.Style{Width: core.Pct(100)},
		OnChange: func(v string) {
			if v == p.Model {
				v = ""
			}
			setModel(v)
		}}
	refresh := ui.IconButton{Icon: lucide.RefreshCw, Label: t.RefreshModels(), TestID: "refresh-models", Size: 32, IconSize: 16, Disabled: set.listing.Get, OnPress: s.listModels}
	return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 4, Width: core.Pt(controlWidth)}, Children: []core.Node{
		core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1}, Children: []core.Node{picker}}, refresh}}
}

// controlWidth is the width of a settings row's control, so the rows' controls line up.
const controlWidth core.Points = 220

// serviceForm is the AI service's settings in parts, which the phone shows in one column and the desktop on two tabs.
type serviceForm struct {
	// intro asks for a service while none is set up; header is the chosen provider with the connection's state and its test.
	intro, header core.Node
	// sections are the chosen provider's settings as cards of rows: the provider, its key and model in main; its address and the picture's size in advanced. They are built in a Dynamic that tracks the provider.
	sections func(cx *core.Scope) (main, advanced []core.Node)
	// changed says whether an advanced setting differs from the preset's.
	changed func() bool
}

// serviceForm sets up the AI service: the provider, its key and where to get one, its model, its address and size, and a test of the connection. Each provider keeps its own settings, so switching back finds them again.
func (s *studio) serviceForm(cx *core.Scope) serviceForm {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	store := cx.Storage()
	secrets := cx.Secrets()
	open := opener(cx)
	setStored := func(field string, sig func(providerSettings) *core.Signal[string]) func(string) {
		return func(v string) {
			p, set := s.current()
			sig(set).Set(v)
			s.resetConnection()
			if field == "url" {
				s.forgetModels()
			}
			if err := store.SetItem(p.storageKey(field), v); err != nil {
				s.toastError(t.SaveFailed(), err)
			}
		}
	}
	setKey := func(v string) {
		_, set := s.current()
		set.key.Set(v)
		s.resetConnection()
		s.forgetModels()
	}
	saveKey := func() {
		s.listModels()
		// A browser has no secure store; the settings say the key lasts as long as the page.
		if !s.keyKept.Get() {
			return
		}
		p, set := s.current()
		account, key := p.keyAccount(), strings.TrimSpace(set.key.Get())
		go func() {
			var err error
			if key == "" {
				err = secrets.Delete(account)
			} else {
				err = secrets.Set(account, key)
			}
			if err == nil {
				return
			}
			cx.Post(func() {
				if errors.Is(err, core.ErrSecretsUnavailable) {
					s.toaster.Show(ui.ToastContent{Type: ui.ToastWarning, Title: t.KeyUnavailable()})
					return
				}
				s.toastError(t.ApiKey(), err)
			})
		}()
	}
	var options []ui.Option[string]
	for _, p := range providers {
		options = append(options, ui.Option[string]{Value: p.ID, Label: p.name(t), TestID: "provider-" + p.ID,
			Leading: core.Icon{SVG: p.icon(), Label: p.name(t), Style: core.Style{FontSize: 18}}})
	}
	picker := ui.Select[string]{Label: t.Provider(), Options: options, Value: s.provider.Get, TestID: "provider", Style: core.Style{Width: core.Pt(controlWidth)},
		OnChange: func(id string) {
			s.provider.Set(id)
			s.showKey.Set(false)
			s.resetConnection()
			if err := store.SetItem(keyProvider, id); err != nil {
				s.toastError(t.SaveFailed(), err)
			}
		}}
	changed := func() bool {
		_, set := s.current()
		return set.url.Get() != "" || set.size.Get() != ""
	}
	privacy := func() string {
		p, _ := s.current()
		return privacyLine(p, s.keyKept.Get(), t)
	}
	sections := func(cx *core.Scope) (main, advanced []core.Node) {
		p, set := s.current()
		keyLabel := t.ApiKey()
		if p.KeyOptional {
			keyLabel = t.ApiKeyOptional()
		}
		keyControl := core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 4, Width: core.Pt(controlWidth)}, Children: []core.Node{
			core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1}, Children: []core.Node{
				core.Dynamic(func(cx *core.Scope) core.Node {
					return ui.Input{Value: set.key.Get, OnChange: setKey, OnBlur: saveKey, Masked: !s.showKey.Get(), Label: keyLabel, Placeholder: p.KeyHint, TestID: "api-key"}
				}),
			}},
			ui.IconButton{TestID: "show-key", Size: 32, IconSize: 16,
				Icon: func() *core.SVG {
					if s.showKey.Get() {
						return lucide.EyeOff()
					}
					return lucide.Eye()
				},
				LabelFunc: func() string {
					if s.showKey.Get() {
						return t.HideKey()
					}
					return t.ShowKey()
				},
				Selected: s.showKey.Get,
				OnPress:  func() { s.showKey.Update(func(v bool) bool { return !v }) }},
		}}
		urlHint := p.BaseURL
		if urlHint == "" {
			urlHint = p.URLHint
		}
		address := ui.FormRow{Title: t.BaseURL(), TestID: "base-url-row", Control: ui.Input{Value: set.url.Get, OnChange: setStored("url", func(ps providerSettings) *core.Signal[string] { return ps.url }),
			Label: t.BaseURL(), Placeholder: urlHint, KeyboardType: core.KeyboardURL, TestID: "base-url", Style: core.Style{Width: core.Pt(controlWidth)}}}
		account := []core.Node{ui.FormRow{Title: t.Provider(), Control: picker}, ui.FormRow{Title: keyLabel, Control: keyControl}}
		if p.KeyURL != "" {
			account = append(account, ui.FormRow{Title: t.GetApiKey(), Detail: host(p.KeyURL), OnPress: open(p.KeyURL), TestID: "get-key"})
		}
		// A provider without an address of its own, such as Azure's, needs the user's: it shows with the key rather than among the advanced settings.
		if p.BaseURL == "" {
			account = append(account, address)
		}
		main = append(main, core.Dynamic(func(cx *core.Scope) core.Node {
			return ui.FormSection{Children: account, Footer: privacy()}
		}))
		modelLabel := t.Model
		if p.Deployment {
			modelLabel = t.Deployment
		}
		setModel := setStored("model", func(ps providerSettings) *core.Signal[string] { return ps.model })
		var model core.Node = ui.Input{Value: set.model.Get, OnChange: setModel, Label: modelLabel(), Placeholder: p.Model, TestID: "model", Style: core.Style{Width: core.Pt(controlWidth)}}
		if canListModels(p) {
			model = s.modelPicker(cx, p, set, setModel)
			// The models are listed once the field shows, then again for a new key or a passed test.
			core.Untracked(cx, func() bool {
				if set.models.Get() == nil {
					s.listModels()
				}
				return true
			})
		}
		modelRow := ui.FormRow{Title: modelLabel(), Control: model}
		main = append(main, core.Dynamic(func(cx *core.Scope) core.Node {
			footer := ""
			if err := set.modelsErr.Get(); err != "" {
				footer = t.ModelsFailed(err)
			}
			return ui.FormSection{Children: []core.Node{modelRow}, Footer: footer, TestID: "model-section"}
		}))

		size := func() string { return or(set.size.Get(), p.Size) }
		sizes := ui.Tabs[string]{Label: t.Size(), Equal: true, Value: size, TestID: "size", Style: core.Style{Width: core.Pt(controlWidth)},
			OnChange: func(v string) {
				// The preset's own size is kept as "", so the setting follows the preset.
				if v == p.Size {
					v = ""
				}
				setStored("size", func(ps providerSettings) *core.Signal[string] { return ps.size })(v)
			},
			Options: []ui.Option[string]{
				{Value: sizePortrait, Label: t.SizePortrait(), TestID: "size-portrait"},
				{Value: sizeSquare, Label: t.SizeSquare(), TestID: "size-square"},
				{Value: sizeLandscape, Label: t.SizeLandscape(), TestID: "size-landscape"},
			}}
		var output []core.Node
		if p.BaseURL != "" {
			output = append(output, address)
		}
		output = append(output, ui.FormRow{Title: t.Size(), Control: sizes})
		reset := ui.FormRow{Title: t.ResetDefaults(), Variant: ui.FormRowAccent, TestID: "reset-defaults", Disabled: func() bool { return !changed() },
			OnPress: func() {
				for _, f := range []struct {
					name string
					sig  *core.Signal[string]
				}{{"url", set.url}, {"size", set.size}} {
					f.sig.Set("")
					store.RemoveItem(p.storageKey(f.name))
				}
				s.resetConnection()
			}}
		advanced = []core.Node{ui.FormSection{Children: output}, ui.FormSection{Children: []core.Node{reset}}}
		return main, advanced
	}
	return serviceForm{
		intro: core.Show{When: func() bool { return !s.configured() }, Then: func(cx *core.Scope) core.Node {
			return core.View{StyleFunc: func() core.Style {
				return core.Style{Padding: core.Symmetric(10, 14), CornerRadius: 12, BorderWidth: 1, BorderColor: colors.Accent().Faded(0.4), Background: colors.Accent().Faded(0.08)}
			}, Children: []core.Node{core.Text{Content: t.ServiceIntro(), StyleFunc: func() core.Style { return core.Style{FontSize: 14, Color: colors.Accent()} }}}}
		}},
		header:   s.serviceHeader(cx),
		sections: sections,
		changed:  changed,
	}
}

// serviceHeader is the chosen provider at a glance: its logo and name, the state of the connection, and the test that checks it.
func (s *studio) serviceHeader(cx *core.Scope) core.Node {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	logo := core.View{StyleFunc: func() core.Style {
		return core.Style{Width: core.Pt(44), Height: core.Pt(44), FlexShrink: 0, CornerRadius: 12, Background: colors.Inset(), AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter}
	}, Children: []core.Node{core.Dynamic(func(cx *core.Scope) core.Node {
		p, _ := s.current()
		return core.Icon{SVG: p.icon(), Label: p.name(t), Style: core.Style{FontSize: 24}}
	})}}
	// state is the connection's state in a word; a failure's reason goes under the row, where it has the card's width.
	state := core.Dynamic(func(cx *core.Scope) core.Node {
		text, testID, color := t.ConnectionUntested(), "conn-untested", colors.Muted
		switch {
		case s.conn.Get() == connTesting:
			text, testID, color = t.ConnectionTesting(), "conn-testing", colors.Accent
		case s.conn.Get() == connOK:
			text, testID, color = t.ConnectionOK(), "conn-ok", colors.Up
		case s.conn.Get() == connFailed:
			text, testID, color = t.ConnectionFailedShort(), "conn-failed-short", colors.Danger
		case !s.configured():
			text, testID = t.StatusUnconfigured(), "conn-unconfigured"
		}
		return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6}, Children: []core.Node{
			core.View{StyleFunc: func() core.Style {
				return core.Style{Width: core.Pt(7), Height: core.Pt(7), CornerRadius: 4, FlexShrink: 0, Background: color()}
			}},
			core.Text{Content: text, TestID: testID, MaxLines: 2, Style: core.Style{FlexShrink: 1}, StyleFunc: func() core.Style { return core.Style{FontSize: 13, Color: color()} }},
		}}
	})
	reason := core.Show{When: func() bool { return s.conn.Get() == connFailed }, Then: func(cx *core.Scope) core.Node {
		return core.Text{ContentFunc: func() string { return t.ConnectionFailed(s.connErr.Get()) }, TestID: "conn-failed", StyleFunc: func() core.Style {
			return core.Style{FontSize: 13, LineHeight: 18, Color: colors.Danger()}
		}}
	}}
	name := core.Text{ContentFunc: func() string {
		p, _ := s.current()
		return p.name(t)
	}, MaxLines: 1, Style: core.Style{FontSize: 16, FontWeight: core.FontWeightSemibold}}
	test := core.Dynamic(func(cx *core.Scope) core.Node {
		text := t.TestConnection()
		if s.conn.Get() == connOK || s.conn.Get() == connFailed {
			text = t.TestAgain()
		}
		return ui.Button{Text: text, TestID: "test-connection", Style: core.Style{CornerRadius: 999},
			Busy: func() bool { return s.conn.Get() == connTesting }, Disabled: func() bool { return strings.TrimSpace(s.config().BaseURL) == "" }, OnPress: s.testConnection}
	})
	return core.View{TestID: "service-header", StyleFunc: func() core.Style {
		return core.Style{Gap: 12, Padding: core.All(14), CornerRadius: ui.RadiusCard, BorderCurve: core.BorderCurveContinuous, Background: ui.Alpha(colors.Text(), 0.045)}
	}, Children: []core.Node{
		core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 14}, Children: []core.Node{
			logo,
			core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 4}, Children: []core.Node{name, state}},
			core.View{Style: core.Style{FlexShrink: 0}, Children: []core.Node{test}},
		}},
		reason,
	}}
}

// host is a URL's host, what a link row names it by.
func host(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		return strings.TrimPrefix(u.Host, "www.")
	}
	return rawURL
}

// serviceSettings is the AI service in one column, as the phone's Me tab shows it, with the address and size under Advanced.
func (s *studio) serviceSettings(cx *core.Scope) core.Node {
	t := locales.For(cx)
	f := s.serviceForm(cx)
	// The settings are the chosen provider's, drawn again when another is chosen.
	sections := core.Dynamic(func(cx *core.Scope) core.Node {
		main, advanced := f.sections(cx)
		// The advanced settings open on their own when the user changed one; read untracked, so changing a field does not draw the settings again.
		open := core.NewSignal(cx, core.Untracked(cx, f.changed))
		return core.View{Style: core.Style{Gap: 16}, Children: append(main, ui.Collapsible{Text: t.AdvancedSettings(), Open: open.Get, OnOpenChange: open.Set, TestID: "advanced",
			Content: core.View{Style: core.Style{Gap: 16, Padding: core.Edges{Top: 12}}, Children: advanced}})}
	})
	return core.View{Style: core.Style{Gap: 16}, Children: []core.Node{f.intro, f.header, sections}}
}

// opener opens a link in the browser.
func opener(cx *core.Scope) func(url string) func() {
	return func(url string) func() { return func() { cx.OpenURL(url, func(bool) {}) } }
}

// appRows are the app's version, the update check where it updates itself, its source and its license.
func (s *studio) appRows(cx *core.Scope) []core.Node {
	t := locales.For(cx)
	open := opener(cx)
	version := core.App().Version
	if version == "" {
		version = t.VersionDevelopment()
	}
	rows := []core.Node{ui.FormRow{Title: t.Version(), Detail: version, OnPress: open(repoURL + "/releases"), TestID: "version"}}
	if updatable(cx.Env()) {
		// updater shows what it finds: Sparkle's or WinSparkle's window on the desktop, an alert that installs the signed APK on Android.
		rows = append(rows, ui.FormRow{Title: t.CheckForUpdates(), OnPress: func() { updater.Check(cx) }, TestID: "check-updates"})
	}
	return append(rows,
		ui.FormRow{Title: t.SourceCode(), Detail: "GitHub", OnPress: open(repoURL)},
		ui.FormRow{Title: t.License(), Detail: "MIT", OnPress: open(repoURL + "/blob/main/LICENSE")})
}

// sourceRows are where the three galleries and the backgrounds come from.
func (s *studio) sourceRows(cx *core.Scope) []core.Node {
	t := locales.For(cx)
	open := opener(cx)
	return []core.Node{
		ui.FormRow{Title: t.SourcesPose(), Detail: "POSE-DB-01", OnPress: open(DefaultSites.Pose)},
		ui.FormRow{Title: t.SourcesOutfit(), Detail: "COSTUME-DB-01", OnPress: open(DefaultSites.Costume)},
		ui.FormRow{Title: t.SourcesFace(), Detail: "FACE REFERENCE · CC0", OnPress: open(DefaultSites.Face)},
		ui.FormRow{Title: t.SourcesBackgrounds(), Detail: "StockSnap · rawpixel · CC0", OnPress: open(repoURL + "/blob/main/NOTICE.md#backgrounds")},
	}
}

// updatable says whether the app updates itself where it runs: the desktop and Android copies from the latest release. The web build is always the latest, and the iOS app, which is built from source, is in no store to ask.
func updatable(env *core.Env) bool {
	if env.Simulated() {
		return false
	}
	switch env.OS() {
	case core.OSMacOS, core.OSWindows, core.OSLinux, core.OSAndroid:
		return true
	}
	return false
}

// appIdentity is the app's icon, name and the line under it.
func appIdentity(cx *core.Scope, line func() string) core.Node {
	t := locales.For(cx)
	return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 14}, Children: []core.Node{
		core.Image{Source: core.ImageSource{URI: "mirra-icon", Data: appIcon}, ResizeMode: core.Cover, Label: t.Brand(),
			Style: core.Style{Width: core.Pt(56), Height: core.Pt(56), FlexShrink: 0, CornerRadius: 14, Overflow: core.Hidden}},
		core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 4}, Children: []core.Node{
			core.Text{Content: t.Brand() + " " + t.BrandTagline(), Style: core.Style{FontSize: 18, FontWeight: core.FontWeightBold}},
			caption(cx, line),
		}},
	}}
}

// aboutHeader is the app's icon, name and terms, heading the desktop's About and the phone's Me tab.
func aboutHeader(cx *core.Scope) core.Node {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	return core.View{StyleFunc: func() core.Style {
		return core.Style{Padding: core.All(16), CornerRadius: ui.RadiusCard, BorderCurve: core.BorderCurveContinuous, Background: ui.Alpha(colors.Text(), 0.045)}
	}, Children: []core.Node{appIdentity(cx, t.OpenSourceTerms)}}
}

// generalRows are the app-wide settings: the language and whether the galleries' 18+ categories show without asking.
func (s *studio) generalRows(cx *core.Scope) []core.Node {
	t := locales.For(cx)
	return []core.Node{
		s.languageRow(cx),
		ui.FormRow{Title: t.AdultSetting(), Detail: t.AdultSettingDetail(), TestID: "adult-setting",
			Control: ui.Switch{Checked: s.adultOK.Get, OnChange: s.setAdultOK, Label: t.AdultSetting(), TestID: "adult-switch"}},
	}
}

// settingsPage is the phone's Me tab: the app, its settings, its AI service and where it comes from.
func (s *studio) settingsPage(cx *core.Scope) core.Node {
	t := locales.For(cx)
	return core.Screen{HeaderHidden: true, Title: t.TabMe(), Style: fill, Children: []core.Node{
		core.Scroll{Style: fill, Children: []core.Node{
			core.View{TestID: "settings", Style: core.Style{Padding: core.All(20), Gap: 20, PaddingEnv: core.EnvEdges{Top: core.EnvSafeArea}}, Children: []core.Node{
				core.View{Style: core.Style{Gap: 4}, Children: []core.Node{eyebrow(cx, t.EyebrowSettings), display(t.TabMe, 30)}},
				aboutHeader(cx),
				ui.FormSection{Title: t.SettingsGeneral(), Children: s.generalRows(cx)},
				core.View{Style: core.Style{Gap: 10}, Children: []core.Node{display(t.AiService, 20), s.serviceSettings(cx)}},
				ui.FormSection{Title: t.AboutSection(), Children: s.appRows(cx)},
				ui.FormSection{Title: t.SourcesSection(), Children: s.sourceRows(cx), Footer: t.SourcesFooter()},
			}},
		}},
	}}
}

// The desktop settings' tabs.
const (
	settingsGeneral  = "general"
	settingsService  = "service"
	settingsAdvanced = "advanced"
	settingsAbout    = "about"
	settingsSources  = "sources"
)

// settingsDialog is the desktop's settings: the app-wide settings, the AI service, its advanced settings, the app and the galleries' sources, each on a tab down the side.
func (s *studio) settingsDialog(cx *core.Scope) core.Node {
	t := locales.For(cx)
	f := s.serviceForm(cx)
	column := func(children ...core.Node) core.Node {
		return core.View{Style: core.Style{Gap: 16}, Children: children}
	}
	page := func(id, title string, icon func() *core.SVG, content core.Node) ui.SettingsCategory {
		return ui.SettingsCategory{ID: id, Title: title, Icon: icon, Groups: []ui.SettingsGroup{{ID: id, Items: []ui.SettingsItem{{ID: id, Content: content}}}}}
	}
	categories := []ui.SettingsCategory{
		page(settingsGeneral, t.SettingsGeneral(), lucide.Languages, ui.FormSection{Children: s.generalRows(cx)}),
		page(settingsService, t.AiService(), lucide.Plug, column(f.intro, f.header, core.Dynamic(func(cx *core.Scope) core.Node {
			main, _ := f.sections(cx)
			return column(main...)
		}))),
		page(settingsAdvanced, t.AdvancedSettings(), lucide.SlidersHorizontal, core.Dynamic(func(cx *core.Scope) core.Node {
			_, advanced := f.sections(cx)
			return column(advanced...)
		})),
		page(settingsAbout, t.AboutSection(), lucide.Info, column(aboutHeader(cx), ui.FormSection{Children: s.appRows(cx)})),
		page(settingsSources, t.SourcesSection(), lucide.Library, ui.FormSection{Children: s.sourceRows(cx), Footer: t.SourcesFooter()}),
	}
	closeDialog := func() { s.settingsShown.Set(false) }
	return ui.SettingsDialog{Open: s.settingsShown.Get, OnOpenChange: func(open bool, _ base.Reason) { s.settingsShown.Set(open) },
		Label: t.Settings(), TestID: "settings-dialog",
		Content: func(narrow func() bool) core.Node {
			return ui.Settings{Categories: categories, Category: s.settingsTab, Narrow: narrow, Title: t.Settings(), HideSearch: true,
				OnClose: closeDialog, CloseLabel: t.Close(), TestID: "settings"}
		}}
}
