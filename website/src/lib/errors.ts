/** Only messages authored for the interface may be shown to people. */
export class PublicError extends Error {
  readonly userFacing = true;
}
export function statusMessage(status: number) {
  switch (status) {
    case 400:
    case 422:
      return "A detail needs another look. Check your entries and try again.";
    case 401:
      return "Please sign in with Google again to continue.";
    case 403:
      return "This action isn’t available for your account. Contact support if you need help.";
    case 404:
      return "We couldn’t find that item. Refresh to check whether it is still available.";
    case 409:
      return "The plan has changed. Refresh before trying that again.";
    case 413:
      return "That file is a little too heavy for this auto. Choose a smaller one.";
    case 429:
      return "One at a time, please. Even the chaya needs a moment. Wait briefly, then retry.";
    default:
      return "That didn’t go through. The service is taking a chaya break. Try again shortly.";
  }
}
export function friendlyError(problem: unknown): string {
  if (problem instanceof PublicError) return problem.message;
  if (
    problem instanceof TypeError ||
    (problem instanceof DOMException && problem.name === "TimeoutError")
  )
    return "Connection lost. The network’s on a chaya break. Check your connection and retry.";
  return "That didn’t go through. Let’s give it another try.";
}
