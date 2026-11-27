package main

import (
	"errors"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// eachStore runs a test against the files the native apps keep and the memory a browser keeps.
func eachStore(t *testing.T, test func(t *testing.T, st pictures, root string)) {
	t.Run("files", func(t *testing.T) { test(t, filePictures{}, t.TempDir()) })
	t.Run("memory", func(t *testing.T) { test(t, newMemPictures(), "/data") })
}

// errFull is what a store says when it has no room for a sidecar.
var errFull = errors.New("disk full")

// fullDisk keeps the pictures writable but refuses every sidecar's write, as a full disk would.
type fullDisk struct{ pictures }

func (p fullDisk) Write(path string, data []byte) error {
	if strings.HasSuffix(path, ".json") {
		return errFull
	}
	return p.pictures.Write(path, data)
}

// TestASidecarThatCannotBeWrittenLeavesThePictureWhereItWas saves a picture whose sidecar cannot be written: the save fails and the picture goes back where it was, so a retry can move it once there is room again.
func TestASidecarThatCannotBeWrittenLeavesThePictureWhereItWas(t *testing.T) {
	eachStore(t, func(t *testing.T, st pictures, root string) {
		full := fullDisk{st}
		t0 := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
		pic, err := writePicture(full, filepath.Join(root, "results"), tinyPNG(t), t0)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := saveWork(full, filepath.Join(root, "works"), pic, t0.Add(time.Minute), workMeta{}); err == nil {
			t.Fatal("a save whose sidecar cannot be written succeeded")
		}
		if left, err := st.List(filepath.Join(root, "results")); err != nil || len(left) != 1 {
			t.Errorf("the results hold %v (err %v), want the picture back", left, err)
		}
		if ws, err := listWorks(st, filepath.Join(root, "works")); err != nil || len(ws) != 0 {
			t.Errorf("the lookbook holds %+v (err %v), want nothing", ws, err)
		}

		// With the sidecar writable again, the same picture saves.
		if _, err := saveWork(st, filepath.Join(root, "works"), pic, t0.Add(2*time.Minute), workMeta{}); err != nil {
			t.Fatal(err)
		}
		if ws, _ := listWorks(st, filepath.Join(root, "works")); len(ws) != 1 {
			t.Errorf("the lookbook holds %+v, want the picture", ws)
		}
	})
}

func TestWorksAreNumberedAndKeepTheirLook(t *testing.T) {
	eachStore(t, func(t *testing.T, st pictures, root string) {
		dir := filepath.Join(root, "works")
		if ws, err := listWorks(st, dir); err != nil || len(ws) != 0 {
			t.Fatalf("a missing directory: %v, %v", ws, err)
		}
		t0 := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
		pending := func(name string, data []byte) string {
			p, err := writePicture(st, filepath.Join(root, "results"), data, t0.Add(time.Duration(len(name))))
			if err != nil {
				t.Fatal(err)
			}
			return p
		}
		coat := &Item{Kind: KindCostume, ID: "WIN-0001", Group: "冬", Title: "camel wool coat"}
		first, err := saveWork(st, dir, pending("a", tinyPNG(t)), t0, workMeta{Costume: coat})
		if err != nil {
			t.Fatal(err)
		}
		second, err := saveWork(st, dir, pending("bb", []byte("\xff\xd8\xff\xe0jpeg")), t0.Add(time.Minute), workMeta{Favorite: true})
		if err != nil {
			t.Fatal(err)
		}
		if first.No != 1 || second.No != 2 || filepath.Ext(second.Path) != ".jpg" {
			t.Errorf("numbers = %d, %d; second = %s", first.No, second.No, second.Path)
		}
		if err := st.Write(filepath.Join(dir, ".DS_Store"), nil); err != nil {
			t.Fatal(err)
		}
		ws, err := listWorks(st, dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(ws) != 2 || ws[0].Path != second.Path || !ws[0].Favorite || ws[1].Costume == nil || *ws[1].Costume != *coat {
			t.Fatalf("works = %+v", ws)
		}
		if left, _ := st.List(filepath.Join(root, "results")); len(left) != 0 {
			t.Errorf("saving left %v among the results", left)
		}

		ws[1].Favorite = true
		if err := writeMeta(st, ws[1]); err != nil {
			t.Fatal(err)
		}
		if err := deleteWork(st, ws[0]); err != nil {
			t.Fatal(err)
		}
		ws, _ = listWorks(st, dir)
		if len(ws) != 1 || ws[0].No != 1 || !ws[0].Favorite {
			t.Errorf("after starring the first and deleting the second: %+v", ws)
		}
		if _, err := st.Read(metaPath(second.Path)); err == nil {
			t.Error("the deleted work's sidecar is still there")
		}
	})
}

func TestUploadsAreNumberedOldestFirst(t *testing.T) {
	eachStore(t, func(t *testing.T, st pictures, root string) {
		dir := filepath.Join(root, "uploads")
		t0 := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
		for _, at := range []time.Time{t0.Add(time.Hour), t0} {
			if _, err := saveUpload(st, dir, tinyPNG(t), at); err != nil {
				t.Fatal(err)
			}
		}
		ups, err := listUploads(st, dir)
		if err != nil || len(ups) != 2 {
			t.Fatalf("uploads = %v, %v", ups, err)
		}
		if ups[0].ID != "MY-01" || ups[1].ID != "MY-02" || ups[0].Group != uploadGroup || !filepath.IsAbs(ups[0].Full) {
			t.Errorf("uploads = %+v", ups)
		}
		if got, want := filepath.Base(ups[0].Full), strconv.FormatInt(t0.UnixNano(), 10)+".png"; got != want {
			t.Errorf("oldest upload = %s, want %s", got, want)
		}
	})
}

func TestStores(t *testing.T) {
	eachStore(t, func(t *testing.T, st pictures, root string) {
		a, nested := filepath.Join(root, "d", "a.png"), filepath.Join(root, "d", "sub", "b.png")
		for _, p := range []string{a, nested} {
			if err := st.Write(p, []byte(p)); err != nil {
				t.Fatal(err)
			}
		}
		if names, _ := st.List(filepath.Join(root, "d")); !slices.Equal(names, []string{"a.png"}) {
			t.Errorf("List = %v, want the dir's own files only", names)
		}
		moved := filepath.Join(root, "e", "a.png")
		if err := st.Rename(a, moved); err != nil {
			t.Fatal(err)
		}
		if data, err := st.Read(moved); err != nil || string(data) != a {
			t.Errorf("after Rename: %q, %v", data, err)
		}
		if _, err := st.Read(a); err == nil {
			t.Error("Rename left the old path")
		}
		if err := st.Remove(filepath.Join(root, "missing.png")); err != nil {
			t.Errorf("removing a missing path: %v", err)
		}
		if names, err := st.List(filepath.Join(root, "never")); err != nil || len(names) != 0 {
			t.Errorf("a dir never written: %v, %v", names, err)
		}
	})
	if (filePictures{}).Lasting() == newMemPictures().Lasting() {
		t.Error("files and memory both say they last the same")
	}
}
