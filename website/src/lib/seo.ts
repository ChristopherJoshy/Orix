export const siteOrigin = "https://orixrideshare.tech";

export const publicPages: Record<
  string,
  { title: string; description: string }
> = {
  "/": {
    title: "ORIX Ride Share | Shared Autos, Travel Partners & Lifts",
    description:
      "Ride share with ORIX: find travel partners going the same way, coordinate a shared auto, or offer a lift. Match routes, departure times and safety preferences.",
  },
  "/about": {
    title: "How ORIX Ride Sharing Works | Routes, Groups & Safety",
    description:
      "Learn how ORIX ride sharing connects travel companions on compatible routes for shared autos and one-time lifts, with Google sign-in and safety preferences.",
  },
  "/download": {
    title: "Download ORIX for Android | Shared Travel App",
    description:
      "Get the Android app from ORIX’s official download page. Check the latest available release, or use ORIX on the web to coordinate your next shared journey.",
  },
  "/privacy": {
    title: "Privacy Notice | ORIX",
    description:
      "Read ORIX’s current privacy notice about account and journey information, optional location and notification permissions, and privacy contact details.",
  },
  "/terms": {
    title: "Community & Travel Terms | ORIX",
    description:
      "Read ORIX’s shared travel guidelines: respect fellow travellers, agree on a safe pickup, understand the service, and report concerns privately.",
  },
  "/account-deletion": {
    title: "Request Account Deletion | ORIX",
    description:
      "Request deletion of your ORIX account with a recent Google sign-in. Find the account deletion form and support contact on the official ORIX website.",
  },
};

export function pageSchema(path: string) {
  const page = publicPages[path];
  if (!page) return null;
  const website = `${siteOrigin}/#website`;
  const publisher = `${siteOrigin}/#organization`;
  const graph: Record<string, unknown>[] = [
    {
      "@type": path === "/about" ? "AboutPage" : "WebPage",
      "@id": `${siteOrigin}${path}#webpage`,
      url: `${siteOrigin}${path}`,
      name: page.title,
      description: page.description,
      isPartOf: { "@id": website },
      inLanguage: "en",
    },
  ];
  if (path === "/")
    graph.push(
      {
        "@type": "WebSite",
        "@id": website,
        url: `${siteOrigin}/`,
        name: "ORIX",
        alternateName: "ORIX Rideshare",
        publisher: { "@id": publisher },
        inLanguage: "en",
      },
      {
        "@type": "Organization",
        "@id": publisher,
        name: "ORIX",
        url: `${siteOrigin}/`,
        logo: `${siteOrigin}/orix-logo.svg`,
      },
    );
  else
    graph.push({
      "@type": "BreadcrumbList",
      itemListElement: [
        {
          "@type": "ListItem",
          position: 1,
          name: "ORIX",
          item: `${siteOrigin}/`,
        },
        {
          "@type": "ListItem",
          position: 2,
          name: page.title.split(" | ")[0],
          item: `${siteOrigin}${path}`,
        },
      ],
    });
  return { "@context": "https://schema.org", "@graph": graph };
}

/** Keep SPA navigation consistent with the metadata served in each public HTML file. */
export function updatePageMetadata(path: string) {
  const page = publicPages[path];
  const meta = (
    selector: string,
    attributes: Record<string, string>,
    content: string,
  ) => {
    let element = document.head.querySelector<HTMLMetaElement>(selector);
    if (!element) {
      element = document.createElement("meta");
      for (const [key, value] of Object.entries(attributes))
        element.setAttribute(key, value);
      document.head.append(element);
    }
    element.content = content;
  };
  meta(
    'meta[name="robots"]',
    { name: "robots" },
    page ? "index, follow, max-image-preview:large" : "noindex, follow",
  );
  meta(
    'meta[name="description"]',
    { name: "description" },
    page?.description ?? "Sign in to manage your ORIX journeys and account.",
  );
  for (const [key, value] of Object.entries({
    title: page?.title ?? "ORIX",
    description: page?.description ?? "",
    url: page ? `${siteOrigin}${path}` : "",
  })) {
    meta(`meta[property="og:${key}"]`, { property: `og:${key}` }, value);
    if (key !== "url")
      meta(`meta[name="twitter:${key}"]`, { name: `twitter:${key}` }, value);
  }
  document.querySelector('link[rel="canonical"]')?.remove();
  if (page) {
    const canonical = document.createElement("link");
    canonical.rel = "canonical";
    canonical.href = `${siteOrigin}${path}`;
    document.head.append(canonical);
  }
  document.getElementById("orix-structured-data")?.remove();
  const schema = pageSchema(path);
  if (schema) {
    const script = document.createElement("script");
    script.id = "orix-structured-data";
    script.type = "application/ld+json";
    script.textContent = JSON.stringify(schema);
    document.head.append(script);
  }
}
