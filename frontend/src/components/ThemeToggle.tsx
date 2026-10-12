import type { Theme } from "../App";

type Props = {
  theme: Theme;
  onToggle: () => void;
  compact?: boolean;
};

export default function ThemeToggle({ theme, onToggle, compact = false }: Props) {
  const dark = theme === "dark";
  return (
    <button
      className={`theme-toggle${compact ? " theme-toggle-compact" : ""}`}
      type="button"
      onClick={onToggle}
      aria-label={`Switch to ${dark ? "light" : "dark"} mode`}
      title={`Switch to ${dark ? "light" : "dark"} mode`}
    >
      <span className="theme-toggle-icon" aria-hidden="true">{dark ? "☼" : "☾"}</span>
      {!compact && <span>{dark ? "Light appearance" : "Dark appearance"}</span>}
    </button>
  );
}
