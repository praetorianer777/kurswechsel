import { act, renderHook } from "@testing-library/react";
import { ApiError } from "../api/client";
import { useApi } from "./useApi";

function deferred<T>() {
  let resolve!: (v: T) => void;
  const promise = new Promise<T>((r) => (resolve = r));
  return { promise, resolve };
}

test("a slow earlier response never replaces a newer one", async () => {
  const first = deferred<string>();
  const second = deferred<string>();
  const loads: Record<string, Promise<string>> = {
    a: first.promise,
    b: second.promise,
  };
  const { result, rerender } = renderHook(
    ({ id }) => useApi(() => loads[id]!, [id]),
    {
      initialProps: { id: "a" },
    },
  );
  expect(result.current.status).toBe("loading");
  rerender({ id: "b" });
  await act(async () => second.resolve("B"));
  await act(async () => first.resolve("A"));
  expect(result.current).toEqual({ status: "ok", data: "B" });
});

test("wraps unexpected errors", async () => {
  const { result } = renderHook(() =>
    useApi(() => Promise.reject(new Error("x")), []),
  );
  await act(async () => {});
  expect(result.current.status).toBe("error");
  if (result.current.status === "error")
    expect(result.current.error).toBeInstanceOf(ApiError);
});
