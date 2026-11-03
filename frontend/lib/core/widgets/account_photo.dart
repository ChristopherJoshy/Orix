import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../config/api_config.dart';
import '../services/auth_service.dart';
import 'pixel_ui.dart';

/// Bearer headers are sent only to our own authenticated avatar endpoint.
class AccountPhoto extends StatefulWidget {
  final String? url;
  final double size;
  const AccountPhoto({super.key, this.url, this.size = 44});
  @override
  State<AccountPhoto> createState() => _AccountPhotoState();
}

class _AccountPhotoState extends State<AccountPhoto> {
  Future<String?>? _token;
  String? _identity, _source;
  Uri? _uri;
  bool _private = false;
  void _configure() {
    final auth = context.read<AuthService>();
    final identity = auth.currentUser?.uid;
    if (_source == widget.url && _identity == identity) return;
    _source = widget.url;
    _identity = identity;
    final parsed = Uri.tryParse(widget.url ?? '');
    _uri = parsed == null || widget.url?.isNotEmpty != true
        ? null
        : ApiConfig.baseUri.resolveUri(parsed);
    if (_uri != null && !{'http', 'https'}.contains(_uri!.scheme)) _uri = null;
    _private =
        _uri?.origin == ApiConfig.baseUri.origin &&
        _uri?.path.startsWith('/api/v1/users/') == true;
    if (_uri?.userInfo.isNotEmpty == true ||
        (_uri?.scheme != 'https' && !_private))
      _uri = null;
    _token = _private ? auth.getIdToken() : Future.value(null);
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _configure();
  }

  @override
  void didUpdateWidget(covariant AccountPhoto old) {
    super.didUpdateWidget(old);
    _configure();
  }

  @override
  Widget build(BuildContext context) => SizedBox(
    width: widget.size,
    height: widget.size,
    child: ColorFiltered(
      colorFilter: const ColorFilter.matrix([
        .2126,
        .7152,
        .0722,
        0,
        0,
        .2126,
        .7152,
        .0722,
        0,
        0,
        .2126,
        .7152,
        .0722,
        0,
        0,
        0,
        0,
        0,
        1,
        0,
      ]),
      child: FutureBuilder<String?>(
        future: _token,
        builder: (context, snapshot) {
          const fallback = OrixLogo(symbolOnly: true, width: 32);
          if (_uri == null || (_private && snapshot.data == null))
            return fallback;
          return CachedNetworkImage(
            imageUrl: _uri.toString(),
            fit: BoxFit.cover,
            cacheKey: _private ? '$_identity:$_uri' : _uri.toString(),
            httpHeaders: _private
                ? {'Authorization': 'Bearer ${snapshot.data}'}
                : null,
            placeholder: (_, _) => fallback,
            errorWidget: (_, _, _) => fallback,
          );
        },
      ),
    ),
  );
}
