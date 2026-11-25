package main

import (
	"context"
	"slices"
	"strings"
	"testing"
)

func ids(items []Item) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.ID)
	}
	return out
}

func TestFetchCatalogs(t *testing.T) {
	srv, sites := newServer(t, &fakeAPI{})
	ctx := context.Background()

	poses, err := FetchCatalog(ctx, srv.Client(), sites, KindPose)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ids(poses.Items), []string{"full_std_P001_front", "full_std_P001_side", "upper_P002_side", "full_sexy_P000_front", "full_sexy_P004_front", "pair_P003_front"}; !slices.Equal(got, want) {
		t.Errorf("pose ids = %v, want %v", got, want)
	}
	if want := []string{"正面", "横"}; !slices.Equal(poses.Groups, want) {
		t.Errorf("pose groups = %v, want the angles %v", poses.Groups, want)
	}
	if got := ids(poses.InGroup("正面", false)); !slices.Equal(got, []string{"full_std_P001_front"}) {
		t.Errorf("front poses for everyone = %v", got)
	}
	if got := ids(poses.InGroup("正面", true)); !slices.Equal(got, []string{"full_sexy_P000_front", "full_sexy_P004_front", "pair_P003_front"}) {
		t.Errorf("adults-only front poses = %v", got)
	}
	if want := []string{"正面"}; !slices.Equal(poses.AdultGroups, want) {
		t.Errorf("adults-only pose groups = %v, want %v", poses.AdultGroups, want)
	}
	first := poses.Items[0]
	if first.Full != sites.Pose+"/img/full_std_P001_front.png" || first.Thumb != sites.Pose+"/thumbs/full_std_P001_front.jpg" || first.Title != "片手を腰に当てて立つ" {
		t.Errorf("first pose = %+v", first)
	}

	costumes, err := FetchCatalog(ctx, srv.Client(), sites, KindCostume)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ids(costumes.Items), []string{"SPR-0001", "LNG-0001", "WIN-0001"}; !slices.Equal(got, want) {
		t.Errorf("costumes = %v, want %v", got, want)
	}
	if want := []string{"春", "冬"}; !slices.Equal(costumes.Groups, want) {
		t.Errorf("costume groups = %v, want %v", costumes.Groups, want)
	}
	if got := ids(costumes.InGroup("", false)); !slices.Equal(got, []string{"SPR-0001", "WIN-0001"}) {
		t.Errorf("costumes for everyone = %v", got)
	}
	if got := ids(costumes.InGroup("ランジェリー", true)); !slices.Equal(got, []string{"LNG-0001"}) || !slices.Equal(costumes.AdultGroups, []string{"ランジェリー"}) {
		t.Errorf("adults-only costumes = %v in %v", got, costumes.AdultGroups)
	}
	if got := costumes.InGroup("冬", false); len(got) != 1 || got[0].ID != "WIN-0001" || got[0].Full != sites.Costume+"/png/WIN-0001.png" {
		t.Errorf("winter costumes = %+v", got)
	}

	faces, err := FetchCatalog(ctx, srv.Client(), sites, KindFace)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(faces.Items); !slices.Equal(got, []string{"f002", "f001"}) {
		t.Errorf("faces = %v, want the face section only", got)
	}
	if want := []string{"ショート", "ロング"}; !slices.Equal(faces.Groups, want) {
		t.Errorf("face groups = %v, want hair lengths shortest first %v", faces.Groups, want)
	}
}

func TestParseRejectsChangedFormat(t *testing.T) {
	for name, parse := range map[string]func([]byte, string) (Catalog, error){
		"pose": parsePoses, "costume": parseCostumes, "face": parseFaces,
	} {
		if _, err := parse([]byte("<html>redesigned</html>"), "https://x"); err == nil || !strings.Contains(err.Error(), "format changed") {
			t.Errorf("%s: err = %v, want a format error", name, err)
		}
	}
	if got, err := parsePoses([]byte(`const D=[{"id":"a","系統":"2人絡み","原寸":"a.png"}];`), "https://x"); err != nil || len(got.Items) != 1 || got.Items[0].People != 2 {
		t.Errorf("a page with a pose from any line should parse: catalog=%+v err=%v", got, err)
	}
}

func TestAdultPose(t *testing.T) {
	for _, c := range []struct {
		line, category string
		adult          bool
	}{
		{"全身1人・標準系", "立ち", false},
		{"上半身1人", "かわいい系", false},
		{"上半身1人", "セクシー・露骨寄り系", true},
		{"上半身1人", "胸寄せ・谷間強調系", true},
		{"全身1人・セクシー系", "座り", true},
		{"2人絡み", "密着", true},
	} {
		if got := adultPose(c.line, c.category); got != c.adult {
			t.Errorf("adultPose(%q, %q) = %v, want %v", c.line, c.category, got, c.adult)
		}
	}
}

func TestPoseAngles(t *testing.T) {
	srv, sites := newServer(t, &fakeAPI{})
	poses, err := FetchCatalog(context.Background(), srv.Client(), sites, KindPose)
	if err != nil {
		t.Fatal(err)
	}
	if got := ids(poses.Angles(poses.Items[0])); !slices.Equal(got, []string{"full_std_P001_front", "full_std_P001_side"}) {
		t.Errorf("angles of P001 = %v", got)
	}
	if got := poses.Angles(Item{ID: "an outfit"}); got != nil {
		t.Errorf("an item of no set has angles %v", got)
	}
	// Angles come in the chips' order, whatever the gallery's.
	c := Catalog{Items: []Item{{ID: "low", Set: "a", Group: "下から"}, {ID: "b", Set: "b", Group: "正面"}, {ID: "front", Set: "a", Group: "正面"}, {ID: "side", Set: "a", Group: "横"}}}
	if got := ids(c.Angles(c.Items[0])); !slices.Equal(got, []string{"front", "side", "low"}) {
		t.Errorf("angles = %v", got)
	}
	if poseSet("全身1人・標準系", "") != "" {
		t.Error("a pose with no number is in a set")
	}
}
