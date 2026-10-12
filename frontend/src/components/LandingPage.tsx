import { useState } from "react";
import { AccountMode } from "../App";
import Reveal from "./Reveal";

type Props = {
  notice: string;
  onAccount: (mode: AccountMode) => void;
};

const steps = [
  { number: "01", title: "Make a little space", detail: "Create your free account with your name, email, and a password." },
  { number: "02", title: "Get it out of your head", detail: "Add the next small thing you want to move forward." },
  { number: "03", title: "Keep your own pace", detail: "Come back to a private list that is yours, on every visit." },
];

export default function LandingPage({ notice, onAccount }: Props) {
  const [menuOpen, setMenuOpen] = useState(false);

  function goToAccount(mode: AccountMode) {
    setMenuOpen(false);
    onAccount(mode);
  }

  return (
    <main className="landing">
      <header className="landing-nav">
        <a className="brand" href="#home" aria-label="Daymark home">
          <span className="brand-mark" aria-hidden="true">d</span>
          <span>daymark</span>
        </a>
        <button
          className="nav-menu-toggle"
          type="button"
          aria-label={menuOpen ? "Close navigation" : "Open navigation"}
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen(!menuOpen)}
        >
          <span /><span />
        </button>
        <nav className={`nav-links${menuOpen ? " nav-links-open" : ""}`} aria-label="Main navigation">
          <a href="#about" onClick={() => setMenuOpen(false)}>About</a>
          <a href="#how-it-works" onClick={() => setMenuOpen(false)}>How it works</a>
          <button className="nav-signin" type="button" onClick={() => goToAccount("signin")}>Sign in</button>
          <button className="button button-small" type="button" onClick={() => goToAccount("signup")}>Get started <span aria-hidden="true">↗</span></button>
        </nav>
      </header>

      {notice && <div className="landing-notice" role="status">{notice}</div>}

      <section className="hero section-wrap" id="home">
        <div className="hero-copy">
          <span className="eyebrow"><span className="eyebrow-dot" /> A calmer way to get things done</span>
          <h1>Make space for<br />your <span>next thing.</span></h1>
          <p className="hero-description">A thoughtful little home for your to-dos. Clear your mind, find your focus, and take things one step at a time.</p>
          <div className="hero-actions">
            <button className="button" type="button" onClick={() => goToAccount("signup")}>Start your free space <span aria-hidden="true">↗</span></button>
            <a className="text-link" href="#how-it-works">Take a look around <span aria-hidden="true">↓</span></a>
          </div>
          <div className="hero-proof"><span className="proof-check">✓</span> Private by design <span className="proof-divider" /> Just you and your list</div>
        </div>

        <div className="hero-art" aria-label="Illustration of a personal Daymark task list">
          <div className="art-leaf leaf-one" />
          <div className="art-leaf leaf-two" />
          <div className="art-dot dot-one" />
          <div className="art-dot dot-two" />
          <div className="preview-card">
            <div className="preview-topline"><span className="preview-date">MONDAY, A FRESH START</span><span className="preview-avatar">J</span></div>
            <h2>Good morning, Jamie.</h2>
            <p className="preview-subtitle">A few things, at your own pace.</p>
            <div className="preview-progress"><span /><span /><span /><span /><span /><small>2 of 5 done</small></div>
            <ul className="preview-tasks">
              <li className="preview-done"><span className="preview-check">✓</span><span>Water the balcony plants</span></li>
              <li className="preview-done"><span className="preview-check">✓</span><span>Send that thoughtful note</span></li>
              <li><span className="preview-empty" /><span>Make a proper lunch</span><span className="task-sun">✳</span></li>
              <li><span className="preview-empty" /><span>Step outside for a while</span></li>
              <li><span className="preview-empty" /><span>Read a few more pages</span></li>
            </ul>
            <div className="preview-footer"><span>✳</span> Small steps count, too.</div>
          </div>
          <div className="floating-note"><span className="floating-icon">✦</span><span><strong>One thing at a time</strong><small>You're doing just fine.</small></span></div>
        </div>
      </section>

      <div className="trust-strip">
        <span>MADE FOR REAL LIFE</span><i /><span>Your own private space</span><i /><span>No pressure to be perfect</span><i /><span>Just one good next step</span>
      </div>

      <section className="about-section section-wrap" id="about">
        <Reveal className="section-intro">
          <span className="eyebrow">A SMALLER KIND OF PRODUCTIVITY</span>
          <h2>Less keeping up.<br /><span>More feeling good.</span></h2>
          <p>Daymark gives your thoughts a place to land, without turning your whole day into a project.</p>
        </Reveal>
        <div className="about-cards">
          <Reveal className="about-card reveal-delay-one">
            <span className="about-icon icon-sprout" aria-hidden="true">✳</span>
            <h3>Gentle by nature</h3>
            <p>No streaks, scores, or guilt. Your list is here to help, not hurry you.</p>
          </Reveal>
          <Reveal className="about-card reveal-delay-two">
            <span className="about-icon icon-shield" aria-hidden="true">⌑</span>
            <h3>Yours alone</h3>
            <p>Your tasks belong to your account. A quiet corner that stays yours.</p>
          </Reveal>
          <Reveal className="about-card reveal-delay-three">
            <span className="about-icon icon-sun" aria-hidden="true">☼</span>
            <h3>Clear and simple</h3>
            <p>Keep the next step close. Let everything else wait until it needs you.</p>
          </Reveal>
        </div>
      </section>

      <section className="how-section" id="how-it-works">
        <div className="section-wrap how-layout">
          <Reveal className="how-intro">
            <span className="eyebrow">HOW IT WORKS</span>
            <h2>Three small steps.<br />A little more <span>headspace.</span></h2>
            <p>Getting started is easy. Staying kind to yourself is the whole point.</p>
            <button className="button" type="button" onClick={() => goToAccount("signup")}>Find your focus <span aria-hidden="true">↗</span></button>
          </Reveal>
          <div className="steps-list">
            {steps.map((step, index) => (
              <Reveal className={`step reveal-delay-${index + 1}`} key={step.number}>
                <span className="step-number">{step.number}</span>
                <div><h3>{step.title}</h3><p>{step.detail}</p></div>
                <span className="step-mark" aria-hidden="true">{index === 0 ? "○" : index === 1 ? "＋" : "✓"}</span>
              </Reveal>
            ))}
          </div>
        </div>
      </section>

      <section className="closing-section section-wrap">
        <Reveal className="closing-card">
          <span className="closing-spark" aria-hidden="true">✳</span>
          <span className="eyebrow">YOUR NEXT CHAPTER CAN BE SMALL</span>
          <h2>Start where you are.</h2>
          <p>Make a little space for the things that matter to you.</p>
          <button className="button button-light" type="button" onClick={() => goToAccount("signup")}>Create your free account <span aria-hidden="true">↗</span></button>
        </Reveal>
      </section>

      <footer className="landing-footer section-wrap">
        <a className="brand" href="#home">
          <span className="brand-mark" aria-hidden="true">d</span>
          <span>daymark</span>
        </a>
        <p>A little more room for what matters.</p>
        <a href="#home" className="footer-top">Back to top <span aria-hidden="true">↑</span></a>
        <div className="footer-bottom"><span>© {new Date().getFullYear()} Daymark</span><span>Made for your own pace.</span></div>
      </footer>
    </main>
  );
}
