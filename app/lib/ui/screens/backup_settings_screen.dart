// File: app/lib/ui/screens/backup_settings_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../state/app_state.dart';

/// Экран настроек бэкапа. Данные о реальном состоянии бэкапа здесь отсутствуют
/// намеренно: ядро исполняет CreateBackup и RestoreBackup по-настоящему с
/// Шага 13, а с Шага 14 умеет класть пакет не только в каталог, но и на
/// WebDAV-сервер; с Шага 15 повторный пакет в ту же цель не портит прежнюю
/// точку восстановления — у него свой слот заголовка, неизменное не выкладывается
/// заново, а ротация держит последние три пакета. Но у этих команд пока нет
/// вызывающего из интерфейса, а поставленные FFI-бинарники собраны раньше них.
class BackupSettingsScreen extends ConsumerWidget {
  const BackupSettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final settings = ref.watch(appSettingsProvider);
    final notifier = ref.read(appSettingsProvider.notifier);

    return Scaffold(
      backgroundColor: const Color(0xFF0F172A),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              padding: const EdgeInsets.all(20),
              decoration: BoxDecoration(
                gradient: const LinearGradient(
                  colors: [Color(0xFF0B1120), Color(0xFF1E293B)],
                  begin: Alignment.topLeft,
                  end: Alignment.bottomRight,
                ),
                borderRadius: BorderRadius.circular(18),
                border: Border.all(color: const Color(0xFF334155)),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Container(
                        padding: const EdgeInsets.all(8),
                        decoration: BoxDecoration(
                          color: const Color(0xFF06B6D4).withValues(alpha: 0.15),
                          borderRadius: BorderRadius.circular(10),
                        ),
                        child: const Icon(Icons.cloud_sync,
                            color: Color(0xFF06B6D4), size: 24),
                      ),
                      const SizedBox(width: 12),
                      const Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              'Zero-Knowledge облачный бэкап',
                              style: TextStyle(
                                  color: Colors.white,
                                  fontWeight: FontWeight.bold,
                                  fontSize: 16),
                            ),
                            Text(
                              'Формат в ядре готов: чанки + зашифрованный manifest, '
                              'приёмники — каталог на диске и WebDAV',
                              style: TextStyle(color: Color(0xFF94A3B8), fontSize: 11),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 16),
                  Container(
                    padding: const EdgeInsets.all(14),
                    decoration: BoxDecoration(
                      color: const Color(0xFF7F1D1D).withValues(alpha: 0.3),
                      borderRadius: BorderRadius.circular(12),
                      border: Border.all(
                          color: const Color(0xFFF87171).withValues(alpha: 0.4)),
                    ),
                    child: const Row(
                      children: [
                        Icon(Icons.info_outline,
                            color: Color(0xFFF87171), size: 18),
                        SizedBox(width: 10),
                        Expanded(
                          child: Text(
                            'Экрана бэкапа в интерфейсе ещё нет: ядро исполняет CreateBackup и '
                            'RestoreBackup по-настоящему (Шаг 13), но ни одна кнопка их не зовёт, '
                            'а поставленные FFI-бинарники собраны раньше этих команд.',
                            style: TextStyle(
                                color: Color(0xFFFCA5A5), fontSize: 12, height: 1.4),
                          ),
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 24),

            const Text('Локальные предпочтения',
                style: TextStyle(
                    color: Colors.white, fontWeight: FontWeight.bold, fontSize: 14)),
            const SizedBox(height: 12),
            // Тот же случай, что и на экране сети: SwitchListTile красит
            // подложку и всплеск на ближайшем Material, поэтому цветной
            // Container вокруг него в debug-сборке роняет assertion. Падение на
            // этой подмене проверено, а не предполагано.
            Material(
              color: const Color(0xFF0B1120),
              shape: const RoundedRectangleBorder(
                borderRadius: BorderRadius.all(Radius.circular(16)),
                side: BorderSide(color: Color(0xFF1E293B)),
              ),
              child: Column(
                children: [
                  SwitchListTile(
                    activeThumbColor: const Color(0xFF10B981),
                    secondary: const Icon(Icons.copy_all, color: Color(0xFFA855F7)),
                    title: const Text('Зеркалировать бэкап в два облака',
                        style: TextStyle(
                            color: Colors.white,
                            fontSize: 13,
                            fontWeight: FontWeight.w600)),
                    subtitle: const Text(
                        'Пока только намерение: сохраняется в памяти интерфейса и никуда не передаётся',
                        style: TextStyle(color: Color(0xFF94A3B8), fontSize: 11)),
                    value: settings.mirrorToCloud,
                    onChanged: notifier.setMirrorToCloud,
                  ),
                ],
              ),
            ),
            const SizedBox(height: 24),

            const Text('Локальное дисковое пространство',
                style: TextStyle(
                    color: Colors.white, fontWeight: FontWeight.bold, fontSize: 14)),
            const SizedBox(height: 12),
            Container(
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: const Color(0xFF0B1120),
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: const Color(0xFF1E293B)),
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      const Text('Лимит кэша медиа:',
                          style: TextStyle(color: Colors.white, fontSize: 13)),
                      Text('${settings.mediaCacheLimitMB} MB',
                          style: const TextStyle(
                              color: Color(0xFF10B981),
                              fontWeight: FontWeight.bold,
                              fontSize: 13)),
                    ],
                  ),
                  Slider(
                    activeColor: const Color(0xFF10B981),
                    inactiveColor: const Color(0xFF334155),
                    min: 512,
                    max: 8192,
                    divisions: 15,
                    value: settings.mediaCacheLimitMB.toDouble(),
                    onChanged: (val) => notifier.setCacheLimit(val.toInt()),
                  ),
                  const Text(
                    'Лимит пока ни к чему не применяется: вытеснение кэша появится '
                    'вместе с реальным хранилищем медиа.',
                    style: TextStyle(color: Color(0xFF94A3B8), fontSize: 11),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
