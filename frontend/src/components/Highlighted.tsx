/** Renders text with the first occurrence of quote marked, ignoring whitespace differences. */
export function Highlighted({ text, quote }: { text: string; quote?: string }) {
  const q = quote?.trim().replace(/^[„“"'»«]+|[„“"'»«]+$/g, "");
  if (!q) return <>{text}</>;
  const pattern = new RegExp(
    q
      .split(/\s+/)
      .map((w) => w.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))
      .join("\\s+"),
  );
  const m = pattern.exec(text);
  if (!m) return <>{text}</>;
  return (
    <>
      {text.slice(0, m.index)}
      <mark className="rounded-sm bg-highlight px-0.5 text-ink">{m[0]}</mark>
      {text.slice(m.index + m[0].length)}
    </>
  );
}
