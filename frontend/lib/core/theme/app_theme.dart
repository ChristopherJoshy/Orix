import 'package:flutter/material.dart';

/// Shared with the website: paper, ink and neutral grays only.
class AppColors {
  AppColors._();
  static const primary = Color(0xFF222222);
  static const primaryLight = Color(0xFF444444);
  static const primaryDark = Color(0xFF111111);
  static const secondary = Color(0xFFFFFFFF);
  static const secondaryLight = secondary;
  static const secondaryDark = Color(0xFFF7F7F7);
  static const tertiary = Color(0xFF666666);
  static const success = Color(0xFF333333);
  static const error = Color(0xFF222222);
  static const warning = Color(0xFF555555);
  static const background = Color(0xFFF7F7F7);
  static const surface = Color(0xFFFFFFFF);
  static const surfaceVariant = Color(0xFFECECEC);
  static const outline = Color(0xFFD5D5D5);
  static const outlineVariant = Color(0xFF8A8A8A);
  static const textPrimary = Color(0xFF222222);
  static const textSecondary = Color(0xFF626262);
  static const textTertiary = Color(0xFF6B6B6B);
  static const textOnPrimary = Color(0xFFFFFFFF);
  static const backgroundLight = background;
  static const surfaceLight = surface;
  static const surfaceVariantLight = surfaceVariant;
  static const outlineLight = outline;
  static const textPrimaryLight = textPrimary;
  static const textSecondaryLight = textSecondary;
  static const backgroundDark = background;
  static const surfaceDark = surface;
  static const surfaceVariantDark = surfaceVariant;
  static const outlineDark = outline;
  static const textPrimaryDark = textPrimary;
  static const textSecondaryDark = textSecondary;
}

extension ColorOpacity on Color {
  Color withAlphaValue(double opacity) =>
      withValues(alpha: opacity.clamp(0, 1));
}

class AppTextStyles {
  AppTextStyles._();
  static const fontFamily = 'Manrope';
  static const displayFamily = 'Pixelify Sans';
  static const displayLarge = TextStyle(
    fontFamily: displayFamily,
    fontSize: 52,
    height: 1.08,
    fontWeight: FontWeight.w500,
  );
  static const displayMedium = TextStyle(
    fontFamily: displayFamily,
    fontSize: 42,
    height: 1.12,
    fontWeight: FontWeight.w500,
  );
  static const headlineLarge = TextStyle(
    fontFamily: displayFamily,
    fontSize: 30,
    height: 1.15,
    fontWeight: FontWeight.w500,
  );
  static const headlineMedium = TextStyle(
    fontFamily: displayFamily,
    fontSize: 26,
    height: 1.18,
    fontWeight: FontWeight.w500,
  );
  static const headlineSmall = TextStyle(
    fontFamily: displayFamily,
    fontSize: 23,
    height: 1.2,
    fontWeight: FontWeight.w500,
  );
  static const titleLarge = TextStyle(
    fontFamily: displayFamily,
    fontSize: 20,
    height: 1.25,
    fontWeight: FontWeight.w500,
  );
  static const titleMedium = TextStyle(
    fontFamily: fontFamily,
    fontSize: 16,
    height: 1.45,
    fontWeight: FontWeight.w600,
  );
  static const titleSmall = TextStyle(
    fontFamily: fontFamily,
    fontSize: 14,
    height: 1.45,
    fontWeight: FontWeight.w600,
  );
  static const bodyLarge = TextStyle(
    fontFamily: fontFamily,
    fontSize: 16,
    height: 1.6,
  );
  static const bodyMedium = TextStyle(
    fontFamily: fontFamily,
    fontSize: 14,
    height: 1.6,
  );
  static const bodySmall = TextStyle(
    fontFamily: fontFamily,
    fontSize: 12,
    height: 1.5,
  );
  static const labelLarge = TextStyle(
    fontFamily: fontFamily,
    fontSize: 14,
    height: 1.35,
    fontWeight: FontWeight.w600,
  );
  static const labelMedium = TextStyle(
    fontFamily: fontFamily,
    fontSize: 12,
    height: 1.4,
    fontWeight: FontWeight.w500,
  );
  static const labelSmall = TextStyle(
    fontFamily: fontFamily,
    fontSize: 11,
    height: 1.4,
    fontWeight: FontWeight.w500,
  );
}

/// Four-pixel stepped corners, without clipping large parts of the hit area.
class PixelBorder extends OutlinedBorder {
  const PixelBorder({super.side});
  @override
  EdgeInsetsGeometry get dimensions => EdgeInsets.all(side.width);
  @override
  ShapeBorder scale(double t) => PixelBorder(side: side.scale(t));
  @override
  PixelBorder copyWith({BorderSide? side}) =>
      PixelBorder(side: side ?? this.side);
  @override
  Path getOuterPath(Rect r, {TextDirection? textDirection}) {
    final d = r.shortestSide < 16 ? 2.0 : 4.0;
    return Path()
      ..moveTo(r.left + d, r.top)
      ..lineTo(r.right - d, r.top)
      ..lineTo(r.right - d, r.top + d)
      ..lineTo(r.right, r.top + d)
      ..lineTo(r.right, r.bottom - d)
      ..lineTo(r.right - d, r.bottom - d)
      ..lineTo(r.right - d, r.bottom)
      ..lineTo(r.left + d, r.bottom)
      ..lineTo(r.left + d, r.bottom - d)
      ..lineTo(r.left, r.bottom - d)
      ..lineTo(r.left, r.top + d)
      ..lineTo(r.left + d, r.top + d)
      ..close();
  }

  @override
  Path getInnerPath(Rect rect, {TextDirection? textDirection}) =>
      getOuterPath(rect.deflate(side.width));
  @override
  void paint(Canvas canvas, Rect rect, {TextDirection? textDirection}) {
    if (side.style == BorderStyle.solid && side.width > 0) {
      canvas.drawPath(
        getOuterPath(rect.deflate(side.width / 2)),
        side.toPaint(),
      );
    }
  }
}

class AppTheme {
  AppTheme._();
  static ThemeData get lightTheme {
    const ink = AppColors.textPrimary;
    const border = OutlineInputBorder(
      borderRadius: BorderRadius.zero,
      borderSide: BorderSide(color: AppColors.outlineVariant),
    );
    final text = const TextTheme(
      displayLarge: AppTextStyles.displayLarge,
      displayMedium: AppTextStyles.displayMedium,
      headlineLarge: AppTextStyles.headlineLarge,
      headlineMedium: AppTextStyles.headlineMedium,
      headlineSmall: AppTextStyles.headlineSmall,
      titleLarge: AppTextStyles.titleLarge,
      titleMedium: AppTextStyles.titleMedium,
      titleSmall: AppTextStyles.titleSmall,
      bodyLarge: AppTextStyles.bodyLarge,
      bodyMedium: AppTextStyles.bodyMedium,
      bodySmall: AppTextStyles.bodySmall,
      labelLarge: AppTextStyles.labelLarge,
      labelMedium: AppTextStyles.labelMedium,
      labelSmall: AppTextStyles.labelSmall,
    ).apply(bodyColor: ink, displayColor: ink);
    final buttons = ButtonStyle(
      minimumSize: const WidgetStatePropertyAll(Size(48, 48)),
      padding: const WidgetStatePropertyAll(
        EdgeInsets.symmetric(horizontal: 18, vertical: 12),
      ),
      shape: const WidgetStatePropertyAll(PixelBorder()),
      textStyle: const WidgetStatePropertyAll(AppTextStyles.labelLarge),
      elevation: const WidgetStatePropertyAll(0),
    );
    return ThemeData(
      useMaterial3: true,
      brightness: Brightness.light,
      fontFamily: AppTextStyles.fontFamily,
      colorScheme: const ColorScheme.light(
        primary: ink,
        onPrimary: Colors.white,
        secondary: ink,
        onSecondary: Colors.white,
        surface: Colors.white,
        onSurface: ink,
        error: ink,
        onError: Colors.white,
        outline: AppColors.outlineVariant,
        outlineVariant: AppColors.outline,
        surfaceContainerHighest: AppColors.surfaceVariant,
      ),
      scaffoldBackgroundColor: AppColors.background,
      textTheme: text,
      appBarTheme: const AppBarTheme(
        centerTitle: false,
        elevation: 0,
        scrolledUnderElevation: 0,
        backgroundColor: AppColors.background,
        foregroundColor: ink,
        titleTextStyle: TextStyle(
          fontFamily: AppTextStyles.displayFamily,
          fontSize: 25,
          height: 1.2,
          color: ink,
        ),
      ),
      cardTheme: const CardThemeData(
        elevation: 0,
        margin: EdgeInsets.zero,
        color: Colors.white,
        surfaceTintColor: Colors.transparent,
        shape: PixelBorder(side: BorderSide(color: AppColors.outline)),
      ),
      filledButtonTheme: FilledButtonThemeData(
        style: buttons.copyWith(
          backgroundColor: WidgetStateProperty.resolveWith(
            (states) => states.contains(WidgetState.disabled)
                ? const Color(0xFFE2E2E2)
                : ink,
          ),
          foregroundColor: WidgetStateProperty.resolveWith(
            (states) => states.contains(WidgetState.disabled)
                ? const Color(0xFF666666)
                : Colors.white,
          ),
        ),
      ),
      elevatedButtonTheme: ElevatedButtonThemeData(
        style: buttons.copyWith(
          backgroundColor: const WidgetStatePropertyAll(ink),
          foregroundColor: const WidgetStatePropertyAll(Colors.white),
        ),
      ),
      outlinedButtonTheme: OutlinedButtonThemeData(
        style: buttons.copyWith(
          foregroundColor: const WidgetStatePropertyAll(ink),
          side: const WidgetStatePropertyAll(
            BorderSide(color: AppColors.outlineVariant),
          ),
        ),
      ),
      textButtonTheme: TextButtonThemeData(
        style: buttons.copyWith(
          foregroundColor: const WidgetStatePropertyAll(ink),
        ),
      ),
      iconButtonTheme: IconButtonThemeData(
        style: IconButton.styleFrom(
          shape: const PixelBorder(),
          minimumSize: const Size(48, 48),
        ),
      ),
      inputDecorationTheme: const InputDecorationTheme(
        filled: true,
        fillColor: Colors.white,
        border: border,
        enabledBorder: border,
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.zero,
          borderSide: BorderSide(color: ink, width: 2),
        ),
        errorBorder: border,
        contentPadding: EdgeInsets.all(16),
        hintStyle: TextStyle(color: AppColors.textTertiary, fontSize: 14),
        labelStyle: TextStyle(color: AppColors.textSecondary, fontSize: 14),
      ),
      navigationBarTheme: const NavigationBarThemeData(
        backgroundColor: Colors.white,
        indicatorColor: ink,
        iconTheme: WidgetStateProperty.fromMap({
          WidgetState.selected: IconThemeData(color: Colors.white, size: 23),
          WidgetState.any: IconThemeData(
            color: AppColors.textSecondary,
            size: 23,
          ),
        }),
        indicatorShape: PixelBorder(),
        labelTextStyle: WidgetStatePropertyAll(AppTextStyles.labelMedium),
        elevation: 0,
      ),
      bottomNavigationBarTheme: const BottomNavigationBarThemeData(
        backgroundColor: Colors.white,
        selectedItemColor: ink,
        unselectedItemColor: AppColors.textSecondary,
        elevation: 0,
        type: BottomNavigationBarType.fixed,
      ),
      floatingActionButtonTheme: const FloatingActionButtonThemeData(
        backgroundColor: ink,
        foregroundColor: Colors.white,
        elevation: 0,
        shape: PixelBorder(),
      ),
      dividerTheme: const DividerThemeData(
        color: AppColors.outline,
        thickness: 1,
        space: 16,
      ),
      snackBarTheme: const SnackBarThemeData(
        backgroundColor: ink,
        contentTextStyle: TextStyle(
          fontFamily: AppTextStyles.fontFamily,
          color: Colors.white,
          fontSize: 14,
          height: 1.5,
        ),
        shape: PixelBorder(),
        behavior: SnackBarBehavior.floating,
        elevation: 0,
      ),
      dialogTheme: const DialogThemeData(
        backgroundColor: Colors.white,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        shape: PixelBorder(side: BorderSide(color: AppColors.outline)),
        titleTextStyle: TextStyle(
          fontFamily: AppTextStyles.displayFamily,
          fontSize: 27,
          color: ink,
        ),
      ),
      bottomSheetTheme: const BottomSheetThemeData(
        backgroundColor: Colors.white,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        shape: PixelBorder(),
      ),
      chipTheme: ChipThemeData(
        backgroundColor: AppColors.surfaceVariant,
        shape: const PixelBorder(),
        labelStyle: AppTextStyles.labelMedium.copyWith(color: ink),
      ),
      badgeTheme: const BadgeThemeData(
        backgroundColor: ink,
        textColor: Colors.white,
      ),
      progressIndicatorTheme: const ProgressIndicatorThemeData(
        color: ink,
        linearTrackColor: AppColors.surfaceVariant,
      ),
      listTileTheme: const ListTileThemeData(
        contentPadding: EdgeInsets.symmetric(horizontal: 16, vertical: 2),
        titleTextStyle: AppTextStyles.titleSmall,
        subtitleTextStyle: AppTextStyles.bodySmall,
        minLeadingWidth: 24,
        horizontalTitleGap: 12,
        iconColor: ink,
        textColor: ink,
        minVerticalPadding: 8,
      ),
      datePickerTheme: const DatePickerThemeData(
        shape: PixelBorder(),
        dayShape: WidgetStatePropertyAll(PixelBorder()),
        backgroundColor: Colors.white,
        surfaceTintColor: Colors.transparent,
        headerBackgroundColor: Colors.white,
        headerForegroundColor: ink,
      ),
      timePickerTheme: const TimePickerThemeData(
        shape: PixelBorder(),
        hourMinuteShape: PixelBorder(),
        backgroundColor: Colors.white,
      ),
      checkboxTheme: const CheckboxThemeData(
        shape: PixelBorder(),
        side: BorderSide(color: AppColors.outlineVariant),
      ),
      popupMenuTheme: const PopupMenuThemeData(
        shape: PixelBorder(),
        color: Colors.white,
        surfaceTintColor: Colors.transparent,
      ),
      tooltipTheme: const TooltipThemeData(
        decoration: ShapeDecoration(color: ink, shape: PixelBorder()),
        textStyle: TextStyle(
          color: Colors.white,
          fontFamily: AppTextStyles.fontFamily,
        ),
      ),
    );
  }

  // The product currently ships one deliberate monochrome appearance.
  static ThemeData get darkTheme => lightTheme;
}

class AppSpacing {
  AppSpacing._();
  static const double xs = 4, sm = 8, md = 16, lg = 24, xl = 32, xxl = 48;
}

class AppRadius {
  AppRadius._();
  static const double sm = 0, md = 0, lg = 0, xl = 0, full = 999;
}
