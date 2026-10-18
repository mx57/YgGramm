// File: app/lib/core_bridge/ygg_bridge.dart
import 'dart:async';
import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';

import 'package:ffi/ffi.dart';
import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:ygggram/gen/yggproto/ygggram.pb.dart';

export 'package:ygggram/gen/yggproto/ygggram.pb.dart'
    show
        AntiDpiMode,
        AntiDpiStatusEvent,
        ChatMessage,
        ChatMessage_Content,
        ContactCard,
        ContactResolvedEvent,
        ConversationInfo,
        CoreEvent,
        CoreRequest,
        CoreResponse,
        FetchContactResp,
        NetworkStatusEvent,
        NodeConfig,
        NodeStatusResp,
        PeerInfo,
        PeerListResp,
        PeerListUpdatedEvent,
        PreKeyBundle,
        PeerTransportType,
        SendMessageReq,
        SendMessageResp,
        TextContent;

// Native function typedefs for C-ABI interoperability
typedef CInitCore = Int32 Function(Pointer<Uint8> configBytes, Int32 length);
typedef DartInitCore = int Function(Pointer<Uint8> configBytes, int length);

typedef CNativeEventCallback = Void Function(Pointer<Uint8> eventData, Int32 length);
typedef CDartEventCallback = void Function(Pointer<Uint8> eventData, int length);

typedef CRegisterEventCallback = Void Function(Pointer<NativeFunction<CNativeEventCallback>> cb);
typedef DartRegisterEventCallback = void Function(Pointer<NativeFunction<CNativeEventCallback>> cb);

typedef CInvokeCommand = Int32 Function(
  Pointer<Uint8> reqBytes,
  Int32 length,
  Pointer<Pointer<Uint8>> respPtr,
  Pointer<Int32> respLen,
);
typedef DartInvokeCommand = int Function(
  Pointer<Uint8> reqBytes,
  int length,
  Pointer<Pointer<Uint8>> respPtr,
  Pointer<Int32> respLen,
);

typedef CFreeMemory = Void Function(Pointer<Uint8> ptr);
typedef DartFreeMemory = void Function(Pointer<Uint8> ptr);

/// Типы событий, которые ядро присылает в поток [YggCoreBridge.eventStream].
enum CoreEventType {
  networkStatus,
  peersUpdated,
  antiDpiStatus,
  incomingMessage,
  messageStatus,
  backupProgress,
  incomingCall,
  contactUpdated,
  unknown,
}

/// Обёртка над protobuf-событием ядра для UI-слоя.
class CoreEventEnvelope {
  final CoreEventType type;
  final CoreEvent event;

  const CoreEventEnvelope(this.type, this.event);

  NetworkStatusEvent? get networkStatus =>
      event.whichEvent() == CoreEvent_Event.networkStatus ? event.networkStatus : null;

  IncomingMessageEvent? get incomingMessage =>
      event.whichEvent() == CoreEvent_Event.incomingMessage ? event.incomingMessage : null;

  /// Судьба исходящего сообщения: «пакет ушёл в оверлей» и «получатель
  /// расшифровал» — разные этапы, UI показывает их раздельно.
  MessageStatusEvent? get messageStatus =>
      event.whichEvent() == CoreEvent_Event.messageStatus ? event.messageStatus : null;

  /// Итог fetch-контакта: бандл дошёл и карта заводятся, либо ответ не сошёлся.
  /// Ждать его внутри команды нельзя, поэтому приходит он событием.
  ContactResolvedEvent? get contactResolved =>
      event.whichEvent() == CoreEvent_Event.contactUpdated ? event.contactUpdated : null;

  /// Вершина оверлея: ядро объявляет её само, когда список пиров меняется по
  /// составу (линок появился, упал, пропал), поэтому UI не обязан опрашивать
  /// GetPeers, чтобы увидеть «пиров: 0» после обрыва.
  PeerListUpdatedEvent? get peersUpdated =>
      event.whichEvent() == CoreEvent_Event.peersUpdated ? event.peersUpdated : null;

  /// Снимок исходящих ссылок: ядро объявляет его по изменению вершины линков,
  /// как и peers_updated. Опросить его командой нельзя — в контракте нет
  /// GetAntiDpiStatus, — поэтому «объявления ещё не было» означает только то,
  /// что оно ещё не пришло, а не «ссылок нет».
  AntiDpiStatusEvent? get antiDpi =>
      event.whichEvent() == CoreEvent_Event.antiDpiStatus ? event.antiDpiStatus : null;

  @override
  String toString() => 'CoreEventEnvelope(${event.whichEvent()}, id=${event.eventId})';
}

/// Ошибка со стороны Go-ядра: неверный код возврата C-ABI или success=false.
class YggCoreException implements Exception {
  final String message;
  const YggCoreException(this.message);

  @override
  String toString() => 'YggCoreException: $message';
}

/// Мост Dart <-> Go C-Shared libygggram.
///
/// Единственный формат на границе — protobuf (`CoreRequest`/`CoreResponse`),
/// сгенерированный из `proto/ygggram.proto`. Раньше здесь парсился самодельный
/// BigEndian-заголовок, а ядро на него отвечало заглушками.
class YggCoreBridge {
  static final YggCoreBridge _instance = YggCoreBridge._internal();
  factory YggCoreBridge() => _instance;
  YggCoreBridge._internal();

  DynamicLibrary? _lib;
  late DartInitCore _initCore;
  late DartRegisterEventCallback _registerEventCallback;
  late DartInvokeCommand _invokeCommand;
  late DartFreeMemory _freeMemory;

  NativeCallable<CNativeEventCallback>? _nativeCallable;
  final StreamController<CoreEventEnvelope> _events =
      StreamController<CoreEventEnvelope>.broadcast();

  bool _isInitialized = false;
  $fixnum.Int64 _requestCounter = $fixnum.Int64.ZERO;

  Stream<CoreEventEnvelope> get eventStream => _events.stream;

  bool get isInitialized => _isInitialized;

  /// Открывает платформу-специфичную библиотеку и привязывает C-функции.
  void loadLibrary({String? customPath}) {
    if (_lib != null) return;

    if (customPath != null && customPath.isNotEmpty) {
      _lib = DynamicLibrary.open(customPath);
    } else if (Platform.isWindows) {
      try {
        _lib = DynamicLibrary.open('ygggram.dll');
      } catch (_) {
        final exeDir = File(Platform.resolvedExecutable).parent.path;
        _lib = DynamicLibrary.open('$exeDir\\ygggram.dll');
      }
    } else if (Platform.isAndroid) {
      _lib = DynamicLibrary.open('libygggram.so');
    } else if (Platform.isLinux) {
      _lib = DynamicLibrary.open('libygggram.so');
    } else if (Platform.isMacOS) {
      _lib = DynamicLibrary.open('libygggram.dylib');
    } else {
      throw UnsupportedError('YggCoreBridge не поддерживает ${Platform.operatingSystem}');
    }

    _initCore = _lib!
        .lookup<NativeFunction<CInitCore>>('InitCore')
        .asFunction<DartInitCore>();
    _registerEventCallback = _lib!
        .lookup<NativeFunction<CRegisterEventCallback>>('RegisterEventCallback')
        .asFunction<DartRegisterEventCallback>();
    _invokeCommand = _lib!
        .lookup<NativeFunction<CInvokeCommand>>('InvokeCommand')
        .asFunction<DartInvokeCommand>();
    _freeMemory = _lib!
        .lookup<NativeFunction<CFreeMemory>>('FreeMemory')
        .asFunction<DartFreeMemory>();

    _nativeCallable = NativeCallable<CNativeEventCallback>.listener(_onNativeEvent);
    _registerEventCallback(_nativeCallable!.nativeFunction);
  }

  /// Готовит ядро: создаёт каталоги и запускает поток событий.
  /// Идентичность при этом не создаётся — это делает [startNode].
  void initialize({required String dataDir, required String cacheDir}) {
    loadLibrary();
    final config = NodeConfig()
      ..dataDir = dataDir
      ..cacheDir = cacheDir;

    final code = _callInitCore(config.writeToBuffer());
    if (code != 0) {
      throw YggCoreException('InitCore вернул код $code');
    }
    _isInitialized = true;
  }

  int _callInitCore(List<int> payload) {
    final ptr = calloc<Uint8>(payload.length);
    try {
      ptr.asTypedList(payload.length).setAll(0, payload);
      return _initCore(ptr, payload.length);
    } finally {
      calloc.free(ptr);
    }
  }

  void _onNativeEvent(Pointer<Uint8> dataPtr, int length) {
    if (dataPtr == nullptr || length <= 0) return;
    final bytes = Uint8List.fromList(dataPtr.asTypedList(length));
    final event = CoreEvent.fromBuffer(bytes);
    _events.add(CoreEventEnvelope(_eventTypeOf(event), event));
  }

  static CoreEventType _eventTypeOf(CoreEvent event) {
    switch (event.whichEvent()) {
      case CoreEvent_Event.networkStatus:
        return CoreEventType.networkStatus;
      case CoreEvent_Event.peersUpdated:
        return CoreEventType.peersUpdated;
      case CoreEvent_Event.antiDpiStatus:
        return CoreEventType.antiDpiStatus;
      case CoreEvent_Event.incomingMessage:
        return CoreEventType.incomingMessage;
      case CoreEvent_Event.messageStatus:
        return CoreEventType.messageStatus;
      case CoreEvent_Event.backupProgress:
        return CoreEventType.backupProgress;
      case CoreEvent_Event.incomingCall:
        return CoreEventType.incomingCall;
      case CoreEvent_Event.contactUpdated:
        return CoreEventType.contactUpdated;
      case CoreEvent_Event.notSet:
        return CoreEventType.unknown;
    }
  }

  /// Единая точка обмена с ядром: protobuf внутри, protobuf снаружи.
  CoreResponse request(CoreRequest request) {
    if (!_isInitialized) {
      throw const YggCoreException('Ядро не инициализировано: сначала initialize()');
    }
    final payload = request.writeToBuffer();
    final reqPtr = calloc<Uint8>(payload.length);
    final respPtrPtr = calloc<Pointer<Uint8>>();
    final respLenPtr = calloc<Int32>();
    try {
      reqPtr.asTypedList(payload.length).setAll(0, payload);
      final code = _invokeCommand(reqPtr, payload.length, respPtrPtr, respLenPtr);
      if (code != 0) {
        throw YggCoreException('InvokeCommand вернул код $code');
      }
      final outPtr = respPtrPtr.value;
      final outLen = respLenPtr.value;
      if (outPtr == nullptr || outLen <= 0) {
        throw const YggCoreException('Ядро вернуло пустой ответ');
      }
      final response = CoreResponse.fromBuffer(outPtr.asTypedList(outLen));
      _freeMemory(outPtr);
      return response;
    } finally {
      calloc.free(reqPtr);
      calloc.free(respPtrPtr);
      calloc.free(respLenPtr);
    }
  }

  /// Исполняет запрос и бросает исключение, если ядро ответило ошибкой.
  CoreResponse _ask(CoreRequest Function($fixnum.Int64 id) build) {
    final response = request(build(++_requestCounter));
    if (!response.success) {
      throw YggCoreException(response.errorMessage.isEmpty
          ? 'ядро отклонило запрос'
          : response.errorMessage);
    }
    return response;
  }

  /// Запускает оверлей-ноду на мнемонике. Пустая мнемоника — ошибка, а не
  /// молчаливая генерация ключа «на лету».
  ///
  /// [listenAddress] обязателен по смыслу: без него ядро поднимает только
  /// исходящие соединения, и никто не сможет придти к нам. Если порт занят,
  /// yggdrasil пишет в лог и остаётся исходящей — это не роняет запуск.
  NodeStatusResp startNode({
    required String mnemonicSeed,
    required String dataDir,
    required String cacheDir,
    String listenAddress = '0.0.0.0',
    int listenPort = 54321,
    List<String> bootstrapPeers = const [],
    bool enableLanMulticast = true,
    AntiDpiMode antiDpiMode = AntiDpiMode.ANTI_DPI_UTLS_CHROME,
  }) {
    final config = NodeConfig()
      ..dataDir = dataDir
      ..cacheDir = cacheDir
      ..mnemonicSeed = mnemonicSeed
      ..antiDpiMode = antiDpiMode
      ..listenAddress = listenAddress
      ..listenPort = listenPort
      ..enableLanMulticast = enableLanMulticast
      ..bootstrapPeers.addAll(bootstrapPeers);
    final response = _ask((id) => CoreRequest(
        requestId: id, startNode: StartNodeReq(config: config)));
    return response.nodeStatus;
  }

  void stopNode() => _ask((id) => CoreRequest(requestId: id, stopNode: StopNodeReq()));

  NodeStatusResp nodeStatus() =>
      _ask((id) => CoreRequest(requestId: id, getNodeStatus: GetNodeStatusReq())).nodeStatus;

  List<PeerInfo> peers() =>
      _ask((id) => CoreRequest(requestId: id, getPeers: GetPeersReq())).peerList.peers;

  List<PeerInfo> connectPeer(String uri) => _ask((id) => CoreRequest(
        requestId: id,
        connectPeer: ConnectPeerReq(peerUri: uri),
      )).peerList.peers;

  PreKeyBundle preKeyBundle() =>
      _ask((id) => CoreRequest(requestId: id, getPrekeyBundle: GetPreKeyBundleReq()))
          .prekeyBundle;

  void addContact(ContactCard card) =>
      _ask((id) => CoreRequest(requestId: id, addContact: AddContactReq(contact: card)));

  /// Завести контакт по одному лишь ed25519-ключу узла (32 байта): ядро шлёт
  /// bundle_req и отвечает событием contact_updated, поэтому здесь возвращается
  /// только факт отправки запроса.
  FetchContactResp fetchContact(List<int> nodeEd25519Pubkey, {String nickname = ''}) => _ask(
          (id) => CoreRequest(
              requestId: id,
              fetchContact: FetchContactReq(
                  nodeEd25519Pubkey: nodeEd25519Pubkey, nickname: nickname)))
      .fetchContact;

  SendMessageResp sendMessage(SendMessageReq message) =>
      _ask((id) => CoreRequest(requestId: id, sendMessage: message)).sendMessage;

  /// Новая 12-словная фраза от ядра: entropy и словарь живут на стороне ядра.
  String generateIdentity() =>
      _ask((id) => CoreRequest(requestId: id, generateIdentity: GenerateIdentityReq()))
          .identity
          .mnemonic;

  List<ConversationInfo> conversations({int limit = 0}) => _ask(
          (id) => CoreRequest(
              requestId: id,
              listConversations: ListConversationsReq(limit: limit)))
      .conversations
      .conversations;

  List<ChatMessage> messages(String conversationId, {int limit = 50}) => _ask(
          (id) => CoreRequest(
              requestId: id,
              getMessages: GetMessagesReq(
                  conversationId: conversationId, limit: limit)))
      .messages
      .messages;

  List<ContactCard> contacts() =>
      _ask((id) => CoreRequest(requestId: id, listContacts: ListContactsReq()))
          .contacts
          .contacts;

  void dispose() {
    _nativeCallable?.close();
    _nativeCallable = null;
    _events.close();
    _isInitialized = false;
  }
}
