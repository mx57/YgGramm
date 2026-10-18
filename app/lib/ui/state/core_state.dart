// File: app/lib/ui/state/core_state.dart
import 'dart:async';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:ygggram/core_bridge/ygg_bridge.dart';
import 'package:ygggram/ui/state/app_state.dart';

/// Что сейчас происходит с ядром.
enum CorePhase {
  /// Ещё не пытались подняться.
  idle,

  /// Идентичности на диске нет — нужен onboarding.
  needsIdentity,

  /// StartNode в полёте.
  starting,

  /// Нода запущена, IPv6 и пиры приходят из ядра.
  online,

  /// Ядро ответило ошибкой; текст лежит в [CoreSnapshot.error].
  failed,
}

class CoreSnapshot {
  final CorePhase phase;
  final String? ipv6;

  /// Наш ed25519-ключ узла в hex — его отдают собеседнику вне сети, потому что
  /// по нему ядро и заводит контакт через bundle_req.
  final String? nodeKeyHex;

  final List<PeerInfo> peers;

  /// Последнее объявление `anti_dpi_status`: что ядро говорит про наш исходящий
  /// путь. Команды «спросить» в контракте нет, поэтому null — это «ещё не
  /// приходило», а не «ссылок нет» или «всё чисто».
  final AntiDpiStatusEvent? antiDpi;
  final List<ContactCard> contacts;
  final List<ConversationInfo> conversations;
  final List<ChatMessage> messages;

  /// Fetch-запросы бандла, на которые ещё нет ответа: адрес узла с подписью
  /// пользователя. Ответ приходит событием, поэтому «в полёте» — отдельное
  /// состояние UI, а не молчание.
  final List<String> pendingContacts;

  /// Последний итог сетевой загрузки контакта: успех или названный отказ.
  final String? contactNote;
  final String? selectedConversationId;
  final String? error;

  const CoreSnapshot({
    this.phase = CorePhase.idle,
    this.ipv6,
    this.nodeKeyHex,
    this.peers = const [],
    this.antiDpi,
    this.contacts = const [],
    this.conversations = const [],
    this.messages = const [],
    this.pendingContacts = const [],
    this.contactNote,
    this.selectedConversationId,
    this.error,
  });

  bool get canSend => contacts.isNotEmpty && selectedConversationId != null;

  CoreSnapshot copyWith({
    CorePhase? phase,
    Object? ipv6 = _keep,
    Object? nodeKeyHex = _keep,
    List<PeerInfo>? peers,
    Object? antiDpi = _keep,
    List<ContactCard>? contacts,
    List<ConversationInfo>? conversations,
    List<ChatMessage>? messages,
    List<String>? pendingContacts,
    Object? contactNote = _keep,
    Object? selectedConversationId = _keep,
    Object? error = _keep,
  }) {
    return CoreSnapshot(
      phase: phase ?? this.phase,
      ipv6: ipv6 == _keep ? this.ipv6 : ipv6 as String?,
      nodeKeyHex: nodeKeyHex == _keep ? this.nodeKeyHex : nodeKeyHex as String?,
      peers: peers ?? this.peers,
      antiDpi: antiDpi == _keep ? this.antiDpi : antiDpi as AntiDpiStatusEvent?,
      contacts: contacts ?? this.contacts,
      conversations: conversations ?? this.conversations,
      messages: messages ?? this.messages,
      pendingContacts: pendingContacts ?? this.pendingContacts,
      contactNote: contactNote == _keep ? this.contactNote : contactNote as String?,
      selectedConversationId: selectedConversationId == _keep
          ? this.selectedConversationId
          : selectedConversationId as String?,
      error: error == _keep ? this.error : error as String?,
    );
  }
}

/// Разделяет «параметр не передавали» и «передан null» в [CoreSnapshot.copyWith].
const Object _keep = Object();

/// Как событие ядра меняет снимок, не перечитывая зашифрованную базу.
///
/// Возвращает null, если снимок остаётся прежним. Вынесено из
/// [CoreController._onEvent] потому, что именно здесь живёт правило, которое
/// раньше ломало онбординг: FFI пульсирует состоянием ноды каждые несколько
/// секунд с самой инициализации, и на свежей установке, где ноды ещё нет потому
/// что нет идентичности, «ядро не запущено» — норма, а не отказ.
CoreSnapshot? applyCoreEvent(CoreSnapshot state, CoreEventEnvelope envelope) {
  final updated = envelope.peersUpdated;
  if (updated != null) {
    return state.copyWith(peers: List<PeerInfo>.unmodifiable(updated.peers));
  }
  // Снимок обхода — только то, что объявило ядро: фаза, адрес и остальное
  // остаются своими, потому что событие описывает линки, а не жизнь ноды.
  final dpi = envelope.antiDpi;
  if (dpi != null) return state.copyWith(antiDpi: dpi);
  final status = envelope.networkStatus;
  if (status == null || _isPulseBeforeStart(state.phase, status.online)) return null;
  return state.copyWith(
    phase: status.online ? CorePhase.online : CorePhase.failed,
    ipv6: status.currentIpv6.isEmpty ? null : status.currentIpv6,
    error: status.online
        ? null
        : (status.statusNote.isEmpty ? 'ядро не в сети' : status.statusNote),
  );
}

/// Пульс «не в сети» до старта ноды не меняет ничего: фаза idle, needsIdentity
/// или starting означает, что мы запускаться ещё не пробовали (или пробуем
/// прямо сейчас), а не что ядро умерло. Иначе поверх шага ввода фразы вырос бы
/// баннер сбоя, а IPv6 и так обнулился бы на пустом месте.
bool _isPulseBeforeStart(CorePhase phase, bool online) =>
    !online &&
    (phase == CorePhase.idle ||
        phase == CorePhase.needsIdentity ||
        phase == CorePhase.starting);

final yggBridgeProvider = Provider<YggCoreBridge>((ref) => YggCoreBridge());

final coreProvider =
    StateNotifierProvider<CoreController, CoreSnapshot>((ref) => CoreController(ref));

/// Единственный владелец состояния сети и данных чата: всё читается через
/// protobuf-контракт из ядра, демо-констант в UI больше нет.
class CoreController extends StateNotifier<CoreSnapshot> {
  CoreController(this._ref) : super(const CoreSnapshot());

  final Ref _ref;
  StreamSubscription<CoreEventEnvelope>? _events;
  Directory? _root;

  /// request_id ядра → подпись незакрытого fetch-запроса. Ключ — тот же
  /// request_id, что вернул FetchContactResp и что несёт событие-итог.
  final Map<$fixnum.Int64, String> _pendingFetches = {};

  YggCoreBridge get _bridge => _ref.read(yggBridgeProvider);

  Future<void> bootstrap() async {
    try {
      final appDir = await getApplicationDocumentsDirectory();
      _root = Directory(p.join(appDir.path, 'ygggram'));
      final dataDir = p.join(_root!.path, 'data');
      final cacheDir = p.join(_root!.path, 'cache');

      _bridge.initialize(dataDir: dataDir, cacheDir: cacheDir);
      _events ??= _bridge.eventStream.listen(_onEvent);

      final mnemonic = await _readStoredMnemonic();
      if (mnemonic == null) {
        state = state.copyWith(phase: CorePhase.needsIdentity);
        return;
      }
      await _startNode(mnemonic);
    } on YggCoreException catch (e) {
      state = state.copyWith(phase: CorePhase.failed, error: e.message);
    } on Object catch (e) {
      // path_provider отсутствует (например, в widget-тесте) — честная
      // ошибка вместо молчаливого «всё работает».
      state = state.copyWith(phase: CorePhase.failed, error: 'Ядро недоступно: $e');
    }
  }

  /// Возвращает новую фразу: показать её пользователю обязан onboarding.
  Future<String> createIdentity() async {
    final mnemonic = _bridge.generateIdentity();
    await _storeMnemonic(mnemonic);
    await _startNode(mnemonic);
    return mnemonic;
  }

  Future<void> restoreIdentity(String mnemonic) => _startNode(mnemonic.trim());

  Future<void> _startNode(String mnemonic) async {
    state = state.copyWith(phase: CorePhase.starting);
    // Незакрытые fetch-запросы прошлого запуска ядра закрывать бессмысленно:
    // нумерация request_id начинается заново, и ответ на старый id уже не найдёт
    // ни ожидание, ни контакт.
    _pendingFetches.clear();
    // Снимок обхода прошлого запуска тем более не наш: ядро при startNode обнуляет
    // сигнатуру вершины и объявляет её заново — пусть даже список ссылок пуст, —
    // так что старая картина просто мешала бы новой.
    state = state.copyWith(
        pendingContacts: const [], contactNote: null, antiDpi: null);
    final root = _root!;
    // Переключатель «обход блокировок» выбирает транспорт ссылок встроенной
    // yggdrasil: tls (рукопожатие похоже на обычный HTTPS) либо чистый tcp.
    final antiDpi = _ref.read(appSettingsProvider).antiDpiOnNextStart
        ? AntiDpiMode.ANTI_DPI_UTLS_CHROME
        : AntiDpiMode.ANTI_DPI_DISABLED;
    final status = _bridge.startNode(
      mnemonicSeed: mnemonic,
      dataDir: p.join(root.path, 'data'),
      cacheDir: p.join(root.path, 'cache'),
      antiDpiMode: antiDpi,
    );
    state = state.copyWith(
      phase: CorePhase.online,
      ipv6: status.myIpv6,
      nodeKeyHex: status.myEd25519Pubkey.isEmpty
          ? null
          : _hex(status.myEd25519Pubkey),
      error: null,
    );
    await refresh();
  }

  Future<void> refresh() async {
    if (state.phase != CorePhase.online) return;
    final selected = state.selectedConversationId ??
        (await _firstConversationId());
    state = state.copyWith(
      peers: _bridge.peers(),
      contacts: _bridge.contacts(),
      conversations: _bridge.conversations(),
      selectedConversationId: selected,
      messages: selected == null ? const [] : _bridge.messages(selected),
    );
  }

  Future<String?> _firstConversationId() async {
    final list = _bridge.conversations(limit: 1);
    return list.isEmpty ? null : list.first.conversationId;
  }

  void selectConversation(String conversationId) {
    state = state.copyWith(
      selectedConversationId: conversationId,
      messages: _bridge.messages(conversationId),
    );
  }

  /// Отправка возможна только по уже добавленному контакту с PreKeyBundle:
  /// без бандла X3DH не построить, и ядро это явно отвергает.
  String? sendText(String text) {
    final trimmed = text.trim();
    final conversationId = state.selectedConversationId;
    if (trimmed.isEmpty || conversationId == null) return null;

    final contact = _contactFor(conversationId);
    if (contact == null) {
      return 'У диалога нет контакта с PreKeyBundle — сообщение некуда шифровать';
    }

    final message = ChatMessage()
      ..conversationId = conversationId
      ..senderNodeIpv6 = state.ipv6 ?? ''
      ..timestampMs = _nowMs()
      ..isOutbound = true
      ..text = (TextContent()..body = trimmed);

    final resp = _bridge.sendMessage(SendMessageReq(
      destinationIpv6: contact.nodeIpv6,
      destinationX25519Pubkey: contact.x25519Pubkey,
      message: message,
    ));
    if (resp.writtenToOverlay) return null;
    if (resp.queuedInOutbox) {
      return 'Маршрута нет: конверт лежит в зашифрованной исходящей очереди, '
          'ядро перешлёт его, когда появится пир';
    }
    return 'Сообщение не ушло: ядро не поставило его ни в оверлей, ни в очередь';
  }

  List<PeerInfo> connectPeer(String uri) => _bridge.connectPeer(uri);

  /// Завести контакт по ed25519-ключу узла, который собеседник передал вне
  /// сети. Возвращает текст проблемы (или null, если запрос ушёл в оверлей):
  /// бандл отвечает не здесь, а событием contact_updated.
  String? fetchContact(String nodeKeyHex, {String nickname = ''}) {
    final key = _parseNodeKey(nodeKeyHex);
    if (key == null) {
      return 'Нужен ed25519-ключ узла: 64 hex-символа (32 байта)';
    }
    if (state.nodeKeyHex != null && _hex(key) == state.nodeKeyHex) {
      return 'Этот ключ принадлежит самой ноде — добавлять себя в контакты нечего';
    }

    final FetchContactResp resp;
    try {
      resp = _bridge.fetchContact(key, nickname: nickname.trim());
    } on YggCoreException catch (e) {
      // Здесь отказ ещё не про сеть: маршрута нет, ключ не разбирается, ядро
      // не запущено. Пользователь обязан увидеть причину, а не крутилку.
      return e.message;
    }
    if (!resp.requested) {
      return 'Ядро не поставило bundle_req в оверлей: запрос не ушёл';
    }
    final label = nickname.trim().isEmpty ? resp.nodeIpv6 : '${nickname.trim()} • ${resp.nodeIpv6}';
    _pendingFetches[resp.requestId] = label;
    state = state.copyWith(
      pendingContacts: List<String>.unmodifiable(_pendingFetches.values),
      contactNote: 'Запрос бандла ушёл узлу ${resp.nodeIpv6}, ждём ответа...',
    );
    return null;
  }

  static const _nodeKeyBytes = 32;

  static List<int>? _parseNodeKey(String input) {
    final cleaned = input.trim().replaceAll(RegExp(r'[\s:;-]'), '').toLowerCase();
    if (cleaned.length != _nodeKeyBytes * 2) return null;
    final out = Uint8List(_nodeKeyBytes);
    for (var i = 0; i < _nodeKeyBytes; i++) {
      final byte = int.tryParse(cleaned.substring(i * 2, i * 2 + 2), radix: 16);
      if (byte == null) return null;
      out[i] = byte;
    }
    return out;
  }

  ContactCard? _contactFor(String conversationId) {
    for (final contact in state.contacts) {
      if (contact.nodeIpv6 == conversationId) return contact;
      // Принятое сообщение ядро кладёт в диалог «1:<hex identity key
      // отправителя>» — имя выбрано на этой стороне, и ответить в такой беседе
      // можно, только найдя карту по тому же ключу.
      if (conversationId.startsWith(_inboundConversationPrefix) &&
          _hex(contact.x25519Pubkey) ==
              conversationId.substring(_inboundConversationPrefix.length)) {
        return contact;
      }
    }
    return null;
  }

  static const _inboundConversationPrefix = '1:';

  static String _hex(List<int> bytes) {
    final out = StringBuffer();
    for (final b in bytes) {
      out.write(b.toRadixString(16).padLeft(2, '0'));
    }
    return out.toString();
  }

  void _onEvent(CoreEventEnvelope envelope) {
    if (envelope.incomingMessage != null || envelope.messageStatus != null) {
      // Сообщение и подтверждение доставки уже лежат в зашифрованной базе
      // ядра: UI перечитывает списки, а не додумывает статус из события.
      unawaited(refresh());
      return;
    }
    final resolved = envelope.contactResolved;
    if (resolved != null) {
      final label = _pendingFetches.remove(resolved.requestId);
      // Итог запроса, которого мы уже не ждём (рестарт ядра, дубликат ответа):
      // выдумывать контакт по нему нельзя.
      if (label == null) return;
      final card = resolved.hasContact() ? resolved.contact : null;
      final note = resolved.error.isNotEmpty
          ? resolved.error
          : 'Контакт ${card?.nickname.isNotEmpty == true ? card!.nickname : (card?.nodeIpv6 ?? label)} '
              'получил бандл и заведён';
      state = state.copyWith(
        pendingContacts: List<String>.unmodifiable(_pendingFetches.values),
        contactNote: note,
      );
      // Карта лежит в базе ядра: список контактов и диалогов перечитывается,
      // а не достраивается из события.
      unawaited(refresh());
      return;
    }
    // Остальные события меняют снимок напрямую: вершина пиров приходит из ядра
    // списком, а «живо ли ядро» — флагом. Ни то, ни другое не лежит в базе,
    // поэтому перечитывать её незачем.
    final next = applyCoreEvent(state, envelope);
    if (next != null) state = next;
  }

  Future<String?> _readStoredMnemonic() async {
    final file = _identityFile;
    if (!await file.exists()) return null;
    final contents = (await file.readAsString()).trim();
    return contents.isEmpty ? null : contents;
  }

  Future<void> _storeMnemonic(String mnemonic) async {
    await _root!.create(recursive: true);
    final file = _identityFile;
    await file.writeAsString(mnemonic, flush: true);
    if (!Platform.isWindows) {
      // dart:io не умеет chmod; на Android/macOS/Linux просим у системы 0600.
      await Process.run('chmod', ['600', file.path]);
    }
  }

  File get _identityFile => File(p.join(_root!.path, 'identity.seed'));

  static $fixnum.Int64 _nowMs() => $fixnum.Int64(DateTime.now().millisecondsSinceEpoch);

  @override
  void dispose() {
    _events?.cancel();
    super.dispose();
  }
}
