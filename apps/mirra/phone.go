package main

import (
	"fmt"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"github.com/yldm-ai/mirra/apps/mirra/locales"
)

// phone is the app in a narrow window: Studio, Wardrobe, Works and Me over a tab bar.
func (s *studio) phone(cx *core.Scope) core.Node {
	t := locales.For(cx)
	page := func(content func(cx *core.Scope) core.Node) func(cx *core.Scope) core.Node {
		return func(cx *core.Scope) core.Node {
			return core.View{Style: fill.Merge(core.Style{Padding: core.Symmetric(0, 20), PaddingEnv: core.EnvEdges{Top: core.EnvSafeArea}}), Children: []core.Node{
				core.View{Style: fill.Merge(core.Style{Padding: core.Edges{Top: 20}}), Children: []core.Node{content(cx)}},
			}}
		}
	}
	return ui.TabShell{Label: t.TabsLabel(), Selected: s.section.Get, OnChange: s.section.Set, Tabs: []ui.ShellTab{
		{Text: t.TabStudio(), Icon: lucide.House, SystemIcon: "house", Content: s.phoneStudio},
		{Text: t.TabWardrobe(), Icon: lucide.Shirt, SystemIcon: "tshirt", Content: page(func(cx *core.Scope) core.Node { return s.wardrobe(cx, 2) })},
		{Text: t.TabWorks(), Icon: lucide.Images, SystemIcon: "photo.on.rectangle", Content: page(func(cx *core.Scope) core.Node { return s.worksPage(cx, 2) })},
		{Text: t.TabMe(), Icon: lucide.User, SystemIcon: "person", Content: s.settingsPage},
	}}
}

func (s *studio) phoneStudio(cx *core.Scope) core.Node {
	return s.studioNav.Stack(fill, func(cx *core.Scope, r route, _ func() int) base.NavPage {
		switch r.page {
		case pageEditor:
			return s.editorPage(cx)
		case pageShot:
			return core.Screen{HeaderHidden: true, Style: fill, Children: []core.Node{s.shotPage(cx)}, OnDismiss: s.dropShot}
		}
		return s.homePage(cx)
	})
}

// brandPill is Mirra's name on a pill, over the home page's picture.
func brandPill(cx *core.Scope) core.Node {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	return core.View{StyleFunc: func() core.Style {
		return core.Style{Padding: core.Symmetric(8, 16), CornerRadius: 999, Background: colors.Text()}
	}, Children: []core.Node{core.Text{Content: t.Brand(), StyleFunc: func() core.Style {
		return core.Style{FontSize: 18, FontWeight: core.FontWeightBold, Color: colors.Background()}
	}}}}
}

func (s *studio) openEditor(k Kind) {
	s.step.Set(k)
	s.studioNav.SetRoutes(route{page: pageHome}, route{page: pageEditor})
}

// firstMissing is the first element still empty, or the pose; the optional background is never missing.
func (s *studio) firstMissing() Kind {
	for _, k := range lookKinds {
		if s.chosen[k].Get() == nil {
			return k
		}
	}
	return KindPose
}

// elementCard is one of the look's three elements on the home page: its picture, number and what was chosen.
func (s *studio) elementCard(cx *core.Scope, k Kind) core.Node {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	chosen := s.chosen[k]
	return core.View{Role: core.RoleButton, TestID: "row-" + kindKey(k),
		LabelFunc: func() string { return slotName(t, k) + ", " + chosenName(t, k, chosen.Get()) },
		OnPress:   func(core.PressEvent) { s.openEditor(k) },
		StyleFunc: func() core.Style {
			return core.Style{FlexGrow: 1, FlexBasis: core.Pt(0), CornerRadius: 18, Overflow: core.Hidden, Background: colors.Surface(), BorderWidth: 1, BorderColor: colors.Border()}
		},
		Children: []core.Node{
			core.View{StyleFunc: func() core.Style {
				return core.Style{Height: core.Pt(92), Background: colors.Inset(), AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Color: colors.Muted()}
			}, Children: []core.Node{core.Show{
				When: func() bool { return chosen.Get() != nil },
				Then: func(*core.Scope) core.Node {
					return core.Image{ResizeMode: cellFit(k), Style: core.Style{Width: core.Pct(100), Height: core.Pct(100)},
						SourceFunc: func() core.ImageSource {
							if it := chosen.Get(); it != nil {
								return s.src(it.Thumb)
							}
							return core.ImageSource{}
						}}
				},
				Else: func(*core.Scope) core.Node { return core.Icon{SVG: lucide.Plus(), Style: core.Style{FontSize: 20}} },
			}}},
			core.View{Style: core.Style{Padding: core.Symmetric(8, 10), Gap: 3}, Children: []core.Node{
				core.Text{Content: slotEyebrow(t, k), MaxLines: 1, StyleFunc: func() core.Style {
					return core.Style{FontSize: 9, LetterSpacing: 1, Color: colors.Muted()}
				}},
				core.Text{ContentFunc: func() string {
					if it := chosen.Get(); it != nil {
						return itemName(t, it)
					}
					return slotName(t, k)
				}, MaxLines: 1, Style: core.Style{FontSize: 14, FontWeight: core.FontWeightSemibold}},
			}},
		},
	}
}

// homePage is today's look: its picture large, named by its outfit, its three elements, and Generate.
func (s *studio) homePage(cx *core.Scope) base.NavPage {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	look := func() Request { return s.look() }
	hero := core.View{Role: core.RoleButton, Label: stepTitle(t, KindFace), OnPress: func(core.PressEvent) { s.openEditor(s.firstMissing()) },
		StyleFunc: func() core.Style { return fill.Merge(core.Style{Background: colors.Inset(), MinHeight: core.Pt(240)}) },
		Children: []core.Node{
			s.heroImage(cx),
			core.View{StyleFunc: func() core.Style {
				return core.Style{Position: core.Absolute, Left: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0), Height: core.Pct(55),
					Gradients: []core.Gradient{core.LinearGradient(180, colors.Background().Faded(0), colors.Background().Faded(0.85), colors.Background())}}
			}},
			core.View{Style: core.Style{Position: core.Absolute, Left: core.Pt(20), Right: core.Pt(20), Bottom: core.Pt(12), Gap: 6}, Children: []core.Node{
				eyebrow(cx, func() string {
					if c := s.chosen[KindCostume].Get(); c != nil && c.Group != uploadGroup {
						return t.EyebrowTodayGroup(groupLabel(t, c.Group))
					}
					return t.EyebrowToday()
				}),
				display(func() string { return lookTitle(t, look()) }, 30),
				caption(cx, func() string { return lookLine(t, look()) }),
			}},
		},
	}
	top := core.View{Style: core.Style{Position: core.Absolute, Top: core.Pt(0), Left: core.Pt(16), Right: core.Pt(16), Direction: core.Row,
		JustifyContent: core.JustifySpaceBetween, AlignItems: core.AlignCenter, PaddingEnv: core.EnvEdges{Top: core.EnvSafeArea}, Padding: core.Edges{Top: 12}},
		Children: []core.Node{brandPill(cx), core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8}, Children: []core.Node{s.surpriseButton(cx), s.statusBadge(cx)}}}}
	var cards []core.Node
	for _, k := range steps {
		cards = append(cards, s.elementCard(cx, k))
	}
	return core.Screen{HeaderHidden: true, Title: t.TabStudio(), Style: fill, Children: []core.Node{
		core.View{Style: fill, Children: []core.Node{hero, top}},
		core.View{Style: core.Style{Padding: core.Symmetric(12, 16), Gap: 12}, Children: []core.Node{
			core.View{Style: core.Style{Direction: core.Row, Gap: 10}, Children: cards},
			s.styleChips(cx),
			s.needHint(cx),
			s.generateButton(cx),
		}},
	}}
}

// editorPage puts the look together a gallery at a time: face, outfit, then pose.
func (s *studio) editorPage(cx *core.Scope) base.NavPage {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	var options []ui.Option[Kind]
	for i, k := range steps {
		n := i + 1
		options = append(options, ui.Option[Kind]{Value: k, TestID: "step-" + kindKey(k),
			Content: core.Text{ContentFunc: func() string {
				label := t.StepTab(fmt.Sprint(n), slotName(t, k))
				if s.chosen[k].Get() != nil && s.step.Get() != k {
					label += " ✓"
				}
				return label
			}}, Label: t.StepTab(fmt.Sprint(n), slotName(t, k))})
	}
	bottom := core.Dynamic(func(cx *core.Scope) core.Node {
		next := func(text string, k Kind) core.Node {
			return ui.Button{Text: text, Size: ui.SizeLarge, TestID: "next", Variant: ui.ButtonSecondary,
				Style: core.Style{Width: core.Pct(100), Height: core.Pt(52), CornerRadius: 999}, OnPress: func() { s.step.Set(k) }}
		}
		switch s.step.Get() {
		case KindFace:
			return next(t.NextCostume(), KindCostume)
		case KindCostume:
			return next(t.NextPose(), KindPose)
		}
		return core.View{Style: core.Style{Gap: 8}, Children: []core.Node{s.styleChips(cx), s.needHint(cx), s.generateButton(cx)}}
	})
	return ui.NavPage(cx, core.Screen{TitleFunc: func() string { return stepTitle(t, s.step.Get()) }, Style: fill, Children: []core.Node{
		core.View{Style: fill.Merge(core.Style{Padding: core.Symmetric(8, 16), Gap: 12}), Children: []core.Node{
			ui.Tabs[Kind]{Label: t.StepsLabel(), Options: options, Value: s.step.Get, OnChange: s.step.Set, Equal: true, TestID: "steps"},
			core.Dynamic(func(cx *core.Scope) core.Node { return s.chosenCard(cx, s.step.Get()) }),
			core.Dynamic(func(cx *core.Scope) core.Node {
				k := s.step.Get()
				columns := 3
				if k == KindFace {
					columns = 4
				}
				return s.library(cx, k, columns)
			}),
		}},
		core.View{StyleFunc: func() core.Style {
			return core.Style{Padding: core.Symmetric(12, 16), Borders: core.Edges{Top: 1}, BorderColor: colors.Border()}
		}, Children: []core.Node{bottom}},
	}}, ui.NavHeader{OnBack: func() { s.studioNav.Pop(1) }, BackLabel: t.Back(),
		Trailing: []core.Node{ui.Button{Text: t.Done(), Variant: ui.ButtonGhost, TestID: "done", OnPress: func() { s.studioNav.Pop(1) }}}})
}

// chosenCard shows what a step has chosen so far, above its gallery.
func (s *studio) chosenCard(cx *core.Scope, k Kind) core.Node {
	t := locales.For(cx)
	colors := ui.ColorsOf(cx)
	chosen := s.chosen[k]
	top := func() string {
		if it := chosen.Get(); it != nil && k != KindFace {
			return t.SelectedItem(it.ID)
		}
		return slotEyebrow(t, k)
	}
	return core.View{TestID: "chosen",
		StyleFunc: func() core.Style {
			return core.Style{Direction: core.Row, Gap: 16, Padding: core.All(12), CornerRadius: 20, Background: colors.Surface(), BorderWidth: 1, BorderColor: colors.Border(), AlignItems: core.AlignCenter}
		},
		Children: []core.Node{
			s.thumb(cx, k, chosen.Get, 72, 96, false),
			core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 6}, Children: []core.Node{
				eyebrow(cx, top),
				display(func() string { return chosenName(t, k, chosen.Get()) }, 20),
				caption(cx, func() string { return itemSub(t, chosen.Get()) }),
			}},
		},
	}
}
