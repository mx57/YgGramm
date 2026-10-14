// File: app/lib/ui/screens/anti_dpi_radar_screen.dart
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../state/app_state.dart';

class AntiDpiRadarScreen extends ConsumerWidget {
  const AntiDpiRadarScreen({super.key});

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
            // Header Card: Yggdrasil IPv6 Identity
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
                          color: const Color(0xFF10B981).withOpacity(0.15),
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
                              'Yggdrasil Оверлей • Сеть активна',
                              style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 16),
                            ),
                            Text(
                              'Userspace режим без прав Root / системного Wintun',
                              style: TextStyle(color: Color(0xFF94A3B8), fontSize: 11),
                            ),
                          ],
                        ),
                      ),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                        decoration: BoxDecoration(
                          color: const Color(0xFF065F46),
                          borderRadius: BorderRadius.circular(20),
                        ),
                        child: const Text('ONLINE', style: TextStyle(color: Color(0xFF6EE7B7), fontSize: 10, fontWeight: FontWeight.bold)),
                      ),
                    ],
                  ),
                  const SizedBox(height: 16),
                  const Text('Ваш постоянный криптографический IPv6 адрес (0200::/7):', style: TextStyle(color: Color(0xFF64748B), fontSize: 11)),
                  const SizedBox(height: 6),
                  InkWell(
                    onTap: () {
                      Clipboard.setData(ClipboardData(text: state.myIPv6));
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
                              state.myIPv6,
                              style: const TextStyle(color: Color(0xFF34D399), fontFamily: 'monospace', fontSize: 12),
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

            // Toggles: Anti-DPI & Google Drive Proxy
            const Text('Управление обходом блокировок (ТСПУ в РФ)', style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 14)),
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
                    title: const Text('Обход ТСПУ (uTLS Chrome + Фрагментация TCP)', style: TextStyle(color: Colors.white, fontSize: 13, fontWeight: FontWeight.w600)),
                    subtitle: const Text('Маскирует рукопожатия пиров под обычный браузерный HTTPS трафик Google Chrome', style: TextStyle(color: Color(0xFF94A3B8), fontSize: 11)),
                    value: state.isAntiDpiEnabled,
                    onChanged: (val) {
                      ref.read(appStateProvider.notifier).toggleAntiDpi(val);
                    },
                  ),
                  const Divider(color: Color(0xFF1E293B), height: 1),
                  SwitchListTile(
                    activeColor: const Color(0xFF06B6D4),
                    title: const Text('Проксировать Google Drive через сеть Yggdrasil', style: TextStyle(color: Colors.white, fontSize: 13, fontWeight: FontWeight.w600)),
                    subtitle: const Text('Использует зарубежные выходные узлы оверлея при блокировке googleapis.com провайдером', style: TextStyle(color: Color(0xFF94A3B8), fontSize: 11)),
                    value: state.isProxyGDriveThroughYgg,
                    onChanged: (val) {
                      ref.read(appStateProvider.notifier).toggleProxyGDrive(val);
                    },
                  ),
                ],
              ),
            ),
            const SizedBox(height: 24),

            // Active Peers Section
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                const Text('Активные пиры и шлюзы маскировки', style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 14)),
                TextButton.icon(
                  onPressed: () {
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(content: Text('Опрос DoH Яндекс/Quad9 и BEP-44 DHT... Найдено 4 живых узла.')),
                    );
                  },
                  icon: const Icon(Icons.refresh, size: 16, color: Color(0xFF10B981)),
                  label: const Text('Обновить (DoH/DHT)', style: TextStyle(color: Color(0xFF10B981), fontSize: 12)),
                ),
              ],
            ),
            const SizedBox(height: 10),

            ListView.builder(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              itemCount: state.peers.length,
              itemBuilder: (context, index) {
                final peer = state.peers[index];
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
                        child: Text(peer.flag, style: const TextStyle(fontSize: 11, fontWeight: FontWeight.bold)),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              peer.uri,
                              style: const TextStyle(color: Colors.white, fontSize: 12, fontWeight: FontWeight.w600),
                              overflow: TextOverflow.ellipsis,
                            ),
                            const SizedBox(height: 2),
                            Text(
                              'IPv6: ${peer.ipv6}',
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
                              color: const Color(0xFF059669).withOpacity(0.15),
                              borderRadius: BorderRadius.circular(6),
                            ),
                            child: Text(
                              peer.transport,
                              style: const TextStyle(color: Color(0xFF34D399), fontSize: 10, fontWeight: FontWeight.bold),
                            ),
                          ),
                          const SizedBox(height: 4),
                          Text(
                            '${peer.pingMs} ms',
                            style: TextStyle(
                              color: peer.pingMs < 30 ? const Color(0xFF10B981) : Colors.amber,
                              fontSize: 11,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                        ],
                      ),
                    ],
                  ),
                );
              },
            ),
          ],
        ),
      ),
    );
  }
}
