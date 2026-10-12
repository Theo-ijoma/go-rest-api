import { useEffect, useState } from "react";
import { apiRequest, APIError, readTodos, readUser, Todo, User } from "./api";
import AccountForm from "./components/AccountForm";
import Dashboard from "./components/Dashboard";
import LandingPage from "./components/LandingPage";

type Screen = "landing" | "account" | "dashboard";
export type AccountMode = "signin" | "signup";

export default function App() {
  const [screen, setScreen] = useState<Screen>("landing");
  const [accountMode, setAccountMode] = useState<AccountMode>("signup");
  const [user, setUser] = useState<User | null>(null);
  const [todos, setTodos] = useState<Todo[]>([]);
  const [todosError, setTodosError] = useState("");
  const [checkingSession, setCheckingSession] = useState(true);
  const [notice, setNotice] = useState("");

  useEffect(() => {
    let active = true;
    async function restoreSession() {
      let sessionRestored = false;
      try {
        const response = await apiRequest("/auth/me");
        const currentUser = await readUser(response);
        sessionRestored = true;
        if (active) {
          setUser(currentUser);
          setScreen("dashboard");
        }

        const todoResponse = await apiRequest("/todos");
        const currentTodos = await readTodos(todoResponse);
        if (active) {
          setTodos(currentTodos);
          setTodosError("");
        }
      } catch (error) {
        if (active) {
          if (sessionRestored && error instanceof APIError && error.status === 401) {
            setUser(null);
            setScreen("landing");
            setNotice("Your session expired. Sign in again to see your tasks.");
          } else if (sessionRestored) {
            setTodosError(error instanceof Error ? error.message : "Unable to load your tasks.");
          } else if (!(error instanceof APIError && error.status === 401)) {
            setNotice(error instanceof Error ? error.message : "Unable to connect to Daymark.");
          }
        }
      } finally {
        if (active) {
          setCheckingSession(false);
        }
      }
    }
    void restoreSession();
    return () => {
      active = false;
    };
  }, []);

  function openAccount(mode: AccountMode) {
    setAccountMode(mode);
    setNotice("");
    setScreen("account");
  }

  function onAuthenticated(authenticatedUser: User, userTodos: Todo[], loadError = "") {
    setUser(authenticatedUser);
    setTodos(userTodos);
    setTodosError(loadError);
    setNotice("");
    setScreen("dashboard");
  }

  async function reloadTodos() {
    try {
      const response = await apiRequest("/todos");
      setTodos(await readTodos(response));
      setTodosError("");
    } catch (error) {
      if (error instanceof APIError && error.status === 401) {
        setUser(null);
        setScreen("landing");
        setNotice("Your session expired. Sign in again to see your tasks.");
        return;
      }
      setTodosError(error instanceof Error ? error.message : "Unable to load your tasks.");
    }
  }

  async function signOut() {
    try {
      await apiRequest("/auth/logout", { method: "POST" });
      setUser(null);
      setTodos([]);
      setNotice("You are signed out. Your next step can wait until you are ready.");
      setScreen("landing");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Unable to sign out. Please try again.");
    }
  }

  if (checkingSession) {
    return (
      <main className="session-loading">
        <span className="brand-mark" aria-hidden="true">d</span>
        <p>Getting your space ready...</p>
      </main>
    );
  }

  if (screen === "dashboard" && user) {
    return <Dashboard user={user} todos={todos} setTodos={setTodos} todosError={todosError} onRetry={() => void reloadTodos()} onSignOut={() => void signOut()} />;
  }

  if (screen === "account") {
    return (
      <AccountForm
        initialMode={accountMode}
        notice={notice}
        onBack={() => setScreen("landing")}
        onModeChange={setAccountMode}
        onAuthenticated={(authenticatedUser, userTodos, loadError) => onAuthenticated(authenticatedUser, userTodos, loadError)}
      />
    );
  }

  return <LandingPage notice={notice} onAccount={openAccount} />;
}
