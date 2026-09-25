export const de = {
  appName: "Kurswechsel",
  tagline:
    "Was Abgeordnete im Bundestag zu einem Thema gesagt haben – chronologisch, mit Originalzitat und Quelle.",
  skipLink: "Zum Inhalt springen",
  nav: {
    label: "Hauptnavigation",
    home: "Start",
    methodology: "Methodik",
  },
  footer: {
    label: "Fußzeile",
    source:
      "Quelle: Plenarprotokolle und Stammdaten des Deutschen Bundestages. Amtliche Werke sind nach § 5 UrhG gemeinfrei.",
    imprint: "Impressum",
    privacy: "Datenschutz",
    code: "Quellcode",
  },
  loading: "Wird geladen …",
  errors: {
    not_found: "Diese Seite gibt es nicht.",
    unknown_topic: "Dieses Thema gibt es nicht.",
    unknown_politician: "Diese Person ist nicht in der Datenbank.",
    unknown_entry: "Diesen Eintrag gibt es nicht mehr.",
    invalid_json: "Die Anfrage war fehlerhaft.",
    message_too_short:
      "Bitte beschreiben Sie den Fehler mit mindestens 10 Zeichen.",
    message_too_long: "Bitte fassen Sie sich kürzer (höchstens 2000 Zeichen).",
    contact_too_long: "Die Kontaktangabe ist zu lang (höchstens 200 Zeichen).",
    too_many_reports:
      "Sie haben gerade mehrere Meldungen geschickt. Bitte versuchen Sie es in einigen Minuten erneut.",
    internal:
      "Auf dem Server ist ein Fehler aufgetreten. Bitte versuchen Sie es später erneut.",
    network:
      "Keine Verbindung zum Server. Bitte prüfen Sie Ihre Internetverbindung.",
  },
  home: {
    title: "Positionen im Zeitverlauf",
    intro:
      "Kurswechsel zeigt, was Abgeordnete des Bundestages seit 2017 zu einem Thema gesagt haben: jede Aussage mit Datum, Originalzitat und Link zum Plenarprotokoll. Ob sich eine Position geändert hat, beurteilen Sie selbst.",
    searchHeading: "Person suchen",
    searchLabel: "Name",
    searchHint:
      "Zum Beispiel „Pistorius“ oder „Müller“. Umlaute dürfen ausgeschrieben werden.",
    topicLabel: "Thema",
    allTopics: "Alle Themen",
    results: (n: number) =>
      n === 0 ? "Keine Treffer." : n === 1 ? "1 Treffer" : `${n} Treffer`,
    statements: (n: number) => (n === 1 ? "1 Aussage" : `${n} Aussagen`),
    speeches: (n: number) => (n === 1 ? "1 Rede" : `${n} Reden`),
    topicsHeading: "Themen",
    topicStats: (people: number, entries: number) =>
      `${entries} Aussagen von ${people} Personen`,
    howHeading: "So funktioniert es",
    how: [
      "Alle Reden im Bundestag seit 2017 werden aus den amtlichen Plenarprotokollen übernommen.",
      "Absätze zu einem Thema werden per Stichwort gefunden und von einem Sprachmodell eingeordnet: dafür, dagegen, neutral oder unklar.",
      "Ändert sich die Einordnung zwischen zwei Sitzungstagen, wird das als Positionswechsel markiert – ohne Wertung.",
    ],
  },
  person: {
    party: "Partei",
    factions: "Fraktionszugehörigkeit",
    period: (p: number) => `${p}. Wahlperiode`,
    since: (d: string) => `seit ${d}`,
    range: (from: string, to: string) => `${from} bis ${to}`,
    notMdB:
      "Kein Mitglied des Bundestages (z. B. Mitglied der Bundesregierung).",
    topicsHeading: "Aussagen nach Thema",
    noTopics:
      "Zu den erfassten Themen liegen keine eingeordneten Aussagen vor.",
  },
  timeline: {
    title: (name: string, topic: string) => `${name}: ${topic}`,
    summary: (entries: number, changes: number) =>
      `${entries === 1 ? "1 Aussage" : `${entries} Aussagen`}, ${
        changes === 1 ? "1 Positionswechsel" : `${changes} Positionswechsel`
      }`,
    disclaimer:
      "Die Einordnung erfolgt automatisch durch ein Sprachmodell und kann falsch sein. Maßgeblich ist immer der Originaltext.",
    methodLink: "Wie die Einordnung funktioniert",
    listLabel: "Aussagen in zeitlicher Reihenfolge",
    empty:
      "Zu diesem Thema liegen von dieser Person keine eingeordneten Aussagen vor.",
    change: "Positionswechsel",
    changeDetail: (before: string, after: string) =>
      `vorher ${before}, jetzt ${after}`,
    quoteLabel: "Kernaussage",
    rationale: "Einordnung",
    protocol: (period: number, session: number) =>
      `Plenarprotokoll ${period}/${session}`,
    page: (p: number) => `S. ${p}`,
    pdf: "Protokoll als PDF",
    newWindow: "(öffnet in neuem Fenster)",
    asRole: (role: string) => `als ${role}`,
    report: "Fehler melden",
  },
  stance: {
    dafuer: "dafür",
    dagegen: "dagegen",
    neutral: "neutral",
    unklar: "unklar",
  },
  stanceHelp: {
    dafuer: "spricht sich für eine Pflicht aus",
    dagegen: "spricht sich gegen eine Pflicht aus",
    neutral: "beschreibt, ohne Position zu beziehen",
    unklar: "Position nicht eindeutig erkennbar",
  },
  legend: "Legende",
  report: {
    title: "Fehler melden",
    intro:
      "Stimmt die Einordnung nicht, fehlt Kontext oder ist das Zitat falsch zugeordnet? Beschreiben Sie kurz, was nicht stimmt.",
    message: "Was ist falsch?",
    messageHint: "Mindestens 10, höchstens 2000 Zeichen.",
    contact: "E-Mail-Adresse (freiwillig)",
    contactHint: "Nur für Rückfragen. Wird nicht veröffentlicht.",
    submit: "Meldung senden",
    cancel: "Abbrechen",
    close: "Schließen",
    sending: "Wird gesendet …",
    thanks: "Vielen Dank! Ihre Meldung ist eingegangen und wird geprüft.",
  },
  methodology: {
    title: "Methodik",
    intro:
      "Kurswechsel ist nur so glaubwürdig wie seine Neutralität. Diese Seite erklärt, woher die Daten stammen, wie sie eingeordnet werden und wo die Grenzen liegen.",
    sections: [
      {
        heading: "Datenquelle",
        body: "Grundlage sind die amtlichen Plenarprotokolle des Deutschen Bundestages ab der 19. Wahlperiode (2017) im XML-Format sowie die Stammdaten der Abgeordneten. Zwischenrufe, Beifall und Äußerungen der Sitzungsleitung werden nicht als Aussagen der Rednerin oder des Redners gewertet. Zitate, die jemand im Plenum vorliest, werden ebenfalls nicht als eigene Aussage gezählt.",
      },
      {
        heading: "Themenzuordnung",
        body: "Absätze werden zunächst über Stichwörter einem Thema zugeordnet, beim Thema Wehrpflicht etwa „Wehrpflicht“, „Wehrdienst“ oder „Musterung“. Das Sprachmodell prüft anschließend, ob der Absatz das Thema tatsächlich behandelt.",
      },
      {
        heading: "Einordnung durch ein Sprachmodell",
        body: "Ein lokal betriebenes Sprachmodell ordnet jeden Absatz als dafür, dagegen, neutral oder unklar ein und nennt die Textstelle, auf die es sich stützt. Diese Stelle wird nur angezeigt, wenn sie wörtlich im Protokoll steht. Das Modell kennt weder Namen noch Partei der Person.",
      },
      {
        heading: "Positionswechsel",
        body: "Verglichen werden Sitzungstage, nicht einzelne Absätze: Überwiegen an einem Tag Aussagen dafür oder dagegen, gilt das als Position dieses Tages. Unterscheidet sie sich vom letzten Tag mit einer Position, wird ein Positionswechsel markiert. Neue Fakten, Koalitionsverträge oder Krisen können einen Wechsel gut erklären – eine Wertung ist damit nicht verbunden.",
      },
      {
        heading: "Genauigkeit",
        body: "Die Trefferquote des Modells wird an einem von Hand eingeordneten Prüfdatensatz gemessen. Die Messung wird veröffentlicht, sobald der Prüfdatensatz unabhängig kontrolliert ist.",
      },
      {
        heading: "Fehler und Korrekturen",
        body: "Jeder Eintrag hat einen Knopf „Fehler melden“. Meldungen werden geprüft; falsche Einordnungen werden korrigiert.",
      },
    ],
  },
  imprint: {
    title: "Impressum",
    body: "Angaben gemäß § 5 DDG werden vor der Veröffentlichung ergänzt.",
  },
  privacy: {
    title: "Datenschutz",
    body: "Diese Website setzt keine Cookies und bindet keine Inhalte Dritter ein. Beim Melden eines Fehlers werden nur die Meldung und – falls angegeben – Ihre E-Mail-Adresse gespeichert, ausschließlich zur Bearbeitung. Die vollständige Datenschutzerklärung wird vor der Veröffentlichung ergänzt.",
  },
  notFound: {
    title: "Seite nicht gefunden",
    body: "Die aufgerufene Adresse gibt es nicht.",
    back: "Zur Startseite",
  },
} as const;

/** The German message for an API error code; unknown codes get the generic one. */
export function errorMessage(code: string): string {
  return Object.hasOwn(de.errors, code)
    ? de.errors[code as keyof typeof de.errors]
    : de.errors.internal;
}
