package training

import (
	"path/filepath"
	"testing"
	"time"
)

func labelled(n int, unsure int) []Item {
	var items []Item
	for i := range n {
		items = append(items, Item{ID: string(rune('a' + i)), Text: "t", Relevant: true, Stance: "dafuer", LabeledBy: "claude", Unsure: i < unsure})
	}
	return items
}

func TestDrawSampleOnceAndReproducibly(t *testing.T) {
	a, b := labelled(20, 5), labelled(20, 5)
	DrawSample(a, 4, 7)
	DrawSample(b, 4, 7)
	st := Summarise(a)
	if st.Sampled != 4 || st.ToReview != 9 {
		t.Fatalf("stats = %+v", st)
	}
	for i := range a {
		if a[i].Sampled != b[i].Sampled {
			t.Fatal("sample not reproducible")
		}
		if a[i].Sampled && a[i].Unsure {
			t.Error("unsure items must not be in the sample; they are reviewed anyway")
		}
	}
	DrawSample(a, 10, 8)
	if Summarise(a).Sampled != 4 {
		t.Error("an existing sample must not be redrawn")
	}
}

func TestAgreementAndFinal(t *testing.T) {
	items := labelled(3, 0)
	for i := range items {
		items[i].Sampled = true
	}
	items[0].Review = &Review{Relevant: true, Stance: "dafuer", At: time.Now()}
	items[1].Review = &Review{Relevant: true, Stance: "dagegen", At: time.Now()}
	st := Summarise(items)
	if st.SampleReviewed != 2 || st.SampleAgreed != 1 || st.Agreement() != 0.5 {
		t.Errorf("stats = %+v", st)
	}
	if rel, s := items[1].Final(); !rel || s != "dagegen" {
		t.Error("Final must prefer the review")
	}
	if rel, s := items[2].Final(); !rel || s != "dafuer" {
		t.Error("Final must fall back to the pre-label")
	}
	if (Stats{}).Agreement() != 0 {
		t.Error("empty agreement")
	}
}

func TestReadWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "train.jsonl")
	items := labelled(2, 1)
	items = append(items, Item{ID: "unlabelled", Text: "x"})
	if err := Write(path, items); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil || len(got) != 3 || !got[0].Unsure || got[2].Labelled() {
		t.Fatalf("read back %+v, %v", got, err)
	}
	if err := Validate(true, "ja"); err == nil {
		t.Error("unknown stance accepted")
	}
	if err := Validate(false, "neutral"); err == nil {
		t.Error("stance on irrelevant accepted")
	}
	if _, err := Read(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("missing file accepted")
	}
}
