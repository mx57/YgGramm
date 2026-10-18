// File: app/lib/ui/screens/onboarding_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../state/core_state.dart';

class OnboardingScreen extends ConsumerStatefulWidget {
  const OnboardingScreen({super.key});

  @override
  ConsumerState<OnboardingScreen> createState() => _OnboardingScreenState();
}

class _OnboardingScreenState extends ConsumerState<OnboardingScreen> {
  bool _busy = false;
  String? _mnemonic;
  String? _error;

  Future<void> _run(Future<void> Function() action) async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await action();
    } on Object catch (e) {
      setState(() => _error = '$e');
    } finally {
      setState(() => _busy = false);
    }
  }

  void _generateNewIdentity() {
    _run(() async {
      final mnemonic = await ref.read(coreProvider.notifier).createIdentity();
      setState(() => _mnemonic = mnemonic);
    });
  }

  void _restoreFromMnemonic() {
    showModalBottomSheet(
      context: context,
      backgroundColor: const Color(0xFF0F172A),
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (sheetContext) => _RestoreSheet(
        onSubmit: (phrase) {
          Navigator.pop(sheetContext);
          _run(() async {
            await ref.read(coreProvider.notifier).restoreIdentity(phrase);
            await ref.read(coreProvider.notifier).refresh();
            if (mounted) Navigator.pop(context);
          });
        },
      ),
    );
  }

  void _copyMnemonic() {
    Clipboard.setData(ClipboardData(text: _mnemonic!));
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Фраза скопирована в буфер обмена')),
    );
  }

  @override
  Widget build(BuildContext context) {
    final phase = ref.watch(coreProvider).phase;

    return Scaffold(
      backgroundColor: const Color(0xFF0F172A),
      appBar: AppBar(
        backgroundColor: const Color(0xFF0B1120),
        title: const Text('Крипто-профиль YggGram',
            style: TextStyle(color: Colors.white, fontSize: 16)),
      ),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 540),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
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
                          color: const Color(0xFF10B981).withValues(alpha: 0.2),
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
                  'Децентрализованный профиль',
                  textAlign: TextAlign.center,
                  style: TextStyle(
                      color: Colors.white, fontSize: 22, fontWeight: FontWeight.bold),
                ),
                const SizedBox(height: 8),
                const Text(
                  'Никаких телефонных номеров или SMS. Профиль — это ключевая пара Ed25519, '
                  'а её адресом служит IPv6 в оверлейной сети Yggdrasil.',
                  textAlign: TextAlign.center,
                  style: TextStyle(color: Color(0xFF94A3B8), fontSize: 13, height: 1.4),
                ),
                const SizedBox(height: 32),

                if (_error != null) ...[
                  _ErrorBanner(message: _error!, onRetry: _generateNewIdentity),
                  const SizedBox(height: 16),
                ],

                if (_mnemonic == null) ...[
                  ElevatedButton.icon(
                    style: ElevatedButton.styleFrom(
                      backgroundColor: const Color(0xFF10B981),
                      foregroundColor: const Color(0xFF0F172A),
                      padding: const EdgeInsets.symmetric(vertical: 16),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                    ),
                    onPressed: _busy || phase == CorePhase.starting
                        ? null
                        : _generateNewIdentity,
                    icon: _busy
                        ? const SizedBox(
                            width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2))
                        : const Icon(Icons.add_moderator),
                    label: const Text(
                      'Создать новый крипто-профиль (12 слов от ядра)',
                      style: TextStyle(fontWeight: FontWeight.bold, fontSize: 14),
                    ),
                  ),
                  const SizedBox(height: 16),
                  OutlinedButton.icon(
                    style: OutlinedButton.styleFrom(
                      foregroundColor: Colors.white,
                      side: const BorderSide(color: Color(0xFF334155)),
                      padding: const EdgeInsets.symmetric(vertical: 14),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                    ),
                    onPressed: _busy ? null : _restoreFromMnemonic,
                    icon: const Icon(Icons.keyboard_outlined, color: Color(0xFF06B6D4)),
                    label: const Text('У меня уже есть фраза — восстановить профиль'),
                  ),
                  const SizedBox(height: 20),
                  const Text(
                    'Восстановление из Google Drive, Яндекс.Диска и по QR-коду пока '
                    'не работает: ядро умеет собирать и раскладывать пакет на диск и на '
                    'WebDAV-сервер, но экрана бэкапа в интерфейсе нет, а поставленные '
                    'FFI-бинарники собраны раньше этих команд.',
                    textAlign: TextAlign.center,
                    style: TextStyle(color: Color(0xFF64748B), fontSize: 11, height: 1.4),
                  ),
                ] else ...[
                  _MnemonicCard(words: _mnemonic!.split(' '), onCopy: _copyMnemonic),
                  const SizedBox(height: 24),
                  ElevatedButton(
                    style: ElevatedButton.styleFrom(
                      backgroundColor: const Color(0xFF10B981),
                      foregroundColor: const Color(0xFF0F172A),
                      padding: const EdgeInsets.symmetric(vertical: 16),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                    ),
                    onPressed: () => Navigator.pop(context),
                    child: const Text('Я сохранил фразу • Войти в YggGram',
                        style: TextStyle(fontWeight: FontWeight.bold)),
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _MnemonicCard extends StatelessWidget {
  const _MnemonicCard({required this.words, required this.onCopy});

  final List<String> words;
  final VoidCallback onCopy;

  @override
  Widget build(BuildContext context) {
    return Container(
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
                'Ваша мнемоническая фраза:',
                style: TextStyle(
                    color: Color(0xFF10B981), fontWeight: FontWeight.bold, fontSize: 13),
              ),
              IconButton(
                icon: const Icon(Icons.copy, color: Color(0xFF94A3B8), size: 18),
                tooltip: 'Скопировать',
                onPressed: onCopy,
              ),
            ],
          ),
          const SizedBox(height: 12),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: List.generate(words.length, (index) {
              return Container(
                padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                decoration: BoxDecoration(
                  color: const Color(0xFF1E293B),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Text(
                  '${index + 1}. ${words[index]}',
                  style: const TextStyle(
                      color: Colors.white, fontSize: 12, fontFamily: 'monospace'),
                ),
              );
            }),
          ),
          const SizedBox(height: 14),
          const Text(
            'Запишите фразу на бумаге. Это единственный ключ к профилю и к зашифрованным бэкапам. '
            'Пока она лежит в приватном каталоге приложения файлом identity.seed — отдельным паролем '
            'не защищена. Фраза — настоящий BIP-39 (английский список из 2048 слов и контрольная '
            'сумма), поэтому её можно перенести в другой BIP-39-кошелёк и получить там тот же seed. '
            'Ключи YgGramm выводятся из seed нашим HKDF, а не по BIP-32, так что в чужом кошельке их '
            'не будет. Фразу, у которой контрольная сумма не сходится, ядро не примет.',
            style: TextStyle(color: Color(0xFFE2E8F0), fontSize: 11, height: 1.4),
          ),
        ],
      ),
    );
  }
}

class _RestoreSheet extends StatefulWidget {
  const _RestoreSheet({required this.onSubmit});

  final void Function(String phrase) onSubmit;

  @override
  State<_RestoreSheet> createState() => _RestoreSheetState();
}

class _RestoreSheetState extends State<_RestoreSheet> {
  final TextEditingController _controller = TextEditingController();
  String? _localError;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _submit() {
    final words = _controller.text.trim().split(RegExp(r'\s+'));
    if (words.length != 12 && words.length != 24) {
      setState(() => _localError = 'Нужно 12 или 24 слова, сейчас ${words.length}');
      return;
    }
    widget.onSubmit(words.join(' '));
  }

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: EdgeInsets.only(
        left: 24,
        right: 24,
        top: 24,
        bottom: MediaQuery.of(context).viewInsets.bottom + 24,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const Text('Восстановить из фразы',
              style: TextStyle(
                  color: Colors.white, fontWeight: FontWeight.bold, fontSize: 16)),
          const SizedBox(height: 8),
          const Text(
            'Ключи заново выведутся из фразы, и адрес IPv6 станет прежним.',
            style: TextStyle(color: Color(0xFF94A3B8), fontSize: 12),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _controller,
            maxLines: 3,
            style: const TextStyle(color: Colors.white, fontFamily: 'monospace', fontSize: 13),
            decoration: InputDecoration(
              hintText: 'absorb banner cabbage ...',
              hintStyle: const TextStyle(color: Color(0xFF475569)),
              filled: true,
              fillColor: const Color(0xFF0B1120),
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(12),
                borderSide: const BorderSide(color: Color(0xFF1E293B)),
              ),
            ),
          ),
          if (_localError != null) ...[
            const SizedBox(height: 10),
            Text(_localError!,
                style: const TextStyle(color: Color(0xFFF87171), fontSize: 12)),
          ],
          const SizedBox(height: 16),
          ElevatedButton(
            style: ElevatedButton.styleFrom(
              backgroundColor: const Color(0xFF10B981),
              foregroundColor: const Color(0xFF0F172A),
              padding: const EdgeInsets.symmetric(vertical: 14),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
            ),
            onPressed: _submit,
            child: const Text('Вывести ключи и запустить ноду',
                style: TextStyle(fontWeight: FontWeight.bold)),
          ),
        ],
      ),
    );
  }
}

class _ErrorBanner extends StatelessWidget {
  const _ErrorBanner({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: const Color(0xFF7F1D1D).withValues(alpha: 0.35),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: const Color(0xFFF87171).withValues(alpha: 0.5)),
      ),
      child: Row(
        children: [
          const Icon(Icons.error_outline, color: Color(0xFFF87171), size: 20),
          const SizedBox(width: 10),
          Expanded(
            child: Text(message,
                style: const TextStyle(color: Color(0xFFFCA5A5), fontSize: 12, height: 1.4)),
          ),
          TextButton(
            onPressed: onRetry,
            child: const Text('Заново', style: TextStyle(color: Color(0xFF10B981))),
          ),
        ],
      ),
    );
  }
}
