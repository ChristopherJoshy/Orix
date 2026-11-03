<p align="center"><img src="assets/images/orix_logo.svg" alt="ORIX" width="160"></p>

# ORIX app

Plan a journey, meet people travelling the same way, and coordinate the trip
with your group. The app uses Google sign-in and the ORIX API.

<p align="center"><img src="assets/images/orix_app_welcome.png" alt="Travelers meeting beside the water" width="360"></p>

## Run locally

1. Install the Flutter and Android tooling for your development machine.
2. Copy `example.env` to `.env` and set the API origin and map configuration.
   `10.0.2.2` reaches the host from an Android emulator; release builds need
   the deployed HTTPS API origin.
3. Add the Firebase Android configuration for your project at
   `android/app/google-services.json`. This file stays out of Git.
4. Run `flutter pub get`, then `flutter run --dart-define-from-file=.env`.

The API and its place-search and routing providers must be configured for their
features to work. Google sign-in also requires the Android signing fingerprints
to be registered with Firebase. See [environment setup](../Docs/environment-setup.md)
and [current integration status](../Docs/integration-progress.md).
