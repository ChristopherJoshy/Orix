import 'dart:io' show HttpDate, HttpException;

import 'package:http/http.dart' as http;

/// Share server cooldowns across polling, preferences and push registration.
/// A throttled mutation is rejected locally, never queued for a later replay.
class RateLimitedClient extends http.BaseClient {
  RateLimitedClient(this._inner, {DateTime Function()? now})
    : _now = now ?? DateTime.now;

  final http.Client _inner;
  final DateTime Function() _now;
  final Map<String, DateTime> _cooldowns = {};

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    final origin = request.url.origin;
    final remaining = _cooldowns[origin]?.difference(_now());
    if (remaining != null && remaining > Duration.zero) {
      return http.StreamedResponse(
        const Stream<List<int>>.empty(),
        429,
        headers: {'retry-after': '${(remaining.inMilliseconds / 1000).ceil()}'},
        request: request,
      );
    }
    _cooldowns.remove(origin);
    final response = await _inner.send(request);
    if (response.statusCode == 429) {
      final guidance = response.headers['retry-after'];
      var delay = const Duration(minutes: 1);
      final seconds = int.tryParse(guidance ?? '');
      if (seconds != null && seconds > 0) {
        delay = Duration(seconds: seconds);
      } else if (guidance != null) {
        try {
          final until = HttpDate.parse(guidance).difference(_now());
          if (until > Duration.zero) delay = until;
        } on HttpException {
          // Missing/malformed guidance must not cause a rapid retry loop.
        }
      }
      _cooldowns[origin] = _now().add(delay);
    }
    return response;
  }

  @override
  void close() => _inner.close();
}
