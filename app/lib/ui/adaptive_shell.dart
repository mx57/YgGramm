// File: app/lib/ui/adaptive_shell.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'screens/anti_dpi_radar_screen.dart';
import 'screens/backup_settings_screen.dart';
import 'screens/onboarding_screen.dart';
import 'state/app_state.dart';

class AdaptiveShell extends ConsumerStatefulWidget {
  const AdaptiveShell({super.key});

  @override
  ConsumerState<AdaptiveShell> createState() => _AdaptiveShellState();
}

class _AdaptiveShellState extends ConsumerState<AdaptiveShell> {
  int _selectedNavIndex = 0; // 0: Chats, 1: Anti-DPI Radar, 2: Cloud Backup, 3: Settings
  bool _showRightPanel = false;
  final TextEditingController _msgController = TextEditingController();

  final List<MessageItem> _demoMessages = [
    const MessageItem(
      id: 'm1',
      sender: 'Алиса',
      text: 'Привет! Сообщение доставлено через прямой Yggdrasil QUIC туннель.',
      time: '16:40',
      isMe: false,
    ),
    const MessageItem(
      id: 'm2',
      sender: 'Я',
      text: 'Отлично! ТСПУ заблокировал обычный интернет, но оверлей работает через uTLS Chrome.',
      time: '16:41',
      isMe: true,
    ),
    const MessageItem(
      id: 'm3',
      sender: 'Алиса',
      text: 'И чанки медиа скачались через P2P-рой без внешних серверов 👍',
      time: '16:42',
      isMe: false,
    ),
  ];

  @override
  void dispose() {
    _msgController.dispose();
    super.dispose();
  }

  void _sendMessage() {
    final text = _msgController.text.trim();
    if (text.isEmpty) return;

    setState(() {
      _demoMessages.add(MessageItem(
        id: 'm_${DateTime.now().millisecondsSinceEpoch}',
        sender: 'Я',
        text: text,
        time: '${DateTime.now().hour}:${DateTime.now().minute.toString().padLeft(2, '0')}',
        isMe: true,
      ));
    });
    _msgController.clear();
  }

  @override
  Widget build(BuildContext context) {
    final appState = ref.watch(appStateProvider);
    final width = MediaQuery.of(context).size.width;
    final isDesktop = width > 800;

    if (!isDesktop) {
      return _buildMobileLayout(context, appState);
    }

    return _buildDesktopLayout(context, appState);
  }

  // ==========================================
  // DESKTOP LAYOUT (3-COLUMN TELEGRAM STYLE)
  // ==========================================
  Widget _buildDesktopLayout(BuildContext context, AppState appState) {
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
                // App Logo
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
            child: _selectedNavIndex == 1
                ? const AntiDpiRadarScreen()
                : _selectedNavIndex == 2
                    ? const BackupSettingsScreen()
                    : _buildDesktopChatSplitView(appState),
          ),
        ],
      ),
    );
  }

  Widget _buildDesktopChatSplitView(AppState appState) {
    final selectedChat = appState.chats.firstWhere(
      (c) => c.id == appState.selectedChatId,
      orElse: () => appState.chats.first,
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
              // Search Header
              Padding(
                padding: const EdgeInsets.all(12),
                child: TextField(
                  style: const TextStyle(color: Colors.white, fontSize: 13),
                  decoration: InputDecoration(
                    hintText: 'Поиск контактов и сообщений...',
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

              // Chat List
              Expanded(
                child: ListView.builder(
                  itemCount: appState.chats.length,
                  itemBuilder: (context, index) {
                    final chat = appState.chats[index];
                    final isSelected = chat.id == appState.selectedChatId;

                    return Container(
                      color: isSelected ? const Color(0xFF1E293B) : Colors.transparent,
                      child: ListTile(
                        onTap: () {
                          ref.read(appStateProvider.notifier).selectChat(chat.id);
                        },
                        leading: CircleAvatar(
                          radius: 20,
                          backgroundColor: isSelected ? const Color(0xFF059669) : const Color(0xFF334155),
                          child: Text(
                            chat.avatarInitials,
                            style: const TextStyle(color: Colors.white, fontSize: 12, fontWeight: FontWeight.bold),
                          ),
                        ),
                        title: Row(
                          children: [
                            Expanded(
                              child: Text(
                                chat.title,
                                style: const TextStyle(color: Colors.white, fontWeight: FontWeight.w600, fontSize: 13),
                                overflow: TextOverflow.ellipsis,
                              ),
                            ),
                            Text(
                              chat.time,
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
                                  style: const TextStyle(color: Colors.black, fontSize: 10, fontWeight: FontWeight.bold),
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
          child: Column(
            children: [
              // Chat Header
              Container(
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
                      child: Text(
                        selectedChat.avatarInitials,
                        style: const TextStyle(color: Colors.white, fontSize: 12, fontWeight: FontWeight.bold),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        mainAxisAlignment: MainAxisAlignment.center,
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            selectedChat.title,
                            style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 14),
                          ),
                          Row(
                            children: [
                              Container(
                                width: 6,
                                height: 6,
                                decoration: const BoxDecoration(
                                  color: Color(0xFF10B981),
                                  shape: BoxShape.circle,
                                ),
                              ),
                              const SizedBox(width: 4),
                              const Text(
                                'End-to-End Encrypted (Double Ratchet) • IPv6: 200::...',
                                style: TextStyle(color: Color(0xFF94A3B8), fontSize: 10),
                              ),
                            ],
                          ),
                        ],
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.call, color: Color(0xFF10B981), size: 20),
                      tooltip: 'Прямой VoIP звонок (Opus FEC по IPv6)',
                      onPressed: () {
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(content: Text('Инициализация прямого звонка по оверлею 200::/7 без STUN...')),
                        );
                      },
                    ),
                    IconButton(
                      icon: const Icon(Icons.info_outline, color: Color(0xFF94A3B8), size: 20),
                      onPressed: () {
                        setState(() {
                          _showRightPanel = !_showRightPanel;
                        });
                      },
                    ),
                  ],
                ),
              ),

              // Messages List
              Expanded(
                child: Container(
                  color: const Color(0xFF0B1120),
                  child: ListView.builder(
                    padding: const EdgeInsets.all(16),
                    itemCount: _demoMessages.length,
                    itemBuilder: (context, index) {
                      final msg = _demoMessages[index];
                      return _buildMessageBubble(msg);
                    },
                  ),
                ),
              ),

              // Message Input Bar
              Container(
                padding: const EdgeInsets.all(12),
                decoration: const BoxDecoration(
                  color: Color(0xFF0F172A),
                  border: Border(top: BorderSide(color: Color(0xFF1E293B))),
                ),
                child: Row(
                  children: [
                    IconButton(
                      icon: const Icon(Icons.attach_file, color: Color(0xFF94A3B8)),
                      tooltip: 'Прикрепить тяжелый файл (P2P-рой 256КБ)',
                      onPressed: () {},
                    ),
                    Expanded(
                      child: TextField(
                        controller: _msgController,
                        style: const TextStyle(color: Colors.white, fontSize: 13),
                        onSubmitted: (_) => _sendMessage(),
                        decoration: InputDecoration(
                          hintText: 'Написать зашифрованное сообщение...',
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
                      icon: const Icon(Icons.mic, color: Color(0xFF94A3B8)),
                      tooltip: 'Голосовое сообщение (Opus)',
                      onPressed: () {},
                    ),
                    IconButton(
                      icon: const Icon(Icons.send, color: Color(0xFF10B981)),
                      onPressed: _sendMessage,
                    ),
                  ],
                ),
              ),
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
                const SizedBox(height: 16),
                Center(
                  child: CircleAvatar(
                    radius: 36,
                    backgroundColor: const Color(0xFF059669),
                    child: Text(selectedChat.avatarInitials, style: const TextStyle(color: Colors.white, fontSize: 20)),
                  ),
                ),
                const SizedBox(height: 12),
                Center(
                  child: Text(
                    selectedChat.title,
                    style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 16),
                  ),
                ),
                const SizedBox(height: 24),
                const Text('Криптография:', style: TextStyle(color: Color(0xFF64748B), fontSize: 11)),
                const SizedBox(height: 4),
                const Text('X3DH + Double Ratchet v1', style: TextStyle(color: Colors.white, fontSize: 12)),
                const SizedBox(height: 12),
                const Text('Слепой релей:', style: TextStyle(color: Color(0xFF64748B), fontSize: 11)),
                const SizedBox(height: 4),
                const Text('3 S&F узла (PoW активен)', style: TextStyle(color: Color(0xFF10B981), fontSize: 12)),
              ],
            ),
          ),
      ],
    );
  }

  // ==========================================
  // MOBILE LAYOUT (ANDROID COMPACT GESTURES)
  // ==========================================
  Widget _buildMobileLayout(BuildContext context, AppState appState) {
    Widget content;
    switch (_selectedNavIndex) {
      case 1:
        content = const AntiDpiRadarScreen();
        break;
      case 2:
        content = const BackupSettingsScreen();
        break;
      default:
        content = ListView.builder(
          itemCount: appState.chats.length,
          itemBuilder: (context, index) {
            final chat = appState.chats[index];
            return Dismissible(
              key: Key(chat.id),
              direction: DismissDirection.startToEnd,
              background: Container(
                color: const Color(0xFF059669),
                alignment: Alignment.centerLeft,
                padding: const EdgeInsets.symmetric(horizontal: 20),
                child: const Icon(Icons.reply, color: Colors.white),
              ),
              child: ListTile(
                leading: CircleAvatar(
                  backgroundColor: const Color(0xFF334155),
                  child: Text(chat.avatarInitials, style: const TextStyle(color: Colors.white)),
                ),
                title: Text(chat.title, style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold)),
                subtitle: Text(chat.lastMessage, style: const TextStyle(color: Color(0xFF94A3B8))),
                trailing: Text(chat.time, style: const TextStyle(color: Color(0xFF64748B), fontSize: 11)),
                onTap: () {
                  // Open full chat screen
                },
              ),
            );
          },
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

  Widget _buildMessageBubble(MessageItem msg) {
    return Align(
      alignment: msg.isMe ? Alignment.centerRight : Alignment.centerLeft,
      child: Container(
        margin: const EdgeInsets.only(bottom: 8),
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        constraints: const BoxConstraints(maxWidth: 480),
        decoration: BoxDecoration(
          color: msg.isMe ? const Color(0xFF065F46) : const Color(0xFF1E293B),
          borderRadius: BorderRadius.circular(16).copyWith(
            bottomRight: msg.isMe ? const Radius.circular(2) : const Radius.circular(16),
            bottomLeft: !msg.isMe ? const Radius.circular(2) : const Radius.circular(16),
          ),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              msg.text,
              style: const TextStyle(color: Colors.white, fontSize: 13),
            ),
            const SizedBox(height: 4),
            Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text(
                  msg.time,
                  style: const TextStyle(color: Color(0xFF94A3B8), fontSize: 10),
                ),
                if (msg.isMe) ...[
                  const SizedBox(width: 4),
                  const Icon(Icons.done_all, color: Color(0xFF10B981), size: 12),
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
