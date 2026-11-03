/// Authentication Service
///
/// Handles Firebase authentication and Google Sign-In (v7.x API).
library;

import 'dart:async';

import '../config/api_config.dart';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/foundation.dart' show kIsWeb;
import 'package:firebase_core/firebase_core.dart';
import 'package:google_sign_in/google_sign_in.dart';

/// Result of authentication
class AuthResult {
  final bool success;
  final String? error;
  final String? errorCode;
  final User? user;

  AuthResult({required this.success, this.error, this.errorCode, this.user});

  factory AuthResult.success(User user) =>
      AuthResult(success: true, user: user);

  factory AuthResult.failure(String error, {String? code}) =>
      AuthResult(success: false, error: error, errorCode: code);
}

/// Authentication service using google_sign_in v7 API
class AuthService {
  final FirebaseAuth _auth = InternalTestConfig.enabled
      ? FirebaseAuth.instanceFor(app: Firebase.app('orix-internal'))
      : FirebaseAuth.instance;

  bool _initialized = false;
  StreamSubscription<GoogleSignInAuthenticationEvent>? _authEventSubscription;
  final Completer<void> _initCompleter = Completer<void>();

  /// Current Firebase user
  User? get currentUser => _auth.currentUser;

  /// Stream of auth state changes
  Stream<User?> get authStateChanges => _auth.authStateChanges();

  /// Check if user is signed in
  bool get isSignedIn => currentUser != null;

  /// Get current user's email
  String? get userEmail => currentUser?.email;

  /// Get Firebase ID token for backend authentication
  Future<String?> getIdToken({bool forceRefresh = false}) async {
    try {
      return await currentUser?.getIdToken(forceRefresh);
    } catch (e) {
      return null;
    }
  }

  /// Initialize Google Sign-In (must be called once at app startup)
  Future<void> initialize() async {
    if (_initialized) return;
    if (InternalTestConfig.enabled) {
      _initialized = true;
      _initCompleter.complete();
      return;
    }

    try {
      // Android may derive this from google-services.json. Other builds can
      // supply their own public OAuth web client ID at build time.
      const serverClientId = String.fromEnvironment('GOOGLE_SERVER_CLIENT_ID');
      await GoogleSignIn.instance.initialize(
        serverClientId: serverClientId.isEmpty ? null : serverClientId,
      );
      _initialized = true;

      // Listen for authentication events
      _authEventSubscription = GoogleSignIn.instance.authenticationEvents
          .listen(_handleAuthEvent, onError: _handleAuthError);

      // NOTE: Removed attemptLightweightAuthentication() call
      // This was triggering the Google account picker popup on app startup
      // which confused users. The popup should only appear when user
      // explicitly taps "Continue with Google" button.

      if (!_initCompleter.isCompleted) {
        _initCompleter.complete();
      }
    } catch (e) {
      if (!_initCompleter.isCompleted) {
        _initCompleter.completeError(e);
      }
    }
  }

  void _handleAuthEvent(GoogleSignInAuthenticationEvent event) {
    // Events handled via Firebase Auth state changes
    // The GoogleSignInAccount is used to create Firebase credential
  }

  void _handleAuthError(Object error) {
    // Authentication errors are handled in signInWithGoogle
  }

  /// Ensure initialization is complete
  Future<void> _ensureInitialized() async {
    if (!_initialized) {
      await initialize();
    }
    await _initCompleter.future;
  }

  /// Sign in with Google (v7 API)
  Future<AuthResult> signInWithGoogle() async {
    try {
      if (kIsWeb && !InternalTestConfig.enabled) {
        final result = await _auth.signInWithPopup(GoogleAuthProvider());
        return result.user == null
            ? AuthResult.failure('Failed to sign in to Firebase')
            : AuthResult.success(result.user!);
      }
      await _ensureInitialized();

      // Check if authenticate is supported on this platform
      if (!GoogleSignIn.instance.supportsAuthenticate()) {
        return AuthResult.failure(
          'Google Sign-In not supported on this platform',
          code: 'unsupported_platform',
        );
      }

      // Trigger the Google Sign-In flow
      final GoogleSignInAccount googleUser = await GoogleSignIn.instance
          .authenticate();

      // Get authentication tokens (v7 uses authentication getter)
      final String? idToken = googleUser.authentication.idToken;

      if (idToken == null) {
        return AuthResult.failure('Failed to get ID token');
      }

      // Create Firebase credential using ID token
      // Note: v7 separates authentication from authorization
      // For Firebase Auth, we only need the idToken
      final credential = GoogleAuthProvider.credential(idToken: idToken);

      // Sign in to Firebase
      final userCredential = await _auth.signInWithCredential(credential);

      if (userCredential.user == null) {
        return AuthResult.failure('Failed to sign in to Firebase');
      }

      return AuthResult.success(userCredential.user!);
    } on GoogleSignInException catch (e) {
      // Handle specific Google Sign-In exceptions
      if (e.code == GoogleSignInExceptionCode.canceled) {
        return AuthResult.failure('Sign-in cancelled', code: 'cancelled');
      }
      return AuthResult.failure(
        e.description ?? 'Google Sign-In failed',
        code: e.code.toString(),
      );
    } on FirebaseAuthException catch (e) {
      return AuthResult.failure(
        e.message ?? 'Authentication failed',
        code: e.code,
      );
    } catch (e) {
      // Log for debugging but show user-friendly message
      return AuthResult.failure('Sign-in failed. Please try again.');
    }
  }

  Future<AuthResult> signInInternal(String email, String password) async {
    if (!InternalTestConfig.enabled) {
      return AuthResult.failure('Internal login is unavailable');
    }
    try {
      final credential = await _auth.signInWithEmailAndPassword(
        email: email,
        password: password,
      );
      final user = credential.user;
      if (user == null) return AuthResult.failure('Account sign-in failed');
      return AuthResult.success(user);
    } on FirebaseAuthException {
      return AuthResult.failure(
        'Check the test account credentials and emulator',
      );
    }
  }

  /// Verify phone number using Firebase Auth
  Future<void> verifyPhoneNumber({
    required String phoneNumber,
    required PhoneVerificationCompleted onVerificationCompleted,
    required PhoneVerificationFailed onVerificationFailed,
    required PhoneCodeSent onCodeSent,
    required PhoneCodeAutoRetrievalTimeout onCodeAutoRetrievalTimeout,
  }) async {
    await _auth.verifyPhoneNumber(
      phoneNumber: phoneNumber,
      verificationCompleted: onVerificationCompleted,
      verificationFailed: onVerificationFailed,
      codeSent: onCodeSent,
      codeAutoRetrievalTimeout: onCodeAutoRetrievalTimeout,
      timeout: const Duration(seconds: 60),
    );
  }

  /// Sign out from both Firebase and Google
  Future<void> signOut() async {
    await _auth.signOut();
    if (_initialized && !InternalTestConfig.enabled) {
      await GoogleSignIn.instance.signOut();
    }
  }

  /// Reauthenticate the existing Firebase user before a sensitive API action.
  Future<bool> reauthenticateGoogle({String? emulatorPassword}) async {
    final user = currentUser;
    if (user == null) return false;
    try {
      if (InternalTestConfig.enabled) {
        final email = user.email;
        if (email == null ||
            emulatorPassword == null ||
            emulatorPassword.isEmpty)
          return false;
        await user.reauthenticateWithCredential(
          EmailAuthProvider.credential(
            email: email,
            password: emulatorPassword,
          ),
        );
        await user.getIdToken(true);
        return currentUser?.uid == user.uid;
      }
      if (kIsWeb) {
        await user.reauthenticateWithPopup(GoogleAuthProvider());
        await user.getIdToken(true);
        return currentUser?.uid == user.uid;
      }
      await _ensureInitialized();
      if (!GoogleSignIn.instance.supportsAuthenticate()) return false;
      final account = await GoogleSignIn.instance.authenticate();
      final token = account.authentication.idToken;
      if (token == null || currentUser?.uid != user.uid) return false;
      await user.reauthenticateWithCredential(
        GoogleAuthProvider.credential(idToken: token),
      );
      await user.getIdToken(true);
      return currentUser?.uid == user.uid;
    } catch (_) {
      return false;
    }
  }

  /// Disconnect - revokes access and signs out
  Future<void> disconnect() async {
    await _auth.signOut();
    if (_initialized && !InternalTestConfig.enabled) {
      await GoogleSignIn.instance.disconnect();
    }
  }

  /// Delete account
  Future<bool> deleteAccount() async {
    try {
      await currentUser?.delete();
      if (_initialized && !InternalTestConfig.enabled) {
        await GoogleSignIn.instance.disconnect();
      }
      return true;
    } catch (e) {
      return false;
    }
  }

  /// Dispose resources
  void dispose() {
    _authEventSubscription?.cancel();
  }
}
