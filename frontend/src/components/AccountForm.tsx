import { FormEvent, useState } from "react";
import type { AccountMode, Theme } from "../App";
import { apiRequest, APIError, readTodos, readUser, Todo, User } from "../api";
import ThemeToggle from "./ThemeToggle";

type Props = {
  mode: AccountMode;
  notice: string;
  theme: Theme;
  onToggleTheme: () => void;
  onBack: () => void;
  onModeChange: (mode: AccountMode) => void;
  onAuthenticated: (user: User, todos: Todo[], loadError?: string) => void;
};

export default function AccountForm({ mode, notice, theme, onToggleTheme, onBack, onModeChange, onAuthenticated }: Props) {
  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const signingUp = mode === "signup";

  function changeMode(nextMode: AccountMode) {
    onModeChange(nextMode);
    setError("");
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setLoading(true);
    try {
      const path = signingUp ? "/auth/register" : "/auth/login";
      const body = signingUp ? { fullName: fullName.trim(), email: email.trim(), password } : { email: email.trim(), password };
      const response = await apiRequest(path, { method: "POST", body: JSON.stringify(body) });
      const user = await readUser(response);
      try {
        const todosResponse = await apiRequest("/todos");
        const todos = await readTodos(todosResponse);
        onAuthenticated(user, todos);
      } catch (caught) {
        if (caught instanceof APIError && caught.status === 401) {
          throw caught;
        }
        onAuthenticated(user, [], caught instanceof Error ? caught.message : "Unable to load your tasks.");
      }
    } catch (caught) {
      setError(caught instanceof APIError ? caught.message : "We couldn't connect. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="account-page">
      <section className="account-visual">
        <a className="brand account-brand" href="/" onClick={(event) => { event.preventDefault(); onBack(); }}>
          <span className="brand-mark" aria-hidden="true">d</span><span>daymark</span>
        </a>
        <div className="account-quote">
          <span className="eyebrow">A LITTLE MORE CLARITY</span>
          <h1>Make room for what matters.</h1>
          <p>A calm, simple place to keep your next steps in view.</p>
        </div>
        <span className="account-visual-foot"><span className="note-dot" /> Your space. Your pace.</span>
        <span className="account-orbit account-orbit-one" />
        <span className="account-orbit account-orbit-two" />
      </section>

      <section className="account-content">
        <ThemeToggle theme={theme} onToggle={onToggleTheme} compact />
        <button className="back-link" type="button" onClick={onBack}><span aria-hidden="true">←</span> Back to Daymark</button>
        <div className="account-card">
          <div className="account-mobile-brand brand">
            <span className="brand-mark" aria-hidden="true">d</span><span>daymark</span>
          </div>
          <span className="eyebrow">{signingUp ? "YOUR OWN LITTLE SPACE" : "GOOD TO HAVE YOU BACK"}</span>
          <h2>{signingUp ? "Start with a fresh page." : "Welcome back."}</h2>
          <p className="account-lead">{signingUp ? "Create your account and make room for what matters." : "Sign in and pick up right where you left off."}</p>
          {(error || notice) && <div className={`notice${error ? " error-notice" : ""}`} role={error ? "alert" : "status"}>{error || notice}</div>}

          <form className="account-form" onSubmit={(event) => void submit(event)}>
            {signingUp && (
              <>
                <label htmlFor="fullName">Your name</label>
                <input id="fullName" name="name" autoComplete="name" autoFocus required maxLength={100} value={fullName} onChange={(event) => setFullName(event.target.value)} placeholder="How should we call you?" />
              </>
            )}
            <label htmlFor="email">Email address</label>
            <input id="email" name="email" type="email" autoComplete="email" autoFocus={!signingUp} required maxLength={254} value={email} onChange={(event) => setEmail(event.target.value)} placeholder="you@example.com" />
            <label htmlFor="password">Password</label>
            <input id="password" name="password" type="password" autoComplete={signingUp ? "new-password" : "current-password"} required minLength={8} maxLength={72} value={password} onChange={(event) => setPassword(event.target.value)} placeholder={signingUp ? "At least 8 characters" : "Your password"} />
            <button className="button account-submit" type="submit" disabled={loading}>
              {loading ? "One moment..." : signingUp ? "Create my account" : "Sign in"} <span aria-hidden="true">↗</span>
            </button>
          </form>
          <p className="account-switch">
            {signingUp ? "Already have a space?" : "New to Daymark?"}
            <button type="button" onClick={() => changeMode(signingUp ? "signin" : "signup")}>{signingUp ? "Sign in" : "Create an account"}</button>
          </p>
          <p className="account-security"><span aria-hidden="true">⌑</span> Your account and tasks stay private to you.</p>
        </div>
      </section>
    </main>
  );
}
