// Firebase's official modular CDN release, pinned from its release notes.
const sdkOrigin = 'https://www.gstatic.com/firebasejs/13.1.0';
const signInButton = document.querySelector('#sign-in');
const signOutButton = document.querySelector('#sign-out');
const deleteButton = document.querySelector('#delete-account');
const confirmation = document.querySelector('#confirm-deletion');
const panel = document.querySelector('#account-panel');
const email = document.querySelector('#account-email');
const status = document.querySelector('#status');
let auth;
let firebase;
let busy = false;
let submitted = false;

function showAccount(user) {
  panel.hidden = !user;
  signInButton.hidden = Boolean(user) || submitted;
  email.textContent = user?.email ?? '';
  confirmation.checked = false;
  deleteButton.disabled = true;
}
function setBusy(value) {
  busy = value;
  signInButton.disabled = value || submitted;
  signOutButton.disabled = value;
  confirmation.disabled = value;
  deleteButton.disabled = value || !confirmation.checked || submitted;
}
function signInMessage(error) {
  switch (error?.code) {
    case 'auth/popup-closed-by-user': return 'Sign-in was cancelled. You can try again.';
    case 'auth/popup-blocked': return 'Allow the Google sign-in popup in your browser, then try again.';
    case 'auth/user-disabled': return 'This sign-in is disabled. Use the support contact below to request deletion.';
    case 'auth/network-request-failed': return 'Google sign-in could not connect. Check your connection and try again.';
    default: return 'Google sign-in is unavailable. Try again or use the support contact below.';
  }
}
confirmation.addEventListener('change', () => {
  deleteButton.disabled = busy || !confirmation.checked || submitted;
});
signInButton.addEventListener('click', async () => {
  if (busy || !auth) return;
  setBusy(true);
  try {
    const provider = new firebase.GoogleAuthProvider();
    provider.setCustomParameters({prompt: 'select_account'});
    const result = await firebase.signInWithPopup(auth, provider);
    showAccount(result.user);
    status.textContent = 'Check the signed-in account and confirm below to request deletion.';
  } catch (error) {
    status.textContent = signInMessage(error);
  } finally {
    setBusy(false);
  }
});
signOutButton.addEventListener('click', async () => {
  if (busy) return;
  setBusy(true);
  try {
    await firebase.signOut(auth);
    showAccount(null);
    status.textContent = 'Sign in with the ORIX account you want to delete.';
  } catch {
    status.textContent = 'Could not clear this sign-in. Close this page before using another account.';
  } finally {
    setBusy(false);
  }
});
deleteButton.addEventListener('click', async () => {
  if (busy || submitted || !confirmation.checked || !auth?.currentUser) return;
  setBusy(true);
  status.textContent = 'Submitting your deletion request…';
  try {
    // Keep the token in memory and headers only, never URLs or browser storage.
    const token = await auth.currentUser.getIdToken(true);
    const response = await fetch('/api/v1/users/me', {
      method: 'DELETE',
      headers: {Authorization: `Bearer ${token}`},
      credentials: 'omit',
      cache: 'no-store',
      redirect: 'error',
      signal: AbortSignal.timeout(20000),
    });
    if (response.status === 202) {
      submitted = true;
      showAccount(null);
      // A sign-out failure must not turn an accepted request into a false error.
      try { await firebase.signOut(auth); } catch { /* Session is memory-only. */ }
      status.textContent = 'Your deletion request was accepted. Your active trips are cancelled and account cleanup will run in the background. You can close this page.';
    } else if (response.status === 401) {
      await firebase.signOut(auth);
      showAccount(null);
      status.textContent = 'For your security, sign in again before requesting deletion.';
    } else if (response.status === 429) {
      status.textContent = 'Too many requests. Wait a minute, then try again.';
    } else {
      status.textContent = 'ORIX could not accept the request. Try again or use the support contact below.';
    }
  } catch {
    status.textContent = 'The response could not be confirmed. Your request may have reached ORIX. Retry or contact support if you are unsure.';
  } finally {
    setBusy(false);
  }
});

async function initialize() {
  try {
    const response = await fetch('/account-deletion/config', {credentials: 'omit', cache: 'no-store', redirect: 'error'});
    if (!response.ok) throw new Error('Configuration unavailable');
    const config = await response.json();
    const [{initializeApp}, authModule] = await Promise.all([
      import(`${sdkOrigin}/firebase-app.js`),
      import(`${sdkOrigin}/firebase-auth.js`),
    ]);
    firebase = authModule;
    auth = firebase.getAuth(initializeApp(config, 'orix-account-deletion'));
    await firebase.setPersistence(auth, firebase.inMemoryPersistence);
    showAccount(null);
    setBusy(false);
    status.textContent = 'Sign in to request deletion. Nothing is deleted until you confirm.';
  } catch {
    signInButton.disabled = true;
    status.textContent = 'Web sign-in is unavailable. Try again later or use the support contact below.';
  }
}
initialize();
