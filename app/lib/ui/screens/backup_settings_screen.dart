// File: app/lib/ui/screens/backup_settings_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../state/app_state.dart';

class BackupSettingsScreen extends ConsumerWidget {
  const BackupSettingsScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(appStateProvider);

    return Scaffold(
      backgroundColor: const Color(0xFF0F172A),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Hero Status Card
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
                          color: const Color(0xFF06B6D4).withOpacity(0.15),
                          borderRadius: BorderRadius.circular(10),
                        ),
                        child: const Icon(Icons.cloud_sync, color: Color(0xFF06B6D4), size: 24),
                      ),
                      const SizedBox(width: 12),
                      const Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              'Zero-Knowledge Облачный Бэкап',
                              style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 16),
                            ),
                            Text(
                              'FastCDC дедупликация • Сжатие zstd • Ключи Argon2id',
                              style: TextStyle(color: Color(0xFF94A3B8), fontSize: 11),
                            ),
                          ],
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 16),
                  const Text(
                    'Ни Google, ни Яндекс не имеют доступа к содержимому ваших чатов. В облако выгружаются только анонимные 256КБ чанки chunks/xx/xxxx.bin и зашифрованный корень manifest.enc.',
                    style: TextStyle(color: Color(0xFFE2E8F0), fontSize: 12, height: 1.4),
                  ),
                  const SizedBox(height: 16),
                  ElevatedButton.icon(
                    style: ElevatedButton.styleFrom(
                      backgroundColor: const Color(0xFF10B981),
                      foregroundColor: const Color(0xFF0F172A),
                      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                    ),
                    onPressed: () {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(content: Text('Синхронизация запущена: FastCDC чанкинг -> Зеркалирование в 2 облака.')),
                      );
                    },
                    icon: const Icon(Icons.sync),
                    label: const Text('Синхронизировать сейчас (24.2 MB)', style: TextStyle(fontWeight: FontWeight.bold)),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 24),

            // Cloud Mirroring Toggles
            const Text('Подключенные облачные провайдеры', style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 14)),
            const SizedBox(height: 12),
            Container(
              decoration: BoxDecoration(
                color: const Color(0xFF0B1120),
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: const Color(0xFF1E293B)),
              ),
              child: Column(
                children: [
                  SwitchListTile(
                    activeColor: const Color(0xFF10B981),
                    secondary: const Icon(Icons.cloud_outlined, color: Colors.blue),
                    title: const Text('Google Drive (appDataFolder)', style: TextStyle(color: Colors.white, fontSize: 13, fontWeight: FontWeight.w600)),
                    subtitle: const Text('Автоматически шифрует и сохраняет копию в скрытой системной папке', style: TextStyle(color: Color(0xFF94A3B8), fontSize: 11)),
                    value: state.isGDriveConnected,
                    onChanged: (val) {
                      ref.read(appStateProvider.notifier).toggleGDrive(val);
                    },
                  ),
                  const Divider(color: Color(0xFF1E293B), height: 1),
                  SwitchListTile(
                    activeColor: const Color(0xFF10B981),
                    secondary: const Icon(Icons.circle, color: Colors.red),
                    title: const Text('Яндекс.Диск (app:/)', style: TextStyle(color: Colors.white, fontSize: 13, fontWeight: FontWeight.w600)),
                    subtitle: const Text('Работает в РФ на максимальной скорости даже при полной блокировке внешнего интернета', style: TextStyle(color: Color(0xFF94A3B8), fontSize: 11)),
                    value: state.isYandexConnected,
                    onChanged: (val) {
                      ref.read(appStateProvider.notifier).toggleYandex(val);
                    },
                  ),
                  const Divider(color: Color(0xFF1E293B), height: 1),
                  SwitchListTile(
                    activeColor: const Color(0xFF10B981),
                    secondary: const Icon(Icons.copy_all, color: Color(0xFFA855F7)),
                    title: const Text('Зеркалирование (Одновременный бэкап в 2 облака)', style: TextStyle(color: Colors.white, fontSize: 13, fontWeight: FontWeight.w600)),
                    subtitle: const Text('Гарантирует доступ к переписке, даже если один из сервисов будет заблокирован или недоступен', style: TextStyle(color: Color(0xFF94A3B8), fontSize: 11)),
                    value: state.isMirroringEnabled,
                    onChanged: (val) {
                      ref.read(appStateProvider.notifier).toggleMirroring(val);
                    },
                  ),
                ],
              ),
            ),
            const SizedBox(height: 24),

            // Cache & Storage Limits
            const Text('Локальное дисковое пространство', style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 14)),
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
                      const Text('Лимит кэша медиа и чанков:', style: TextStyle(color: Colors.white, fontSize: 13)),
                      Text('${state.mediaCacheLimitMB} MB', style: const TextStyle(color: Color(0xFF10B981), fontWeight: FontWeight.bold, fontSize: 13)),
                    ],
                  ),
                  Slider(
                    activeColor: const Color(0xFF10B981),
                    inactiveColor: const Color(0xFF334155),
                    min: 512,
                    max: 8192,
                    divisions: 15,
                    value: state.mediaCacheLimitMB.toDouble(),
                    onChanged: (val) {
                      ref.read(appStateProvider.notifier).setCacheLimit(val.toInt());
                    },
                  ),
                  const Text(
                    'Старые зашифрованные медиафайлы автоматически вытесняются с диска при превышении лимита и повторно подгружаются из P2P-роя или облака по требованию.',
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
