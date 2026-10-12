import { FormEvent, useState } from "react";
import type { Theme } from "../App";
import { apiRequest, APIError, Todo, User } from "../api";
import ThemeToggle from "./ThemeToggle";

type Props = {
  user: User;
  todos: Todo[];
  todosError: string;
  setTodos: (todos: Todo[] | ((current: Todo[]) => Todo[])) => void;
  theme: Theme;
  onToggleTheme: () => void;
  onRetry: () => void;
  onSignOut: () => void;
};

type TaskFilter = "all" | "open" | "done";

const filters: { id: TaskFilter; label: string }[] = [
  { id: "all", label: "Everything" },
  { id: "open", label: "To do" },
  { id: "done", label: "Done" },
];

export default function Dashboard({
  user,
  todos,
  todosError,
  setTodos,
  theme,
  onToggleTheme,
  onRetry,
  onSignOut,
}: Props) {
  const [newTitle, setNewTitle] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [filter, setFilter] = useState<TaskFilter>("all");
  const completedCount = todos.filter((todo) => todo.completed).length;
  const remainingCount = todos.length - completedCount;
  const completion = todos.length === 0 ? 0 : Math.round((completedCount / todos.length) * 100);
  const visibleTodos = todos.filter((todo) => {
    if (filter === "open") return !todo.completed;
    if (filter === "done") return todo.completed;
    return true;
  });
  const firstName = user.fullName.trim().split(/\s+/)[0];
  const hour = new Date().getHours();
  const greeting = hour < 12 ? "Good morning" : hour < 18 ? "Good afternoon" : "Good evening";
  const today = new Intl.DateTimeFormat(undefined, { weekday: "long", month: "long", day: "numeric" }).format(new Date());

  function handleError(caught: unknown) {
    if (caught instanceof APIError && caught.status === 401) {
      onSignOut();
      return;
    }
    setError(caught instanceof Error ? caught.message : "That did not work. Try again.");
  }

  async function addTodo(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const title = newTitle.trim();
    if (!title || saving) return;

    setSaving(true);
    setError("");
    try {
      const response = await apiRequest("/todos", { method: "POST", body: JSON.stringify({ title }) });
      const todo = (await response.json()) as Todo;
      setTodos((current) => [todo, ...current]);
      setNewTitle("");
      setFilter("all");
    } catch (caught) {
      handleError(caught);
    } finally {
      setSaving(false);
    }
  }

  async function toggleTodo(todo: Todo) {
    setError("");
    try {
      const response = await apiRequest(`/todos/${todo.id}`, {
        method: "PUT",
        body: JSON.stringify({ title: todo.title, completed: !todo.completed }),
      });
      const updated = (await response.json()) as Todo;
      setTodos((current) => current.map((item) => item.id === updated.id ? updated : item));
    } catch (caught) {
      handleError(caught);
    }
  }

  async function deleteTodo(todo: Todo) {
    setError("");
    try {
      await apiRequest(`/todos/${todo.id}`, { method: "DELETE" });
      setTodos((current) => current.filter((item) => item.id !== todo.id));
    } catch (caught) {
      handleError(caught);
    }
  }

  return (
    <main className="app-shell dashboard-shell">
      <aside className="dashboard-sidebar" aria-label="Dashboard sidebar">
        <a className="brand sidebar-brand" href="#overview">
          <span className="brand-mark" aria-hidden="true">d</span><span>daymark</span>
        </a>

        <nav className="sidebar-nav" aria-label="Dashboard navigation">
          <a className="sidebar-link active" href="#overview"><span aria-hidden="true">◫</span> Overview</a>
          <a className="sidebar-link" href="#your-tasks"><span aria-hidden="true">☷</span> Tasks <span className="sidebar-count">{todos.length}</span></a>
          <div className="mobile-sidebar-actions">
            <ThemeToggle theme={theme} onToggle={onToggleTheme} compact />
            <button className="signout-icon" type="button" onClick={onSignOut} aria-label="Sign out" title="Sign out">↗</button>
          </div>
        </nav>

        <div className="sidebar-bottom">
          <ThemeToggle theme={theme} onToggle={onToggleTheme} />
          <div className="sidebar-profile">
            <span className="profile-avatar" aria-hidden="true">{firstName.charAt(0).toUpperCase()}</span>
            <span className="profile-copy"><strong>{firstName}</strong><small>Personal space</small></span>
            <button className="signout-icon" type="button" onClick={onSignOut} aria-label="Sign out" title="Sign out">↗</button>
          </div>
        </div>
      </aside>

      <div className="dashboard-main">
        <section className="dashboard-content" id="overview">
          <div className="dashboard-heading">
            <div>
              <span className="eyebrow">{today.toUpperCase()}</span>
              <h1>{greeting}, {firstName}<span className="heading-period">.</span></h1>
              <p>{remainingCount === 0 ? "Nothing pressing. Enjoy the space." : `${remainingCount} ${remainingCount === 1 ? "task" : "tasks"} left, at your own pace.`}</p>
            </div>
            <div className="day-mark" aria-hidden="true"><span>✳</span></div>
          </div>

          {error && <div className="notice error-notice" role="alert">{error}</div>}

          <div className="dashboard-stats">
            <article className="stat-card stat-card-focus">
              <div className="stat-card-top"><span>Still on your list</span><span className="stat-icon" aria-hidden="true">↗</span></div>
              <strong className="stat-value">{remainingCount.toString().padStart(2, "0")}</strong>
              <span className="stat-note">{remainingCount === 1 ? "one thing to do" : "left to do"}</span>
            </article>
            <article className="stat-card">
              <div className="stat-card-top"><span>Already done</span><span className="stat-icon stat-icon-done" aria-hidden="true">✓</span></div>
              <strong className="stat-value">{completedCount.toString().padStart(2, "0")}</strong>
              <span className="stat-note">done and dusted</span>
            </article>
            <article className="stat-card progress-stat">
              <div className="stat-card-top"><span>Your pace</span><span className="progress-percent">{completion}%</span></div>
              <div className="progress-track" role="progressbar" aria-label="Tasks completed" aria-valuemin={0} aria-valuemax={100} aria-valuenow={completion}>
                <span style={{ width: `${completion}%` }} />
              </div>
              <span className="stat-note">{todos.length === 0 ? "A clean slate" : `${completedCount} of ${todos.length} done`}</span>
            </article>
          </div>

          <section className="tasks-panel" id="your-tasks" aria-labelledby="tasks-heading">
            <div className="tasks-heading">
              <div>
                <span className="eyebrow">ON YOUR MIND</span>
                <h2 id="tasks-heading">Tasks</h2>
              </div>
              <div className="task-filters" role="group" aria-label="Filter tasks">
                {filters.map((item) => (
                  <button
                    className={`filter-button${filter === item.id ? " selected" : ""}`}
                    key={item.id}
                    type="button"
                    aria-pressed={filter === item.id}
                    onClick={() => setFilter(item.id)}
                  >
                    {item.label}
                  </button>
                ))}
              </div>
            </div>

            <form className="add-form" onSubmit={(event) => void addTodo(event)}>
              <span className="add-icon" aria-hidden="true">+</span>
              <input
                aria-label="Add a task"
                value={newTitle}
                onChange={(event) => setNewTitle(event.target.value)}
                placeholder="Add a task…"
                maxLength={200}
              />
              <button className="button add-button" type="submit" disabled={saving || !newTitle.trim()}>
                {saving ? "Adding…" : "Add task"} <span aria-hidden="true">↗</span>
              </button>
            </form>

            {todosError ? (
              <div className="empty-state list-error">
                <span className="empty-illustration" aria-hidden="true">!</span>
                <strong>Your list didn’t load.</strong>
                <span>{todosError}</span>
                <button className="retry-button" type="button" onClick={onRetry}>Try again</button>
              </div>
            ) : visibleTodos.length === 0 ? (
              <div className="empty-state">
                <span className="empty-illustration" aria-hidden="true">{filter === "done" ? "✓" : "✳"}</span>
                <strong>{filter === "done" ? "Nothing finished just yet." : filter === "open" ? "You’re all caught up." : "A little room to breathe."}</strong>
                <span>{filter === "all" ? "Add a task above when something comes to mind." : "Your list will find its next thing when you do."}</span>
              </div>
            ) : (
              <ul className="todo-list">
                {visibleTodos.map((todo) => (
                  <li className={`todo-row${todo.completed ? " is-complete" : ""}`} key={todo.id}>
                    <button
                      className="check-button"
                      type="button"
                      aria-label={todo.completed ? `Mark ${todo.title} as not done` : `Mark ${todo.title} as done`}
                      aria-pressed={todo.completed}
                      onClick={() => void toggleTodo(todo)}
                    >
                      {todo.completed && <span aria-hidden="true">✓</span>}
                    </button>
                    <span className="todo-title">{todo.title}</span>
                    <button className="delete-button" type="button" aria-label={`Remove ${todo.title}`} onClick={() => void deleteTodo(todo)}>
                      <span aria-hidden="true">×</span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
            <div className="tasks-footer"><span className="footer-spark" aria-hidden="true">✳</span> One thing at a time.</div>
          </section>
        </section>
      </div>
    </main>
  );
}
