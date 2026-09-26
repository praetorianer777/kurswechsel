package stance

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/praetorianer777/kurswechsel/internal/topic"
)

func lp(token string, p float64) logprob { return logprob{Token: token, Logprob: math.Log(p)} }

func TestLetterProbs(t *testing.T) {
	got, err := letterProbs([]logprob{lp("B", 0.6), lp(" b", 0.1), lp("C", 0.2), lp("Die", 0.1)})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got["B"]-0.7/0.9) > 1e-9 || math.Abs(got["C"]-0.2/0.9) > 1e-9 {
		t.Errorf("probs = %v", got)
	}
	if _, err := letterProbs([]logprob{lp("Die", 1)}); err == nil {
		t.Error("want error without any letter")
	}
}

func TestProbsAnswer(t *testing.T) {
	p := "Einleitung. Die Wehrpflicht muss zurück."
	a := probsAnswer(map[string]float64{"B": 0.8, "C": 0.2}, 0, p, topic.Wehrpflicht)
	if !a.Relevant || a.Stance != For || a.Confidence != 0.8 || a.Quote != "Die Wehrpflicht muss zurück." || !strings.Contains(a.Rationale, "80 %") {
		t.Errorf("answer = %+v", a)
	}
	if a := probsAnswer(map[string]float64{"C": 0.55, "B": 0.45}, 0.7, p, topic.Wehrpflicht); a.Stance != Unclear {
		t.Errorf("uncertain side = %+v, want unklar", a)
	}
	if a := probsAnswer(map[string]float64{"D": 0.5, "B": 0.4}, 0.7, p, topic.Wehrpflicht); a.Stance != Neutral {
		t.Errorf("threshold must only affect sides: %+v", a)
	}
	if a := probsAnswer(map[string]float64{"A": 0.9}, 0, p, topic.Wehrpflicht); a.Relevant || a.Quote != "" {
		t.Errorf("irrelevant = %+v", a)
	}
}

func TestOllamaProbsStyle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		if req["logprobs"] != true || req["format"] != nil || req["options"].(map[string]any)["num_predict"] != float64(1) {
			t.Errorf("request = %v", req)
		}
		w.Write([]byte(`{"message":{"content":"C"},"logprobs":[{"token":"C","logprob":-0.1,"top_logprobs":[{"token":"C","logprob":-0.1},{"token":"B","logprob":-2.5}]}]}`))
	}))
	defer srv.Close()
	o := &Ollama{BaseURL: srv.URL, Model: "m", Style: StyleProbs, MinConfidence: 0.5}
	a, err := o.Classify(context.Background(), topic.Wehrpflicht, "Die Wehrpflicht bleibt ausgesetzt.")
	if err != nil || a.Stance != Against || a.Confidence < 0.9 {
		t.Errorf("answer = %+v, %v", a, err)
	}
	if o.Name() != "ollama/m+probs@0.50" || o.Prompt() != ProbsPromptVersion || (&Ollama{Model: "m", Style: StyleProbs}).Name() != "ollama/m+probs" {
		t.Error("name or prompt version")
	}
}

func TestOllamaProbsWithoutLogprobs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"message":{"content":"C"}}`))
	}))
	defer srv.Close()
	if _, err := (&Ollama{BaseURL: srv.URL, Style: StyleProbs}).Classify(context.Background(), topic.Wehrpflicht, "x"); err == nil {
		t.Error("want error when the server sends no logprobs")
	}
}
