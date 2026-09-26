// Package corpus is the spikes' throwaway loader for Bundestag plenary
// protocols. The production parser lives in the backend module; this one only
// has to be good enough to feed the benchmarks with real data.
package corpus

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Speech is one <rede>, reduced to the paragraphs its speaker said.
type Speech struct {
	ID         string
	Period     int
	Session    int
	Date       time.Time
	SpeakerID  string
	Speaker    string
	Party      string
	Paragraphs []string
}

type redner struct {
	ID       string `xml:"id,attr"`
	Titel    string `xml:"name>titel"`
	Vorname  string `xml:"name>vorname"`
	Nachname string `xml:"name>nachname"`
	Fraktion string `xml:"name>fraktion"`
	Rolle    string `xml:"name>rolle>rolle_lang"`
}

type item struct {
	XMLName xml.Name
	Klasse  string  `xml:"klasse,attr"`
	Redner  *redner `xml:"redner"`
	Text    string  `xml:",chardata"`
}

type rede struct {
	ID    string `xml:"id,attr"`
	Items []item `xml:",any"`
}

// Parse reads one plenary protocol and returns its speeches in document order.
func Parse(r io.Reader) ([]Speech, error) {
	dec := xml.NewDecoder(r)
	var (
		out     []Speech
		period  int
		session int
		date    time.Time
	)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "dbtplenarprotokoll":
			for _, a := range se.Attr {
				switch a.Name.Local {
				case "wahlperiode":
					fmt.Sscan(a.Value, &period)
				case "sitzung-nr":
					fmt.Sscan(a.Value, &session)
				case "sitzung-datum":
					if date, err = time.Parse("02.01.2006", a.Value); err != nil {
						return nil, fmt.Errorf("sitzung-datum: %w", err)
					}
				}
			}
		case "rede":
			var rd rede
			if err := dec.DecodeElement(&rd, &se); err != nil {
				return nil, err
			}
			if s, ok := toSpeech(rd); ok {
				s.Period, s.Session, s.Date = period, session, date
				out = append(out, s)
			}
		}
	}
}

func toSpeech(rd rede) (Speech, bool) {
	s := Speech{ID: rd.ID}
	// A <name> element hands the floor to the presiding officer; the speaker
	// only has it back after the next "redner" paragraph that names them.
	speaking := false
	for _, it := range rd.Items {
		switch {
		case it.XMLName.Local == "p" && it.Klasse == "redner" && it.Redner != nil:
			if s.SpeakerID == "" {
				r := it.Redner
				s.SpeakerID = r.ID
				s.Speaker = strings.TrimSpace(strings.Join([]string{r.Titel, r.Vorname, r.Nachname}, " "))
				s.Party = r.Fraktion
				if s.Party == "" {
					s.Party = r.Rolle
				}
			}
			speaking = it.Redner.ID == s.SpeakerID
		case it.XMLName.Local == "name":
			speaking = false
		case it.XMLName.Local == "p" && speaking:
			if t := normalize(it.Text); t != "" {
				s.Paragraphs = append(s.Paragraphs, t)
			}
		}
	}
	return s, s.SpeakerID != "" && len(s.Paragraphs) > 0
}

func normalize(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// LoadDir parses every *.xml below dir, ordered by file name.
func LoadDir(dir string) ([]Speech, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*", "*.xml"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	var all []Speech
	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			return nil, err
		}
		sp, err := Parse(fh)
		fh.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
		all = append(all, sp...)
	}
	return all, nil
}
