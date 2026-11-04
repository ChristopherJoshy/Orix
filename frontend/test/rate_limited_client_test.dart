import 'dart:io' show HttpDate;

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:orix/core/services/rate_limited_client.dart';

void main() {
  for (final guidance in ['60', 'date', 'missing', 'invalid']) {
    test('shares $guidance cooldown without replaying mutations', () async {
      var now = DateTime.utc(2026, 11, 4);
      var calls = 0;
      final client = RateLimitedClient(
        MockClient((request) async {
          calls++;
          if (calls > 1) return http.Response('{}', 200);
          return http.Response(
            '',
            429,
            headers: {
              if (guidance != 'missing')
                'retry-after': guidance == 'date'
                    ? HttpDate.format(now.add(const Duration(minutes: 1)))
                    : guidance,
            },
          );
        }),
        now: () => now,
      );
      addTearDown(client.close);
      final origin = 'https://api.example';
      expect(
        (await client.get(Uri.parse('$origin/notifications'))).statusCode,
        429,
      );
      expect((await client.put(Uri.parse('$origin/users/me'))).statusCode, 429);
      expect(calls, 1);
      now = now.add(const Duration(seconds: 61));
      expect(
        (await client.get(Uri.parse('$origin/preferences'))).statusCode,
        200,
      );
      expect(calls, 2);
    });
  }
}
