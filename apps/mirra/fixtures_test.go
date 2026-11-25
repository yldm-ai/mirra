package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"runtime"
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

// fakeServer is what a test's HTTP answers: the shape of httptest.Server — URL, Client, Close — but the handler serves the client's requests in the process rather than over a listener, which a web (js/wasm) test cannot open.
type fakeServer struct {
	URL    string
	client *http.Client
}

// Client is the client that reaches the handler.
func (f *fakeServer) Client() *http.Client { return f.client }

// Close does nothing; there is no listener to stop.
func (f *fakeServer) Close() {}

// serve is httptest.NewServer without a network: the handler answers client.Do in the process, so a test works on a target with no sockets, the web among them.
func serve(h http.Handler) *fakeServer {
	return &fakeServer{URL: "https://fake.test", client: &http.Client{Transport: handlerTransport{h}}}
}

// handlerTransport serves h for the client's requests, running the handler concurrently so it hears a request's context end while it is still open.
type handlerTransport struct{ h http.Handler }

func (t handlerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		t.h.ServeHTTP(w, req)
	}()
	select {
	case <-done:
		// A response the request's context outlasted never reaches the client, even one that finished at the same moment.
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		return w.Result(), nil
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}
}

// failTransport answers every request with a network error, as a server that is not there does; explain reads it as a network failure on any target.
type failTransport struct{}

func (failTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}
}

// testDir is the directory a test keeps the app's files in: a real temporary one on a platform with a file system, a made-up one on the web, where the store is memory and the path only names its entries.
func testDir(t testing.TB) string {
	t.Helper()
	if runtime.GOOS == "js" {
		return "/data/" + strings.ReplaceAll(t.Name(), "/", "-")
	}
	return t.TempDir()
}

// testFiles is the pictures store the platform keeps: files where it has a file system, memory on the web, matching defaultPictures.
func testFiles(t testing.TB) (pictures, string) {
	t.Helper()
	if runtime.GOOS == "js" {
		return newMemPictures(), testDir(t)
	}
	return filePictures{}, testDir(t)
}

// noFiles skips a subtest that needs the file system's store on a target that has none.
func noFiles(t testing.TB) {
	t.Helper()
	if runtime.GOOS == "js" {
		t.Skip("the browser's store is memory, which the other subtest runs")
	}
}

// newServer serves the three galleries under /pose, /costume and /face, and the image API under /v1.
func newServer(t testing.TB, api *fakeAPI) (*fakeServer, Sites) {
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
	srv := serve(mux)
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
