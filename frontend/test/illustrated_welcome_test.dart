import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'package:orix/core/widgets/orix_illustration.dart';
import 'package:orix/core/widgets/pixel_ui.dart';

void main() {
  testWidgets(
    'Illustrated welcome adapts and keeps sign-in reachable with enlarged text',
    (tester) async {
      for (final size in [
        const Size(320, 568),
        const Size(390, 844),
        const Size(900, 700),
      ]) {
        for (final scale in [1.0, 2.0]) {
          tester.view.physicalSize = size;
          tester.view.devicePixelRatio = 1;
          await tester.pumpWidget(
            MaterialApp(
              theme: AppTheme.lightTheme,
              home: MediaQuery(
                data: MediaQueryData(
                  size: size,
                  textScaler: TextScaler.linear(scale),
                ),
                child: Scaffold(
                  body: SafeArea(
                    child: OrixWelcomeLayout(
                      header: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const OrixLogo(width: 132),
                          const SizedBox(height: 28),
                          Text(
                            'Go together.',
                            style: AppTheme.lightTheme.textTheme.headlineLarge,
                          ),
                          const Text(
                            'A familiar route. A little company. Sign in to find your people.',
                          ),
                        ],
                      ),
                      controls: Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          FilledButton(
                            onPressed: () {},
                            child: const Text('Continue with Google'),
                          ),
                          const Text(
                            'Review the community terms and privacy notice during setup.',
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
            ),
          );
          await tester.pumpAndSettle();
          expect(
            tester.takeException(),
            isNull,
            reason: '${size.width}px at ${scale}x',
          );
          await tester.ensureVisible(find.text('Continue with Google'));
          await tester.pumpAndSettle();
          expect(
            find.text('Continue with Google').hitTestable(),
            findsOneWidget,
          );
        }
      }
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
    },
  );
  testWidgets('App companions artwork loads in a narrow introduction', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 568);
    tester.view.devicePixelRatio = 1;
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightTheme,
        home: const Scaffold(
          body: SingleChildScrollView(
            child: Column(
              children: [
                OrixIllustration(
                  scene: OrixIllustrationScene.companions,
                  height: 180,
                ),
                Text('Your details'),
                TextField(
                  decoration: InputDecoration(labelText: 'Display name'),
                ),
              ],
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
  });
}
