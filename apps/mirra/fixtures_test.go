package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// The fixtures copy the galleries' formats with made-up rows; no real gallery data is checked in.

const posePage = `<!doctype html><html><body><script>
const META={"lines": ["全身1人・標準系", "2人絡み"]};
const D=[{"id": "full_sexy_P000_front", "系統": "全身1人・セクシー系", "ポーズ番号": "P000", "分類": "立ち", "ポーズ名": "S字立ち", "アングル": "正面", "サムネ": "thumbs/s.jpg", "原寸": "img/s.png"},
{"id": "full_std_P001_front", "系統": "全身1人・標準系", "ポーズ番号": "P001", "分類": "立ち", "ポーズ名": "片手を腰に当てて立つ", "アングル": "正面", "サムネ": "thumbs/full_std_P001_front.jpg", "原寸": "img/full_std_P001_front.png"},
{"id": "full_std_P001_side", "系統": "全身1人・標準系", "ポーズ番号": "P001", "分類": "立ち", "ポーズ名": "片手を腰に当てて立つ", "アングル": "横", "サムネ": "thumbs/full_std_P001_side.jpg", "原寸": "img/full_std_P001_side.png"},
{"id": "upper_P002_side", "系統": "上半身1人", "ポーズ番号": "P002", "分類": "ひねり", "ポーズ名": "振り返る", "アングル": "横", "サムネ": "thumbs/upper_P002_side.jpg", "原寸": "img/upper_P002_side.png"},
{"id": "pair_P003_front", "系統": "2人絡み", "ポーズ番号": "P003", "分類": "密着", "ポーズ名": "x", "アングル": "正面", "サムネ": "thumbs/pair.jpg", "原寸": "img/pair.png"},
{"id": "full_sexy_P004_front", "系統": "全身1人・セクシー系", "ポーズ番号": "P004", "分類": "セクシー・露骨寄り系", "ポーズ名": "x", "アングル": "正面", "サムネ": "thumbs/x.jpg", "原寸": "img/x.png"}];
const ANG=["正面"];
</script></body></html>`

const costumePage = `<html><script>
const META={"total": 3};
const D=[{"id": "SPR-0001", "カテゴリ": "春", "キーワード": "ivory knit denim skirt", "サムネ": "thumbs/SPR-0001.webp", "画像": "img/SPR-0001.webp", "原寸": "png/SPR-0001.png"},
{"id": "LNG-0001", "カテゴリ": "ランジェリー", "キーワード": "x", "サムネ": "thumbs/LNG-0001.webp", "画像": "img/LNG-0001.webp", "原寸": "png/LNG-0001.png"},
{"id": "WIN-0001", "カテゴリ": "冬", "キーワード": "camel wool coat", "サムネ": "thumbs/WIN-0001.webp", "画像": "img/WIN-0001.webp", "原寸": "png/WIN-0001.png"}];
</script></html>`

const faceScript = `window.GALLERY_IMAGES = [
  {"id": "f002", "section": "face", "hair": "ロング", "style": "ゆる巻き", "w": 1122, "h": 1402, "full": "img/full/f002.png", "thumb": "img/thumb/f002.webp"},
  {"id": "f001", "section": "face", "hair": "ショート", "style": "", "w": 1122, "h": 1402, "full": "img/full/f001.png", "thumb": "img/thumb/f001.webp"},
  {"id": "s001", "section": "styling", "hair": "ロング", "style": "ゆる巻き", "w": 1122, "h": 1402, "full": "img/full/s001.png", "thumb": "img/thumb/s001.webp"}
];
`

func tinyPNG(t testing.TB) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// fakeAPI records the image edit requests it answers.
type fakeAPI struct {
	mu       sync.Mutex
	requests []editRequest
	status   int
	errBody  string
	// hang keeps each request open until the client gives up.
	hang bool
}

type editRequest struct {
	Auth, Model, Prompt, Size, N string
	Images                       []string // file names, in order
	Downloads                    []string // gallery paths fetched before this request
}

func (f *fakeAPI) last() editRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.requests) == 0 {
		return editRequest{}
	}
	return f.requests[len(f.requests)-1]
}

// count reads the recorded request count under the same mutex the handler records with: tests poll it while the fake server answers, and an unguarded read races that append.
func (f *fakeAPI) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

// newServer serves the three galleries under /pose, /costume and /face, and the image API under /v1.
func newServer(t testing.TB, api *fakeAPI) (*httptest.Server, Sites) {
	t.Helper()
	pic := tinyPNG(t)
	var mu sync.Mutex
	var downloads []string
	mux := http.NewServeMux()
	mux.HandleFunc("/pose/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pose/" {
			io.WriteString(w, posePage)
			return
		}
		mu.Lock()
		downloads = append(downloads, r.URL.Path)
		mu.Unlock()
		w.Write(pic)
	})
	mux.HandleFunc("/costume/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/costume/" {
			io.WriteString(w, costumePage)
			return
		}
		mu.Lock()
		downloads = append(downloads, r.URL.Path)
		mu.Unlock()
		w.Write(pic)
	})
	mux.HandleFunc("/face/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/face/data/images.js" {
			io.WriteString(w, faceScript)
			return
		}
		mu.Lock()
		downloads = append(downloads, r.URL.Path)
		mu.Unlock()
		w.Write(pic)
	})
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer bad" {
			w.WriteHeader(http.StatusUnauthorized)
			io.WriteString(w, `{"error":{"message":"Incorrect API key provided."}}`)
			return
		}
		io.WriteString(w, `{"data":[{"id":"test-model"},{"id":"gpt-image-2"},{"id":"gpt-5"}]}`)
	})
	mux.HandleFunc("/v1/images/edits", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req := editRequest{Auth: r.Header.Get("Authorization"), Model: r.FormValue("model"), Prompt: r.FormValue("prompt"), Size: r.FormValue("size"), N: r.FormValue("n")}
		for _, fh := range r.MultipartForm.File["image[]"] {
			req.Images = append(req.Images, fh.Filename)
		}
		mu.Lock()
		req.Downloads = append([]string(nil), downloads...)
		mu.Unlock()
		api.mu.Lock()
		api.requests = append(api.requests, req)
		status, errBody, hang := api.status, api.errBody, api.hang
		api.mu.Unlock()
		if hang {
			<-r.Context().Done()
			return
		}
		if status != 0 {
			w.WriteHeader(status)
			io.WriteString(w, errBody)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		n, _ := strconv.Atoi(req.N)
		var data []string
		for range max(n, 1) {
			data = append(data, `{"b64_json":"`+base64.StdEncoding.EncodeToString(pic)+`"}`)
		}
		io.WriteString(w, `{"data":[`+strings.Join(data, ",")+`]}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, Sites{Pose: srv.URL + "/pose", Costume: srv.URL + "/costume", Face: srv.URL + "/face"}
}

func hasAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
