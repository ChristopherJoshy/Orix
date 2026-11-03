/// Public policy URL is supplied by the operator at build time.
library;

import 'package:url_launcher/url_launcher.dart';

class PrivacyConfig {
  PrivacyConfig._();
  static Uri? get policy {
    const value = String.fromEnvironment('PRIVACY_POLICY_URL');
    final uri = Uri.tryParse(value);
    return uri != null &&
            uri.scheme == 'https' &&
            uri.host.isNotEmpty &&
            uri.userInfo.isEmpty
        ? uri
        : null;
  }

  static Future<bool> openPolicy() async {
    final uri = policy;
    if (uri == null) return false;
    try {
      return await launchUrl(uri, mode: LaunchMode.externalApplication);
    } catch (_) {
      return false;
    }
  }
}
