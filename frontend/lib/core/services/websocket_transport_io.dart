import 'package:web_socket_channel/io.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

WebSocketChannel openAuthenticatedSocket(Uri uri, String token) =>
    IOWebSocketChannel.connect(
      uri,
      headers: {'Authorization': 'Bearer $token'},
      protocols: const ['orix.v1'],
      connectTimeout: const Duration(seconds: 10),
    );
