// Package demo fills a database with fictional speeches for development and
// the end-to-end tests. All people and statements are invented; the IDs lie
// outside the Bundestag's range so they can never be mistaken for real MdBs.
package demo

import (
	"context"
	"fmt"
	"time"

	"github.com/praetorianer777/kurswechsel/internal/bundestag"
	"github.com/praetorianer777/kurswechsel/internal/classify"
	"github.com/praetorianer777/kurswechsel/internal/stance"
	"github.com/praetorianer777/kurswechsel/internal/store"
	"github.com/praetorianer777/kurswechsel/internal/topic"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

// People are the fictional speakers.
var People = []bundestag.Politician{
	{
		ID: "90000001", Title: "Dr.", First: "Erika", Last: "Musterfrau", Party: "Beispielpartei", Gender: "weiblich",
		Memberships: []bundestag.Membership{
			{Period: 19, Faction: "Fraktion A", From: day(2017, 10, 24), To: day(2021, 10, 26)},
			{Period: 20, Faction: "Fraktion A", From: day(2021, 10, 26), To: day(2025, 3, 25)},
		},
	},
	{
		ID: "90000002", First: "Max", Last: "Müstermann", Party: "Musterpartei", Gender: "männlich",
		Memberships: []bundestag.Membership{{Period: 20, Faction: "Fraktion B", From: day(2021, 10, 26), To: day(2025, 3, 25)}},
	},
}

type speech struct {
	date       time.Time
	period     int
	session    int
	speaker    int
	paragraphs []string
}

var speeches = []speech{
	{day(2018, 9, 27), 19, 51, 0, []string{
		"Wir müssen die Wehrpflicht wieder einführen, denn die Bundeswehr findet keinen Nachwuchs.",
		"Außerdem brauchen wir mehr Geld für die Ausrüstung.",
	}},
	{day(2019, 11, 28), 19, 130, 1, []string{
		"Der Wehrdienst muss freiwillig bleiben, niemand soll gezwungen werden.",
	}},
	{day(2020, 11, 20), 19, 192, 0, []string{
		"Ein verpflichtendes Gesellschaftsjahr für alle jungen Menschen wäre ein Gewinn.",
	}},
	{day(2022, 6, 2), 20, 40, 0, []string{
		"Heute beraten wir den Bericht zur Musterung in erster Lesung.",
	}},
	{day(2023, 3, 16), 20, 92, 0, []string{
		"Der Wehrdienst muss freiwillig bleiben; wir setzen auf gute Bedingungen statt auf Zwang.",
		"Glauben Sie wirklich, dass eine Musterung allein das Personalproblem löst?",
	}},
	{day(2024, 10, 17), 20, 193, 1, []string{
		"Wir sollten die Wehrpflicht wieder einführen, die Lage hat sich grundlegend geändert.",
	}},
}

// Seed stores the fictional data and classifies it with the fake classifier.
func Seed(ctx context.Context, st *store.Store) error {
	if err := st.UpsertPoliticians(ctx, People); err != nil {
		return err
	}
	for i, sp := range speeches {
		p := People[sp.speaker]
		faction := p.FactionOn(sp.date)
		paras := make([]bundestag.Paragraph, len(sp.paragraphs))
		for j, text := range sp.paragraphs {
			paras[j] = bundestag.Paragraph{Text: text, Page: 1000 + 10*i}
		}
		proto := bundestag.Protocol{
			Period: sp.period, Session: sp.session, Date: sp.date,
			Speeches: []bundestag.Speech{{
				ID:         fmt.Sprintf("DEMO%02d", i+1),
				Page:       1000 + 10*i,
				Speaker:    bundestag.Speaker{ID: p.ID, Title: p.Title, First: p.First, Last: p.Last, Faction: faction},
				Paragraphs: paras,
			}},
		}
		url := fmt.Sprintf("%s/%d/%d%03d.xml", bundestag.DefaultProtocolBase, sp.period, sp.period, sp.session)
		if err := st.SaveProtocol(ctx, proto, url, time.Now()); err != nil {
			return err
		}
	}
	_, err := classify.Run(ctx, st, stance.Fake{}, topic.Wehrpflicht, classify.Options{})
	return err
}
