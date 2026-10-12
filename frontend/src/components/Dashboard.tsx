import { FormEvent, useState } from "react";
import { apiRequest, APIError, Todo, User } from "../api";

type Props = {
  user: User;
  todos: Todo[];
  todosError: string;
  setTodos: (todos: Todo[] | ((current: Todo[]) => Todo[])) => void;
  onRetry: () => void;
  onSignOut: () => void;
};

export default function Dashboard({ user, todos, todosError, setTodos, onRetry, onSignOut }: Props) {
  const [newTitle, setNewTitle] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const completedCount = todos.filter((todo) => todo.completed).length;

  function handleError(caught: unknown) {
    if (caught instanceof APIError && caught.status === 401) {
      onSignOut();
      return;
    }
    setError(caught instanceof Error ? caught.message : "Something went wrong. Please try again.");
  }

  async function addTodo(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!newTitle.trim()) {
      return;
    }
    setSaving(true);
    setError("");
    try {
      const response = await apiRequest("/todos", { method: "POST", body: JSON.stringify({ title: newTitle.trim() }) });
      const todo = (await response.json()) as Todo;
      setTodos((current) => [todo, ...current]);
      setNewTitle("");
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
      setTodos((current) => current.map((item) => (item.id === updated.id ? updated : item)));
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
    <main className="app-shell">
      <header className="topbar">
        <a className="brand" href="/">
          <span className="brand-mark" aria-hidden="true">d</span><span>daymark</span>
        </a>
        <div className="topbar-right">
          <span className="account-greeting">A little space for {user.fullName.split(/\s+/)[0]}</span>
          <button className="text-button" type="button" onClick={onSignOut}>Sign out</button>
        </div>
      </header>

      <section className="dashboard">
        <div className="welcome-row">
          <div>
            <span className="eyebrow">YOUR SPACE</span>
            <h1>Good things, one step at a time.</h1>
            <p className="muted">Keep track of what you want to get done.</p>
          </div>
          <div className="date-stamp">{new Intl.DateTimeFormat(undefined, { weekday: "long", month: "short", day: "numeric" }).format(new Date())}</div>
        </div>

        {error && <div className="notice error-notice" role="alert">{error}</div>}

        <div className="summary-grid">
          <article className="summary-card">
            <span className="summary-icon total-icon" aria-hidden="true">↗</span>
            <div><span className="summary-label">On your list</span><strong>{todos.length}</strong></div>
          </article>
          <article className="summary-card">
            <span className="summary-icon done-icon" aria-hidden="true">✓</span>
            <div><span className="summary-label">Completed</span><strong>{completedCount}</strong></div>
          </article>
          <article className="summary-card focus-card">
            <span className="focus-spark" aria-hidden="true">✳</span>
            <div><span className="summary-label">A gentle reminder</span><strong>Progress over perfection.</strong></div>
          </article>
        </div>

        <section className="tasks-panel" aria-labelledby="tasks-heading">
          <div className="tasks-heading">
            <div><h2 id="tasks-heading">Your tasks <span className="task-count">{todos.length}</span></h2><p className="muted">A clear list makes a lighter day.</p></div>
            <span className="completion-label">{completedCount} of {todos.length} done</span>
          </div>
          <form className="add-form" onSubmit={(event) => void addTodo(event)}>
            <span className="add-icon" aria-hidden="true">+</span>
            <input aria-label="New task" value={newTitle} onChange={(event) => setNewTitle(event.target.value)} placeholder="What would you like to do?" maxLength={200} />
            <button className="button add-button" type="submit" disabled={saving || !newTitle.trim()}>{saving ? "Adding..." : "Add task"} <span aria-hidden="true">↗</span></button>
          </form>
          {todosError ? (
            <div className="empty-state list-error">
              <span className="empty-illustration" aria-hidden="true">!</span>
              <strong>We could not load your tasks.</strong>
              <span>{todosError}</span>
              <button className="retry-button" type="button" onClick={onRetry}>Try again</button>
            </div>
          ) : todos.length === 0 ? (
            <div className="empty-state"><span className="empty-illustration" aria-hidden="true">✳</span><strong>A fresh start.</strong><span>Your list is clear. Add a task above when you are ready.</span></div>
          ) : (
            <ul className="todo-list">
              {todos.map((todo) => (
                <li className={`todo-row${todo.completed ? " is-complete" : ""}`} key={todo.id}>
                  <button className="check-button" type="button" aria-label={todo.completed ? `Mark ${todo.title} as incomplete` : `Complete ${todo.title}`} aria-pressed={todo.completed} onClick={() => void toggleTodo(todo)}>
                    {todo.completed && <span aria-hidden="true">✓</span>}
                  </button>
                  <span className="todo-title">{todo.title}</span>
                  <button className="delete-button" type="button" aria-label={`Delete ${todo.title}`} onClick={() => void deleteTodo(todo)}><span aria-hidden="true">×</span></button>
                </li>
              ))}
            </ul>
          )}
          <div className="tasks-footer"><span className="footer-spark" aria-hidden="true">✳</span> Small steps still move you forward.</div>
        </section>
        <footer className="page-footer">Made for a little more focus <span aria-hidden="true">·</span> Daymark</footer>
      </section>
    </main>
  );
}
