import { SceneArt } from "../components/SceneArt";
import "./Auth.css";
import { AccountDeletion } from "../components/AccountDeletion";
import { AboutBuilder } from "../components/AboutBuilder";
import { Link, useParams } from "react-router-dom";
import { ArrowLeft, ArrowUpRight, RefreshCw } from "lucide-react";
import { Button, LinkButton, Logo } from "../components/ui";
import { PixelCity, Rickshaw } from "../components/PixelArt";
const content: Record<
  string,
  { title: string; intro: string; sections: [string, string][] }
> = {
  about: {
    title: "A little company. A better way.",
    intro: "ORIX helps people going the same way find each other.",
    sections: [
      [
        "Share the everyday.",
        "Set your route, time and preferences. ORIX automatically finds a compatible pool. To offer a lift, register your route and availability separately. ORIX coordinates people; it does not operate taxis or process fares.",
      ],
      [
        "Built around your comfort.",
        "Google sign-in, safety preferences, blocking, reporting and support help you manage your journey. Account, journey, group, support and moderation screens use the connected Orix service.",
      ],
    ],
  },
  privacy: {
    title: "Your journey. Your privacy.",
    intro: "How Orix uses your account and travel information.",
    sections: [
      [
        "Information used by Orix",
        "Google sign-in verifies your account through Firebase. Account setup sends your phone number, optional gender, profile photo and safety acknowledgements to the Orix API. Location and notification permissions are optional. Analytics is not enabled by default.",
      ],
      [
        "Your controls",
        "You can decline optional permissions and choose places manually. Browser settings let you revoke permissions or clear browser preferences. Account and journey changes are sent to the connected Orix service. Contact Christopherjoshy4@gmail.com for account or privacy questions.",
      ],
      [
        "Before a public launch",
        "The operator must publish a complete privacy policy covering the real service, processors, location use, retention, account deletion, contacts, and user rights. This preview notice is not the final service policy.",
      ],
    ],
  },
  terms: {
    title: "Good company starts with care.",
    intro: "A few things to keep the journey kind.",
    sections: [
      [
        "Respect the people you meet.",
        "Agree on your plan before leaving. Meet in a safe public place. Respect others’ boundaries and safety preferences. Harassment, discrimination, and misuse of personal information have no place here.",
      ],
      [
        "Know what ORIX does.",
        "ORIX helps people coordinate shared journeys. It does not provide transport, guarantee a match, verify every vehicle, collect fares, or act as an emergency service.",
      ],
      [
        "Make an informed plan",
        "Check the people, vehicle and pickup point before travelling. You decide whether to confirm a group. Report concerns privately and contact local emergency services when immediate help is needed.",
      ],
    ],
  },
  "account-deletion": {
    title: "Your account. Your decision.",
    intro: "You should be able to leave as easily as you joined.",
    sections: [
      [
        "Delete a real account",
        "Account deletion requires a recent Google sign-in. Use the form below to send your deletion request to the connected Orix service.",
      ],
      [
        "What needs to happen",
        "The service must explain which profile, ride, chat, and authentication records are deleted, any legally required retention, and how deletion progress is confirmed. A queued request must never be presented as completed deletion.",
      ],
      [
        "Need help?",
        "Contact Christopherjoshy4@gmail.com if you cannot access the account you want to delete.",
      ],
    ],
  },
};
export function PublicInfo({ page }: { page: string }) {
  const data = content[page] || content.about;
  return (
    <div className="public-page">
      <header className="landing-nav">
        <Logo />
        <Link className="text-link" to="/">
          Back to ORIX <ArrowLeft size={16} />
        </Link>
      </header>
      <main id="main" className="public-info">
        {page === "about" ? (
          <SceneArt scene="companions" className="public-scene" />
        ) : (
          <Rickshaw />
        )}
        <h1>{data.title}</h1>
        <p className="intro">{data.intro}</p>
        {data.sections.map(([title, text]) => (
          <section key={title}>
            <h2>{title}</h2>
            <p>{text}</p>
          </section>
        ))}
        {page === "about" && <AboutBuilder />}
        {page === "account-deletion" && <AccountDeletion />}
        <LinkButton to="/login">
          Get started <ArrowUpRight size={17} />
        </LinkButton>
      </main>
    </div>
  );
}
const statuses: Record<string, [string, string]> = {
  maintenance: [
    "A little pause. A better journey.",
    "ORIX is taking a moment to get things ready. Please check back shortly.",
  ],
  offline: [
    "A little off the grid.",
    "Check your connection and try again. Your next journey can wait a moment.",
  ],
  suspended: [
    "Let’s take a little pause.",
    "Your access may be temporarily restricted. Contact support to understand the next step or make an appeal.",
  ],
  banned: [
    "A moment to make things right.",
    "If your account is restricted, the support team can help you understand your options and submit an appeal.",
  ],
  update: [
    "A little newer. A little better.",
    "An updated version is needed to continue. Check the official app listing or reconnect to the website.",
  ],
};
export function StatusPage() {
  const { kind } = useParams();
  const data = statuses[kind || ""] || [
    "A little detour.",
    "We couldn’t find that page. Let’s get you back on your way.",
  ];
  return (
    <main className="status-page" id="main">
      <Logo />
      <div>
        <Rickshaw />
        <h1>{data[0]}</h1>
        <p>{data[1]}</p>
        <div className="hero-actions">
          <LinkButton to="/">
            Back to ORIX <ArrowLeft size={17} />
          </LinkButton>
          {kind === "offline" ? (
            <Button
              variant="secondary"
              onClick={() => window.location.reload()}
            >
              <RefreshCw size={17} />
              Try again
            </Button>
          ) : (
            <LinkButton to="/app/support" secondary>
              Get support <ArrowUpRight size={17} />
            </LinkButton>
          )}
        </div>
        {kind && (
          <small>
            This is a preview of the {kind} screen, not a live account or
            service status.
          </small>
        )}
      </div>
      <PixelCity compact />
    </main>
  );
}
