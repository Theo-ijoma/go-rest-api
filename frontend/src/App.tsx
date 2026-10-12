import { useEffect, useState } from "react";
import { SpeedInsights } from "@vercel/speed-insights/react";
import { apiRequest, APIError, readTodos, readUser, Todo, User } from "./api";
import AccountForm from "./components/AccountForm";
import Dashboard from "./components/Dashboard";
import LandingPage from "./components/LandingPage";

type Route = "/" | "/login" | "/register" | "/dashboard";
export type AccountMode = "signin" | "signup";
export type Theme = "light" | "dark";

function initialTheme(): Theme {
  try {
    const stored = window.localStorage.getItem("daymark-theme");
    if (stored === "light" || stored === "dark") {
      return stored;
    }
  } catch {
    // Use the operating-system preference when browser storage is unavailable.
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function currentRoute(): Route {
  const path = window.location.pathname.replace(/\/+$/, "") || "/";
  if (path === "/login" || path === "/register" || path === "/dashboard") {
    return path;
  }
  return "/";
}

export default function App() {
  const [route, setRoute] = useState<Route>(currentRoute);
  const [user, setUser] = useState<User | null>(null);
  const [todos, setTodos] = useState<Todo[]>([]);
  const [todosError, setTodosError] = useState("");
  const [checkingSession, setCheckingSession] = useState(true);
  const [notice, setNotice] = useState("");
  const [theme, setTheme] = useState<Theme>(initialTheme);

  function navigate(nextRoute: Route, replace = false) {
    if (window.location.pathname !== nextRoute) {
      if (replace) {
        window.history.replaceState(null, "", nextRoute);
      } else {
        window.history.pushState(null, "", nextRoute);
      }
    }
    setRoute(nextRoute);
  }

  useEffect(() => {
    function handlePopState() {
      setRoute(currentRoute());
    }
    window.addEventListener("popstate", handlePopState);
    const path = window.location.pathname.replace(/\/+$/, "") || "/";
    if (path !== route) {
      window.history.replaceState(null, "", route);
    }
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    try {
      window.localStorage.setItem("daymark-theme", theme);
    } catch {
      // Theme remains active for this visit if browser storage is unavailable.
    }
  }, [theme]);

  function toggleTheme() {
    setTheme((current) => current === "dark" ? "light" : "dark");
  }

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
          navigate("/dashboard", true);
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
            navigate("/login", true);
            setNotice("Your session expired. Sign in again to see your tasks.");
          } else if (sessionRestored) {
            setTodosError(error instanceof Error ? error.message : "Unable to load your tasks.");
          } else if (!(error instanceof APIError && error.status === 401)) {
            setNotice(error instanceof Error ? error.message : "Unable to connect to Daymark.");
          } else if (route === "/dashboard") {
            navigate("/", true);
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
    setNotice("");
    navigate(mode === "signin" ? "/login" : "/register");
  }

  function onAuthenticated(authenticatedUser: User, userTodos: Todo[], loadError = "") {
    setUser(authenticatedUser);
    setTodos(userTodos);
    setTodosError(loadError);
    setNotice("");
    navigate("/dashboard");
  }

  async function reloadTodos() {
    try {
      const response = await apiRequest("/todos");
      setTodos(await readTodos(response));
      setTodosError("");
    } catch (error) {
      if (error instanceof APIError && error.status === 401) {
        setUser(null);
        navigate("/login", true);
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
      setNotice("");
      navigate("/");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Unable to sign out. Please try again.");
    }
  }

  if (checkingSession) {
    return (
      <>
        <main className="session-loading">
          <span className="brand-mark" aria-hidden="true">d</span>
          <p>Getting your space ready...</p>
        </main>
        <SpeedInsights />
      </>
    );
  }

  if (route === "/dashboard" && user) {
    return (
      <>
        <Dashboard user={user} todos={todos} setTodos={setTodos} todosError={todosError} theme={theme} onToggleTheme={toggleTheme} onRetry={() => void reloadTodos()} onSignOut={() => void signOut()} />
        <SpeedInsights />
      </>
    );
  }

  if (route === "/login" || route === "/register") {
    return (
      <>
        <AccountForm
          mode={route === "/login" ? "signin" : "signup"}
          notice={notice}
          theme={theme}
          onToggleTheme={toggleTheme}
          onBack={() => navigate("/")}
          onModeChange={openAccount}
          onAuthenticated={(authenticatedUser, userTodos, loadError) => onAuthenticated(authenticatedUser, userTodos, loadError)}
        />
        <SpeedInsights />
      </>
    );
  }

  return (
    <>
      <LandingPage notice={notice} theme={theme} onToggleTheme={toggleTheme} onAccount={openAccount} />
      <SpeedInsights />
    </>
  );
}
