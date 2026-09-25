import { de } from "./i18n/de";

export function App() {
  return (
    <main className="mx-auto max-w-3xl px-4 py-8">
      <h1 className="text-3xl font-bold">{de.appName}</h1>
      <p className="mt-2 text-lg">{de.tagline}</p>
    </main>
  );
}
