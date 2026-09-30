import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

/// The single source of truth for how ProductionFlow looks.
///
/// Rules this file keeps, deliberately:
///   * **No gradients.** Anywhere. Depth comes from M3 surface containers —
///     `surface` → `surfaceContainerLow` → `surfaceContainer` → `…High` — not
///     from shadows or gradients. Cards sit at elevation 0 with a hairline
///     [ColorScheme.outlineVariant] border.
///   * **One accent.** [ColorScheme.tertiary] (ochre) marks the one thing on a
///     screen that matters most. If two things on a screen use it, one of them
///     is wrong.
///   * Both palettes grow from the same dark brown seed, then the surface and
///     outline roles are pinned by hand: `fromSeed` alone produces greys that
///     are technically warm but read as grey. Light should read cream, dark
///     should read espresso.
class AppTheme {
  const AppTheme._();

  /// Dark brown. Everything else is derived from or tuned around this.
  static const Color seed = Color(0xFF4A2B18);

  /// Corner radii. M3 shapes, nothing custom.
  static const double radiusCard = 16;
  static const double radiusControl = 12;

  /// One spacing rhythm, used everywhere. Whitespace is a feature here — the
  /// people reading this app are on set, tired, looking at a phone in daylight.
  static const double space1 = 4;
  static const double space2 = 8;
  static const double space3 = 12;
  static const double space4 = 16;
  static const double space5 = 24;
  static const double space6 = 32;
  static const double space7 = 48;
  static const double space8 = 72;

  /// Layouts switch from phone to desktop at this width.
  static const double breakpoint = 900;

  /// Text never runs wider than this; long lines are tiring to read.
  static const double maxContentWidth = 1120;

  static ThemeData get light => _build(_lightScheme);
  static ThemeData get dark => _build(_darkScheme);

  // --- Palettes -------------------------------------------------------------

  /// Warm beige and cream. `DynamicSchemeVariant.content` keeps more of the
  /// seed's chroma than the default `tonalSpot`, which matters for a brown.
  static final ColorScheme _lightScheme = ColorScheme.fromSeed(
    seedColor: seed,
    dynamicSchemeVariant: DynamicSchemeVariant.content,
  ).copyWith(
    primary: const Color(0xFF7A4A24),
    onPrimary: const Color(0xFFFFF8F0),
    primaryContainer: const Color(0xFFF0DCC6),
    onPrimaryContainer: const Color(0xFF2E1A0A),

    secondary: const Color(0xFF6B5B4A),
    onSecondary: const Color(0xFFFFF8F0),
    secondaryContainer: const Color(0xFFE9DFD1),
    onSecondaryContainer: const Color(0xFF261D14),

    // The accent. Ochre, so it reads as emphasis rather than as a warning.
    tertiary: const Color(0xFF8F5A0F),
    onTertiary: const Color(0xFFFFF8F0),
    tertiaryContainer: const Color(0xFFFBE4B8),
    onTertiaryContainer: const Color(0xFF2C1C00),

    // Cream, warming as it goes up the elevation ladder.
    surface: const Color(0xFFFBF6EE),
    onSurface: const Color(0xFF241A12),
    onSurfaceVariant: const Color(0xFF5C4C3D),
    surfaceContainerLowest: const Color(0xFFFFFFFF),
    surfaceContainerLow: const Color(0xFFF7F0E6),
    surfaceContainer: const Color(0xFFF2EADE),
    surfaceContainerHigh: const Color(0xFFECE3D5),
    surfaceContainerHighest: const Color(0xFFE6DCCB),
    inverseSurface: const Color(0xFF362A21),
    onInverseSurface: const Color(0xFFF7EEE3),

    outline: const Color(0xFF8C7A68),
    outlineVariant: const Color(0xFFD9CCBA),

    error: const Color(0xFF8E2F24),
    onError: const Color(0xFFFFF8F0),
    errorContainer: const Color(0xFFF9DCD7),
    onErrorContainer: const Color(0xFF370C07),
  );

  /// Espresso. Near-black, but brown near-black — no blue in it anywhere.
  static final ColorScheme _darkScheme = ColorScheme.fromSeed(
    seedColor: seed,
    brightness: Brightness.dark,
    dynamicSchemeVariant: DynamicSchemeVariant.content,
  ).copyWith(
    primary: const Color(0xFFEAB78C),
    onPrimary: const Color(0xFF3E2110),
    primaryContainer: const Color(0xFF5A321A),
    onPrimaryContainer: const Color(0xFFFFDCC2),

    secondary: const Color(0xFFD5C3AE),
    onSecondary: const Color(0xFF382E23),
    secondaryContainer: const Color(0xFF4A3D30),
    onSecondaryContainer: const Color(0xFFF1E3D2),

    tertiary: const Color(0xFFEBC07A),
    onTertiary: const Color(0xFF412C00),
    tertiaryContainer: const Color(0xFF5D4208),
    onTertiaryContainer: const Color(0xFFFFE2AE),

    surface: const Color(0xFF181210),
    onSurface: const Color(0xFFF2E6D8),
    onSurfaceVariant: const Color(0xFFCBB9A6),
    surfaceContainerLowest: const Color(0xFF120D0B),
    surfaceContainerLow: const Color(0xFF1F1815),
    surfaceContainer: const Color(0xFF251D18),
    surfaceContainerHigh: const Color(0xFF30251F),
    surfaceContainerHighest: const Color(0xFF3B2F27),
    inverseSurface: const Color(0xFFF2E6D8),
    onInverseSurface: const Color(0xFF241A12),

    outline: const Color(0xFF9E8A76),
    outlineVariant: const Color(0xFF4A3C32),

    error: const Color(0xFFF2B4A8),
    onError: const Color(0xFF561610),
    errorContainer: const Color(0xFF73241B),
    onErrorContainer: const Color(0xFFFFDAD3),
  );

  // --- Type -----------------------------------------------------------------

  /// Fraunces for display and headline, Inter for everything you actually read.
  /// The scale is deliberately wide: 52 down to 12, with little in the middle,
  /// so hierarchy is obvious at a glance instead of needing to be compared.
  static TextTheme _textTheme(ColorScheme scheme) {
    final display = GoogleFonts.frauncesTextTheme();
    final body = GoogleFonts.interTextTheme();

    return TextTheme(
      displayLarge: display.displayLarge?.copyWith(
        fontSize: 60,
        height: 1.02,
        fontWeight: FontWeight.w600,
        letterSpacing: -1.6,
      ),
      displayMedium: display.displayMedium?.copyWith(
        fontSize: 52,
        height: 1.04,
        fontWeight: FontWeight.w600,
        letterSpacing: -1.3,
      ),
      displaySmall: display.displaySmall?.copyWith(
        fontSize: 40,
        height: 1.08,
        fontWeight: FontWeight.w600,
        letterSpacing: -0.9,
      ),
      headlineLarge: display.headlineLarge?.copyWith(
        fontSize: 32,
        height: 1.14,
        fontWeight: FontWeight.w600,
        letterSpacing: -0.6,
      ),
      headlineMedium: display.headlineMedium?.copyWith(
        fontSize: 26,
        height: 1.2,
        fontWeight: FontWeight.w600,
        letterSpacing: -0.3,
      ),
      headlineSmall: display.headlineSmall?.copyWith(
        fontSize: 21,
        height: 1.26,
        fontWeight: FontWeight.w600,
      ),
      titleLarge: body.titleLarge?.copyWith(
        fontSize: 19,
        height: 1.3,
        fontWeight: FontWeight.w600,
        letterSpacing: -0.2,
      ),
      titleMedium: body.titleMedium?.copyWith(
        fontSize: 16,
        height: 1.35,
        fontWeight: FontWeight.w600,
      ),
      titleSmall: body.titleSmall?.copyWith(
        fontSize: 14,
        height: 1.4,
        fontWeight: FontWeight.w600,
      ),
      bodyLarge: body.bodyLarge?.copyWith(fontSize: 17, height: 1.55),
      bodyMedium: body.bodyMedium?.copyWith(fontSize: 15, height: 1.55),
      bodySmall: body.bodySmall?.copyWith(
        fontSize: 13,
        height: 1.5,
        color: scheme.onSurfaceVariant,
      ),
      labelLarge: body.labelLarge?.copyWith(
        fontSize: 14,
        fontWeight: FontWeight.w600,
        letterSpacing: 0.1,
      ),
      labelMedium: body.labelMedium?.copyWith(
        fontSize: 12,
        fontWeight: FontWeight.w600,
        letterSpacing: 0.4,
      ),
      // Used for the eyebrow labels above headings.
      labelSmall: body.labelSmall?.copyWith(
        fontSize: 12,
        fontWeight: FontWeight.w600,
        letterSpacing: 1.1,
      ),
    ).apply(bodyColor: scheme.onSurface, displayColor: scheme.onSurface);
  }

  // --- Component themes -----------------------------------------------------

  static ThemeData _build(ColorScheme scheme) {
    final text = _textTheme(scheme);
    final controlShape = RoundedRectangleBorder(
      borderRadius: BorderRadius.circular(radiusControl),
    );

    return ThemeData(
      useMaterial3: true,
      colorScheme: scheme,
      scaffoldBackgroundColor: scheme.surface,
      textTheme: text,

      // Flat by default: no drop shadows, no tint-on-scroll surprises.
      appBarTheme: AppBarTheme(
        backgroundColor: scheme.surface,
        foregroundColor: scheme.onSurface,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        scrolledUnderElevation: 0,
        centerTitle: false,
        titleTextStyle: text.titleLarge,
      ),

      cardTheme: CardThemeData(
        color: scheme.surfaceContainerLow,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        margin: EdgeInsets.zero,
        shape: RoundedRectangleBorder(
          borderRadius: BorderRadius.circular(radiusCard),
          side: BorderSide(color: scheme.outlineVariant),
        ),
      ),

      dividerTheme: DividerThemeData(
        color: scheme.outlineVariant,
        thickness: 1,
        space: space5,
      ),

      filledButtonTheme: FilledButtonThemeData(
        style: FilledButton.styleFrom(
          shape: controlShape,
          padding: const EdgeInsets.symmetric(horizontal: space5, vertical: space4),
          textStyle: text.labelLarge,
          minimumSize: const Size(0, 52),
          elevation: 0,
        ),
      ),

      outlinedButtonTheme: OutlinedButtonThemeData(
        style: OutlinedButton.styleFrom(
          shape: controlShape,
          padding: const EdgeInsets.symmetric(horizontal: space5, vertical: space4),
          textStyle: text.labelLarge,
          minimumSize: const Size(0, 52),
          side: BorderSide(color: scheme.outline),
        ),
      ),

      textButtonTheme: TextButtonThemeData(
        style: TextButton.styleFrom(
          shape: controlShape,
          padding: const EdgeInsets.symmetric(horizontal: space3, vertical: space3),
          textStyle: text.labelLarge,
        ),
      ),

      chipTheme: ChipThemeData(
        backgroundColor: scheme.surfaceContainerHigh,
        side: BorderSide(color: scheme.outlineVariant),
        labelStyle: text.labelMedium,
        shape: const StadiumBorder(),
        padding: const EdgeInsets.symmetric(horizontal: space3, vertical: space2),
      ),

      inputDecorationTheme: InputDecorationTheme(
        filled: true,
        fillColor: scheme.surfaceContainerLow,
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radiusControl),
          borderSide: BorderSide(color: scheme.outlineVariant),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radiusControl),
          borderSide: BorderSide(color: scheme.outlineVariant),
        ),
        focusedBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(radiusControl),
          borderSide: BorderSide(color: scheme.primary, width: 2),
        ),
        contentPadding: const EdgeInsets.symmetric(horizontal: space4, vertical: space4),
      ),

      navigationBarTheme: NavigationBarThemeData(
        backgroundColor: scheme.surfaceContainerLow,
        indicatorColor: scheme.primaryContainer,
        surfaceTintColor: Colors.transparent,
        elevation: 0,
        height: 68,
        labelTextStyle: WidgetStatePropertyAll(text.labelMedium),
      ),

      navigationRailTheme: NavigationRailThemeData(
        backgroundColor: scheme.surfaceContainerLow,
        indicatorColor: scheme.primaryContainer,
        selectedLabelTextStyle: text.labelMedium?.copyWith(color: scheme.onSurface),
        unselectedLabelTextStyle: text.labelMedium?.copyWith(color: scheme.onSurfaceVariant),
        elevation: 0,
      ),

      listTileTheme: ListTileThemeData(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(radiusControl)),
        titleTextStyle: text.titleMedium,
        subtitleTextStyle: text.bodySmall,
      ),

      snackBarTheme: SnackBarThemeData(
        backgroundColor: scheme.inverseSurface,
        contentTextStyle: text.bodyMedium?.copyWith(color: scheme.onInverseSurface),
        behavior: SnackBarBehavior.floating,
        elevation: 0,
        shape: controlShape,
      ),

      splashFactory: InkSparkle.splashFactory,
    );
  }
}
