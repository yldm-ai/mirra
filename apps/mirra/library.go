package main

import (
	"slices"
	"strings"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"github.com/yldm-ai/mirra/apps/mirra/locales"
)

// displayFont sets titles in a serif, the platform's own where it has one for Chinese.
const displayFont = `"Songti SC", "Noto Serif CJK SC", "Source Han Serif SC", SimSun, Georgia, serif`

// loaded is the catalog res holds, empty until its first load.
func loaded(res *core.Resource[Catalog]) Catalog {
	c, _ := res.Get()
	return c
}

// eyebrow is the small spaced-out label above a title, such as THIS LOOK.
func eyebrow(cx *core.Scope, text func() string) core.Node {
	colors := ui.ColorsOf(cx)
	return core.Text{ContentFunc: text, StyleFunc: func() core.Style {
		return core.Style{FontSize: 11, FontWeight: core.FontWeightSemibold, LetterSpacing: 2, TextTransform: core.TextTransformUppercase, Color: colors.Accent()}
	}}
}

// display is a title set in the serif.
func display(text func() string, size core.Points) core.Node {
	return core.Text{ContentFunc: text, Role: core.RoleHeader, MaxLines: 2,
		Style: core.Style{FontFamily: displayFont, FontSize: size, FontWeight: core.FontWeightBold, LineHeight: size * 1.25}}
}

func caption(cx *core.Scope, text func() string) core.Node {
	colors := ui.ColorsOf(cx)
	return core.Text{ContentFunc: text, MaxLines: 1, StyleFunc: func() core.Style { return core.Style{FontSize: 13, Color: colors.Muted()} }}
}

func slotName(t locales.Messages, k Kind) string {
	switch k {
	case KindFace:
		return t.SlotFace()
	case KindCostume:
		return t.SlotCostume()
	case KindBackground:
		return t.SlotBackground()
	default:
		return t.SlotPose()
	}
}

// slotEyebrow numbers a gallery as the look's element, such as 01 FACE.
func slotEyebrow(t locales.Messages, k Kind) string {
	switch k {
	case KindFace:
		return t.EyebrowFace()
	case KindCostume:
		return t.EyebrowOutfit()
	case KindBackground:
		return t.EyebrowBackground()
	default:
		return t.EyebrowPose()
	}
}

func stepTitle(t locales.Messages, k Kind) string {
	switch k {
	case KindFace:
		return t.StepFace()
	case KindCostume:
		return t.StepCostume()
	case KindBackground:
		return t.StepBackground()
	default:
		return t.StepPose()
	}
}

func kindKey(k Kind) string {
	switch k {
	case KindFace:
		return "face"
	case KindCostume:
		return "costume"
	case KindBackground:
		return "background"
	default:
		return "pose"
	}
}

// groupLabel names a chip: a gallery's angle, category or hair length in the user's language.
func groupLabel(t locales.Messages, g string) string {
	switch g {
	case "正面":
		return t.AngleFront()
	case "斜め":
		return t.AngleThreeQuarter()
	case "横":
		return t.AngleSide()
	case "上から":
		return t.AngleHigh()
	case "下から":
		return t.AngleLow()
	case "ショート":
		return t.HairShort()
	case "ミディアム":
		return t.HairMedium()
	case "セミロング":
		return t.HairSemiLong()
	case "ロング":
		return t.HairLong()
	case "春":
		return t.GroupSpring()
	case "夏":
		return t.GroupSummer()
	case "秋":
		return t.GroupAutumn()
	case "冬":
		return t.GroupWinter()
	case "パーティ":
		return t.GroupParty()
	case "水着":
		return t.GroupSwimwear()
	case "ランジェリー":
		return t.GroupLingerie()
	case "セクシー":
		return t.GroupSexy()
	case groupIndoor:
		return t.GroupIndoor()
	case groupCity:
		return t.GroupCity()
	case groupNature:
		return t.GroupNature()
	case uploadGroup:
		return t.MyUploads()
	}
	return g
}

// itemName is what a chosen item is called: an outfit's keywords or the user's upload, a pose's name, a face's hair length, a background's group.
func itemName(t locales.Messages, it *Item) string {
	if it == nil {
		return t.NotChosen()
	}
	switch {
	case it.Kind == KindFace || it.Kind == KindBackground:
		return groupLabel(t, it.Group)
	case it.Group == uploadGroup:
		return t.UploadName(strings.TrimPrefix(it.ID, uploadPrefix))
	}
	return it.Title
}

// chosenName is what a look's element is called: its item's name, or for the optional background "No background" while none is chosen.
func chosenName(t locales.Messages, k Kind, it *Item) string {
	if it == nil && k == KindBackground {
		return t.NoBackground()
	}
	return itemName(t, it)
}

// itemSub is the line under an item's name: an outfit's category and code, a pose's angle, a face's code.
func itemSub(t locales.Messages, it *Item) string {
	if it == nil {
		return ""
	}
	switch it.Kind {
	case KindFace, KindBackground:
		return it.ID
	case KindPose:
		return groupLabel(t, it.Group)
	}
	if it.Group == uploadGroup {
		return t.MyUploads()
	}
	return groupLabel(t, it.Group) + " · " + it.ID
}

// lookLine sums up the look in a line: face, pose and angle.
func lookLine(t locales.Messages, r Request) string {
	var parts []string
	if r.Face != nil {
		parts = append(parts, itemName(t, r.Face))
	}
	if r.Pose != nil {
		angle := groupLabel(t, r.Pose.Group)
		if r.sheet() {
			angle = t.AllAngles()
		}
		parts = append(parts, r.Pose.Title, angle)
	}
	return strings.Join(parts, " · ")
}

// lookTitle names the look by its outfit, else by its pose.
func lookTitle(t locales.Messages, r Request) string {
	switch {
	case r.Costume != nil:
		return itemName(t, r.Costume)
	case r.Pose != nil:
		return r.Pose.Title
	}
	return t.LookTitle()
}

func cellAspect(k Kind) float32 {
	switch k {
	case KindFace:
		return 0.8
	case KindBackground:
		return 1
	}
	return 0.75
}

// cellFit fills a face's or a background's cell, a photograph cropped, and fits a pose's or an outfit's whole, a figure on white.
func cellFit(k Kind) core.ResizeMode {
	if k == KindFace || k == KindBackground {
		return core.Cover
	}
	return core.Contain
}

type chipOption struct{ value, label string }

// pill is one pill-shaped filter, filled while on, or outlined in the accent while on for a switch, which is on beside the chosen filter.
func pill(cx *core.Scope, label, testID string, on func() bool, onPress func(), toggle bool) core.Node {
	colors := ui.ColorsOf(cx)
	return core.View{Role: core.RoleButton, Label: label, Selected: on, TestID: testID,
		OnPress: func(core.PressEvent) { onPress() },
		StyleFunc: func() core.Style {
			st := core.Style{Padding: core.Symmetric(7, 16), CornerRadius: 999, BorderWidth: 1, BorderColor: colors.Border(), Color: colors.Text()}
			switch {
			case on() && toggle:
				st.Background, st.BorderColor, st.Color = colors.Accent().Faded(0.12), colors.Accent(), colors.Accent()
			case on():
				st.Background, st.BorderColor, st.Color = colors.Text(), colors.Text(), colors.Background()
			}
			return st
		},
		Children: []core.Node{core.Text{Content: label, Style: core.Style{FontSize: 14}}},
	}
}

// chips is a row of pill-shaped filters that scrolls sideways when it does not fit, with extra pills after them.
func chips(cx *core.Scope, label, testID string, options []chipOption, value func() string, onChange func(string), extra ...core.Node) core.Node {
	var pills []core.Node
	for _, o := range options {
		pills = append(pills, pill(cx, o.label, testID+"-"+o.label, func() bool { return value() == o.value }, func() { onChange(o.value) }, false))
	}
	pills = append(pills, extra...)
	// A scroll container's minimum height is 0, so a long grid beside it would squeeze it away without one.
	return core.Scroll{Horizontal: true, HideIndicator: true, Style: core.Style{Height: core.Pt(52), MinHeight: core.Pt(52), FlexShrink: 0}, Children: []core.Node{
		core.View{Role: core.RoleGroup, Label: label, Style: core.Style{Direction: core.Row, Gap: 8, Padding: core.Symmetric(8, 0), AlignItems: core.AlignCenter}, Children: pills},
	}}
}

// itemCard is one picture to choose in a grid: the image, then its name and a muted line, outlined in the accent while chosen. An outfit carries a heart to keep it in the wardrobe's favorites.
func (s *studio) itemCard(cx *core.Scope, k Kind, item func() Item, on func() bool, sub func() string, onPress func()) core.Node {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	name := func() string {
		it := item()
		return itemName(t, &it)
	}
	picture := core.View{StyleFunc: func() core.Style {
		return core.Style{Width: core.Pct(100), AspectRatio: cellAspect(k), Background: colors.Inset()}
	}, Children: []core.Node{core.Image{ResizeMode: cellFit(k), Style: core.Style{Width: core.Pct(100), Height: core.Pct(100)},
		SourceFunc: func() core.ImageSource { return s.src(item().Thumb) }}}}
	children := []core.Node{picture}
	if k != KindFace && k != KindBackground {
		children = append(children, core.View{Style: core.Style{Padding: core.Symmetric(8, 10), Gap: 2}, Children: []core.Node{
			core.Text{ContentFunc: name, MaxLines: 1, Style: core.Style{FontSize: 14, FontWeight: core.FontWeightMedium}},
			caption(cx, sub),
		}})
	}
	card := core.View{Role: core.RoleButton, LabelFunc: func() string {
		it := item()
		if it.Kind == KindFace || it.Kind == KindBackground {
			return name() + " · " + it.ID
		}
		return name()
	}, Selected: on,
		OnPress: func(core.PressEvent) { onPress() },
		StyleFunc: func() core.Style {
			st := core.Style{CornerRadius: 16, Overflow: core.Hidden, BorderWidth: 1, BorderColor: colors.Border(), Background: colors.Surface()}
			if on() {
				st.BorderColor, st.BorderWidth = colors.Accent(), 2
			}
			return st
		},
		Children: children,
	}
	if k != KindCostume {
		return card
	}
	fav := func() bool { return s.isFavorite(item()) }
	return core.View{Children: []core.Node{
		card,
		core.View{Style: core.Style{Position: core.Absolute, Top: core.Pt(8), Right: core.Pt(8)}, Children: []core.Node{
			ui.IconButton{Icon: lucide.Heart, Variant: ui.IconButtonRaised, Size: 30, IconSize: 15, Selected: fav,
				LabelFunc: func() string {
					if fav() {
						return t.Unfavorite()
					}
					return t.Favorite()
				},
				Tint: func() core.Color {
					if fav() {
						return colors.Danger()
					}
					return colors.Text()
				},
				OnPress: func() { s.toggleFavoriteOutfit(item()) }},
		}},
	}}
}

// library is one gallery to choose from: its chips and a grid of cards, columns wide.
func (s *studio) library(cx *core.Scope, k Kind, columns int) core.Node {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	res := s.catalogs[k]
	filter := s.filter[k]
	chosen := s.chosen[k]
	adult := s.adult[k]
	items := core.NewMemoFunc(cx, func() []Item { return loaded(res).InGroup(filter.Get(), adult.Get()) }, nil)
	at := func(i int) (Item, bool) {
		list := items.Get()
		if i < 0 || i >= len(list) {
			return Item{}, false
		}
		return list[i], true
	}
	filters := core.Dynamic(func(cx *core.Scope) core.Node {
		cat := loaded(res)
		groups := cat.Groups
		if adult.Get() {
			groups = cat.AdultGroups
		}
		if len(groups) < 2 && len(cat.AdultGroups) == 0 {
			return nil
		}
		var options []chipOption
		// Poses always show one angle: every angle of every pose at once is the same pose five times.
		if k != KindPose {
			options = append(options, chipOption{"", t.All()})
		}
		for _, g := range groups {
			options = append(options, chipOption{g, groupLabel(t, g)})
		}
		var extra []core.Node
		if k == KindPose {
			extra = append(extra, core.Show{When: func() bool { return s.angles(chosen.Get()) != nil }, Then: func(cx *core.Scope) core.Node {
				return pill(cx, t.AllAngles(), "chip-sheet", s.sheet.Get, func() { s.sheet.Set(!s.sheet.Get()) }, true)
			}})
		}
		if len(cat.AdultGroups) > 0 {
			extra = append(extra, pill(cx, t.AdultChip(), "chip-adult", adult.Get, func() { s.toggleAdult(k) }, true))
		}
		return chips(cx, t.FiltersLabel(), "chip", options, filter.Get, filter.Set, extra...)
	})
	grid := core.List{Style: fill, NumColumns: columns, Gap: 10, ColumnGap: 10, EstimatedSize: 200,
		Count: func() int { return len(items.Get()) },
		Key: func(i int) any {
			it, _ := at(i)
			return it.ID
		},
		Row: func(cx *core.Scope, row core.ListRow) core.Node {
			item := func() Item {
				it, _ := at(row.Index())
				return it
			}
			on := func() bool {
				c := chosen.Get()
				return c != nil && c.ID == item().ID
			}
			sub := func() string {
				it := item()
				if k == KindCostume {
					return it.ID
				}
				return groupLabel(t, it.Group)
			}
			return s.itemCard(cx, k, item, on, sub, func() {
				it, ok := at(row.Index())
				switch {
				case !ok:
				// The background is optional: pressing the chosen one again goes back to none.
				case k == KindBackground && on():
					s.clearBackground()
				default:
					s.choose(k, it)
				}
			})
		},
	}
	state := core.Dynamic(func(cx *core.Scope) core.Node {
		if err := res.Err(); err != nil && len(loaded(res).Items) == 0 {
			return core.View{Style: fill.Merge(core.Style{AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 12, Padding: core.All(24)}), Children: []core.Node{
				core.Text{Content: t.LoadFailed(), Style: ui.TypeHeading},
				core.Text{Content: explain(t, err), StyleFunc: func() core.Style {
					return ui.TypeCaption.Merge(core.Style{Color: colors.Muted(), TextAlign: core.TextAlignCenter})
				}},
				ui.Button{Text: t.Retry(), Variant: ui.ButtonSecondary, OnPress: res.Refetch, TestID: "retry"},
			}}
		}
		if len(loaded(res).Items) == 0 {
			return core.View{Style: fill.Merge(core.Style{AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter}), Children: []core.Node{
				ui.Spinner{Label: stepTitle(t, k)},
			}}
		}
		return core.View{Style: fill, Children: []core.Node{filters, grid}}
	})
	return core.View{Style: fill, TestID: "library", Children: []core.Node{state}}
}

// thumb is a chosen item's small picture, or an empty well.
func (s *studio) thumb(cx *core.Scope, k Kind, it func() *Item, w, h core.Points, round bool) core.Node {
	colors := ui.ColorsOf(cx)
	radius := core.Points(12)
	if round {
		radius = w / 2
	}
	return core.View{
		StyleFunc: func() core.Style {
			return core.Style{Width: core.Pt(w), Height: core.Pt(h), CornerRadius: radius, Overflow: core.Hidden, Background: colors.Inset(),
				AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Color: colors.Muted(), FlexShrink: 0}
		},
		Children: []core.Node{core.Show{
			When: func() bool { return it() != nil },
			Then: func(*core.Scope) core.Node {
				return core.Image{ResizeMode: cellFit(k), Style: core.Style{Width: core.Pct(100), Height: core.Pct(100)},
					SourceFunc: func() core.ImageSource {
						if v := it(); v != nil {
							return s.src(v.Thumb)
						}
						return core.ImageSource{}
					}}
			},
			Else: func(*core.Scope) core.Node { return core.Icon{SVG: lucide.Plus(), Style: core.Style{FontSize: 18}} },
		}},
	}
}

// lookRow is one element of the look: its picture, its number and gallery, and what was chosen, outlined while its gallery is open on the desktop.
func (s *studio) lookRow(cx *core.Scope, k Kind, onPress func()) core.Node {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	chosen := s.chosen[k]
	value := func() string {
		if it := chosen.Get(); it != nil || k == KindBackground {
			return chosenName(t, k, it)
		}
		return slotName(t, k)
	}
	active := func() bool { return s.wide() && s.step.Get() == k }
	return core.View{Role: core.RoleButton, TestID: "row-" + kindKey(k), Selected: active,
		LabelFunc: func() string { return slotName(t, k) + ", " + chosenName(t, k, chosen.Get()) },
		OnPress:   func(core.PressEvent) { onPress() },
		StyleFunc: func() core.Style {
			st := core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 14, Padding: core.Symmetric(10, 12), CornerRadius: 16,
				Background: colors.Background(), BorderWidth: 1, BorderColor: colors.Border()}
			if active() {
				st.BorderColor, st.Background = colors.Accent(), colors.Inset()
			}
			return st
		},
		Children: []core.Node{
			s.thumb(cx, k, chosen.Get, 48, 48, k == KindFace),
			core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 4}, Children: []core.Node{
				core.Text{Content: slotEyebrow(t, k), StyleFunc: func() core.Style {
					return core.Style{FontSize: 10, LetterSpacing: 1.5, Color: colors.Muted()}
				}},
				core.Text{ContentFunc: value, MaxLines: 1, Style: core.Style{FontSize: 15, FontWeight: core.FontWeightSemibold}},
			}},
			core.Show{When: func() bool { return k == KindBackground && chosen.Get() != nil }, Then: func(cx *core.Scope) core.Node {
				return ui.IconButton{Icon: lucide.X, Label: t.RemoveBackground(), TestID: "clear-background", OnPress: s.clearBackground}
			}},
		},
	}
}

// heroItem is the picture the look leads with: the outfit, else the pose, else the face.
func (s *studio) heroItem() *Item {
	for _, k := range []Kind{KindCostume, KindPose, KindFace} {
		if it := s.chosen[k].Get(); it != nil {
			return it
		}
	}
	return nil
}

func (s *studio) heroImage(cx *core.Scope) core.Node {
	colors := ui.ColorsOf(cx)
	return core.Show{
		When: func() bool { return s.heroItem() != nil },
		Then: func(*core.Scope) core.Node {
			return core.Image{ResizeMode: core.Contain, TestID: "hero", Style: core.Style{Width: core.Pct(100), Height: core.Pct(100)},
				SourceFunc: func() core.ImageSource {
					if it := s.heroItem(); it != nil {
						return s.src(it.Thumb)
					}
					return core.ImageSource{}
				}}
		},
		Else: func(*core.Scope) core.Node {
			return core.View{StyleFunc: func() core.Style {
				return core.Style{FlexGrow: 1, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Color: colors.Muted()}
			}, Children: []core.Node{core.Icon{SVG: lucide.Sparkles(), Style: core.Style{FontSize: 40}}}}
		},
	}
}

// serviceName names the chosen AI service.
func (s *studio) serviceName(t locales.Messages) string {
	p, _ := s.current()
	return p.name(t)
}

// statusText says whether the AI service is set up and answered its test.
func (s *studio) statusText(t locales.Messages) string {
	if !s.configured() {
		return t.StatusUnconfigured()
	}
	name := s.serviceName(t)
	switch s.conn.Get() {
	case connOK:
		return t.StatusConnected(name)
	case connFailed:
		return t.StatusFailed(name)
	case connTesting:
		return t.StatusTesting(name)
	}
	return t.StatusUntested(name)
}

// statusBadge is the service's status as a pill with a colored dot, opening its settings.
func (s *studio) statusBadge(cx *core.Scope) core.Node {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	return core.View{Role: core.RoleButton, TestID: "status", LabelFunc: func() string { return s.statusText(t) },
		OnPress: func(core.PressEvent) { s.openSettings() },
		StyleFunc: func() core.Style {
			return core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8, Padding: core.Symmetric(8, 14), CornerRadius: 999,
				BorderWidth: 1, BorderColor: colors.Border(), Background: colors.Surface().Faded(0.85)}
		},
		Children: []core.Node{
			core.View{StyleFunc: func() core.Style {
				dot := colors.Accent()
				switch {
				case !s.configured():
				case s.conn.Get() == connOK:
					dot = ui.ColorsOf(cx).Up()
				case s.conn.Get() == connFailed:
					dot = colors.Danger()
				default:
					dot = colors.Muted()
				}
				return core.Style{Width: core.Pt(7), Height: core.Pt(7), CornerRadius: 4, Background: dot}
			}},
			core.Text{ContentFunc: func() string { return s.statusText(t) }, Style: core.Style{FontSize: 13}},
		},
	}
}

// surpriseButton puts a look together at random.
func (s *studio) surpriseButton(cx *core.Scope) core.Node {
	return ui.IconButton{Icon: lucide.Dices, Label: locales.For(cx).Surprise(), Variant: ui.IconButtonRaised, TestID: "surprise", OnPress: s.surprise}
}

// generateButton starts the look, or opens the service's settings while none is set up.
func (s *studio) generateButton(cx *core.Scope) core.Node {
	t := locales.For(cx)
	return core.Dynamic(func(cx *core.Scope) core.Node {
		text := t.Generate()
		if !s.configured() {
			text = t.GenerateNeedsService()
		}
		return ui.Button{Text: text, Icon: lucide.Sparkles, Size: ui.SizeLarge, TestID: "generate",
			Style:    core.Style{Width: core.Pct(100), Height: core.Pt(56), CornerRadius: 999},
			Disabled: func() bool { return s.configured() && s.look().Validate() != nil },
			Busy:     s.busy.Get, OnPress: s.generate}
	})
}

// needHint says why Generate is off: no pose yet.
func (s *studio) needHint(cx *core.Scope) core.Node {
	t := locales.For(cx)
	return core.Show{When: func() bool { return s.configured() && s.look().Validate() != nil }, Then: func(cx *core.Scope) core.Node {
		colors := ui.ColorsOf(cx)
		return core.Text{Content: t.NeedPose(), TestID: "hint", StyleFunc: func() core.Style {
			return ui.TypeCaption.Merge(core.Style{Color: colors.Muted(), TextAlign: core.TextAlignCenter})
		}}
	}}
}

// styleLabel is a style's name on its chip.
func styleLabel(t locales.Messages, st Style) string {
	switch st {
	case StyleStreet:
		return t.StyleStreet()
	case StyleWhite:
		return t.StyleWhite()
	case StyleFilm:
		return t.StyleFilm()
	case StyleAnime:
		return t.StyleAnime()
	}
	return t.StyleStudio()
}

// styleChips chooses the look's style.
func (s *studio) styleChips(cx *core.Scope) core.Node {
	t := locales.For(cx)
	var options []chipOption
	for _, st := range Styles {
		options = append(options, chipOption{value: string(st), label: styleLabel(t, st)})
	}
	return chips(cx, t.StyleLabel(), "style", options, func() string { return string(s.style.Get()) }, func(v string) { s.setStyle(Style(v)) })
}

func (s *studio) noteInput(cx *core.Scope) core.Node {
	t := locales.For(cx)
	return ui.Input{Value: s.note.Get, OnChange: s.note.Set, Multiline: true, Label: t.NoteLabel(), Placeholder: t.NotePlaceholder(), TestID: "note"}
}

// generatingScreen covers the app while a look is made, with a way to stop it.
func (s *studio) generatingScreen(cx *core.Scope) core.Node {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	return core.View{TestID: "generating", Role: core.RoleAlert, Label: t.Generating(),
		StyleFunc: func() core.Style {
			return core.Style{Position: core.Absolute, Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0), Left: core.Pt(0),
				Background: colors.Background().Faded(0.96), Color: colors.Text(), AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 28}
		},
		Children: []core.Node{
			core.View{StyleFunc: func() core.Style {
				return core.Style{Width: core.Pt(200), Height: core.Pt(268), CornerRadius: 24, Background: colors.Surface(), BorderWidth: 1, BorderColor: colors.Border(),
					AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter}
			}, Children: []core.Node{ui.Spinner{Size: 44, Label: t.Generating()}}},
			eyebrow(cx, t.Generating),
			ui.Button{Text: t.Cancel(), Variant: ui.ButtonOutline, OnPress: s.cancelGenerating, TestID: "cancel", Style: core.Style{CornerRadius: 999}},
		},
	}
}

// toggleAdult switches a gallery between its items for everyone and its adults-only ones, asking first whether the user is 18 or older.
func (s *studio) toggleAdult(k Kind) {
	show := func(adult bool) {
		cat := loaded(s.catalogs[k])
		groups := cat.Groups
		if adult {
			groups = cat.AdultGroups
		}
		s.adult[k].Set(adult)
		// Poses keep their angle where the other side has it; the rest start on All.
		if g := s.filter[k].Get(); g != "" && !slices.Contains(groups, g) {
			g = ""
			if k == KindPose && len(groups) > 0 {
				g = groups[0]
			}
			s.filter[k].Set(g)
		}
	}
	if s.adult[k].Get() {
		show(false)
		return
	}
	if s.adultOK.Get() {
		show(true)
		return
	}
	s.confirmAdult(func() { show(true) })
}

// confirmAdult asks the user whether they are 18 or older and, if they say so, keeps the answer and calls then.
func (s *studio) confirmAdult(then func()) {
	t := locales.For(s.cx)
	base.ShowAlertDialog(s.cx, base.AlertDialog{Title: t.AdultConfirm(), Description: t.AdultConfirmBody(), Actions: []core.AlertAction{
		{Label: t.AdultNo(), Role: core.AlertCancel}, {Label: t.AdultYes()},
	}}, func(label string) {
		if label != t.AdultYes() {
			return
		}
		s.adultOK.Set(true)
		if err := s.cx.Storage().SetItem(keyAdult, "1"); err != nil {
			s.toastError(t.SaveFailed(), err)
		}
		if then != nil {
			then()
		}
	})
}

// setAdultOK is the settings' 18+ switch: on asks the age question, off forgets the answer and hides any 18+ category on show, so the galleries ask again.
func (s *studio) setAdultOK(on bool) {
	if on {
		s.confirmAdult(nil)
		return
	}
	s.adultOK.Set(false)
	s.cx.Storage().RemoveItem(keyAdult)
	for k := range s.adult {
		if s.adult[k].Get() {
			s.toggleAdult(Kind(k))
		}
	}
}
