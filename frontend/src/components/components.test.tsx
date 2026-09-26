import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { entry, expectAccessible, mockFetch } from "../test/helpers";
import { Highlighted } from "./Highlighted";
import { ReportDialog } from "./ReportDialog";
import { StanceBadge, StanceLegend } from "./StanceBadge";
import { TimelineItem } from "./TimelineItem";

describe("StanceBadge", () => {
  test.each([
    ["dafuer", "dafür"],
    ["dagegen", "dagegen"],
    ["neutral", "neutral"],
    ["unklar", "unklar"],
  ] as const)(
    "%s shows its label as text, not only colour",
    (stance, label) => {
      render(<StanceBadge stance={stance} />);
      expect(screen.getByText(label)).toBeVisible();
    },
  );

  test("legend explains every stance", async () => {
    const { container } = render(<StanceLegend />);
    expect(
      screen.getByRole("heading", { name: "Legende" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText("spricht sich gegen eine Pflicht aus"),
    ).toBeInTheDocument();
    await expectAccessible(container);
  });
});

describe("Highlighted", () => {
  test("marks the quote even across different whitespace", () => {
    render(
      <Highlighted
        text={"Wir müssen die\nWehrpflicht wieder einführen."}
        quote="„die Wehrpflicht wieder einführen“"
      />,
    );
    expect(
      screen.getByText(/die\s+Wehrpflicht wieder einführen/, {
        selector: "mark",
      }),
    ).toBeInTheDocument();
  });

  test("shows plain text when the quote is not in it", () => {
    const { container } = render(
      <Highlighted text="Ganz anderer Text." quote="nicht enthalten" />,
    );
    expect(container.querySelector("mark")).toBeNull();
    expect(container).toHaveTextContent("Ganz anderer Text.");
  });

  test("escapes regular expression characters in the quote", () => {
    const { container } = render(
      <Highlighted
        text="Kosten (geschätzt) 3 Mrd.?"
        quote="(geschätzt) 3 Mrd.?"
      />,
    );
    expect(container.querySelector("mark")).toHaveTextContent(
      "(geschätzt) 3 Mrd.?",
    );
  });
});

describe("TimelineItem", () => {
  const renderItem = (e = entry()) =>
    render(
      <MemoryRouter>
        <ol>
          <TimelineItem entry={e} topic="wehrpflicht" />
        </ol>
      </MemoryRouter>,
    );

  test("shows date, stance, source and quote", async () => {
    const { container } = renderItem();
    const article = screen.getByRole("article", { name: "1. März 2019" });
    expect(within(article).getByText("dafür")).toBeInTheDocument();
    expect(
      within(article).getByText("Plenarprotokoll 19/50, S. 5500 · CDU/CSU"),
    ).toBeInTheDocument();
    const pdf = within(article).getByRole("link", {
      name: /Protokoll als PDF \(öffnet in neuem Fenster\)/,
    });
    expect(pdf).toHaveAttribute(
      "href",
      "https://dserver.bundestag.de/btp/19/19050.pdf",
    );
    expect(pdf).toHaveAttribute("rel", "noopener noreferrer");
    expect(container.querySelector("mark")).toHaveTextContent(
      "die Wehrpflicht wieder einführen",
    );
    expect(container.querySelector("time")).toHaveAttribute(
      "dateTime",
      "2019-03-01",
    );
    await expectAccessible(container);
  });

  test("labels a change of position in words", () => {
    renderItem(
      entry({ stance: "dagegen", change: true, previous_stance: "dafuer" }),
    );
    expect(screen.getByText("Positionswechsel:")).toBeInTheDocument();
    expect(screen.getByText("vorher dafür, jetzt dagegen")).toBeInTheDocument();
  });

  test("names the role when there is no faction", () => {
    renderItem(
      entry({
        faction: undefined,
        role: "Bundesminister der Verteidigung",
        page: undefined,
      }),
    );
    expect(
      screen.getByText(
        "Plenarprotokoll 19/50 · als Bundesminister der Verteidigung",
      ),
    ).toBeInTheDocument();
  });
});

describe("ReportDialog", () => {
  const open = async () => {
    const user = userEvent.setup();
    const view = render(
      <ReportDialog
        paragraphId={7}
        topic="wehrpflicht"
        context="1. März 2019"
      />,
    );
    await user.click(
      screen.getByRole("button", { name: "Fehler melden: 1. März 2019" }),
    );
    return { user, ...view };
  };

  test("rejects a too short message without calling the server", async () => {
    const fetch = mockFetch({});
    const { user } = await open();
    await user.type(screen.getByLabelText("Was ist falsch?"), "kurz");
    await user.click(screen.getByRole("button", { name: "Meldung senden" }));
    expect(screen.getByRole("alert")).toHaveTextContent(
      "mindestens 10 Zeichen",
    );
    expect(screen.getByLabelText("Was ist falsch?")).toHaveAttribute(
      "aria-invalid",
      "true",
    );
    expect(fetch).not.toHaveBeenCalled();
  });

  test("sends a report and thanks the reader", async () => {
    const fetch = mockFetch({
      "/api/reports": { status: 201, body: { status: "received" } },
    });
    const { user, container } = await open();
    await expectAccessible(container);
    await user.type(
      screen.getByLabelText("Was ist falsch?"),
      "Das Zitat ist aus dem Zusammenhang gerissen.",
    );
    await user.type(
      screen.getByLabelText("E-Mail-Adresse (freiwillig)"),
      "a@example.org",
    );
    await user.click(screen.getByRole("button", { name: "Meldung senden" }));
    expect(await screen.findByRole("status")).toHaveTextContent("Vielen Dank!");
    const body = JSON.parse(fetch.mock.calls[0]![1]!.body as string);
    expect(body).toEqual({
      paragraph_id: 7,
      topic: "wehrpflicht",
      message: "Das Zitat ist aus dem Zusammenhang gerissen.",
      contact: "a@example.org",
    });
    await user.click(screen.getByRole("button", { name: "Schließen" }));
    expect(container.querySelector("dialog")).not.toHaveAttribute("open");
  });

  test("shows the server's reason in German", async () => {
    mockFetch({
      "/api/reports": { status: 429, body: { error: "too_many_reports" } },
    });
    const { user } = await open();
    await user.type(
      screen.getByLabelText("Was ist falsch?"),
      "Das Zitat ist aus dem Zusammenhang gerissen.",
    );
    await user.click(screen.getByRole("button", { name: "Meldung senden" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "mehrere Meldungen",
    );
  });
});
