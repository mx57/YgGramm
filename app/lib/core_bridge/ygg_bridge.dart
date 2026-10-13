// File: app/lib/core_bridge/ygg_bridge.dart
import 'dart:async';
import 'dart:ffi';
import 'dart:io';
import 'dart:typed_data';
import 'package:ffi/ffi.dart';

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

/// High-level event types emitted by Go Core into Dart Stream
enum CoreEventType {
  networkStatus,
  peersUpdated,
  antiDpiStatus,
  incomingMessage,
  backupProgress,
  incomingCall,
  unknown,
}

/// Generic container for incoming CoreEvent
class CoreEvent {
  final int eventId;
  final int timestampMs;
  final CoreEventType type;
  final Uint8List rawPayload;
  final Map<String, dynamic> parsedData;

  CoreEvent({
    required this.eventId,
    required this.timestampMs,
    required this.type,
    required this.rawPayload,
    required this.parsedData,
  });

  @override
  String toString() =>
      'CoreEvent(id: $eventId, type: $type, time: $timestampMs, data: $parsedData)';
}

/// Singleton Bridge interface that connects Flutter Dart to Go C-Shared libygggram
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
  final StreamController<CoreEvent> _eventStreamController =
      StreamController<CoreEvent>.broadcast();

  bool _isInitialized = false;

  /// Reactive Stream of all asynchronous events coming from the Yggdrasil core
  Stream<CoreEvent> get eventStream => _eventStreamController.stream;

  bool get isInitialized => _isInitialized;

  /// Loads the platform-specific dynamic library (.dll on Windows, .so on Android)
  void loadLibrary({String? customPath}) {
    if (_lib != null) return;

    if (customPath != null && customPath.isNotEmpty) {
      _lib = DynamicLibrary.open(customPath);
    } else if (Platform.isWindows) {
      // Look in standard Windows application directory or current process dir
      try {
        _lib = DynamicLibrary.open('ygggram.dll');
      } catch (_) {
        // Fallback to local executable folder
        final exeDir = File(Platform.resolvedExecutable).parent.path;
        final candidate = '$exeDir\\ygggram.dll';
        _lib = DynamicLibrary.open(candidate);
      }
    } else if (Platform.isAndroid) {
      // Android loads directly from jniLibs
      _lib = DynamicLibrary.open('libygggram.so');
    } else if (Platform.isLinux) {
      _lib = DynamicLibrary.open('libygggram.so');
    } else if (Platform.isMacOS) {
      _lib = DynamicLibrary.open('libygggram.dylib');
    } else {
      throw UnsupportedError('Platform ${Platform.operatingSystem} is not supported by YggCoreBridge');
    }

    // Bind C functions
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

    _setupNativeCallback();
  }

  /// Sets up thread-safe NativeCallable.listener for asynchronous Go-to-Dart events
  void _setupNativeCallback() {
    // NativeCallable.listener routes C callbacks directly to Dart Isolate event loop
    _nativeCallable = NativeCallable<CNativeEventCallback>.listener(
      _onNativeEventReceived,
    );

    _registerEventCallback(_nativeCallable!.nativeFunction);
  }

  /// Internal handler invoked when Go triggers a background event
  void _onNativeEventReceived(Pointer<Uint8> dataPtr, int length) {
    if (dataPtr == nullptr || length <= 0) return;

    final bytes = dataPtr.asTypedList(length);
    final copy = Uint8List.fromList(bytes);

    final event = _parseEvent(copy);
    _eventStreamController.add(event);
  }

  /// Decodes binary wire protocol event
  CoreEvent _parseEvent(Uint8List bytes) {
    if (bytes.length < 18) {
      return CoreEvent(
        eventId: 0,
        timestampMs: DateTime.now().millisecondsSinceEpoch,
        type: CoreEventType.unknown,
        rawPayload: bytes,
        parsedData: {'rawLength': bytes.length},
      );
    }

    final byteData = ByteData.sublistView(bytes);
    // Header magic: 'YGG\x02' = 0x59 0x47 0x47 0x02
    final eventId = byteData.getUint64(4, Endian.big);
    final timestamp = byteData.getUint64(12, Endian.big);
    final eventTypeByte = bytes[20];

    CoreEventType type = CoreEventType.unknown;
    final Map<String, dynamic> data = {};

    switch (eventTypeByte) {
      case 0x01: // NetworkStatusEvent
        type = CoreEventType.networkStatus;
        if (bytes.length >= 41) {
          data['online'] = bytes[21] == 0x01;
          data['activePeers'] = byteData.getUint32(22, Endian.big);
          data['rxBytes'] = byteData.getUint64(26, Endian.big);
          data['txBytes'] = byteData.getUint64(34, Endian.big);
          if (bytes.length >= 43) {
            final ipLen = byteData.getUint16(42, Endian.big);
            if (bytes.length >= 44 + ipLen) {
              data['ipv6'] = String.fromCharCodes(bytes.sublist(44, 44 + ipLen));
            }
          }
        }
        break;
      case 0x02:
        type = CoreEventType.peersUpdated;
        break;
      case 0x03:
        type = CoreEventType.antiDpiStatus;
        break;
      case 0x04:
        type = CoreEventType.incomingMessage;
        break;
      case 0x05:
        type = CoreEventType.backupProgress;
        break;
      case 0x06:
        type = CoreEventType.incomingCall;
        break;
    }

    return CoreEvent(
      eventId: eventId,
      timestampMs: timestamp,
      type: type,
      rawPayload: bytes,
      parsedData: data,
    );
  }

  /// Initializes the Yggdrasil Go Core with local path and configuration
  Future<bool> initialize({
    required String dataDir,
    String? mnemonicSeed,
    int antiDpiMode = 1,
    String antiDpiSni = 'www.cloudflare.com',
  }) async {
    loadLibrary();

    // Pack basic configuration
    final dataDirBytes = Uint8List.fromList(dataDir.codeUnits);
    final totalLen = 2 + dataDirBytes.length;
    final buffer = calloc<Uint8>(totalLen);

    // Store length as BigEndian Uint16
    buffer[0] = (dataDirBytes.length >> 8) & 0xFF;
    buffer[1] = dataDirBytes.length & 0xFF;

    for (int i = 0; i < dataDirBytes.length; i++) {
      buffer[2 + i] = dataDirBytes[i];
    }

    final result = _initCore(buffer, totalLen);
    calloc.free(buffer);

    _isInitialized = (result == 0);
    return _isInitialized;
  }

  /// Sends a synchronous or RPC command into Go Core and parses the binary response
  Uint8List invokeCommand({required int commandType, Uint8List? payload}) {
    if (!_isInitialized) {
      throw StateError('YggCoreBridge is not initialized. Call initialize() first.');
    }

    final pBytes = payload ?? Uint8List(0);
    final reqLen = 10 + pBytes.length;
    final reqPtr = calloc<Uint8>(reqLen);

    // Request ID (8 bytes) + Command Type (2 bytes)
    final reqId = DateTime.now().microsecondsSinceEpoch;
    final bd = ByteData(10);
    bd.setUint64(0, reqId, Endian.big);
    bd.setUint16(8, commandType, Endian.big);

    for (int i = 0; i < 10; i++) {
      reqPtr[i] = bd.getUint8(i);
    }
    for (int i = 0; i < pBytes.length; i++) {
      reqPtr[10 + i] = pBytes[i];
    }

    final respPtrPtr = calloc<Pointer<Uint8>>();
    final respLenPtr = calloc<Int32>();

    try {
      final code = _invokeCommand(reqPtr, reqLen, respPtrPtr, respLenPtr);
      if (code != 0) {
        throw Exception('InvokeCommand returned error code: $code');
      }

      final outPtr = respPtrPtr.value;
      final outLen = respLenPtr.value;

      if (outPtr == nullptr || outLen <= 0) {
        return Uint8List(0);
      }

      final resultBytes = Uint8List.fromList(outPtr.asTypedList(outLen));

      // Free C memory allocated by Go C.CBytes()
      _freeMemory(outPtr);

      return resultBytes;
    } finally {
      calloc.free(reqPtr);
      calloc.free(respPtrPtr);
      calloc.free(respLenPtr);
    }
  }

  /// Convenience method: Start Yggdrasil Node
  void startNode() {
    invokeCommand(commandType: 1);
  }

  /// Convenience method: Stop Yggdrasil Node
  void stopNode() {
    invokeCommand(commandType: 2);
  }

  /// Convenience method: Query Node Status
  Uint8List getNodeStatus() {
    return invokeCommand(commandType: 3);
  }

  /// Convenience method: Query Connected Peers
  Uint8List getPeers() {
    return invokeCommand(commandType: 4);
  }

  /// Convenience method: Send Message
  void sendMessage(Uint8List messagePayload) {
    invokeCommand(commandType: 5, payload: messagePayload);
  }

  /// Clean disposal of native listener callbacks and stream controllers
  void dispose() {
    _nativeCallable?.close();
    _nativeCallable = null;
    _eventStreamController.close();
  }
}
