package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	ailogo "github.com/yldm-tech/ai-logo/nsgo"
)

// apiCall is what a fake provider received.
type apiCall struct {
	Method, Path, Query string
	Header              http.Header
	Body                map[string]any
	Form                map[string][]string
	Files               []string
}

// fakeProvider answers path with answer and records every call; answer gets the server's URL, for links to pictures it serves at /pic.png.
func fakeProvider(t *testing.T, path string, answer func(url string, w http.ResponseWriter, c apiCall)) (*fakeServer, func() []apiCall) {
	t.Helper()
	var mu sync.Mutex
	var calls []apiCall
	var srv *fakeServer
	srv = serve(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pic.png" {
			w.Write(tinyPNG(t))
			return
		}
		c := apiCall{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Header: r.Header.Clone()}
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
			if err := r.ParseMultipartForm(32 << 20); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			c.Form = r.MultipartForm.Value
			for _, fh := range r.MultipartForm.File["image[]"] {
				c.Files = append(c.Files, fh.Filename)
			}
		} else if r.Body != nil {
			data, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(data, &c.Body)
		}
		mu.Lock()
		calls = append(calls, c)
		mu.Unlock()
		if r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		answer(srv.URL, w, c)
	}))
	return srv, func() []apiCall {
		mu.Lock()
		defer mu.Unlock()
		return append([]apiCall(nil), calls...)
	}
}

// testParts are two reference images, a PNG and a JPEG-typed one.
func testParts(t *testing.T) []part {
	return []part{{name: "pose.png", mime: "image/png", data: tinyPNG(t)}, {name: "outfit.jpg", mime: "image/jpeg", data: []byte("jpeg")}}
}

func b64(t *testing.T) string { return base64.StdEncoding.EncodeToString(tinyPNG(t)) }

// wantPictures checks that pics are n copies of the test picture.
func wantPictures(t *testing.T, pics [][]byte, err error, n int) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if len(pics) != n {
		t.Fatalf("got %d pictures, want %d", len(pics), n)
	}
	for _, p := range pics {
		if !bytes.Equal(p, tinyPNG(t)) {
			t.Fatal("a picture is not the one the API sent")
		}
	}
}

func asList(v any) []any {
	l, _ := v.([]any)
	return l
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func TestGeminiRequest(t *testing.T) {
	srv, calls := fakeProvider(t, "/v1beta/models/img-model:generateContent", func(_ string, w http.ResponseWriter, _ apiCall) {
		io.WriteString(w, `{"candidates":[{"content":{"parts":[{"text":"Here you go"},{"inlineData":{"mimeType":"image/png","data":"`+b64(t)+`"}}]}}]}`)
	})
	cfg := Config{Protocol: protoGemini, BaseURL: srv.URL + "/v1beta", APIKey: " gk ", Model: "img-model", Size: "1024x1536"}
	pics, err := generate(context.Background(), srv.Client(), cfg, "the prompt", testParts(t), 2)
	wantPictures(t, pics, err, 2)
	got := calls()
	if len(got) != 2 {
		t.Fatalf("calls = %d, want one a picture", len(got))
	}
	c := got[0]
	if c.Header.Get("x-goog-api-key") != "gk" || c.Header.Get("Authorization") != "" {
		t.Errorf("headers = %v", c.Header)
	}
	parts := asList(asMap(asList(c.Body["contents"])[0])["parts"])
	if len(parts) != 3 || asMap(parts[0])["text"] != "the prompt" {
		t.Fatalf("parts = %v, want the prompt and two images", parts)
	}
	img := asMap(asMap(parts[2])["inlineData"])
	if img["mimeType"] != "image/jpeg" || img["data"] != base64.StdEncoding.EncodeToString([]byte("jpeg")) {
		t.Errorf("second image = %v, want plain base64 of its bytes", img)
	}
	if ratio := asMap(asMap(c.Body["generationConfig"])["imageConfig"])["aspectRatio"]; ratio != "2:3" {
		t.Errorf("aspectRatio = %v, want 2:3 for 1024x1536", ratio)
	}
}

func TestGeminiBlocked(t *testing.T) {
	srv, _ := fakeProvider(t, "/v1beta/models/m:generateContent", func(_ string, w http.ResponseWriter, _ apiCall) {
		io.WriteString(w, `{"promptFeedback":{"blockReason":"SAFETY"}}`)
	})
	_, err := generate(context.Background(), srv.Client(), Config{Protocol: protoGemini, BaseURL: srv.URL + "/v1beta", APIKey: "k", Model: "m"}, "p", testParts(t), 1)
	if err == nil || !strings.Contains(err.Error(), "SAFETY") {
		t.Errorf("err = %v, want the block reason", err)
	}
}

func TestSeedreamRequest(t *testing.T) {
	srv, calls := fakeProvider(t, "/api/v3/images/generations", func(_ string, w http.ResponseWriter, _ apiCall) {
		io.WriteString(w, `{"data":[{"b64_json":"`+b64(t)+`","size":"1024x1536"}]}`)
	})
	cfg := Config{Protocol: protoSeedream, BaseURL: srv.URL + "/api/v3/", APIKey: "ark", Model: "seedream", Size: "1024x1536"}
	pics, err := generate(context.Background(), srv.Client(), cfg, "p", testParts(t), 3)
	wantPictures(t, pics, err, 3)
	got := calls()
	if len(got) != 3 {
		t.Fatalf("calls = %d, want one a picture", len(got))
	}
	c := got[0]
	images := asList(c.Body["image"])
	if c.Header.Get("Authorization") != "Bearer ark" || len(images) != 2 || images[0] != "data:image/png;base64,"+b64(t) {
		t.Errorf("auth %q, images %v", c.Header.Get("Authorization"), images)
	}
	if c.Body["size"] != "1024x1536" || c.Body["response_format"] != "b64_json" || c.Body["watermark"] != false || c.Body["model"] != "seedream" {
		t.Errorf("body = %v", c.Body)
	}
}

func TestXAIRequest(t *testing.T) {
	srv, calls := fakeProvider(t, "/v1/images/edits", func(_ string, w http.ResponseWriter, _ apiCall) {
		io.WriteString(w, `{"data":[{"b64_json":"`+b64(t)+`"},{"b64_json":"`+b64(t)+`"}]}`)
	})
	cfg := Config{Protocol: protoXAI, BaseURL: srv.URL + "/v1", APIKey: "xk", Model: "grok", Size: "1024x1536"}
	pics, err := generate(context.Background(), srv.Client(), cfg, "p", testParts(t), 2)
	wantPictures(t, pics, err, 2)
	c := calls()[0]
	images := asList(c.Body["images"])
	if len(images) != 2 || asMap(images[1])["url"] != "data:image/jpeg;base64,"+base64.StdEncoding.EncodeToString([]byte("jpeg")) {
		t.Errorf("images = %v", images)
	}
	if c.Body["n"] != float64(2) || c.Body["aspect_ratio"] != "2:3" || c.Body["response_format"] != "b64_json" || c.Header.Get("Authorization") != "Bearer xk" {
		t.Errorf("body = %v", c.Body)
	}
}

func TestOpenRouterRequest(t *testing.T) {
	srv, calls := fakeProvider(t, "/api/v1/images", func(_ string, w http.ResponseWriter, _ apiCall) {
		io.WriteString(w, `{"data":[{"b64_json":"`+b64(t)+`","media_type":"image/png"}]}`)
	})
	cfg := Config{Protocol: protoOpenRouter, BaseURL: srv.URL + "/api/v1", APIKey: "or", Model: "vendor/model", Size: "1024x1536"}
	pics, err := generate(context.Background(), srv.Client(), cfg, "p", testParts(t), 1)
	wantPictures(t, pics, err, 1)
	c := calls()[0]
	refs := asList(c.Body["input_references"])
	if len(refs) != 2 || asMap(refs[0])["type"] != "image_url" || asMap(asMap(refs[0])["image_url"])["url"] != "data:image/png;base64,"+b64(t) {
		t.Errorf("input_references = %v", refs)
	}
	if c.Body["model"] != "vendor/model" || c.Body["aspect_ratio"] != "2:3" || c.Body["n"] != float64(1) {
		t.Errorf("body = %v", c.Body)
	}
}

func TestDashScopeRequest(t *testing.T) {
	srv, calls := fakeProvider(t, "/api/v1"+dashScopePath, func(url string, w http.ResponseWriter, _ apiCall) {
		io.WriteString(w, `{"output":{"choices":[{"message":{"role":"assistant","content":[{"image":"`+url+`/pic.png"},{"image":"`+url+`/pic.png"}]}}]},"request_id":"r"}`)
	})
	cfg := Config{Protocol: protoDashScope, BaseURL: srv.URL + "/api/v1", APIKey: "ds", Model: "qwen-image", Size: "1024x1536"}
	pics, err := generate(context.Background(), srv.Client(), cfg, "the prompt", testParts(t), 2)
	wantPictures(t, pics, err, 2)
	c := calls()[0]
	content := asList(asMap(asList(asMap(c.Body["input"])["messages"])[0])["content"])
	if len(content) != 3 || asMap(content[0])["image"] != "data:image/png;base64,"+b64(t) || asMap(content[2])["text"] != "the prompt" {
		t.Errorf("content = %v, want the images, then the prompt", content)
	}
	params := asMap(c.Body["parameters"])
	if params["size"] != "1024*1536" || params["n"] != float64(2) || params["watermark"] != false {
		t.Errorf("parameters = %v", params)
	}
}

func TestDashScopeError(t *testing.T) {
	srv, _ := fakeProvider(t, "/api/v1"+dashScopePath, func(_ string, w http.ResponseWriter, _ apiCall) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"request_id":"r","code":"InvalidApiKey","message":"Invalid API-key provided."}`)
	})
	_, err := generate(context.Background(), srv.Client(), Config{Protocol: protoDashScope, BaseURL: srv.URL + "/api/v1", APIKey: "bad"}, "p", testParts(t), 1)
	if err == nil || !strings.Contains(err.Error(), "Invalid API-key") {
		t.Errorf("err = %v, want DashScope's message", err)
	}
}

func TestAzureRequest(t *testing.T) {
	srv, calls := fakeProvider(t, "/openai/v1/images/edits", func(_ string, w http.ResponseWriter, _ apiCall) {
		io.WriteString(w, `{"data":[{"b64_json":"`+b64(t)+`"}]}`)
	})
	cfg := Config{Protocol: protoAzure, BaseURL: srv.URL + "/openai/v1", APIKey: "az", Model: "my-deployment", Size: "1024x1536"}
	pics, err := generate(context.Background(), srv.Client(), cfg, "p", testParts(t), 1)
	wantPictures(t, pics, err, 1)
	c := calls()[0]
	if c.Query != "api-version=preview" || c.Header.Get("api-key") != "az" || c.Header.Get("Authorization") != "" {
		t.Errorf("query %q, headers %v", c.Query, c.Header)
	}
	if len(c.Files) != 2 || c.Form["model"][0] != "my-deployment" {
		t.Errorf("form = %v, files = %v", c.Form, c.Files)
	}
}

// TestRepeatReportsTheFailure fails one of three calls: the error is that call's, not the cancellation of the others.
func TestRepeatReportsTheFailure(t *testing.T) {
	var mu sync.Mutex
	n := 0
	srv, _ := fakeProvider(t, "/api/v3/images/generations", func(_ string, w http.ResponseWriter, _ apiCall) {
		mu.Lock()
		n++
		first := n == 1
		mu.Unlock()
		if first {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"error":{"code":"InputImageSensitiveContentDetected","message":"The input image may contain sensitive content."}}`)
			return
		}
		io.WriteString(w, `{"data":[{"b64_json":"`+b64(t)+`"}]}`)
	})
	_, err := generate(context.Background(), srv.Client(), Config{Protocol: protoSeedream, BaseURL: srv.URL + "/api/v3", APIKey: "k"}, "p", testParts(t), 3)
	if err == nil || !strings.Contains(err.Error(), "sensitive content") {
		t.Errorf("err = %v, want the failed call's message", err)
	}
}

func TestCheckConnectionProtocols(t *testing.T) {
	status := http.StatusBadRequest
	srv, _ := fakeProvider(t, "/api/v3/images/generations", func(_ string, w http.ResponseWriter, _ apiCall) {
		w.WriteHeader(status)
		io.WriteString(w, `{"error":{"code":"MissingParameter","message":"The request failed because it is missing the model parameter."}}`)
	})
	ark := Config{Protocol: protoSeedream, BaseURL: srv.URL + "/api/v3", APIKey: "k"}
	if err := CheckConnection(context.Background(), srv.Client(), ark); err != nil {
		t.Errorf("a valid Ark key, which gets a 400 for the empty request: %v", err)
	}
	status = http.StatusUnauthorized
	if err := CheckConnection(context.Background(), srv.Client(), ark); err == nil {
		t.Error("an Ark key the API refused passed")
	}

	models, modelCalls := fakeProvider(t, "/v1beta/models", func(_ string, w http.ResponseWriter, _ apiCall) {
		io.WriteString(w, `{"models":[]}`)
	})
	if err := CheckConnection(context.Background(), models.Client(), Config{Protocol: protoGemini, BaseURL: models.URL + "/v1beta", APIKey: "gk"}); err != nil {
		t.Fatal(err)
	}
	if c := modelCalls()[0]; c.Method != http.MethodGet || c.Header.Get("x-goog-api-key") != "gk" {
		t.Errorf("Gemini's test = %s with %v", c.Method, c.Header)
	}
	azure, azureCalls := fakeProvider(t, "/openai/v1/models", func(_ string, w http.ResponseWriter, _ apiCall) {
		io.WriteString(w, `{"data":[]}`)
	})
	if err := CheckConnection(context.Background(), azure.Client(), Config{Protocol: protoAzure, BaseURL: azure.URL + "/openai/v1", APIKey: "az"}); err != nil {
		t.Fatal(err)
	}
	if c := azureCalls()[0]; c.Header.Get("api-key") != "az" {
		t.Errorf("Azure's test headers = %v", c.Header)
	}
	// OpenRouter lists its models to anyone, so its test asks about the key.
	keyStatus := http.StatusOK
	openrouter, keyCalls := fakeProvider(t, "/api/v1/key", func(_ string, w http.ResponseWriter, _ apiCall) {
		w.WriteHeader(keyStatus)
		io.WriteString(w, `{"data":{"label":"sk-or-…"}}`)
	})
	or := Config{Protocol: protoOpenRouter, BaseURL: openrouter.URL + "/api/v1", APIKey: "ork"}
	if err := CheckConnection(context.Background(), openrouter.Client(), or); err != nil {
		t.Fatal(err)
	}
	if c := keyCalls()[0]; c.Method != http.MethodGet || c.Header.Get("Authorization") != "Bearer ork" {
		t.Errorf("OpenRouter's test = %s with %v", c.Method, c.Header)
	}
	keyStatus = http.StatusUnauthorized
	if err := CheckConnection(context.Background(), openrouter.Client(), or); err == nil {
		t.Error("an OpenRouter key the API refused passed")
	}
}

func TestAspectRatio(t *testing.T) {
	for _, c := range []struct{ size, want string }{
		{"1024x1536", "2:3"}, {"1536x1024", "3:2"}, {"1024*1024", "1:1"}, {"1080x1920", "9:16"}, {"3:4", "3:4"}, {"", "2:3"},
	} {
		if got := aspectRatio(c.size, geminiRatios); got != c.want {
			t.Errorf("aspectRatio(%q) = %q, want %q", c.size, got, c.want)
		}
	}
}

// TestSafetyRefusals reads each service's way of saying its safety system refused the request as errBlocked, and nothing else as it.
func TestSafetyRefusals(t *testing.T) {
	answer := func(status int, body string) *http.Response {
		return &http.Response{StatusCode: status, Status: http.StatusText(status), Body: io.NopCloser(strings.NewReader(body))}
	}
	for _, c := range []struct {
		name    string
		err     error
		blocked bool
	}{
		{"OpenAI", apiError(answer(400, `{"error":{"message":"Your request was rejected by the safety system.","type":"image_generation_user_error","code":"moderation_blocked"}}`)), true},
		{"Azure", apiError(answer(400, `{"error":{"code":"content_filter","message":"The response was filtered"}}`)), true},
		{"DashScope", apiError(answer(400, `{"code":"DataInspectionFailed","message":"Input data may contain inappropriate content."}`)), true},
		{"Ark", apiError(answer(400, `{"error":{"code":"InputImageSensitiveContentDetected","message":"The request failed because the input image may contain sensitive information."}}`)), true},
		{"a bad size", apiError(answer(400, `{"error":{"message":"Invalid size","code":"invalid_value"}}`)), false},
		{"a refused key", apiError(answer(401, `{"error":{"message":"Incorrect API key provided","code":"invalid_api_key"}}`)), false},
		{"a numeric code", apiError(answer(400, `{"error":{"message":"bad","code":400}}`)), false},
	} {
		if got := errors.Is(c.err, errBlocked); got != c.blocked {
			t.Errorf("%s: blocked = %v, want %v (%v)", c.name, got, c.blocked, c.err)
		}
	}

	gemini := func(body string) error {
		srv := serve(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) }))
		_, err := generate(context.Background(), srv.Client(), Config{Protocol: protoGemini, BaseURL: srv.URL, APIKey: "k", Model: "m"}, "p", nil, 1)
		return err
	}
	if err := gemini(`{"candidates":[{"finishReason":"IMAGE_SAFETY"}]}`); !errors.Is(err, errBlocked) {
		t.Errorf("Gemini's IMAGE_SAFETY: %v", err)
	}
	if err := gemini(`{"candidates":[{"finishReason":"MAX_TOKENS"}]}`); err == nil || errors.Is(err, errBlocked) {
		t.Errorf("Gemini's MAX_TOKENS: %v", err)
	}

	ark := serve(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":[{"error":{"code":"OutputImageSensitiveContentDetected","message":"The generated image may contain sensitive information."}}]}`)
	}))
	if _, err := generate(context.Background(), ark.Client(), Config{Protocol: protoSeedream, BaseURL: ark.URL, APIKey: "k", Model: "m"}, "p", nil, 1); !errors.Is(err, errBlocked) {
		t.Errorf("Ark's per-image refusal: %v", err)
	}
}

// TestPresetLogos parses every preset's mark, in color or plain as icon draws it, so a logo name the library doesn't have fails here rather than when the settings draw it.
func TestPresetLogos(t *testing.T) {
	for _, p := range providers {
		if p.Logo == "" {
			continue
		}
		_, colorErr := ailogo.Parse(p.Logo, ailogo.Color)
		if _, monoErr := ailogo.Parse(p.Logo, ailogo.Mono); colorErr != nil && monoErr != nil {
			t.Errorf("%s: %v", p.ID, monoErr)
		}
	}
	if _, err := ailogo.Parse("everyapi", ailogo.Color); err != nil {
		t.Errorf("EveryAPI has no color mark: %v", err)
	}
}

func TestEveryAPIIsOpenAICompatible(t *testing.T) {
	p := providerByID("everyapi")
	if p.ID != "everyapi" || p.Protocol != protoOpenAI || p.BaseURL != "https://api.everyapi.ai/v1" || p.Model != defaultModel {
		t.Errorf("EveryAPI preset = %+v", p)
	}
}

// TestListModels reads each provider's list of models into the image models' IDs.
func TestListModels(t *testing.T) {
	var gotPath, gotAuth string
	answer := ""
	srv := serve(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.RequestURI(), r.Header.Get("Authorization")+r.Header.Get("x-goog-api-key")
		io.WriteString(w, answer)
	}))
	for _, c := range []struct {
		name, answer, path string
		protocol           protocol
		want               []string
	}{
		{"OpenAI", `{"data":[{"id":"gpt-5"},{"id":"gpt-image-2"},{"id":"dall-e-3"},{"id":"gpt-image-2"}]}`, "/models", protoOpenAI, []string{"dall-e-3", "gpt-image-2"}},
		{"a server of no image names", `{"data":[{"id":"my-model"},{"id":"another"}]}`, "/models", protoOpenAI, []string{"another", "my-model"}},
		{"Gemini", `{"models":[{"name":"models/gemini-3-pro"},{"name":"models/gemini-3-pro-image"},{"name":"models/gemini-nano-banana-2.1"}]}`, "/models?pageSize=1000", protoGemini, []string{"gemini-3-pro-image", "gemini-nano-banana-2.1"}},
		{"xAI", `{"models":[{"id":"grok-imagine-image-2.0"}]}`, "/image-generation-models", protoXAI, []string{"grok-imagine-image-2.0"}},
		{"OpenRouter", `{"data":[{"id":"a/img","architecture":{"output_modalities":["image","text"]}},{"id":"a/txt","architecture":{"output_modalities":["text"]}}]}`, "/models?output_modalities=image", protoOpenRouter, []string{"a/img"}},
		{"Ark", `{"data":[{"id":"doubao-seed-2","domain":"LLM"},{"id":"doubao-seedream-5-5","domain":"ImageGeneration"},{"id":"x","domain":"VLM","modalities":{"output_modalities":["image"]}}]}`, "/models", protoSeedream, []string{"doubao-seedream-5-5", "x"}},
	} {
		answer = c.answer
		got, err := ListModels(context.Background(), srv.Client(), Config{Protocol: c.protocol, BaseURL: srv.URL + "/", APIKey: "k"})
		if err != nil || !slices.Equal(got, c.want) {
			t.Errorf("%s: %v, %v; want %v", c.name, got, err, c.want)
		}
		if gotPath != c.path || !strings.Contains(gotAuth, "k") {
			t.Errorf("%s asked %s with %q", c.name, gotPath, gotAuth)
		}
	}

	refused := serve(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":{"message":"bad key"}}`)
	}))
	var status *statusError
	if _, err := ListModels(context.Background(), refused.Client(), Config{BaseURL: refused.URL, APIKey: "k"}); !errors.As(err, &status) || status.Code != http.StatusUnauthorized {
		t.Errorf("a refused key: %v", err)
	}
	for _, p := range []string{"dashscope", "azure"} {
		if canListModels(providerByID(p)) {
			t.Errorf("%s's models are listed", p)
		}
	}
}
