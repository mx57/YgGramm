// File: app/lib/ui/state/app_state.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';

// Chat model
class ChatItem {
  final String id;
  final String title;
  final String lastMessage;
  final String time;
  final int unreadCount;
  final String avatarInitials;
  final bool isChannel;
  final bool isOnline;

  const ChatItem({
    required this.id,
    required this.title,
    required this.lastMessage,
    required this.time,
    this.unreadCount = 0,
    required this.avatarInitials,
    this.isChannel = false,
    this.isOnline = true,
  });
}

// Message model
class MessageItem {
  final String id;
  final String sender;
  final String text;
  final String time;
  final bool isMe;
  final bool isDelivered;
  final bool isRead;
  final String? replyTo;

  const MessageItem({
    required this.id,
    required this.sender,
    required this.text,
    required this.time,
    required this.isMe,
    this.isDelivered = true,
    this.isRead = true,
    this.replyTo,
  });
}

// Peer model
class PeerStatus {
  final String uri;
  final String ipv6;
  final int pingMs;
  final String transport; // "uTLS Chrome", "WSS CDN", "ShadowTLS", "LAN Multicast"
  final String flag;
  final bool isConnected;

  const PeerStatus({
    required this.uri,
    required this.ipv6,
    required this.pingMs,
    required this.transport,
    required this.flag,
    this.isConnected = true,
  });
}

// App Settings & State
class AppState {
  final String myIPv6;
  final bool isAntiDpiEnabled;
  final bool isProxyGDriveThroughYgg;
  final bool isMirroringEnabled;
  final bool isGDriveConnected;
  final bool isYandexConnected;
  final int mediaCacheLimitMB;
  final List<ChatItem> chats;
  final List<PeerStatus> peers;
  final String? selectedChatId;

  const AppState({
    this.myIPv6 = '0200:4f2a:819b:90c1:da24:5512:eed8:102f',
    this.isAntiDpiEnabled = true,
    this.isProxyGDriveThroughYgg = true,
    this.isMirroringEnabled = true,
    this.isGDriveConnected = true,
    this.isYandexConnected = true,
    this.mediaCacheLimitMB = 2048,
    this.chats = const [
      ChatItem(
        id: '1',
        title: 'Алиса',
        lastMessage: 'Зашифрованный файл получен через P2P рой!',
        time: '16:42',
        avatarInitials: 'АЛ',
        isOnline: true,
      ),
      ChatItem(
        id: '2',
        title: 'DevOps & Mesh',
        lastMessage: 'Новая нода в Москве поднята на порту 54321',
        time: '16:35',
        unreadCount: 3,
        avatarInitials: 'DM',
        isChannel: true,
      ),
      ChatItem(
        id: '3',
        title: 'YggGram Новости',
        lastMessage: 'Обновление v1.0: Полный обход ТСПУ через uTLS',
        time: '15:10',
        avatarInitials: 'YG',
        isChannel: true,
      ),
      ChatItem(
        id: '4',
        title: 'Борис',
        lastMessage: 'Звонок по IPv6 без STUN прошел отлично',
        time: 'Вчера',
        avatarInitials: 'БО',
        isOnline: false,
      ),
    ],
    this.peers = const [
      PeerStatus(
        uri: 'tls://msk-node.yggdrasil.network:54321',
        ipv6: '0200:1a4b:9812:ffaa::1',
        pingMs: 14,
        transport: 'uTLS Chrome',
        flag: '🇷🇺 MSK',
      ),
      PeerStatus(
        uri: 'wss://cdn-spb.ygggram.org/ws',
        ipv6: '0200:44bc:7721:e10a::4',
        pingMs: 22,
        transport: 'WSS CDN',
        flag: '🇷🇺 SPB',
      ),
      PeerStatus(
        uri: 'shadowtls://fin-bridge.ygggram.org:8443',
        ipv6: '0200:88fa:112a:00bb::9',
        pingMs: 38,
        transport: 'ShadowTLS v3',
        flag: '🇫🇮 HEL',
      ),
      PeerStatus(
        uri: 'udp://[fe80::1]:54328 (LAN Wi-Fi)',
        ipv6: '0200:99cc:33aa:2211::5',
        pingMs: 2,
        transport: 'LAN Multicast',
        flag: '🏠 LAN',
      ),
    ],
    this.selectedChatId = '1',
  });

  AppState copyWith({
    String? myIPv6,
    bool? isAntiDpiEnabled,
    bool? isProxyGDriveThroughYgg,
    bool? isMirroringEnabled,
    bool? isGDriveConnected,
    bool? isYandexConnected,
    int? mediaCacheLimitMB,
    List<ChatItem>? chats,
    List<PeerStatus>? peers,
    String? selectedChatId,
  }) {
    return AppState(
      myIPv6: myIPv6 ?? this.myIPv6,
      isAntiDpiEnabled: isAntiDpiEnabled ?? this.isAntiDpiEnabled,
      isProxyGDriveThroughYgg: isProxyGDriveThroughYgg ?? this.isProxyGDriveThroughYgg,
      isMirroringEnabled: isMirroringEnabled ?? this.isMirroringEnabled,
      isGDriveConnected: isGDriveConnected ?? this.isGDriveConnected,
      isYandexConnected: isYandexConnected ?? this.isYandexConnected,
      mediaCacheLimitMB: mediaCacheLimitMB ?? this.mediaCacheLimitMB,
      chats: chats ?? this.chats,
      peers: peers ?? this.peers,
      selectedChatId: selectedChatId ?? this.selectedChatId,
    );
  }
}

class AppStateNotifier extends StateNotifier<AppState> {
  AppStateNotifier() : super(const AppState());

  void selectChat(String chatId) {
    state = state.copyWith(selectedChatId: chatId);
  }

  void toggleAntiDpi(bool enabled) {
    state = state.copyWith(isAntiDpiEnabled: enabled);
  }

  void toggleProxyGDrive(bool enabled) {
    state = state.copyWith(isProxyGDriveThroughYgg: enabled);
  }

  void toggleMirroring(bool enabled) {
    state = state.copyWith(isMirroringEnabled: enabled);
  }

  void toggleGDrive(bool connected) {
    state = state.copyWith(isGDriveConnected: connected);
  }

  void toggleYandex(bool connected) {
    state = state.copyWith(isYandexConnected: connected);
  }

  void setCacheLimit(int limitMB) {
    state = state.copyWith(mediaCacheLimitMB: limitMB);
  }
}

final appStateProvider = StateNotifierProvider<AppStateNotifier, AppState>((ref) {
  return AppStateNotifier();
});
