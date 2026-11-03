/// Public tile configuration shared by picker and route previews.
library;

import 'package:flutter/material.dart';
import 'package:flutter_map/flutter_map.dart';
import 'package:url_launcher/url_launcher.dart';

class MapConfig {
  MapConfig._();
  static const tileUrl = String.fromEnvironment(
    'MAP_TILE_URL',
    defaultValue: 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
  );
  static const attribution = String.fromEnvironment(
    'MAP_ATTRIBUTION',
    defaultValue: '© OpenStreetMap contributors',
  );
  static const attributionUrl = String.fromEnvironment(
    'MAP_ATTRIBUTION_URL',
    defaultValue: 'https://www.openstreetmap.org/copyright',
  );
  static Widget tiles() => ColorFiltered(
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
    child: TileLayer(
      urlTemplate: tileUrl,
      userAgentPackageName: 'com.orix.app',
    ),
  );
  static Widget credits() => RichAttributionWidget(
    attributions: [
      TextSourceAttribution(
        attribution,
        onTap: () {
          final uri = Uri.tryParse(attributionUrl);
          if (uri?.scheme == 'https' &&
              uri!.host.isNotEmpty &&
              uri.userInfo.isEmpty) {
            launchUrl(uri, mode: LaunchMode.externalApplication);
          }
        },
      ),
    ],
  );
}
