package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	if err := (Request{Face: &Item{}, Costume: &Item{}}).Validate(); !errors.Is(err, ErrNoPose) {
		t.Errorf("without a pose: %v", err)
	}
	if err := (Request{Pose: &Item{}}).Validate(); err != nil {
		t.Errorf("a pose alone: %v", err)
	}
}

func TestPromptNumbersImagesInOrder(t *testing.T) {
	p := Request{Pose: &Item{}, Face: &Item{}, Note: "  golden hour  "}.Prompt()
	if !hasAll(p, "Image 1 is a faceless gray mannequin", "Image 2 is a portrait", "plain everyday clothes", "Also: golden hour") || strings.Contains(p, "Image 3") {
		t.Errorf("pose + face prompt = %q", p)
	}
	p = Request{Pose: &Item{}, Costume: &Item{}, Face: &Item{}}.Prompt()
	if !hasAll(p, "Image 2 shows an outfit", "Image 3 is a portrait") || strings.Contains(p, "everyday clothes") || strings.Contains(p, "natural adult face") {
		t.Errorf("pose + outfit + face prompt = %q", p)
	}
}

func TestTwoPersonPosesAskForTwoPeople(t *testing.T) {
	pair := &Item{People: 2}
	p := Request{Pose: pair, Costume: &Item{}, Face: &Item{}, Note: "laughing"}.Prompt()
	if !hasAll(p, "two faceless gray mannequins", "two adult people", "Image 2 shows an outfit", "dress the main person", "Image 3 is a portrait", "give the main person that face", "Also: laughing", "these two adults") || strings.Contains(p, "single adult person") || strings.Contains(p, "one adult person") {
		t.Errorf("two-person prompt = %q", p)
	}
	p = Request{Pose: pair}.Prompt()
	if !hasAll(p, "Dress both", "Give both natural, different adult faces") || strings.Contains(p, "Image 2") {
		t.Errorf("two-person pose alone = %q", p)
	}
	if p := (Request{Pose: &Item{People: 1}}).Prompt(); !strings.Contains(p, "single adult person") {
		t.Errorf("one-person pose = %q", p)
	}
}

func TestGenerateSendsReferencesInOrder(t *testing.T) {
	api := &fakeAPI{}
	srv, sites := newServer(t, api)
	ctx := context.Background()
	poses, _ := FetchCatalog(ctx, http.DefaultClient, sites, KindPose)
	costumes, _ := FetchCatalog(ctx, http.DefaultClient, sites, KindCostume)
	faces, _ := FetchCatalog(ctx, http.DefaultClient, sites, KindFace)
	winter := costumes.InGroup("冬", false)[0]

	cfg := Config{BaseURL: srv.URL + "/v1/", APIKey: " sk-test ", Size: "1024x1024"}
	got, err := Generate(ctx, http.DefaultClient, cfg, Request{Pose: &poses.Items[0], Costume: &winter, Face: &faces.Items[1], Count: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !bytes.Equal(got[0], tinyPNG(t)) || !bytes.Equal(got[1], tinyPNG(t)) {
		t.Errorf("Generate returned %d pictures, want the API's 2 decoded", len(got))
	}
	req := api.last()
	if req.Auth != "Bearer sk-test" || req.Model != defaultModel || req.Size != "1024x1024" || req.N != "2" {
		t.Errorf("request = %+v", req)
	}
	if want := []string{"full_std_P001_front.png", "WIN-0001.png", "f001.png"}; !slices.Equal(req.Images, want) {
		t.Errorf("images = %v, want pose, outfit, face: %v", req.Images, want)
	}
	// Only the three chosen images are downloaded, one each.
	if want := []string{"/pose/img/full_std_P001_front.png", "/costume/png/WIN-0001.png", "/face/img/full/f001.png"}; !slices.Equal(req.Downloads, want) {
		t.Errorf("downloads = %v, want %v", req.Downloads, want)
	}
}

func TestGenerateBoundsCount(t *testing.T) {
	api := &fakeAPI{}
	srv, _ := newServer(t, api)
	pose := &Item{Full: srv.URL + "/pose/img/a.png"}
	for _, tc := range []struct {
		count int
		want  string
	}{{0, "1"}, {maxCount + 5, "4"}} {
		if _, err := Generate(context.Background(), http.DefaultClient, Config{BaseURL: srv.URL + "/v1", APIKey: "k"}, Request{Pose: pose, Count: tc.count}); err != nil {
			t.Fatal(err)
		}
		if got := api.last().N; got != tc.want {
			t.Errorf("Count %d sent n = %s, want %s", tc.count, got, tc.want)
		}
	}
}

func TestGenerateErrors(t *testing.T) {
	api := &fakeAPI{status: http.StatusBadRequest, errBody: `{"error":{"message":"Your request was rejected by the safety system."}}`}
	srv, _ := newServer(t, api)
	pose := &Item{Full: srv.URL + "/pose/img/a.png"}
	_, err := Generate(context.Background(), http.DefaultClient, Config{BaseURL: srv.URL + "/v1", APIKey: "k"}, Request{Pose: pose})
	if err == nil || !strings.Contains(err.Error(), "safety system") {
		t.Errorf("err = %v, want the API's message", err)
	}
	if _, err := Generate(context.Background(), http.DefaultClient, Config{}, Request{Pose: pose}); !errors.Is(err, ErrNoAPIKey) {
		t.Errorf("without a key: err = %v", err)
	}
	if _, err := Generate(context.Background(), http.DefaultClient, Config{APIKey: "k"}, Request{}); !errors.Is(err, ErrNoPose) {
		t.Errorf("without a pose: err = %v", err)
	}
}

func TestGenerateLocalWithoutKey(t *testing.T) {
	api := &fakeAPI{}
	srv, _ := newServer(t, api)
	pose := &Item{Full: srv.URL + "/pose/img/a.png"}
	if _, err := Generate(context.Background(), http.DefaultClient, Config{BaseURL: srv.URL + "/v1", Local: true}, Request{Pose: pose}); err != nil {
		t.Fatal(err)
	}
	if got := api.last().Auth; got != "" {
		t.Errorf("a keyless local request sent Authorization %q", got)
	}
}

// seenEdit is an edit request as a server the user runs received it.
type seenEdit struct {
	Auth, Model, Prompt, Size, N string
	Images                       []string
}

// TestGenerateAgainstALocalServer pins the contract of a server the user runs, as stable-diffusion.cpp's and a gateway's answer it: OpenAI's multipart edit with every reference an image[] part, a model field the server is free to ignore, and the pictures as base64 alone.
func TestGenerateAgainstALocalServer(t *testing.T) {
	var mu sync.Mutex
	var seen []seenEdit
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			io.WriteString(w, `{"data":[{"id":"juggernautXL_v9"},{"id":"sd-cpp-local"}]}`)
		case "/v1/images/edits":
			if err := r.ParseMultipartForm(32 << 20); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			e := seenEdit{Auth: r.Header.Get("Authorization"), Model: r.FormValue("model"), Prompt: r.FormValue("prompt"), Size: r.FormValue("size"), N: r.FormValue("n")}
			for _, fh := range r.MultipartForm.File["image[]"] {
				e.Images = append(e.Images, fh.Filename)
			}
			mu.Lock()
			seen = append(seen, e)
			mu.Unlock()
			// The server routes by the model it loaded or its settings', not the request's, and answers with base64 alone.
			io.WriteString(w, `{"data":[{"b64_json":"`+base64.StdEncoding.EncodeToString(tinyPNG(t))+`"}]}`)
		default:
			w.Write(tinyPNG(t))
		}
	}))
	t.Cleanup(srv.Close)
	item := func(name string) *Item { return &Item{Full: srv.URL + "/" + name} }
	r := Request{Pose: item("pose.png"), Costume: item("outfit.png"), Face: item("face.png")}

	got, err := Generate(context.Background(), srv.Client(), Config{BaseURL: srv.URL + "/v1", Local: true}, r)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !bytes.Equal(got[0], tinyPNG(t)) {
		t.Errorf("Generate returned %d pictures, want the server's one decoded", len(got))
	}
	mu.Lock()
	first := seen[0]
	mu.Unlock()
	if first.Auth != "" {
		t.Errorf("a keyless local request sent Authorization %q", first.Auth)
	}
	if first.Model != defaultModel {
		t.Errorf("model = %q, want the app's, which such a server is free to ignore", first.Model)
	}
	if want := []string{"pose.png", "outfit.png", "face.png"}; !slices.Equal(first.Images, want) {
		t.Errorf("images = %v, want pose, outfit, face: %v", first.Images, want)
	}
	if first.Size != defaultSize || first.N != "1" {
		t.Errorf("size = %q, n = %s", first.Size, first.N)
	}
	if !hasAll(first.Prompt, "Image 1 is a faceless gray mannequin", "Image 2 shows an outfit", "Image 3 is a portrait") {
		t.Errorf("prompt = %q", first.Prompt)
	}

	// A gateway takes any key, so a typed one goes out as it stands.
	if _, err := Generate(context.Background(), srv.Client(), Config{BaseURL: srv.URL + "/v1", APIKey: "magpie-acme"}, r); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	second := seen[1]
	mu.Unlock()
	if second.Auth != "Bearer magpie-acme" {
		t.Errorf("Authorization = %q", second.Auth)
	}
	// Such a server's catalog names need not be an image model's, so every model it lists is offered, and its connection test asks for no key.
	ids, err := ListModels(context.Background(), srv.Client(), Config{BaseURL: srv.URL + "/v1", Local: true})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"juggernautXL_v9", "sd-cpp-local"}; !slices.Equal(ids, want) {
		t.Errorf("models = %v, want every one the server lists: %v", ids, want)
	}
	if err := CheckConnection(context.Background(), srv.Client(), Config{BaseURL: srv.URL + "/v1", Local: true}); err != nil {
		t.Errorf("a keyless server's connection test: %v", err)
	}
}

func TestGenerateReadsUploadedClothes(t *testing.T) {
	api := &fakeAPI{}
	srv, _ := newServer(t, api)
	up, err := saveUpload(filePictures{}, t.TempDir(), tinyPNG(t), time.Unix(0, 42))
	if err != nil {
		t.Fatal(err)
	}
	pose := &Item{Full: srv.URL + "/pose/img/a.png"}
	if _, err := Generate(context.Background(), http.DefaultClient, Config{BaseURL: srv.URL + "/v1", APIKey: "k"},
		Request{Pose: pose, Costume: &Item{Kind: KindCostume, ID: "MY-01", Group: uploadGroup, Full: up}}); err != nil {
		t.Fatal(err)
	}
	if got := api.last().Images; !slices.Equal(got, []string{"a.png", "42.png"}) {
		t.Errorf("images = %v", got)
	}
}

func TestCheckConnection(t *testing.T) {
	srv, _ := newServer(t, &fakeAPI{})
	if err := CheckConnection(context.Background(), http.DefaultClient, Config{BaseURL: srv.URL + "/v1/", APIKey: "k"}); err != nil {
		t.Errorf("a good key: %v", err)
	}
	err := CheckConnection(context.Background(), http.DefaultClient, Config{BaseURL: srv.URL + "/v1", APIKey: "bad"})
	if err == nil || !strings.Contains(err.Error(), "Incorrect API key") {
		t.Errorf("a bad key: %v", err)
	}
	if err := CheckConnection(context.Background(), http.DefaultClient, Config{BaseURL: srv.URL + "/nowhere"}); err == nil {
		t.Error("a wrong address passed")
	}
}

func TestSheetSendsTheAnglesAsOneImage(t *testing.T) {
	api := &fakeAPI{}
	srv, sites := newServer(t, api)
	ctx := context.Background()
	poses, _ := FetchCatalog(ctx, http.DefaultClient, sites, KindPose)
	faces, _ := FetchCatalog(ctx, http.DefaultClient, sites, KindFace)
	views := poses.Angles(poses.Items[0])
	r := Request{Pose: &views[0], Views: []*Item{&views[0], &views[1]}, Face: &faces.Items[1]}
	if _, err := Generate(ctx, http.DefaultClient, Config{BaseURL: srv.URL + "/v1", APIKey: "k", Size: "1024x1536"}, r); err != nil {
		t.Fatal(err)
	}
	req := api.last()
	if want := []string{"angles.png", "f001.png"}; !slices.Equal(req.Images, want) {
		t.Errorf("images = %v, want %v", req.Images, want)
	}
	if req.Size != "1536x1024" {
		t.Errorf("size = %q, want the portrait size on its side", req.Size)
	}
	if !hasAll(req.Prompt, "from 2 camera angles in 2 panels", "Image 2 is a portrait", "2 evenly spaced panels") || strings.Contains(req.Prompt, "Image 3") {
		t.Errorf("prompt = %q", req.Prompt)
	}
	if want := []string{"/pose/img/full_std_P001_front.png", "/pose/img/full_std_P001_side.png", "/face/img/full/f001.png"}; !slices.Equal(req.Downloads, want) {
		t.Errorf("downloads = %v, want %v", req.Downloads, want)
	}

	if p := (Request{Pose: &views[0], Views: []*Item{&views[0], &views[1]}, Costume: &Item{}}).Prompt(); !hasAll(p, "Image 2 shows an outfit", "natural adult face") {
		t.Errorf("sheet with an outfit = %q", p)
	}
	pair := &Item{People: 2}
	if p := (Request{Pose: pair, Views: []*Item{pair, pair}}).Prompt(); strings.Contains(p, "panels") {
		t.Errorf("a two-person pose asked for a sheet: %q", p)
	}
	for in, want := range map[string]string{"1024x1536": "1536x1024", "1024*1536": "1536x1024", "1536x1024": "1536x1024", "1024x1024": "1024x1024", "2K": "2K"} {
		if got := landscape(in); got != want {
			t.Errorf("landscape(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSideBySide(t *testing.T) {
	pic := part{name: "a.png", data: tinyPNG(t)}
	out, err := sideBySide([]part{pic, pic})
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(out.data))
	if err != nil {
		t.Fatal(err)
	}
	// Two 2×3 images scaled to 768 high are 512 wide each, with a 24-pixel gap around them.
	if b := img.Bounds(); b.Dx() != 24+512+24+512+24 || b.Dy() != 768+48 {
		t.Errorf("sheet is %v", b)
	}
	if _, err := sideBySide([]part{{name: "bad", data: []byte("not an image")}}); err == nil {
		t.Error("an unreadable reference made a sheet")
	}
}

func TestStylesSetTheScene(t *testing.T) {
	if p := (Request{Pose: &Item{}}).Prompt(); !hasAll(p, "Create one photorealistic photo of a single adult person", "on a plain light studio background.") {
		t.Errorf("default style = %q", p)
	}
	if p := (Request{Pose: &Item{}, Style: StyleStreet}).Prompt(); !hasAll(p, "candid street-style photo of a single adult person", "on a city street") || strings.Contains(p, "studio background") {
		t.Errorf("street = %q", p)
	}
	sheet := Request{Pose: &Item{}, Views: []*Item{{}, {}}, Style: StyleStreet}
	if p := sheet.Prompt(); !hasAll(p, "2 panels side by side", "candid street-style photo", "on a city street") {
		t.Errorf("street sheet = %q", p)
	}
	if p := (Request{Pose: &Item{}, Style: StyleAnime}).Prompt(); !strings.Contains(p, "anime illustration of a single adult person") || strings.Contains(p, "photorealistic") {
		t.Errorf("anime = %q", p)
	}
	if p := (Request{Pose: &Item{People: 2}, Style: StyleFilm}).Prompt(); !hasAll(p, "35mm film photograph of two adult people", "golden-hour light. Call the more prominent figure") {
		t.Errorf("two-person film = %q", p)
	}
}

func TestSheetFollowsTheStyle(t *testing.T) {
	a, b := &Item{Group: "正面"}, &Item{Group: "横"}
	if p := (Request{Pose: a, Views: []*Item{a, b}, Style: StyleAnime}).Prompt(); !hasAll(p, "Create one high-quality anime illustration laid out the same way", "soft cel shading") || strings.Contains(p, "photorealistic") {
		t.Errorf("anime sheet = %q", p)
	}
}

// TestBackgroundCatalog lists the shipped backgrounds by group, each with a picture the app reads.
func TestBackgroundCatalog(t *testing.T) {
	c := backgroundCatalog()
	if !slices.Equal(c.Groups, backgroundGroups) {
		t.Errorf("groups = %v, want %v", c.Groups, backgroundGroups)
	}
	if len(c.Items) < 100 {
		t.Fatalf("%d backgrounds, want at least 100", len(c.Items))
	}
	seen := map[string]bool{}
	group := 0
	for _, it := range c.Items {
		if seen[it.ID] {
			t.Errorf("%s is listed twice", it.ID)
		}
		seen[it.ID] = true
		for group < len(c.Groups) && it.Group != c.Groups[group] {
			group++
		}
		if group == len(c.Groups) {
			t.Errorf("%s is out of its group's order", it.ID)
		}
		data, ok := backgroundData(it.Full)
		if !ok || http.DetectContentType(data) != "image/jpeg" {
			t.Errorf("%s has no JPEG picture", it.ID)
		}
	}
	if _, ok := backgroundData("https://example.com/a.jpg"); ok {
		t.Error("a URL read as a shipped background")
	}
}

// TestBackgroundIsSentLast sends the background after the other references, names it in the prompt by its number in place of the style's scene, and leaves the prompt as it was without one.
func TestBackgroundIsSentLast(t *testing.T) {
	api := &fakeAPI{}
	srv, sites := newServer(t, api)
	ctx := context.Background()
	poses, _ := FetchCatalog(ctx, http.DefaultClient, sites, KindPose)
	faces, _ := FetchCatalog(ctx, http.DefaultClient, sites, KindFace)
	bg := backgroundCatalog().Items[0]
	r := Request{Pose: &poses.Items[0], Face: &faces.Items[1], Background: &bg, Style: StyleStreet}
	if _, err := Generate(ctx, http.DefaultClient, Config{BaseURL: srv.URL + "/v1", APIKey: "k"}, r); err != nil {
		t.Fatal(err)
	}
	req := api.last()
	if want := bg.ID + ".jpg"; len(req.Images) != 3 || req.Images[2] != want {
		t.Errorf("images = %v, want the background %s last", req.Images, want)
	}
	if !hasAll(req.Prompt, "in the place shown in image 3", "Image 3 is a background with no one in it") || strings.Contains(req.Prompt, "city street") {
		t.Errorf("prompt = %q", req.Prompt)
	}
	r.Background = nil
	if p := r.Prompt(); strings.Contains(p, "background with no one") || !strings.Contains(p, "city street") {
		t.Errorf("without a background, prompt = %q", p)
	}
	pair := &Item{People: 2}
	if p := (Request{Pose: pair, Costume: &Item{}, Background: &bg}).Prompt(); !hasAll(p, "in the place shown in image 3", "Image 3 is a background") {
		t.Errorf("two people with a background = %q", p)
	}
}
