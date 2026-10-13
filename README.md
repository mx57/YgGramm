# 🛡️ YggGram

> **Децентрализованный кроссплатформенный мессенджер с защитой от анализа трафика (DPI / ТСПУ), сквозным шифрованием и работой в оверлейной сети Yggdrasil.**
> 
> *Поддерживаемые платформы:* **Windows (x64 / ARM64)** & **Android (ARM64 / x86_64)**  
> *Ядро:* **Pure Go 1.23+ (C-Shared Library, Userspace)** • *Клиент:* **Flutter 3.24+ (Dart FFI & Riverpod)**

---

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white)](https://golang.org)
[![Flutter Version](https://img.shields.io/badge/Flutter-3.24+-02569B?logo=flutter&logoColor=white)](https://flutter.dev)
[![Overlay Network](https://img.shields.io/badge/Overlay-Yggdrasil_0200::/7-green)](https://yggdrasil-network.github.io/)
[![Anti-DPI](https://img.shields.io/badge/Anti--DPI-uTLS_Chrome_%2B_ShadowTLS-purple)](#-обход-блокировок-и-анти-dpi-тспу)
[![E2E Encryption](https://img.shields.io/badge/E2E-X3DH_%2B_Double_Ratchet-success)](#-криптографическая-подсистема)

---

## ⚡ Ключевые возможности

### 🌐 Сеть Yggdrasil строго в Userspace (Без Root и TUN)
- **Zero-Privilege:** Работает без системного драйвера TUN/Wintun, не требует прав Администратора на Windows и слота VPN (`VpnService`) на Android.
- **Глобальная адресация `0200::/7`:** Каждое устройство получает постоянный, самоаутентифицируемый IPv6-адрес, детерминированно вычисленный из открытого ключа `Ed25519`.
- **LAN Multicast Discovery:** Автоматический поиск соседей в локальной сети Wi-Fi или при раздаче точки доступа со смартфона (`239.255.255.250:54328` и `[ff02::1]:54328`). Обмен сообщениями работает **без интернета**.

### 🛡️ Обход блокировок и Анти-DPI (ТСПУ в РФ)
- **TCP Record Fragmentation:** Разрезание первого пакета `TLS ClientHello` на границе 5-байтового заголовка TLS-записи сбивает сигнатурные анализаторы ТСПУ, отслеживающие доменное имя в SNI.
- **uTLS Chrome Fingerprint:** Эмуляция отпечатка `HelloChrome_Auto` и ALPN `h2, http/1.1` — соединения неотличимы от обычного браузера Google Chrome.
- **WSS через CDN:** Туннелирование трафика внутри веб-сокетов поверх Cloudflare или GCore (`wss://`). Блокировка невозможна без блокировки всего CDN.
- **ShadowTLS v3:** Двойное рукопожатие с маскировкой под разрешенные российские ресурсы (`yandex.ru`, `vk.com`) и HMAC-авторизацией.
- **Цензуроустойчивый Bootstrap:** Поиск живых узлов через **DNS-over-HTTPS** (Яндекс.DNS, Quad9) и **BitTorrent Mainline DHT** (BEP-44) с проверкой подписи `Ed25519`.

### 🔐 Криптография и Приватность
- **Идентичность без SIM-карт:** Аккаунт генерируется из мнемонической фразы **BIP-39** (12 или 24 слова).
- **Сквозное шифрование (E2EE):** **X3DH** (Extended Triple Diffie-Hellman) для старта офлайн-диалогов + **Double Ratchet** на базе `crypto/ecdh.X25519` с хранением пропущенных ключей (out-of-order delivery).
- **Скрытие отправителя (Sealed Sender):** Промежуточные узлы видят только слепой токен почтового ящика (`MailboxID`), не зная, кто отправил сообщение.
- **Групповые чаты (MLS / Sender Keys):** Симметричные цепочки отправки с ротацией эпох и прямой секретностью (Forward Secrecy).

### 📦 Гарантированная офлайн-доставка и Пуши
- **Слепые Store-and-Forward (S&F) релеи:** Любой десктопный узел на Windows или VPS может выступать релеем для мобильных устройств.
- **Защита от спама (Hashcash PoW):** Требование доказательства работы (16 нулевых бит) и квота 25 МБ на анонимный ящик.
- **Мульти-провайдер пробуждения:** Поддержка **UnifiedPush** (ntfy/Gotify), обезличенного шлюза **FCM/HMS** внутри Yggdrasil и адаптивного энергосберегающего фонового пуллинга.

### 💾 Zero-Knowledge Cloud Backup (Двойное Зеркалирование)
- **FastCDC чанкинг:** Контентно-определяемое нарезание базы данных и медиафайлов на блоки по 64 КБ – 1 МБ с дедупликацией.
- **Сжатие и шифрование:** Сжатие `zlib` и шифрование ключом `Argon2id` на стороне клиента.
- **Одновременное зеркалирование:** Автоматическая синхронизация в **Google Drive** (`appDataFolder`) и **Яндекс.Диск** (`app:/`) одновременно. Блокировка одного облака не нарушает доступ.

### 🎬 Мультимедиа, P2P-рой и Звонки
- **Swarm File Transfer:** Потоковое шифрование файлов по 256 КБ, мгновенный BlurHash-превью и автоматическая P2P-раздача скачанных блоков участниками групп и каналов.
- **Merkle-DAG вещательные каналы:** Подписанные посты объединяются в DAG и ретранслируются подписчиками по протоколу Gossipsub. Канал невозможно заблокировать даже при отключении автора.
- **Звонки без STUN/TURN:** Сигналинг внутри Double Ratchet, прямая передача голоса (Opus 48 кГц с защитой от потерь пакетов Inband FEC) и видео (VP9/AV1) напрямую по IPv6 оверлея `200::/7`.

---

## 🏛️ Архитектура Монорепозитория

```
ygggram/
├── proto/
│   └── ygggram.proto             # Protobuf v3 схемы (NodeConfig, SealedEnvelope, ChatMessage, CoreEvent)
├── core/                         # Pure Go 1.23+ ядро (Userspace C-Shared Library)
│   ├── cmd/libygggram/main.go    # C-ABI экспорты: InitCore, RegisterEventCallback, InvokeCommand
│   └── pkg/
│       ├── yggnode/              # OverlayPacketConn, виртуальные потоки, LAN Multicast Beacon
│       ├── transport/            # Anti-DPI dialer (uTLS, фрагментация, WSS), DoH, DHT BEP-44
│       ├── crypto/               # BIP-39, X3DH, Double Ratchet, Sealed Sender, Group MLS
│       ├── delivery/             # S&F релеи, Hashcash PoW, UnifiedPush, адаптивный пуллинг
│       ├── contacts/             # Kademlia DHT ContactCard, Private Set Intersection (PSI), Alfis DNS
│       ├── storage/              # Локальная AEAD БД, FTS полнотекстовый поиск, CRDT HLC
│       ├── backup/               # FastCDC чанкинг, провайдеры (GDrive с прокси, Yandex.Disk, LAN)
│       ├── media/                # P2P рой, 256КБ чанки, BlurHash
│       ├── channels/             # Signed Merkle-DAG каналы, Gossipsub ретрансляция
│       └── calls/                # E2E WebRTC звонки по IPv6 без STUN/TURN, Opus FEC
├── app/                          # Flutter 3.24+ клиент
│   ├── lib/
│   │   ├── main.dart             # Входная точка приложения с Riverpod ProviderScope
│   │   ├── core_bridge/          # Dart FFI мост через NativeCallable.listener
│   │   ├── platform/             # Windows трей, автозапуск HKCU, домашний мост
│   │   └── ui/                   # Адаптивный 3-колоночный Desktop и Android интерфейс
│   └── android/app/src/main/     # Kotlin: Android 14+ Foreground Service, WakeLock, RemoteInput
├── scripts/
│   ├── build_windows.ps1         # PowerShell сборка ygggram.dll (MinGW-w64)
│   └── build_android.sh          # Bash сборка libygggram.so (Android NDK Clang)
└── Makefile                      # Единая система сборки
```

---

## 🚀 Инструкция по сборке

### 1. Предварительные требования
- **Go 1.23+** с включенной поддержкой CGO.
- **Flutter SDK 3.24+**.
- **Протокол Protobuf компилятор (`protoc`)**.
- Для Windows: **MinGW-w64 GCC** (в PATH).
- Для Android: **Android NDK (r25c+)**.

---

### 2. Генерация Protobuf Stubs
```bash
make proto
```

---

### 3. Сборка для Windows (x64)

```powershell
# Через PowerShell
.\scripts\build_windows.ps1 -Arch x64 -Release

# Либо через Make
make windows
```
*Результат:* `build/windows/x64/ygggram.dll` и автоматическое копирование в `app/windows/runner/assets/`.

---

### 4. Сборка для Android (arm64-v8a / x86_64)

```bash
# Экспортируйте путь к NDK
export ANDROID_NDK_HOME="/path/to/android-ndk"

# Сборка libygggram.so
make android
```
*Результат:* `libygggram.so` компилируется с SONAME и размещается в `app/android/app/src/main/jniLibs/{arm64-v8a,x86_64}/`.

---

### 5. Запуск Flutter-приложения

```bash
cd app
flutter pub get

# Запуск на Windows Desktop
flutter run -d windows

# Запуск на подключенном Android устройстве
flutter run -d android
```

---

## 🔒 Безопасность и модель угроз

| Угроза | Решение в YggGram |
| :--- | :--- |
| **Сигнатурная блокировка DPI / ТСПУ** | TCP Record Fragmentation + подмена отпечатка uTLS под Chrome 120+. |
| **Блокировка IP-адресов серверов** | Децентрализованная сеть Yggdrasil: нет центральных серверов, каждый узел является маршрутизатором. |
| **Анализ графа связей на релеях** | Механизм **Sealed Sender**: релеи видят только слепой `MailboxID` и PoW, но не отправителя. |
| **Блокировка зарубежных облаков** | Двойное одновременное зеркалирование в Google Drive и отечественный Яндекс.Диск. |
| **Слушка и MITM-атаки** | Сквозное шифрование X3DH + Double Ratchet с прямой (PFS) и пост-компрометационной защитой. |
| **Цензура публичных каналов** | Signed Merkle-DAG: подписчики ретранслируют посты между собой, канал живет даже без автора. |
| **Блокировка VoIP-звонков** | Прямой обмен медиапотоками по IPv6 сети Yggdrasil без внешних серверов STUN/TURN. |

---

## 📄 Лицензия

Проект распространяется под открытой лицензией **Apache License 2.0**.
