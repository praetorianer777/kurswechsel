import { useId, useRef, useState, type FormEvent } from "react";
import { api, ApiError } from "../api/client";
import { de } from "../i18n/de";

type State =
  | { kind: "idle" }
  | { kind: "sending" }
  | { kind: "sent" }
  | { kind: "error"; message: string };

/**
 * A "report an error" button with its form in a modal dialog. The native
 * <dialog> traps focus, closes on Escape and returns focus to the button.
 */
export function ReportDialog({
  paragraphId,
  topic,
  context,
}: {
  paragraphId: number;
  topic: string;
  /** Names the entry for screen readers, e.g. its date. */
  context: string;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [state, setState] = useState<State>({ kind: "idle" });
  const [message, setMessage] = useState("");
  const [contact, setContact] = useState("");
  const id = useId();

  const open = () => {
    setState({ kind: "idle" });
    dialog.current?.showModal();
  };
  const close = () => dialog.current?.close();

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (message.trim().length < 10) {
      setState({ kind: "error", message: de.errors.message_too_short });
      return;
    }
    setState({ kind: "sending" });
    try {
      await api.report({ paragraph_id: paragraphId, topic, message, contact });
      setState({ kind: "sent" });
      setMessage("");
      setContact("");
    } catch (err) {
      setState({
        kind: "error",
        message: err instanceof ApiError ? err.message : de.errors.internal,
      });
    }
  };

  const errorId = `${id}-error`;
  const hintId = `${id}-hint`;
  const contactHintId = `${id}-contact-hint`;

  return (
    <>
      <button
        type="button"
        onClick={open}
        aria-label={`${de.timeline.report}: ${context}`}
        className="inline-flex min-h-11 items-center rounded-md px-3 text-sm underline underline-offset-4 hover:bg-line/40"
      >
        {de.timeline.report}
      </button>
      <dialog
        ref={dialog}
        aria-labelledby={`${id}-title`}
        className="m-auto w-[min(36rem,calc(100vw-2rem))] rounded-lg border border-line bg-surface p-0 text-ink shadow-xl backdrop:bg-black/50"
      >
        <div className="flex flex-col gap-4 p-5">
          <h2 id={`${id}-title`} className="text-xl font-bold">
            {de.report.title}
          </h2>
          {state.kind === "sent" ? (
            <>
              <p role="status">{de.report.thanks}</p>
              <div className="flex justify-end">
                <button type="button" onClick={close} className="btn-primary">
                  {de.report.close}
                </button>
              </div>
            </>
          ) : (
            <form onSubmit={submit} noValidate className="flex flex-col gap-4">
              <p className="text-muted">{de.report.intro}</p>
              <div className="flex flex-col gap-1">
                <label htmlFor={`${id}-message`} className="font-semibold">
                  {de.report.message}
                </label>
                <textarea
                  id={`${id}-message`}
                  value={message}
                  onChange={(e) => setMessage(e.target.value)}
                  required
                  minLength={10}
                  maxLength={2000}
                  rows={5}
                  aria-describedby={`${hintId}${state.kind === "error" ? ` ${errorId}` : ""}`}
                  aria-invalid={state.kind === "error" ? true : undefined}
                  className="field"
                />
                <p id={hintId} className="text-sm text-muted">
                  {de.report.messageHint}
                </p>
              </div>
              <div className="flex flex-col gap-1">
                <label htmlFor={`${id}-contact`} className="font-semibold">
                  {de.report.contact}
                </label>
                <input
                  id={`${id}-contact`}
                  type="email"
                  autoComplete="email"
                  value={contact}
                  onChange={(e) => setContact(e.target.value)}
                  maxLength={200}
                  aria-describedby={contactHintId}
                  className="field"
                />
                <p id={contactHintId} className="text-sm text-muted">
                  {de.report.contactHint}
                </p>
              </div>
              {state.kind === "error" && (
                <p
                  id={errorId}
                  role="alert"
                  className="font-semibold text-danger"
                >
                  {state.message}
                </p>
              )}
              <div className="flex flex-wrap justify-end gap-2">
                <button type="button" onClick={close} className="btn-secondary">
                  {de.report.cancel}
                </button>
                <button
                  type="submit"
                  disabled={state.kind === "sending"}
                  className="btn-primary"
                >
                  {state.kind === "sending"
                    ? de.report.sending
                    : de.report.submit}
                </button>
              </div>
            </form>
          )}
        </div>
      </dialog>
    </>
  );
}
