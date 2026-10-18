// File: app/lib/ui/screens/anti_dpi_radar_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../core_bridge/ygg_bridge.dart';
import '../state/app_state.dart';
import '../state/core_state.dart';
import '../adaptive_shell.dart';

/// Экран сети: адрес, пиры и подключение берутся из ядра через protobuf.
class AntiDpiRadarScreen extends ConsumerStatefulWidget {
  const AntiDpiRadarScreen({super.key});

  @override
  ConsumerState<AntiDpiRadarScreen> createState() => _AntiDpiRadarScreenState();
}

class _AntiDpiRadarScreenState extends ConsumerState<AntiDpiRadarScreen> {
  final TextEditingController _peerUri = TextEditingController();

  @override
  void dispose() {
    _peerUri.dispose();
    super.dispose();
  }

  Future<void> _connectPeer() async {
    final uri = _peerUri.text.trim();
    if (uri.isEmpty) return;
    final controller = ref.read(coreProvider.notifier);
    String message;
    try {
      final peers = controller.connectPeer(uri);
      message = 'PIR добавлен: $uri • активных пиров: ${peers.length}';
      await controller.refresh();
    } on YggCoreException catch (e) {
      message = e.message;
    }
    _peerUri.clear();
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(message)),
    );
  }

  @override
  Widget build(BuildContext context) {
    final snapshot = ref.watch(coreProvider);
    final settings = ref.watch(appSettingsProvider);
    final online = snapshot.phase == CorePhase.online;

    return Scaffold(
      backgroundColor: const Color(0xFF0F172A),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            CorePhaseBanner(snapshot: snapshot),
            const SizedBox(height: 16),

            // Header Card: реальный IPv6 этой ноды
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
                          color: const Color(0xFF10B981).withValues(alpha: 0.15),
                          borderRadius: BorderRadius.circular(10),
                        ),
                        child: const Icon(Icons.radar, color: Color(0xFF10B981), size: 24),
                      ),
                      const SizedBox(width: 12),
                      const Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              'Yggdrasil-оверлей • userspace без root',
                              style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 16),
                            ),
                            Text(
                              'Без системного TUN/Wintun: пакеты идут через PacketConn ядра',
                              style: TextStyle(color: Color(0xFF94A3B8), fontSize: 11),
                            ),
                          ],
                        ),
                      ),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                        decoration: BoxDecoration(
                          color: online ? const Color(0xFF065F46) : const Color(0xFF1E293B),
                          borderRadius: BorderRadius.circular(20),
                        ),
                        child: Text(
                          online ? 'ОНЛАЙН' : 'НЕ В СЕТИ',
                          style: TextStyle(
                            color: online ? const Color(0xFF6EE7B7) : const Color(0xFF94A3B8),
                            fontSize: 10,
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 16),
                  const Text('Адрес, выведенный из Ed25519 ключа (0200::/7):',
                      style: TextStyle(color: Color(0xFF64748B), fontSize: 11)),
                  const SizedBox(height: 6),
                  InkWell(
                    onTap: snapshot.ipv6 == null
                        ? null
                        : () {
                            Clipboard.setData(ClipboardData(text: snapshot.ipv6!));
                            ScaffoldMessenger.of(context).showSnackBar(
                              const SnackBar(content: Text('IPv6 адрес скопирован в буфер обмена')),
                            );
                          },
                    borderRadius: BorderRadius.circular(8),
                    child: Container(
                      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                      decoration: BoxDecoration(
                        color: const Color(0xFF0F172A),
                        borderRadius: BorderRadius.circular(8),
                        border: Border.all(color: const Color(0xFF1E293B)),
                      ),
                      child: Row(
                        children: [
                          Expanded(
                            child: Text(
                              snapshot.ipv6 ?? 'нода не запущена',
                              style: TextStyle(
                                color: snapshot.ipv6 == null
                                    ? const Color(0xFF64748B)
                                    : const Color(0xFF34D399),
                                fontFamily: 'monospace',
                                fontSize: 12,
                              ),
                            ),
                          ),
                          const Icon(Icons.copy, color: Color(0xFF94A3B8), size: 16),
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 24),

            const Text('Обход блокировок', style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 14)),
            const SizedBox(height: 12),
            // Material, а не Container с цветом: ListTile рисует подложку и
            // всплеск на ближайшем Material, а цветная обёртка между ними эти
            // эффекты прячет — в debug-сборке это assertion, а не мелочь.
            Material(
              color: const Color(0xFF0B1120),
              shape: const RoundedRectangleBorder(
                borderRadius: BorderRadius.all(Radius.circular(16)),
                side: BorderSide(color: Color(0xFF1E293B)),
              ),
              child: SwitchListTile(
                activeThumbColor: const Color(0xFF10B981),
                title: const Text('TLS-обёртка ссылок yggdrasil вместо чистого TCP',
                    style: TextStyle(color: Colors.white, fontSize: 13, fontWeight: FontWeight.w600)),
                subtitle: Text(
                  settings.antiDpiOnNextStart
                      ? 'Передано ядру при старте ноды. Отпечаток TLS пока стандартный для Go, не Chrome'
                      : 'Применится на следующем StartNode: ссылки пойдут открытым TCP',
                  style: const TextStyle(color: Color(0xFF94A3B8), fontSize: 11),
                ),
                value: settings.antiDpiOnNextStart,
                onChanged: (val) =>
                    ref.read(appSettingsProvider.notifier).setAntiDpiOnNextStart(val),
              ),
            ),
            const SizedBox(height: 12),
            AntiDpiLiveStatus(snapshot: snapshot),
            const SizedBox(height: 24),

            // Реальные пиры
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text('Пиры ядра (${snapshot.peers.length})',
                    style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 14)),
                TextButton.icon(
                  onPressed: () => ref.read(coreProvider.notifier).refresh(),
                  icon: const Icon(Icons.refresh, size: 16, color: Color(0xFF10B981)),
                  label: const Text('Обновить', style: TextStyle(color: Color(0xFF10B981), fontSize: 12)),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _peerUri,
                    style: const TextStyle(color: Colors.white, fontSize: 12),
                    onSubmitted: (_) => _connectPeer(),
                    decoration: const InputDecoration(
                      hintText: 'tls://host:54321?sni=example.com',
                      hintStyle: TextStyle(color: Color(0xFF64748B), fontSize: 12),
                      filled: true,
                      fillColor: Color(0xFF0B1120),
                      contentPadding: EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                      border: OutlineInputBorder(
                        borderRadius: BorderRadius.all(Radius.circular(10)),
                        borderSide: BorderSide(color: Color(0xFF1E293B)),
                      ),
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                IconButton(
                  icon: const Icon(Icons.link, color: Color(0xFF10B981)),
                  tooltip: 'Добавить пир',
                  onPressed: _connectPeer,
                ),
              ],
            ),
            const SizedBox(height: 10),

            if (snapshot.peers.isEmpty)
              const _NoPeersHint()
            else
              ...snapshot.peers.map(_peerTile),
          ],
        ),
      ),
    );
  }

  Widget _peerTile(PeerInfo peer) {
    return Container(
      margin: const EdgeInsets.only(bottom: 10),
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: const Color(0xFF0B1120),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: const Color(0xFF1E293B)),
      ),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
            decoration: BoxDecoration(
              color: const Color(0xFF1E293B),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Text(
              peer.isConnected ? 'есть' : 'нет',
              style: const TextStyle(fontSize: 11, fontWeight: FontWeight.bold, color: Color(0xFF94A3B8)),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(peer.uri,
                    style: const TextStyle(color: Colors.white, fontSize: 12, fontWeight: FontWeight.w600),
                    overflow: TextOverflow.ellipsis),
                const SizedBox(height: 2),
                Text(
                  peer.remoteAddress.isEmpty ? peer.yggdrasilIpv6 : peer.remoteAddress,
                  style: const TextStyle(color: Color(0xFF64748B), fontSize: 11, fontFamily: 'monospace'),
                  overflow: TextOverflow.ellipsis,
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                decoration: BoxDecoration(
                  color: const Color(0xFF059669).withValues(alpha: 0.15),
                  borderRadius: BorderRadius.circular(6),
                ),
                child: Text(
                  _transportLabel(peer.transport),
                  style: const TextStyle(color: Color(0xFF34D399), fontSize: 10, fontWeight: FontWeight.bold),
                ),
              ),
              const SizedBox(height: 4),
              Text(
                peer.latencyMs == 0 ? 'rtt не измерен' : '${peer.latencyMs} ms',
                style: const TextStyle(color: Color(0xFF94A3B8), fontSize: 11),
              ),
            ],
          ),
        ],
      ),
    );
  }

  static String _transportLabel(PeerTransportType transport) {
    // Protobuf-перечисление — это класс, а не Dart enum, поэтому switch
    // не доказуемо исчерпывающий: неизвестное значение показываем как есть.
    if (transport == PeerTransportType.TRANSPORT_TCP) return 'TCP';
    if (transport == PeerTransportType.TRANSPORT_TLS_UTLS) return 'TLS';
    if (transport == PeerTransportType.TRANSPORT_WSS) return 'WS/WSS';
    if (transport == PeerTransportType.TRANSPORT_SHADOWTLS) return 'ShadowTLS';
    if (transport == PeerTransportType.TRANSPORT_QUIC_OBFS) return 'QUIC';
    return '${transport.value}';
  }
}

/// Живое объявление ядра про исходящие ссылки. Подписи полей повторяют то, как
/// их считает ядро (Шаг 16-d), и ни на шаг не шире: режим — намерение из
/// конфигурации старта, SNI — из URI ссылки, успех — «поднята хотя бы одна
/// исходящая ссылка», резервного переключения пути в живом коде нет вовсе.
/// Здесь нечего додумывать: событие — единственный источник, а команды
/// «спросить состояние обхода» в контракте нет.
class AntiDpiLiveStatus extends StatelessWidget {
  const AntiDpiLiveStatus({super.key, required this.snapshot});

  final CoreSnapshot snapshot;

  @override
  Widget build(BuildContext context) {
    final dpi = snapshot.antiDpi;

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: const Color(0xFF0B1120),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: const Color(0xFF1E293B)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(
                Icons.fiber_manual_record,
                size: 12,
                color: dpi == null
                    ? const Color(0xFF64748B)
                    : dpi.handshakeSuccess
                        ? const Color(0xFF34D399)
                        : const Color(0xFFF87171),
              ),
              const SizedBox(width: 8),
              const Text('Что объявило ядро',
                  style: TextStyle(
                      color: Colors.white,
                      fontSize: 12,
                      fontWeight: FontWeight.w600)),
            ],
          ),
          if (dpi == null)
            const Padding(
              padding: EdgeInsets.only(top: 8),
              child: Text(
                'Объявления ещё не было. Ядро присылает это событие, когда '
                'вершина исходящих ссылок меняется, а опросить его нечем — '
                'значит, тут нет «ссылок нет», есть «мы ещё не знаем».',
                style: TextStyle(color: Color(0xFF64748B), fontSize: 11, height: 1.35),
              ),
            )
          else
            ...[
              _row('Профиль', _modeLabel(dpi.mode)),
              const Padding(
                padding: EdgeInsets.only(top: 2, left: 88),
                child: Text(
                  'Намерение из конфигурации: рукопожатие строит встроенная '
                  'yggdrasil своим crypto/tls, поэтому отпечаток здесь описан, а '
                  'не измерен.',
                  style: TextStyle(color: Color(0xFF64748B), fontSize: 10, height: 1.35),
                ),
              ),
              _row('SNI', dpi.currentSni.isEmpty
                  ? 'ни у одной исходящей ссылки имени нет'
                  : dpi.currentSni),
              _row('Рукопожатие', dpi.handshakeSuccess
                  ? 'поднята хотя бы одна исходящая ссылка'
                  : 'ни одна исходящая ссылка не поднята'),
              _row('Резерв', dpi.fallbackTriggered
                  ? 'ядро сообщило о переходе на другой путь'
                  : 'переключения пути нет'),
              _row('Ссылки', dpi.transportDetails.isEmpty
                  ? 'ядро не прислало описания ссылок'
                  : dpi.transportDetails),
            ],
        ],
      ),
    );
  }

  static Widget _row(String label, String value) {
    return Padding(
      padding: const EdgeInsets.only(top: 6),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 88,
            child: Text(label,
                style: const TextStyle(color: Color(0xFF64748B), fontSize: 11)),
          ),
          Expanded(
            child: Text(value,
                style: const TextStyle(
                    color: Color(0xFF94A3B8), fontSize: 11, height: 1.35)),
          ),
        ],
      ),
    );
  }

  static String _modeLabel(AntiDpiMode mode) {
    // Та же причина, что у подписи транспорта: protobuf-перечисление — класс, а
    // не Dart enum, поэтому ветка «неизвестное значение» обязана быть явной, а
    // не молчаливым «отключено».
    if (mode == AntiDpiMode.ANTI_DPI_DISABLED) return 'без обхода, открытые TCP';
    if (mode == AntiDpiMode.ANTI_DPI_UTLS_CHROME) return 'TLS, профиль Chrome';
    if (mode == AntiDpiMode.ANTI_DPI_UTLS_IOS) return 'TLS, профиль iOS';
    if (mode == AntiDpiMode.ANTI_DPI_SHADOW_TLS_V3) return 'ShadowTLS v3';
    if (mode == AntiDpiMode.ANTI_DPI_WEBSOCKET_CDN) return 'WS/WSS через CDN';
    if (mode == AntiDpiMode.ANTI_DPI_UDP_QUIC_OBFS) return 'QUIC-обфускация';
    return 'профиль ${mode.value}';
  }
}

class _NoPeersHint extends StatelessWidget {
  const _NoPeersHint();

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: const Color(0xFF0B1120),
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: const Color(0xFF1E293B)),
      ),
      child: const Row(
        children: [
          Icon(Icons.info_outline, color: Color(0xFF64748B), size: 18),
          SizedBox(width: 10),
          Expanded(
            child: Text(
              'Пиров нет. Встроенная yggdrasil поднимает только те ссылки, что '
              'добавлены ниже: списка DoH/BEP-44-шлюзов клиент пока не задаёт.',
              style: TextStyle(color: Color(0xFF94A3B8), fontSize: 12),
            ),
          ),
        ],
      ),
    );
  }
}
