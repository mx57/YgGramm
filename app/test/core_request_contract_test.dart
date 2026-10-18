// File: app/test/core_request_contract_test.dart
//
// Ядро разбирает CoreRequest оператором switch, клиент собирает запросы
// именованными аргументами конструктора, и ни одна из сторон не замечает, что
// какую-то из семнадцати веток контракта из UI не посылают вовсе. Про это в
// README всегда писал человек, и ровно поэтому проверка переехала в тест:
// та же дисциплина, что на стороне ядра (core/pkg/rpc/contract_test.go), но с
// другой ноги контракта.
//
//   - список веток берётся не из памяти автора строки, а из сгенерированного
//     protobuf: enum CoreRequest_Request и именованные параметры
//     factory CoreRequest({...}). Сначала эти два списка сверяются между
//     собой, потому что поиск по исходникам идёт по имени параметра: если
//     генератор назовёт ветку и параметр по-разному, проверка молча перестала
//     бы находить вызовы и осталась «зелёной» впустую;
//   - «клиент зовёт» означает, что имя ветки встретилось в lib/ как
//     именованный аргумент (startNode:) или как присваивание поля
//     (.startNode =). Каталог lib/gen/ исключён: там есть имя каждой ветки, и
//     поиск по всему дереву никогда бы не увидел отсутствия вызова. Комментарии
//     вырезаются, чтобы упоминание команды в прозе не сошло за вызов.
//
// Правило: добавил команду в protobuf и исполнителя в ядро — тест попросит
// решить, зовёт ли её клиент, и записать ответ в реестр. Молча зелёным он не
// бывает ни при добавлении ветки, ни при её удалении.

import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:ygggram/gen/yggproto/ygggram.pb.dart';

/// Ветки, которые клиент действительно посылает: по каждой в
/// lib/core_bridge/ygg_bridge.dart есть конструктор запроса.
const _uiSentCommands = <String>[
  'startNode',
  'stopNode',
  'sendMessage',
  'connectPeer',
  'getPeers',
  'getNodeStatus',
  'addContact',
  'getPrekeyBundle',
  'generateIdentity',
  'listConversations',
  'getMessages',
  'listContacts',
  'fetchContact',
];

/// Ветки, до которых из интерфейса не доходят. Причины разные, и тест их не
/// различает — различает комментарий рядом:
///   - createBackup и restoreBackup ядро исполняет по-настоящему (с Шага 13),
///     но мост их не зовёт: экрана бэкапа в UI пока нет, а вызов из Dart
///     появился бы вместе с ним;
///   - disconnectPeer и sfCollect ядро исполняет, но мост их не зовёт:
///     пира можно добавить, убрать — нет, а слепое релео опрашивается самим
///     ядром, а не по запросу из UI.
const _uiUnsentCommands = <String>[
  'createBackup',
  'restoreBackup',
  'disconnectPeer',
  'sfCollect',
];

/// Корень Flutter-пакета: flutter test запускается из app/, но путь лучше
/// не предполагать — иначе тест падал бы на рабочем каталоге, а не на факте.
Directory _packageRoot() {
  var dir = Directory.current;
  for (var i = 0; i < 6; i++) {
    final pubspec = File('${dir.path}${Platform.pathSeparator}pubspec.yaml');
    if (pubspec.existsSync() &&
        pubspec.readAsStringSync().contains('name: ygggram')) {
      return dir;
    }
    final parent = dir.parent;
    if (parent.path == dir.path) break;
    dir = parent;
  }
  throw StateError(
      'корень пакета ygggram не найден вверх от ${Directory.current.path} — '
      'запускай тест из каталога app/');
}

/// Имена веток из enum, порождённого protobuf. notSet — не команда, а
/// признак пустого oneof, и в реестре её быть не должно.
List<String> _enumBranches() => CoreRequest_Request.values
    .map((v) => v.name)
    .where((n) => n != 'notSet')
    .toList();

String _generatedStub() {
  final path = '${_packageRoot().path}${Platform.pathSeparator}'
      'lib${Platform.pathSeparator}gen${Platform.pathSeparator}yggproto'
      '${Platform.pathSeparator}ygggram.pb.dart';
  final file = File(path);
  if (!file.existsSync()) {
    throw StateError('сгенерированный контракт не читается ($path): '
        'выполни buf generate');
  }
  return file.readAsStringSync();
}

/// Имена именованных параметров конструктора запроса — именно так ветка
/// пишется в месте вызова.
List<String> _constructorFields(String stub) {
  const marker = 'factory CoreRequest({';
  final start = stub.indexOf(marker);
  if (start < 0) {
    throw StateError(
        'в сгенерированном protobuf нет $marker — генератор переписал контракт, '
        'обнови тест');
  }
  final end = stub.indexOf('})', start);
  if (end < 0) {
    throw StateError('список параметров factory CoreRequest не закрылся');
  }
  final body = stub.substring(start + marker.length, end);
  final names = <String>[];
  for (final chunk in body.split(',')) {
    final trimmed = chunk.trim();
    if (trimmed.isEmpty) continue;
    final match = RegExp(r'([A-Za-z0-9_]+)$').firstMatch(trimmed);
    if (match == null) continue; // хвост блока: закрывающая скобка
    final name = match.group(1)!;
    if (name == 'requestId') continue; // не ветка oneof, а общий заголовок
    names.add(name);
  }
  if (names.isEmpty) {
    throw StateError('у factory CoreRequest нет ни одного параметра — '
        'проверять нечего');
  }
  return names;
}

/// Нетестовые и негенерируемые исходники клиента: rel-путь → содержимое без
/// строчных комментариев.
Map<String, String> _clientSources() {
  final lib = Directory('${_packageRoot().path}${Platform.pathSeparator}lib');
  final out = <String, String>{};
  for (final entity in lib.listSync(recursive: true)) {
    if (entity is! File || !entity.path.endsWith('.dart')) continue;
    final rel = entity.path.substring(lib.path.length + 1);
    final slash = rel.replaceAll(r'\', '/');
    if (slash.startsWith('gen/')) continue;
    final text = entity.readAsStringSync();
    final kept = StringBuffer();
    for (final line in text.split('\n')) {
      final cut = line.indexOf('//');
      kept.writeln(cut < 0 ? line : line.substring(0, cut));
    }
    out[slash] = kept.toString();
  }
  if (out.isEmpty) {
    throw StateError('в $lib нет исходников клиента — тест ничего не проверяет');
  }
  return out;
}

bool _isSentByClient(String source, String branch) =>
    RegExp('(?:^|[^A-Za-z0-9_])$branch *:', multiLine: true).hasMatch(source) ||
    RegExp('\\.$branch *=').hasMatch(source);

Set<String> _sent(Set<String> universe, Map<String, String> sources) => {
      for (final b in universe)
        if (sources.values.any((text) => _isSentByClient(text, b))) b,
    };

void main() {
  test('имена веток CoreRequest в enum и в конструкторе совпадают', () {
    final stub = _generatedStub();
    final enumNames = _enumBranches().toSet();
    final ctorNames = _constructorFields(stub).toSet();

    // Дубликат параметра означал бы, что разбор блока конструктора поехал.
    expect(_constructorFields(stub).length, ctorNames.length,
        reason: 'имена параметров конструктора повторились: блок разобран неверно');

    expect(ctorNames.difference(enumNames), isEmpty,
        reason: 'конструктор CoreRequest принимает ветку, которой нет в enum — '
            'поиск по имени ветки искал бы то, чего в вызове не пишут');
    expect(enumNames.difference(ctorNames), isEmpty,
        reason: 'у ветки enum нет именованного параметра — генератор назвал её '
            'иначе, и проверка «зовёт ли клиент» потеряла опору');
  });

  test('реестр «клиент зовёт / не зовёт» покрывает все ветки CoreRequest', () {
    final branches = _enumBranches().toSet();
    final sources = _clientSources();
    final sent = _sent(branches, sources);

    for (final listed in _uiSentCommands) {
      expect(branches, contains(listed),
          reason: '$listed числится зовёмой, а в контракте такой ветки нет: '
              'поле из protobuf убрали, а реестр не почистили');
    }
    for (final listed in _uiUnsentCommands) {
      expect(branches, contains(listed),
          reason: '$listed числится не зовёмой, а в контракте такой ветки нет');
    }
    expect(_uiSentCommands.toSet().intersection(_uiUnsentCommands.toSet()),
        isEmpty,
        reason: 'ветка записана одновременно в зовемые и в не зовемые');

    final listedSent = _uiSentCommands.toSet();
    expect(sent.difference(listedSent), isEmpty,
        reason: 'из UI шлют команду, которой нет в _uiSentCommands: добавь её '
            'в реестр и опиши в README/AGENTS, где у неё место в интерфейсе '
            '(список найденных вызовов: $sent)');
    expect(listedSent.difference(sent), isEmpty,
        reason: 'команда помечена зовемой, но в lib/ её конструктор не найден: '
            'либо мост её потерял, либо генератор переименовал параметр');

    final unsent = branches.difference(sent);
    expect(unsent, _uiUnsentCommands.toSet(),
        reason: 'реестр не зовемых веток разошёлся с исходниками: для новой '
            'ветки protobuf нужно решение, зовёт ли её клиент, а снятую — '
            'убрать из списка. Разница: ${unsent.union(_uiUnsentCommands.toSet()).difference(unsent.intersection(_uiUnsentCommands.toSet()))}');
  });
}
