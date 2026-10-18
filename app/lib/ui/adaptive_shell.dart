// File: app/lib/ui/adaptive_shell.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../core_bridge/ygg_bridge.dart';
import 'state/core_state.dart';
import 'screens/anti_dpi_radar_screen.dart';
import 'screens/backup_settings_screen.dart';
import 'screens/onboarding_screen.dart';

class AdaptiveShell extends ConsumerStatefulWidget {
  const AdaptiveShell({super.key});

  @override
  ConsumerState<AdaptiveShell> createState() => _AdaptiveShellState();
}

class _AdaptiveShellState extends ConsumerState<AdaptiveShell> {
  int _selectedNavIndex = 0; // 0: Чаты, 1: Anti-DPI Radar, 2: Бэкап
  bool _showRightPanel = false;
  final TextEditingController _msgController = TextEditingController();

  @override
  void initState() {
    super.initState();
    // Ядро поднимается сразу: UI показывает реальное состояние, а не макет.
    Future.microtask(() => ref.read(coreProvider.notifier).bootstrap());
  }

  @override
  void dispose() {
    _msgController.dispose();
    super.dispose();
  }

  Future<void> _sendMessage(CoreSnapshot snapshot) async {
    final controller = ref.read(coreProvider.notifier);
    final warning = controller.sendText(_msgController.text);
    _msgController.clear();
    await controller.refresh();
    if (!mounted) return;
    if (warning != null) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(warning)));
    }
  }

  /// Диалог «добавить контакт по ключу узла»: ядро само идёт за PreKeyBundle,
  /// поэтому здесь спрашивается только ed25519-ключ собеседника и локальное имя.
  Future<void> _addContact() async {
    final snapshot = ref.read(coreProvider);
    if (snapshot.phase != CorePhase.online) {
      ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
          content: Text('Ядро не в сети: сначала поднимите ноду в крипто-профиле')));
      return;
    }
    final controller = ref.read(coreProvider.notifier);
    await showDialog<void>(
      context: context,
      builder: (dialogContext) => AddContactDialog(
        nodeKeyHex: snapshot.nodeKeyHex ?? '',
        onSubmit: (key, nickname) => controller.fetchContact(key, nickname: nickname),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final snapshot = ref.watch(coreProvider);
    final width = MediaQuery.of(context).size.width;
    final isDesktop = width > 800;

    if (!isDesktop) {
      return _buildMobileLayout(context, snapshot);
    }
    return _buildDesktopLayout(context, snapshot);
  }

  // ==========================================
  // DESKTOP LAYOUT (3-COLUMN TELEGRAM STYLE)
  // ==========================================
  Widget _buildDesktopLayout(BuildContext context, CoreSnapshot snapshot) {
    return Scaffold(
      backgroundColor: const Color(0xFF0F172A),
      body: Row(
        children: [
          // 1. Navigation Rail (Leftmost)
          Container(
            width: 72,
            decoration: const BoxDecoration(
              color: Color(0xFF0B1120),
              border: Border(right: BorderSide(color: Color(0xFF1E293B))),
            ),
            child: Column(
              children: [
                const SizedBox(height: 16),
                Container(
                  width: 44,
                  height: 44,
                  decoration: BoxDecoration(
                    gradient: const LinearGradient(
                      colors: [Color(0xFF059669), Color(0xFF06B6D4)],
                      begin: Alignment.topLeft,
                      end: Alignment.bottomRight,
                    ),
                    borderRadius: BorderRadius.circular(14),
                  ),
                  child: const Icon(Icons.shield, color: Colors.white, size: 24),
                ),
                const SizedBox(height: 24),

                _navIcon(0, Icons.chat_bubble_outline, Icons.chat_bubble, 'Чаты'),
                _navIcon(1, Icons.radar_outlined, Icons.radar, 'Anti-DPI'),
                _navIcon(2, Icons.cloud_sync_outlined, Icons.cloud_sync, 'Бэкап'),
                const Spacer(),
                _statusDot(snapshot),
                const SizedBox(height: 12),
                IconButton(
                  icon: const Icon(Icons.vpn_key_outlined, color: Color(0xFF94A3B8)),
                  tooltip: 'Крипто-профиль / Мнемоника',
                  onPressed: () {
                    Navigator.push(
                      context,
                      MaterialPageRoute(builder: (_) => const OnboardingScreen()),
                    );
                  },
                ),
                const SizedBox(height: 16),
              ],
            ),
          ),

          // 2. Main Area depending on Selected Tab
          Expanded(
            child: Column(
              children: [
                CorePhaseBanner(snapshot: snapshot),
                Expanded(
                  child: _selectedNavIndex == 1
                      ? const AntiDpiRadarScreen()
                      : _selectedNavIndex == 2
                          ? const BackupSettingsScreen()
                          : _buildDesktopChatSplitView(snapshot),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildDesktopChatSplitView(CoreSnapshot snapshot) {
    final conversations = snapshot.conversations;
    final selected = conversations.firstWhere(
      (c) => c.conversationId == snapshot.selectedConversationId,
      orElse: () => ConversationInfo(),
    );

    return Row(
      children: [
        // Dialog List (Width: 320px)
        Container(
          width: 320,
          decoration: const BoxDecoration(
            color: Color(0xFF0F172A),
            border: Border(right: BorderSide(color: Color(0xFF1E293B))),
          ),
          child: Column(
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(12, 12, 4, 0),
                child: Row(
                  children: [
                    Expanded(
                      child: TextField(
                        style: const TextStyle(color: Colors.white, fontSize: 13),
                        decoration: InputDecoration(
                          hintText: 'Поиск по реальной базе...',
                          hintStyle: const TextStyle(color: Color(0xFF64748B), fontSize: 13),
                          prefixIcon: const Icon(Icons.search, color: Color(0xFF64748B), size: 18),
                          filled: true,
                          fillColor: const Color(0xFF1E293B),
                          contentPadding: EdgeInsets.zero,
                          border: OutlineInputBorder(
                            borderRadius: BorderRadius.circular(10),
                            borderSide: BorderSide.none,
                          ),
                        ),
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.person_add_alt_outlined,
                          color: Color(0xFF10B981), size: 20),
                      tooltip: 'Добавить контакт по ключу узла',
                      onPressed: _addContact,
                    ),
                  ],
                ),
              ),
              ContactNotes(snapshot: snapshot),

              Expanded(
                child: conversations.isEmpty
                    ? const _EmptyState(
                        icon: Icons.forum_outlined,
                        title: 'Диалогов пока нет',
                        hint: 'Список строится из зашифрованной базы ядра: '
                            'сообщения появятся, когда дойдут до получателя.',
                      )
                    : ListView.builder(
                        itemCount: conversations.length,
                        itemBuilder: (context, index) {
                          final chat = conversations[index];
                          final isSelected =
                              chat.conversationId == snapshot.selectedConversationId;
                          return Container(
                            color: isSelected ? const Color(0xFF1E293B) : Colors.transparent,
                            child: ListTile(
                              onTap: () {
                                ref
                                    .read(coreProvider.notifier)
                                    .selectConversation(chat.conversationId);
                              },
                              leading: CircleAvatar(
                                radius: 20,
                                backgroundColor: isSelected
                                    ? const Color(0xFF059669)
                                    : const Color(0xFF334155),
                                child: Text(
                                  _initials(chat.conversationId),
                                  style: const TextStyle(
                                      color: Colors.white,
                                      fontSize: 11,
                                      fontWeight: FontWeight.bold),
                                ),
                              ),
                              title: Row(
                                children: [
                                  Expanded(
                                    child: Text(
                                      chat.conversationId,
                                      style: const TextStyle(
                                          color: Colors.white,
                                          fontWeight: FontWeight.w600,
                                          fontSize: 13),
                                      overflow: TextOverflow.ellipsis,
                                    ),
                                  ),
                                  Text(
                                    _formatTime(chat.lastTimestampMs.toInt()),
                                    style: const TextStyle(color: Color(0xFF64748B), fontSize: 11),
                                  ),
                                ],
                              ),
                              subtitle: Row(
                                children: [
                                  Expanded(
                                    child: Text(
                                      chat.lastMessage,
                                      style: const TextStyle(color: Color(0xFF94A3B8), fontSize: 12),
                                      maxLines: 1,
                                      overflow: TextOverflow.ellipsis,
                                    ),
                                  ),
                                  if (chat.unreadCount > 0)
                                    Container(
                                      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                                      decoration: BoxDecoration(
                                        color: const Color(0xFF10B981),
                                        borderRadius: BorderRadius.circular(10),
                                      ),
                                      child: Text(
                                        '${chat.unreadCount}',
                                        style: const TextStyle(
                                            color: Colors.black,
                                            fontSize: 10,
                                            fontWeight: FontWeight.bold),
                                      ),
                                    ),
                                ],
                              ),
                            ),
                          );
                        },
                      ),
              ),
            ],
          ),
        ),

        // Active Chat Area
        Expanded(
          child: selected.conversationId.isEmpty
              ? const _EmptyState(
                  icon: Icons.lock_outline,
                  title: 'Диалог не выбран',
                  hint: 'Откройте диалог слева или добавьте контакт с PreKeyBundle.',
                )
              : Column(
                  children: [
                    _chatHeader(selected.conversationId, snapshot),
                    Expanded(
                      child: Container(
                        color: const Color(0xFF0B1120),
                        child: snapshot.messages.isEmpty
                            ? const _EmptyState(
                                icon: Icons.mail_outline_outlined,
                                title: 'Сообщений нет',
                                hint: 'Пусто — потому что в зашифрованной базе ядра ещё ничего нет.',
                              )
                            : ListView.builder(
                                padding: const EdgeInsets.all(16),
                                itemCount: snapshot.messages.length,
                                itemBuilder: (context, index) {
                                  // Ядро отдаёт сообщения от новых к старым.
                                  final msg = snapshot.messages[snapshot.messages.length - 1 - index];
                                  return _buildMessageBubble(msg);
                                },
                              ),
                      ),
                    ),
                    _inputBar(snapshot),
                  ],
                ),
        ),

        // Right Info Panel (Optional)
        if (_showRightPanel)
          Container(
            width: 260,
            decoration: const BoxDecoration(
              color: Color(0xFF0F172A),
              border: Border(left: BorderSide(color: Color(0xFF1E293B))),
            ),
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text(
                  'Информация о чате',
                  style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 14),
                ),
                const SizedBox(height: 24),
                const Text('Наш IPv6 (0200::/7):', style: TextStyle(color: Color(0xFF64748B), fontSize: 11)),
                const SizedBox(height: 4),
                Text(
                  snapshot.ipv6 ?? 'нода не запущена',
                  style: const TextStyle(color: Colors.white, fontSize: 12, fontFamily: 'monospace'),
                ),
                const SizedBox(height: 12),
                const Text('Пиров у ноды:', style: TextStyle(color: Color(0xFF64748B), fontSize: 11)),
                const SizedBox(height: 4),
                Text('${snapshot.peers.length}',
                    style: const TextStyle(color: Colors.white, fontSize: 12)),
                const SizedBox(height: 12),
                const Text('Контактов с бандлом:',
                    style: TextStyle(color: Color(0xFF64748B), fontSize: 11)),
                const SizedBox(height: 4),
                Text('${snapshot.contacts.length}',
                    style: const TextStyle(color: Colors.white, fontSize: 12)),
              ],
            ),
          ),
      ],
    );
  }

  Widget _chatHeader(String conversationId, CoreSnapshot snapshot) {
    return Container(
      height: 58,
      padding: const EdgeInsets.symmetric(horizontal: 16),
      decoration: const BoxDecoration(
        color: Color(0xFF0F172A),
        border: Border(bottom: BorderSide(color: Color(0xFF1E293B))),
      ),
      child: Row(
        children: [
          CircleAvatar(
            radius: 18,
            backgroundColor: const Color(0xFF059669),
            child: Text(_initials(conversationId),
                style: const TextStyle(color: Colors.white, fontSize: 11, fontWeight: FontWeight.bold)),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(conversationId,
                    style: const TextStyle(
                        color: Colors.white, fontWeight: FontWeight.bold, fontSize: 14)),
                Row(
                  children: [
                    Container(
                      width: 6,
                      height: 6,
                      decoration: BoxDecoration(
                        color: snapshot.phase == CorePhase.online
                            ? const Color(0xFF10B981)
                            : const Color(0xFF64748B),
                        shape: BoxShape.circle,
                      ),
                    ),
                    const SizedBox(width: 4),
                    Expanded(
                      child: Text(
                        snapshot.ipv6 == null
                            ? 'X3DH + Double Ratchet • нода не запущена'
                            : 'X3DH + Double Ratchet • IPv6: ${snapshot.ipv6}',
                        style: const TextStyle(color: Color(0xFF94A3B8), fontSize: 10),
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                  ],
                ),
              ],
            ),
          ),
          IconButton(
            icon: const Icon(Icons.info_outline, color: Color(0xFF94A3B8), size: 20),
            onPressed: () => setState(() => _showRightPanel = !_showRightPanel),
          ),
        ],
      ),
    );
  }

  Widget _inputBar(CoreSnapshot snapshot) {
    final disabled = !snapshot.canSend;
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: const BoxDecoration(
        color: Color(0xFF0F172A),
        border: Border(top: BorderSide(color: Color(0xFF1E293B))),
      ),
      child: Row(
        children: [
          Expanded(
            child: TextField(
              controller: _msgController,
              onSubmitted: (_) => _sendMessage(snapshot),
              decoration: InputDecoration(
                hintText: disabled
                    ? 'Нужен контакт с PreKeyBundle — тогда сообщение можно зашифровать'
                    : 'Написать зашифрованное сообщение...',
                hintStyle: const TextStyle(color: Color(0xFF64748B), fontSize: 13),
                filled: true,
                fillColor: const Color(0xFF1E293B),
                contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(24),
                  borderSide: BorderSide.none,
                ),
              ),
            ),
          ),
          const SizedBox(width: 8),
          IconButton(
            icon: const Icon(Icons.send, color: Color(0xFF10B981)),
            tooltip: disabled ? 'Нет контакта с PreKeyBundle' : 'Зашифровать и отправить',
            onPressed: disabled ? null : () => _sendMessage(snapshot),
          ),
        ],
      ),
    );
  }

  // ==========================================
  // MOBILE LAYOUT (ANDROID COMPACT GESTURES)
  // ==========================================
  Widget _buildMobileLayout(BuildContext context, CoreSnapshot snapshot) {
    Widget content;
    switch (_selectedNavIndex) {
      case 1:
        content = const AntiDpiRadarScreen();
        break;
      case 2:
        content = const BackupSettingsScreen();
        break;
      default:
        content = Column(
          children: [
            CorePhaseBanner(snapshot: snapshot),
            ContactNotes(snapshot: snapshot),
            Expanded(
              child: snapshot.conversations.isEmpty
                  ? const _EmptyState(
                      icon: Icons.forum_outlined,
                      title: 'Диалогов пока нет',
                      hint: 'Данные читаются из зашифрованной базы ядра.',
                    )
                  : ListView.builder(
                      itemCount: snapshot.conversations.length,
                      itemBuilder: (context, index) {
                        final chat = snapshot.conversations[index];
                        return ListTile(
                          leading: CircleAvatar(
                            backgroundColor: const Color(0xFF334155),
                            child: Text(_initials(chat.conversationId),
                                style: const TextStyle(color: Colors.white, fontSize: 11)),
                          ),
                          title: Text(chat.conversationId,
                              style: const TextStyle(
                                  color: Colors.white, fontWeight: FontWeight.bold)),
                          subtitle: Text(chat.lastMessage,
                              style: const TextStyle(color: Color(0xFF94A3B8))),
                          trailing: Text(_formatTime(chat.lastTimestampMs.toInt()),
                              style: const TextStyle(color: Color(0xFF64748B), fontSize: 11)),
                          onTap: () {
                            ref
                                .read(coreProvider.notifier)
                                .selectConversation(chat.conversationId);
                            Navigator.push(
                              context,
                              MaterialPageRoute(builder: (_) => const ConversationScreen()),
                            );
                          },
                        );
                      },
                    ),
            ),
          ],
        );
    }

    return Scaffold(
      backgroundColor: const Color(0xFF0F172A),
      appBar: AppBar(
        backgroundColor: const Color(0xFF0B1120),
        title: Text(
          _selectedNavIndex == 1
              ? 'Anti-DPI Радар'
              : _selectedNavIndex == 2
                  ? 'Zero-Knowledge Бэкап'
                  : 'YggGram',
          style: const TextStyle(color: Colors.white, fontSize: 18),
        ),
        actions: [
          _statusDot(snapshot),
          IconButton(
            icon: const Icon(Icons.person_add_alt_outlined, color: Color(0xFF10B981)),
            tooltip: 'Добавить контакт по ключу узла',
            onPressed: _addContact,
          ),
          IconButton(
            icon: const Icon(Icons.vpn_key_outlined, color: Colors.white),
            onPressed: () {
              Navigator.push(context, MaterialPageRoute(builder: (_) => const OnboardingScreen()));
            },
          ),
        ],
      ),
      body: content,
      bottomNavigationBar: BottomNavigationBar(
        backgroundColor: const Color(0xFF0B1120),
        selectedItemColor: const Color(0xFF10B981),
        unselectedItemColor: const Color(0xFF64748B),
        currentIndex: _selectedNavIndex,
        onTap: (idx) => setState(() => _selectedNavIndex = idx),
        items: const [
          BottomNavigationBarItem(icon: Icon(Icons.chat_bubble_outline), label: 'Чаты'),
          BottomNavigationBarItem(icon: Icon(Icons.radar), label: 'Anti-DPI'),
          BottomNavigationBarItem(icon: Icon(Icons.cloud_sync), label: 'Бэкап'),
        ],
      ),
    );
  }

  Widget _statusDot(CoreSnapshot snapshot) {
    final online = snapshot.phase == CorePhase.online;
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12),
      child: Tooltip(
        message: snapshot.ipv6 ?? snapshot.error ?? snapshot.phase.name,
        child: Container(
          width: 10,
          height: 10,
          decoration: BoxDecoration(
            color: online ? const Color(0xFF10B981) : const Color(0xFF64748B),
            shape: BoxShape.circle,
          ),
        ),
      ),
    );
  }

  Widget _buildMessageBubble(ChatMessage msg) {
    final isMe = msg.isOutbound;
    final body = msg.whichContent() == ChatMessage_Content.text
        ? msg.text.body
        : '[${msg.whichContent().name}]';

    return Align(
      alignment: isMe ? Alignment.centerRight : Alignment.centerLeft,
      child: Container(
        margin: const EdgeInsets.only(bottom: 8),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        constraints: const BoxConstraints(maxWidth: 480),
        decoration: BoxDecoration(
          color: isMe ? const Color(0xFF065F46) : const Color(0xFF1E293B),
          borderRadius: BorderRadius.circular(16).copyWith(
            bottomRight: isMe ? const Radius.circular(2) : const Radius.circular(16),
            bottomLeft: !isMe ? const Radius.circular(2) : const Radius.circular(16),
          ),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(body, style: const TextStyle(color: Colors.white, fontSize: 13)),
            const SizedBox(height: 4),
            Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(_formatTime(msg.timestampMs.toInt()),
                    style: const TextStyle(color: Color(0xFF94A3B8), fontSize: 10)),
                if (isMe) ...[
                  const SizedBox(width: 4),
                  // Одинокая галочка — пакет ушёл нам в оверлей, двойная —
                  // получатель расшифровал сообщение и прислал ACK. Вторую
                  // нельзя рисовать «по умолчанию»: её даёт только
                  // DeliveryAckContent.
                  Icon(
                    msg.delivered ? Icons.done_all : Icons.done,
                    color: msg.delivered
                        ? const Color(0xFF10B981)
                        : const Color(0xFF94A3B8),
                    size: 12,
                  ),
                ],
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _navIcon(int index, IconData outline, IconData solid, String tooltip) {
    final isSelected = _selectedNavIndex == index;
    return IconButton(
      icon: Icon(
        isSelected ? solid : outline,
        color: isSelected ? const Color(0xFF10B981) : const Color(0xFF64748B),
      ),
      tooltip: tooltip,
      onPressed: () => setState(() => _selectedNavIndex = index),
    );
  }
}

/// Баннер честного состояния ядра: без идентичности, в процессе старта или
/// с реальной ошибкой от C-ABI.
class CorePhaseBanner extends ConsumerWidget {
  const CorePhaseBanner({super.key, required this.snapshot});

  final CoreSnapshot snapshot;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final (color, text) = switch (snapshot.phase) {
      CorePhase.idle => (const Color(0xFF1E293B), 'Ядро ещё не запускалось'),
      CorePhase.needsIdentity => (
          const Color(0xFF065F46),
          'Нужна крипто-идентичность: сгенерируйте фразу или восстановите её'
        ),
      CorePhase.starting => (const Color(0xFF1E293B), 'Поднимаем userspace-ноду...'),
      CorePhase.online => (
          const Color(0xFF065F46),
          'Онлайн • ${snapshot.ipv6 ?? ''} • пиров: ${snapshot.peers.length}'
        ),
      CorePhase.failed => (const Color(0xFF7F1D1D), snapshot.error ?? 'Ядро сообщило об ошибке'),
    };

    return Material(
      color: color,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        child: Row(
          children: [
            Expanded(
              child: Text(text,
                  style: const TextStyle(color: Colors.white, fontSize: 12),
                  overflow: TextOverflow.ellipsis),
            ),
            if (snapshot.phase == CorePhase.needsIdentity ||
                snapshot.phase == CorePhase.failed)
              TextButton(
                onPressed: () {
                  Navigator.push(
                    context,
                    MaterialPageRoute(builder: (_) => const OnboardingScreen()),
                  );
                },
                child: const Text('К крипто-профилю',
                    style: TextStyle(color: Colors.white, fontSize: 12)),
              ),
            if (snapshot.phase == CorePhase.failed)
              TextButton(
                onPressed: () => ref.read(coreProvider.notifier).bootstrap(),
                child: const Text('Повторить',
                    style: TextStyle(color: Colors.white, fontSize: 12)),
              ),
          ],
        ),
      ),
    );
  }
}

/// Чем закончилась сетевая загрузка PreKeyBundle: незакрытые запросы и последняя
/// новость. Молча гасить индикатор нельзя — «узел не ответил» и «бандл не прошёл
/// проверку» для пользователя значат разное.
class ContactNotes extends StatelessWidget {
  const ContactNotes({super.key, required this.snapshot});

  final CoreSnapshot snapshot;

  @override
  Widget build(BuildContext context) {
    if (snapshot.pendingContacts.isEmpty && snapshot.contactNote == null) {
      return const SizedBox.shrink();
    }
    final waiting = snapshot.pendingContacts.isNotEmpty;
    return Container(
      margin: const EdgeInsets.fromLTRB(12, 8, 12, 0),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: BoxDecoration(
        color: const Color(0xFF1E293B),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Row(
        children: [
          waiting
              ? const SizedBox(
                  width: 14,
                  height: 14,
                  child: CircularProgressIndicator(
                      strokeWidth: 2, color: Color(0xFF10B981)))
              : const Icon(Icons.mark_chat_read_outlined,
                  size: 16, color: Color(0xFF10B981)),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(snapshot.contactNote ?? '',
                    style: const TextStyle(color: Colors.white, fontSize: 12),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis),
                if (waiting)
                  Text('В ожидании ответа: ${snapshot.pendingContacts.length}',
                      style: const TextStyle(color: Color(0xFF64748B), fontSize: 11)),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

/// Диалог «добавить контакт по ключу узла». Имя локальное: бандл его не несёт,
/// nickname выбирает тот, кто добавляет.
class AddContactDialog extends StatefulWidget {
  const AddContactDialog({
    super.key,
    required this.nodeKeyHex,
    required this.onSubmit,
  });

  /// Наш ключ узла — его собеседник передаёт вне сети, им и пользуются, чтобы
  /// добавить нас.
  final String nodeKeyHex;

  /// Возвращает текст проблемы или null, если bundle_req ушёл в оверлей.
  final String? Function(String nodeKeyHex, String nickname) onSubmit;

  @override
  State<AddContactDialog> createState() => _AddContactDialogState();
}

class _AddContactDialogState extends State<AddContactDialog> {
  final TextEditingController _key = TextEditingController();
  final TextEditingController _nickname = TextEditingController();
  String? _problem;

  @override
  void dispose() {
    _key.dispose();
    _nickname.dispose();
    super.dispose();
  }

  void _submit() {
    final problem = widget.onSubmit(_key.text, _nickname.text);
    if (problem != null) {
      setState(() => _problem = problem);
      return;
    }
    Navigator.of(context).pop();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      backgroundColor: const Color(0xFF0F172A),
      title: const Text('Контакт по ключу узла',
          style: TextStyle(color: Colors.white, fontSize: 16)),
      content: SizedBox(
        width: 420,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text(
              'Ядро попросит у этого узла его PreKeyBundle и заведёт контакт, '
              'если ответ придёт от того же ключа и пройдёт проверку связки ключей.',
              style: TextStyle(color: Color(0xFF94A3B8), fontSize: 12),
            ),
            const SizedBox(height: 14),
            TextField(
              controller: _key,
              style: const TextStyle(color: Colors.white, fontSize: 12),
              decoration: const InputDecoration(
                labelText: 'Ed25519-ключ узла (64 hex-символа)',
                labelStyle: TextStyle(color: Color(0xFF64748B), fontSize: 12),
                hintText: 'напр. 1bf2…',
                hintStyle: TextStyle(color: Color(0xFF334155), fontSize: 11),
              ),
            ),
            const SizedBox(height: 10),
            TextField(
              controller: _nickname,
              style: const TextStyle(color: Colors.white, fontSize: 13),
              onSubmitted: (_) => _submit(),
              decoration: const InputDecoration(
                labelText: 'Имя (локальное, для вас)',
                labelStyle: TextStyle(color: Color(0xFF64748B), fontSize: 12),
              ),
            ),
            const SizedBox(height: 12),
            const Text('Ваш ключ узла — передайте его собеседнику:',
                style: TextStyle(color: Color(0xFF64748B), fontSize: 11)),
            SelectableText(
              widget.nodeKeyHex.isEmpty ? 'нода не запущена' : widget.nodeKeyHex,
              style: const TextStyle(
                  color: Colors.white, fontSize: 11, fontFamily: 'monospace'),
            ),
            if (_problem != null) ...[
              const SizedBox(height: 10),
              Text(_problem!,
                  style: const TextStyle(color: Color(0xFFFCA5A5), fontSize: 12)),
            ],
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Отмена', style: TextStyle(color: Color(0xFF94A3B8))),
        ),
        FilledButton(
          onPressed: _submit,
          child: const Text('Запросить бандл'),
        ),
      ],
    );
  }
}

class ConversationScreen extends ConsumerWidget {
  const ConversationScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final snapshot = ref.watch(coreProvider);
    return Scaffold(
      backgroundColor: const Color(0xFF0F172A),
      appBar: AppBar(
        backgroundColor: const Color(0xFF0B1120),
        title: Text(snapshot.selectedConversationId ?? 'Диалог',
            style: const TextStyle(color: Colors.white, fontSize: 15)),
      ),
      body: snapshot.messages.isEmpty
          ? const Center(
              child: Text('Сообщений нет',
                  style: TextStyle(color: Color(0xFF94A3B8), fontSize: 13)),
            )
          : ListView.builder(
              padding: const EdgeInsets.all(12),
              itemCount: snapshot.messages.length,
              itemBuilder: (context, index) {
                final msg = snapshot.messages[snapshot.messages.length - 1 - index];
                return ListTile(
                  title: Text(
                    msg.whichContent() == ChatMessage_Content.text ? msg.text.body : '',
                    style: const TextStyle(color: Colors.white, fontSize: 13),
                  ),
                  subtitle: Text(
                    '${msg.senderNodeIpv6} • ${_formatTime(msg.timestampMs.toInt())}',
                    style: const TextStyle(color: Color(0xFF64748B), fontSize: 11),
                  ),
                );
              },
            ),
    );
  }
}

class _EmptyState extends StatelessWidget {
  const _EmptyState({required this.icon, required this.title, required this.hint});

  final IconData icon;
  final String title;
  final String hint;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(icon, color: const Color(0xFF334155), size: 40),
            const SizedBox(height: 12),
            Text(title,
                style: const TextStyle(
                    color: Colors.white, fontWeight: FontWeight.bold, fontSize: 14)),
            const SizedBox(height: 6),
            Text(hint,
                textAlign: TextAlign.center,
                style: const TextStyle(color: Color(0xFF64748B), fontSize: 12)),
          ],
        ),
      ),
    );
  }
}

String _initials(String id) {
  final cleaned = id.replaceAll(RegExp('[^A-Za-z0-9А-Яа-я]'), '');
  return cleaned.isEmpty
      ? '?'
      : cleaned.substring(0, cleaned.length >= 2 ? 2 : 1).toUpperCase();
}

String _formatTime(int ms) {
  if (ms <= 0) return '';
  final local = DateTime.fromMillisecondsSinceEpoch(ms);
  return '${local.hour.toString().padLeft(2, '0')}:'
      '${local.minute.toString().padLeft(2, '0')}';
}
