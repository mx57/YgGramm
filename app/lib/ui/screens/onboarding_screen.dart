// File: app/lib/ui/screens/onboarding_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

class OnboardingScreen extends StatefulWidget {
  const OnboardingScreen({super.key});

  @override
  State<OnboardingScreen> createState() => _OnboardingScreenState();
}

class _OnboardingScreenState extends State<OnboardingScreen> {
  bool _isGenerating = false;
  List<String> _mnemonicWords = [];
  bool _showMnemonic = false;

  final List<String> _sampleMnemonic = [
    'absorb', 'banner', 'cascade', 'dynamic',
    'entropy', 'falcon', 'galaxy', 'harbor',
    'impulse', 'jungle', 'kinetic', 'legacy'
  ];

  void _generateNewIdentity() {
    setState(() {
      _isGenerating = true;
    });

    Future.delayed(const Duration(milliseconds: 600), () {
      setState(() {
        _isGenerating = false;
        _mnemonicWords = _sampleMnemonic;
        _showMnemonic = true;
      });
    });
  }

  void _copyMnemonic() {
    Clipboard.setData(ClipboardData(text: _mnemonicWords.join(' ')));
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Мнемоническая фраза скопирована в буфер обмена')),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFF0F172A),
      appBar: AppBar(
        backgroundColor: const Color(0xFF0B1120),
        title: const Text('Крипто-Профиль YggGram', style: TextStyle(color: Colors.white, fontSize: 16)),
      ),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 540),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                // Shield Header
                Center(
                  child: Container(
                    width: 72,
                    height: 72,
                    decoration: BoxDecoration(
                      gradient: const LinearGradient(
                        colors: [Color(0xFF059669), Color(0xFF06B6D4)],
                      ),
                      borderRadius: BorderRadius.circular(22),
                      boxShadow: [
                        BoxShadow(
                          color: const Color(0xFF10B981).withOpacity(0.2),
                          blurRadius: 20,
                          offset: const Offset(0, 8),
                        ),
                      ],
                    ),
                    child: const Icon(Icons.security, color: Colors.white, size: 36),
                  ),
                ),
                const SizedBox(height: 20),
                const Text(
                  'Децентрализованный Профиль',
                  textAlign: TextAlign.center,
                  style: TextStyle(color: Colors.white, fontSize: 22, fontWeight: FontWeight.bold),
                ),
                const SizedBox(height: 8),
                const Text(
                  'Никаких телефонных номеров или SMS. Ваш профиль базируется на ключевой паре Ed25519 в оверлейной сети Yggdrasil.',
                  textAlign: TextAlign.center,
                  style: TextStyle(color: Color(0xFF94A3B8), fontSize: 13, height: 1.4),
                ),
                const SizedBox(height: 32),

                if (!_showMnemonic) ...[
                  // Option 1: Create New Profile
                  ElevatedButton.icon(
                    style: ElevatedButton.styleFrom(
                      backgroundColor: const Color(0xFF10B981),
                      foregroundColor: const Color(0xFF0F172A),
                      padding: const EdgeInsets.symmetric(vertical: 16),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                    ),
                    onPressed: _isGenerating ? null : _generateNewIdentity,
                    icon: _isGenerating
                        ? const SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2))
                        : const Icon(Icons.add_moderator),
                    label: const Text('Создать новый крипто-профиль (BIP-39)', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 14)),
                  ),
                  const SizedBox(height: 16),

                  // Option 2: Restore from Backup
                  OutlinedButton.icon(
                    style: OutlinedButton.styleFrom(
                      foregroundColor: Colors.white,
                      side: const BorderSide(color: Color(0xFF334155)),
                      padding: const EdgeInsets.symmetric(vertical: 14),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                    ),
                    onPressed: () => _showRestoreModal(context),
                    icon: const Icon(Icons.cloud_download_outlined, color: Color(0xFF06B6D4)),
                    label: const Text('Восстановить из облачного бэкапа'),
                  ),
                  const SizedBox(height: 12),

                  // Option 3: Pair via QR LAN
                  OutlinedButton.icon(
                    style: OutlinedButton.styleFrom(
                      foregroundColor: Colors.white,
                      side: const BorderSide(color: Color(0xFF334155)),
                      padding: const EdgeInsets.symmetric(vertical: 14),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                    ),
                    onPressed: () => _showQRPairingModal(context),
                    icon: const Icon(Icons.qr_code_scanner, color: Color(0xFFA855F7)),
                    label: const Text('Привязать по QR-коду к ПК / Смартфону (LAN)'),
                  ),
                ] else ...[
                  // Display Generated Mnemonic
                  Container(
                    padding: const EdgeInsets.all(18),
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
                            const Text(
                              'Ваша мнемоническая фраза (12 слов):',
                              style: TextStyle(color: Color(0xFF10B981), fontWeight: FontWeight.bold, fontSize: 13),
                            ),
                            IconButton(
                              icon: const Icon(Icons.copy, color: Color(0xFF94A3B8), size: 18),
                              tooltip: 'Скопировать',
                              onPressed: _copyMnemonic,
                            ),
                          ],
                        ),
                        const SizedBox(height: 12),
                        Wrap(
                          spacing: 8,
                          runSpacing: 8,
                          children: List.generate(_mnemonicWords.length, (index) {
                            return Container(
                              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                              decoration: BoxDecoration(
                                color: const Color(0xFF1E293B),
                                borderRadius: BorderRadius.circular(8),
                              ),
                              child: Text(
                                '${index + 1}. ${_mnemonicWords[index]}',
                                style: const TextStyle(color: Colors.white, fontSize: 12, fontFamily: 'monospace'),
                              ),
                            );
                          }),
                        ),
                        const SizedBox(height: 14),
                        const Text(
                          '⚠️ Запишите эту фразу на бумаге! Она является единственным ключом к вашему аккаунту и зашифрованным бэкапам.',
                          style: TextStyle(color: Color(0xFFE2E8F0), fontSize: 11, height: 1.4),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(height: 24),
                  ElevatedButton(
                    style: ElevatedButton.styleFrom(
                      backgroundColor: const Color(0xFF10B981),
                      foregroundColor: const Color(0xFF0F172A),
                      padding: const EdgeInsets.symmetric(vertical: 16),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                    ),
                    onPressed: () {
                      Navigator.pop(context);
                    },
                    child: const Text('Я сохранил фразу • Войти в YggGram', style: TextStyle(fontWeight: FontWeight.bold)),
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }

  void _showRestoreModal(BuildContext context) {
    showModalBottomSheet(
      context: context,
      backgroundColor: const Color(0xFF0F172A),
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (_) {
        return Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Text(
                'Выберите источник бэкапа',
                style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 16),
              ),
              const SizedBox(height: 16),
              _cloudOptionTile(
                icon: Icons.cloud_outlined,
                color: Colors.blue,
                title: 'Google Drive (appDataFolder)',
                subtitle: 'Проксируется через оверлей при блокировках',
                onTap: () => Navigator.pop(context),
              ),
              _cloudOptionTile(
                icon: Icons.circle,
                color: Colors.red,
                title: 'Яндекс.Диск (app:/)',
                subtitle: 'Российское облако на полной скорости в РФ',
                onTap: () => Navigator.pop(context),
              ),
              _cloudOptionTile(
                icon: Icons.folder_shared_outlined,
                color: Colors.cyan,
                title: 'WebDAV (Mail.ru Cloud / Nextcloud)',
                subtitle: 'Собственный сервер или персональный NAS',
                onTap: () => Navigator.pop(context),
              ),
            ],
          ),
        );
      },
    );
  }

  Widget _cloudOptionTile({
    required IconData icon,
    required Color color,
    required String title,
    required String subtitle,
    required VoidCallback onTap,
  }) {
    return ListTile(
      leading: Icon(icon, color: color),
      title: Text(title, style: const TextStyle(color: Colors.white, fontSize: 13, fontWeight: FontWeight.w600)),
      subtitle: Text(subtitle, style: const TextStyle(color: Color(0xFF94A3B8), fontSize: 11)),
      onTap: onTap,
    );
  }

  void _showQRPairingModal(BuildContext context) {
    showDialog(
      context: context,
      builder: (_) => AlertDialog(
        backgroundColor: const Color(0xFF0F172A),
        title: const Text('Привязка к устройству (LAN P2P)', style: TextStyle(color: Colors.white, fontSize: 16)),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 180,
              height: 180,
              decoration: BoxDecoration(
                color: Colors.white,
                borderRadius: BorderRadius.circular(16),
              ),
              child: const Icon(Icons.qr_code, size: 140, color: Colors.black),
            ),
            const SizedBox(height: 16),
            const Text(
              'Отсканируйте этот QR-код в YggGram на вашем ПК или ноутбуке для мгновенного переноса ключей через Wi-Fi.',
              textAlign: TextAlign.center,
              style: TextStyle(color: Color(0xFF94A3B8), fontSize: 12),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Закрыть', style: TextStyle(color: Color(0xFF10B981))),
          ),
        ],
      ),
    );
  }
}
