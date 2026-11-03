/// Centralized Time Utilities - All datetime parsing from API should use these.
library;

class TimeUtils {
  TimeUtils._();

  /// Parse UTC datetime string from API and convert to local.
  static DateTime parseUtc(String isoString) {
    final value = isoString.trim();
    final hasZone = RegExp(
      r'(?:Z|[+-]\d{2}:?\d{2})$',
      caseSensitive: false,
    ).hasMatch(value);
    return DateTime.parse(hasZone ? value : '${value}Z').toLocal();
  }

  static DateTime? tryParseUtc(Object? value) {
    if (value is! String || value.isEmpty) return null;
    try {
      return parseUtc(value);
    } on FormatException {
      return null;
    }
  }

  /// Get current timezone offset in minutes for API requests.
  static int get timezoneOffsetMinutes =>
      DateTime.now().timeZoneOffset.inMinutes;

  /// Format datetime relative to now.
  static String formatRelative(DateTime time) {
    final now = DateTime.now();
    final diff = now.difference(time);

    if (diff.inMinutes < 1) return 'Just now';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    if (diff.inDays < 7) return '${diff.inDays}d ago';
    return '${(diff.inDays / 7).floor()}w ago';
  }

  /// Format time for display (e.g., "2:30 PM").
  static String formatTime(DateTime time) {
    final hour = time.hour;
    final minute = time.minute.toString().padLeft(2, '0');
    final period = hour >= 12 ? 'PM' : 'AM';
    final h12 = hour > 12 ? hour - 12 : (hour == 0 ? 12 : hour);
    return '$h12:$minute $period';
  }
}
