/// Authentication Provider
///
/// Manages authentication state and user session.
library;

import 'dart:async';

import 'package:flutter/widgets.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:go_router/go_router.dart';

import '../services/auth_service.dart';
import '../services/api_service.dart';
import '../services/websocket_service.dart';
import '../services/notification_service.dart';
import '../router/app_router.dart';
import '../utils/error_helper.dart';
import '../utils/time_utils.dart';
import '../config/api_config.dart';

/// User status from backend
enum UserStatus { active, suspended, banned }

/// Backend user data
class BackendUser {
  final String occupation;
  final String userId;
  final String email;
  final String displayName;
  final String? photoUrl;
  final String? phone;
  final String? gender;
  final bool isRider;
  final bool onboardingCompleted;
  final UserStatus status;
  final String? banReason;
  final DateTime? suspensionExpiresAt;

  BackendUser({
    required this.userId,
    required this.email,
    required this.displayName,
    this.occupation = 'other',
    this.photoUrl,
    this.phone,
    this.gender,
    this.isRider = false,
    this.onboardingCompleted = false,
    this.status = UserStatus.active,
    this.banReason,
    this.suspensionExpiresAt,
  });

  factory BackendUser.fromJson(Map<String, dynamic> json) {
    String? photoUrl = json['photo_url'] as String?;
    if (photoUrl != null) {
      if (photoUrl.isEmpty) {
        photoUrl = null;
      } else if (photoUrl.startsWith('/')) {
        photoUrl = '${ApiConfig.baseUrl}$photoUrl';
      }
    }

    return BackendUser(
      userId: json['user_id'] as String,
      email: json['email'] as String,
      displayName: json['display_name'] as String,
      occupation: json['occupation'] as String? ?? 'other',
      photoUrl: photoUrl,
      phone: json['phone'] as String?,
      gender: json['gender'] as String?,
      isRider: json['is_rider'] as bool? ?? false,
      onboardingCompleted: json['onboarding_completed'] as bool? ?? false,
      status: _parseStatus(json['status'] as String?),
      banReason: json['ban_reason'] as String?,
      suspensionExpiresAt: json['suspension_expires_at'] != null
          ? TimeUtils.parseUtc(json['suspension_expires_at'] as String)
          : null,
    );
  }

  static UserStatus _parseStatus(String? status) {
    switch (status) {
      case 'suspended':
        return UserStatus.suspended;
      case 'banned':
        return UserStatus.banned;
      default:
        return UserStatus.active;
    }
  }
}

/// Authentication provider
class AuthProvider extends ChangeNotifier {
  final AuthService _authService;
  final ApiService _apiService;
  final WebSocketService _webSocketService;
  final NotificationService? _notificationService;

  Future<void> Function()? beforeSignOut;
  StreamSubscription<User?>? _authSubscription;

  bool _isLoading = true;
  bool _isAuthenticated = false;
  bool _hasNetworkError = false;
  // System Config

  User? _firebaseUser;
  BackendUser? _backendUser;
  String? _error;

  AuthProvider(
    this._authService,
    this._apiService,
    this._webSocketService, [
    this._notificationService,
  ]) {
    _apiService.setAuthService(_authService);
    _initErrorHelper();
    _init();
  }

  void _initErrorHelper() {
    ErrorHelper.getUserInfo = () {
      if (_backendUser != null) {
        return {
          'user_id': _backendUser!.userId,
          'email': _backendUser!.email,
          'display_name': _backendUser!.displayName,
        };
      } else if (_firebaseUser != null) {
        return {'user_id': _firebaseUser!.uid, 'email': _firebaseUser!.email};
      }
      return {};
    };

    ErrorHelper.getAuthToken = () async {
      return await _authService.getIdToken();
    };
  }

  // Getters
  bool get isLoading => _isLoading;
  bool get isAuthenticated => _isAuthenticated;
  bool get hasNetworkError => _hasNetworkError;

  bool get isOnboarded => _backendUser?.onboardingCompleted ?? false;
  bool get isBanned => _backendUser?.status == UserStatus.banned;
  bool get isSuspended => _backendUser?.status == UserStatus.suspended;
  User? get firebaseUser => _firebaseUser;
  BackendUser? get user => _backendUser;
  String? get error => _error;

  /// Initialize auth state
  Future<void> _init() async {
    _isLoading = true;
    notifyListeners();

    try {
      await _authService.initialize();
      _authSubscription = _authService.authStateChanges.listen(
        _onAuthStateChanged,
      );
    } catch (e) {
      _error = e.toString();
      _isLoading = false;
      notifyListeners();
    }

    // Listen for real-time account status changes (ban/suspension/maintenance)
    _webSocketService.on('user_banned', _onBanned);
    _webSocketService.on('user_suspended', _onSuspended);
    _webSocketService.on('maintenance_mode', _onMaintenance);
  }

  /// Handle real-time ban event - immediately redirect
  void _onBanned(Map<String, dynamic> data) {
    debugPrint('🚫 BANNED event received: $data');
    _handleSystemRedirect('/banned');
  }

  /// Handle real-time suspension event - immediately redirect
  void _onSuspended(Map<String, dynamic> data) {
    debugPrint('⏸️ SUSPENDED event received: $data');
    _handleSystemRedirect('/suspended');
  }

  /// Handle maintenance mode event - immediately redirect
  void _onMaintenance(Map<String, dynamic> data) {
    debugPrint('🔧 MAINTENANCE event received: $data');
    _handleSystemRedirect('/maintenance');
  }

  /// Immediately redirect to system screen
  void _handleSystemRedirect(String path) {
    // Import and navigate directly
    final context = _getNavigatorContext();
    if (context != null) {
      debugPrint('Immediately redirecting to $path');
      GoRouter.of(context).go(path);
    }
    // Also refresh to update local state
    refreshUser();
  }

  /// Get navigator context for immediate redirects
  BuildContext? _getNavigatorContext() {
    try {
      return AppRouter.rootNavigatorKey.currentContext;
    } catch (_) {
      return null;
    }
  }

  /// Handle auth state changes
  Future<void> _onAuthStateChanged(User? user) async {
    if (_firebaseUser?.uid != user?.uid) {
      _webSocketService.disconnect();
      _isAuthenticated = false;
      _backendUser = null;
      _isLoading = true;
      notifyListeners();
    }
    _firebaseUser = user;

    if (user == null) {
      _isAuthenticated = false;
      _backendUser = null;
      _webSocketService.disconnect(); // Disconnect on logout/session end
    } else {
      await _verifyWithBackend();
    }

    _isLoading = false;
    notifyListeners();
  }

  /// Verify token with backend and get user data
  Future<void> _verifyWithBackend() async {
    final identity = _authService.currentUser?.uid;
    try {
      final token = await _authService.getIdToken();
      if (token == null) {
        _isAuthenticated = false;
        _hasNetworkError = false;
        return;
      }

      final response = await _apiService.post<Map<String, dynamic>>(
        '/auth/verify',
        body: {'id_token': token},
        withAuth: false,
      );

      if (_authService.currentUser?.uid != identity) return;
      if (response.success && response.data != null) {
        _isAuthenticated = true;
        _hasNetworkError = false;

        final verifiedUser = BackendUser.fromJson(response.data!);
        if (_backendUser?.userId != verifiedUser.userId) {
          _webSocketService.disconnect();
        }
        _backendUser = verifiedUser;
        _error = null;

        if (_backendUser!.onboardingCompleted) {
          unawaited(
            _webSocketService.connect(
              token,
              tokenProvider: () => _authService.getIdToken(forceRefresh: true),
            ),
          );
        } else {
          _webSocketService.disconnect();
        }
      } else {
        _isAuthenticated = false;
        _error = response.error;

        _hasNetworkError = response.statusCode == 0;
      }
    } catch (e) {
      _isAuthenticated = false;
      _hasNetworkError = true;
      // Sanitize error for user
      _error = 'Connection failed. Please check your internet.';
    }
  }

  /// Sign in with Google
  Future<bool> signInWithGoogle() async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    final result = await _authService.signInWithGoogle();

    if (!result.success) {
      _error = result.error;
      _isLoading = false;
      notifyListeners();
      return false;
    }

    // Explicitly verify to ensure state is updated and loading cleared,
    // handling cases where authStateChanges might not fire (e.g. re-login).
    _firebaseUser = result.user;
    await _verifyWithBackend();

    _isLoading = false;
    notifyListeners();
    return true;
  }

  Future<bool> signInInternal(String email, String password) async {
    _isLoading = true;
    _error = null;
    notifyListeners();
    final result = await _authService.signInInternal(email, password);
    if (result.success) {
      _firebaseUser = result.user;
      await _verifyWithBackend();
    } else {
      _error = result.error;
    }
    _isLoading = false;
    notifyListeners();
    return result.success && _isAuthenticated;
  }

  /// Complete onboarding
  Future<bool> completeOnboarding({
    required String phone,
    required String? gender,
    required bool safetyConsent,
    bool offerLifts = false,
    bool photoClear = false,
    int tourVersion = 0,
    String occupation = 'other',
  }) async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    final response = await _apiService.post<Map<String, dynamic>>(
      '/auth/onboard',
      body: {
        'phone': phone,
        'gender': gender,
        'safety_consent': safetyConsent,
        'privacy_consent': true,
        'offer_lifts': offerLifts,
        'photo_clear': photoClear,
        'tour_version': tourVersion,
        'occupation': occupation,
      },
    );

    if (response.success) {
      // Refresh user data
      await _verifyWithBackend();
      _isLoading = false;
      notifyListeners();
      return true;
    } else {
      _error = response.error;
      _isLoading = false;
      notifyListeners();
      return false;
    }
  }

  /// Sign out
  Future<void> signOut() async {
    var registrationSettled = true;
    try {
      await beforeSignOut?.call();
    } catch (_) {
      registrationSettled = false;
    }
    final token = _notificationService?.fcmToken;
    if (registrationSettled &&
        token != null &&
        token.isNotEmpty &&
        _apiService.authIdentity != null) {
      try {
        await _apiService
            .post<Map<String, dynamic>>(
              '/users/me/push-token/unregister',
              body: {'token': token},
            )
            .timeout(const Duration(seconds: 5));
      } catch (_) {
        // Signing out remains possible offline; the token may remain registered.
      }
    }
    await _authService.signOut();
    _isAuthenticated = false;
    _hasNetworkError = false;
    _backendUser = null;
    _error = null;
    _webSocketService.disconnect(); // Explicit disconnect
    notifyListeners();
  }

  bool _deletionInProgress = false;
  Future<String?> requestAccountDeletion({String? emulatorPassword}) async {
    if (_deletionInProgress)
      return 'A deletion request is already in progress.';
    _deletionInProgress = true;
    try {
      return await _requestAccountDeletion(emulatorPassword: emulatorPassword);
    } finally {
      _deletionInProgress = false;
    }
  }

  Future<String?> _requestAccountDeletion({String? emulatorPassword}) async {
    final identity = _apiService.authIdentity;
    if (identity == null ||
        !await _authService.reauthenticateGoogle(
          emulatorPassword: emulatorPassword,
        ) ||
        _apiService.authIdentity != identity) {
      return 'Google reauthentication did not complete. Please try again.';
    }
    final response = await _apiService.delete<Map<String, dynamic>>(
      '/users/me',
    );
    if (_apiService.authIdentity != identity)
      return 'The signed-in account changed. Please try again.';
    if (!response.success)
      return response.error ?? 'Could not confirm the deletion request.';
    await _authService.signOut();
    _isAuthenticated = false;
    _backendUser = null;
    _webSocketService.disconnect();
    notifyListeners();
    return null;
  }

  /// Refresh user data
  Future<void> refreshUser() async {
    await _verifyWithBackend();
    notifyListeners();
  }

  @override
  void dispose() {
    _authSubscription?.cancel();
    super.dispose();
  }
}
