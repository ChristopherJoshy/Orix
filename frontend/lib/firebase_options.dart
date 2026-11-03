/// Public Firebase client identifiers supplied by the project owner.
import 'package:firebase_core/firebase_core.dart' show FirebaseOptions;
import 'package:flutter/foundation.dart'
    show defaultTargetPlatform, kIsWeb, TargetPlatform;

class DefaultFirebaseOptions {
  static FirebaseOptions get currentPlatform {
    if (kIsWeb) return web;
    if (defaultTargetPlatform == TargetPlatform.android) return android;
    throw UnsupportedError('Firebase is configured for Android and web only.');
  }

  static const android = FirebaseOptions(
    apiKey: "AIzaSyAyvwB8r_vapTOFQWGdFxu0yCMCQ9BWvrk",
    appId: "1:130898233956:android:8ad780cf27bd24a1b52d71",
    messagingSenderId: "130898233956",
    projectId: "orixrideshare",
    storageBucket: "orixrideshare.firebasestorage.app",
  );

  static const web = FirebaseOptions(
    apiKey: "AIzaSyDMu49MmVbSJvFo7phY4WIWw7NP60Y50rs",
    authDomain: "orixrideshare.firebaseapp.com",
    projectId: "orixrideshare",
    storageBucket: "orixrideshare.firebasestorage.app",
    messagingSenderId: "130898233956",
    appId: "1:130898233956:web:f90d66ca77df002ab52d71",
    measurementId: "G-8VM3CWPH8D",
  );
}
