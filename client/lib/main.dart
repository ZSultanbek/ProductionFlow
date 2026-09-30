import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'core/router.dart';
import 'core/theme.dart';

void main() {
  // ProviderScope is installed now so the first feature that needs state does
  // not have to touch this file. Nothing reads from it yet.
  runApp(const ProviderScope(child: ProductionFlowApp()));
}

class ProductionFlowApp extends StatefulWidget {
  const ProductionFlowApp({super.key});

  @override
  State<ProductionFlowApp> createState() => _ProductionFlowAppState();
}

class _ProductionFlowAppState extends State<ProductionFlowApp> {
  // One router per app instance, built once and kept for the widget's lifetime.
  final GoRouter _router = createAppRouter();

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'ProductionFlow',
      debugShowCheckedModeBanner: false,
      routerConfig: _router,
      theme: AppTheme.light,
      darkTheme: AppTheme.dark,
      // Follow the device. Both palettes are designed, so neither is a fallback.
      themeMode: ThemeMode.system,
    );
  }
}
