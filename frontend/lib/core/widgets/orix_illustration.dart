import 'package:flutter/material.dart';

enum OrixIllustrationScene { welcome, companions }

/// App-specific editorial art. Route, identity and action information remains text.
class OrixIllustration extends StatelessWidget {
  final OrixIllustrationScene scene;
  final double? width, height;
  const OrixIllustration({
    super.key,
    required this.scene,
    this.width,
    this.height,
  });
  @override
  Widget build(BuildContext context) => ColorFiltered(
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
    child: Image.asset(
      scene == OrixIllustrationScene.welcome
          ? 'assets/images/orix_app_welcome.png'
          : 'assets/images/orix_app_companions.png',
      width: width,
      height: height,
      fit: BoxFit.contain,
      filterQuality: FilterQuality.none,
      excludeFromSemantics: true,
    ),
  );
}

/// Large windows place art beside sign-in; compact windows preserve reading order.
class OrixWelcomeLayout extends StatelessWidget {
  final Widget header, controls;
  const OrixWelcomeLayout({
    super.key,
    required this.header,
    required this.controls,
  });
  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, bounds) {
      final wide = bounds.maxWidth >= 840;
      final padding = bounds.maxWidth < 380 ? 20.0 : 28.0;
      final form = ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 380),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            header,
            if (!wide) ...[
              const SizedBox(height: 24),
              OrixIllustration(
                scene: OrixIllustrationScene.welcome,
                height: (bounds.maxWidth * .56).clamp(150.0, 240.0),
              ),
            ],
            const SizedBox(height: 28),
            controls,
          ],
        ),
      );
      return SingleChildScrollView(
        padding: EdgeInsets.all(padding),
        child: ConstrainedBox(
          constraints: BoxConstraints(
            minHeight: (bounds.maxHeight - padding * 2).clamp(
              0,
              double.infinity,
            ),
          ),
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 1000),
              child: wide
                  ? Row(
                      children: [
                        const Expanded(
                          flex: 5,
                          child: OrixIllustration(
                            scene: OrixIllustrationScene.welcome,
                            height: 460,
                          ),
                        ),
                        const SizedBox(width: 64),
                        Expanded(flex: 4, child: form),
                      ],
                    )
                  : form,
            ),
          ),
        ),
      );
    },
  );
}
