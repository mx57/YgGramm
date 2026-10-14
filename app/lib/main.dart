// File: app/lib/main.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'ui/adaptive_shell.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(
    const ProviderScope(
      child: YggGramApp(),
    ),
  );
}

class YggGramApp extends StatelessWidget {
  const YggGramApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'YggGram',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        brightness: Brightness.dark,
        scaffoldBackgroundColor: const Color(0xFF0F172A),
        colorScheme: const ColorScheme.dark(
          primary: Color(0xFF10B981),
          secondary: Color(0xFF06B6D4),
          surface: Color(0xFF0B1120),
        ),
        fontFamily: 'Roboto',
      ),
      home: const AdaptiveShell(),
    );
  }
}
