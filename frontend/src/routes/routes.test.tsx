import { act, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { App } from "../App";
import {
  expectAccessible,
  mockFetch,
  renderAt,
  timelineResponse,
} from "../test/helpers";

const topics = [
  {
    slug: "wehrpflicht",
    name: "Wehrpflicht",
    question: "Soll der Wehrdienst verpflichtend sein?",
    people: 12,
    entries: 80,
  },
];

describe("layout", () => {
  test("offers a skip link first and has all landmarks", async () => {
    mockFetch({ "/api/topics": { body: topics } });
    const { container } = renderAt(<App />);
    await screen.findByText("Soll der Wehrdienst verpflichtend sein?");
    const links = screen.getAllByRole("link");
    expect(links[0]).toHaveTextContent("Zum Inhalt springen");
    expect(links[0]).toHaveAttribute("href", "#inhalt");
    expect(screen.getByRole("banner")).toBeInTheDocument();
    expect(
      screen.getByRole("navigation", { name: "Hauptnavigation" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("main")).toHaveAttribute("id", "inhalt");
    expect(screen.getByRole("contentinfo")).toBeInTheDocument();
    expect(document.title).toBe("Positionen im Zeitverlauf – Kurswechsel");
    await expectAccessible(container);
  });
});

describe("home", () => {
  test("searches people as you type and links to their page", async () => {
    const fetch = mockFetch({
      "/api/topics": { body: topics },
      "/api/politicians?q=m%C3%BCller": {
        body: [
          {
            id: "11000001",
            name: "Jürgen Müller",
            faction: "CDU/CSU",
            entries: 42,
          },
        ],
      },
    });
    const user = userEvent.setup();
    renderAt(<App />);
    await user.type(screen.getByRole("searchbox", { name: "Name" }), "müller");
    const link = await screen.findByRole("link", { name: /Jürgen Müller/ });
    expect(link).toHaveAttribute("href", "/person/11000001");
    expect(screen.getByText("1 Treffer")).toBeInTheDocument();
    expect(screen.getByText("CDU/CSU · 42 Reden")).toBeInTheDocument();
    expect(
      fetch.mock.calls.filter(([u]) =>
        String(u).startsWith("/api/politicians"),
      ),
    ).toHaveLength(1);
  });

  test("a topic lists people with statements and links straight to the timeline", async () => {
    mockFetch({
      "/api/topics": { body: topics },
      "/api/politicians?topic=wehrpflicht": {
        body: [
          {
            id: "11000001",
            name: "Jürgen Müller",
            faction: "CDU/CSU",
            entries: 3,
          },
        ],
      },
    });
    const user = userEvent.setup();
    renderAt(<App />);
    await user.click(
      await screen.findByRole("button", { name: "Thema: Wehrpflicht" }),
    );
    const link = await screen.findByRole("link", { name: /Jürgen Müller/ });
    expect(link).toHaveAttribute("href", "/person/11000001/wehrpflicht");
    expect(screen.getByText("CDU/CSU · 3 Aussagen")).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Thema" })).toHaveValue(
      "wehrpflicht",
    );
  });

  test("says so when nothing matches", async () => {
    mockFetch({
      "/api/topics": { body: topics },
      "/api/politicians?q=xyz": { body: [] },
    });
    renderAt(<App />, "/?q=xyz");
    expect(await screen.findByText("Keine Treffer.")).toBeInTheDocument();
  });

  test("explains a network failure", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        throw new TypeError("Failed to fetch");
      }),
    );
    renderAt(<App />);
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Keine Verbindung zum Server",
    );
  });
});

describe("person", () => {
  test("shows topics and faction history", async () => {
    mockFetch({
      "/api/politicians/11000001": {
        body: {
          id: "11000001",
          name: "Jürgen Müller",
          party: "CDU",
          is_mdb: true,
          factions: [
            {
              period: 19,
              faction: "CDU/CSU",
              from: "2017-10-24",
              to: "2021-10-26",
            },
          ],
          topics: [{ slug: "wehrpflicht", name: "Wehrpflicht", entries: 3 }],
        },
      },
    });
    const { container } = renderAt(<App />, "/person/11000001");
    expect(
      await screen.findByRole("heading", { level: 1, name: "Jürgen Müller" }),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Wehrpflicht/ })).toHaveAttribute(
      "href",
      "/person/11000001/wehrpflicht",
    );
    expect(
      screen.getByText(
        "(19. Wahlperiode, 24. Oktober 2017 bis 26. Oktober 2021)",
      ),
    ).toBeInTheDocument();
    await expectAccessible(container);
  });

  test("explains unknown people", async () => {
    mockFetch({
      "/api/politicians/x": {
        status: 404,
        body: { error: "unknown_politician" },
      },
    });
    renderAt(<App />, "/person/x");
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Diese Person ist nicht in der Datenbank.",
    );
  });

  test("members of the government without a seat", async () => {
    mockFetch({
      "/api/politicians/9": {
        body: {
          id: "9",
          name: "Boris Minister",
          is_mdb: false,
          factions: [],
          topics: [],
        },
      },
    });
    renderAt(<App />, "/person/9");
    expect(
      await screen.findByText(/Kein Mitglied des Bundestages/),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/keine eingeordneten Aussagen/),
    ).toBeInTheDocument();
  });
});

describe("timeline", () => {
  test("lists statements in order with the change marked", async () => {
    mockFetch({
      "/api/timeline?politician=11000001&topic=wehrpflicht": {
        body: timelineResponse,
      },
    });
    const { container } = renderAt(<App />, "/person/11000001/wehrpflicht");
    expect(
      await screen.findByText("2 Aussagen, 1 Positionswechsel"),
    ).toBeInTheDocument();
    const articles = screen.getAllByRole("article");
    expect(
      articles.map((a) => within(a).getByRole("heading").textContent),
    ).toEqual(["1. März 2019", "1. März 2024"]);
    expect(
      within(articles[1]!).getByText("vorher dafür, jetzt dagegen"),
    ).toBeInTheDocument();
    expect(screen.getByText(/kann falsch sein/)).toBeInTheDocument();
    expect(document.title).toBe("Jürgen Müller: Wehrpflicht – Kurswechsel");
    await expectAccessible(container);
  });

  test("an empty timeline says so", async () => {
    mockFetch({
      "/api/timeline?politician=1&topic=wehrpflicht": {
        body: { ...timelineResponse, changes: 0, entries: [] },
      },
    });
    renderAt(<App />, "/person/1/wehrpflicht");
    expect(
      await screen.findByText(/keine eingeordneten Aussagen vor/),
    ).toBeInTheDocument();
  });
});

describe("static pages", () => {
  test.each([
    ["/methodik", "Methodik"],
    ["/impressum", "Impressum"],
    ["/datenschutz", "Datenschutz"],
    ["/gibt-es-nicht", "Seite nicht gefunden"],
  ])("%s", async (path, heading) => {
    mockFetch({});
    const { container } = renderAt(<App />, path);
    expect(
      screen.getByRole("heading", { level: 1, name: heading }),
    ).toBeInTheDocument();
    await act(async () => {});
    await expectAccessible(container);
  });
});

describe("methodology", () => {
  test("shows the measured accuracy from a reviewed evaluation", async () => {
    mockFetch({
      "/api/evaluation?topic=wehrpflicht": {
        body: {
          topic: "wehrpflicht",
          classifier: "ollama/qwen3:30b-a3b",
          prompt_version: "v1",
          items: 45,
          relevance_precision: 0.9,
          relevance_recall: 0.97,
          stance_accuracy: 0.72,
          stance_macro_f1: 0.6,
          flip_rate: 0.05,
          gold_reviewed: true,
          created_at: "2026-10-01T12:00:00Z",
        },
      },
    });
    const { container } = renderAt(<App />, "/methodik");
    const table = await screen.findByRole("table", {
      name: /Messung zum Thema Wehrpflicht vom 1. Oktober 2026/,
    });
    expect(
      within(table).getByRole("row", { name: /Einordnung stimmt/ }),
    ).toHaveTextContent("72 %");
    expect(
      screen.getByText(
        "Grundlage: 45 geprüfte Absätze, Modell ollama/qwen3:30b-a3b.",
      ),
    ).toBeInTheDocument();
    await expectAccessible(container);
  });

  test("says the measurement is pending without a reviewed evaluation", async () => {
    mockFetch({
      "/api/evaluation?topic=wehrpflicht": {
        status: 404,
        body: { error: "no_evaluation" },
      },
    });
    renderAt(<App />, "/methodik");
    expect(
      await screen.findByText(
        /sobald der Prüfdatensatz von einer Person kontrolliert ist/,
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole("table")).toBeNull();
  });
});
