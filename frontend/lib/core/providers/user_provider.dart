/// User Provider
///
/// Manages profile updates and push-token synchronization.
library;

import 'dart:async';

import 'package:flutter/widgets.dart';
import '../config/api_config.dart';

import '../services/api_service.dart';
import '../services/notification_service.dart';
import 'auth_provider.dart';

/// User provider for profile management
class UserProvider extends ChangeNotifier with WidgetsBindingObserver {
  final ApiService _apiService;
  final NotificationService _notificationService;
  AuthProvider? _authProvider;

  bool _disposed = false;
  bool _isLoading = false;
  String? _error;

  String? _lastSyncedToken;
  String? _lastSyncedIdentity;
  bool _isSyncingToken = false;
  bool _signingOut = false;
  Completer<void>? _syncDone;
  bool _preparingPush = false;

  UserProvider(this._apiService, this._notificationService) {
    WidgetsBinding.instance.addObserver(this);
    _notificationService.onTokenRefresh = (token) {
      if (_authProvider?.user != null) {
        _syncFcmToken();
      }
    };
  }

  bool get isLoading => _isLoading;
  String? get error => _error;

  /// Update auth reference and sync FCM token if logged in
  void updateAuth(AuthProvider auth) {
    _authProvider = auth;
    auth.beforeSignOut = _prepareSignOut;
    if (auth.user == null) _signingOut = false;
    if (auth.user != null) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        unawaited(_prepareAutomaticPush());
      });
    } else {
      _lastSyncedToken = null;
      _lastSyncedIdentity = null;
    }
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      _lastSyncedToken = null;
      unawaited(_prepareAutomaticPush());
    }
  }

  Future<void> _prepareAutomaticPush() async {
    if (_disposed ||
        _preparingPush ||
        _signingOut ||
        InternalTestConfig.enabled ||
        _authProvider?.isAuthenticated != true ||
        _authProvider?.isOnboarded != true) {
      return;
    }
    final identity = _apiService.authIdentity;
    _preparingPush = true;
    try {
      if (!await _notificationService.prepareAutomaticPush()) return;
      if (_disposed || identity != _apiService.authIdentity || _signingOut) {
        return;
      }
      await _syncFcmToken();
    } catch (_) {
      // Settings exposes explicit recovery; next foreground entry retries setup.
      if (!_disposed && identity == _apiService.authIdentity) {
        _error =
            'Phone alerts could not connect. Retry in notification settings.';
      }
    } finally {
      _preparingPush = false;
    }
  }

  Future<void> _syncFcmToken() async {
    if (_disposed || _isSyncingToken || _signingOut) return;
    final identity = _apiService.authIdentity;
    final token = _notificationService.fcmToken;
    if (identity == null ||
        _authProvider?.user == null ||
        token == null ||
        (token == _lastSyncedToken && identity == _lastSyncedIdentity)) {
      return;
    }
    _isSyncingToken = true;
    final done = Completer<void>();
    _syncDone = done;
    try {
      final response = await _apiService.put<Map<String, dynamic>>(
        '/users/me',
        body: {'fcm_token': token},
        retryTransient: true,
      );
      if (response.success && identity == _apiService.authIdentity) {
        _lastSyncedIdentity = identity;
        _lastSyncedToken = token;
      }
    } finally {
      _isSyncingToken = false;
      if (!done.isCompleted) done.complete();
      if (identical(_syncDone, done)) _syncDone = null;
      // A token refresh or account switch can happen while the request is in flight.
      if (_apiService.authIdentity != identity ||
          _notificationService.fcmToken != token) {
        if (_authProvider?.user != null) _syncFcmToken();
      }
    }
  }

  /// Re-register this installation after permission changes or a failed sync.
  Future<bool> enablePhoneNotifications() async {
    final identity = _apiService.authIdentity;
    if (identity == null || _authProvider?.user == null) return false;
    if (!await _notificationService.requestPermission()) return false;
    if (_disposed || identity != _apiService.authIdentity) return false;
    final pending = _syncDone;
    if (pending != null) await pending.future;
    if (_disposed || identity != _apiService.authIdentity) return false;
    _lastSyncedToken = null;
    await _syncFcmToken();
    return identity == _apiService.authIdentity &&
        _lastSyncedIdentity == identity &&
        _lastSyncedToken != null;
  }

  Future<void> _prepareSignOut() async {
    _signingOut = true;
    final pending = _syncDone;
    if (pending != null) {
      await pending.future.timeout(const Duration(seconds: 35));
    }
  }

  /// Apply a profile edit only to the account that initiated it.
  Future<bool> updateProfile({
    String? displayName,
    String? photoUrl,
    String? fcmToken,
  }) async {
    final identity = _apiService.authIdentity;
    final accountId = _authProvider?.user?.userId;
    if (_disposed || _isLoading || identity == null || accountId == null) {
      return false;
    }
    _isLoading = true;
    _error = null;
    notifyListeners();
    bool current() =>
        !_disposed &&
        _apiService.authIdentity == identity &&
        _authProvider?.user?.userId == accountId;
    try {
      final body = <String, dynamic>{
        if (displayName != null) 'display_name': displayName,
        if (photoUrl != null) 'photo_url': photoUrl,
        if (fcmToken != null) 'fcm_token': fcmToken,
      };
      final response = await _apiService.put<Map<String, dynamic>>(
        '/users/me',
        body: body,
      );
      if (!current()) return false;
      if (!response.success) {
        _error = response.error ?? 'Could not update profile.';
        return false;
      }
      await _authProvider?.refreshUser();
      return current();
    } catch (_) {
      if (current()) {
        _error = 'Could not confirm profile update. Refresh and try again.';
      }
      return false;
    } finally {
      if (!_disposed) {
        _isLoading = false;
        notifyListeners();
      }
    }
  }

  @override
  void dispose() {
    _disposed = true;
    WidgetsBinding.instance.removeObserver(this);
    if (_authProvider?.beforeSignOut == _prepareSignOut) {
      _authProvider?.beforeSignOut = null;
    }
    _notificationService.onTokenRefresh = null;
    super.dispose();
  }
}
