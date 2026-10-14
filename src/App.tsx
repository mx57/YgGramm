/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useState, useEffect } from 'react';
import {
  Terminal,
  Shield,
  Layers,
  Cpu,
  Smartphone,
  Monitor,
  Code2,
  Share2,
  Lock,
  Radio,
  FileCode,
  CheckCircle2,
  RefreshCw,
  FolderTree,
  Send,
  Zap,
  Globe,
  Database,
  ExternalLink,
  Copy,
  Check
} from 'lucide-react';

interface FileViewerItem {
  path: string;
  badge: string;
  type: 'proto' | 'go' | 'dart' | 'script' | 'make';
  description: string;
}

const FILES_LIST: FileViewerItem[] = [
  {
    path: 'README.md',
    badge: 'Docs & Specs',
    type: 'script',
    description: 'Complete project documentation: feature matrix, architecture, Anti-DPI bypass, and build guides',
  },
  {
    path: 'AGENTS.md',
    badge: 'Agent Guidelines',
    type: 'script',
    description: 'Strict engineering standards for LLM agents: pure Go C-ABI, userspace networking, and zero-slop rules',
  },
  {
    path: 'DESIGN_SYSTEM.md',
    badge: 'Design System',
    type: 'script',
    description: 'Obsidian & Emerald luxury design system (Telegram Desktop × Linear), adaptive layout, and tokens',
  },
  {
    path: 'scripts/jules_setup.sh',
    badge: 'VM Automation',
    type: 'script',
    description: 'Automated Jules VM setup script: Go 1.23, Flutter 3.24, MinGW-w64, NDK, and validation suite',
  },
  {
    path: 'app/lib/ui/adaptive_shell.dart',
    badge: 'Flutter Adaptive',
    type: 'dart',
    description: 'Responsive UI: 3-column Telegram Desktop layout on Windows (>800px) and gesture navigation on Android',
  },
  {
    path: 'app/lib/ui/screens/anti_dpi_radar_screen.dart',
    badge: 'Anti-DPI Radar',
    type: 'dart',
    description: 'Visual network radar: IPv6 200::/7 identity, ping RTT, uTLS/WSS badges, TSPU evasion, and Google Drive proxy',
  },
  {
    path: 'app/lib/ui/screens/backup_settings_screen.dart',
    badge: 'Zero-Knowledge Backup',
    type: 'dart',
    description: 'Dual-cloud mirroring (Google Drive + Yandex.Disk simultaneously in Russia), FastCDC chunks, and cache limits',
  },
  {
    path: 'app/lib/ui/screens/onboarding_screen.dart',
    badge: 'BIP-39 Onboarding',
    type: 'dart',
    description: '12-word BIP-39 mnemonic generator, cloud restore from Google/Yandex/WebDAV, and LAN QR-code device pairing',
  },
  {
    path: 'app/lib/ui/state/app_state.dart',
    badge: 'Riverpod State',
    type: 'dart',
    description: 'Type-safe state management: ChatItem, MessageItem, PeerStatus, and reactive AppStateNotifier',
  },
  {
    path: 'app/android/app/src/main/kotlin/org/ygggram/service/YggForegroundService.kt',
    badge: 'Android Service',
    type: 'script',
    description: 'Android 14+ specialUse Foreground Service: keeps Go P2P core alive with screen off, ongoing notification',
  },
  {
    path: 'app/android/app/src/main/kotlin/org/ygggram/service/PushWakeupReceiver.kt',
    badge: 'Android Push',
    type: 'script',
    description: 'Wakeup Receiver: 5-sec WakeLock, S&F polling, NotificationCompat.MessagingStyle with Direct Reply',
  },
  {
    path: 'app/android/app/src/main/kotlin/org/ygggram/service/NetworkChangeCallback.kt',
    badge: 'Android Net',
    type: 'script',
    description: 'ConnectivityManager callback: Wi-Fi <-> Cellular handover detection and instant ReconnectPeers()',
  },
  {
    path: 'app/lib/platform/windows_integration.dart',
    badge: 'Windows Tray & Reg',
    type: 'dart',
    description: 'System tray minimize on [X], HKCU registry autostart, unread taskbar badge, and Domestic Bridge mode',
  },
  {
    path: 'core/pkg/media/swarm_transfer.go',
    badge: 'P2P Swarm Media',
    type: 'go',
    description: '256KB chunk streaming encryption (AEAD), Merkle tree, BlurHash preview, and multi-peer swarm seeding',
  },
  {
    path: 'core/pkg/channels/dag_channel.go',
    badge: 'Merkle-DAG Channels',
    type: 'go',
    description: 'Signed Merkle-DAG broadcast channels with epidemic Gossipsub relaying for offline author resistance',
  },
  {
    path: 'core/pkg/calls/ygg_webrtc.go',
    badge: 'Direct Ygg VoIP',
    type: 'go',
    description: 'E2E WebRTC signaling over Double Ratchet and direct Opus FEC / VP9 media streaming via 200::/7 without STUN/TURN',
  },
  {
    path: 'core/pkg/storage/encrypted_db.go',
    badge: 'Encrypted DB & FTS',
    type: 'go',
    description: 'On-disk AES-256-GCM AEAD vault, full-text tokenized search index, and CRDT Hybrid Logical Clocks',
  },
  {
    path: 'core/pkg/backup/snapshot_engine.go',
    badge: 'FastCDC Backup',
    type: 'go',
    description: 'Zero-Knowledge Content-Defined Chunking (64KB–1MB), deduplication, zlib compression, and Argon2id keys',
  },
  {
    path: 'core/pkg/backup/providers.go',
    badge: 'Cloud & LAN Sync',
    type: 'go',
    description: 'Unified storage: Google Drive (with Yggdrasil proxy), Yandex.Disk app:/, WebDAV Mail.ru, and direct P2P LAN sync',
  },
  {
    path: 'core/pkg/delivery/sf_relay.go',
    badge: 'Store-and-Forward',
    type: 'go',
    description: 'S&F Relay Server & MessageDispatcher: Blind MailboxID, Hashcash PoW, Quota 25MB, Direct QUIC fast path',
  },
  {
    path: 'core/pkg/delivery/wakeup_push.go',
    badge: 'Wakeup Push',
    type: 'go',
    description: 'Android blind wake: UnifiedPush ntfy, Anonymized FCM/HMS inside Yggdrasil, Adaptive battery-saving polling',
  },
  {
    path: 'core/pkg/contacts/discovery.go',
    badge: 'Contacts & DNS',
    type: 'go',
    description: 'Kademlia DHT ContactCard, Private Set Intersection (blinded OPRF phone matching), and Alfis DNS @username',
  },
  {
    path: 'core/pkg/crypto/identity.go',
    badge: 'BIP-39 & Keys',
    type: 'go',
    description: 'BIP-39 mnemonic, HKDF-SHA512 derivation, Master Ed25519/X25519, Device Certificates, Backup key',
  },
  {
    path: 'core/pkg/crypto/ratchet.go',
    badge: 'X3DH & Ratchet',
    type: 'go',
    description: 'Extended Triple Diffie-Hellman + Double Ratchet (KDF + DH X25519), out-of-order skipped key cache',
  },
  {
    path: 'core/pkg/crypto/sealed_sender.go',
    badge: 'Sealed Sender',
    type: 'go',
    description: 'Anonymous envelope: hidden sender, blind MailboxID routing for S&F relays, and Hashcash PoW',
  },
  {
    path: 'core/pkg/crypto/group_mls.go',
    badge: 'Group MLS',
    type: 'go',
    description: 'Sender Keys group architecture: symmetric ratchet chains, epoch rotations, forward secrecy',
  },
  {
    path: 'core/pkg/yggnode/node.go',
    badge: 'Userspace Core',
    type: 'go',
    description: 'Embedded Yggdrasil node: net.PacketConn over 200::/7, stream multiplexer, and Wi-Fi LAN multicast discovery',
  },
  {
    path: 'core/pkg/transport/antidpi.go',
    badge: 'Anti-DPI Dialer',
    type: 'go',
    description: 'TSPU evasion: TCP Record Fragmentation, uTLS Chrome fingerprinting, WSS-CDN over Cloudflare, ShadowTLS v3',
  },
  {
    path: 'core/pkg/transport/bootstrap.go',
    badge: 'Resilient Bootstrap',
    type: 'go',
    description: 'Censorship-resistant peer discovery: Yandex/Quad9 DoH TXT parsing, BEP-44 DHT Ed25519 verification, RTT failover ladder',
  },
  {
    path: 'core/cmd/libygggram/main.go',
    badge: 'Go C-ABI',
    type: 'go',
    description: 'Pure Go Userspace C-Shared library with exported InitCore, RegisterEventCallback, InvokeCommand',
  },
  {
    path: 'proto/ygggram.proto',
    badge: 'Protobuf v3',
    type: 'proto',
    description: 'Universal binary schema: NodeConfig, SealedEnvelope, ChatMessage, CoreRequest, CoreEvent',
  },
  {
    path: 'app/lib/core_bridge/ygg_bridge.dart',
    badge: 'Dart FFI',
    type: 'dart',
    description: 'Flutter NativeCallable.listener FFI bridge converting Go C-threads into reactive Stream<CoreEvent>',
  },
  {
    path: 'Makefile',
    badge: 'Build System',
    type: 'make',
    description: 'Unified Makefile for Protobuf generation and cross-compiling ygggram.dll and libygggram.so',
  },
  {
    path: 'scripts/build_windows.ps1',
    badge: 'PowerShell',
    type: 'script',
    description: 'Windows x64/ARM64 build script with CGO & MinGW-w64 toolchain checks',
  },
  {
    path: 'scripts/build_android.sh',
    badge: 'Bash / NDK',
    type: 'script',
    description: 'Android arm64-v8a / x86_64 cross-compilation script targeting NDK Clang',
  },
];

export default function App() {
  const [activeTab, setActiveTab] = useState<'overview' | 'files' | 'bridge_simulator' | 'anti_dpi'>('overview');
  const [selectedFilePath, setSelectedFilePath] = useState<string>('proto/ygggram.proto');
  const [copied, setCopied] = useState<boolean>(false);

  // Simulator state
  const [platform, setPlatform] = useState<'windows' | 'android'>('windows');
  const [antiDpiMode, setAntiDpiMode] = useState<string>('uTLS HelloChrome_Auto');
  const [isCoreRunning, setIsCoreRunning] = useState<boolean>(true);
  const [nodeIpv6, setNodeIpv6] = useState<string>('0200:4f2a:819b:90c1:da24:5512:eed8:102f');
  const [peerCount, setPeerCount] = useState<number>(4);
  const [rxBytes, setRxBytes] = useState<number>(142980);
  const [txBytes, setTxBytes] = useState<number>(85210);
  const [simulatedLogs, setSimulatedLogs] = useState<string[]>([
    '[FFI Bridge] DynamicLibrary loaded: ygggram.dll (x64)',
    '[CoreEngine] Userspace network stack initialized without TUN/Wintun root requirement',
    '[Anti-DPI] Dialer wrapped via utls.HelloChrome_Auto (SNI: www.cloudflare.com)',
    '[Yggdrasil] Node IPv6 assigned: 0200:4f2a:819b:90c1:da24:5512:eed8:102f',
    '[NativeCallable.listener] Event stream attached, listening to CoreEvent heartbeat',
  ]);
  const [outgoingText, setOutgoingText] = useState<string>('Привет из Yggdrasil! Трафик замаскирован под Chrome TLS.');

  // Auto increment simulated bytes
  useEffect(() => {
    if (!isCoreRunning) return;
    const interval = setInterval(() => {
      setRxBytes(prev => prev + Math.floor(Math.random() * 450) + 50);
      setTxBytes(prev => prev + Math.floor(Math.random() * 210) + 20);
    }, 2500);
    return () => clearInterval(interval);
  }, [isCoreRunning]);

  const handleCopyCode = () => {
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleSendMessage = () => {
    if (!outgoingText.trim()) return;
    const newLog = `[Event] Dispatched SealedEnvelope (DoubleRatchet + ChaCha20-Poly1305) Payload: "${outgoingText}"`;
    setSimulatedLogs(prev => [newLog, ...prev.slice(0, 15)]);
    setTxBytes(prev => prev + outgoingText.length + 96);
    setOutgoingText('');
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col font-sans selection:bg-emerald-500/30 selection:text-emerald-200">
      {/* Top Navigation Bar */}
      <header className="border-b border-slate-800 bg-slate-900/80 backdrop-blur sticky top-0 z-40 px-4 py-3">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-gradient-to-tr from-emerald-600 via-teal-500 to-cyan-400 flex items-center justify-center shadow-lg shadow-emerald-500/20">
              <Shield className="w-5 h-5 text-slate-950 font-bold" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <span className="font-bold text-lg tracking-tight text-white">YggGram</span>
                <span className="px-2 py-0.5 text-xs font-medium rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/20">
                  Core Foundation v1.0
                </span>
                <span className="px-2 py-0.5 text-xs font-medium rounded-full bg-cyan-500/10 text-cyan-400 border border-cyan-500/20">
                  Anti-DPI / ТСПУ
                </span>
              </div>
              <p className="text-xs text-slate-400">
                Overlay Messenger • Userspace Go 1.23+ • Flutter 3.24+ Dart FFI • Windows & Android
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <nav className="flex bg-slate-950/70 p-1 rounded-xl border border-slate-800 text-xs">
              <button
                onClick={() => setActiveTab('overview')}
                className={`px-3 py-1.5 rounded-lg transition-all font-medium ${
                  activeTab === 'overview'
                    ? 'bg-slate-800 text-white shadow-sm'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                Архитектура
              </button>
              <button
                onClick={() => setActiveTab('files')}
                className={`px-3 py-1.5 rounded-lg transition-all font-medium flex items-center gap-1.5 ${
                  activeTab === 'files'
                    ? 'bg-slate-800 text-white shadow-sm'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                <FolderTree className="w-3.5 h-3.5" />
                Файлы проекта
              </button>
              <button
                onClick={() => setActiveTab('bridge_simulator')}
                className={`px-3 py-1.5 rounded-lg transition-all font-medium flex items-center gap-1.5 ${
                  activeTab === 'bridge_simulator'
                    ? 'bg-slate-800 text-white shadow-sm'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                <Cpu className="w-3.5 h-3.5 text-emerald-400" />
                FFI Симулятор
              </button>
              <button
                onClick={() => setActiveTab('anti_dpi')}
                className={`px-3 py-1.5 rounded-lg transition-all font-medium flex items-center gap-1.5 ${
                  activeTab === 'anti_dpi'
                    ? 'bg-slate-800 text-white shadow-sm'
                    : 'text-slate-400 hover:text-slate-200'
                }`}
              >
                <Radio className="w-3.5 h-3.5 text-cyan-400" />
                Anti-DPI Матрица
              </button>
            </nav>
          </div>
        </div>
      </header>

      {/* Main Content Area */}
      <main className="flex-1 max-w-7xl w-full mx-auto p-4 md:p-6 space-y-6">
        {activeTab === 'overview' && (
          <div className="space-y-6">
            {/* Hero Highlights */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-5 relative overflow-hidden group hover:border-emerald-500/40 transition-colors">
                <div className="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
                  <Lock className="w-24 h-24 text-emerald-400" />
                </div>
                <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 flex items-center justify-center mb-3">
                  <Shield className="w-5 h-5" />
                </div>
                <h3 className="font-semibold text-white text-base">Userspace Без Root / Wintun</h3>
                <p className="text-xs text-slate-400 mt-1 leading-relaxed">
                  Go-ядро работает полностью в пространстве пользователя без создания системных сетевых адаптеров, TAP/TUN и без прав администратора.
                </p>
                <div className="mt-4 flex items-center gap-2 text-xs text-emerald-400 font-mono">
                  <CheckCircle2 className="w-3.5 h-3.5" /> net.PacketConn + gVisor/QUIC
                </div>
              </div>

              <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-5 relative overflow-hidden group hover:border-cyan-500/40 transition-colors">
                <div className="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
                  <Globe className="w-24 h-24 text-cyan-400" />
                </div>
                <div className="w-10 h-10 rounded-xl bg-cyan-500/10 border border-cyan-500/20 text-cyan-400 flex items-center justify-center mb-3">
                  <Radio className="w-5 h-5" />
                </div>
                <h3 className="font-semibold text-white text-base">Анти-DPI и Обход ТСПУ</h3>
                <p className="text-xs text-slate-400 mt-1 leading-relaxed">
                  Маскировка трафика через uTLS (HelloChrome_Auto, HelloIOS_Auto), ShadowTLS v3, WebSocket через CDN и UDP QUIC padding.
                </p>
                <div className="mt-4 flex items-center gap-2 text-xs text-cyan-400 font-mono">
                  <CheckCircle2 className="w-3.5 h-3.5" /> Бессигнатурный протокол
                </div>
              </div>

              <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-5 relative overflow-hidden group hover:border-indigo-500/40 transition-colors">
                <div className="absolute top-0 right-0 p-4 opacity-10 group-hover:opacity-20 transition-opacity">
                  <Cpu className="w-24 h-24 text-indigo-400" />
                </div>
                <div className="w-10 h-10 rounded-xl bg-indigo-500/10 border border-indigo-500/20 text-indigo-400 flex items-center justify-center mb-3">
                  <Layers className="w-5 h-5" />
                </div>
                <h3 className="font-semibold text-white text-base">Асинхронный Dart FFI Мост</h3>
                <p className="text-xs text-slate-400 mt-1 leading-relaxed">
                  Реактивный NativeCallable.listener для связи между Go-потоками и изолятом Flutter без блокировки UI-потока.
                </p>
                <div className="mt-4 flex items-center gap-2 text-xs text-indigo-400 font-mono">
                  <CheckCircle2 className="w-3.5 h-3.5" /> Protobuf v3 + Ring Buffer
                </div>
              </div>
            </div>

            {/* Architecture Pipeline Map */}
            <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-6">
              <h3 className="text-sm font-semibold uppercase tracking-wider text-slate-300 mb-4 flex items-center gap-2">
                <Layers className="w-4 h-4 text-emerald-400" /> Сквозная Схема Взаимодействия (Monorepo Blueprint)
              </h3>

              <div className="grid grid-cols-1 lg:grid-cols-4 gap-4 relative">
                {/* Layer 1: Flutter UI */}
                <div className="bg-slate-950/70 border border-slate-800/80 rounded-xl p-4 flex flex-col justify-between">
                  <div>
                    <div className="flex items-center gap-2 mb-2">
                      <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
                      <span className="text-xs font-semibold text-emerald-400 uppercase tracking-wider">Слой 1: Flutter 3.24+</span>
                    </div>
                    <div className="text-sm font-medium text-white">Кроссплатформенный UI</div>
                    <p className="text-xs text-slate-400 mt-1">
                      Windows Desktop (Tray, Split-view) и Android Material 3 (Foreground Service).
                    </p>
                  </div>
                  <div className="mt-4 pt-3 border-t border-slate-900 text-[11px] font-mono text-slate-400">
                    app/lib/core_bridge/ygg_bridge.dart
                  </div>
                </div>

                {/* Layer 2: FFI C-ABI */}
                <div className="bg-slate-950/70 border border-indigo-900/40 rounded-xl p-4 flex flex-col justify-between">
                  <div>
                    <div className="flex items-center gap-2 mb-2">
                      <span className="w-2 h-2 rounded-full bg-indigo-400"></span>
                      <span className="text-xs font-semibold text-indigo-400 uppercase tracking-wider">Слой 2: C-ABI Bridge</span>
                    </div>
                    <div className="text-sm font-medium text-white">dart:ffi + NativeCallable</div>
                    <p className="text-xs text-slate-400 mt-1">
                      Загрузка ygggram.dll / libygggram.so. Сериализация в Protobuf v3, нулевое копирование.
                    </p>
                  </div>
                  <div className="mt-4 pt-3 border-t border-slate-900 text-[11px] font-mono text-indigo-300">
                    InitCore / InvokeCommand / Callback
                  </div>
                </div>

                {/* Layer 3: Go 1.23 Userspace Core */}
                <div className="bg-slate-950/70 border border-cyan-900/40 rounded-xl p-4 flex flex-col justify-between">
                  <div>
                    <div className="flex items-center gap-2 mb-2">
                      <span className="w-2 h-2 rounded-full bg-cyan-400"></span>
                      <span className="text-xs font-semibold text-cyan-400 uppercase tracking-wider">Слой 3: Go Core</span>
                    </div>
                    <div className="text-sm font-medium text-white">Yggdrasil Userspace Stack</div>
                    <p className="text-xs text-slate-400 mt-1">
                      Криптография BIP-39, Ed25519, X25519, Double Ratchet, XChaCha20-Poly1305, PebbleDB.
                    </p>
                  </div>
                  <div className="mt-4 pt-3 border-t border-slate-900 text-[11px] font-mono text-cyan-300">
                    core/cmd/libygggram/main.go
                  </div>
                </div>

                {/* Layer 4: Anti-DPI & Overlay Network */}
                <div className="bg-slate-950/70 border border-purple-900/40 rounded-xl p-4 flex flex-col justify-between">
                  <div>
                    <div className="flex items-center gap-2 mb-2">
                      <span className="w-2 h-2 rounded-full bg-purple-400"></span>
                      <span className="text-xs font-semibold text-purple-400 uppercase tracking-wider">Слой 4: Anti-DPI Транспорт</span>
                    </div>
                    <div className="text-sm font-medium text-white">Overlay & Обход Блокировок</div>
                    <p className="text-xs text-slate-400 mt-1">
                      uTLS Chrome TLS Fingerprint, ShadowTLS v3, WebSocket CDN wss://, DHT BEP-44, DoH.
                    </p>
                  </div>
                  <div className="mt-4 pt-3 border-t border-slate-900 text-[11px] font-mono text-purple-300">
                    E2E SealedEnvelope
                  </div>
                </div>
              </div>
            </div>

            {/* Quick Stats & Compilation Targets */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-5">
                <h4 className="text-sm font-semibold text-slate-200 mb-3 flex items-center gap-2">
                  <Monitor className="w-4 h-4 text-emerald-400" />
                  Платформа Windows (x64 / ARM64)
                </h4>
                <ul className="space-y-2 text-xs text-slate-400">
                  <li className="flex items-center justify-between p-2 rounded-lg bg-slate-950/50">
                    <span>Сборка библиотеки:</span>
                    <span className="font-mono text-slate-200">ygggram.dll (c-shared)</span>
                  </li>
                  <li className="flex items-center justify-between p-2 rounded-lg bg-slate-950/50">
                    <span>Путь к данным:</span>
                    <span className="font-mono text-slate-200">%APPDATA%\YggGram</span>
                  </li>
                  <li className="flex items-center justify-between p-2 rounded-lg bg-slate-950/50">
                    <span>Компилятор:</span>
                    <span className="font-mono text-slate-200">MinGW-w64 GCC / Clang</span>
                  </li>
                  <li className="flex items-center justify-between p-2 rounded-lg bg-slate-950/50">
                    <span>Интеграция:</span>
                    <span className="font-mono text-emerald-400">System Tray + Auto-run</span>
                  </li>
                </ul>
              </div>

              <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-5">
                <h4 className="text-sm font-semibold text-slate-200 mb-3 flex items-center gap-2">
                  <Smartphone className="w-4 h-4 text-cyan-400" />
                  Платформа Android (ARM64 / x86_64)
                </h4>
                <ul className="space-y-2 text-xs text-slate-400">
                  <li className="flex items-center justify-between p-2 rounded-lg bg-slate-950/50">
                    <span>Сборка библиотеки:</span>
                    <span className="font-mono text-slate-200">libygggram.so (NDK clang)</span>
                  </li>
                  <li className="flex items-center justify-between p-2 rounded-lg bg-slate-950/50">
                    <span>Путь к данным:</span>
                    <span className="font-mono text-slate-200">context.filesDir</span>
                  </li>
                  <li className="flex items-center justify-between p-2 rounded-lg bg-slate-950/50">
                    <span>ABI цели:</span>
                    <span className="font-mono text-slate-200">arm64-v8a, x86_64 (API 24+)</span>
                  </li>
                  <li className="flex items-center justify-between p-2 rounded-lg bg-slate-950/50">
                    <span>Служба:</span>
                    <span className="font-mono text-cyan-400">Foreground Service + Battery Whitelist</span>
                  </li>
                </ul>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'files' && (
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
            {/* File Selector Sidebar */}
            <div className="lg:col-span-4 space-y-2">
              <div className="text-xs font-semibold uppercase tracking-wider text-slate-400 mb-3">
                Файлы текущего этапа
              </div>
              {FILES_LIST.map(file => (
                <button
                  key={file.path}
                  onClick={() => setSelectedFilePath(file.path)}
                  className={`w-full text-left p-3 rounded-xl border transition-all ${
                    selectedFilePath === file.path
                      ? 'bg-slate-900 border-emerald-500/50 shadow-sm'
                      : 'bg-slate-950/60 border-slate-800/80 hover:bg-slate-900/60'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-mono font-medium text-slate-200 truncate">
                      {file.path.split('/').pop()}
                    </span>
                    <span className="text-[10px] px-2 py-0.5 rounded-full bg-slate-800 text-slate-300 font-mono">
                      {file.badge}
                    </span>
                  </div>
                  <p className="text-[11px] text-slate-400 mt-1 line-clamp-2 leading-relaxed">
                    {file.description}
                  </p>
                </button>
              ))}
            </div>

            {/* File Content Preview */}
            <div className="lg:col-span-8 bg-slate-900/70 border border-slate-800 rounded-2xl overflow-hidden flex flex-col">
              <div className="bg-slate-950/80 px-4 py-3 border-b border-slate-800 flex items-center justify-between">
                <div className="flex items-center gap-2 font-mono text-xs text-slate-300">
                  <FileCode className="w-4 h-4 text-emerald-400" />
                  <span>/{selectedFilePath}</span>
                </div>
                <button
                  onClick={handleCopyCode}
                  className="flex items-center gap-1.5 px-2.5 py-1 text-xs rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 transition-colors"
                >
                  {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                  {copied ? 'Скопировано' : 'Копировать'}
                </button>
              </div>

              <div className="p-4 overflow-x-auto text-xs font-mono text-slate-300 leading-relaxed max-h-[600px] overflow-y-auto">
                {selectedFilePath === 'README.md' && (
                  <pre className="text-emerald-300">
{`# 🛡️ YggGram — GitHub Documentation
- Userspace Yggdrasil overlay network (0200::/7) without TUN/Wintun or root.
- Anti-DPI: TCP record fragmentation, uTLS Chrome 120+, WSS CDN, ShadowTLS v3.
- E2EE: BIP-39 seed -> X3DH + Double Ratchet -> Sealed Sender -> Group MLS.
- Offline: S&F relays with Hashcash PoW, UnifiedPush, Anonymized FCM proxy.
- Backups: Zero-Knowledge FastCDC chunks mirrored to Google Drive & Yandex.Disk.
- Calls: Direct WebRTC Opus FEC / VP9 streaming over IPv6 without STUN/TURN.`}
                  </pre>
                )}

                {selectedFilePath === 'AGENTS.md' && (
                  <pre className="text-cyan-300">
{`# 🤖 AGENTS.md — AI Engineering Standards
- Architecture: Pure Go 1.23+ C-Shared library + Dart FFI (NativeCallable.listener).
- Memory: Always allocate via C.CBytes and free via FreeMemory(ptr) C-ABI.
- Network: Strict userspace only. Never require Wintun or android.net.VpnService.
- Contracts: Changes must start with proto/ygggram.proto.
- Quality: Zero placeholders or TODOs allowed.`}
                  </pre>
                )}

                {selectedFilePath === 'DESIGN_SYSTEM.md' && (
                  <pre className="text-emerald-300">
{`# 🎨 DESIGN_SYSTEM.md — Obsidian & Emerald Design System
- Aesthetic: Telegram Desktop × Linear — Zero-latency feel, information density.
- Palette: Obsidian Deep (#080B10, #0E131B, #161D28) + Electric Emerald (#00E5A3) + Cyber Cyan (#00D2FF).
- Typography: Inter (UI text) + JetBrains Mono (IPv6 200::/7, Ed25519 keys, hashes).
- Desktop Layout (>900px): 3-column split view (Folders 64px, Chat List 320px, Active Chat, Drawer 300px).
- Android Layout: 1-handed ergonomic navigation, Swipe-to-reply, Haptic feedback, FLAG_SECURE.
- Anti-DPI Radar: Live peer RTT, TSPU bypass mode indicator (Direct P2P, WSS CDN, ShadowTLS).`}
                  </pre>
                )}

                {selectedFilePath === 'scripts/jules_setup.sh' && (
                  <pre className="text-purple-300">
{`#!/usr/bin/env bash
# File: scripts/jules_setup.sh — Automated Jules VM Setup Script
- [1/6] Installs build-essential, MinGW-w64 (gcc-mingw-w64-x86-64), protobuf-compiler, cmake.
- [2/6] Verifies / installs Go 1.23+ toolchain.
- [3/6] Configures protoc-gen-go generator.
- [4/6] Checks Android NDK r26+ environment.
- [5/6] Configures Flutter 3.24+ SDK.
- [6/6] Runs mandatory monorepo validation:
       * protoc compilation of ygggram.proto
       * go vet & unit tests in core/
       * Windows x64 DLL cross-compilation: build/ygggram.dll
       * flutter analyze in app/`}
                  </pre>
                )}

                {selectedFilePath === 'app/lib/ui/adaptive_shell.dart' && (
                  <pre className="text-teal-300">
{`// File: app/lib/ui/adaptive_shell.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

// Adaptive Responsive Shell:
// - Windows Desktop (>800px): 3-column Telegram Desktop split view (Nav Rail + Dialogs + Chat + Info Drawer)
// - Android Smartphone (<=800px): Compact bottom bar, swipe-to-reply gestures, smooth animations
class AdaptiveShell extends ConsumerStatefulWidget {
    @override
    Widget build(BuildContext context, WidgetRef ref) {
        final isDesktop = MediaQuery.of(context).size.width > 800;
        return isDesktop ? _buildDesktopLayout(...) : _buildMobileLayout(...);
    }
}`}
                  </pre>
                )}

                {selectedFilePath === 'app/lib/ui/screens/anti_dpi_radar_screen.dart' && (
                  <pre className="text-cyan-300">
{`// File: app/lib/ui/screens/anti_dpi_radar_screen.dart
import 'package:flutter/material.dart';

// Anti-DPI & Yggdrasil Network Radar:
// 1. Permanent Yggdrasil IPv6 address card (0200::/7).
// 2. Peer monitor with ping RTT, location badges, and transport tags:
//    [uTLS Chrome], [WSS CDN], [ShadowTLS v3], [LAN Multicast].
// 3. Evasion toggles: TSPU packet fragmentation & Google Drive proxy through Yggdrasil.`}
                  </pre>
                )}

                {selectedFilePath === 'app/lib/ui/screens/backup_settings_screen.dart' && (
                  <pre className="text-purple-300">
{`// File: app/lib/ui/screens/backup_settings_screen.dart
import 'package:flutter/material.dart';

// Zero-Knowledge Cloud Backup:
// 1. Dual mirroring: Google Drive (appDataFolder) + Yandex.Disk (app:/) simultaneously.
// 2. FastCDC content-defined chunking (64KB-1MB) with Argon2id encrypted blocks.
// 3. Media cache disk limit slider (512MB to 8GB) and force sync trigger.`}
                  </pre>
                )}

                {selectedFilePath === 'app/lib/ui/screens/onboarding_screen.dart' && (
                  <pre className="text-amber-300">
{`// File: app/lib/ui/screens/onboarding_screen.dart
import 'package:flutter/material.dart';

// Cryptographic Onboarding & Identity:
// 1. Generates 12-word BIP-39 mnemonic phrase without phone numbers or SMS.
// 2. Cloud restore choice: Google Drive, Yandex.Disk, or WebDAV.
// 3. Pair with nearby desktop/phone via local LAN Wi-Fi QR code.`}
                  </pre>
                )}

                {selectedFilePath === 'app/lib/ui/state/app_state.dart' && (
                  <pre className="text-emerald-300">
{`// File: app/lib/ui/state/app_state.dart
import 'package:flutter_riverpod/flutter_riverpod.dart';

// Riverpod State Management:
// - AppState: active chat, unread counts, peer latency list, anti-DPI toggles.
// - AppStateNotifier: reactive methods for selecting chats, switching evasion modes.
final appStateProvider = StateNotifierProvider<AppStateNotifier, AppState>((ref) => AppStateNotifier());`}
                  </pre>
                )}

                {selectedFilePath === 'app/android/app/src/main/kotlin/org/ygggram/service/YggForegroundService.kt' && (
                  <pre className="text-emerald-300">
{`// File: app/android/app/src/main/kotlin/org/ygggram/service/YggForegroundService.kt
package org.ygggram.service

// Android 14+ (API 34) Foreground Service (specialUse / connectedDevice)
// Keeps embedded Go Yggdrasil core active in background without battery optimizations killing it.
class YggForegroundService : Service() {
    private var wakeLock: PowerManager.WakeLock? = null
    private var networkChangeCallback: NetworkChangeCallback? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        // Starts foreground notification with peer count and Yggdrasil IPv6
        // Acquires Partial WakeLock to hold radio state
        return START_STICKY
    }
}`}
                  </pre>
                )}

                {selectedFilePath === 'app/android/app/src/main/kotlin/org/ygggram/service/PushWakeupReceiver.kt' && (
                  <pre className="text-cyan-300">
{`// File: app/android/app/src/main/kotlin/org/ygggram/service/PushWakeupReceiver.kt
package org.ygggram.service

// Push Wakeup Receiver (UnifiedPush & FCM blind ping handler):
// 1. Holds 5-second WakeLock to wake CPU without draining battery.
// 2. Polls S&F relays in Go-core, decrypts message via Double Ratchet.
// 3. Posts NotificationCompat.MessagingStyle with RemoteInput Direct Reply!
class PushWakeupReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        // Direct reply handler & background S&F fetch
    }
}`}
                  </pre>
                )}

                {selectedFilePath === 'app/android/app/src/main/kotlin/org/ygggram/service/NetworkChangeCallback.kt' && (
                  <pre className="text-indigo-300">
{`// File: app/android/app/src/main/kotlin/org/ygggram/service/NetworkChangeCallback.kt
package org.ygggram.service

// Monitors network handovers (Wi-Fi <-> Cellular, VPN toggles)
// Triggers instant ReconnectPeers() in Go core so sockets rebind immediately!
class NetworkChangeCallback(context: Context) : ConnectivityManager.NetworkCallback() {
    override fun onAvailable(network: Network) { ... }
    override fun onLost(network: Network) { ... }
}`}
                  </pre>
                )}

                {selectedFilePath === 'app/lib/platform/windows_integration.dart' && (
                  <pre className="text-purple-300">
{`// File: app/lib/platform/windows_integration.dart
import 'dart:io';

// Windows Desktop Integration:
// 1. System Tray: minimizes to notification area on [X] close button.
// 2. Autostart: pure HKCU\\Software\\Microsoft\\Windows\\CurrentVersion\\Run registry toggle (no admin rights).
// 3. Unread badge counter on Windows Taskbar icon.
// 4. "Help the Network" mode: acts as domestic relay bridge for user's mobile devices!
class WindowsIntegrationService {
    Future<bool> setAutostart(bool enable) async { ... }
    Future<void> setDomesticRelayBridgeMode(bool enable) async { ... }
    void minimizeToTray() { ... }
}`}
                  </pre>
                )}

                {selectedFilePath === 'core/pkg/media/swarm_transfer.go' && (
                  <pre className="text-teal-300">
{`// File: core/pkg/media/swarm_transfer.go
package media

// Heavy File P2P Swarm Transfer inside Yggdrasil:
// 1. 256KB chunking + AEAD (XChaCha20-Poly1305 / AES-256-GCM) + Merkle root.
// 2. BlurHash compact visual preview encoding (e.g. "LEHV6nWB2yk8py...").
// 3. Multi-threaded chunk downloading over QUIC with automatic local caching.
// 4. Autonomous swarm seeding: peers seed downloaded chunks directly to other overlay nodes.

func (sm *SwarmManager) EncryptAndCreateManifest(...) (*MediaManifest, error) { ... }
func (sm *SwarmManager) DownloadSwarmFile(...) ([]byte, error) { ... }
func ComputeBlurHash(imgBytes []byte) string { ... }`}
                  </pre>
                )}

                {selectedFilePath === 'core/pkg/channels/dag_channel.go' && (
                  <pre className="text-cyan-300">
{`// File: core/pkg/channels/dag_channel.go
package channels

// Cryptographic Signed Merkle-DAG Broadcast Channels:
// 1. Author signs post linking current DAG leaf heads as parents.
// 2. Subscribers synchronize the DAG via epidemic Gossipsub protocol.
// 3. Subscribers continuously relay posts and media to each other, creating
//    a censorship-immune channel that remains live even if author is offline.

type PostNode struct {
    PostID    string            // CID: SHA-256(canonical JSON)
    ChannelID string            // e.g. "@tech_insider"
    AuthorPub ed25519.PublicKey
    ParentIDs []string          // Merkle causal parents
    Title     string
    Body      string
    Signature []byte
}

func (m *DAGChannelManager) PublishPost(...) (*PostNode, error) { ... }
func (m *DAGChannelManager) IngestRemotePost(post PostNode) (bool, error) { ... }`}
                  </pre>
                )}

                {selectedFilePath === 'core/pkg/calls/ygg_webrtc.go' && (
                  <pre className="text-purple-300">
{`// File: core/pkg/calls/ygg_webrtc.go
package calls

// Direct Yggdrasil IPv6 WebRTC VoIP & Video Calls (No STUN/TURN Required):
// 1. Call signaling (Offer/Answer/Candidate/Hangup) inside E2E Double Ratchet channel.
// 2. Direct RTP media transport over Yggdrasil 200::/7 IPv6 address (port assigned on-the-fly).
// 3. Audio: Opus 48kHz with Inband FEC (Forward Error Correction) for packet-loss immunity.
// 4. Video: VP9/AV1 with MTU fragmentation + Windows screen sharing support.

func (cm *CallManager) InitiateCall(remoteIPv6 string, hasVideo, screenShare bool) (*CallSignalPayload, error) { ... }
func (cm *CallManager) AnswerCall(offer CallSignalPayload) (*CallSignalPayload, error) { ... }
func (s *DirectMediaSession) SendAudioPacket(opusBytes []byte) error { ... }`}
                  </pre>
                )}

                {selectedFilePath === 'core/pkg/storage/encrypted_db.go' && (
                  <pre className="text-emerald-300">
{`// File: core/pkg/storage/encrypted_db.go
package storage

// Zero-Knowledge Encrypted Local Database & CRDT
// 1. Record AEAD encryption (AES-256-GCM / XChaCha20-Poly1305 equivalent).
// 2. Tokenized inverted Full-Text Search (FTS) index over decrypted records.
// 3. Decentralized multi-device state synchronization via Hybrid Logical Clocks (HLC):
//    (mark read, pins, folders, "Delete for Everyone" tombstones).

func (db *EncryptedDB) SaveMessage(msg *StoredMessage) error { ... }
func (db *EncryptedDB) SearchMessages(query string, limit int) []*StoredMessage { ... }
func (db *EncryptedDB) ApplyCRDTOperation(op CRDTOperation) error { ... }`}
                  </pre>
                )}

                {selectedFilePath === 'core/pkg/backup/snapshot_engine.go' && (
                  <pre className="text-cyan-300">
{`// File: core/pkg/backup/snapshot_engine.go
package backup

// Zero-Knowledge Chunked Cloud Backup
// 1. Content-Defined Chunking (FastCDC algorithm): 64KB min, 256KB target, 1MB max.
// 2. Local deduplication: only new unseen chunk hashes are uploaded.
// 3. Compression (zlib) + AEAD encryption using master backup key.
// 4. Anonymized chunk tree layout: chunks/{hash[:2]}/{hash}.bin + manifest.enc.

func FastCDCSplit(data []byte) [][]byte { ... }
func (e *SnapshotEngine) ProcessFileIntoChunks(raw []byte) ([]ChunkDescriptor, uint64, error) { ... }
func (e *SnapshotEngine) CreateManifest(desc []ChunkDescriptor, plainBytes, chunkBytes uint64) ([]byte, error) { ... }`}
                  </pre>
                )}

                {selectedFilePath === 'core/pkg/backup/providers.go' && (
                  <pre className="text-purple-300">
{`// File: core/pkg/backup/providers.go
package backup

// Unified CloudProvider interface with 4 resilient backends:
// a) GoogleDriveProvider: hidden appDataFolder with Yggdrasil proxy fallback if Google is blocked.
// b) YandexDiskProvider: domestic Russian cloud (app:/) operating at line-rate in white-list mode.
// c) WebDAVProvider: Mail.ru Cloud and Nextcloud compatibility.
// d) LanDeviceSync: direct P2P chunk synchronization between Android and Windows over Wi-Fi / Yggdrasil.

type CloudProvider interface {
    UploadChunk(ctx context.Context, hash string, data []byte) error
    DownloadChunk(ctx context.Context, hash string) ([]byte, error)
    UploadManifest(ctx context.Context, manifest []byte) error
    DownloadManifest(ctx context.Context) ([]byte, error)
}`}
                  </pre>
                )}

                {selectedFilePath === 'core/pkg/delivery/sf_relay.go' && (
                  <pre className="text-teal-300">
{`// File: core/pkg/delivery/sf_relay.go
package delivery

// Store-and-Forward Relay: buffers offline messages for sleeping devices
// 1. Client registers blind MailboxID = BLAKE3/HMAC(PubKey + RotatingSalt) across 3 relays.
// 2. Anti-spam Hashcash PoW (16 zero bits) + 25 MB quota per mailbox + 14-day TTL.
// 3. MessageDispatcher: tries Direct QUIC delivery (1200 ms timeout), falls back to 3 S&F nodes.

func (s *SFRelayServer) PutEnvelope(mailboxID [32]byte, powNonce uint64, data []byte) error { ... }
func (s *SFRelayServer) PollMailbox(mailboxID [32]byte, recipientEdPub ed25519.PublicKey, authSig []byte, ts int64) ([]StoredEnvelope, error) { ... }
func (d *MessageDispatcher) DispatchMessage(ctx context.Context, ips []string, pub []byte, env *crypto.SealedEnvelope) (*DispatchResult, error) { ... }`}
                  </pre>
                )}

                {selectedFilePath === 'core/pkg/delivery/wakeup_push.go' && (
                  <pre className="text-cyan-300">
{`// File: core/pkg/delivery/wakeup_push.go
package delivery

// Multi-provider Android Wakeup Ping:
// a) UnifiedPush (RFC spec): zero-payload blind wakeup to self-hosted or public ntfy/gotify.
// b) Anonymized FCM/HMS: routed via internal Yggdrasil proxy gateway (Google never sees sender IP).
// c) Adaptive Battery-Saving Polling: dynamic sleep intervals (25s active -> 90s screen-off -> 600s Doze mode).

func (m *WakeupManager) SendUnifiedPushWakeup(ctx context.Context, endpoint string) error { ... }
func (m *WakeupManager) SendAnonymizedFCMWakeup(ctx context.Context, gatewayIPv6, pushToken string) error { ... }
func (m *WakeupManager) UpdateDevicePowerState(screenOn, powerSaver bool) { ... }`}
                  </pre>
                )}

                {selectedFilePath === 'core/pkg/contacts/discovery.go' && (
                  <pre className="text-amber-300">
{`// File: core/pkg/contacts/discovery.go
package contacts

// Contact Discovery & Synchronization:
// 1. Kademlia DHT: published signed ContactCard (device IPv6s, S&F relays, PreKeyBundle).
// 2. Private Set Intersection (PSI): blind OPRF phone matching (H_blind = H(phone) * salt) without revealing phone book!
// 3. Short @username resolver: native Alfis DNS (.ygg domain) query with DHT cache fallback.

func (m *ContactDiscoveryManager) SignAndPublishCard(card *ContactCard, priv ed25519.PrivateKey) error { ... }
func BlindIdentifier(identifier string) (*BlindContactItem, error) { ... }
func (m *ContactDiscoveryManager) ResolveUsername(ctx context.Context, username string) (*ContactCard, error) { ... }`}
                  </pre>
                )}

                {selectedFilePath === 'core/pkg/crypto/identity.go' && (
                  <pre className="text-emerald-300">
{`// File: core/pkg/crypto/identity.go
package crypto

// BIP-39 Mnemonic -> PBKDF2 512-bit Seed -> HKDF-SHA512 Key Derivation
type MasterIdentity struct {
    Mnemonic        string
    Seed            []byte             // 512 bits
    MasterEd25519   ed25519.PrivateKey // m/44'/9999'/0'
    MasterX25519    *ecdh.PrivateKey   // m/44'/9999'/1'
    BackupMasterKey [32]byte           // m/44'/9999'/3' (Cloud Zero-Knowledge backup)
}

// Issues cryptographically signed sub-device certificate (e.g. Android phone or Windows PC)
func (m *MasterIdentity) IssueDeviceIdentity(deviceID, deviceName string) (*DeviceIdentity, error) {
    // 1. Derives DeviceEd25519 + DeviceX25519 via HKDF
    // 2. Signs DeviceCertificate with MasterEd25519
}
func VerifyDeviceCertificate(masterPub ed25519.PublicKey, cert DeviceCertificate) bool { ... }`}
                  </pre>
                )}

                {selectedFilePath === 'core/pkg/crypto/ratchet.go' && (
                  <pre className="text-cyan-300">
{`// File: core/pkg/crypto/ratchet.go
package crypto

// X3DH (Extended Triple Diffie-Hellman) offline dialogue initiation
func PerformX3DHInitiator(senderIK *ecdh.PrivateKey, senderEdPub ed25519.PublicKey, bundle PreKeyBundle) (*X3DHHandshakeResult, error) {
    // DH1 = ECDH(IK_A, SPK_B)
    // DH2 = ECDH(EK_A, IK_B)
    // DH3 = ECDH(EK_A, SPK_B)
    // DH4 = ECDH(EK_A, OPK_B)
    // SK = HKDF-SHA256(DH1 || DH2 || DH3 [|| DH4])
}

// Double Ratchet (KDF-chain ratchet + asymmetric X25519 DH ratchet)
type DoubleRatchetSession struct {
    dhsPrivateKey   *ecdh.PrivateKey
    dhrPublicKey    *ecdh.PublicKey
    rootKey         []byte
    sendingChainKey []byte
    recvChainKey    []byte
    skippedKeys     map[string]SkippedKey // Handles out-of-order packets with TTL
}
func (s *DoubleRatchetSession) Encrypt(plaintext, associatedData []byte) (*EncryptedRatchetMessage, error) { ... }
func (s *DoubleRatchetSession) Decrypt(msg EncryptedRatchetMessage, associatedData []byte) ([]byte, error) { ... }`}
                  </pre>
                )}

                {selectedFilePath === 'core/pkg/crypto/sealed_sender.go' && (
                  <pre className="text-indigo-300">
{`// File: core/pkg/crypto/sealed_sender.go
package crypto

// Sealed Envelope: hides sender from Store-and-Forward (S&F) relays
// Relays only see blind MailboxID and PoW Hashcash token.
type SealedEnvelope struct {
    MailboxID       [32]byte // HMAC-SHA256(recipient_x25519_pub, epoch_day)
    EphemeralX25519 []byte   // 32-byte ephemeral public key
    Nonce           []byte   // 12-byte AEAD nonce
    EncryptedData   []byte   // Hidden inner sender identity + ratchet message
    PoWNonce        uint64   // Anti-spam Hashcash proof
}

func PackSealedEnvelope(...) (*SealedEnvelope, error) { ... }
func UnpackSealedEnvelope(...) (*SealedPayload, error) { ... }`}
                  </pre>
                )}

                {selectedFilePath === 'core/pkg/crypto/group_mls.go' && (
                  <pre className="text-purple-300">
{`// File: core/pkg/crypto/group_mls.go
package crypto

// Sender Keys group architecture (MLS / Signal specification)
// Each group participant maintains a symmetric ratchet chain.
type GroupSession struct {
    GroupID      string
    Epoch        uint64
    MyDeviceID   string
    MySigningKey ed25519.PrivateKey
    MyChainKey   []byte
    Members      map[string]*GroupMemberInfo
    SenderChains map[string]*SenderKeyState
}

func (g *GroupSession) RotateEpoch() (*SenderKeyDistributionMessage, error) { ... }
func (g *GroupSession) EncryptGroupMessage(plaintext []byte) (*EncryptedGroupMessage, error) { ... }
func (g *GroupSession) DecryptGroupMessage(msg *EncryptedGroupMessage) ([]byte, error) { ... }`}
                  </pre>
                )}

                {selectedFilePath === 'core/pkg/yggnode/node.go' && (
                  <pre className="text-teal-300">
{`// File: core/pkg/yggnode/node.go
package yggnode

// YggNode implements Userspace Overlay routing over 0200::/7
// Without root privileges, TAP/TUN drivers or Wintun.dll.
type YggNode struct {
    nodePrivKey   ed25519.PrivateKey
    nodePubKey    ed25519.PublicKey
    nodeIPv6      net.IP
    packetConn    *OverlayPacketConn
    streamMux     *OverlayStreamMux
    peerManager   *transport.PeerManager
    antiDpiDialer *transport.AntiDpiDialer
    lanPeers      map[string]*LANPeer
}

// Zero-root bidirectional stream multiplexer
func (m *OverlayStreamMux) OpenStream(targetIP net.IP) (*OverlayStream, error) { ... }
func (m *OverlayStreamMux) AcceptStream() (*OverlayStream, error) { ... }

// Automatic Wi-Fi / Hotspot LAN Multicast Discovery (239.255.255.250:54328)
// Discovers offline peers without internet or central bootstrap nodes!
func (n *YggNode) runLANDiscoveryBeacon() { ... }
func (n *YggNode) runLANDiscoveryListener() { ... }`}
                  </pre>
                )}

                {selectedFilePath === 'core/pkg/transport/antidpi.go' && (
                  <pre className="text-cyan-300">
{`// File: core/pkg/transport/antidpi.go
package transport

// Anti-DPI / ТСПУ Circumvention Dialer
// 1. TCP Record Fragmentation (splits ClientHello to defeat SNI signature matchers)
// 2. uTLS Chrome TLS fingerprint spoofing (HelloChrome_Auto, ALPN h2, http/1.1)
// 3. WSS-CDN encapsulation (binary WebSocket over Cloudflare / GCore edge)
// 4. ShadowTLS v3 dual-handshake with whitelisted SNI (yandex.ru / vk.com)

type fragmentedConn struct {
    net.Conn
    splitPoint int // e.g. 5 bytes (TLS record header split)
    wroteFirst bool
}

func (d *AntiDpiDialer) dialWebSocketCDN(ctx context.Context, addr string) (net.Conn, error) {
    // Upgrades TLS stream to WebSocket, frames Yggdrasil packets in 0x02 Binary frames
}

func (d *AntiDpiDialer) dialShadowTLS(ctx context.Context, addr string) (net.Conn, error) {
    // Sends authenticated HMAC token before proxy handoff
}`}
                  </pre>
                )}

                {selectedFilePath === 'core/pkg/transport/bootstrap.go' && (
                  <pre className="text-amber-300">
{`// File: core/pkg/transport/bootstrap.go
package transport

// Censorship-resistant peer discovery:
// 1. DNS-over-HTTPS (DoH) via Yandex.DNS (https://common.dns.yandex.net/dns-query)
//    and Quad9 to read signed base64 TXT peer records.
// 2. BitTorrent Mainline DHT (BEP-44) mutable item resolver with Ed25519 signature checks.
// 3. PeerManager with active RTT health-checks and failover escalation ladder:
//    Direct TLS -> uTLS Fragmented -> WSS CDN -> Domestic Peer Bridge

func (pm *PeerManager) FetchPeersViaDoH(ctx context.Context, domainName string) ([]string, error) { ... }
func (pm *PeerManager) VerifyAndLoadBEP44Item(item DHTMutableItem) ([]string, error) { ... }
func (pm *PeerManager) EscalateTransportLadder(currentMode AntiDpiMode) AntiDpiMode { ... }`}
                  </pre>
                )}

                {selectedFilePath === 'proto/ygggram.proto' && (
                  <pre className="text-emerald-300">
{`syntax = "proto3";
package ygggram;

option go_package = "ygggram/core/proto;yggproto";
option java_package = "org.ygggram.proto";

enum AntiDpiMode {
  ANTI_DPI_DISABLED = 0;
  ANTI_DPI_UTLS_CHROME = 1;
  ANTI_DPI_UTLS_IOS = 2;
  ANTI_DPI_SHADOW_TLS_V3 = 3;
  ANTI_DPI_WEBSOCKET_CDN = 4;
  ANTI_DPI_UDP_QUIC_OBFS = 5;
}

message NodeConfig {
  string data_dir = 1;
  string cache_dir = 2;
  string mnemonic_seed = 3;
  bytes ed25519_node_privkey = 4;
  bytes x25519_chat_privkey = 5;
  bytes master_key_256 = 6;
  AntiDpiMode anti_dpi_mode = 7;
  string anti_dpi_sni = 8;
  repeated string bootstrap_peers = 10;
  bool userspace_only = 15;
}

message SealedEnvelope {
  bytes recipient_x25519_pubkey = 1;
  bytes ephemeral_x25519_pubkey = 2;
  bytes encrypted_payload = 3;
  bytes nonce = 4;
  uint64 timestamp = 5;
  uint64 pow_nonce = 6;
  bytes blind_token = 7;
}

message ChatMessage {
  string message_id = 1;
  string conversation_id = 2;
  string sender_node_ipv6 = 3;
  uint64 timestamp_ms = 5;
  oneof content {
    TextContent text = 7;
    ReplyContent reply = 8;
    ReactionContent reaction = 9;
    MediaManifest media = 10;
    CallSignalContent call_signal = 11;
  }
}

message CoreRequest {
  uint64 request_id = 1;
  oneof request {
    StartNodeReq start_node = 2;
    StopNodeReq stop_node = 3;
    SendMessageReq send_message = 4;
    ConnectPeerReq connect_peer = 5;
    GetPeersReq get_peers = 7;
    GetNodeStatusReq get_node_status = 8;
  }
}

message CoreEvent {
  uint64 event_id = 1;
  uint64 timestamp_ms = 2;
  oneof event {
    NetworkStatusEvent network_status = 3;
    PeerListUpdatedEvent peers_updated = 4;
    AntiDpiStatusEvent anti_dpi_status = 5;
    IncomingMessageEvent incoming_message = 6;
    BackupProgressEvent backup_progress = 7;
    IncomingCallEvent incoming_call = 8;
  }
}`}
                  </pre>
                )}

                {selectedFilePath === 'core/cmd/libygggram/main.go' && (
                  <pre className="text-cyan-300">
{`package main

/*
#include <stdlib.h>
#include <stdint.h>

typedef void (*EventCallback)(const uint8_t* event_data, int length);

static inline void invoke_event_callback(EventCallback cb, const uint8_t* data, int length) {
    if (cb != NULL) {
        cb(data, length);
    }
}
*/
import "C"

//export InitCore
func InitCore(configBytes *C.uchar, length C.int) C.int {
    // Configures data dirs (%APPDATA% or Android filesDir)
    // Generates Ed25519 node key and X25519 chat key
    // Starts Userspace network loop and event ticker
    return 0
}

//export RegisterEventCallback
func RegisterEventCallback(cb C.EventCallback) {
    // Stores callback pointer for NativeCallable.listener
}

//export InvokeCommand
func InvokeCommand(reqBytes *C.uchar, length C.int, respPtr **C.uchar, respLen *C.int) C.int {
    // Dispatches RPC commands and returns C.CBytes memory
    return 0
}

//export FreeMemory
func FreeMemory(ptr *C.uchar) {
    if ptr != nil {
        C.free(unsafe.Pointer(ptr))
    }
}`}
                  </pre>
                )}

                {selectedFilePath === 'app/lib/core_bridge/ygg_bridge.dart' && (
                  <pre className="text-indigo-300">
{`class YggCoreBridge {
  static final YggCoreBridge _instance = YggCoreBridge._internal();
  factory YggCoreBridge() => _instance;

  DynamicLibrary? _lib;
  NativeCallable<CNativeEventCallback>? _nativeCallable;
  final StreamController<CoreEvent> _eventStreamController =
      StreamController<CoreEvent>.broadcast();

  Stream<CoreEvent> get eventStream => _eventStreamController.stream;

  void loadLibrary({String? customPath}) {
    if (Platform.isWindows) {
      _lib = DynamicLibrary.open('ygggram.dll');
    } else if (Platform.isAndroid) {
      _lib = DynamicLibrary.open('libygggram.so');
    }
    _setupNativeCallback();
  }

  void _setupNativeCallback() {
    _nativeCallable = NativeCallable<CNativeEventCallback>.listener(
      _onNativeEventReceived,
    );
    _registerEventCallback(_nativeCallable!.nativeFunction);
  }

  Uint8List invokeCommand({required int commandType, Uint8List? payload}) {
    // Marshals request, calls C function, reads response, calls FreeMemory()
  }
}`}
                  </pre>
                )}

                {selectedFilePath === 'Makefile' && (
                  <pre className="text-amber-300">
{`all: proto windows android

proto:
	protoc --proto_path=proto --go_out=core/pkg/yggproto --dart_out=app/lib/proto proto/ygggram.proto

windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \\
		go build -buildmode=c-shared -ldflags="-s -w" -o build/windows/x64/ygggram.dll ./core/cmd/libygggram

android-arm64:
	bash scripts/build_android.sh arm64-v8a

android-x64:
	bash scripts/build_android.sh x86_64`}
                  </pre>
                )}

                {(selectedFilePath === 'scripts/build_windows.ps1' || selectedFilePath === 'scripts/build_android.sh') && (
                  <pre className="text-teal-300">
{`# Automated Cross-Compilation Toolchain
# Configures CGO, Target Triples, NDK Clang / MinGW-w64
# Outputs shared libraries and deploys them to Flutter runner assets`}
                  </pre>
                )}
              </div>
            </div>
          </div>
        )}

        {activeTab === 'bridge_simulator' && (
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
            {/* Control & Parameters */}
            <div className="lg:col-span-5 space-y-4">
              <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-5 space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="font-semibold text-white text-sm flex items-center gap-2">
                    <Cpu className="w-4 h-4 text-emerald-400" /> Параметры FFI Моста
                  </h3>
                  <span className={`px-2 py-0.5 rounded-full text-[10px] font-mono ${
                    isCoreRunning ? 'bg-emerald-500/20 text-emerald-400' : 'bg-rose-500/20 text-rose-400'
                  }`}>
                    {isCoreRunning ? 'ACTIVE' : 'STOPPED'}
                  </span>
                </div>

                {/* Target OS Switcher */}
                <div>
                  <label className="text-xs text-slate-400 font-medium block mb-1.5">Целевая ОС (FFI Target)</label>
                  <div className="grid grid-cols-2 gap-2">
                    <button
                      onClick={() => setPlatform('windows')}
                      className={`p-2.5 rounded-xl border text-xs font-medium flex items-center justify-center gap-2 transition-all ${
                        platform === 'windows'
                          ? 'bg-emerald-500/10 border-emerald-500/40 text-emerald-300'
                          : 'bg-slate-950/60 border-slate-800 text-slate-400'
                      }`}
                    >
                      <Monitor className="w-4 h-4" /> Windows (ygggram.dll)
                    </button>
                    <button
                      onClick={() => setPlatform('android')}
                      className={`p-2.5 rounded-xl border text-xs font-medium flex items-center justify-center gap-2 transition-all ${
                        platform === 'android'
                          ? 'bg-cyan-500/10 border-cyan-500/40 text-cyan-300'
                          : 'bg-slate-950/60 border-slate-800 text-slate-400'
                      }`}
                    >
                      <Smartphone className="w-4 h-4" /> Android (libygggram.so)
                    </button>
                  </div>
                </div>

                {/* Anti-DPI Profile */}
                <div>
                  <label className="text-xs text-slate-400 font-medium block mb-1.5">Режим обхода DPI / ТСПУ</label>
                  <select
                    value={antiDpiMode}
                    onChange={(e) => setAntiDpiMode(e.target.value)}
                    className="w-full bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs text-slate-200 focus:outline-none focus:border-emerald-500"
                  >
                    <option value="uTLS HelloChrome_Auto">uTLS (HelloChrome_Auto) — TLS Fingerprint Spoofing</option>
                    <option value="uTLS HelloIOS_Auto">uTLS (HelloIOS_Auto) — Apple Safari Spoofing</option>
                    <option value="ShadowTLS v3">ShadowTLS v3 — Dual Handshake via whitelisted SNI</option>
                    <option value="WebSocket CDN (wss://)">WebSocket (wss://) — Cloudflare CDN Masking</option>
                    <option value="QUIC UDP Obfs">QUIC UDP Padding — Zero-signature randomized jitter</option>
                  </select>
                </div>

                {/* Storage Path */}
                <div>
                  <label className="text-xs text-slate-400 font-medium block mb-1.5">Каталог БД (Zero-Root)</label>
                  <div className="bg-slate-950 px-3 py-2 rounded-xl border border-slate-800 text-xs font-mono text-slate-300">
                    {platform === 'windows' ? '%APPDATA%\\YggGram\\data' : '/data/user/0/org.ygggram/files'}
                  </div>
                </div>

                {/* Actions */}
                <div className="flex gap-2 pt-2">
                  <button
                    onClick={() => {
                      setIsCoreRunning(!isCoreRunning);
                      setSimulatedLogs(prev => [
                        `[Bridge] invokeCommand: ${isCoreRunning ? 'StopNodeReq' : 'StartNodeReq'} dispatched`,
                        ...prev
                      ]);
                    }}
                    className={`flex-1 py-2 px-3 rounded-xl text-xs font-medium transition-colors flex items-center justify-center gap-1.5 ${
                      isCoreRunning
                        ? 'bg-rose-500/10 hover:bg-rose-500/20 text-rose-400 border border-rose-500/20'
                        : 'bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-semibold'
                    }`}
                  >
                    {isCoreRunning ? 'Остановить узел' : 'Запустить узел'}
                  </button>
                  <button
                    onClick={() => {
                      setRxBytes(0);
                      setTxBytes(0);
                      setSimulatedLogs(['[Bridge] Counters reset']);
                    }}
                    className="p-2 rounded-xl bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs transition-colors"
                    title="Сброс счетчиков"
                  >
                    <RefreshCw className="w-4 h-4" />
                  </button>
                </div>
              </div>

              {/* Node IPv6 Card */}
              <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-4">
                <div className="text-xs text-slate-400 mb-1">Детерминированный Yggdrasil IPv6 адрес:</div>
                <div className="font-mono text-xs text-emerald-400 bg-slate-950 p-2.5 rounded-xl border border-slate-800 select-all">
                  {nodeIpv6}
                </div>
                <div className="mt-3 grid grid-cols-3 gap-2 text-center text-xs">
                  <div className="bg-slate-950/70 p-2 rounded-lg border border-slate-800/60">
                    <div className="text-slate-400 text-[10px]">Пиры</div>
                    <div className="font-bold text-white mt-0.5">{peerCount}</div>
                  </div>
                  <div className="bg-slate-950/70 p-2 rounded-lg border border-slate-800/60">
                    <div className="text-slate-400 text-[10px]">Входящие (RX)</div>
                    <div className="font-bold text-cyan-400 mt-0.5">{(rxBytes / 1024).toFixed(1)} KB</div>
                  </div>
                  <div className="bg-slate-950/70 p-2 rounded-lg border border-slate-800/60">
                    <div className="text-slate-400 text-[10px]">Исходящие (TX)</div>
                    <div className="font-bold text-indigo-400 mt-0.5">{(txBytes / 1024).toFixed(1)} KB</div>
                  </div>
                </div>
              </div>
            </div>

            {/* Live Event Stream / Dispatcher */}
            <div className="lg:col-span-7 bg-slate-900/60 border border-slate-800 rounded-2xl flex flex-col overflow-hidden">
              <div className="bg-slate-950/80 px-4 py-3 border-b border-slate-800 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Terminal className="w-4 h-4 text-emerald-400" />
                  <span className="text-xs font-semibold text-slate-200">
                    Реактивный Stream&lt;CoreEvent&gt; (NativeCallable.listener)
                  </span>
                </div>
                <span className="text-[10px] font-mono text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full border border-emerald-500/20">
                  Ring Buffer OK
                </span>
              </div>

              {/* Event Stream Log */}
              <div className="p-4 flex-1 font-mono text-xs space-y-2 overflow-y-auto max-h-[360px] bg-slate-950/40">
                {simulatedLogs.map((log, index) => (
                  <div
                    key={index}
                    className="p-2 rounded-lg bg-slate-900/60 border border-slate-800/60 text-slate-300 flex items-start gap-2"
                  >
                    <span className="text-emerald-500 text-[10px] mt-0.5">▶</span>
                    <span className="break-all">{log}</span>
                  </div>
                ))}
              </div>

              {/* Send SealedEnvelope input */}
              <div className="p-3 border-t border-slate-800 bg-slate-950/70">
                <div className="text-[11px] text-slate-400 mb-1 flex items-center justify-between">
                  <span>Отправка тестового E2E конверта SealedEnvelope (RPC 0x05):</span>
                  <span className="text-emerald-400 font-mono text-[10px]">Double Ratchet + ChaCha20</span>
                </div>
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={outgoingText}
                    onChange={(e) => setOutgoingText(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && handleSendMessage()}
                    placeholder="Введите текст сообщения для шифрования..."
                    className="flex-1 bg-slate-900 border border-slate-800 rounded-xl px-3 py-2 text-xs text-slate-200 focus:outline-none focus:border-emerald-500"
                  />
                  <button
                    onClick={handleSendMessage}
                    className="bg-emerald-500 hover:bg-emerald-400 text-slate-950 font-semibold px-4 py-2 rounded-xl text-xs flex items-center gap-1.5 transition-colors"
                  >
                    <Send className="w-3.5 h-3.5" />
                    Отправить
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'anti_dpi' && (
          <div className="space-y-4">
            <div className="bg-slate-900/60 border border-slate-800 rounded-2xl p-6">
              <h3 className="text-base font-semibold text-white mb-2 flex items-center gap-2">
                <Radio className="w-5 h-5 text-cyan-400" />
                Механизмы Защиты от Анализа Трафика (DPI / ТСПУ)
              </h3>
              <p className="text-xs text-slate-400 max-w-3xl leading-relaxed">
                В классическом Yggdrasil соединения пиров имеют характерный TLS handshake или бинарные заголовки, легко определяемые современными сигнатурными фильтрами ТСПУ. YggGram встраивает многоуровневый эшелон обхода.
              </p>

              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 mt-6">
                <div className="bg-slate-950/70 border border-slate-800 rounded-xl p-4">
                  <div className="w-8 h-8 rounded-lg bg-emerald-500/10 text-emerald-400 flex items-center justify-center font-bold text-xs mb-3">
                    01
                  </div>
                  <h4 className="text-sm font-semibold text-white">uTLS Fingerprint Spoofing</h4>
                  <p className="text-xs text-slate-400 mt-1">
                    Подмена ClientHello сигнатур на легитимные отпечатки Google Chrome (HelloChrome_Auto) или Safari iOS. ТСПУ видит обычный браузерный сеанс.
                  </p>
                </div>

                <div className="bg-slate-950/70 border border-slate-800 rounded-xl p-4">
                  <div className="w-8 h-8 rounded-lg bg-cyan-500/10 text-cyan-400 flex items-center justify-center font-bold text-xs mb-3">
                    02
                  </div>
                  <h4 className="text-sm font-semibold text-white">WebSocket поверх CDN (wss://)</h4>
                  <p className="text-xs text-slate-400 mt-1">
                    Туннелирование пирингового трафика внутри стандартного HTTPS WebSocket через Cloudflare / CDN провайдеров. Блокировка невозможна без блокировки всего CDN.
                  </p>
                </div>

                <div className="bg-slate-950/70 border border-slate-800 rounded-xl p-4">
                  <div className="w-8 h-8 rounded-lg bg-indigo-500/10 text-indigo-400 flex items-center justify-center font-bold text-xs mb-3">
                    03
                  </div>
                  <h4 className="text-sm font-semibold text-white">ShadowTLS v3 Tunneling</h4>
                  <p className="text-xs text-slate-400 mt-1">
                    Двойной рукопожатие: начальный TLS выполняется с реальным сервером (например, cloudflare.com), а после валидации пароля передается в Yggdrasil узел.
                  </p>
                </div>

                <div className="bg-slate-950/70 border border-slate-800 rounded-xl p-4">
                  <div className="w-8 h-8 rounded-lg bg-purple-500/10 text-purple-400 flex items-center justify-center font-bold text-xs mb-3">
                    04
                  </div>
                  <h4 className="text-sm font-semibold text-white">QUIC UDP Padding & Obfs</h4>
                  <p className="text-xs text-slate-400 mt-1">
                    Случайный паддинг размеров пакетов и джиттер интервалов между пакетами предотвращают тайминг-атаки и статистический ML-анализ потоков.
                  </p>
                </div>

                <div className="bg-slate-950/70 border border-slate-800 rounded-xl p-4">
                  <div className="w-8 h-8 rounded-lg bg-amber-500/10 text-amber-400 flex items-center justify-center font-bold text-xs mb-3">
                    05
                  </div>
                  <h4 className="text-sm font-semibold text-white">DHT BEP-44 & DoH Bootstrap</h4>
                  <p className="text-xs text-slate-400 mt-1">
                    Поиск начальных входных узлов через децентрализованную BitTorrent Mainline DHT и DNS-over-HTTPS даже при блокировке официальных GitHub репозиториев.
                  </p>
                </div>

                <div className="bg-slate-950/70 border border-slate-800 rounded-xl p-4">
                  <div className="w-8 h-8 rounded-lg bg-rose-500/10 text-rose-400 flex items-center justify-center font-bold text-xs mb-3">
                    06
                  </div>
                  <h4 className="text-sm font-semibold text-white">Sealed Sender & S&F</h4>
                  <p className="text-xs text-slate-400 mt-1">
                    Скрытие адреса отправителя: промежуточные релеи видят только слепой токен и recipient key, не зная, от кого исходит сообщение.
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-800/80 bg-slate-950 py-4 px-6 text-xs text-slate-500 flex flex-col md:flex-row items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span className="w-2 h-2 rounded-full bg-emerald-400"></span>
          <span>YggGram Engine • Ready for compilation via Makefile & build scripts</span>
        </div>
        <div className="flex items-center gap-4 text-slate-400 font-mono text-[11px]">
          <span>Windows (ygggram.dll)</span>
          <span>•</span>
          <span>Android (libygggram.so)</span>
          <span>•</span>
          <span>Protobuf v3</span>
        </div>
      </footer>
    </div>
  );
}
