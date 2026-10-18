// File: app/test/widget_test.dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:ygggram/main.dart';
import 'package:ygggram/core_bridge/ygg_bridge.dart';
import 'package:ygggram/gen/yggproto/ygggram.pb.dart' show BackupProgressEvent;
import 'package:ygggram/ui/adaptive_shell.dart';
import 'package:ygggram/ui/screens/anti_dpi_radar_screen.dart';
import 'package:ygggram/ui/screens/backup_settings_screen.dart';
import 'package:ygggram/ui/state/core_state.dart';

/// 64 hex-символа — годный ed25519-ключ узла с точки зрения формата.
const _validKey = 'aa00bb11cc22dd33ee44ff5566778899aa00bb11cc22dd33ee44ff5566778899';

void main() {
  testWidgets('каркас приложения поднимается без исключений', (tester) async {
    await tester.pumpWidget(const ProviderScope(child: YggGramApp()));
    await tester.pump();

    expect(tester.takeException(), isNull);
    expect(find.byType(AdaptiveShell), findsOneWidget);
  });

  test('ключ узла проверяется до обращения к ядру', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    final controller = container.read(coreProvider.notifier);

    // Кривой ввод не должен доходить до FFI: ядро вызывается только по годным
    // 32 байтам.
    for (final bad in ['1bf2', 'zz' * 32, _validKey.substring(1)]) {
      expect(controller.fetchContact(bad),
          'Нужен ed25519-ключ узла: 64 hex-символа (32 байта)',
          reason: 'формат ключа $bad пропущен дальше валидатора');
    }

    // Годный формат доходит до ядра — и там честно получает «не инициализировано»,
    // а не молчаливый отказ валидатора.
    expect(controller.fetchContact(_validKey), contains('Ядро не инициализировано'));
    expect(controller.state.pendingContacts, isEmpty);
  });

  test('разделители в hex-ключе не ломают разбор', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    final spaced = <String>[
      for (var i = 0; i < _validKey.length; i += 2) _validKey.substring(i, i + 2),
    ].join(':');
    final problem = container.read(coreProvider.notifier).fetchContact(spaced);
    expect(problem, isNot('Нужен ed25519-ключ узла: 64 hex-символа (32 байта)'));
  });

  testWidgets('диалог контакта передаёт ключ и закрывается при успехе',
      (tester) async {
    String? sentKey;
    String? sentName;
    await tester.pumpWidget(MaterialApp(
      home: Scaffold(
        body: Builder(
          builder: (context) => ElevatedButton(
            onPressed: () => showDialog<void>(
              context: context,
              builder: (_) => AddContactDialog(
                nodeKeyHex: _validKey,
                onSubmit: (key, nickname) {
                  sentKey = key;
                  sentName = nickname;
                  return null;
                },
              ),
            ),
            child: const Text('открыть'),
          ),
        ),
      ),
    ));

    await tester.tap(find.text('открыть'));
    await tester.pumpAndSettle();
    expect(find.text(_validKey), findsOneWidget);

    await tester.enterText(find.byType(TextField).at(0), _validKey);
    await tester.enterText(find.byType(TextField).at(1), 'Саша');
    await tester.tap(find.text('Запросить бандл'));
    await tester.pumpAndSettle();

    expect(sentKey, _validKey);
    expect(sentName, 'Саша');
    expect(find.byType(AddContactDialog), findsNothing);
  });

  testWidgets('отказ остаётся в диалоге и назван словами', (tester) async {
    await tester.pumpWidget(MaterialApp(
      home: Scaffold(
        body: Builder(
          builder: (context) => ElevatedButton(
            onPressed: () => showDialog<void>(
              context: context,
              builder: (_) => AddContactDialog(
                nodeKeyHex: '',
                onSubmit: (_, __) => 'маршрута к узлу нет — сначала ConnectPeer',
              ),
            ),
            child: const Text('открыть'),
          ),
        ),
      ),
    ));

    await tester.tap(find.text('открыть'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField).at(0), _validKey);
    await tester.tap(find.text('Запросить бандл'));
    await tester.pumpAndSettle();

    expect(find.byType(AddContactDialog), findsOneWidget);
    expect(find.text('маршрута к узлу нет — сначала ConnectPeer'), findsOneWidget);
  });

  testWidgets('баннер показывает незакрытый запрос и последний итог',
      (tester) async {
    await tester.pumpWidget(const MaterialApp(
      home: Scaffold(
        body: ContactNotes(
          snapshot: CoreSnapshot(
            pendingContacts: ['Саша /7fc9:abcd'],
            contactNote: 'Узел не ответил за 30s',
          ),
        ),
      ),
    ));

    expect(find.text('Узел не ответил за 30s'), findsOneWidget);
    expect(find.text('В ожидании ответа: 1'), findsOneWidget);

    await tester.pumpWidget(const MaterialApp(
      home: Scaffold(body: ContactNotes(snapshot: CoreSnapshot())),
    ));
    expect(find.byType(CircularProgressIndicator), findsNothing);
  });

  test('событие вершины обновляет список пиров, ничего не перечитывая', () {
    const start = CoreSnapshot(phase: CorePhase.online, ipv6: '/7fc9:aaaa');
    final event = CoreEvent()
      ..peersUpdated = (PeerListUpdatedEvent()
        ..peers.add(PeerInfo()
          ..uri = 'tcp://127.0.0.1:8666'
          ..yggdrasilIpv6 = '/7fc9:bbbb'
          ..transport = PeerTransportType.TRANSPORT_TCP
          ..isConnected = true));

    final next = applyCoreEvent(
        start, CoreEventEnvelope(CoreEventType.peersUpdated, event));

    expect(next, isNotNull);
    expect(next!.peers, hasLength(1));
    expect(next.peers.first.yggdrasilIpv6, '/7fc9:bbbb');
    expect(next.peers.first.isConnected, isTrue);
    // Вершина — не отказ и не старт: фаза и адрес свои остаются как были.
    expect(next.phase, CorePhase.online);
    expect(next.ipv6, '/7fc9:aaaa');
    expect(next.error, isNull);
  });

  test('пустая вершина приходит пустым списком, а не прежним', () {
    final event = CoreEvent()
      ..peersUpdated = (PeerListUpdatedEvent()
        ..peers.add(PeerInfo()
          ..uri = 'tcp://127.0.0.1:8666'
          ..yggdrasilIpv6 = '/7fc9:bbbb'
          ..isConnected = true));
    final snapshot = CoreSnapshot(
      phase: CorePhase.online,
      peers: [PeerInfo()
        ..uri = 'tcp://127.0.0.1:1'
        ..isConnected = true],
    );

    final withPeer = applyCoreEvent(
        snapshot, CoreEventEnvelope(CoreEventType.peersUpdated, event));
    expect(withPeer!.peers, hasLength(1));

    final emptied = applyCoreEvent(
        withPeer,
        CoreEventEnvelope(
            CoreEventType.peersUpdated, CoreEvent()..peersUpdated = PeerListUpdatedEvent()));
    expect(emptied!.peers, isEmpty,
        reason: 'линок пропал, а UI продолжает показывать «пиров: 1»');
  });

  test('пульс «не в сети» до старта ноды не превращает онбординг в сбой', () {
    final offline = CoreEvent()
      ..networkStatus = (NetworkStatusEvent()
        ..online = false
        ..statusNote = 'ядро не запущено');

    for (final phase in [
      CorePhase.idle,
      CorePhase.needsIdentity,
      CorePhase.starting,
    ]) {
      expect(
          applyCoreEvent(CoreSnapshot(phase: phase),
              CoreEventEnvelope(CoreEventType.networkStatus, offline)),
          isNull,
          reason: 'пульс живого, но ещё не запущенного ядра при фазе $phase '
              'не отказ');
    }

    // Совсем другое дело — ядро, которое мы считали работающим.
    final failed = applyCoreEvent(
        const CoreSnapshot(phase: CorePhase.online, ipv6: '/7fc9:aaaa'),
        CoreEventEnvelope(CoreEventType.networkStatus, offline));
    expect(failed!.phase, CorePhase.failed);
    expect(failed.error, 'ядро не запущено');
    expect(failed.ipv6, isNull);

    final back = applyCoreEvent(
        failed,
        CoreEventEnvelope(
            CoreEventType.networkStatus,
            CoreEvent()
              ..networkStatus = (NetworkStatusEvent()
                ..online = true
                ..currentIpv6 = '/7fc9:aaaa')));
    expect(back!.phase, CorePhase.online);
    expect(back.ipv6, '/7fc9:aaaa');
    expect(back.error, isNull);
  });

  test('незнакомое событие снимок не трогает', () {
    expect(
        applyCoreEvent(const CoreSnapshot(phase: CorePhase.online),
            CoreEventEnvelope(CoreEventType.unknown, CoreEvent())),
        isNull);
  });

  test('объявление обхода ложится в снимок и не трогает фазу', () {
    const start = CoreSnapshot(
        phase: CorePhase.failed, ipv6: '/7fc9:aaaa', error: 'ядро не в сети');
    final event = CoreEvent()
      ..antiDpiStatus = (AntiDpiStatusEvent()
        ..mode = AntiDpiMode.ANTI_DPI_UTLS_CHROME
        ..currentSni = 'cdn.example.org'
        ..handshakeSuccess = true
        ..transportDetails = 'исходящих 2, поднято 1: tls:// ×1 (поднято 1; SNI = cdn.example.org)');

    final next = applyCoreEvent(
        start, CoreEventEnvelope(CoreEventType.antiDpiStatus, event));

    expect(next, isNotNull);
    expect(next!.antiDpi!.currentSni, 'cdn.example.org');
    expect(next.antiDpi!.transportDetails, contains('поднято 1'));
    // Событие описывает линки, а не жизнь ноды: отказ оно в онлайн не превращает.
    expect(next.phase, CorePhase.failed);
    expect(next.error, 'ядро не в сети');
    expect(next.ipv6, '/7fc9:aaaa');
  });

  test('метка события без содержимого снимок не трогает', () {
    // Тип события UI получает из oneof, поэтому envelope с типом
    // antiDpiStatus и пустым ядром — это не «ссылок нет», а «пришёл мусор»:
    // выдумывать по нему картину нельзя.
    expect(
        applyCoreEvent(const CoreSnapshot(phase: CorePhase.online),
            CoreEventEnvelope(CoreEventType.antiDpiStatus, CoreEvent())),
        isNull);
  });

  test('прогресс бэкапа до снимка не доходит: у команды нет вызывающего из UI',
      () {
    // backup_progress ядро публикует по-настоящему, но CreateBackup/RestoreBackup
    // из клиента не зовут (реестр в core_request_contract_test.go), так что на
    // этом клиенте такое событие физически не может прийти. Проверка держит
    // границу: поле в контракте есть, а места в снимке пока нет.
    final event = CoreEvent()
      ..backupProgress = (BackupProgressEvent()
        ..totalChunks = 3
        ..completedChunks = 1
        ..currentAction = 'chunks_written');
    expect(
        applyCoreEvent(const CoreSnapshot(phase: CorePhase.online),
            CoreEventEnvelope(CoreEventType.backupProgress, event)),
        isNull);
  });

  testWidgets('экран показывает объявленное состояние ссылок', (tester) async {
    final dpi = AntiDpiStatusEvent()
      ..mode = AntiDpiMode.ANTI_DPI_UTLS_CHROME
      ..currentSni = 'cdn.example.org'
      ..handshakeSuccess = true
      ..transportDetails =
          'исходящих 2, поднято 1: tcp:// ×1 (поднято 0; TLS нет), tls:// ×1 (поднято 1; SNI = cdn.example.org)';

    await tester.pumpWidget(MaterialApp(
      home: Scaffold(body: AntiDpiLiveStatus(snapshot: CoreSnapshot(antiDpi: dpi))),
    ));

    expect(find.text('TLS, профиль Chrome'), findsOneWidget);
    expect(find.text('cdn.example.org'), findsOneWidget);
    expect(find.text('поднята хотя бы одна исходящая ссылка'), findsOneWidget);
    expect(find.text('переключения пути нет'), findsOneWidget);
    expect(find.textContaining('исходящих 2, поднято 1'), findsOneWidget);
    // Граница события на экране: профиль — намерение, а не измерение.
    expect(find.textContaining('отпечаток здесь описан'), findsOneWidget);
  });

  testWidgets('без объявления экран не выдумывает пустую вершину',
      (tester) async {
    await tester.pumpWidget(const MaterialApp(
      home: Scaffold(body: AntiDpiLiveStatus(snapshot: CoreSnapshot())),
    ));

    expect(find.textContaining('Объявления ещё не было'), findsOneWidget);
    expect(find.text('Профиль'), findsNothing);
    expect(find.textContaining('поднято'), findsNothing);
  });

  testWidgets('экран сети держит блок объявления внутри себя', (tester) async {
    // Отдельно от содержимого: виджет, который никто не вставил в сборку
    // экрана, — это тот же недостоверный статус, только на стороне UI.
    await tester.pumpWidget(const MaterialApp(
      home: ProviderScope(child: AntiDpiRadarScreen()),
    ));

    expect(tester.takeException(), isNull);
    expect(find.byType(AntiDpiLiveStatus), findsOneWidget);
    expect(find.textContaining('Объявления ещё не было'), findsOneWidget);
  });

  testWidgets('экран бэкапа поднимается без assertion про подложку ListTile',
      (tester) async {
    // Тот же класс ошибки, что поймал предыдущий тест: SwitchListTile на
    // цветном Container вместо Material. Здесь он ничем не отличается, поэтому
    // экран проверяется целиком, а не по одному виджету.
    await tester.pumpWidget(const ProviderScope(
      child: MaterialApp(home: BackupSettingsScreen()),
    ));

    expect(tester.takeException(), isNull);
    expect(find.text('Зеркалировать бэкап в два облака'), findsOneWidget);
  });
}
