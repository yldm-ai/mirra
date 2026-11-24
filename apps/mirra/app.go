// Command mirra is Mirra (镜衣): it puts a face, an outfit and a pose from three public reference galleries together and has an OpenAI-compatible image model, in the cloud or on the user's own machine, draw the result.
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/i18n"
	"github.com/nsgo-dev/nsgo/ui"
	"github.com/yldm-ai/mirra/apps/mirra/locales"
)

func main() {
	app := App{Client: &http.Client{Timeout: 5 * time.Minute}, Sites: DefaultSites}
	nsgo.Run(app.Root, nsgo.WindowSize(1280, 860))
}

// appIcon is the app's icon, which the settings show beside its name.
//
//go:embed assets/Icon.png
var appIcon []byte

//go:embed theme.yaml
var themeYAML []byte

// repoURL is where the app's source lives.
const repoURL = "https://github.com/yldm-ai/mirra"

// App is the app with what it reaches outside: tests give it local servers and a temporary directory.
type App struct {
	Client *http.Client
	Sites  Sites
	// DataDir keeps works and uploads; the app's data directory when empty.
	DataDir string
	// Pictures keeps them; files, or memory in a browser, when nil.
	Pictures pictures
}

// wideWidth is the window width, in points, from which the app shows the desktop studio instead of the phone's tabs.
const wideWidth = 900

// Sections of the app, in the tab bar's order. The desktop shows settings in a dialog rather than as a section.
const (
	sectionStudio = iota
	sectionWardrobe
	sectionWorks
	sectionSettings
)

// steps is the order the look is put together in; the background is optional and comes last.
var steps = []Kind{KindFace, KindCostume, KindPose, KindBackground}

// lookKinds are the elements a look is made of, without the optional background: what Surprise me picks and the editor opens on while one is empty.
var lookKinds = []Kind{KindFace, KindCostume, KindPose}

type page int

const (
	pageHome page = iota
	pageEditor
	pageShot
)

type route struct{ page page }

// Storage keys and secret accounts.
const (
	keyFavorites      = "favoriteOutfits"
	resultsFolderName = "results"
)

// shot is a generated picture on show: just made, with the look that made it, or a saved work opened again.
type shot struct {
	Path    string
	Req     Request
	Attempt int
	Saved   *work
}

// studio is the app's state, shared by the phone and desktop layouts.
type studio struct {
	App
	pics    pictures
	cx      *core.Scope
	toaster *ui.Toaster
	section *core.Signal[int]
	// step is the gallery being chosen from: the phone editor's step and the desktop library's tab.
	step *core.Signal[Kind]
	// chosen holds the look's face, outfit, pose and background, by Kind.
	chosen [kindCount]*core.Signal[*Item]
	// filter is each gallery's chip, "" for all.
	filter [kindCount]*core.Signal[string]
	// adult says whether each gallery shows its adults-only items instead of those for everyone.
	adult [kindCount]*core.Signal[bool]
	// adultOK is the user's answer that they are 18 or older, asked once and kept, since works and favorites from the adults-only categories show without asking.
	adultOK *core.Signal[bool]
	note    *core.Signal[string]
	// sheet asks for every camera angle of the pose side by side, rather than the chosen one.
	sheet *core.Signal[bool]
	// style is the look's style, kept for the next launch.
	style *core.Signal[Style]

	busy   *core.Signal[bool]
	cancel context.CancelFunc
	shot   *core.Signal[*shot]
	zoom   *core.Signal[int]

	// provider is the chosen AI service's ID, and settings what the user set for each one.
	provider               *core.Signal[string]
	settings               map[string]providerSettings
	conn                   *core.Signal[connState]
	connErr                *core.Signal[string]
	showKey, settingsShown *core.Signal[bool]
	// settingsTab is the desktop settings' tab on show.
	settingsTab *core.Signal[string]
	// keyKept says whether the key outlives the page: false in a browser, which has no secure store.
	keyKept *core.Signal[bool]

	works       *core.Signal[[]work]
	favoritesOn *core.Signal[bool]
	favorites   *core.Signal[[]Item]
	uploads     *core.Signal[[]Item]
	closet      *core.Signal[int]
	catalogs    [kindCount]*core.Resource[Catalog]

	studioNav, worksNav *base.Nav[route]
}

var fill = core.Style{FlexGrow: 1, FlexShrink: 1, FlexBasis: core.Pt(0)}

func itemsEqual(a, b []Item) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Root is the app: the desktop studio in a wide window, the phone's tabs otherwise, with the generating screen and toasts over either.
func (a App) Root(cx *core.Scope) core.Node {
	theme, err := core.ParseTheme(themeYAML)
	if err != nil {
		panic(err) // the embedded theme is part of the build
	}
	cx.SetTheme(theme)
	cx.SetAppearance(core.AppearanceDark)
	rememberLanguage(cx)
	s := newStudio(cx, a)
	colors := ui.ColorsOf(cx)
	env := cx.Env()
	// Only macOS: Windows and Linux would draw a menu bar into the window, which the app's own top bar already serves.
	if env.OS() == core.OSMacOS {
		cx.Effect(func() { desktop.SetApplicationMenu(cx, s.menu(cx, locales.For(cx))) })
	}
	return core.View{
		StyleFunc: func() core.Style {
			return core.Style{FlexGrow: 1, FlexShrink: 1, Background: colors.Background(), Color: colors.Text()}
		},
		Children: []core.Node{
			// A new language builds the app again in it, keeping what is open: the state is the studio's, outside the build.
			core.Dynamic(func(cx *core.Scope) core.Node {
				i18n.Language(cx)
				// What is built mounts untracked, outside this function.
				return core.Func(func(cx *core.Scope) core.Node {
					return core.View{Style: fill, Children: []core.Node{
						core.Show{
							When: func() bool { return env.Window().Width >= wideWidth },
							Then: s.desktop,
							Else: s.phone,
						},
						core.Show{When: s.busy.Get, Then: s.generatingScreen},
					}}
				})
			}),
			ui.Toasts{Toaster: s.toaster, Placement: ui.ToastTop},
		},
	}
}

func newStudio(cx *core.Scope, a App) *studio {
	if a.Client == nil {
		a.Client = http.DefaultClient
	}
	store := cx.Storage()
	stored := func(key, fallback string) *core.Signal[string] {
		v, ok := store.GetItem(key)
		if !ok {
			v = fallback
		}
		return core.NewSignal(cx, v)
	}
	migrated, legacyAccount := migrateSettings(store)
	chosen, _ := store.GetItem(keyProvider)
	settings := map[string]providerSettings{}
	for _, p := range providers {
		settings[p.ID] = providerSettings{url: stored(p.storageKey("url"), ""), key: core.NewSignal(cx, ""),
			models: core.NewSignalFunc[[]string](cx, nil, slices.Equal), listing: core.NewSignal(cx, false), modelsErr: core.NewSignal(cx, ""),
			model: stored(p.storageKey("model"), ""), size: stored(p.storageKey("size"), "")}
	}
	var favorites []Item
	if v, ok := store.GetItem(keyFavorites); ok {
		_ = json.Unmarshal([]byte(v), &favorites)
	}
	if a.Pictures == nil {
		a.Pictures = defaultPictures()
	}
	s := &studio{
		App:           a,
		pics:          a.Pictures,
		cx:            cx,
		toaster:       ui.NewToaster(cx),
		section:       core.NewSignal(cx, sectionStudio),
		step:          core.NewSignal(cx, KindFace),
		note:          core.NewSignal(cx, ""),
		sheet:         core.NewSignal(cx, false),
		style:         core.NewSignal(cx, storedStyle(store)),
		busy:          core.NewSignal(cx, false),
		shot:          core.NewSignal[*shot](cx, nil),
		zoom:          core.NewSignal(cx, 100),
		provider:      core.NewSignal(cx, providerByID(chosen).ID),
		settings:      settings,
		conn:          core.NewSignal(cx, connUntested),
		connErr:       core.NewSignal(cx, ""),
		showKey:       core.NewSignal(cx, false),
		settingsShown: core.NewSignal(cx, false),
		settingsTab:   core.NewSignal(cx, settingsService),
		keyKept:       core.NewSignal(cx, true),
		works:         core.NewSignalFunc[[]work](cx, nil, nil),
		favoritesOn:   core.NewSignal(cx, false),
		favorites:     core.NewSignalFunc(cx, favorites, itemsEqual),
		uploads:       core.NewSignalFunc[[]Item](cx, nil, itemsEqual),
		closet:        core.NewSignal(cx, closetGallery),
		adultOK:       core.NewSignal(cx, isStored(store, keyAdult)),
	}
	for k := range s.chosen {
		kind := Kind(k)
		s.chosen[k] = core.NewSignal[*Item](cx, nil)
		s.filter[k] = core.NewSignal(cx, "")
		s.adult[k] = core.NewSignal(cx, false)
		s.catalogs[k] = core.NewResource(cx, func() (Kind, bool) { return kind, true },
			func(ctx context.Context, k Kind, _ core.ResourceInfo) (Catalog, error) {
				return FetchCatalog(ctx, s.Client, s.Sites, k)
			})
	}
	// Poses open on the front view rather than every angle of every pose at once.
	s.filter[KindPose].Set(poseAngleOrder[0])
	s.studioNav = base.NewNav(cx, route{page: pageHome})
	s.worksNav = base.NewNav(cx, route{page: pageHome})

	secrets := cx.Secrets()
	worksDir, uploadsDir, st := s.dir("works"), s.dir("uploads"), s.pics
	go func() {
		keys, keysErr := readKeys(secrets, migrated, legacyAccount)
		list, listErr := listWorks(st, worksDir)
		uploads, uploadsErr := listUploads(st, uploadsDir)
		clearResults(st, s.dir(resultsFolderName))
		cx.Post(func() {
			// A browser has no secure store: the key then lasts as long as the page.
			s.keyKept.Set(!errors.Is(keysErr, core.ErrSecretsUnavailable))
			for id, key := range keys {
				if sig := s.settings[id].key; sig.Get() == "" {
					sig.Set(key)
				}
			}
			if listErr == nil {
				s.works.Set(list)
			}
			if uploadsErr == nil {
				s.uploads.Set(uploads)
			}
		})
	}()
	return s
}

// dir is a folder of the app's data: works, uploads, or results not saved yet.
func (s *studio) dir(name string) string {
	root := s.DataDir
	if root == "" {
		root = s.cx.DataDir()
	}
	if root == "" {
		root = filepath.Join(os.TempDir(), "mirra")
	}
	return filepath.Join(root, name)
}

func (s *studio) choose(k Kind, it Item) {
	s.chosen[k].Set(&it)
	if sh := s.shot.Get(); sh != nil && sh.Saved == nil {
		s.shot.Set(nil)
	}
}

// clearBackground goes back to no background, which leaves the scene to the style.
func (s *studio) clearBackground() {
	s.chosen[KindBackground].Set(nil)
	if sh := s.shot.Get(); sh != nil && sh.Saved == nil {
		s.shot.Set(nil)
	}
}

// surprise chooses a face, an outfit and a pose at random, each from what its gallery shows: its chip and, only where the user turned it on, its adults-only items. The background stays as the user left it.
func (s *studio) surprise() {
	for _, k := range lookKinds {
		items := loaded(s.catalogs[k]).InGroup(s.filter[k].Get(), s.adult[k].Get())
		if len(items) > 0 {
			s.choose(k, items[rand.IntN(len(items))])
		}
	}
}

// look is the look as chosen; reading it inside a binding follows every choice.
func (s *studio) look() Request {
	r := Request{Face: s.chosen[KindFace].Get(), Costume: s.chosen[KindCostume].Get(), Pose: s.chosen[KindPose].Get(), Background: s.chosen[KindBackground].Get(),
		Note: s.note.Get(), Style: s.style.Get()}
	if s.sheet.Get() {
		r.Views = s.angles(r.Pose)
	}
	return r
}

// angles are every camera angle of a one-person pose, nil for a two-person pose, one that has a single angle, or none.
func (s *studio) angles(pose *Item) []*Item {
	if pose == nil || pose.People == 2 {
		return nil
	}
	items := loaded(s.catalogs[KindPose]).Angles(*pose)
	if len(items) < 2 {
		return nil
	}
	out := make([]*Item, len(items))
	for i := range items {
		out[i] = &items[i]
	}
	return out
}

func (s *studio) wide() bool { return s.cx.Env().Window().Width >= wideWidth }

func (s *studio) toastError(title string, err error) {
	s.toaster.Show(ui.ToastContent{Type: ui.ToastError, Title: title, Description: explain(locales.For(s.cx), err)})
}

// current is the chosen provider and what the user set for it.
func (s *studio) current() (provider, providerSettings) {
	p := providerByID(s.provider.Get())
	return p, s.settings[p.ID]
}

// config is the image API as set up in settings: the chosen provider's, with the user's address, model and size over its preset's.
func (s *studio) config() Config {
	p, set := s.current()
	return Config{Protocol: p.Protocol, BaseURL: or(set.url.Get(), p.BaseURL), APIKey: set.key.Get(), Model: or(set.model.Get(), p.Model),
		Size: or(set.size.Get(), p.Size), Local: p.KeyOptional, ReadLocal: s.pics.Read}
}

// configured says whether a Generate can be sent: an address, and a key unless the server is the user's own.
func (s *studio) configured() bool {
	cfg := s.config()
	return strings.TrimSpace(cfg.BaseURL) != "" && (cfg.Local || strings.TrimSpace(cfg.APIKey) != "")
}

// openSettings shows the AI service settings: a dialog on the desktop, the Me tab on a phone.
func (s *studio) openSettings() {
	if s.wide() {
		s.settingsTab.Set(settingsService)
		s.settingsShown.Set(true)
		return
	}
	s.section.Set(sectionSettings)
}

// generate makes the studio's look, as the first try.
func (s *studio) generate() { s.generateLook(s.look(), 1) }

// regenerate makes the shot on show again, or again with another outfit.
func (s *studio) regenerate(costume *Item) {
	sh := s.shot.Get()
	if sh == nil {
		return
	}
	req := sh.Req
	if costume != nil {
		req.Costume = costume
		s.chosen[KindCostume].Set(costume)
	}
	s.generateLook(req, sh.Attempt+1)
}

// generateLook asks the image API for req and shows the picture, kept aside until the user saves it to Works.
func (s *studio) generateLook(req Request, attempt int) {
	t := locales.For(s.cx)
	if s.busy.Get() {
		return
	}
	// Without a service the button says so, and pressing it opens the settings even before a pose is chosen.
	if !s.configured() {
		s.toaster.Show(ui.ToastContent{Type: ui.ToastWarning, Title: t.NeedService()})
		s.openSettings()
		return
	}
	if req.Validate() != nil {
		return
	}
	cfg := s.config()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	s.cancel = cancel
	s.busy.Set(true)
	dir, now, client, st := s.dir(resultsFolderName), s.cx.Now(), s.Client, s.pics
	go func() {
		pictures, err := Generate(ctx, client, cfg, req)
		// Whether the user cancelled is read before this goroutine's own cancel, which only releases the context.
		cancelled := errors.Is(ctx.Err(), context.Canceled)
		cancel()
		var path string
		if err == nil {
			path, err = writePicture(st, dir, pictures[0], now)
		}
		s.cx.Post(func() {
			s.busy.Set(false)
			s.cancel = nil
			if cancelled {
				return
			}
			if err != nil {
				s.toastError(t.GenerateFailed(), err)
				return
			}
			s.showShot(&shot{Path: path, Req: req, Attempt: attempt})
		})
	}()
}

func (s *studio) cancelGenerating() {
	if s.cancel != nil {
		s.cancel()
	}
}

// showShot puts a picture on show: in the desktop's canvas, or as the phone studio's result page.
func (s *studio) showShot(sh *shot) {
	s.shot.Set(sh)
	s.section.Set(sectionStudio)
	if !s.wide() {
		s.studioNav.SetRoutes(route{page: pageHome}, route{page: pageShot})
	}
}

// closeShot leaves the picture on show and goes back to the look.
func (s *studio) closeShot() {
	s.shot.Set(nil)
	if !s.wide() {
		s.studioNav.SetRoutes(route{page: pageHome})
	}
}

// openWork shows a saved work with the look it was made from, which Generate again starts from.
func (s *studio) openWork(w work) {
	for k, it := range map[Kind]*Item{KindFace: w.Face, KindCostume: w.Costume, KindPose: w.Pose} {
		if it != nil {
			s.chosen[k].Set(it)
		}
	}
	// A work without a background was made without one, so none is chosen for Generate again.
	s.chosen[KindBackground].Set(w.Background)
	s.sheet.Set(len(w.Views) > 1)
	s.setStyle(w.Style)
	saved := w
	s.showShot(&shot{Path: w.Path, Req: Request{Face: w.Face, Costume: w.Costume, Pose: w.Pose, Background: w.Background, Views: w.Views, Style: w.Style}, Saved: &saved})
}

// saveShot keeps the picture on show in Works, numbered after the others.
func (s *studio) saveShot(favorite bool) {
	sh := s.shot.Get()
	if sh == nil || sh.Saved != nil {
		return
	}
	t := locales.For(s.cx)
	dir, now, st := s.dir("works"), s.cx.Now(), s.pics
	meta := workMeta{Favorite: favorite, Face: sh.Req.Face, Costume: sh.Req.Costume, Pose: sh.Req.Pose, Background: sh.Req.Background, Views: sh.Req.Views, Style: sh.Req.Style}
	go func() {
		w, err := saveWork(st, dir, sh.Path, now, meta)
		s.cx.Post(func() {
			if err != nil {
				s.toastError(t.SaveFailed(), err)
				return
			}
			s.works.Update(func(ws []work) []work { return append([]work{w}, ws...) })
			next := *sh
			next.Path, next.Saved = w.Path, &w
			s.shot.Set(&next)
			s.toaster.Show(ui.ToastContent{Type: ui.ToastSuccess, Title: t.SavedToWorks(workNo(w.No)), Action: t.SeeWorks(), OnAction: func() { s.section.Set(sectionWorks) }})
		})
	}()
}

// toggleFavorite stars or unstars the picture on show, saving it first when it is not in Works yet.
func (s *studio) toggleFavorite() {
	sh := s.shot.Get()
	if sh == nil {
		return
	}
	if sh.Saved == nil {
		s.saveShot(true)
		return
	}
	w := *sh.Saved
	w.Favorite = !w.Favorite
	s.updateWork(w)
	next := *sh
	next.Saved = &w
	s.shot.Set(&next)
}

// updateWork writes a work's changed sidecar and shows it in Works.
func (s *studio) updateWork(w work) {
	t := locales.For(s.cx)
	s.works.Update(func(ws []work) []work {
		out := append([]work(nil), ws...)
		for i := range out {
			if out[i].Path == w.Path {
				out[i] = w
			}
		}
		return out
	})
	st := s.pics
	go func() {
		if err := writeMeta(st, w); err != nil {
			s.cx.Post(func() { s.toastError(t.SaveFailed(), err) })
		}
	}()
}

// discardShot deletes the picture on show, from Works too when it was saved, after asking.
func (s *studio) discardShot() {
	sh := s.shot.Get()
	if sh == nil {
		return
	}
	t := locales.For(s.cx)
	base.ShowAlertDialog(s.cx, base.AlertDialog{Title: t.DeleteConfirm(), Description: t.DeleteConfirmBody(), Actions: []core.AlertAction{
		{Label: t.Cancel(), Role: core.AlertCancel}, {Label: t.Delete(), Role: core.AlertDestructive},
	}}, func(label string) {
		if label != t.Delete() {
			return
		}
		st := s.pics
		go func() {
			var err error
			if sh.Saved != nil {
				err = deleteWork(st, *sh.Saved)
			} else {
				err = st.Remove(sh.Path)
			}
			s.cx.Post(func() {
				if err != nil {
					s.toastError(t.Delete(), err)
					return
				}
				if sh.Saved != nil {
					s.works.Update(func(ws []work) []work {
						out := make([]work, 0, len(ws))
						for _, x := range ws {
							if x.Path != sh.Saved.Path {
								out = append(out, x)
							}
						}
						return out
					})
				}
				s.toaster.Show(ui.ToastContent{Title: t.Deleted()})
				s.closeShot()
			})
		}()
	})
}

// saveFile exports a picture the app keeps through the system's save dialog, or a browser's download.
func (s *studio) saveFile(cx *core.Scope, path string) {
	t := locales.For(cx)
	opts := core.SaveOptions{Path: path, Name: "mirra" + filepath.Ext(path)}
	if !s.pics.Lasting() {
		data, err := s.pics.Read(path)
		if err != nil {
			s.toastError(t.SaveFailed(), err)
			return
		}
		opts = core.SaveOptions{Data: data, Name: opts.Name}
	}
	cx.SaveFile(opts, func(saved bool, _ string, err error) {
		switch {
		case err != nil:
			s.toastError(t.SaveFailed(), err)
		case saved:
			s.toaster.Show(ui.ToastContent{Type: ui.ToastSuccess, Title: t.Exported()})
		}
	})
}

// src shows an image: a background the app ships, one it keeps from its pictures, else from its URL.
func (s *studio) src(uri string) core.ImageSource {
	if data, ok := backgroundData(uri); ok {
		return core.ImageSource{URI: uri, Data: data}
	}
	if filepath.IsAbs(uri) {
		return s.pics.Source(uri)
	}
	return core.Src(uri)
}

// keyStyle is the storage key of the look's style.
const keyStyle = "style"

// storedStyle is the style chosen last, the studio's when none was or it is no longer offered.
func storedStyle(store core.Storage) Style {
	v, _ := store.GetItem(keyStyle)
	if st := Style(v); slices.Contains(Styles, st) {
		return st
	}
	return StyleStudio
}

// setStyle chooses the look's style and keeps it for the next launch.
func (s *studio) setStyle(st Style) {
	s.style.Set(st)
	_ = s.cx.Storage().SetItem(keyStyle, string(st))
}

// keyAdult is the storage key that holds "1" once the user said they are 18 or older.
const keyAdult = "adult-ok"

// isStored says whether key holds "1".
func isStored(store core.Storage, key string) bool {
	v, ok := store.GetItem(key)
	return ok && v == "1"
}
