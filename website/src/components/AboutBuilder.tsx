export function AboutBuilder() {
  return (
    <section className="about-builder" aria-labelledby="builder-title">
      <span className="home-section-index">The person behind ORIX</span>
      <h2 id="builder-title">Built by Christopher Joshy.</h2>
      <p>
        A full-stack, systems and mobile developer who enjoys building practical
        software, from Flutter apps to real-time multiplayer systems. ORIX
        brings that interest in connecting people to everyday journeys.
      </p>
      <blockquote>
        “Life is like music. Its highs and lows are what make it beautiful.”
      </blockquote>
      <p className="builder-signature">Christopher Joshy</p>
      <nav aria-label="Christopher Joshy’s profiles">
        <a
          href="https://github.com/ChristopherJoshy/"
          target="_blank"
          rel="noreferrer"
        >
          GitHub ↗
        </a>
        <a
          href="https://www.linkedin.com/in/christopher-joshy-272a77290/"
          target="_blank"
          rel="noreferrer"
        >
          LinkedIn ↗
        </a>
      </nav>
    </section>
  );
}
