// Package bundestag reads the German Bundestag's open data: plenary protocol
// XML (from the 19th legislative period on) and the MdB master data.
package bundestag

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Protocol is one plenary session.
type Protocol struct {
	Period   int
	Session  int
	Date     time.Time
	Speeches []Speech
}

// Speech is one <rede>, reduced to what its speaker said.
type Speech struct {
	ID      string
	Speaker Speaker
	// Page is the printed page the speech starts on.
	Page       int
	Paragraphs []Paragraph
}

// Speaker is the person holding the speech, as the protocol names them.
type Speaker struct {
	// ID is the MdB master-data ID; members of the government who are not
	// MdBs have IDs outside that range.
	ID      string
	Title   string
	First   string
	Last    string
	Suffix  string
	Faction string
	Role    string
}

// Name is the speaker's display name.
func (s Speaker) Name() string {
	parts := []string{s.Title, s.First, s.Last, s.Suffix}
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " ")
}

// Paragraph is one paragraph the speaker said.
type Paragraph struct {
	Text string
	// Quote marks a quotation block (klasse "Z"): words the speaker reads out,
	// not necessarily their own position.
	Quote bool
	Page  int
}

type redner struct {
	ID     string `xml:"id,attr"`
	Titel  string `xml:"name>titel"`
	Vor    string `xml:"name>vorname"`
	Nach   string `xml:"name>nachname"`
	Ort    string `xml:"name>ortszusatz"`
	Zusatz string `xml:"name>namenszusatz"`
	Frak   string `xml:"name>fraktion"`
	Rolle  string `xml:"name>rolle>rolle_lang"`
}

// speechClasses are the paragraph classes that carry spoken text inside a
// <rede>. Headings (T_*), voting lists (AL_*) and similar layout classes are
// not speech.
var speechClasses = map[string]bool{"J": true, "J_1": true, "O": true, "Z": true, "p": true, "": true}

// ParseProtocol reads one plenary protocol in document order.
//
// Interjections (<kommentar>), footnotes and everything the presiding officer
// says in between are left out: after a <name> element the floor belongs to
// someone else until the next "redner" paragraph names the speaker again.
func ParseProtocol(r io.Reader) (Protocol, error) {
	var (
		p        Protocol
		dec      = xml.NewDecoder(r)
		page     int
		cur      *Speech
		speaking bool
		// Paragraphs can nest; only the outermost one is collected.
		pDepth   int
		pClass   string
		pRedner  bool
		pText    strings.Builder
		pPage    int
		sawProto bool
	)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return p, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "dbtplenarprotokoll":
				sawProto = true
				if err := protocolAttrs(&p, t.Attr); err != nil {
					return p, err
				}
			case "a":
				if n, ok := pageAnchor(t.Attr); ok {
					page = n
				}
			case "rede":
				cur = &Speech{ID: attr(t.Attr, "id"), Page: page}
				speaking = false
			case "redner":
				if cur == nil {
					if err := dec.Skip(); err != nil {
						return p, err
					}
					continue
				}
				var rd redner
				if err := dec.DecodeElement(&rd, &t); err != nil {
					return p, err
				}
				if cur.Speaker.ID == "" {
					cur.Speaker = Speaker{
						ID: rd.ID, Title: trim(rd.Titel), First: trim(rd.Vor), Last: trim(rd.Nach),
						Suffix: trim(rd.Zusatz), Faction: trim(rd.Frak), Role: trim(rd.Rolle),
					}
				}
				speaking = rd.ID == cur.Speaker.ID
			case "name":
				if cur != nil && pDepth == 0 {
					speaking = false
					if err := dec.Skip(); err != nil {
						return p, err
					}
				}
			case "kommentar", "fussnote":
				if err := dec.Skip(); err != nil {
					return p, err
				}
			case "p":
				if cur == nil {
					continue
				}
				pDepth++
				if pDepth == 1 {
					pClass = attr(t.Attr, "klasse")
					pRedner = pClass == "redner"
					pText.Reset()
					pPage = page
				}
			}
		case xml.CharData:
			if pDepth > 0 && !pRedner {
				pText.Write(t)
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "p":
				if pDepth == 0 {
					continue
				}
				pDepth--
				if pDepth > 0 || pRedner || !speaking || !speechClasses[pClass] {
					continue
				}
				if text := normalize(pText.String()); text != "" {
					cur.Paragraphs = append(cur.Paragraphs, Paragraph{Text: text, Quote: pClass == "Z", Page: pPage})
				}
			case "rede":
				if cur != nil && cur.Speaker.ID != "" && len(cur.Paragraphs) > 0 {
					p.Speeches = append(p.Speeches, *cur)
				}
				cur = nil
			}
		}
	}
	if !sawProto {
		return p, fmt.Errorf("not a plenary protocol: no <dbtplenarprotokoll> element")
	}
	return p, nil
}

func protocolAttrs(p *Protocol, attrs []xml.Attr) error {
	var err error
	for _, a := range attrs {
		switch a.Name.Local {
		case "wahlperiode":
			if p.Period, err = strconv.Atoi(a.Value); err != nil {
				return fmt.Errorf("wahlperiode %q: %w", a.Value, err)
			}
		case "sitzung-nr":
			if p.Session, err = strconv.Atoi(a.Value); err != nil {
				return fmt.Errorf("sitzung-nr %q: %w", a.Value, err)
			}
		case "sitzung-datum":
			if p.Date, err = time.Parse("02.01.2006", a.Value); err != nil {
				return fmt.Errorf("sitzung-datum %q: %w", a.Value, err)
			}
		}
	}
	if p.Period == 0 || p.Session == 0 || p.Date.IsZero() {
		return fmt.Errorf("protocol header incomplete: period %d, session %d, date %v", p.Period, p.Session, p.Date)
	}
	return nil
}

// pageAnchor reads <a typ="druckseitennummer" name="S12074"> (some files use
// href instead of name).
func pageAnchor(attrs []xml.Attr) (int, bool) {
	if attr(attrs, "typ") != "druckseitennummer" {
		return 0, false
	}
	v := attr(attrs, "name")
	if v == "" {
		v = attr(attrs, "href")
	}
	n, err := strconv.Atoi(strings.TrimPrefix(v, "S"))
	return n, err == nil
}

func attr(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func trim(s string) string { return strings.TrimSpace(s) }

func normalize(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
