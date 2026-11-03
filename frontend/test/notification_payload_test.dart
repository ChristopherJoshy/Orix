import 'package:flutter_test/flutter_test.dart';
import 'package:orix/core/services/notification_service.dart';

void main() {
  test(
    'engagement taps open home without a group and still require identity',
    () {
      final data = <String, dynamic>{
        'type': 'engagement_reminder',
        'recipient_user_id': 'backend-user',
        'recipient_firebase_uid': 'firebase-user',
      };
      expect(
        ConversationNotificationPayload.fromData(data)?['type'],
        'engagement_reminder',
      );
      data.remove('recipient_user_id');
      expect(ConversationNotificationPayload.fromData(data), isNull);
    },
  );
  test(
    'group tap retains destination and requires both recipient identities',
    () {
      final data = <String, dynamic>{
        'type': 'group_rejected',
        'group_id': 'g',
        'recipient_user_id': 'backend-user',
        'recipient_firebase_uid': 'firebase-user',
      };
      final payload = ConversationNotificationPayload.fromData(data);
      expect(payload?['group_id'], 'g');
      expect(payload?['type'], 'group_rejected');
      data.remove('recipient_firebase_uid');
      expect(ConversationNotificationPayload.fromData(data), isNull);
    },
  );
  test('server alarm round trips punctuation and UTC offsets', () {
    final data = {
      'type': 'RIDE_ALARM',
      'group_id': 'g',
      'ride_time': '2026-11-03T08:00:00+05:30',
      'pickup_summary': 'Gate | A:2',
    };
    expect(RideAlarmPayload.decode(RideAlarmPayload.encode(data)), {
      'group_id': 'g',
      'ride_time': data['ride_time'],
      'pickup_summary': data['pickup_summary'],
    });
  });
  test('ordinary reminders and completed rides are not departure alarms', () {
    for (final type in ['ride_reminder', 'ride_completed', 'chat_message']) {
      expect(
        RideAlarmPayload.fromData({'type': type, 'group_id': 'g'}),
        isNull,
      );
    }
  });
  test('malformed payloads and missing groups are ignored', () {
    for (final payload in [
      null,
      '',
      'invalid',
      '{}',
      '[]',
      '{"type":"RIDE_ALARM","group_id":""}',
    ]) {
      expect(RideAlarmPayload.decode(payload), isNull);
    }
  });
  test('older delivered pipe payload is still readable', () {
    expect(RideAlarmPayload.decode('show_alarm|g|08:30|Gate | A'), {
      'group_id': 'g',
      'ride_time': '08:30',
      'pickup_summary': 'Gate | A',
    });
  });
}
