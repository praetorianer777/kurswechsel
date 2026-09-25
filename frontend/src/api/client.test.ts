import { api, ApiError } from "./client";
import { mockFetch } from "../test/helpers";

test("maps error codes to German messages", async () => {
  mockFetch({
    "/api/politicians/x": {
      status: 404,
      body: { error: "unknown_politician" },
    },
  });
  const err = await api.politician("x").catch((e: unknown) => e);
  expect(err).toBeInstanceOf(ApiError);
  expect(err).toMatchObject({
    code: "unknown_politician",
    status: 404,
    message: "Diese Person ist nicht in der Datenbank.",
  });
});

test("falls back to a generic message for unknown codes and non-JSON bodies", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => new Response("<html>502</html>", { status: 502 })),
  );
  await expect(api.topics()).rejects.toMatchObject({
    code: "internal",
    status: 502,
  });
  mockFetch({ "/api/topics": { status: 418, body: { error: "teapot" } } });
  await expect(api.topics()).rejects.toMatchObject({
    message: expect.stringContaining("Server"),
  });
});

test("encodes query parameters", async () => {
  const fetch = mockFetch({
    "/api/politicians?q=M%C3%BCller+%26+Co&topic=wehrpflicht": { body: [] },
  });
  await api.politicians("Müller & Co", "wehrpflicht");
  expect(fetch).toHaveBeenCalledTimes(1);
});

test("passes aborts through instead of reporting a network error", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => {
      throw new DOMException("aborted", "AbortError");
    }),
  );
  await expect(api.topics()).rejects.toMatchObject({ name: "AbortError" });
});
