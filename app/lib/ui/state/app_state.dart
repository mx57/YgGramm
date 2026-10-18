// File: app/lib/ui/state/app_state.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Локальные предпочтения интерфейса. Сеть, идентичность, пиры, диалоги и
/// сообщения живут в [coreProvider] и читаются только через ядро — здесь их
/// больше нет, потому что демонстрационные значения подменяли реальность.
class AppSettings {
  final bool antiDpiOnNextStart;
  final bool mirrorToCloud;
  final int mediaCacheLimitMB;

  const AppSettings({
    this.antiDpiOnNextStart = true,
    this.mirrorToCloud = false,
    this.mediaCacheLimitMB = 2048,
  });

  AppSettings copyWith({
    bool? antiDpiOnNextStart,
    bool? mirrorToCloud,
    int? mediaCacheLimitMB,
  }) {
    return AppSettings(
      antiDpiOnNextStart: antiDpiOnNextStart ?? this.antiDpiOnNextStart,
      mirrorToCloud: mirrorToCloud ?? this.mirrorToCloud,
      mediaCacheLimitMB: mediaCacheLimitMB ?? this.mediaCacheLimitMB,
    );
  }
}

class AppSettingsNotifier extends StateNotifier<AppSettings> {
  AppSettingsNotifier() : super(const AppSettings());

  void setAntiDpiOnNextStart(bool enabled) =>
      state = state.copyWith(antiDpiOnNextStart: enabled);

  void setMirrorToCloud(bool enabled) => state = state.copyWith(mirrorToCloud: enabled);

  void setCacheLimit(int limitMB) => state = state.copyWith(mediaCacheLimitMB: limitMB);
}

final appSettingsProvider =
    StateNotifierProvider<AppSettingsNotifier, AppSettings>((ref) => AppSettingsNotifier());
