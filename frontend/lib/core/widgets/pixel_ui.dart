import 'dart:math' as math;
import 'package:flutter/material.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:go_router/go_router.dart';
import '../theme/app_theme.dart';

void navigateOrix(BuildContext context, String path) => context.go(path);

class OrixLogo extends StatelessWidget {
  final bool symbolOnly;
  final double width;
  const OrixLogo({super.key, this.symbolOnly = false, this.width = 116});
  @override
  Widget build(BuildContext context) => Semantics(
    label: 'Orix',
    image: true,
    child: SvgPicture.asset(
      symbolOnly
          ? 'assets/icons/orix-symbol.svg'
          : 'assets/images/orix_logo.svg',
      width: width,
      excludeFromSemantics: true,
    ),
  );
}

/// One content measure and spacing system for forms, lists and detail screens.
class OrixPage extends StatelessWidget {
  final String title;
  final String? description;
  final List<Widget> children;
  final List<Widget>? actions;
  final Widget? bottom;
  final double maxWidth;
  final bool canPop;
  const OrixPage({
    super.key,
    required this.title,
    this.description,
    required this.children,
    this.actions,
    this.bottom,
    this.maxWidth = 760,
    this.canPop = true,
  });
  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final navigation = bottom;
      final showSideNavigation =
          constraints.maxWidth >= 840 && navigation is OrixNavigation;
      final content = SafeArea(
        top: false,
        child: Align(
          alignment: Alignment.topCenter,
          child: ConstrainedBox(
            constraints: BoxConstraints(maxWidth: maxWidth),
            child: ListView(
              padding: EdgeInsets.fromLTRB(
                constraints.maxWidth < 600 ? 20 : 32,
                16,
                constraints.maxWidth < 600 ? 20 : 32,
                32,
              ),
              children: [
                Text(title, style: Theme.of(context).textTheme.headlineLarge),
                if (description != null)
                  Padding(
                    padding: const EdgeInsets.only(top: 8),
                    child: Text(
                      description!,
                      style: const TextStyle(
                        color: AppColors.textSecondary,
                        height: 1.6,
                      ),
                    ),
                  ),
                const SizedBox(height: 24),
                ...children,
              ],
            ),
          ),
        ),
      );
      return Scaffold(
        appBar: AppBar(
          title: const OrixLogo(width: 100),
          actions: actions,
          automaticallyImplyLeading: canPop,
        ),
        bottomNavigationBar: showSideNavigation ? null : bottom,
        body: showSideNavigation
            ? Row(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  _OrixSideNavigation(selected: navigation.selected),
                  const VerticalDivider(width: 1),
                  Expanded(child: content),
                ],
              )
            : content,
      );
    },
  );
}

class PixelSection extends StatelessWidget {
  final String? title;
  final Widget child;
  const PixelSection({super.key, this.title, required this.child});
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(bottom: 20),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (title != null) ...[
          Text(title!, style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 12),
        ],
        Card(
          child: Padding(padding: const EdgeInsets.all(16), child: child),
        ),
      ],
    ),
  );
}

enum PixelScene {
  loading,
  searching,
  found,
  noMatch,
  liftOffered,
  cancelled,
  travelling,
  completed,
  reportSent,
  uploading,
  uploaded,
  error,
  connectionLost,
}

/// Outcomes and travel settle once; searching and loading loop while waiting. Repaint boundaries
/// isolate the tiny sprite canvas from the surrounding lists and text.
class PixelAnimation extends StatefulWidget {
  final PixelScene scene;
  final double size;
  final String? label;
  final Color? color;
  const PixelAnimation({
    super.key,
    required this.scene,
    this.size = 200,
    this.label,
    this.color,
  });
  @override
  State<PixelAnimation> createState() => _PixelAnimationState();
}

class _PixelAnimationState extends State<PixelAnimation>
    with SingleTickerProviderStateMixin, WidgetsBindingObserver {
  late final AnimationController _clock;
  bool _foreground = true;
  bool get _loop => [
    PixelScene.loading,
    PixelScene.searching,
    PixelScene.uploading,
    PixelScene.connectionLost,
  ].contains(widget.scene);
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _clock = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 1800),
    );
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _sync();
  }

  @override
  void didUpdateWidget(covariant PixelAnimation old) {
    super.didUpdateWidget(old);
    if (old.scene != widget.scene) {
      _clock.reset();
      _sync();
    }
  }

  void _sync() {
    if (!mounted) return;
    final reduce =
        MediaQuery.disableAnimationsOf(context) ||
        WidgetsBinding
            .instance
            .platformDispatcher
            .accessibilityFeatures
            .reduceMotion;
    if (reduce || !TickerMode.valuesOf(context).enabled || !_foreground) {
      _clock.stop();
      _clock.value = 1;
    } else if (_loop) {
      if (!_clock.isAnimating) _clock.repeat();
    } else {
      _clock.forward();
    }
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _foreground = state == AppLifecycleState.resumed;
    _sync();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _clock.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Semantics(
    label: widget.label ?? widget.scene.name,
    image: true,
    child: RepaintBoundary(
      child: SizedBox(
        width: widget.size,
        height: widget.size * .6,
        child: AnimatedBuilder(
          animation: _clock,
          builder: (context, _) => CustomPaint(
            painter: _ScenePainter(
              widget.scene,
              (_clock.value * 18).floor() / 18,
              ink: widget.color ?? AppColors.primary,
              pixelRatio: MediaQuery.devicePixelRatioOf(context),
            ),
          ),
        ),
      ),
    ),
  );
}

class _ScenePainter extends CustomPainter {
  final PixelScene scene;
  final double time;
  final Color ink;
  final double pixelRatio;
  _ScenePainter(
    this.scene,
    this.time, {
    this.ink = AppColors.primary,
    required this.pixelRatio,
  });
  static const gray = Color(0xFF999999), pale = Color(0xFFDDDDDD);
  @override
  void paint(Canvas c, Size size) {
    // Integer physical-pixel coordinates keep the sprite crisp at arbitrary DPI.
    final width = scene == PixelScene.loading ? 48.0 : 120.0;
    final height = scene == PixelScene.loading ? 24.0 : 72.0;
    final physicalScale = math.max(
      1,
      (math.min(size.width / width, size.height / height) * pixelRatio).floor(),
    );
    final s = physicalScale / pixelRatio;
    c.translate(
      (((size.width - width * s) / 2) * pixelRatio).floor() / pixelRatio,
      (((size.height - height * s) / 2) * pixelRatio).floor() / pixelRatio,
    );
    c.scale(s);
    void rect(double x, double y, double w, double h, [Color? color]) =>
        c.drawRect(
          Rect.fromLTWH(x.roundToDouble(), y.roundToDouble(), w, h),
          Paint()
            ..isAntiAlias = false
            ..color = color ?? ink,
        );
    void person(double x, double y, [Color? color]) {
      rect(x + 3, y, 8, 3, color);
      rect(x, y + 3, 14, 8, color);
      rect(x + 3, y + 11, 8, 3, color);
      rect(x + 3, y + 3, 8, 7, AppColors.background);
      rect(x, y + 17, 14, 4, color);
      rect(x, y + 21, 4, 10, color);
      rect(x + 10, y + 21, 4, 10, color);
      rect(x + 4, y + 21, 6, 4, color);
    }

    void car(double x, double y) {
      rect(x + 5, y, 20, 4);
      rect(x + 1, y + 4, 30, 13);
      rect(x - 2, y + 12, 36, 9);
      rect(x + 4, y + 5, 11, 7, AppColors.background);
      rect(x + 18, y + 5, 9, 7, AppColors.background);
      rect(x + 3, y + 21, 6, 4);
      rect(x + 24, y + 21, 6, 4);
      rect(x, y + 15, 3, 3, gray);
      rect(x + 29, y + 15, 3, 3, gray);
    }

    void check(double x, double y) {
      rect(x, y + 5, 4, 4);
      rect(x + 4, y + 9, 4, 4);
      rect(x + 8, y + 5, 4, 4);
      rect(x + 12, y + 1, 4, 4);
      rect(x + 16, y - 3, 4, 4);
    }

    final t = Curves.easeOutCubic.transform(time.clamp(0, 1));
    if (scene == PixelScene.loading) {
      for (var i = 0.0; i < 4; i++) {
        final active = ((time * 4).floor() % 4) == i;
        rect(6 + i * 10, 10 - (active ? 3 : 0), 5, 5, active ? ink : pale);
      }
      return;
    }
    for (var i = 0.0; i < 10; i++) {
      rect(8 + i * 11, 61, 7, 2, pale);
    }
    switch (scene) {
      case PixelScene.searching:
        person(12, 27, gray);
        person(94, 27, gray);
        final q = (time * 4).floor() % 4;
        for (var i = 0.0; i < 4; i++) {
          rect(38 + i * 11, 48, 4, 4, i == q ? ink : pale);
        }
        rect(53, 15, 14, 2);
        rect(51, 17, 2, 12);
        rect(67, 17, 2, 12);
        rect(53, 29, 14, 2);
        rect(68, 30, 7, 3);
      case PixelScene.found:
        person(10 + 25 * t, 27);
        person(96 - 25 * t, 27);
        if (time > .4) {
          check(52, 10);
          for (var i = 0.0; i < 4; i++) {
            rect(26 + i * 21, 12 + (i % 2) * 6, 3, 3, gray);
          }
        }
      case PixelScene.noMatch:
        person(15, 27, gray);
        rect(49, 45, 16 + (1 - t) * 12, 2, gray);
        rect(68, 45, 5, 2, pale);
        final vacancy = 8 * (1 - t);
        rect(87 + vacancy, 42, 13, 2, gray);
        rect(87 + vacancy, 52, 13, 2, gray);
        rect(85 + vacancy, 44, 2, 8, gray);
        rect(100 + vacancy, 44, 2, 8, gray);
        for (var i = 0.0; i < (3 * t).ceil(); i++) {
          rect(51 + i * 5, 24, 2, 2, gray);
        }
      case PixelScene.liftOffered:
        car(25 + 12 * t, 34);
        rect(84, 28 - 8 * t, 3, 20);
        rect(78, 29 - 8 * t, 3, 3);
        rect(81, 26 - 8 * t, 3, 3);
        rect(87, 26 - 8 * t, 3, 3);
        rect(90, 29 - 8 * t, 3, 3);
        person(10, 28, gray);
      case PixelScene.cancelled:
        person(12, 27, gray);
        for (var i = 0.0; i < (6 * (1 - t)).ceil(); i++) {
          rect(34 + i * 9, 48, 5, 2, gray);
        }
        for (var i = 0.0; i < 5; i++) {
          rect(73 + i * 4, 25 + i * 4, 4, 4);
          rect(89 - i * 4, 25 + i * 4, 4, 4);
        }
      case PixelScene.travelling:
        car(43, 32 + (time * 8).floor() % 2);
        for (var i = 0.0; i < 5; i++) {
          rect(((i * 27 - time * 50) % 130) - 5, 56, 12, 2, gray);
        }
        rect(14, 37, 14, 2, pale);
        rect(8, 43, 17, 2, pale);
      case PixelScene.completed:
        person(26, 27);
        rect(85, 16, 3, 44);
        for (var x = 0.0; x < 4; x++) {
          for (var y = 0.0; y < 3; y++) {
            rect(88 + x * 4, 16 + y * 4, 4, 4, (x + y) % 2 == 0 ? ink : pale);
          }
        }
        if (time > .25) check(52, 26);
      case PixelScene.reportSent:
        rect(36, 30 - 10 * t, 46, 26);
        rect(40, 34 - 10 * t, 38, 18, AppColors.background);
        for (var i = 0.0; i < 5; i++) {
          rect(40 + i * 4, 34 + i * 2 - 10 * t, 4, 2);
          rect(74 - i * 4, 34 + i * 2 - 10 * t, 4, 2);
        }
        if (time > .5) check(85, 21);
      case PixelScene.uploading:
      case PixelScene.uploaded:
        rect(25, 14, 42, 43);
        rect(29, 18, 34, 35, AppColors.background);
        person(39, 20);
        if (scene == PixelScene.uploading) {
          final rise = (time * 12).floor() % 12;
          rect(83, 37.0 - rise, 4, 20);
          for (var i = 0.0; i < 3; i++) {
            rect(75 + i * 4, 37 - rise - i * 4, 4, 4);
            rect(91 - i * 4, 37 - rise - i * 4, 4, 4);
          }
        } else if (time > .2) {
          check(77, 29);
        }
      case PixelScene.connectionLost:
        rect(28, 34, 30, 4);
        rect(28, 38, 4, 22);
        rect(54, 38, 4, 22);
        rect(28, 56, 30, 4);
        rect(58, 40, 10, 4);
        rect(64, 44, 4, 8);
        rect(58, 50, 10, 4);
        rect(24, 64, 38, 3, gray);
        final steam = (time * 3).floor() % 3;
        rect(35, 20.0 - steam, 3, 10, gray);
        rect(46, 16.0 - steam, 3, 12, gray);
        rect(78, 22, 20, 4, gray);
        rect(74, 26, 4, 4, gray);
        rect(98, 26, 4, 4, gray);
        rect(82, 32, 12, 4, gray);
        rect(86, 42, 4, 4, gray);
        for (var i = 0.0; i < 3; i++) {
          rect(76 + i * 4, 46 + i * 4, 4, 4);
          rect(92 - i * 4, 46 + i * 4, 4, 4);
        }
      case PixelScene.error:
        person(14, 27, gray);
        final gap = 8 * t;
        rect(40, 56, 26 - gap, 3);
        rect(66 + gap, 56, 36 - gap, 3);
        final warning = 6 * (1 - t);
        rect(64, 24 + warning, 4, 15);
        rect(64, 44 + warning, 4, 4);
        rect(65 - gap, 55, 3, 6, gray);
        rect(64 + gap, 58, 3, 6, gray);
      case PixelScene.loading:
        break;
    }
  }

  @override
  bool shouldRepaint(_ScenePainter old) =>
      old.scene != scene ||
      old.time != time ||
      old.ink != ink ||
      old.pixelRatio != pixelRatio;
}

class PixelLoader extends StatelessWidget {
  final double? strokeWidth;
  final Color? color;
  const PixelLoader({super.key, this.strokeWidth, this.color});
  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final compact =
          constraints.hasBoundedHeight && constraints.maxHeight < 64;
      final ink =
          color ??
          DefaultTextStyle.of(context).style.color ??
          AppColors.primary;
      final mark = ColorFiltered(
        colorFilter: ColorFilter.mode(ink, BlendMode.srcIn),
        child: OrixLogo(symbolOnly: true, width: compact ? 20 : 42),
      );
      if (compact) return Center(child: mark);
      return SizedBox(
        width: 96,
        height: 82,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            mark,
            const SizedBox(height: 8),
            PixelAnimation(
              scene: PixelScene.loading,
              size: 48,
              label: 'Loading',
              color: ink,
            ),
          ],
        ),
      );
    },
  );
}

/// Keep a control's busy state within its normal line height.
class PixelButtonLoader extends StatelessWidget {
  const PixelButtonLoader({super.key});

  @override
  Widget build(BuildContext context) => Semantics(
    label: 'Working',
    liveRegion: true,
    child: const ExcludeSemantics(
      child: SizedBox(width: 24, height: 24, child: PixelLoader()),
    ),
  );
}

const _navigationPaths = ['/home', '/history', '/my-group', '/profile'];
const _navigationLabels = ['Home', 'Journeys', 'Groups', 'You'];
const _navigationIcons = [
  Icons.home_outlined,
  Icons.route_outlined,
  Icons.chat_bubble_outline,
  Icons.person_outline,
];

/// Scrollable so every destination remains reachable in short landscape windows
/// and when the user enlarges system text.
class _OrixSideNavigation extends StatelessWidget {
  final int selected;
  const _OrixSideNavigation({required this.selected});

  @override
  Widget build(BuildContext context) => SizedBox(
    width: 168,
    child: SafeArea(
      top: false,
      child: ListView(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 24),
        children: [
          for (var index = 0; index < _navigationPaths.length; index++)
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: Semantics(
                selected: index == selected,
                child: TextButton(
                  style: TextButton.styleFrom(
                    foregroundColor: index == selected
                        ? Colors.white
                        : AppColors.textPrimary,
                    backgroundColor: index == selected
                        ? AppColors.primary
                        : Colors.transparent,
                    padding: const EdgeInsets.all(12),
                    alignment: Alignment.centerLeft,
                  ),
                  onPressed: () {
                    if (index != selected) {
                      navigateOrix(context, _navigationPaths[index]);
                    }
                  },
                  child: Wrap(
                    spacing: 12,
                    runSpacing: 8,
                    crossAxisAlignment: WrapCrossAlignment.center,
                    children: [
                      Icon(_navigationIcons[index], size: 22),
                      Text(_navigationLabels[index]),
                    ],
                  ),
                ),
              ),
            ),
        ],
      ),
    ),
  );
}

class OrixNavigation extends StatelessWidget {
  final int selected;
  const OrixNavigation({super.key, required this.selected});
  @override
  Widget build(BuildContext context) => DecoratedBox(
    decoration: const BoxDecoration(
      border: Border(top: BorderSide(color: AppColors.outline)),
    ),
    child: NavigationBar(
      height: 72,
      selectedIndex: selected,
      onDestinationSelected: (index) {
        if (index != selected) navigateOrix(context, _navigationPaths[index]);
      },
      destinations: const [
        NavigationDestination(
          icon: Icon(Icons.home_outlined),
          selectedIcon: Icon(Icons.home),
          label: 'Home',
        ),
        NavigationDestination(
          icon: Icon(Icons.route_outlined),
          label: 'Journeys',
        ),
        NavigationDestination(
          icon: Icon(Icons.chat_bubble_outline),
          label: 'Groups',
        ),
        NavigationDestination(icon: Icon(Icons.person_outline), label: 'You'),
      ],
    ),
  );
}

class OrixMessageBubble extends StatelessWidget {
  final String author, content;
  final String? time;
  final bool mine;
  final Widget? status;
  const OrixMessageBubble({
    super.key,
    required this.author,
    required this.content,
    this.time,
    required this.mine,
    this.status,
  });
  @override
  Widget build(BuildContext context) => Align(
    alignment: mine ? Alignment.centerRight : Alignment.centerLeft,
    child: FractionallySizedBox(
      widthFactor: .9,
      child: Align(
        alignment: mine ? Alignment.centerRight : Alignment.centerLeft,
        child: Container(
          constraints: const BoxConstraints(maxWidth: 540),
          margin: const EdgeInsets.only(bottom: 20),
          padding: const EdgeInsets.all(18),
          decoration: ShapeDecoration(
            color: mine ? AppColors.primary : Colors.white,
            shape: const PixelBorder(
              side: BorderSide(color: AppColors.outline),
            ),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                author,
                style: TextStyle(
                  fontSize: 12,
                  color: mine ? Colors.white70 : AppColors.textSecondary,
                ),
              ),
              const SizedBox(height: 8),
              SelectableText(
                content,
                style: TextStyle(
                  color: mine ? Colors.white : AppColors.textPrimary,
                  fontSize: 14,
                  height: 1.65,
                ),
              ),
              if (time != null) ...[
                const SizedBox(height: 8),
                Text(
                  time!,
                  style: TextStyle(
                    fontSize: 11,
                    color: mine ? Colors.white70 : AppColors.textSecondary,
                  ),
                ),
              ],
              if (status != null) status!,
            ],
          ),
        ),
      ),
    ),
  );
}

class PixelToggle extends StatelessWidget {
  final Widget title;
  final Widget? subtitle;
  final bool value;
  final ValueChanged<bool>? onChanged;
  final EdgeInsetsGeometry? contentPadding;
  const PixelToggle({
    super.key,
    required this.title,
    this.subtitle,
    required this.value,
    required this.onChanged,
    this.contentPadding,
  });
  @override
  Widget build(BuildContext context) => MergeSemantics(
    child: Semantics(
      toggled: value,
      enabled: onChanged != null,
      child: InkWell(
        onTap: onChanged == null ? null : () => onChanged!(!value),
        child: Padding(
          padding: contentPadding ?? const EdgeInsets.symmetric(vertical: 12),
          child: Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    DefaultTextStyle.merge(
                      style: AppTextStyles.titleSmall,
                      child: title,
                    ),
                    if (subtitle != null) ...[
                      const SizedBox(height: 6),
                      DefaultTextStyle.merge(
                        style: AppTextStyles.bodySmall.copyWith(
                          color: AppColors.textSecondary,
                        ),
                        child: subtitle!,
                      ),
                    ],
                  ],
                ),
              ),
              const SizedBox(width: 20),
              SizedBox(
                width: 48,
                height: 48,
                child: Center(
                  child: Container(
                    width: 44,
                    height: 28,
                    padding: const EdgeInsets.all(4),
                    decoration: ShapeDecoration(
                      color: value
                          ? AppColors.primary
                          : const Color(0xFF888888),
                      shape: const PixelBorder(),
                    ),
                    child: Align(
                      alignment: value
                          ? Alignment.centerRight
                          : Alignment.centerLeft,
                      child: Container(
                        width: 18,
                        height: 18,
                        color: Colors.white,
                      ),
                    ),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    ),
  );
}
