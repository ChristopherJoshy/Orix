import { AboutBuilder } from "../components/AboutBuilder";
import { Link } from "react-router-dom";
import { Logo } from "../components/ui";
import "./Landing.css";

function PixelArrow() {
  return (
    <svg
      width="16"
      height="16"
      viewBox="0 0 16 16"
      fill="currentColor"
      aria-hidden="true"
    >
      <path d="M3 3h10v10h-2V7l-7 7-2-2 7-7H3z" />
    </svg>
  );
}

const steps = [
  [
    "Start with your route.",
    "Choose your starting point, destination and departure time. We’ll look for people with nearby starts and the same destination.",
  ],
  [
    "Make it comfortable.",
    "Set your preferences, including women-only matching and whether you’re open to a lift. ORIX brings compatible requests together.",
  ],
  [
    "Meet before you move.",
    "See who’s in your group, agree on a pickup in the group chat, and confirm your plan together.",
  ],
];
const questions = [
  [
    "Do I need to choose a travel mode?",
    "No. Share your route and preferences. ORIX matches compatible people into a pool, whether you share an auto, find travel companions, or accept an available lift.",
  ],
  [
    "How close should our starting points be?",
    "Matching considers starting points within 300 metres of your selected area. Your destination must be the same, and your departure windows and safety preferences must align.",
  ],
  [
    "What if a group isn’t right for me?",
    "You can reject the group and search again. The people you rejected are excluded from matching for that journey. You can also block a person or report a concern privately.",
  ],
  [
    "Can I offer a lift?",
    "Offer a seat for a specific journey with your route, departure time and available seats. Recurring lift availability is not available yet.",
  ],
];

export default function Landing() {
  return (
    <div className="orix-home">
      <header className="home-header home-width">
        <Logo />
        <nav aria-label="Main navigation">
          <a href="#how-it-works">How it works</a>
          <a href="#your-comfort">Your comfort</a>
        </nav>
        <Link className="home-login" to="/login">
          Log in <PixelArrow />
        </Link>
      </header>
      <main id="main">
        <section className="home-hero home-width" aria-labelledby="home-title">
          <div className="home-hero-copy">
            <p className="home-intro">
              <span aria-hidden="true" /> Shared autos, travel partners and
              lifts
            </p>
            <h1 id="home-title">
              Your route.
              <br />
              Good company.
            </h1>
            <p className="home-lede">
              Find people going the same way.
              <br className="home-desktop-break" /> Share an auto, meet a travel
              companion,
              <br className="home-desktop-break" /> or offer a seat.
            </p>
            <div className="home-actions">
              <Link className="home-button" to="/login">
                Plan a journey <PixelArrow />
              </Link>
              <a className="home-text-link" href="#how-it-works">
                How it works <span aria-hidden="true">↓</span>
              </a>
            </div>
          </div>
          <figure className="home-hero-art">
            <img
              src="/art/orix-street-corner.png"
              width="1122"
              height="1402"
              fetchPriority="high"
              alt="Pixel illustration of two travellers meeting beside an auto-rickshaw on a palm-lined Kerala street."
            />
            <figcaption>Everyday routes. Shared stories.</figcaption>
          </figure>
        </section>
        <div className="home-route-note home-width">
          <span className="home-route-glyph" aria-hidden="true">
            <i />
            <b />
            <i />
          </span>
          <p>
            Nearby starts.
            <br className="home-mobile-break" /> The same destination.
          </p>
          <span>Made for going together.</span>
        </div>
        <section
          className="home-how home-width home-section"
          id="how-it-works"
          aria-labelledby="how-title"
        >
          <div className="home-section-title">
            <span className="home-section-index">How it works</span>
            <h2 id="how-title">
              A shared route.
              <br />A simple plan.
            </h2>
            <p>
              You bring the destination.
              <br />
              We help you find the company.
            </p>
          </div>
          <ol className="home-steps">
            {steps.map(([title, text], index) => (
              <li key={title}>
                <span className="home-step-number" aria-hidden="true">
                  0{index + 1}
                </span>
                <div>
                  <h3>{title}</h3>
                  <p>{text}</p>
                </div>
              </li>
            ))}
          </ol>
        </section>
        <section
          className="home-comfort"
          id="your-comfort"
          aria-labelledby="comfort-title"
        >
          <div className="home-width home-comfort-inner">
            <div>
              <span className="home-section-index">Your comfort</span>
              <h2 id="comfort-title">
                Good company
                <br />
                starts with
                <br />
                your comfort.
              </h2>
            </div>
            <div className="home-comfort-copy">
              <p className="home-comfort-lede">
                Get to know your group.
                <br />
                Then decide to go.
              </p>
              <ul>
                <li>See names and profile photos before confirming.</li>
                <li>Choose women-only matching when you need it.</li>
                <li>Pass on a group and look for another fit.</li>
                <li>Block or report a concern privately.</li>
              </ul>
              <Link className="home-inline-link" to="/login">
                Make your first plan <PixelArrow />
              </Link>
            </div>
          </div>
        </section>
        <section
          className="home-lift home-width home-section"
          aria-labelledby="lift-title"
        >
          <div>
            <span className="home-section-index">Offer a lift</span>
            <h2 id="lift-title">
              An empty seat.
              <br />
              Someone’s way home.
            </h2>
          </div>
          <div>
            <p>
              Already heading that way? Register your route, time and available
              seats. Your lift can become part of someone else’s everyday
              journey.
            </p>
            <Link className="home-text-link" to="/login?next=/app/rider">
              Offer a lift <PixelArrow />
            </Link>
          </div>
        </section>
        <section
          className="home-faq home-width home-section"
          aria-labelledby="faq-title"
        >
          <div>
            <span className="home-section-index">A few things to know</span>
            <h2 id="faq-title">Before you go.</h2>
          </div>
          <div>
            {questions.map(([question, answer]) => (
              <details key={question}>
                <summary>
                  {question}
                  <span className="home-faq-toggle" aria-hidden="true" />
                </summary>
                <p>{answer}</p>
              </details>
            ))}
          </div>
        </section>
        <section className="home-end home-width">
          <p>Same direction?</p>
          <h2>Let’s go together.</h2>
          <Link className="home-button" to="/login">
            Plan a journey <PixelArrow />
          </Link>
        </section>
      </main>
      <div className="home-width">
        <AboutBuilder />
      </div>
      <footer className="home-footer home-width">
        <Logo />
        <p>Less alone. More along.</p>
        <nav aria-label="Footer navigation">
          <Link to="/privacy">Privacy</Link>
          <Link to="/terms">Terms</Link>
          <a href="mailto:Christopherjoshy4@gmail.com">Contact</a>
        </nav>
        <span>© {new Date().getFullYear()} ORIX</span>
      </footer>
    </div>
  );
}
