/// Shared REST/WebSocket origin. Private credentials never belong in build defines.
library;

import 'package:flutter/foundation.dart';

class ApiConfig {
  static const Duration timeout = Duration(seconds: 30);
  static const String apiVersion = 'v1';

  static final Uri _origin = _resolveOrigin();
  static String get baseUrl => _origin.toString();
  static Uri get baseUri => _origin;

  static Uri _resolveOrigin() {
    const configured = String.fromEnvironment('API_BASE_URL');
    if (configured.trim().isNotEmpty) {
      return parseOrigin(configured, requireHttps: !kDebugMode);
    }
    if (!kDebugMode) {
      throw const FormatException('API_BASE_URL is required for this build.');
    }
    final host = !kIsWeb && defaultTargetPlatform == TargetPlatform.android
        ? '10.0.2.2'
        : 'localhost';
    return Uri.parse('http://$host:8080');
  }

  /// Accept an origin only, so endpoint paths cannot accidentally duplicate or
  /// inherit credentials, query parameters or fragments from configuration.
  static Uri parseOrigin(String value, {required bool requireHttps}) {
    final uri = Uri.tryParse(value.trim());
    if (uri == null ||
        !uri.hasAuthority ||
        uri.host.isEmpty ||
        uri.userInfo.isNotEmpty ||
        !{'http', 'https'}.contains(uri.scheme) ||
        (uri.path.isNotEmpty && uri.path != '/') ||
        uri.hasQuery ||
        uri.hasFragment ||
        (requireHttps && uri.scheme != 'https') ||
        uri.port < 1 ||
        uri.port > 65535) {
      throw const FormatException(
        'API_BASE_URL must be a valid service origin.',
      );
    }
    return uri.replace(path: '');
  }

  static bool get isConfigured {
    try {
      return _origin.hasAuthority;
    } on FormatException {
      return false;
    }
  }
}

/// Local-only account switching; release builds cannot enable this mode.
class InternalTestConfig {
  static const enabled =
      kDebugMode && bool.fromEnvironment('ENABLE_INTERNAL_TEST_LOGIN');
  static const host = String.fromEnvironment(
    'TEST_AUTH_EMULATOR_HOST',
    defaultValue: '10.0.2.2',
  );
  static const port = int.fromEnvironment(
    'TEST_AUTH_EMULATOR_PORT',
    defaultValue: 9099,
  );
  static const projectId = 'demo-orix';
}
