import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import '../config/api_config.dart';

class ErrorMessages {
  static const String noInternet =
      'Connection lost. The network is on a chaya break. Check your connection and retry.';
  static const String connectionFailed =
      'Could not reach Orix. The network missed its KSRTC. Try again.';
  static const String timeout = 'That took the scenic route. Please try again.';
  static const String serverError =
      'Orix is taking a chaya break. Please try again shortly.';
  static const String unknownError =
      'That did not go through. Let us try again.';
  static const String parseFailed = unknownError;
  static const String locationSearchFailed = unknownError;
  static const String uploadFailed = 'Image upload failed. Please try again.';
  static const String storageError = unknownError;

  /// User-friendly error messages mapped from backend errors and HTTP status codes
  static String getFriendly(String backendError, {int statusCode = 0}) {
    // Normalize the error message
    final lower = backendError.toLowerCase().trim();

    // Match common backend errors
    if (lower.contains('already exists')) {
      return 'This item already exists.';
    }
    if (lower.contains('not found')) {
      return 'Item not found or already removed.';
    }
    if (lower.contains('unauthorized') ||
        lower.contains('invalid token') ||
        lower.contains('not authenticated')) {
      return 'Please log in again.';
    }
    if (lower.contains('forbidden') || lower.contains('not allowed')) {
      return 'You don\'t have permission to do this.';
    }
    if (lower.contains('rate limit') || lower.contains('too many requests')) {
      return 'Too many requests. Please wait a moment and try again.';
    }
    if (lower.contains('invalid request') || lower.contains('bad request')) {
      return 'Please check your input and try again.';
    }
    if (lower.contains('server error') || lower.contains('internal error')) {
      return 'Orix is taking a chaya break. Please try again shortly.';
    }
    if (lower.contains('timeout')) {
      return timeout;
    }
    if (lower.contains('connection') || lower.contains('network')) {
      return connectionFailed;
    }

    // Map HTTP status codes
    switch (statusCode) {
      case 400:
        return 'Please check your input and try again.';
      case 401:
        return 'Please log in again.';
      case 403:
        return 'You don\'t have permission to do this.';
      case 404:
        return 'Item not found or already removed.';
      case 429:
        return 'Too many requests. Please wait a moment and try again.';
      case 500:
      case 502:
      case 503:
        return serverError;
      default:
        return unknownError;
    }
  }
}

class ErrorHelper {
  static Map<String, dynamic> Function()? getUserInfo;
  static Future<String?> Function()? getAuthToken;

  static void logError(
    Object error, [
    StackTrace? stackTrace,
    String? context,
  ]) async {
    try {
      final url = Uri.parse('${ApiConfig.baseUrl}/api/v1/logs/client');
      final headers = <String, String>{'Content-Type': 'application/json'};

      if (getAuthToken != null) {
        try {
          final token = await getAuthToken!();
          if (token != null) {
            headers['Authorization'] = 'Bearer $token';
          }
        } catch (_) {}
      }

      final contextStr = context ?? 'Frontend Error';

      String level = 'error';
      final errStr = error.toString().toLowerCase();
      if (errStr.contains('warning')) {
        level = 'warning';
      } else if (errStr.contains('info')) {
        level = 'info';
      }

      final body = jsonEncode({
        'level': level,
        'message': 'Client operation failed (${error.runtimeType})',
        'context': contextStr,
        'stack_trace': null,
        'metadata': <String, dynamic>{},
      });

      http.post(url, headers: headers, body: body).catchError((_) {
        return http.Response('', 500);
      });
    } catch (_) {}
  }

  static String getUserFriendlyMessage(
    Object error, {
    StackTrace? stackTrace,
    String? context,
  }) {
    logError(error, stackTrace, context);

    if (error is TimeoutException ||
        error.toString().toLowerCase().contains('timeout')) {
      return ErrorMessages.timeout;
    }

    if (error is SocketException ||
        error.toString().contains('ClientException') ||
        error.toString().toLowerCase().contains('socket') ||
        error.toString().toLowerCase().contains('connection') ||
        error.toString().toLowerCase().contains('network')) {
      return ErrorMessages.noInternet;
    }

    if (error is HttpException) {
      return ErrorMessages.serverError;
    }

    return ErrorMessages.unknownError;
  }

  static String getApiErrorMessage(
    Object error, {
    StackTrace? stackTrace,
    String? context,
  }) {
    return getUserFriendlyMessage(
      error,
      stackTrace: stackTrace,
      context: context ?? 'API Error',
    );
  }

  static String getLocationErrorMessage(
    Object error, {
    StackTrace? stackTrace,
  }) {
    logError(error, stackTrace, 'Location Search');
    if (error is TimeoutException || error is SocketException) {
      return ErrorMessages.noInternet;
    }
    return ErrorMessages.unknownError;
  }

  static String getAuthErrorMessage(Object error, {StackTrace? stackTrace}) {
    logError(error, stackTrace, 'Auth Error');
    if (error is TimeoutException || error is SocketException) {
      return ErrorMessages.noInternet;
    }
    return ErrorMessages.unknownError;
  }

  static void logDebug(String message, {String? context}) {
    if (kDebugMode) {
      debugPrint('[DEBUG] ${context ?? "App"}: $message');
    }
  }
}
