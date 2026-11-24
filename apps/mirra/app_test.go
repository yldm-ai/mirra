package main

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
	"github.com/yldm-ai/mirra/apps/mirra/locales"
)

// TestMain gives the platform the main thread under nsgo test -target.
func TestMain(m *testing.M) { nsgotest.Main(m) }

const wait = 5 * time.Second

// start runs the app against local galleries and a local image API, set up as OpenAI's at the test's address, keeping its data in a temporary directory; key "" leaves OpenAI without one.
func start(t *testing.T, api *fakeAPI, key string, opts ...nsgotest.Option) (*nsgotest.App, string) {
	t.Helper()
	srv, sites := newServer(t, api)
	dir := t.TempDir()
	secrets := map[string]string{}
	if key != "" {
		secrets[providerByID(providerOpenAI).keyAccount()] = key
	}
	// The tests read English words; a target otherwise shows the machine's language.
	opts = append([]nsgotest.Option{
		nsgotest.Environment(core.Environment{Locale: "en"}),
		nsgotest.Storage(testStorage(srv.URL)),
		nsgotest.Secrets(secrets),
	}, opts...)
	app := nsgotest.Start(t, App{Client: srv.Client(), Sites: sites, DataDir: dir}.Root, opts...)
	return app, dir
}

// testStorage points OpenAI and the OpenAI-compatible server at the test's image API.
func testStorage(url string) map[string]string {
	openai, custom := providerByID(providerOpenAI), providerByID(providerCustom)
	return map[string]string{keyProvider: openai.ID, openai.storageKey("url"): url + "/v1", openai.storageKey("model"): "test-model", custom.storageKey("url"): url + "/v1"}
}

func works(t *testing.T, dir string) []work {
	t.Helper()
	ws, err := listWorks(filePictures{}, filepath.Join(dir, "works"))
	if err != nil {
		t.Fatal(err)
	}
	return ws
}

// choose presses a gallery cell until it reports itself chosen. On the macOS target a press while a pushed page is still moving in is lost, and neither Settle nor the view tree says when the move ends, so there the press first waits out the transition: https://github.com/nsgo-dev/nsgo/issues/854
func choose(t *testing.T, app *nsgotest.App, selector string) {
	t.Helper()
	app.WaitFor(selector, wait)
	if nsgotest.Target() != "" {
		time.Sleep(time.Second)
	}
	app.Press(selector)
	app.WaitUntil(wait, func() bool { return app.Find(selector).Props.Selected })
}

// The cells' labels are the galleries' own words, the same in every language; a face is labeled by its hair length and ID, so its selector matches the ID alone.
const (
	faceShort   = "label:/ · f001$/"
	outfitCoat  = "label:camel wool coat"
	outfitKnit  = "label:ivory knit denim skirt"
	poseStand   = "label:片手を腰に当てて立つ"
	poseTurning = "label:振り返る"
	// outfitLingerie is the adults-only outfit, labeled by its keywords.
	outfitLingerie = "label:x"
)

// makeLook puts a face, an outfit and a pose together on a phone, and generates.
func makeLook(t *testing.T, app *nsgotest.App) {
	t.Helper()
	app.Press("id:row-face")
	choose(t, app, faceShort)
	app.Press("id:next")
	choose(t, app, outfitCoat)
	app.Press("id:next")
	if app.Exists(poseTurning) {
		t.Error("the pose step shows a side view while the front chip is on")
	}
	choose(t, app, poseStand)
	app.Press("id:generate")
	app.WaitFor("id:shot-image", wait)
}

func TestPhoneLookIsKeptOnlyWhenSaved(t *testing.T) {
	api := &fakeAPI{}
	app, dir := start(t, api, "sk-test")
	if app.Enabled("id:generate") {
		t.Error("Generate is enabled with nothing chosen")
	}
	makeLook(t, app)

	req := api.last()
	if req.Auth != "Bearer sk-test" || req.Model != "test-model" || req.N != "1" {
		t.Errorf("request = %+v", req)
	}
	if want := []string{"full_std_P001_front.png", "WIN-0001.png", "f001.png"}; !slices.Equal(req.Images, want) {
		t.Errorf("images = %v, want %v", req.Images, want)
	}
	if n := len(works(t, dir)); n != 0 {
		t.Fatalf("a picture not saved is in Works: %d", n)
	}
	app.Press("id:save-to-works")
	app.WaitFor("id:saved", wait)
	ws := works(t, dir)
	if len(ws) != 1 || ws[0].No != 1 || ws[0].Costume == nil || ws[0].Costume.ID != "WIN-0001" || ws[0].Pose == nil || ws[0].Face == nil {
		t.Fatalf("works = %+v", ws)
	}
}

func TestRegenerateAndSwapOutfit(t *testing.T) {
	api := &fakeAPI{}
	app, _ := start(t, api, "k")
	makeLook(t, app)
	app.Press("id:regenerate")
	app.WaitUntil(wait, func() bool { return len(api.requests) == 2 })
	app.WaitFor("text:/TRY 2/", wait)

	app.Press("id:swap-0")
	app.WaitUntil(wait, func() bool { return len(api.requests) == 3 })
	app.WaitFor("text:/TRY 3/", wait)
	if got := api.last().Images; len(got) != 3 || got[1] != "SPR-0001.png" {
		t.Errorf("swapped images = %v, want the other outfit", got)
	}
}

func TestFavoriteSavesAndFilters(t *testing.T) {
	app, dir := start(t, &fakeAPI{}, "k")
	makeLook(t, app)
	app.Press("id:favorite")
	app.WaitFor("id:saved", wait)
	if ws := works(t, dir); len(ws) != 1 || !ws[0].Favorite {
		t.Fatalf("works = %+v, want one favorite", ws)
	}
	app.Press("id:close-shot")
	app.Press("text:Works")
	app.Press("id:works-favorites")
	app.WaitFor("label:Open No.01", wait)
}

func TestDeleteAShot(t *testing.T) {
	app, dir := start(t, &fakeAPI{}, "k", nsgotest.PendingAlerts())
	makeLook(t, app)
	app.Press("id:save-to-works")
	app.WaitFor("id:saved", wait)
	app.Press("id:delete")
	app.Answer(1)
	app.WaitForGone("id:shot-image", wait)
	if ws := works(t, dir); len(ws) != 0 {
		t.Errorf("works after deleting = %+v", ws)
	}
}

func TestCancelGenerating(t *testing.T) {
	api := &fakeAPI{hang: true}
	app, _ := start(t, api, "k")
	app.Press("id:row-pose")
	choose(t, app, poseStand)
	app.Press("id:generate")
	app.WaitFor("id:generating", wait)
	app.Press("id:cancel")
	app.WaitForGone("id:generating", wait)
	if app.Exists("role:alert") {
		t.Error("cancelling showed an error")
	}
	if app.Exists("id:shot-image") {
		t.Error("a cancelled look showed a picture")
	}
}

func TestNoServiceOpensSettings(t *testing.T) {
	api := &fakeAPI{}
	app, _ := start(t, api, "")
	if got := app.Find("id:generate").Props.Label; !strings.Contains(got, "AI service") {
		t.Errorf("Generate says %q without a service", got)
	}
	app.Press("id:generate")
	app.WaitFor("id:settings", wait)
	if len(api.requests) != 0 {
		t.Error("the app called the image API without a key")
	}
}

func TestLocalServiceNeedsNoKey(t *testing.T) {
	api := &fakeAPI{}
	app, _ := start(t, api, "")
	app.Press("text:Me")
	app.Press("id:provider")
	app.Press("role:option Custom (OpenAI-compatible)")
	app.Press("id:test-connection")
	app.WaitFor("id:conn-ok", wait)
	app.Press("text:Studio")
	app.Press("id:row-pose")
	choose(t, app, poseStand)
	app.Press("id:generate")
	app.WaitFor("id:shot-image", wait)
	if got := api.last().Auth; got != "" {
		t.Errorf("a keyless local request sent Authorization %q", got)
	}
}

// TestProvidersKeepTheirSettings switches to another provider, which starts on its own preset, and back to OpenAI, whose key and address are still there.
func TestProvidersKeepTheirSettings(t *testing.T) {
	app, _ := start(t, &fakeAPI{}, "k")
	app.Press("text:Me")
	app.Find("id:get-key")
	app.Press("id:provider")
	app.Press("role:option Google Gemini")
	app.WaitFor("id:conn-unconfigured", wait)
	app.Type("id:api-key", "g-key")
	app.WaitFor("id:conn-untested", wait)
	app.Press("id:provider")
	app.Press("role:option /^OpenAI$/")
	app.WaitFor("id:service-header >> text:OpenAI", wait)
	app.Find("id:conn-untested")
	app.Press("id:provider")
	app.Press("role:option stable-diffusion.cpp")
	if app.Exists("id:get-key") {
		t.Error("a server the user runs offers a link to get a key")
	}
	app.Press("id:provider")
	app.Press("role:option /^OpenAI$/")
	app.Press("id:test-connection")
	app.WaitFor("id:conn-ok", wait)
}

// TestOldSettingsCarryOver starts on the settings an earlier version kept, a local server with its own key, and generates with them.
func TestOldSettingsCarryOver(t *testing.T) {
	api := &fakeAPI{}
	srv, sites := newServer(t, api)
	app := nsgotest.Start(t, App{Client: srv.Client(), Sites: sites, DataDir: t.TempDir()}.Root,
		nsgotest.Environment(core.Environment{Locale: "en"}),
		nsgotest.Storage(map[string]string{legacyService: "local", legacyLocalURL: srv.URL + "/v1", legacyModel: "old-model"}),
		nsgotest.Secrets(map[string]string{legacyLocalKey: "local-key"}))
	app.Press("id:row-pose")
	choose(t, app, poseStand)
	app.WaitUntil(wait, func() bool { return !strings.Contains(app.Find("id:generate").Props.Label, "AI service") })
	app.Press("id:generate")
	app.WaitFor("id:shot-image", wait)
	if req := api.last(); req.Auth != "Bearer local-key" || req.Model != "old-model" {
		t.Errorf("request = %+v, want the old server's key and model", req)
	}
}

// TestAdvancedSettings opens on the provider and key alone, keeps the address, model and size under Advanced, sends the size chosen there, and resets them to the preset's.
func TestAdvancedSettings(t *testing.T) {
	fresh := nsgotest.Start(t, App{DataDir: t.TempDir()}.Root, nsgotest.Environment(core.Environment{Locale: "en"}))
	fresh.Press("text:Me")
	fresh.Find("id:api-key")
	fresh.Find("id:model")
	fresh.Find("id:version")
	if fresh.Exists("id:base-url") {
		t.Error("a fresh install shows the address before Advanced is opened")
	}
	fresh.Press("id:advanced >> role:button")
	fresh.WaitFor("id:base-url", wait)
	fresh.Find("id:size-portrait")
	// Azure's model is the deployment the user named, so it shows with the key and the address.
	fresh.Press("id:provider")
	fresh.Press("role:option Azure OpenAI")
	fresh.WaitFor("text:Deployment name", wait)
	fresh.Find("id:model")
	fresh.Find("id:base-url")

	api := &fakeAPI{}
	app, _ := start(t, api, "k")
	app.Press("text:Me")
	// The test's address and model differ from OpenAI's, so Advanced opens on its own.
	app.Find("id:base-url")
	app.Press("id:size-square")
	app.Press("text:Studio")
	app.Press("id:row-pose")
	choose(t, app, poseStand)
	app.Press("id:generate")
	app.WaitFor("id:shot-image", wait)
	if got := api.last().Size; got != sizeSquare {
		t.Errorf("size = %q, want the square one chosen", got)
	}
	app.Press("id:close-shot")
	app.Press("text:Me")
	app.Press("id:reset-defaults")
	app.WaitUntil(wait, func() bool { return app.Find("id:reset-defaults").Props.Disabled })
	if !app.Find("id:size-portrait").Props.Selected {
		t.Error("resetting left the size other than the preset's portrait")
	}
}

// TestDesktopSettingsTabs puts the desktop settings on tabs down the side: the service first, the address, model and size on their own tab, and Generate without a service back on the service's.
func TestDesktopSettingsTabs(t *testing.T) {
	app := nsgotest.Start(t, App{DataDir: t.TempDir()}.Root, nsgotest.Environment(core.Environment{Locale: "en"}), nsgotest.Size(1440, 1000))
	app.Press("id:open-settings")
	app.WaitFor("id:api-key", wait)
	app.Find("id:test-connection")
	// The model is chosen beside the key, whose models it lists.
	app.Find("id:model")
	if app.Exists("id:base-url") || app.Exists("id:version") {
		t.Error("the service's tab shows another tab's settings")
	}
	app.Press("id:settings-nav-advanced")
	app.WaitFor("id:base-url", wait)
	app.Find("id:size-portrait")
	if app.Exists("id:api-key") || app.Exists("id:model") {
		t.Error("the advanced tab shows the key or the model")
	}
	app.Press("id:settings-nav-sources")
	app.WaitFor("text:Poses", wait)
	app.Press("id:settings-close")
	// The dialog leaves once its motion ends, which runs on the test's clock.
	app.Advance(200 * time.Millisecond)
	app.WaitForGone("id:settings-dialog", wait)
	app.Press("id:generate")
	app.WaitFor("id:api-key", wait)
}

// TestCheckForUpdatesWhereTheAppUpdates offers the check on the platforms whose copies update themselves from the releases, and not on the web or iOS.
func TestCheckForUpdatesWhereTheAppUpdates(t *testing.T) {
	for _, c := range []struct {
		os   core.OS
		want bool
	}{{core.OSMacOS, true}, {core.OSWindows, true}, {core.OSAndroid, true}, {core.OSWeb, false}, {core.OSIOS, false}} {
		// A target runs on its own OS, which an Environment does not change.
		if nsgotest.Target() != "" && c.os != core.OSMacOS {
			continue
		}
		app := nsgotest.Start(t, App{DataDir: t.TempDir()}.Root, nsgotest.Environment(core.Environment{Locale: "en", OS: c.os}), nsgotest.Size(1440, 1000))
		app.Press("id:open-settings")
		app.Press("id:settings-nav-about")
		app.WaitFor("id:version", wait)
		if got := app.Exists("id:check-updates"); got != c.want {
			t.Errorf("on %v the check is offered: %v, want %v", c.os, got, c.want)
		}
	}
}

func TestConnectionTestReportsFailure(t *testing.T) {
	app, _ := start(t, &fakeAPI{}, "bad")
	app.Press("text:Me")
	app.Press("id:test-connection")
	app.WaitFor("id:conn-failed", wait)
	app.Find("id:conn-failed-short")
	if got := app.Find("id:conn-failed").Text; !strings.Contains(got, "refused the key") {
		t.Errorf("the reason under the header is %q", got)
	}
}

// TestAdultChipAsksAge keeps the adults-only outfits out of sight until the user says they are 18 or older.
func TestAdultChipAsksAge(t *testing.T) {
	app, _ := start(t, &fakeAPI{}, "k", nsgotest.PendingAlerts())
	app.Press("text:Wardrobe")
	app.Press("id:closet-gallery")
	app.WaitFor(outfitKnit, wait)
	if app.Exists(outfitLingerie) {
		t.Fatal("an adults-only outfit shows before the user was asked")
	}
	app.Press("id:chip-adult")
	app.Answer(0)
	if app.Exists(outfitLingerie) || app.Find("id:chip-adult").Props.Selected {
		t.Fatal("answering under 18 showed the adults-only outfits")
	}
	app.Press("id:chip-adult")
	app.Answer(1)
	app.WaitFor(outfitLingerie, wait)
	if app.Exists(outfitKnit) {
		t.Error("the adults-only outfits show with those for everyone")
	}
	app.Find("id:chip-Lingerie")
	app.Press("id:chip-adult")
	app.WaitFor(outfitKnit, wait)
	app.Press("id:chip-adult")
	app.WaitFor(outfitLingerie, wait)
}

// TestAgeAnswerIsKept remembers that the user is 18 or older, since works and favorites from the adults-only categories show again on the next launch.
func TestAgeAnswerIsKept(t *testing.T) {
	app, _ := start(t, &fakeAPI{}, "k", nsgotest.PendingAlerts())
	app.Press("text:Wardrobe")
	app.Press("id:closet-gallery")
	app.WaitFor(outfitKnit, wait)
	app.Press("id:chip-adult")
	app.Answer(1)
	app.WaitFor(outfitLingerie, wait)
	if v, ok := app.Storage().GetItem(keyAdult); !ok || v != "1" {
		t.Fatalf("the answer was not kept: %q, %v", v, ok)
	}

	api := &fakeAPI{}
	srv, _ := newServer(t, api)
	stored := testStorage(srv.URL)
	stored[keyAdult] = "1"
	again, _ := start(t, api, "k", nsgotest.PendingAlerts(), nsgotest.Storage(stored))
	again.Press("text:Wardrobe")
	again.Press("id:closet-gallery")
	again.WaitFor(outfitKnit, wait)
	again.Press("id:chip-adult")
	again.WaitFor(outfitLingerie, wait)
	if a := again.Alerts(); len(a) != 0 {
		t.Errorf("asked again: %+v", a)
	}
}

func TestWardrobeFavorites(t *testing.T) {
	app, _ := start(t, &fakeAPI{}, "k")
	app.Press("text:Wardrobe")
	app.Press("id:closet-favorites")
	app.Find("text:No favorites yet")
	app.Press("id:closet-gallery")
	app.WaitFor(outfitKnit, wait)
	app.Press("label:Add to favorites >> nth=0")
	app.Press("id:closet-favorites")
	app.WaitFor(outfitKnit, wait)
}

func TestUploadedClothesAreWorn(t *testing.T) {
	if nsgotest.Target() != "" {
		t.Skip("a target never hands the test's PickImages answer to the app: https://github.com/nsgo-dev/nsgo/issues/866")
	}
	api := &fakeAPI{}
	app, _ := start(t, api, "k")
	app.Press("text:Wardrobe")
	app.PickImages = func(core.PickRequest) []core.PickedFile {
		return []core.PickedFile{{Name: "shirt.png", MIME: "image/png", Data: tinyPNG(t)}}
	}
	app.Press("id:upload")
	app.WaitFor("label:My clothes 01", wait)
	app.WaitFor("id:wearing", wait)

	app.Press("id:wear-generate")
	app.WaitFor("id:steps", wait)
	choose(t, app, poseStand)
	app.Press("id:generate")
	app.WaitFor("id:shot-image", wait)
	if got := api.last().Images; len(got) != 2 || got[1] == "WIN-0001.png" || !strings.HasSuffix(got[1], ".png") {
		t.Errorf("images = %v, want the pose and the uploaded clothes", got)
	}
}

func TestGalleryErrorOffersRetry(t *testing.T) {
	app := nsgotest.Start(t, App{Sites: Sites{Pose: "http://127.0.0.1:1", Costume: "http://127.0.0.1:1", Face: "http://127.0.0.1:1"}, DataDir: t.TempDir()}.Root)
	app.Press("id:row-costume")
	app.WaitFor("id:retry", wait)
}

func TestDesktopStudio(t *testing.T) {
	api := &fakeAPI{}
	app, dir := start(t, api, "k", nsgotest.Size(1440, 1000))
	app.Find("id:desktop")
	choose(t, app, faceShort)
	app.Press("id:library-costume")
	choose(t, app, outfitCoat)
	app.Press("id:quick-pose")
	choose(t, app, poseStand)
	app.Press("id:zoom-in")
	app.Find("text:125%")
	app.Press("id:generate")
	app.WaitFor("id:shot-image", wait)
	if want := []string{"full_std_P001_front.png", "WIN-0001.png", "f001.png"}; !slices.Equal(api.last().Images, want) {
		t.Errorf("images = %v, want %v", api.last().Images, want)
	}
	app.Press("id:save-to-works")
	app.WaitFor("id:saved", wait)
	if ws := works(t, dir); len(ws) != 1 {
		t.Errorf("works = %d, want 1", len(ws))
	}
	app.Press("id:close-shot")
	app.WaitFor("id:zoom", wait)
	app.Press("id:open-settings")
	app.WaitFor("id:settings", wait)
}

// TestPicturesInMemory runs the look as the web version does, with nothing written to disk: the picture is saved to Works in memory and the page says it lasts only as long as the page.
func TestPicturesInMemory(t *testing.T) {
	api := &fakeAPI{}
	srv, sites := newServer(t, api)
	mem := newMemPictures()
	app := nsgotest.Start(t, App{Client: srv.Client(), Sites: sites, DataDir: "/web", Pictures: mem}.Root,
		nsgotest.Environment(core.Environment{Locale: "en"}),
		nsgotest.Storage(testStorage(srv.URL)),
		nsgotest.Secrets(map[string]string{providerByID(providerOpenAI).keyAccount(): "k"}))
	app.Press("id:row-pose")
	choose(t, app, poseStand)
	app.Press("id:generate")
	app.WaitFor("id:shot-image", wait)
	if got := app.Find("id:shot-image").Image.Source; len(got.Data) == 0 {
		t.Errorf("the result shows from %q, not the memory it is kept in", got.URI)
	}
	app.Press("id:save-to-works")
	app.WaitFor("id:saved", wait)
	ws, err := listWorks(mem, "/web/works")
	if err != nil || len(ws) != 1 {
		t.Fatalf("works in memory = %v, %v", ws, err)
	}
	app.Press("id:close-shot")
	app.Press("text:Works")
	app.WaitFor("id:page-only", wait)
}

// TestMacMenuOpensSettingsAndChecksForUpdates gives the macOS menu bar the app menu's Settings… and Check for Updates…, where Mac users look for them.
func TestMacMenuOpensSettingsAndChecksForUpdates(t *testing.T) {
	if nsgotest.Target() != "" && nsgotest.Target() != "macos" {
		t.Skip("the menu is macOS's")
	}
	app := nsgotest.Start(t, App{DataDir: t.TempDir()}.Root, nsgotest.Environment(core.Environment{Locale: "en", OS: core.OSMacOS}), nsgotest.Size(1440, 1000))
	app.ChooseMenu("Mirra", "Settings…")
	app.WaitFor("id:settings", wait)
	if nsgotest.Target() == "" {
		app.ChooseMenu("Mirra", "Check for Updates…")
		if checks := app.UpdateChecks; len(checks) != 1 || checks[0] {
			t.Errorf("update checks %v, want one asked for", checks)
		}
	}
	other := nsgotest.Start(t, App{DataDir: t.TempDir()}.Root, nsgotest.Environment(core.Environment{Locale: "en", OS: core.OSWindows}), nsgotest.Size(1440, 1000))
	if nsgotest.Target() == "" && len(other.Menu) != 0 {
		t.Errorf("Windows got a menu bar: %v", other.Menu)
	}
}

func TestSheetOfAllAngles(t *testing.T) {
	api := &fakeAPI{}
	app, dir := start(t, api, "k")
	app.Press("id:row-face")
	choose(t, app, faceShort)
	app.Press("id:next")
	choose(t, app, outfitCoat)
	app.Press("id:next")
	if app.Exists("id:chip-sheet") {
		t.Error("All angles shows before a pose is chosen")
	}
	choose(t, app, poseStand)
	app.WaitFor("id:chip-sheet", wait)
	app.Press("id:chip-sheet")
	app.WaitUntil(wait, func() bool { return app.Find("id:chip-sheet").Props.Selected })
	app.Press("id:generate")
	app.WaitFor("id:shot-image", wait)
	if got := api.last().Images; !slices.Equal(got, []string{"angles.png", "WIN-0001.png", "f001.png"}) {
		t.Errorf("images = %v", got)
	}
	app.Press("id:save-to-works")
	app.WaitFor("id:saved", wait)
	ws := works(t, dir)
	if len(ws) != 1 || len(ws[0].Views) != 2 || ws[0].Views[1].ID != "full_std_P001_side" {
		t.Fatalf("works = %+v", ws)
	}
	app.Press("id:regenerate")
	app.WaitUntil(wait, func() bool { return len(api.requests) == 2 })
	if got := api.last().Images; len(got) == 0 || got[0] != "angles.png" {
		t.Errorf("Generate again sent %v, not the sheet", got)
	}
}

func TestSurpriseChoosesAShownLook(t *testing.T) {
	api := &fakeAPI{}
	app, _ := start(t, api, "k")
	app.WaitFor("id:surprise", wait)
	// The galleries load after the first frame; a press before they have is a no-op.
	app.WaitUntil(wait, func() bool {
		app.Press("id:surprise")
		return app.Enabled("id:generate")
	})
	app.Press("id:generate")
	app.WaitFor("id:shot-image", wait)
	got := api.last().Images
	if len(got) != 3 || got[0] != "full_std_P001_front.png" || (got[1] != "SPR-0001.png" && got[1] != "WIN-0001.png") || (got[2] != "f001.png" && got[2] != "f002.png") {
		t.Errorf("images = %v, want the front pose, an outfit and a face for everyone", got)
	}
}

// TestSurpriseKeepsAdultChoicesOut turns a gallery's 18+ chip on, then asks for a surprise from the phone's home, where no gallery is showing, and checks the draw stayed with the items for everyone: an 18+ chip is a choice made against its own gallery, not against the dice on another screen.
func TestSurpriseKeepsAdultChoicesOut(t *testing.T) {
	api := &fakeAPI{}
	app, _ := start(t, api, "k", nsgotest.PendingAlerts())
	app.Press("text:Wardrobe")
	app.Press("id:closet-gallery")
	app.WaitFor(outfitKnit, wait)
	app.Press("id:chip-adult")
	app.Answer(1)
	app.WaitFor(outfitLingerie, wait)

	app.Press("text:Studio")
	app.WaitFor("id:surprise", wait)
	// The galleries load after the first frame; a press before they have is a no-op.
	app.WaitUntil(wait, func() bool {
		app.Press("id:surprise")
		return app.Enabled("id:generate")
	})
	app.Press("id:generate")
	app.WaitFor("id:shot-image", wait)
	got := api.last().Images
	for _, img := range got {
		if img == "LNG-0001.png" || img == "full_sexy_P000_front.png" || img == "full_sexy_P004_front.png" {
			t.Errorf("images = %v, want no adults-only item in a surprise made on the home screen", got)
		}
	}
	if len(got) != 3 || got[0] != "full_std_P001_front.png" || (got[1] != "SPR-0001.png" && got[1] != "WIN-0001.png") || (got[2] != "f001.png" && got[2] != "f002.png") {
		t.Errorf("images = %v, want the front pose, an outfit and a face for everyone", got)
	}
}

// TestDesktopColumnsResize opens the side columns at a share of the window, wider in a wider one, lets the user drag the gutter between them, and keeps what they dragged to for the next launch.
func TestDesktopColumnsResize(t *testing.T) {
	width := func(app *nsgotest.App, id string) core.Points { return app.Find("id:" + id).Frame.Size.Width }
	small := nsgotest.Start(t, App{DataDir: t.TempDir()}.Root, nsgotest.Environment(core.Environment{Locale: "en"}), nsgotest.Size(1280, 900))
	large := nsgotest.Start(t, App{DataDir: t.TempDir()}.Root, nsgotest.Environment(core.Environment{Locale: "en"}), nsgotest.Size(2400, 1300))
	if s, l := width(small, "column-library"), width(large, "column-library"); l <= s {
		t.Errorf("the library is %v wide in the large window and %v in the small one, want it wider in the large", l, s)
	}
	if s, l := width(small, "column-look"), width(large, "column-look"); l <= s {
		t.Errorf("the look is %v wide in the large window and %v in the small one, want it wider in the large", l, s)
	}

	dir := t.TempDir()
	app := nsgotest.Start(t, App{DataDir: dir}.Root, nsgotest.Environment(core.Environment{Locale: "en"}), nsgotest.Size(1440, 1000))
	before := width(app, "column-library")
	app.Drag("id:columns-handle-0", core.Point{X: 20}, core.Point{X: 40}, core.Point{X: 60})
	after := width(app, "column-library")
	if after-before < 50 {
		t.Fatalf("dragging the gutter 60 points took the library from %v to %v", before, after)
	}
	share, ok := app.Storage().GetItem(keyLibraryShare)
	if !ok {
		t.Fatal("the dragged width was not kept")
	}
	again := nsgotest.Start(t, App{DataDir: dir}.Root, nsgotest.Environment(core.Environment{Locale: "en"}), nsgotest.Size(1440, 1000),
		nsgotest.Storage(map[string]string{keyLibraryShare: share}))
	if got := width(again, "column-library"); got < after-2 || got > after+2 {
		t.Errorf("the next launch opens the library %v wide, want the %v it was dragged to", got, after)
	}
}

// TestLanguageSetting switches the app's language in the settings, on the phone and on the desktop, at once and for the next launch, and follows the system's again.
func TestLanguageSetting(t *testing.T) {
	dir := t.TempDir()
	app := nsgotest.Start(t, App{DataDir: dir}.Root, nsgotest.Environment(core.Environment{Locale: "en"}))
	app.Press("text:Me")
	app.Press("id:language")
	app.Press("role:option 简体中文")
	app.WaitFor("text:工作室", wait)
	if got, _ := app.Storage().GetItem(keyLanguage); got != "zh-Hans" {
		t.Errorf("kept %q, want zh-Hans", got)
	}
	again := nsgotest.Start(t, App{DataDir: dir}.Root, nsgotest.Environment(core.Environment{Locale: "en"}),
		nsgotest.Storage(map[string]string{keyLanguage: "zh-Hans"}))
	again.Find("text:工作室")
	again.Press("text:我")
	again.Press("id:language")
	again.Press("role:option 跟随系统")
	again.WaitFor("text:Studio", wait)
	if _, ok := again.Storage().GetItem(keyLanguage); ok {
		t.Error("following the system keeps a language")
	}

	desk := nsgotest.Start(t, App{DataDir: t.TempDir()}.Root, nsgotest.Environment(core.Environment{Locale: "en"}), nsgotest.Size(1440, 1000))
	desk.Press("id:open-settings")
	desk.Press("id:settings-nav-general")
	desk.Press("id:language")
	desk.Press("role:option 简体中文")
	// The settings stay open, in the new language.
	desk.WaitFor("role:header 设置", wait)
	desk.Find("id:settings-nav-general")
	desk.Find("id:settings-dialog")
}

func TestStyleIsSentKeptAndRestored(t *testing.T) {
	api := &fakeAPI{}
	app, dir := start(t, api, "k")
	app.Press("id:style-Street")
	makeLook(t, app)
	if p := api.last().Prompt; !strings.Contains(p, "on a city street") {
		t.Errorf("prompt = %q", p)
	}
	if v, _ := app.Storage().GetItem(keyStyle); v != string(StyleStreet) {
		t.Errorf("stored style = %q", v)
	}
	app.Press("id:save-to-works")
	app.WaitFor("id:saved", wait)
	if ws := works(t, dir); len(ws) != 1 || ws[0].Style != StyleStreet {
		t.Fatalf("works = %+v", ws)
	}

	srv, _ := newServer(t, api)
	stored := testStorage(srv.URL)
	stored[keyStyle] = string(StyleFilm)
	again, _ := start(t, api, "k", nsgotest.Storage(stored))
	again.WaitFor("id:style-Film", wait)
	if !again.Find("id:style-Film").Props.Selected || again.Find("id:style-Studio").Props.Selected {
		t.Error("the stored style is not the one chosen")
	}
}

func TestModelIsChosenFromTheListedOnes(t *testing.T) {
	api := &fakeAPI{}
	app, _ := start(t, api, "k")
	app.Press("text:Me")
	app.WaitFor("id:model", wait)
	app.Press("id:model")
	app.WaitFor("role:option gpt-image-2", wait)
	if app.Exists("role:option gpt-5") {
		t.Error("a chat model is offered")
	}
	app.Press("role:option gpt-image-2")
	// gpt-image-2 is OpenAI's preset, so the setting follows the preset again.
	if v, _ := app.Storage().GetItem(providerByID(providerOpenAI).storageKey("model")); v != "" {
		t.Errorf("stored model = %q, want the preset's", v)
	}
	app.Press("text:Studio")
	app.Press("id:row-pose")
	choose(t, app, poseStand)
	app.Press("id:generate")
	app.WaitFor("id:shot-image", wait)
	if got := api.last().Model; got != "gpt-image-2" {
		t.Errorf("model = %q", got)
	}
}

// TestEveryLanguageNamesItself lists each catalog in the language menu under its own name, so a catalog that lacks one does not show as English.
func TestEveryLanguageNamesItself(t *testing.T) {
	seen := map[string]string{}
	for _, lang := range locales.Catalog.Languages() {
		name := locales.In(lang).LanguageName()
		if other, ok := seen[name]; ok {
			t.Errorf("%s and %s are both named %q", other, lang, name)
		}
		seen[name] = lang
	}
	if len(seen) < 11 {
		t.Errorf("the menu offers %d languages, want 11", len(seen))
	}
}

// TestBackgroundTab starts without a background, sends the one chosen on the background tab with the look, keeps it with the work, and goes back to none.
func TestBackgroundTab(t *testing.T) {
	api := &fakeAPI{}
	app, dir := start(t, api, "k", nsgotest.Size(1440, 1000))
	app.Find("text:No background")
	app.Press("id:library-pose")
	choose(t, app, poseStand)
	app.Press("id:library-background")
	bg := backgroundCatalog().Items[0]
	choose(t, app, "label:Indoor · "+bg.ID)
	app.WaitFor("id:clear-background", wait)
	app.Press("id:generate")
	app.WaitFor("id:shot-image", wait)
	if got := api.last().Images; len(got) == 0 || got[len(got)-1] != bg.ID+".jpg" {
		t.Errorf("images = %v, want the background last", got)
	}
	app.Press("id:save-to-works")
	app.WaitFor("id:saved", wait)
	if ws := works(t, dir); len(ws) != 1 || ws[0].Background == nil || ws[0].Background.ID != bg.ID {
		t.Errorf("works = %+v, want one keeping the background", ws)
	}
	app.Press("id:close-shot")
	app.Press("id:clear-background")
	app.WaitFor("text:No background", wait)
	app.Press("id:generate")
	app.WaitFor("id:shot-image", wait)
	if got := api.last().Images; slices.Contains(got, bg.ID+".jpg") {
		t.Errorf("images = %v after removing the background", got)
	}
	// Pressing the chosen background again also goes back to none.
	app.Press("id:close-shot")
	app.Press("id:library-background")
	choose(t, app, "label:Indoor · "+bg.ID)
	app.WaitFor("id:clear-background", wait)
	app.Press("label:Indoor · " + bg.ID)
	app.WaitFor("text:No background", wait)
}

// TestAdultSetting turns the galleries' 18+ categories on in the settings, which asks the age once, and off again, which forgets the answer and leaves an 18+ category on show.
func TestAdultSetting(t *testing.T) {
	app, _ := start(t, &fakeAPI{}, "k", nsgotest.PendingAlerts())
	app.Press("text:Me")
	if app.Find("id:adult-switch").Props.Checked {
		t.Fatal("the 18+ switch is on before the user was asked")
	}
	app.Press("id:adult-switch")
	app.Answer(0)
	if app.Find("id:adult-switch").Props.Checked {
		t.Error("answering under 18 turned the switch on")
	}
	app.Press("id:adult-switch")
	app.Answer(1)
	app.WaitUntil(wait, func() bool { return app.Find("id:adult-switch").Props.Checked })
	if !isStored(app.Storage(), keyAdult) {
		t.Error("the answer was not kept")
	}
	// With the answer kept, the wardrobe's 18+ chip shows its outfits without asking.
	app.Press("text:Wardrobe")
	app.Press("id:closet-gallery")
	app.WaitFor(outfitKnit, wait)
	app.Press("id:chip-adult")
	app.WaitFor(outfitLingerie, wait)
	app.Press("text:Me")
	app.Press("id:adult-switch")
	app.WaitUntil(wait, func() bool { return !app.Find("id:adult-switch").Props.Checked })
	if isStored(app.Storage(), keyAdult) {
		t.Error("turning the switch off kept the answer")
	}
	app.Press("text:Wardrobe")
	app.WaitFor(outfitKnit, wait)
	if app.Exists(outfitLingerie) {
		t.Error("an 18+ category stayed on show after the switch was turned off")
	}
}

// TestServiceHeader shows the chosen provider with the connection's state, untested until the test passes.
func TestServiceHeader(t *testing.T) {
	app, _ := start(t, &fakeAPI{}, "k", nsgotest.Size(1440, 1000))
	app.Press("id:open-settings")
	app.Press("id:settings-nav-service")
	app.WaitFor("id:service-header", wait)
	app.Find("id:conn-untested")
	app.Press("id:test-connection")
	app.WaitFor("id:conn-ok", wait)
	if app.Exists("id:conn-untested") {
		t.Error("the header still says untested after the test passed")
	}
}
