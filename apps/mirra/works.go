package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// A work is a generated picture the user saved to the lookbook, named by the time it was made. Its sidecar, the same name with .json, keeps its number, whether it is a favorite and the references it was made from.
type work struct {
	Path    string
	Created time.Time
	workMeta
}

type workMeta struct {
	No       int   `json:"no"`
	Favorite bool  `json:"favorite,omitempty"`
	Face     *Item `json:"face,omitempty"`
	Costume  *Item `json:"costume,omitempty"`
	Pose     *Item `json:"pose,omitempty"`
	// Background is the scene the person was set in, nil for the style's own.
	Background *Item `json:"background,omitempty"`
	// Views are the angles of a sheet, the pose's every camera angle side by side.
	Views []*Item `json:"views,omitempty"`
	Style Style   `json:"style,omitempty"`
}

func metaPath(picture string) string {
	return strings.TrimSuffix(picture, filepath.Ext(picture)) + ".json"
}

func writeMeta(st pictures, w work) error {
	data, err := json.Marshal(w.workMeta)
	if err != nil {
		return err
	}
	return st.Write(metaPath(w.Path), data)
}

// writePicture puts data in dir under the time it was made.
func writePicture(st pictures, dir string, data []byte, at time.Time) (string, error) {
	path := filepath.Join(dir, fmt.Sprintf("%d%s", at.UnixNano(), imageExt(data)))
	return path, st.Write(path, data)
}

// saveWork moves a generated picture into the lookbook in dir as its next number.
func saveWork(st pictures, dir, picture string, at time.Time, meta workMeta) (work, error) {
	existing, err := listWorks(st, dir)
	if err != nil {
		return work{}, err
	}
	for _, w := range existing {
		meta.No = max(meta.No, w.No)
	}
	meta.No++
	path := filepath.Join(dir, fmt.Sprintf("%d%s", at.UnixNano(), filepath.Ext(picture)))
	if err := st.Rename(picture, path); err != nil {
		return work{}, err
	}
	w := work{Path: path, Created: at, workMeta: meta}
	if err := writeMeta(st, w); err != nil {
		return work{}, err
	}
	return w, nil
}

// deleteWork removes a work's picture and sidecar.
func deleteWork(st pictures, w work) error {
	for _, p := range []string{w.Path, metaPath(w.Path)} {
		if err := st.Remove(p); err != nil {
			return err
		}
	}
	return nil
}

// clearResults removes what a previous launch generated but never saved: only the shot on show can reach a picture in results/, and a new launch opens without one, so whatever is left there can never be reached again.
func clearResults(st pictures, dir string) {
	names, err := st.List(dir)
	if err != nil {
		return
	}
	for _, name := range names {
		_ = st.Remove(filepath.Join(dir, name))
	}
}

// listWorks returns the works in dir, newest first.
func listWorks(st pictures, dir string) ([]work, error) {
	names, err := st.List(dir)
	if err != nil {
		return nil, err
	}
	var out []work
	for _, name := range names {
		ext := filepath.Ext(name)
		nanos, err := strconv.ParseInt(strings.TrimSuffix(name, ext), 10, 64)
		if err != nil || ext == ".json" {
			continue
		}
		w := work{Path: filepath.Join(dir, name), Created: time.Unix(0, nanos)}
		if data, err := st.Read(metaPath(w.Path)); err == nil {
			_ = json.Unmarshal(data, &w.workMeta)
		}
		out = append(out, w)
	}
	slices.SortFunc(out, func(a, b work) int { return b.Created.Compare(a.Created) })
	return out, nil
}

func imageExt(data []byte) string {
	switch {
	case len(data) > 3 && string(data[:3]) == "\xff\xd8\xff":
		return ".jpg"
	case len(data) > 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return ".webp"
	default:
		return ".png"
	}
}

// workNo writes a work's number as the lookbook shows it, two digits at least.
func workNo(n int) string { return fmt.Sprintf("%02d", n) }
