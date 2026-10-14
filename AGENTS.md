# AGENTS.md — YggGram Architecture & Engineering Rules for Jules

## 1. Mission
Ты разрабатываешь **YggGram** — флагманский кроссплатформенный децентрализованный мессенджер для **Windows (x64)** и **Android (ARM64)** на базе оверлейной сети **Yggdrasil (`200::/7`)**.
Два главных приоритета проекта:
1. **Абсолютная живучесть связи в условиях жесткой цензуры (ТСПУ / DPI / «белые списки» в РФ).**
2. **Безупречный UX и дизайн уровня Telegram Desktop / Linear (Zero-Config старт, никаких сложных настроек для пользователя, мгновенный отклик).**

## 2. Monorepo Structure
- `proto/ygggram.proto` — единый Protobuf v3 контракт между Go-ядром и Flutter UI, а также схема сетевых P2P-пакетов.
- `core/` — бэкенд-ядро на **Pure Go 1.23+**:
  - `core/cmd/libygggram/` — C-ABI библиотека (`-buildmode=c-shared`), собираемая в `ygggram.dll` (Windows) и `libygggram.so` (Android).
  - `core/pkg/yggnode/` — встраиваемый узел `yggdrasil-go` строго в Userspace (поверх `ipv6rwc` + `quic-go`), БЕЗ системных TUN/Wintun драйверов и БЕЗ прав администратора/root.
  - `core/pkg/censorship/` — многослойный движок обхода ТСПУ/DPI:
    * Userspace TCP Desync & TLS Record Fragmentation (разрезание ClientHello/SNI).
    * `uTLS` (мимикрия отпечатка под Chrome 130 / YaBrowser).
    * `ShadowTLS v3 / Reality` (маскировка рукопожатия под домены из «белого списка» РФ: `st.ozone.ru`, `yastatic.net`, `vk.com`).
    * `XHTTP / WSS-CDN` (туннелирование поверх WebSocket/HTTP2 через CDN).
    * `TrafficMorpher` (статистический паддинг пакетов и джиттер для защиты голосовых звонков и чатов от блокировки по ML-профилю трафика).
    * `BootstrapEngine` (поиск пиров через DoH Яндекс/Quad9, BitTorrent Mainline DHT BEP-44, внутрироссийские узлы-мосты и «Дружеский мост» внутри инвайт-ссылок).
  - `core/pkg/crypto/` — BIP-39 -> SLIP-0010, протокол Signal (X3DH + Double Ratchet), групповой TreeKEM/MLS, конверты `SealedSender`.
  - `core/pkg/delivery/` — слепые Store-and-Forward (S&F) релеи с Hashcash PoW и мульти-пуш пробуждение (UnifiedPush/ntfy, проксированный FCM, адаптивный QUIC Keep-Alive).
  - `core/pkg/contacts/` — Kademlia DHT, поиск контактов через слепое пересечение OPRF (PSI), короткие имена `@username` (Alfis/DHT) и генератор `MagicInviteLink` (ссылка/QR-код, содержащий встроенные рабочие мосты отправителя).
  - `core/pkg/storage/` — локальная зашифрованная БД на чистом Go (`go.etcd.io/bbolt` + `XChaCha20-Poly1305` + FTS полнотекстовый индекс) и синхронизация устройств через HLC CRDT.
  - `core/pkg/backup/` — инкрементальный Zero-Knowledge бэкап (FastCDC + Zstd + `Argon2id` + `XChaCha20-Poly1305`) с поддержкой **Яндекс.Диска**, **Google Drive (`appDataFolder` напрямую или через Yggdrasil-прокси)**, **WebDAV** и **LAN P2P Sync**.
  - `core/pkg/media/`, `core/pkg/channels/`, `core/pkg/calls/` — роевая загрузка файлов по QUIC, каналы на базе Signed Merkle-DAG и P2P-звонки через QUIC Datagrams с защитой от ТСПУ.
- `app/` — клиентское приложение на **Flutter 3.24+ (Windows + Android)**, строго следующее правилам из `DESIGN_SYSTEM.md`.

## 3. Mandatory Engineering Rules for Jules
1. **NO PLACEHOLDERS OR STUBS:** Запрещено писать `// TODO`, `// implement here`, пустые методы или фейковые заглушки. Каждый файл пишется полностью.
2. **NO C-SQLITE DEPENDENCIES:** Для бесшовной кросс-компиляции под Windows и Android используй только Pure-Go базы данных (`go.etcd.io/bbolt`) с шифрованием `golang.org/x/crypto/chacha20poly1305`.
3. **MANDATORY VM VERIFICATION BEFORE PR:** Перед завершением любой задачи выполни в терминале VM:
   - `cd core && go vet ./...`
   - `cd core && go test -v -race ./...`
   - `cd core && GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -buildmode=c-shared -o build/ygggram.dll ./cmd/libygggram`
   - Если изменялся каталог `app/`: обязательно выполни `cd app && flutter analyze && flutter test`.
