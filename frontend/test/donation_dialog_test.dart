import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:orix/core/config/donation_config.dart';
import 'package:orix/core/theme/app_theme.dart';
import 'package:orix/core/widgets/donation_dialog.dart';

void main() {
  test('Payment payload leaves the donor free to choose an amount', () {
    final uri = Uri.parse(DonationConfig.upi);
    expect(uri.queryParameters['pa'], DonationConfig.upiId);
    expect(uri.queryParameters.containsKey('am'), isFalse);
    expect(DonationPrompt.day(DateTime(2026, 11, 3, 23, 59)), '2026-11-3');
    expect(DonationPrompt.day(DateTime(2026, 11, 4)), '2026-11-4');
  });

  testWidgets('Daily prompt is dismissible and persists its shown day', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues({});
    late BuildContext page;
    await tester.pumpWidget(
      MaterialApp(
        theme: AppTheme.lightTheme,
        home: Builder(
          builder: (context) {
            page = context;
            return const Scaffold();
          },
        ),
      ),
    );
    final showing = DonationPrompt.maybeShow(page);
    await tester.pumpAndSettle();
    expect(find.byType(DonationDialog), findsOneWidget);
    final prefs = await SharedPreferences.getInstance();
    expect(
      prefs.getString(DonationPrompt.preferenceKey),
      DonationPrompt.day(DateTime.now()),
    );
    await tester.tap(find.text('Maybe later'));
    await tester.pumpAndSettle();
    await showing;
    await DonationPrompt.maybeShow(page);
    await tester.pumpAndSettle();
    expect(find.byType(DonationDialog), findsNothing);
  });

  testWidgets(
    'Missing Google Pay shows actionable failure without payment success',
    (tester) async {
      debugDefaultTargetPlatformOverride = TargetPlatform.android;
      addTearDown(() => debugDefaultTargetPlatformOverride = null);
      const channel = MethodChannel('orix/donations');
      tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(channel, (
        call,
      ) async {
        expect(call.method, 'openGooglePay');
        expect(call.arguments, DonationConfig.upi);
        return false;
      });
      addTearDown(
        () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
          channel,
          null,
        ),
      );
      for (final scale in [1.0, 2.0]) {
        await tester.pumpWidget(
          MaterialApp(
            theme: AppTheme.lightTheme,
            home: MediaQuery(
              data: MediaQueryData(textScaler: TextScaler.linear(scale)),
              child: const Scaffold(body: DonationDialog()),
            ),
          ),
        );
        await tester.pumpAndSettle();
        await tester.ensureVisible(find.text('Donate'));
        await tester.pumpAndSettle();
        await tester.tap(find.text('Donate'));
        await tester.pumpAndSettle();
        expect(
          find.textContaining('Google Pay could not open.'),
          findsOneWidget,
        );
        expect(find.text('Donate'), findsOneWidget);
        expect(tester.takeException(), isNull);
      }
      debugDefaultTargetPlatformOverride = null;
    },
  );
}
