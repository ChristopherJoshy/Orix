import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:orix/core/config/api_config.dart';
import 'package:orix/core/services/app_version_service.dart';

void main() {
  test('APK updates require confirmed availability and use our own API', () {
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    try {
      final unavailable = AppVersionInfo(
        currentVersion: '2.1.2',
        minVersion: '2.0.0',
        updateRequired: false,
        playStoreUrl: 'https://untrusted.example/app',
      );
      expect(unavailable.storeUrl, isEmpty);
      final available = AppVersionInfo(
        currentVersion: '2.1.2',
        minVersion: '2.0.0',
        updateRequired: false,
        androidAPKAvailable: true,
      );
      final link = Uri.parse(available.storeUrl);
      expect(link.origin, ApiConfig.baseUri.origin);
      expect(link.path, '/downloads/android.apk');
      debugDefaultTargetPlatformOverride = TargetPlatform.iOS;
      expect(available.storeUrl, isEmpty);
    } finally {
      debugDefaultTargetPlatformOverride = null;
    }
  });
}
