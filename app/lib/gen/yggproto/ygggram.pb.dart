// This is a generated file - do not edit.
//
// Generated from ygggram.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import 'ygggram.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'ygggram.pbenum.dart';

class NodeConfig extends $pb.GeneratedMessage {
  factory NodeConfig({
    $core.String? dataDir,
    $core.String? cacheDir,
    $core.String? mnemonicSeed,
    $core.List<$core.int>? ed25519NodePrivkey,
    $core.List<$core.int>? x25519ChatPrivkey,
    $core.List<$core.int>? masterKey256,
    AntiDpiMode? antiDpiMode,
    $core.String? antiDpiSni,
    $core.String? antiDpiPassword,
    $core.Iterable<$core.String>? bootstrapPeers,
    $core.Iterable<$core.String>? dohServers,
    $core.bool? enableDhtBep44,
    $core.String? listenAddress,
    $core.int? listenPort,
    $core.bool? userspaceOnly,
    $core.int? connectionTimeoutSeconds,
    $core.int? keepaliveIntervalSeconds,
    $core.bool? enableLanMulticast,
    $core.bool? sfRelayEnabled,
    $core.Iterable<$core.List<$core.int>>? sfRelayNodeKeys,
    $core.bool? servePrekeyBundles,
  }) {
    final result = NodeConfig._();
    if (dataDir != null) result.dataDir = dataDir;
    if (cacheDir != null) result.cacheDir = cacheDir;
    if (mnemonicSeed != null) result.mnemonicSeed = mnemonicSeed;
    if (ed25519NodePrivkey != null)
      result.ed25519NodePrivkey = ed25519NodePrivkey;
    if (x25519ChatPrivkey != null) result.x25519ChatPrivkey = x25519ChatPrivkey;
    if (masterKey256 != null) result.masterKey256 = masterKey256;
    if (antiDpiMode != null) result.antiDpiMode = antiDpiMode;
    if (antiDpiSni != null) result.antiDpiSni = antiDpiSni;
    if (antiDpiPassword != null) result.antiDpiPassword = antiDpiPassword;
    if (bootstrapPeers != null) result.bootstrapPeers.addAll(bootstrapPeers);
    if (dohServers != null) result.dohServers.addAll(dohServers);
    if (enableDhtBep44 != null) result.enableDhtBep44 = enableDhtBep44;
    if (listenAddress != null) result.listenAddress = listenAddress;
    if (listenPort != null) result.listenPort = listenPort;
    if (userspaceOnly != null) result.userspaceOnly = userspaceOnly;
    if (connectionTimeoutSeconds != null)
      result.connectionTimeoutSeconds = connectionTimeoutSeconds;
    if (keepaliveIntervalSeconds != null)
      result.keepaliveIntervalSeconds = keepaliveIntervalSeconds;
    if (enableLanMulticast != null)
      result.enableLanMulticast = enableLanMulticast;
    if (sfRelayEnabled != null) result.sfRelayEnabled = sfRelayEnabled;
    if (sfRelayNodeKeys != null) result.sfRelayNodeKeys.addAll(sfRelayNodeKeys);
    if (servePrekeyBundles != null)
      result.servePrekeyBundles = servePrekeyBundles;
    return result;
  }

  NodeConfig._();

  factory NodeConfig.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      NodeConfig()..mergeFromBuffer(data, registry);
  factory NodeConfig.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      NodeConfig()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'NodeConfig',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: NodeConfig.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'dataDir')
    ..aOS(2, _omitFieldNames ? '' : 'cacheDir')
    ..aOS(3, _omitFieldNames ? '' : 'mnemonicSeed')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'ed25519NodePrivkey', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'x25519ChatPrivkey', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        6, _omitFieldNames ? '' : 'masterKey256', $pb.PbFieldType.OY,
        protoName: 'master_key_256')
    ..aE<AntiDpiMode>(7, _omitFieldNames ? '' : 'antiDpiMode',
        enumValues: AntiDpiMode.values)
    ..aOS(8, _omitFieldNames ? '' : 'antiDpiSni')
    ..aOS(9, _omitFieldNames ? '' : 'antiDpiPassword')
    ..pPS(10, _omitFieldNames ? '' : 'bootstrapPeers')
    ..pPS(11, _omitFieldNames ? '' : 'dohServers')
    ..aOB(12, _omitFieldNames ? '' : 'enableDhtBep44')
    ..aOS(13, _omitFieldNames ? '' : 'listenAddress')
    ..aI(14, _omitFieldNames ? '' : 'listenPort')
    ..aOB(15, _omitFieldNames ? '' : 'userspaceOnly')
    ..aI(16, _omitFieldNames ? '' : 'connectionTimeoutSeconds',
        fieldType: $pb.PbFieldType.OU3)
    ..aI(17, _omitFieldNames ? '' : 'keepaliveIntervalSeconds',
        fieldType: $pb.PbFieldType.OU3)
    ..aOB(18, _omitFieldNames ? '' : 'enableLanMulticast')
    ..aOB(19, _omitFieldNames ? '' : 'sfRelayEnabled')
    ..p<$core.List<$core.int>>(
        20, _omitFieldNames ? '' : 'sfRelayNodeKeys', $pb.PbFieldType.PY)
    ..aOB(21, _omitFieldNames ? '' : 'servePrekeyBundles')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NodeConfig clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NodeConfig copyWith(void Function(NodeConfig) updates) =>
      super.copyWith((message) => updates(message as NodeConfig)) as NodeConfig;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use NodeConfig() / NodeConfig.new instead')
  static NodeConfig create() => NodeConfig._();
  static $pb.GeneratedMessage $_createMessage() => NodeConfig._();
  @$core.override
  NodeConfig createEmptyInstance() => NodeConfig._();
  @$core.pragma('dart2js:noInline')
  static NodeConfig getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<NodeConfig>(NodeConfig.$_createMessage);
  static NodeConfig? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get dataDir => $_getSZ(0);
  @$pb.TagNumber(1)
  set dataDir($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDataDir() => $_has(0);
  @$pb.TagNumber(1)
  void clearDataDir() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get cacheDir => $_getSZ(1);
  @$pb.TagNumber(2)
  set cacheDir($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCacheDir() => $_has(1);
  @$pb.TagNumber(2)
  void clearCacheDir() => $_clearField(2);

  /// Фраза восстановления: 12 или 24 слова английского списка BIP-39 с
  /// контрольной суммой. Ядро отвергает фразу, у которой суммы не сходится, а не
  /// выводит из неё «на всякий случай» другой набор ключей.
  @$pb.TagNumber(3)
  $core.String get mnemonicSeed => $_getSZ(2);
  @$pb.TagNumber(3)
  set mnemonicSeed($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMnemonicSeed() => $_has(2);
  @$pb.TagNumber(3)
  void clearMnemonicSeed() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get ed25519NodePrivkey => $_getN(3);
  @$pb.TagNumber(4)
  set ed25519NodePrivkey($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasEd25519NodePrivkey() => $_has(3);
  @$pb.TagNumber(4)
  void clearEd25519NodePrivkey() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.List<$core.int> get x25519ChatPrivkey => $_getN(4);
  @$pb.TagNumber(5)
  set x25519ChatPrivkey($core.List<$core.int> value) => $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasX25519ChatPrivkey() => $_has(4);
  @$pb.TagNumber(5)
  void clearX25519ChatPrivkey() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.List<$core.int> get masterKey256 => $_getN(5);
  @$pb.TagNumber(6)
  set masterKey256($core.List<$core.int> value) => $_setBytes(5, value);
  @$pb.TagNumber(6)
  $core.bool hasMasterKey256() => $_has(5);
  @$pb.TagNumber(6)
  void clearMasterKey256() => $_clearField(6);

  @$pb.TagNumber(7)
  AntiDpiMode get antiDpiMode => $_getN(6);
  @$pb.TagNumber(7)
  set antiDpiMode(AntiDpiMode value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasAntiDpiMode() => $_has(6);
  @$pb.TagNumber(7)
  void clearAntiDpiMode() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get antiDpiSni => $_getSZ(7);
  @$pb.TagNumber(8)
  set antiDpiSni($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasAntiDpiSni() => $_has(7);
  @$pb.TagNumber(8)
  void clearAntiDpiSni() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.String get antiDpiPassword => $_getSZ(8);
  @$pb.TagNumber(9)
  set antiDpiPassword($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasAntiDpiPassword() => $_has(8);
  @$pb.TagNumber(9)
  void clearAntiDpiPassword() => $_clearField(9);

  @$pb.TagNumber(10)
  $pb.PbList<$core.String> get bootstrapPeers => $_getList(9);

  @$pb.TagNumber(11)
  $pb.PbList<$core.String> get dohServers => $_getList(10);

  @$pb.TagNumber(12)
  $core.bool get enableDhtBep44 => $_getBF(11);
  @$pb.TagNumber(12)
  set enableDhtBep44($core.bool value) => $_setBool(11, value);
  @$pb.TagNumber(12)
  $core.bool hasEnableDhtBep44() => $_has(11);
  @$pb.TagNumber(12)
  void clearEnableDhtBep44() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.String get listenAddress => $_getSZ(12);
  @$pb.TagNumber(13)
  set listenAddress($core.String value) => $_setString(12, value);
  @$pb.TagNumber(13)
  $core.bool hasListenAddress() => $_has(12);
  @$pb.TagNumber(13)
  void clearListenAddress() => $_clearField(13);

  @$pb.TagNumber(14)
  $core.int get listenPort => $_getIZ(13);
  @$pb.TagNumber(14)
  set listenPort($core.int value) => $_setSignedInt32(13, value);
  @$pb.TagNumber(14)
  $core.bool hasListenPort() => $_has(13);
  @$pb.TagNumber(14)
  void clearListenPort() => $_clearField(14);

  @$pb.TagNumber(15)
  $core.bool get userspaceOnly => $_getBF(14);
  @$pb.TagNumber(15)
  set userspaceOnly($core.bool value) => $_setBool(14, value);
  @$pb.TagNumber(15)
  $core.bool hasUserspaceOnly() => $_has(14);
  @$pb.TagNumber(15)
  void clearUserspaceOnly() => $_clearField(15);

  @$pb.TagNumber(16)
  $core.int get connectionTimeoutSeconds => $_getIZ(15);
  @$pb.TagNumber(16)
  set connectionTimeoutSeconds($core.int value) =>
      $_setUnsignedInt32(15, value);
  @$pb.TagNumber(16)
  $core.bool hasConnectionTimeoutSeconds() => $_has(15);
  @$pb.TagNumber(16)
  void clearConnectionTimeoutSeconds() => $_clearField(16);

  @$pb.TagNumber(17)
  $core.int get keepaliveIntervalSeconds => $_getIZ(16);
  @$pb.TagNumber(17)
  set keepaliveIntervalSeconds($core.int value) =>
      $_setUnsignedInt32(16, value);
  @$pb.TagNumber(17)
  $core.bool hasKeepaliveIntervalSeconds() => $_has(16);
  @$pb.TagNumber(17)
  void clearKeepaliveIntervalSeconds() => $_clearField(17);

  /// Wire-совместимый LAN-мультикаст Yggdrasil (ff02::114:9001). По умолчанию
  /// выключен: в тестах и CI подписка на IPv6-группу недоступна, а молча
  /// деградировать сеть ноды нельзя.
  @$pb.TagNumber(18)
  $core.bool get enableLanMulticast => $_getBF(17);
  @$pb.TagNumber(18)
  set enableLanMulticast($core.bool value) => $_setBool(17, value);
  @$pb.TagNumber(18)
  $core.bool hasEnableLanMulticast() => $_has(17);
  @$pb.TagNumber(18)
  void clearEnableLanMulticast() => $_clearField(18);

  /// Этот узел работает слепым S&F-релеем: хранит чужие конверты, пока их не
  /// заберёт адресат.
  @$pb.TagNumber(19)
  $core.bool get sfRelayEnabled => $_getBF(18);
  @$pb.TagNumber(19)
  set sfRelayEnabled($core.bool value) => $_setBool(18, value);
  @$pb.TagNumber(19)
  $core.bool hasSfRelayEnabled() => $_has(18);
  @$pb.TagNumber(19)
  void clearSfRelayEnabled() => $_clearField(19);

  /// Ed25519-ключи нод-релеев, куда класть конверты офлайн-адресата и у кого их
  /// забирать. Ключ ноды — адрес в оверлее, поэтому отдельного IPv6 не нужно.
  @$pb.TagNumber(20)
  $pb.PbList<$core.List<$core.int>> get sfRelayNodeKeys => $_getList(19);

  /// Отдавать ли свой PreKeyBundle по запросу чужого узла (bundle_req). По
  /// умолчанию включено: без этого контакта нельзя добавить, кроме как
  /// переслать карту байтами вне сети. Выключается явно — бандл раскрывает
  /// спрашивающему набор открытых ключей этой идентичности.
  @$pb.TagNumber(21)
  $core.bool get servePrekeyBundles => $_getBF(20);
  @$pb.TagNumber(21)
  set servePrekeyBundles($core.bool value) => $_setBool(20, value);
  @$pb.TagNumber(21)
  $core.bool hasServePrekeyBundles() => $_has(20);
  @$pb.TagNumber(21)
  void clearServePrekeyBundles() => $_clearField(21);
}

class SealedEnvelope extends $pb.GeneratedMessage {
  factory SealedEnvelope({
    $core.List<$core.int>? recipientX25519Pubkey,
    $core.List<$core.int>? ephemeralX25519Pubkey,
    $core.List<$core.int>? encryptedPayload,
    $core.List<$core.int>? nonce,
    $fixnum.Int64? timestamp,
    $fixnum.Int64? powNonce,
    $core.List<$core.int>? blindToken,
    $core.int? routingHops,
  }) {
    final result = SealedEnvelope._();
    if (recipientX25519Pubkey != null)
      result.recipientX25519Pubkey = recipientX25519Pubkey;
    if (ephemeralX25519Pubkey != null)
      result.ephemeralX25519Pubkey = ephemeralX25519Pubkey;
    if (encryptedPayload != null) result.encryptedPayload = encryptedPayload;
    if (nonce != null) result.nonce = nonce;
    if (timestamp != null) result.timestamp = timestamp;
    if (powNonce != null) result.powNonce = powNonce;
    if (blindToken != null) result.blindToken = blindToken;
    if (routingHops != null) result.routingHops = routingHops;
    return result;
  }

  SealedEnvelope._();

  factory SealedEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SealedEnvelope()..mergeFromBuffer(data, registry);
  factory SealedEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SealedEnvelope()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SealedEnvelope',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: SealedEnvelope.$_createMessage)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'recipientX25519Pubkey', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'ephemeralX25519Pubkey', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'encryptedPayload', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'nonce', $pb.PbFieldType.OY)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'timestamp', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'powNonce', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$core.List<$core.int>>(
        7, _omitFieldNames ? '' : 'blindToken', $pb.PbFieldType.OY)
    ..aI(8, _omitFieldNames ? '' : 'routingHops',
        fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SealedEnvelope clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SealedEnvelope copyWith(void Function(SealedEnvelope) updates) =>
      super.copyWith((message) => updates(message as SealedEnvelope))
          as SealedEnvelope;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use SealedEnvelope() / SealedEnvelope.new instead')
  static SealedEnvelope create() => SealedEnvelope._();
  static $pb.GeneratedMessage $_createMessage() => SealedEnvelope._();
  @$core.override
  SealedEnvelope createEmptyInstance() => SealedEnvelope._();
  @$core.pragma('dart2js:noInline')
  static SealedEnvelope getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SealedEnvelope>(
          SealedEnvelope.$_createMessage);
  static SealedEnvelope? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get recipientX25519Pubkey => $_getN(0);
  @$pb.TagNumber(1)
  set recipientX25519Pubkey($core.List<$core.int> value) =>
      $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRecipientX25519Pubkey() => $_has(0);
  @$pb.TagNumber(1)
  void clearRecipientX25519Pubkey() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get ephemeralX25519Pubkey => $_getN(1);
  @$pb.TagNumber(2)
  set ephemeralX25519Pubkey($core.List<$core.int> value) =>
      $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEphemeralX25519Pubkey() => $_has(1);
  @$pb.TagNumber(2)
  void clearEphemeralX25519Pubkey() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get encryptedPayload => $_getN(2);
  @$pb.TagNumber(3)
  set encryptedPayload($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasEncryptedPayload() => $_has(2);
  @$pb.TagNumber(3)
  void clearEncryptedPayload() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get nonce => $_getN(3);
  @$pb.TagNumber(4)
  set nonce($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasNonce() => $_has(3);
  @$pb.TagNumber(4)
  void clearNonce() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get timestamp => $_getI64(4);
  @$pb.TagNumber(5)
  set timestamp($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasTimestamp() => $_has(4);
  @$pb.TagNumber(5)
  void clearTimestamp() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get powNonce => $_getI64(5);
  @$pb.TagNumber(6)
  set powNonce($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasPowNonce() => $_has(5);
  @$pb.TagNumber(6)
  void clearPowNonce() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.List<$core.int> get blindToken => $_getN(6);
  @$pb.TagNumber(7)
  set blindToken($core.List<$core.int> value) => $_setBytes(6, value);
  @$pb.TagNumber(7)
  $core.bool hasBlindToken() => $_has(6);
  @$pb.TagNumber(7)
  void clearBlindToken() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.int get routingHops => $_getIZ(7);
  @$pb.TagNumber(8)
  set routingHops($core.int value) => $_setUnsignedInt32(7, value);
  @$pb.TagNumber(8)
  $core.bool hasRoutingHops() => $_has(7);
  @$pb.TagNumber(8)
  void clearRoutingHops() => $_clearField(8);
}

enum WireFrame_Body {
  chat,
  sfPut,
  sfPull,
  sfReply,
  bundleReq,
  bundleResp,
  notSet
}

/// Кадр поверх оверлея. До Шага 6 на проводе лежал только SealedEnvelope, и
/// любой другой пакет означал бы «испорченный конверт». Служебному трафику
/// S&F-релея нужна собственная оболочка: не самописный префикс-байт, а oneof,
/// который читает тот же protobuf-парсер.
class WireFrame extends $pb.GeneratedMessage {
  factory WireFrame({
    SealedEnvelope? chat,
    SfPut? sfPut,
    SfPull? sfPull,
    SfReply? sfReply,
    BundleReq? bundleReq,
    BundleResp? bundleResp,
  }) {
    final result = WireFrame._();
    if (chat != null) result.chat = chat;
    if (sfPut != null) result.sfPut = sfPut;
    if (sfPull != null) result.sfPull = sfPull;
    if (sfReply != null) result.sfReply = sfReply;
    if (bundleReq != null) result.bundleReq = bundleReq;
    if (bundleResp != null) result.bundleResp = bundleResp;
    return result;
  }

  WireFrame._();

  factory WireFrame.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      WireFrame()..mergeFromBuffer(data, registry);
  factory WireFrame.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      WireFrame()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, WireFrame_Body> _WireFrame_BodyByTag = {
    1: WireFrame_Body.chat,
    2: WireFrame_Body.sfPut,
    3: WireFrame_Body.sfPull,
    4: WireFrame_Body.sfReply,
    5: WireFrame_Body.bundleReq,
    6: WireFrame_Body.bundleResp,
    0: WireFrame_Body.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WireFrame',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: WireFrame.$_createMessage)
    ..oo(0, [1, 2, 3, 4, 5, 6])
    ..aOM<SealedEnvelope>(1, _omitFieldNames ? '' : 'chat',
        subBuilder: SealedEnvelope.$_createMessage)
    ..aOM<SfPut>(2, _omitFieldNames ? '' : 'sfPut',
        subBuilder: SfPut.$_createMessage)
    ..aOM<SfPull>(3, _omitFieldNames ? '' : 'sfPull',
        subBuilder: SfPull.$_createMessage)
    ..aOM<SfReply>(4, _omitFieldNames ? '' : 'sfReply',
        subBuilder: SfReply.$_createMessage)
    ..aOM<BundleReq>(5, _omitFieldNames ? '' : 'bundleReq',
        subBuilder: BundleReq.$_createMessage)
    ..aOM<BundleResp>(6, _omitFieldNames ? '' : 'bundleResp',
        subBuilder: BundleResp.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WireFrame clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WireFrame copyWith(void Function(WireFrame) updates) =>
      super.copyWith((message) => updates(message as WireFrame)) as WireFrame;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use WireFrame() / WireFrame.new instead')
  static WireFrame create() => WireFrame._();
  static $pb.GeneratedMessage $_createMessage() => WireFrame._();
  @$core.override
  WireFrame createEmptyInstance() => WireFrame._();
  @$core.pragma('dart2js:noInline')
  static WireFrame getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WireFrame>(WireFrame.$_createMessage);
  static WireFrame? _defaultInstance;

  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  @$pb.TagNumber(6)
  WireFrame_Body whichBody() => _WireFrame_BodyByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  @$pb.TagNumber(6)
  void clearBody() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  SealedEnvelope get chat => $_getN(0);
  @$pb.TagNumber(1)
  set chat(SealedEnvelope value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasChat() => $_has(0);
  @$pb.TagNumber(1)
  void clearChat() => $_clearField(1);
  @$pb.TagNumber(1)
  SealedEnvelope ensureChat() => $_ensure(0);

  @$pb.TagNumber(2)
  SfPut get sfPut => $_getN(1);
  @$pb.TagNumber(2)
  set sfPut(SfPut value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasSfPut() => $_has(1);
  @$pb.TagNumber(2)
  void clearSfPut() => $_clearField(2);
  @$pb.TagNumber(2)
  SfPut ensureSfPut() => $_ensure(1);

  @$pb.TagNumber(3)
  SfPull get sfPull => $_getN(2);
  @$pb.TagNumber(3)
  set sfPull(SfPull value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasSfPull() => $_has(2);
  @$pb.TagNumber(3)
  void clearSfPull() => $_clearField(3);
  @$pb.TagNumber(3)
  SfPull ensureSfPull() => $_ensure(2);

  @$pb.TagNumber(4)
  SfReply get sfReply => $_getN(3);
  @$pb.TagNumber(4)
  set sfReply(SfReply value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasSfReply() => $_has(3);
  @$pb.TagNumber(4)
  void clearSfReply() => $_clearField(4);
  @$pb.TagNumber(4)
  SfReply ensureSfReply() => $_ensure(3);

  @$pb.TagNumber(5)
  BundleReq get bundleReq => $_getN(4);
  @$pb.TagNumber(5)
  set bundleReq(BundleReq value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasBundleReq() => $_has(4);
  @$pb.TagNumber(5)
  void clearBundleReq() => $_clearField(5);
  @$pb.TagNumber(5)
  BundleReq ensureBundleReq() => $_ensure(4);

  @$pb.TagNumber(6)
  BundleResp get bundleResp => $_getN(5);
  @$pb.TagNumber(6)
  set bundleResp(BundleResp value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasBundleResp() => $_has(5);
  @$pb.TagNumber(6)
  void clearBundleResp() => $_clearField(6);
  @$pb.TagNumber(6)
  BundleResp ensureBundleResp() => $_ensure(5);
}

/// Связка мастер-ключей в открытом виде. Релей обязан проверить, что X25519,
/// из которого выведен ящик, подписан тем же Ed25519, что расписал запрос.
class WireIdentityProof extends $pb.GeneratedMessage {
  factory WireIdentityProof({
    $core.List<$core.int>? masterEd25519,
    $core.List<$core.int>? masterX25519,
    $core.List<$core.int>? binding,
    $core.String? deviceId,
    $core.String? deviceName,
    $core.List<$core.int>? deviceEd25519,
    $core.List<$core.int>? deviceX25519,
    $fixnum.Int64? deviceCreatedAt,
    $core.List<$core.int>? deviceCertSig,
  }) {
    final result = WireIdentityProof._();
    if (masterEd25519 != null) result.masterEd25519 = masterEd25519;
    if (masterX25519 != null) result.masterX25519 = masterX25519;
    if (binding != null) result.binding = binding;
    if (deviceId != null) result.deviceId = deviceId;
    if (deviceName != null) result.deviceName = deviceName;
    if (deviceEd25519 != null) result.deviceEd25519 = deviceEd25519;
    if (deviceX25519 != null) result.deviceX25519 = deviceX25519;
    if (deviceCreatedAt != null) result.deviceCreatedAt = deviceCreatedAt;
    if (deviceCertSig != null) result.deviceCertSig = deviceCertSig;
    return result;
  }

  WireIdentityProof._();

  factory WireIdentityProof.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      WireIdentityProof()..mergeFromBuffer(data, registry);
  factory WireIdentityProof.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      WireIdentityProof()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WireIdentityProof',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: WireIdentityProof.$_createMessage)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'masterEd25519', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'masterX25519', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'binding', $pb.PbFieldType.OY)
    ..aOS(4, _omitFieldNames ? '' : 'deviceId')
    ..aOS(5, _omitFieldNames ? '' : 'deviceName')
    ..a<$core.List<$core.int>>(
        6, _omitFieldNames ? '' : 'deviceEd25519', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        7, _omitFieldNames ? '' : 'deviceX25519', $pb.PbFieldType.OY)
    ..aInt64(8, _omitFieldNames ? '' : 'deviceCreatedAt')
    ..a<$core.List<$core.int>>(
        9, _omitFieldNames ? '' : 'deviceCertSig', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WireIdentityProof clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WireIdentityProof copyWith(void Function(WireIdentityProof) updates) =>
      super.copyWith((message) => updates(message as WireIdentityProof))
          as WireIdentityProof;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use WireIdentityProof() / WireIdentityProof.new instead')
  static WireIdentityProof create() => WireIdentityProof._();
  static $pb.GeneratedMessage $_createMessage() => WireIdentityProof._();
  @$core.override
  WireIdentityProof createEmptyInstance() => WireIdentityProof._();
  @$core.pragma('dart2js:noInline')
  static WireIdentityProof getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<WireIdentityProof>(
          WireIdentityProof.$_createMessage);
  static WireIdentityProof? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get masterEd25519 => $_getN(0);
  @$pb.TagNumber(1)
  set masterEd25519($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMasterEd25519() => $_has(0);
  @$pb.TagNumber(1)
  void clearMasterEd25519() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get masterX25519 => $_getN(1);
  @$pb.TagNumber(2)
  set masterX25519($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMasterX25519() => $_has(1);
  @$pb.TagNumber(2)
  void clearMasterX25519() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get binding => $_getN(2);
  @$pb.TagNumber(3)
  set binding($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasBinding() => $_has(2);
  @$pb.TagNumber(3)
  void clearBinding() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get deviceId => $_getSZ(3);
  @$pb.TagNumber(4)
  set deviceId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDeviceId() => $_has(3);
  @$pb.TagNumber(4)
  void clearDeviceId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get deviceName => $_getSZ(4);
  @$pb.TagNumber(5)
  set deviceName($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDeviceName() => $_has(4);
  @$pb.TagNumber(5)
  void clearDeviceName() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.List<$core.int> get deviceEd25519 => $_getN(5);
  @$pb.TagNumber(6)
  set deviceEd25519($core.List<$core.int> value) => $_setBytes(5, value);
  @$pb.TagNumber(6)
  $core.bool hasDeviceEd25519() => $_has(5);
  @$pb.TagNumber(6)
  void clearDeviceEd25519() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.List<$core.int> get deviceX25519 => $_getN(6);
  @$pb.TagNumber(7)
  set deviceX25519($core.List<$core.int> value) => $_setBytes(6, value);
  @$pb.TagNumber(7)
  $core.bool hasDeviceX25519() => $_has(6);
  @$pb.TagNumber(7)
  void clearDeviceX25519() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get deviceCreatedAt => $_getI64(7);
  @$pb.TagNumber(8)
  set deviceCreatedAt($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasDeviceCreatedAt() => $_has(7);
  @$pb.TagNumber(8)
  void clearDeviceCreatedAt() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.List<$core.int> get deviceCertSig => $_getN(8);
  @$pb.TagNumber(9)
  set deviceCertSig($core.List<$core.int> value) => $_setBytes(8, value);
  @$pb.TagNumber(9)
  $core.bool hasDeviceCertSig() => $_has(8);
  @$pb.TagNumber(9)
  void clearDeviceCertSig() => $_clearField(9);
}

/// Кладу в слепой ящик: релей начинку не читает, он проверяет работу (PoW
/// лежит в самом конверте), квоту и срок хранения. Mailbox_id обязан совпасть с
/// blind_token конверта: разнобой означал бы, что письмо проиндексировано чужим
/// ящиком, то есть адресат его никогда не найдёт.
class SfPut extends $pb.GeneratedMessage {
  factory SfPut({
    $core.List<$core.int>? mailboxId,
    SealedEnvelope? envelope,
  }) {
    final result = SfPut._();
    if (mailboxId != null) result.mailboxId = mailboxId;
    if (envelope != null) result.envelope = envelope;
    return result;
  }

  SfPut._();

  factory SfPut.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SfPut()..mergeFromBuffer(data, registry);
  factory SfPut.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SfPut()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SfPut',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: SfPut.$_createMessage)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'mailboxId', $pb.PbFieldType.OY)
    ..aOM<SealedEnvelope>(2, _omitFieldNames ? '' : 'envelope',
        subBuilder: SealedEnvelope.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SfPut clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SfPut copyWith(void Function(SfPut) updates) =>
      super.copyWith((message) => updates(message as SfPut)) as SfPut;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use SfPut() / SfPut.new instead')
  static SfPut create() => SfPut._();
  static $pb.GeneratedMessage $_createMessage() => SfPut._();
  @$core.override
  SfPut createEmptyInstance() => SfPut._();
  @$core.pragma('dart2js:noInline')
  static SfPut getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SfPut>(SfPut.$_createMessage);
  static SfPut? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get mailboxId => $_getN(0);
  @$pb.TagNumber(1)
  set mailboxId($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMailboxId() => $_has(0);
  @$pb.TagNumber(1)
  void clearMailboxId() => $_clearField(1);

  @$pb.TagNumber(2)
  SealedEnvelope get envelope => $_getN(1);
  @$pb.TagNumber(2)
  set envelope(SealedEnvelope value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasEnvelope() => $_has(1);
  @$pb.TagNumber(2)
  void clearEnvelope() => $_clearField(2);
  @$pb.TagNumber(2)
  SealedEnvelope ensureEnvelope() => $_ensure(1);
}

/// Забираю из ящика. Подпись мастер-Ed25519 поверх домена, ящика, эпохи и метки
/// времени, а proof доказывает, что этот Ed25519 привязан к X25519, из которого
/// ящик выведен: без связки релей отдал бы почту всякому, кто посчитал
/// открытый ключ получателя.
class SfPull extends $pb.GeneratedMessage {
  factory SfPull({
    $core.List<$core.int>? mailboxId,
    $fixnum.Int64? epochDay,
    $fixnum.Int64? timestampS,
    $core.List<$core.int>? signature,
    WireIdentityProof? proof,
  }) {
    final result = SfPull._();
    if (mailboxId != null) result.mailboxId = mailboxId;
    if (epochDay != null) result.epochDay = epochDay;
    if (timestampS != null) result.timestampS = timestampS;
    if (signature != null) result.signature = signature;
    if (proof != null) result.proof = proof;
    return result;
  }

  SfPull._();

  factory SfPull.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SfPull()..mergeFromBuffer(data, registry);
  factory SfPull.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SfPull()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SfPull',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: SfPull.$_createMessage)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'mailboxId', $pb.PbFieldType.OY)
    ..aInt64(2, _omitFieldNames ? '' : 'epochDay')
    ..aInt64(3, _omitFieldNames ? '' : 'timestampS')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'signature', $pb.PbFieldType.OY)
    ..aOM<WireIdentityProof>(5, _omitFieldNames ? '' : 'proof',
        subBuilder: WireIdentityProof.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SfPull clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SfPull copyWith(void Function(SfPull) updates) =>
      super.copyWith((message) => updates(message as SfPull)) as SfPull;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use SfPull() / SfPull.new instead')
  static SfPull create() => SfPull._();
  static $pb.GeneratedMessage $_createMessage() => SfPull._();
  @$core.override
  SfPull createEmptyInstance() => SfPull._();
  @$core.pragma('dart2js:noInline')
  static SfPull getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SfPull>(SfPull.$_createMessage);
  static SfPull? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get mailboxId => $_getN(0);
  @$pb.TagNumber(1)
  set mailboxId($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMailboxId() => $_has(0);
  @$pb.TagNumber(1)
  void clearMailboxId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get epochDay => $_getI64(1);
  @$pb.TagNumber(2)
  set epochDay($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEpochDay() => $_has(1);
  @$pb.TagNumber(2)
  void clearEpochDay() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get timestampS => $_getI64(2);
  @$pb.TagNumber(3)
  set timestampS($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTimestampS() => $_has(2);
  @$pb.TagNumber(3)
  void clearTimestampS() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get signature => $_getN(3);
  @$pb.TagNumber(4)
  set signature($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSignature() => $_has(3);
  @$pb.TagNumber(4)
  void clearSignature() => $_clearField(4);

  @$pb.TagNumber(5)
  WireIdentityProof get proof => $_getN(4);
  @$pb.TagNumber(5)
  set proof(WireIdentityProof value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasProof() => $_has(4);
  @$pb.TagNumber(5)
  void clearProof() => $_clearField(5);
  @$pb.TagNumber(5)
  WireIdentityProof ensureProof() => $_ensure(4);
}

class SfReply extends $pb.GeneratedMessage {
  factory SfReply({
    $core.Iterable<SealedEnvelope>? envelopes,
    $core.String? error,
  }) {
    final result = SfReply._();
    if (envelopes != null) result.envelopes.addAll(envelopes);
    if (error != null) result.error = error;
    return result;
  }

  SfReply._();

  factory SfReply.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SfReply()..mergeFromBuffer(data, registry);
  factory SfReply.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SfReply()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SfReply',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: SfReply.$_createMessage)
    ..pPM<SealedEnvelope>(1, _omitFieldNames ? '' : 'envelopes',
        subBuilder: SealedEnvelope.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'error')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SfReply clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SfReply copyWith(void Function(SfReply) updates) =>
      super.copyWith((message) => updates(message as SfReply)) as SfReply;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use SfReply() / SfReply.new instead')
  static SfReply create() => SfReply._();
  static $pb.GeneratedMessage $_createMessage() => SfReply._();
  @$core.override
  SfReply createEmptyInstance() => SfReply._();
  @$core.pragma('dart2js:noInline')
  static SfReply getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SfReply>(SfReply.$_createMessage);
  static SfReply? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<SealedEnvelope> get envelopes => $_getList(0);

  /// Пустой список без ошибки — ящик пуст. Ошибка обязана быть названа: «нет
  /// почты» и «тебя не пустили» — это разные новости для UI и для счётчиков.
  @$pb.TagNumber(2)
  $core.String get error => $_getSZ(1);
  @$pb.TagNumber(2)
  set error($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasError() => $_has(1);
  @$pb.TagNumber(2)
  void clearError() => $_clearField(2);
}

class PreKeyBundle extends $pb.GeneratedMessage {
  factory PreKeyBundle({
    $core.List<$core.int>? identityEd25519Pubkey,
    $core.List<$core.int>? identityX25519Pubkey,
    $core.List<$core.int>? signedPrekey,
    $core.List<$core.int>? signedPrekeySignature,
    $core.Iterable<$core.List<$core.int>>? oneTimePrekeys,
    $fixnum.Int64? prekeyTimestamp,
    $core.int? keyId,
    $core.List<$core.int>? nodeEd25519Pubkey,
    $core.List<$core.int>? nodeKeyProof,
  }) {
    final result = PreKeyBundle._();
    if (identityEd25519Pubkey != null)
      result.identityEd25519Pubkey = identityEd25519Pubkey;
    if (identityX25519Pubkey != null)
      result.identityX25519Pubkey = identityX25519Pubkey;
    if (signedPrekey != null) result.signedPrekey = signedPrekey;
    if (signedPrekeySignature != null)
      result.signedPrekeySignature = signedPrekeySignature;
    if (oneTimePrekeys != null) result.oneTimePrekeys.addAll(oneTimePrekeys);
    if (prekeyTimestamp != null) result.prekeyTimestamp = prekeyTimestamp;
    if (keyId != null) result.keyId = keyId;
    if (nodeEd25519Pubkey != null) result.nodeEd25519Pubkey = nodeEd25519Pubkey;
    if (nodeKeyProof != null) result.nodeKeyProof = nodeKeyProof;
    return result;
  }

  PreKeyBundle._();

  factory PreKeyBundle.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      PreKeyBundle()..mergeFromBuffer(data, registry);
  factory PreKeyBundle.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      PreKeyBundle()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PreKeyBundle',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: PreKeyBundle.$_createMessage)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'identityEd25519Pubkey', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'identityX25519Pubkey', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'signedPrekey', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'signedPrekeySignature', $pb.PbFieldType.OY)
    ..p<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'oneTimePrekeys', $pb.PbFieldType.PY)
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'prekeyTimestamp', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(7, _omitFieldNames ? '' : 'keyId', fieldType: $pb.PbFieldType.OU3)
    ..a<$core.List<$core.int>>(
        8, _omitFieldNames ? '' : 'nodeEd25519Pubkey', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        9, _omitFieldNames ? '' : 'nodeKeyProof', $pb.PbFieldType.OY)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PreKeyBundle clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PreKeyBundle copyWith(void Function(PreKeyBundle) updates) =>
      super.copyWith((message) => updates(message as PreKeyBundle))
          as PreKeyBundle;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use PreKeyBundle() / PreKeyBundle.new instead')
  static PreKeyBundle create() => PreKeyBundle._();
  static $pb.GeneratedMessage $_createMessage() => PreKeyBundle._();
  @$core.override
  PreKeyBundle createEmptyInstance() => PreKeyBundle._();
  @$core.pragma('dart2js:noInline')
  static PreKeyBundle getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<PreKeyBundle>(
          PreKeyBundle.$_createMessage);
  static PreKeyBundle? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<$core.int> get identityEd25519Pubkey => $_getN(0);
  @$pb.TagNumber(1)
  set identityEd25519Pubkey($core.List<$core.int> value) =>
      $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasIdentityEd25519Pubkey() => $_has(0);
  @$pb.TagNumber(1)
  void clearIdentityEd25519Pubkey() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get identityX25519Pubkey => $_getN(1);
  @$pb.TagNumber(2)
  set identityX25519Pubkey($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasIdentityX25519Pubkey() => $_has(1);
  @$pb.TagNumber(2)
  void clearIdentityX25519Pubkey() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get signedPrekey => $_getN(2);
  @$pb.TagNumber(3)
  set signedPrekey($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSignedPrekey() => $_has(2);
  @$pb.TagNumber(3)
  void clearSignedPrekey() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get signedPrekeySignature => $_getN(3);
  @$pb.TagNumber(4)
  set signedPrekeySignature($core.List<$core.int> value) =>
      $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSignedPrekeySignature() => $_has(3);
  @$pb.TagNumber(4)
  void clearSignedPrekeySignature() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbList<$core.List<$core.int>> get oneTimePrekeys => $_getList(4);

  @$pb.TagNumber(6)
  $fixnum.Int64 get prekeyTimestamp => $_getI64(5);
  @$pb.TagNumber(6)
  set prekeyTimestamp($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasPrekeyTimestamp() => $_has(5);
  @$pb.TagNumber(6)
  void clearPrekeyTimestamp() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.int get keyId => $_getIZ(6);
  @$pb.TagNumber(7)
  set keyId($core.int value) => $_setUnsignedInt32(6, value);
  @$pb.TagNumber(7)
  $core.bool hasKeyId() => $_has(6);
  @$pb.TagNumber(7)
  void clearKeyId() => $_clearField(7);

  /// Ключ ноды Yggdrasil, по которому этому контакту реально пишутся пакеты, и
  /// подпись мастер-Ed25519, заверяющая пару «identity-X25519 ↔ этот ключ».
  /// Без второй строки карта контакта — просто два набора байтов: адрес узла
  /// может принадлежать кому угодно, и шифрование честной идентичности ушло бы в
  /// чужой узел. Проверяется при AddContact.
  @$pb.TagNumber(8)
  $core.List<$core.int> get nodeEd25519Pubkey => $_getN(7);
  @$pb.TagNumber(8)
  set nodeEd25519Pubkey($core.List<$core.int> value) => $_setBytes(7, value);
  @$pb.TagNumber(8)
  $core.bool hasNodeEd25519Pubkey() => $_has(7);
  @$pb.TagNumber(8)
  void clearNodeEd25519Pubkey() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.List<$core.int> get nodeKeyProof => $_getN(8);
  @$pb.TagNumber(9)
  set nodeKeyProof($core.List<$core.int> value) => $_setBytes(8, value);
  @$pb.TagNumber(9)
  $core.bool hasNodeKeyProof() => $_has(8);
  @$pb.TagNumber(9)
  void clearNodeKeyProof() => $_clearField(9);
}

/// Запрос чужого бандла поверх оверлея. Без него контакт можно завести только
/// переслав ContactCard байтами вне сети. Request_id связывает ответ с запросом:
/// ответов может быть несколько (узел переспрашивает, дубликат тонет на
/// холодном маршруте), и брать в работу надо тот, что дошёл первым.
class BundleReq extends $pb.GeneratedMessage {
  factory BundleReq({
    $fixnum.Int64? requestId,
  }) {
    final result = BundleReq._();
    if (requestId != null) result.requestId = requestId;
    return result;
  }

  BundleReq._();

  factory BundleReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      BundleReq()..mergeFromBuffer(data, registry);
  factory BundleReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      BundleReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'BundleReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: BundleReq.$_createMessage)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'requestId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BundleReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BundleReq copyWith(void Function(BundleReq) updates) =>
      super.copyWith((message) => updates(message as BundleReq)) as BundleReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use BundleReq() / BundleReq.new instead')
  static BundleReq create() => BundleReq._();
  static $pb.GeneratedMessage $_createMessage() => BundleReq._();
  @$core.override
  BundleReq createEmptyInstance() => BundleReq._();
  @$core.pragma('dart2js:noInline')
  static BundleReq getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<BundleReq>(BundleReq.$_createMessage);
  static BundleReq? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get requestId => $_getI64(0);
  @$pb.TagNumber(1)
  set requestId($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);
}

/// Ответ на bundle_req. Ошибка обязана быть названа словами: «бандлов не
/// раздаём» и «такого ключа у нас нет» для клиента значит разное, и молча
/// промолчать — значит отдать ожидание вместо новости.
///
/// Ответ ничем не заверяется сверх самого бандла: доверие держится на том, что
/// (а) пакет пришёл от узла, чей ed25519-ключ стоит адресом в оверлее, (б)
/// ключ ноды в бандле совпал с запрошенным, (в) карта прошла ворота
/// verifyContact — сертификат ноды от той же идентичности, один и тот же
/// X25519 в карте и в бандле.
class BundleResp extends $pb.GeneratedMessage {
  factory BundleResp({
    $fixnum.Int64? requestId,
    PreKeyBundle? bundle,
    $core.String? error,
  }) {
    final result = BundleResp._();
    if (requestId != null) result.requestId = requestId;
    if (bundle != null) result.bundle = bundle;
    if (error != null) result.error = error;
    return result;
  }

  BundleResp._();

  factory BundleResp.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      BundleResp()..mergeFromBuffer(data, registry);
  factory BundleResp.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      BundleResp()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'BundleResp',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: BundleResp.$_createMessage)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'requestId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<PreKeyBundle>(2, _omitFieldNames ? '' : 'bundle',
        subBuilder: PreKeyBundle.$_createMessage)
    ..aOS(3, _omitFieldNames ? '' : 'error')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BundleResp clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BundleResp copyWith(void Function(BundleResp) updates) =>
      super.copyWith((message) => updates(message as BundleResp)) as BundleResp;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use BundleResp() / BundleResp.new instead')
  static BundleResp create() => BundleResp._();
  static $pb.GeneratedMessage $_createMessage() => BundleResp._();
  @$core.override
  BundleResp createEmptyInstance() => BundleResp._();
  @$core.pragma('dart2js:noInline')
  static BundleResp getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<BundleResp>(BundleResp.$_createMessage);
  static BundleResp? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get requestId => $_getI64(0);
  @$pb.TagNumber(1)
  set requestId($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  PreKeyBundle get bundle => $_getN(1);
  @$pb.TagNumber(2)
  set bundle(PreKeyBundle value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasBundle() => $_has(1);
  @$pb.TagNumber(2)
  void clearBundle() => $_clearField(2);
  @$pb.TagNumber(2)
  PreKeyBundle ensureBundle() => $_ensure(1);

  @$pb.TagNumber(3)
  $core.String get error => $_getSZ(2);
  @$pb.TagNumber(3)
  set error($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasError() => $_has(2);
  @$pb.TagNumber(3)
  void clearError() => $_clearField(3);
}

class TextContent extends $pb.GeneratedMessage {
  factory TextContent({
    $core.String? body,
    $core.Iterable<$core.String>? mentions,
  }) {
    final result = TextContent._();
    if (body != null) result.body = body;
    if (mentions != null) result.mentions.addAll(mentions);
    return result;
  }

  TextContent._();

  factory TextContent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      TextContent()..mergeFromBuffer(data, registry);
  factory TextContent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      TextContent()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TextContent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: TextContent.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'body')
    ..pPS(2, _omitFieldNames ? '' : 'mentions')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TextContent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TextContent copyWith(void Function(TextContent) updates) =>
      super.copyWith((message) => updates(message as TextContent))
          as TextContent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use TextContent() / TextContent.new instead')
  static TextContent create() => TextContent._();
  static $pb.GeneratedMessage $_createMessage() => TextContent._();
  @$core.override
  TextContent createEmptyInstance() => TextContent._();
  @$core.pragma('dart2js:noInline')
  static TextContent getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<TextContent>(
          TextContent.$_createMessage);
  static TextContent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get body => $_getSZ(0);
  @$pb.TagNumber(1)
  set body($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasBody() => $_has(0);
  @$pb.TagNumber(1)
  void clearBody() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<$core.String> get mentions => $_getList(1);
}

class ReplyContent extends $pb.GeneratedMessage {
  factory ReplyContent({
    $core.String? replyToMessageId,
    $core.String? body,
    $core.String? quoteSnippet,
  }) {
    final result = ReplyContent._();
    if (replyToMessageId != null) result.replyToMessageId = replyToMessageId;
    if (body != null) result.body = body;
    if (quoteSnippet != null) result.quoteSnippet = quoteSnippet;
    return result;
  }

  ReplyContent._();

  factory ReplyContent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ReplyContent()..mergeFromBuffer(data, registry);
  factory ReplyContent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ReplyContent()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ReplyContent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: ReplyContent.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'replyToMessageId')
    ..aOS(2, _omitFieldNames ? '' : 'body')
    ..aOS(3, _omitFieldNames ? '' : 'quoteSnippet')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReplyContent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReplyContent copyWith(void Function(ReplyContent) updates) =>
      super.copyWith((message) => updates(message as ReplyContent))
          as ReplyContent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use ReplyContent() / ReplyContent.new instead')
  static ReplyContent create() => ReplyContent._();
  static $pb.GeneratedMessage $_createMessage() => ReplyContent._();
  @$core.override
  ReplyContent createEmptyInstance() => ReplyContent._();
  @$core.pragma('dart2js:noInline')
  static ReplyContent getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ReplyContent>(
          ReplyContent.$_createMessage);
  static ReplyContent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get replyToMessageId => $_getSZ(0);
  @$pb.TagNumber(1)
  set replyToMessageId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasReplyToMessageId() => $_has(0);
  @$pb.TagNumber(1)
  void clearReplyToMessageId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get body => $_getSZ(1);
  @$pb.TagNumber(2)
  set body($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasBody() => $_has(1);
  @$pb.TagNumber(2)
  void clearBody() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get quoteSnippet => $_getSZ(2);
  @$pb.TagNumber(3)
  set quoteSnippet($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasQuoteSnippet() => $_has(2);
  @$pb.TagNumber(3)
  void clearQuoteSnippet() => $_clearField(3);
}

class ReactionContent extends $pb.GeneratedMessage {
  factory ReactionContent({
    $core.String? targetMessageId,
    $core.String? emoji,
    $core.bool? isRemoved,
  }) {
    final result = ReactionContent._();
    if (targetMessageId != null) result.targetMessageId = targetMessageId;
    if (emoji != null) result.emoji = emoji;
    if (isRemoved != null) result.isRemoved = isRemoved;
    return result;
  }

  ReactionContent._();

  factory ReactionContent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ReactionContent()..mergeFromBuffer(data, registry);
  factory ReactionContent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ReactionContent()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ReactionContent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: ReactionContent.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'targetMessageId')
    ..aOS(2, _omitFieldNames ? '' : 'emoji')
    ..aOB(3, _omitFieldNames ? '' : 'isRemoved')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReactionContent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ReactionContent copyWith(void Function(ReactionContent) updates) =>
      super.copyWith((message) => updates(message as ReactionContent))
          as ReactionContent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use ReactionContent() / ReactionContent.new instead')
  static ReactionContent create() => ReactionContent._();
  static $pb.GeneratedMessage $_createMessage() => ReactionContent._();
  @$core.override
  ReactionContent createEmptyInstance() => ReactionContent._();
  @$core.pragma('dart2js:noInline')
  static ReactionContent getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ReactionContent>(
          ReactionContent.$_createMessage);
  static ReactionContent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get targetMessageId => $_getSZ(0);
  @$pb.TagNumber(1)
  set targetMessageId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTargetMessageId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTargetMessageId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get emoji => $_getSZ(1);
  @$pb.TagNumber(2)
  set emoji($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEmoji() => $_has(1);
  @$pb.TagNumber(2)
  void clearEmoji() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get isRemoved => $_getBF(2);
  @$pb.TagNumber(3)
  set isRemoved($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasIsRemoved() => $_has(2);
  @$pb.TagNumber(3)
  void clearIsRemoved() => $_clearField(3);
}

class MediaManifest extends $pb.GeneratedMessage {
  factory MediaManifest({
    $core.String? fileName,
    $core.String? mimeType,
    $fixnum.Int64? fileSize,
    $core.List<$core.int>? sha256Hash,
    $core.List<$core.int>? xchachaKey,
    $core.List<$core.int>? xchachaNonce,
    $core.Iterable<$core.List<$core.int>>? chunkHashes,
    $core.List<$core.int>? thumbnailBytes,
    $core.int? width,
    $core.int? height,
    $core.int? durationSeconds,
  }) {
    final result = MediaManifest._();
    if (fileName != null) result.fileName = fileName;
    if (mimeType != null) result.mimeType = mimeType;
    if (fileSize != null) result.fileSize = fileSize;
    if (sha256Hash != null) result.sha256Hash = sha256Hash;
    if (xchachaKey != null) result.xchachaKey = xchachaKey;
    if (xchachaNonce != null) result.xchachaNonce = xchachaNonce;
    if (chunkHashes != null) result.chunkHashes.addAll(chunkHashes);
    if (thumbnailBytes != null) result.thumbnailBytes = thumbnailBytes;
    if (width != null) result.width = width;
    if (height != null) result.height = height;
    if (durationSeconds != null) result.durationSeconds = durationSeconds;
    return result;
  }

  MediaManifest._();

  factory MediaManifest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      MediaManifest()..mergeFromBuffer(data, registry);
  factory MediaManifest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      MediaManifest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'MediaManifest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: MediaManifest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'fileName')
    ..aOS(2, _omitFieldNames ? '' : 'mimeType')
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'fileSize', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'sha256Hash', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'xchachaKey', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        6, _omitFieldNames ? '' : 'xchachaNonce', $pb.PbFieldType.OY)
    ..p<$core.List<$core.int>>(
        7, _omitFieldNames ? '' : 'chunkHashes', $pb.PbFieldType.PY)
    ..a<$core.List<$core.int>>(
        8, _omitFieldNames ? '' : 'thumbnailBytes', $pb.PbFieldType.OY)
    ..aI(9, _omitFieldNames ? '' : 'width', fieldType: $pb.PbFieldType.OU3)
    ..aI(10, _omitFieldNames ? '' : 'height', fieldType: $pb.PbFieldType.OU3)
    ..aI(11, _omitFieldNames ? '' : 'durationSeconds',
        fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MediaManifest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MediaManifest copyWith(void Function(MediaManifest) updates) =>
      super.copyWith((message) => updates(message as MediaManifest))
          as MediaManifest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use MediaManifest() / MediaManifest.new instead')
  static MediaManifest create() => MediaManifest._();
  static $pb.GeneratedMessage $_createMessage() => MediaManifest._();
  @$core.override
  MediaManifest createEmptyInstance() => MediaManifest._();
  @$core.pragma('dart2js:noInline')
  static MediaManifest getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<MediaManifest>(
          MediaManifest.$_createMessage);
  static MediaManifest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get fileName => $_getSZ(0);
  @$pb.TagNumber(1)
  set fileName($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasFileName() => $_has(0);
  @$pb.TagNumber(1)
  void clearFileName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get mimeType => $_getSZ(1);
  @$pb.TagNumber(2)
  set mimeType($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMimeType() => $_has(1);
  @$pb.TagNumber(2)
  void clearMimeType() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get fileSize => $_getI64(2);
  @$pb.TagNumber(3)
  set fileSize($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasFileSize() => $_has(2);
  @$pb.TagNumber(3)
  void clearFileSize() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get sha256Hash => $_getN(3);
  @$pb.TagNumber(4)
  set sha256Hash($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSha256Hash() => $_has(3);
  @$pb.TagNumber(4)
  void clearSha256Hash() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.List<$core.int> get xchachaKey => $_getN(4);
  @$pb.TagNumber(5)
  set xchachaKey($core.List<$core.int> value) => $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasXchachaKey() => $_has(4);
  @$pb.TagNumber(5)
  void clearXchachaKey() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.List<$core.int> get xchachaNonce => $_getN(5);
  @$pb.TagNumber(6)
  set xchachaNonce($core.List<$core.int> value) => $_setBytes(5, value);
  @$pb.TagNumber(6)
  $core.bool hasXchachaNonce() => $_has(5);
  @$pb.TagNumber(6)
  void clearXchachaNonce() => $_clearField(6);

  @$pb.TagNumber(7)
  $pb.PbList<$core.List<$core.int>> get chunkHashes => $_getList(6);

  @$pb.TagNumber(8)
  $core.List<$core.int> get thumbnailBytes => $_getN(7);
  @$pb.TagNumber(8)
  set thumbnailBytes($core.List<$core.int> value) => $_setBytes(7, value);
  @$pb.TagNumber(8)
  $core.bool hasThumbnailBytes() => $_has(7);
  @$pb.TagNumber(8)
  void clearThumbnailBytes() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.int get width => $_getIZ(8);
  @$pb.TagNumber(9)
  set width($core.int value) => $_setUnsignedInt32(8, value);
  @$pb.TagNumber(9)
  $core.bool hasWidth() => $_has(8);
  @$pb.TagNumber(9)
  void clearWidth() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.int get height => $_getIZ(9);
  @$pb.TagNumber(10)
  set height($core.int value) => $_setUnsignedInt32(9, value);
  @$pb.TagNumber(10)
  $core.bool hasHeight() => $_has(9);
  @$pb.TagNumber(10)
  void clearHeight() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.int get durationSeconds => $_getIZ(10);
  @$pb.TagNumber(11)
  set durationSeconds($core.int value) => $_setUnsignedInt32(10, value);
  @$pb.TagNumber(11)
  $core.bool hasDurationSeconds() => $_has(10);
  @$pb.TagNumber(11)
  void clearDurationSeconds() => $_clearField(11);
}

class CallSignalContent extends $pb.GeneratedMessage {
  factory CallSignalContent({
    $core.String? callId,
    CallSignalType? signalType,
    $core.String? sdpPayload,
    $core.String? iceCandidate,
    $core.bool? isVideo,
  }) {
    final result = CallSignalContent._();
    if (callId != null) result.callId = callId;
    if (signalType != null) result.signalType = signalType;
    if (sdpPayload != null) result.sdpPayload = sdpPayload;
    if (iceCandidate != null) result.iceCandidate = iceCandidate;
    if (isVideo != null) result.isVideo = isVideo;
    return result;
  }

  CallSignalContent._();

  factory CallSignalContent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CallSignalContent()..mergeFromBuffer(data, registry);
  factory CallSignalContent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CallSignalContent()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CallSignalContent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: CallSignalContent.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'callId')
    ..aE<CallSignalType>(2, _omitFieldNames ? '' : 'signalType',
        enumValues: CallSignalType.values)
    ..aOS(3, _omitFieldNames ? '' : 'sdpPayload')
    ..aOS(4, _omitFieldNames ? '' : 'iceCandidate')
    ..aOB(5, _omitFieldNames ? '' : 'isVideo')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CallSignalContent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CallSignalContent copyWith(void Function(CallSignalContent) updates) =>
      super.copyWith((message) => updates(message as CallSignalContent))
          as CallSignalContent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use CallSignalContent() / CallSignalContent.new instead')
  static CallSignalContent create() => CallSignalContent._();
  static $pb.GeneratedMessage $_createMessage() => CallSignalContent._();
  @$core.override
  CallSignalContent createEmptyInstance() => CallSignalContent._();
  @$core.pragma('dart2js:noInline')
  static CallSignalContent getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CallSignalContent>(
          CallSignalContent.$_createMessage);
  static CallSignalContent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get callId => $_getSZ(0);
  @$pb.TagNumber(1)
  set callId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCallId() => $_has(0);
  @$pb.TagNumber(1)
  void clearCallId() => $_clearField(1);

  @$pb.TagNumber(2)
  CallSignalType get signalType => $_getN(1);
  @$pb.TagNumber(2)
  set signalType(CallSignalType value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasSignalType() => $_has(1);
  @$pb.TagNumber(2)
  void clearSignalType() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get sdpPayload => $_getSZ(2);
  @$pb.TagNumber(3)
  set sdpPayload($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSdpPayload() => $_has(2);
  @$pb.TagNumber(3)
  void clearSdpPayload() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get iceCandidate => $_getSZ(3);
  @$pb.TagNumber(4)
  set iceCandidate($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasIceCandidate() => $_has(3);
  @$pb.TagNumber(4)
  void clearIceCandidate() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get isVideo => $_getBF(4);
  @$pb.TagNumber(5)
  set isVideo($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasIsVideo() => $_has(4);
  @$pb.TagNumber(5)
  void clearIsVideo() => $_clearField(5);
}

/// Подтверждение доставки. Идёт тем же сквозным сеансом Double Ratchet, что и
/// обычные сообщения: S&F-релей и оверлей его не читают, а отправитель верит ему,
/// потому что шифротекст мог открыть только получатель.
class DeliveryAckContent extends $pb.GeneratedMessage {
  factory DeliveryAckContent({
    $core.String? acknowledgesMessageId,
  }) {
    final result = DeliveryAckContent._();
    if (acknowledgesMessageId != null)
      result.acknowledgesMessageId = acknowledgesMessageId;
    return result;
  }

  DeliveryAckContent._();

  factory DeliveryAckContent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      DeliveryAckContent()..mergeFromBuffer(data, registry);
  factory DeliveryAckContent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      DeliveryAckContent()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DeliveryAckContent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: DeliveryAckContent.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'acknowledgesMessageId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeliveryAckContent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeliveryAckContent copyWith(void Function(DeliveryAckContent) updates) =>
      super.copyWith((message) => updates(message as DeliveryAckContent))
          as DeliveryAckContent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use DeliveryAckContent() / DeliveryAckContent.new instead')
  static DeliveryAckContent create() => DeliveryAckContent._();
  static $pb.GeneratedMessage $_createMessage() => DeliveryAckContent._();
  @$core.override
  DeliveryAckContent createEmptyInstance() => DeliveryAckContent._();
  @$core.pragma('dart2js:noInline')
  static DeliveryAckContent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DeliveryAckContent>(
          DeliveryAckContent.$_createMessage);
  static DeliveryAckContent? _defaultInstance;

  /// message_id принятого и сохранённого сообщения.
  @$pb.TagNumber(1)
  $core.String get acknowledgesMessageId => $_getSZ(0);
  @$pb.TagNumber(1)
  set acknowledgesMessageId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAcknowledgesMessageId() => $_has(0);
  @$pb.TagNumber(1)
  void clearAcknowledgesMessageId() => $_clearField(1);
}

enum ChatMessage_Content {
  text,
  reply,
  reaction,
  media,
  callSignal,
  deliveryAck,
  notSet
}

class ChatMessage extends $pb.GeneratedMessage {
  factory ChatMessage({
    $core.String? messageId,
    $core.String? conversationId,
    $core.String? senderNodeIpv6,
    $core.List<$core.int>? senderX25519Pubkey,
    $fixnum.Int64? timestampMs,
    $core.bool? isOutbound,
    TextContent? text,
    ReplyContent? reply,
    ReactionContent? reaction,
    MediaManifest? media,
    CallSignalContent? callSignal,
    $core.bool? delivered,
    DeliveryAckContent? deliveryAck,
  }) {
    final result = ChatMessage._();
    if (messageId != null) result.messageId = messageId;
    if (conversationId != null) result.conversationId = conversationId;
    if (senderNodeIpv6 != null) result.senderNodeIpv6 = senderNodeIpv6;
    if (senderX25519Pubkey != null)
      result.senderX25519Pubkey = senderX25519Pubkey;
    if (timestampMs != null) result.timestampMs = timestampMs;
    if (isOutbound != null) result.isOutbound = isOutbound;
    if (text != null) result.text = text;
    if (reply != null) result.reply = reply;
    if (reaction != null) result.reaction = reaction;
    if (media != null) result.media = media;
    if (callSignal != null) result.callSignal = callSignal;
    if (delivered != null) result.delivered = delivered;
    if (deliveryAck != null) result.deliveryAck = deliveryAck;
    return result;
  }

  ChatMessage._();

  factory ChatMessage.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ChatMessage()..mergeFromBuffer(data, registry);
  factory ChatMessage.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ChatMessage()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, ChatMessage_Content>
      _ChatMessage_ContentByTag = {
    7: ChatMessage_Content.text,
    8: ChatMessage_Content.reply,
    9: ChatMessage_Content.reaction,
    10: ChatMessage_Content.media,
    11: ChatMessage_Content.callSignal,
    13: ChatMessage_Content.deliveryAck,
    0: ChatMessage_Content.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ChatMessage',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: ChatMessage.$_createMessage)
    ..oo(0, [7, 8, 9, 10, 11, 13])
    ..aOS(1, _omitFieldNames ? '' : 'messageId')
    ..aOS(2, _omitFieldNames ? '' : 'conversationId')
    ..aOS(3, _omitFieldNames ? '' : 'senderNodeIpv6')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'senderX25519Pubkey', $pb.PbFieldType.OY)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'timestampMs', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(6, _omitFieldNames ? '' : 'isOutbound')
    ..aOM<TextContent>(7, _omitFieldNames ? '' : 'text',
        subBuilder: TextContent.$_createMessage)
    ..aOM<ReplyContent>(8, _omitFieldNames ? '' : 'reply',
        subBuilder: ReplyContent.$_createMessage)
    ..aOM<ReactionContent>(9, _omitFieldNames ? '' : 'reaction',
        subBuilder: ReactionContent.$_createMessage)
    ..aOM<MediaManifest>(10, _omitFieldNames ? '' : 'media',
        subBuilder: MediaManifest.$_createMessage)
    ..aOM<CallSignalContent>(11, _omitFieldNames ? '' : 'callSignal',
        subBuilder: CallSignalContent.$_createMessage)
    ..aOB(12, _omitFieldNames ? '' : 'delivered')
    ..aOM<DeliveryAckContent>(13, _omitFieldNames ? '' : 'deliveryAck',
        subBuilder: DeliveryAckContent.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChatMessage clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ChatMessage copyWith(void Function(ChatMessage) updates) =>
      super.copyWith((message) => updates(message as ChatMessage))
          as ChatMessage;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use ChatMessage() / ChatMessage.new instead')
  static ChatMessage create() => ChatMessage._();
  static $pb.GeneratedMessage $_createMessage() => ChatMessage._();
  @$core.override
  ChatMessage createEmptyInstance() => ChatMessage._();
  @$core.pragma('dart2js:noInline')
  static ChatMessage getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ChatMessage>(
          ChatMessage.$_createMessage);
  static ChatMessage? _defaultInstance;

  @$pb.TagNumber(7)
  @$pb.TagNumber(8)
  @$pb.TagNumber(9)
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(13)
  ChatMessage_Content whichContent() =>
      _ChatMessage_ContentByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(7)
  @$pb.TagNumber(8)
  @$pb.TagNumber(9)
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(13)
  void clearContent() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $core.String get messageId => $_getSZ(0);
  @$pb.TagNumber(1)
  set messageId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMessageId() => $_has(0);
  @$pb.TagNumber(1)
  void clearMessageId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get conversationId => $_getSZ(1);
  @$pb.TagNumber(2)
  set conversationId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasConversationId() => $_has(1);
  @$pb.TagNumber(2)
  void clearConversationId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get senderNodeIpv6 => $_getSZ(2);
  @$pb.TagNumber(3)
  set senderNodeIpv6($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSenderNodeIpv6() => $_has(2);
  @$pb.TagNumber(3)
  void clearSenderNodeIpv6() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get senderX25519Pubkey => $_getN(3);
  @$pb.TagNumber(4)
  set senderX25519Pubkey($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSenderX25519Pubkey() => $_has(3);
  @$pb.TagNumber(4)
  void clearSenderX25519Pubkey() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get timestampMs => $_getI64(4);
  @$pb.TagNumber(5)
  set timestampMs($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasTimestampMs() => $_has(4);
  @$pb.TagNumber(5)
  void clearTimestampMs() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get isOutbound => $_getBF(5);
  @$pb.TagNumber(6)
  set isOutbound($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasIsOutbound() => $_has(5);
  @$pb.TagNumber(6)
  void clearIsOutbound() => $_clearField(6);

  @$pb.TagNumber(7)
  TextContent get text => $_getN(6);
  @$pb.TagNumber(7)
  set text(TextContent value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasText() => $_has(6);
  @$pb.TagNumber(7)
  void clearText() => $_clearField(7);
  @$pb.TagNumber(7)
  TextContent ensureText() => $_ensure(6);

  @$pb.TagNumber(8)
  ReplyContent get reply => $_getN(7);
  @$pb.TagNumber(8)
  set reply(ReplyContent value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasReply() => $_has(7);
  @$pb.TagNumber(8)
  void clearReply() => $_clearField(8);
  @$pb.TagNumber(8)
  ReplyContent ensureReply() => $_ensure(7);

  @$pb.TagNumber(9)
  ReactionContent get reaction => $_getN(8);
  @$pb.TagNumber(9)
  set reaction(ReactionContent value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasReaction() => $_has(8);
  @$pb.TagNumber(9)
  void clearReaction() => $_clearField(9);
  @$pb.TagNumber(9)
  ReactionContent ensureReaction() => $_ensure(8);

  @$pb.TagNumber(10)
  MediaManifest get media => $_getN(9);
  @$pb.TagNumber(10)
  set media(MediaManifest value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasMedia() => $_has(9);
  @$pb.TagNumber(10)
  void clearMedia() => $_clearField(10);
  @$pb.TagNumber(10)
  MediaManifest ensureMedia() => $_ensure(9);

  @$pb.TagNumber(11)
  CallSignalContent get callSignal => $_getN(10);
  @$pb.TagNumber(11)
  set callSignal(CallSignalContent value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasCallSignal() => $_has(10);
  @$pb.TagNumber(11)
  void clearCallSignal() => $_clearField(11);
  @$pb.TagNumber(11)
  CallSignalContent ensureCallSignal() => $_ensure(10);

  /// Сквозное подтверждение получателя (получается из DeliveryAckContent).
  /// Отдельное поле, а не часть oneof: ack — это служебное содержимое, а
  /// «доставлено» — состояние уже показанного сообщения.
  @$pb.TagNumber(12)
  $core.bool get delivered => $_getBF(11);
  @$pb.TagNumber(12)
  set delivered($core.bool value) => $_setBool(11, value);
  @$pb.TagNumber(12)
  $core.bool hasDelivered() => $_has(11);
  @$pb.TagNumber(12)
  void clearDelivered() => $_clearField(12);

  @$pb.TagNumber(13)
  DeliveryAckContent get deliveryAck => $_getN(12);
  @$pb.TagNumber(13)
  set deliveryAck(DeliveryAckContent value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasDeliveryAck() => $_has(12);
  @$pb.TagNumber(13)
  void clearDeliveryAck() => $_clearField(13);
  @$pb.TagNumber(13)
  DeliveryAckContent ensureDeliveryAck() => $_ensure(12);
}

class ContactCard extends $pb.GeneratedMessage {
  factory ContactCard({
    $core.String? nodeIpv6,
    $core.List<$core.int>? ed25519Pubkey,
    $core.List<$core.int>? x25519Pubkey,
    $core.String? nickname,
    $core.String? bio,
    $core.List<$core.int>? avatarHash,
    $fixnum.Int64? updatedAt,
    $core.bool? isVerified,
    PreKeyBundle? bundle,
  }) {
    final result = ContactCard._();
    if (nodeIpv6 != null) result.nodeIpv6 = nodeIpv6;
    if (ed25519Pubkey != null) result.ed25519Pubkey = ed25519Pubkey;
    if (x25519Pubkey != null) result.x25519Pubkey = x25519Pubkey;
    if (nickname != null) result.nickname = nickname;
    if (bio != null) result.bio = bio;
    if (avatarHash != null) result.avatarHash = avatarHash;
    if (updatedAt != null) result.updatedAt = updatedAt;
    if (isVerified != null) result.isVerified = isVerified;
    if (bundle != null) result.bundle = bundle;
    return result;
  }

  ContactCard._();

  factory ContactCard.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ContactCard()..mergeFromBuffer(data, registry);
  factory ContactCard.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ContactCard()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ContactCard',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: ContactCard.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'nodeIpv6')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'ed25519Pubkey', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'x25519Pubkey', $pb.PbFieldType.OY)
    ..aOS(4, _omitFieldNames ? '' : 'nickname')
    ..aOS(5, _omitFieldNames ? '' : 'bio')
    ..a<$core.List<$core.int>>(
        6, _omitFieldNames ? '' : 'avatarHash', $pb.PbFieldType.OY)
    ..a<$fixnum.Int64>(
        7, _omitFieldNames ? '' : 'updatedAt', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(8, _omitFieldNames ? '' : 'isVerified')
    ..aOM<PreKeyBundle>(9, _omitFieldNames ? '' : 'bundle',
        subBuilder: PreKeyBundle.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ContactCard clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ContactCard copyWith(void Function(ContactCard) updates) =>
      super.copyWith((message) => updates(message as ContactCard))
          as ContactCard;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use ContactCard() / ContactCard.new instead')
  static ContactCard create() => ContactCard._();
  static $pb.GeneratedMessage $_createMessage() => ContactCard._();
  @$core.override
  ContactCard createEmptyInstance() => ContactCard._();
  @$core.pragma('dart2js:noInline')
  static ContactCard getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ContactCard>(
          ContactCard.$_createMessage);
  static ContactCard? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get nodeIpv6 => $_getSZ(0);
  @$pb.TagNumber(1)
  set nodeIpv6($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasNodeIpv6() => $_has(0);
  @$pb.TagNumber(1)
  void clearNodeIpv6() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get ed25519Pubkey => $_getN(1);
  @$pb.TagNumber(2)
  set ed25519Pubkey($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEd25519Pubkey() => $_has(1);
  @$pb.TagNumber(2)
  void clearEd25519Pubkey() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get x25519Pubkey => $_getN(2);
  @$pb.TagNumber(3)
  set x25519Pubkey($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasX25519Pubkey() => $_has(2);
  @$pb.TagNumber(3)
  void clearX25519Pubkey() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get nickname => $_getSZ(3);
  @$pb.TagNumber(4)
  set nickname($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasNickname() => $_has(3);
  @$pb.TagNumber(4)
  void clearNickname() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get bio => $_getSZ(4);
  @$pb.TagNumber(5)
  set bio($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasBio() => $_has(4);
  @$pb.TagNumber(5)
  void clearBio() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.List<$core.int> get avatarHash => $_getN(5);
  @$pb.TagNumber(6)
  set avatarHash($core.List<$core.int> value) => $_setBytes(5, value);
  @$pb.TagNumber(6)
  $core.bool hasAvatarHash() => $_has(5);
  @$pb.TagNumber(6)
  void clearAvatarHash() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get updatedAt => $_getI64(6);
  @$pb.TagNumber(7)
  set updatedAt($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasUpdatedAt() => $_has(6);
  @$pb.TagNumber(7)
  void clearUpdatedAt() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.bool get isVerified => $_getBF(7);
  @$pb.TagNumber(8)
  set isVerified($core.bool value) => $_setBool(7, value);
  @$pb.TagNumber(8)
  $core.bool hasIsVerified() => $_has(7);
  @$pb.TagNumber(8)
  void clearIsVerified() => $_clearField(8);

  /// Бандл получателя обязателен для X3DH: без SignedPreKey/OneTimePreKey
  /// отправить сквозное сообщение невозможно.
  @$pb.TagNumber(9)
  PreKeyBundle get bundle => $_getN(8);
  @$pb.TagNumber(9)
  set bundle(PreKeyBundle value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasBundle() => $_has(8);
  @$pb.TagNumber(9)
  void clearBundle() => $_clearField(9);
  @$pb.TagNumber(9)
  PreKeyBundle ensureBundle() => $_ensure(8);
}

class GetPreKeyBundleReq extends $pb.GeneratedMessage {
  factory GetPreKeyBundleReq() => GetPreKeyBundleReq._();

  GetPreKeyBundleReq._();

  factory GetPreKeyBundleReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetPreKeyBundleReq()..mergeFromBuffer(data, registry);
  factory GetPreKeyBundleReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetPreKeyBundleReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetPreKeyBundleReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: GetPreKeyBundleReq.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetPreKeyBundleReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetPreKeyBundleReq copyWith(void Function(GetPreKeyBundleReq) updates) =>
      super.copyWith((message) => updates(message as GetPreKeyBundleReq))
          as GetPreKeyBundleReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use GetPreKeyBundleReq() / GetPreKeyBundleReq.new instead')
  static GetPreKeyBundleReq create() => GetPreKeyBundleReq._();
  static $pb.GeneratedMessage $_createMessage() => GetPreKeyBundleReq._();
  @$core.override
  GetPreKeyBundleReq createEmptyInstance() => GetPreKeyBundleReq._();
  @$core.pragma('dart2js:noInline')
  static GetPreKeyBundleReq getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetPreKeyBundleReq>(
          GetPreKeyBundleReq.$_createMessage);
  static GetPreKeyBundleReq? _defaultInstance;
}

class BackupChunk extends $pb.GeneratedMessage {
  factory BackupChunk({
    $core.String? chunkId,
    $fixnum.Int64? offset,
    $core.int? size,
    $core.List<$core.int>? sha256Hash,
    $core.String? storageProvider,
  }) {
    final result = BackupChunk._();
    if (chunkId != null) result.chunkId = chunkId;
    if (offset != null) result.offset = offset;
    if (size != null) result.size = size;
    if (sha256Hash != null) result.sha256Hash = sha256Hash;
    if (storageProvider != null) result.storageProvider = storageProvider;
    return result;
  }

  BackupChunk._();

  factory BackupChunk.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      BackupChunk()..mergeFromBuffer(data, registry);
  factory BackupChunk.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      BackupChunk()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'BackupChunk',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: BackupChunk.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'chunkId')
    ..a<$fixnum.Int64>(2, _omitFieldNames ? '' : 'offset', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(3, _omitFieldNames ? '' : 'size', fieldType: $pb.PbFieldType.OU3)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'sha256Hash', $pb.PbFieldType.OY)
    ..aOS(5, _omitFieldNames ? '' : 'storageProvider')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BackupChunk clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BackupChunk copyWith(void Function(BackupChunk) updates) =>
      super.copyWith((message) => updates(message as BackupChunk))
          as BackupChunk;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use BackupChunk() / BackupChunk.new instead')
  static BackupChunk create() => BackupChunk._();
  static $pb.GeneratedMessage $_createMessage() => BackupChunk._();
  @$core.override
  BackupChunk createEmptyInstance() => BackupChunk._();
  @$core.pragma('dart2js:noInline')
  static BackupChunk getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<BackupChunk>(
          BackupChunk.$_createMessage);
  static BackupChunk? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get chunkId => $_getSZ(0);
  @$pb.TagNumber(1)
  set chunkId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasChunkId() => $_has(0);
  @$pb.TagNumber(1)
  void clearChunkId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get offset => $_getI64(1);
  @$pb.TagNumber(2)
  set offset($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasOffset() => $_has(1);
  @$pb.TagNumber(2)
  void clearOffset() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get size => $_getIZ(2);
  @$pb.TagNumber(3)
  set size($core.int value) => $_setUnsignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSize() => $_has(2);
  @$pb.TagNumber(3)
  void clearSize() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get sha256Hash => $_getN(3);
  @$pb.TagNumber(4)
  set sha256Hash($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSha256Hash() => $_has(3);
  @$pb.TagNumber(4)
  void clearSha256Hash() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get storageProvider => $_getSZ(4);
  @$pb.TagNumber(5)
  set storageProvider($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasStorageProvider() => $_has(4);
  @$pb.TagNumber(5)
  void clearStorageProvider() => $_clearField(5);
}

class BackupManifest extends $pb.GeneratedMessage {
  factory BackupManifest({
    $fixnum.Int64? backupId,
    $fixnum.Int64? createdAt,
    $fixnum.Int64? totalBytes,
    $core.int? chunkCount,
    $core.List<$core.int>? rootMerkleHash,
    $core.List<$core.int>? argon2idSalt,
    $core.String? compressionAlgo,
    $core.String? encryptionAlgo,
    $core.Iterable<BackupChunk>? chunks,
  }) {
    final result = BackupManifest._();
    if (backupId != null) result.backupId = backupId;
    if (createdAt != null) result.createdAt = createdAt;
    if (totalBytes != null) result.totalBytes = totalBytes;
    if (chunkCount != null) result.chunkCount = chunkCount;
    if (rootMerkleHash != null) result.rootMerkleHash = rootMerkleHash;
    if (argon2idSalt != null) result.argon2idSalt = argon2idSalt;
    if (compressionAlgo != null) result.compressionAlgo = compressionAlgo;
    if (encryptionAlgo != null) result.encryptionAlgo = encryptionAlgo;
    if (chunks != null) result.chunks.addAll(chunks);
    return result;
  }

  BackupManifest._();

  factory BackupManifest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      BackupManifest()..mergeFromBuffer(data, registry);
  factory BackupManifest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      BackupManifest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'BackupManifest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: BackupManifest.$_createMessage)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'backupId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'createdAt', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'totalBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(4, _omitFieldNames ? '' : 'chunkCount', fieldType: $pb.PbFieldType.OU3)
    ..a<$core.List<$core.int>>(
        5, _omitFieldNames ? '' : 'rootMerkleHash', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        6, _omitFieldNames ? '' : 'argon2idSalt', $pb.PbFieldType.OY)
    ..aOS(7, _omitFieldNames ? '' : 'compressionAlgo')
    ..aOS(8, _omitFieldNames ? '' : 'encryptionAlgo')
    ..pPM<BackupChunk>(9, _omitFieldNames ? '' : 'chunks',
        subBuilder: BackupChunk.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BackupManifest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BackupManifest copyWith(void Function(BackupManifest) updates) =>
      super.copyWith((message) => updates(message as BackupManifest))
          as BackupManifest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use BackupManifest() / BackupManifest.new instead')
  static BackupManifest create() => BackupManifest._();
  static $pb.GeneratedMessage $_createMessage() => BackupManifest._();
  @$core.override
  BackupManifest createEmptyInstance() => BackupManifest._();
  @$core.pragma('dart2js:noInline')
  static BackupManifest getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<BackupManifest>(
          BackupManifest.$_createMessage);
  static BackupManifest? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get backupId => $_getI64(0);
  @$pb.TagNumber(1)
  set backupId($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasBackupId() => $_has(0);
  @$pb.TagNumber(1)
  void clearBackupId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get createdAt => $_getI64(1);
  @$pb.TagNumber(2)
  set createdAt($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCreatedAt() => $_has(1);
  @$pb.TagNumber(2)
  void clearCreatedAt() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get totalBytes => $_getI64(2);
  @$pb.TagNumber(3)
  set totalBytes($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTotalBytes() => $_has(2);
  @$pb.TagNumber(3)
  void clearTotalBytes() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get chunkCount => $_getIZ(3);
  @$pb.TagNumber(4)
  set chunkCount($core.int value) => $_setUnsignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasChunkCount() => $_has(3);
  @$pb.TagNumber(4)
  void clearChunkCount() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.List<$core.int> get rootMerkleHash => $_getN(4);
  @$pb.TagNumber(5)
  set rootMerkleHash($core.List<$core.int> value) => $_setBytes(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRootMerkleHash() => $_has(4);
  @$pb.TagNumber(5)
  void clearRootMerkleHash() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.List<$core.int> get argon2idSalt => $_getN(5);
  @$pb.TagNumber(6)
  set argon2idSalt($core.List<$core.int> value) => $_setBytes(5, value);
  @$pb.TagNumber(6)
  $core.bool hasArgon2idSalt() => $_has(5);
  @$pb.TagNumber(6)
  void clearArgon2idSalt() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get compressionAlgo => $_getSZ(6);
  @$pb.TagNumber(7)
  set compressionAlgo($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasCompressionAlgo() => $_has(6);
  @$pb.TagNumber(7)
  void clearCompressionAlgo() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get encryptionAlgo => $_getSZ(7);
  @$pb.TagNumber(8)
  set encryptionAlgo($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasEncryptionAlgo() => $_has(7);
  @$pb.TagNumber(8)
  void clearEncryptionAlgo() => $_clearField(8);

  @$pb.TagNumber(9)
  $pb.PbList<BackupChunk> get chunks => $_getList(8);
}

class StartNodeReq extends $pb.GeneratedMessage {
  factory StartNodeReq({
    NodeConfig? config,
  }) {
    final result = StartNodeReq._();
    if (config != null) result.config = config;
    return result;
  }

  StartNodeReq._();

  factory StartNodeReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StartNodeReq()..mergeFromBuffer(data, registry);
  factory StartNodeReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StartNodeReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StartNodeReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: StartNodeReq.$_createMessage)
    ..aOM<NodeConfig>(1, _omitFieldNames ? '' : 'config',
        subBuilder: NodeConfig.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartNodeReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartNodeReq copyWith(void Function(StartNodeReq) updates) =>
      super.copyWith((message) => updates(message as StartNodeReq))
          as StartNodeReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use StartNodeReq() / StartNodeReq.new instead')
  static StartNodeReq create() => StartNodeReq._();
  static $pb.GeneratedMessage $_createMessage() => StartNodeReq._();
  @$core.override
  StartNodeReq createEmptyInstance() => StartNodeReq._();
  @$core.pragma('dart2js:noInline')
  static StartNodeReq getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<StartNodeReq>(
          StartNodeReq.$_createMessage);
  static StartNodeReq? _defaultInstance;

  @$pb.TagNumber(1)
  NodeConfig get config => $_getN(0);
  @$pb.TagNumber(1)
  set config(NodeConfig value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasConfig() => $_has(0);
  @$pb.TagNumber(1)
  void clearConfig() => $_clearField(1);
  @$pb.TagNumber(1)
  NodeConfig ensureConfig() => $_ensure(0);
}

class StopNodeReq extends $pb.GeneratedMessage {
  factory StopNodeReq({
    $core.bool? force,
  }) {
    final result = StopNodeReq._();
    if (force != null) result.force = force;
    return result;
  }

  StopNodeReq._();

  factory StopNodeReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StopNodeReq()..mergeFromBuffer(data, registry);
  factory StopNodeReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StopNodeReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StopNodeReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: StopNodeReq.$_createMessage)
    ..aOB(1, _omitFieldNames ? '' : 'force')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopNodeReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopNodeReq copyWith(void Function(StopNodeReq) updates) =>
      super.copyWith((message) => updates(message as StopNodeReq))
          as StopNodeReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use StopNodeReq() / StopNodeReq.new instead')
  static StopNodeReq create() => StopNodeReq._();
  static $pb.GeneratedMessage $_createMessage() => StopNodeReq._();
  @$core.override
  StopNodeReq createEmptyInstance() => StopNodeReq._();
  @$core.pragma('dart2js:noInline')
  static StopNodeReq getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<StopNodeReq>(
          StopNodeReq.$_createMessage);
  static StopNodeReq? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get force => $_getBF(0);
  @$pb.TagNumber(1)
  set force($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasForce() => $_has(0);
  @$pb.TagNumber(1)
  void clearForce() => $_clearField(1);
}

class SendMessageReq extends $pb.GeneratedMessage {
  factory SendMessageReq({
    $core.String? destinationIpv6,
    $core.List<$core.int>? destinationX25519Pubkey,
    ChatMessage? message,
  }) {
    final result = SendMessageReq._();
    if (destinationIpv6 != null) result.destinationIpv6 = destinationIpv6;
    if (destinationX25519Pubkey != null)
      result.destinationX25519Pubkey = destinationX25519Pubkey;
    if (message != null) result.message = message;
    return result;
  }

  SendMessageReq._();

  factory SendMessageReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SendMessageReq()..mergeFromBuffer(data, registry);
  factory SendMessageReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SendMessageReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SendMessageReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: SendMessageReq.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'destinationIpv6')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'destinationX25519Pubkey', $pb.PbFieldType.OY)
    ..aOM<ChatMessage>(3, _omitFieldNames ? '' : 'message',
        subBuilder: ChatMessage.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SendMessageReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SendMessageReq copyWith(void Function(SendMessageReq) updates) =>
      super.copyWith((message) => updates(message as SendMessageReq))
          as SendMessageReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use SendMessageReq() / SendMessageReq.new instead')
  static SendMessageReq create() => SendMessageReq._();
  static $pb.GeneratedMessage $_createMessage() => SendMessageReq._();
  @$core.override
  SendMessageReq createEmptyInstance() => SendMessageReq._();
  @$core.pragma('dart2js:noInline')
  static SendMessageReq getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SendMessageReq>(
          SendMessageReq.$_createMessage);
  static SendMessageReq? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get destinationIpv6 => $_getSZ(0);
  @$pb.TagNumber(1)
  set destinationIpv6($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDestinationIpv6() => $_has(0);
  @$pb.TagNumber(1)
  void clearDestinationIpv6() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get destinationX25519Pubkey => $_getN(1);
  @$pb.TagNumber(2)
  set destinationX25519Pubkey($core.List<$core.int> value) =>
      $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDestinationX25519Pubkey() => $_has(1);
  @$pb.TagNumber(2)
  void clearDestinationX25519Pubkey() => $_clearField(2);

  @$pb.TagNumber(3)
  ChatMessage get message => $_getN(2);
  @$pb.TagNumber(3)
  set message(ChatMessage value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasMessage() => $_has(2);
  @$pb.TagNumber(3)
  void clearMessage() => $_clearField(3);
  @$pb.TagNumber(3)
  ChatMessage ensureMessage() => $_ensure(2);
}

class ConnectPeerReq extends $pb.GeneratedMessage {
  factory ConnectPeerReq({
    $core.String? peerUri,
    PeerTransportType? transport,
    $core.String? customSni,
  }) {
    final result = ConnectPeerReq._();
    if (peerUri != null) result.peerUri = peerUri;
    if (transport != null) result.transport = transport;
    if (customSni != null) result.customSni = customSni;
    return result;
  }

  ConnectPeerReq._();

  factory ConnectPeerReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ConnectPeerReq()..mergeFromBuffer(data, registry);
  factory ConnectPeerReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ConnectPeerReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConnectPeerReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: ConnectPeerReq.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'peerUri')
    ..aE<PeerTransportType>(2, _omitFieldNames ? '' : 'transport',
        enumValues: PeerTransportType.values)
    ..aOS(3, _omitFieldNames ? '' : 'customSni')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectPeerReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectPeerReq copyWith(void Function(ConnectPeerReq) updates) =>
      super.copyWith((message) => updates(message as ConnectPeerReq))
          as ConnectPeerReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use ConnectPeerReq() / ConnectPeerReq.new instead')
  static ConnectPeerReq create() => ConnectPeerReq._();
  static $pb.GeneratedMessage $_createMessage() => ConnectPeerReq._();
  @$core.override
  ConnectPeerReq createEmptyInstance() => ConnectPeerReq._();
  @$core.pragma('dart2js:noInline')
  static ConnectPeerReq getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ConnectPeerReq>(
          ConnectPeerReq.$_createMessage);
  static ConnectPeerReq? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get peerUri => $_getSZ(0);
  @$pb.TagNumber(1)
  set peerUri($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPeerUri() => $_has(0);
  @$pb.TagNumber(1)
  void clearPeerUri() => $_clearField(1);

  @$pb.TagNumber(2)
  PeerTransportType get transport => $_getN(1);
  @$pb.TagNumber(2)
  set transport(PeerTransportType value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTransport() => $_has(1);
  @$pb.TagNumber(2)
  void clearTransport() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get customSni => $_getSZ(2);
  @$pb.TagNumber(3)
  set customSni($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCustomSni() => $_has(2);
  @$pb.TagNumber(3)
  void clearCustomSni() => $_clearField(3);
}

class DisconnectPeerReq extends $pb.GeneratedMessage {
  factory DisconnectPeerReq({
    $core.String? peerUri,
  }) {
    final result = DisconnectPeerReq._();
    if (peerUri != null) result.peerUri = peerUri;
    return result;
  }

  DisconnectPeerReq._();

  factory DisconnectPeerReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      DisconnectPeerReq()..mergeFromBuffer(data, registry);
  factory DisconnectPeerReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      DisconnectPeerReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DisconnectPeerReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: DisconnectPeerReq.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'peerUri')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DisconnectPeerReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DisconnectPeerReq copyWith(void Function(DisconnectPeerReq) updates) =>
      super.copyWith((message) => updates(message as DisconnectPeerReq))
          as DisconnectPeerReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use DisconnectPeerReq() / DisconnectPeerReq.new instead')
  static DisconnectPeerReq create() => DisconnectPeerReq._();
  static $pb.GeneratedMessage $_createMessage() => DisconnectPeerReq._();
  @$core.override
  DisconnectPeerReq createEmptyInstance() => DisconnectPeerReq._();
  @$core.pragma('dart2js:noInline')
  static DisconnectPeerReq getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<DisconnectPeerReq>(
          DisconnectPeerReq.$_createMessage);
  static DisconnectPeerReq? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get peerUri => $_getSZ(0);
  @$pb.TagNumber(1)
  set peerUri($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPeerUri() => $_has(0);
  @$pb.TagNumber(1)
  void clearPeerUri() => $_clearField(1);
}

class GetPeersReq extends $pb.GeneratedMessage {
  factory GetPeersReq() => GetPeersReq._();

  GetPeersReq._();

  factory GetPeersReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetPeersReq()..mergeFromBuffer(data, registry);
  factory GetPeersReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetPeersReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetPeersReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: GetPeersReq.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetPeersReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetPeersReq copyWith(void Function(GetPeersReq) updates) =>
      super.copyWith((message) => updates(message as GetPeersReq))
          as GetPeersReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use GetPeersReq() / GetPeersReq.new instead')
  static GetPeersReq create() => GetPeersReq._();
  static $pb.GeneratedMessage $_createMessage() => GetPeersReq._();
  @$core.override
  GetPeersReq createEmptyInstance() => GetPeersReq._();
  @$core.pragma('dart2js:noInline')
  static GetPeersReq getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetPeersReq>(
          GetPeersReq.$_createMessage);
  static GetPeersReq? _defaultInstance;
}

class GetNodeStatusReq extends $pb.GeneratedMessage {
  factory GetNodeStatusReq() => GetNodeStatusReq._();

  GetNodeStatusReq._();

  factory GetNodeStatusReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetNodeStatusReq()..mergeFromBuffer(data, registry);
  factory GetNodeStatusReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetNodeStatusReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetNodeStatusReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: GetNodeStatusReq.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetNodeStatusReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetNodeStatusReq copyWith(void Function(GetNodeStatusReq) updates) =>
      super.copyWith((message) => updates(message as GetNodeStatusReq))
          as GetNodeStatusReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use GetNodeStatusReq() / GetNodeStatusReq.new instead')
  static GetNodeStatusReq create() => GetNodeStatusReq._();
  static $pb.GeneratedMessage $_createMessage() => GetNodeStatusReq._();
  @$core.override
  GetNodeStatusReq createEmptyInstance() => GetNodeStatusReq._();
  @$core.pragma('dart2js:noInline')
  static GetNodeStatusReq getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetNodeStatusReq>(
          GetNodeStatusReq.$_createMessage);
  static GetNodeStatusReq? _defaultInstance;
}

class CreateBackupReq extends $pb.GeneratedMessage {
  factory CreateBackupReq({
    $core.String? destinationTarget,
    $core.String? encryptionPassphrase,
  }) {
    final result = CreateBackupReq._();
    if (destinationTarget != null) result.destinationTarget = destinationTarget;
    if (encryptionPassphrase != null)
      result.encryptionPassphrase = encryptionPassphrase;
    return result;
  }

  CreateBackupReq._();

  factory CreateBackupReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CreateBackupReq()..mergeFromBuffer(data, registry);
  factory CreateBackupReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CreateBackupReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateBackupReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: CreateBackupReq.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'destinationTarget')
    ..aOS(2, _omitFieldNames ? '' : 'encryptionPassphrase')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateBackupReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateBackupReq copyWith(void Function(CreateBackupReq) updates) =>
      super.copyWith((message) => updates(message as CreateBackupReq))
          as CreateBackupReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use CreateBackupReq() / CreateBackupReq.new instead')
  static CreateBackupReq create() => CreateBackupReq._();
  static $pb.GeneratedMessage $_createMessage() => CreateBackupReq._();
  @$core.override
  CreateBackupReq createEmptyInstance() => CreateBackupReq._();
  @$core.pragma('dart2js:noInline')
  static CreateBackupReq getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CreateBackupReq>(
          CreateBackupReq.$_createMessage);
  static CreateBackupReq? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get destinationTarget => $_getSZ(0);
  @$pb.TagNumber(1)
  set destinationTarget($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDestinationTarget() => $_has(0);
  @$pb.TagNumber(1)
  void clearDestinationTarget() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get encryptionPassphrase => $_getSZ(1);
  @$pb.TagNumber(2)
  set encryptionPassphrase($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEncryptionPassphrase() => $_has(1);
  @$pb.TagNumber(2)
  void clearEncryptionPassphrase() => $_clearField(2);
}

class RestoreBackupReq extends $pb.GeneratedMessage {
  factory RestoreBackupReq({
    $core.String? sourceTarget,
    $core.String? encryptionPassphrase,
  }) {
    final result = RestoreBackupReq._();
    if (sourceTarget != null) result.sourceTarget = sourceTarget;
    if (encryptionPassphrase != null)
      result.encryptionPassphrase = encryptionPassphrase;
    return result;
  }

  RestoreBackupReq._();

  factory RestoreBackupReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RestoreBackupReq()..mergeFromBuffer(data, registry);
  factory RestoreBackupReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RestoreBackupReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RestoreBackupReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: RestoreBackupReq.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'sourceTarget')
    ..aOS(2, _omitFieldNames ? '' : 'encryptionPassphrase')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RestoreBackupReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RestoreBackupReq copyWith(void Function(RestoreBackupReq) updates) =>
      super.copyWith((message) => updates(message as RestoreBackupReq))
          as RestoreBackupReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use RestoreBackupReq() / RestoreBackupReq.new instead')
  static RestoreBackupReq create() => RestoreBackupReq._();
  static $pb.GeneratedMessage $_createMessage() => RestoreBackupReq._();
  @$core.override
  RestoreBackupReq createEmptyInstance() => RestoreBackupReq._();
  @$core.pragma('dart2js:noInline')
  static RestoreBackupReq getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<RestoreBackupReq>(
          RestoreBackupReq.$_createMessage);
  static RestoreBackupReq? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get sourceTarget => $_getSZ(0);
  @$pb.TagNumber(1)
  set sourceTarget($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSourceTarget() => $_has(0);
  @$pb.TagNumber(1)
  void clearSourceTarget() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get encryptionPassphrase => $_getSZ(1);
  @$pb.TagNumber(2)
  set encryptionPassphrase($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEncryptionPassphrase() => $_has(1);
  @$pb.TagNumber(2)
  void clearEncryptionPassphrase() => $_clearField(2);
}

class AddContactReq extends $pb.GeneratedMessage {
  factory AddContactReq({
    ContactCard? contact,
  }) {
    final result = AddContactReq._();
    if (contact != null) result.contact = contact;
    return result;
  }

  AddContactReq._();

  factory AddContactReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      AddContactReq()..mergeFromBuffer(data, registry);
  factory AddContactReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      AddContactReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AddContactReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: AddContactReq.$_createMessage)
    ..aOM<ContactCard>(1, _omitFieldNames ? '' : 'contact',
        subBuilder: ContactCard.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AddContactReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AddContactReq copyWith(void Function(AddContactReq) updates) =>
      super.copyWith((message) => updates(message as AddContactReq))
          as AddContactReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use AddContactReq() / AddContactReq.new instead')
  static AddContactReq create() => AddContactReq._();
  static $pb.GeneratedMessage $_createMessage() => AddContactReq._();
  @$core.override
  AddContactReq createEmptyInstance() => AddContactReq._();
  @$core.pragma('dart2js:noInline')
  static AddContactReq getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<AddContactReq>(
          AddContactReq.$_createMessage);
  static AddContactReq? _defaultInstance;

  @$pb.TagNumber(1)
  ContactCard get contact => $_getN(0);
  @$pb.TagNumber(1)
  set contact(ContactCard value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasContact() => $_has(0);
  @$pb.TagNumber(1)
  void clearContact() => $_clearField(1);
  @$pb.TagNumber(1)
  ContactCard ensureContact() => $_ensure(0);
}

/// Генерация фразы делает ядро: entropy и словарь должны остаться на одной
/// стороне границы вместе с производными от них ключами.
class GenerateIdentityReq extends $pb.GeneratedMessage {
  factory GenerateIdentityReq() => GenerateIdentityReq._();

  GenerateIdentityReq._();

  factory GenerateIdentityReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GenerateIdentityReq()..mergeFromBuffer(data, registry);
  factory GenerateIdentityReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GenerateIdentityReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GenerateIdentityReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: GenerateIdentityReq.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GenerateIdentityReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GenerateIdentityReq copyWith(void Function(GenerateIdentityReq) updates) =>
      super.copyWith((message) => updates(message as GenerateIdentityReq))
          as GenerateIdentityReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core
      .Deprecated('Use GenerateIdentityReq() / GenerateIdentityReq.new instead')
  static GenerateIdentityReq create() => GenerateIdentityReq._();
  static $pb.GeneratedMessage $_createMessage() => GenerateIdentityReq._();
  @$core.override
  GenerateIdentityReq createEmptyInstance() => GenerateIdentityReq._();
  @$core.pragma('dart2js:noInline')
  static GenerateIdentityReq getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GenerateIdentityReq>(
          GenerateIdentityReq.$_createMessage);
  static GenerateIdentityReq? _defaultInstance;
}

class IdentityResp extends $pb.GeneratedMessage {
  factory IdentityResp({
    $core.String? mnemonic,
  }) {
    final result = IdentityResp._();
    if (mnemonic != null) result.mnemonic = mnemonic;
    return result;
  }

  IdentityResp._();

  factory IdentityResp.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      IdentityResp()..mergeFromBuffer(data, registry);
  factory IdentityResp.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      IdentityResp()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'IdentityResp',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: IdentityResp.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'mnemonic')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  IdentityResp clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  IdentityResp copyWith(void Function(IdentityResp) updates) =>
      super.copyWith((message) => updates(message as IdentityResp))
          as IdentityResp;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use IdentityResp() / IdentityResp.new instead')
  static IdentityResp create() => IdentityResp._();
  static $pb.GeneratedMessage $_createMessage() => IdentityResp._();
  @$core.override
  IdentityResp createEmptyInstance() => IdentityResp._();
  @$core.pragma('dart2js:noInline')
  static IdentityResp getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<IdentityResp>(
          IdentityResp.$_createMessage);
  static IdentityResp? _defaultInstance;

  /// Ровно те же байты, что потом принимает mnemonic_seed: показывать и хранить
  /// надо эту строку, а не её пересобранный где-то вариант.
  @$pb.TagNumber(1)
  $core.String get mnemonic => $_getSZ(0);
  @$pb.TagNumber(1)
  set mnemonic($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMnemonic() => $_has(0);
  @$pb.TagNumber(1)
  void clearMnemonic() => $_clearField(1);
}

/// Запросы чтения для UI: список строится по реальной зашифрованной базе,
/// а не по демо-константам в коде клиента.
class ConversationInfo extends $pb.GeneratedMessage {
  factory ConversationInfo({
    $core.String? conversationId,
    $core.String? lastMessage,
    $fixnum.Int64? lastTimestampMs,
    $core.int? messageCount,
    $core.int? unreadCount,
  }) {
    final result = ConversationInfo._();
    if (conversationId != null) result.conversationId = conversationId;
    if (lastMessage != null) result.lastMessage = lastMessage;
    if (lastTimestampMs != null) result.lastTimestampMs = lastTimestampMs;
    if (messageCount != null) result.messageCount = messageCount;
    if (unreadCount != null) result.unreadCount = unreadCount;
    return result;
  }

  ConversationInfo._();

  factory ConversationInfo.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ConversationInfo()..mergeFromBuffer(data, registry);
  factory ConversationInfo.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ConversationInfo()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConversationInfo',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: ConversationInfo.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'conversationId')
    ..aOS(2, _omitFieldNames ? '' : 'lastMessage')
    ..aInt64(3, _omitFieldNames ? '' : 'lastTimestampMs')
    ..aI(4, _omitFieldNames ? '' : 'messageCount',
        fieldType: $pb.PbFieldType.OU3)
    ..aI(5, _omitFieldNames ? '' : 'unreadCount',
        fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConversationInfo clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConversationInfo copyWith(void Function(ConversationInfo) updates) =>
      super.copyWith((message) => updates(message as ConversationInfo))
          as ConversationInfo;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use ConversationInfo() / ConversationInfo.new instead')
  static ConversationInfo create() => ConversationInfo._();
  static $pb.GeneratedMessage $_createMessage() => ConversationInfo._();
  @$core.override
  ConversationInfo createEmptyInstance() => ConversationInfo._();
  @$core.pragma('dart2js:noInline')
  static ConversationInfo getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ConversationInfo>(
          ConversationInfo.$_createMessage);
  static ConversationInfo? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get conversationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set conversationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasConversationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearConversationId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get lastMessage => $_getSZ(1);
  @$pb.TagNumber(2)
  set lastMessage($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLastMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearLastMessage() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get lastTimestampMs => $_getI64(2);
  @$pb.TagNumber(3)
  set lastTimestampMs($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasLastTimestampMs() => $_has(2);
  @$pb.TagNumber(3)
  void clearLastTimestampMs() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get messageCount => $_getIZ(3);
  @$pb.TagNumber(4)
  set messageCount($core.int value) => $_setUnsignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasMessageCount() => $_has(3);
  @$pb.TagNumber(4)
  void clearMessageCount() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get unreadCount => $_getIZ(4);
  @$pb.TagNumber(5)
  set unreadCount($core.int value) => $_setUnsignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasUnreadCount() => $_has(4);
  @$pb.TagNumber(5)
  void clearUnreadCount() => $_clearField(5);
}

class ListConversationsReq extends $pb.GeneratedMessage {
  factory ListConversationsReq({
    $core.int? limit,
  }) {
    final result = ListConversationsReq._();
    if (limit != null) result.limit = limit;
    return result;
  }

  ListConversationsReq._();

  factory ListConversationsReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListConversationsReq()..mergeFromBuffer(data, registry);
  factory ListConversationsReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListConversationsReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListConversationsReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: ListConversationsReq.$_createMessage)
    ..aI(1, _omitFieldNames ? '' : 'limit', fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListConversationsReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListConversationsReq copyWith(void Function(ListConversationsReq) updates) =>
      super.copyWith((message) => updates(message as ListConversationsReq))
          as ListConversationsReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ListConversationsReq() / ListConversationsReq.new instead')
  static ListConversationsReq create() => ListConversationsReq._();
  static $pb.GeneratedMessage $_createMessage() => ListConversationsReq._();
  @$core.override
  ListConversationsReq createEmptyInstance() => ListConversationsReq._();
  @$core.pragma('dart2js:noInline')
  static ListConversationsReq getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListConversationsReq>(
          ListConversationsReq.$_createMessage);
  static ListConversationsReq? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get limit => $_getIZ(0);
  @$pb.TagNumber(1)
  set limit($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasLimit() => $_has(0);
  @$pb.TagNumber(1)
  void clearLimit() => $_clearField(1);
}

class ConversationListResp extends $pb.GeneratedMessage {
  factory ConversationListResp({
    $core.Iterable<ConversationInfo>? conversations,
  }) {
    final result = ConversationListResp._();
    if (conversations != null) result.conversations.addAll(conversations);
    return result;
  }

  ConversationListResp._();

  factory ConversationListResp.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ConversationListResp()..mergeFromBuffer(data, registry);
  factory ConversationListResp.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ConversationListResp()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConversationListResp',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: ConversationListResp.$_createMessage)
    ..pPM<ConversationInfo>(1, _omitFieldNames ? '' : 'conversations',
        subBuilder: ConversationInfo.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConversationListResp clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConversationListResp copyWith(void Function(ConversationListResp) updates) =>
      super.copyWith((message) => updates(message as ConversationListResp))
          as ConversationListResp;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ConversationListResp() / ConversationListResp.new instead')
  static ConversationListResp create() => ConversationListResp._();
  static $pb.GeneratedMessage $_createMessage() => ConversationListResp._();
  @$core.override
  ConversationListResp createEmptyInstance() => ConversationListResp._();
  @$core.pragma('dart2js:noInline')
  static ConversationListResp getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConversationListResp>(
          ConversationListResp.$_createMessage);
  static ConversationListResp? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<ConversationInfo> get conversations => $_getList(0);
}

class GetMessagesReq extends $pb.GeneratedMessage {
  factory GetMessagesReq({
    $core.String? conversationId,
    $core.int? limit,
  }) {
    final result = GetMessagesReq._();
    if (conversationId != null) result.conversationId = conversationId;
    if (limit != null) result.limit = limit;
    return result;
  }

  GetMessagesReq._();

  factory GetMessagesReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetMessagesReq()..mergeFromBuffer(data, registry);
  factory GetMessagesReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetMessagesReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetMessagesReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: GetMessagesReq.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'conversationId')
    ..aI(2, _omitFieldNames ? '' : 'limit', fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetMessagesReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetMessagesReq copyWith(void Function(GetMessagesReq) updates) =>
      super.copyWith((message) => updates(message as GetMessagesReq))
          as GetMessagesReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use GetMessagesReq() / GetMessagesReq.new instead')
  static GetMessagesReq create() => GetMessagesReq._();
  static $pb.GeneratedMessage $_createMessage() => GetMessagesReq._();
  @$core.override
  GetMessagesReq createEmptyInstance() => GetMessagesReq._();
  @$core.pragma('dart2js:noInline')
  static GetMessagesReq getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetMessagesReq>(
          GetMessagesReq.$_createMessage);
  static GetMessagesReq? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get conversationId => $_getSZ(0);
  @$pb.TagNumber(1)
  set conversationId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasConversationId() => $_has(0);
  @$pb.TagNumber(1)
  void clearConversationId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get limit => $_getIZ(1);
  @$pb.TagNumber(2)
  set limit($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLimit() => $_has(1);
  @$pb.TagNumber(2)
  void clearLimit() => $_clearField(2);
}

class MessageListResp extends $pb.GeneratedMessage {
  factory MessageListResp({
    $core.Iterable<ChatMessage>? messages,
  }) {
    final result = MessageListResp._();
    if (messages != null) result.messages.addAll(messages);
    return result;
  }

  MessageListResp._();

  factory MessageListResp.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      MessageListResp()..mergeFromBuffer(data, registry);
  factory MessageListResp.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      MessageListResp()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'MessageListResp',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: MessageListResp.$_createMessage)
    ..pPM<ChatMessage>(1, _omitFieldNames ? '' : 'messages',
        subBuilder: ChatMessage.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MessageListResp clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MessageListResp copyWith(void Function(MessageListResp) updates) =>
      super.copyWith((message) => updates(message as MessageListResp))
          as MessageListResp;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use MessageListResp() / MessageListResp.new instead')
  static MessageListResp create() => MessageListResp._();
  static $pb.GeneratedMessage $_createMessage() => MessageListResp._();
  @$core.override
  MessageListResp createEmptyInstance() => MessageListResp._();
  @$core.pragma('dart2js:noInline')
  static MessageListResp getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<MessageListResp>(
          MessageListResp.$_createMessage);
  static MessageListResp? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<ChatMessage> get messages => $_getList(0);
}

class ListContactsReq extends $pb.GeneratedMessage {
  factory ListContactsReq() => ListContactsReq._();

  ListContactsReq._();

  factory ListContactsReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListContactsReq()..mergeFromBuffer(data, registry);
  factory ListContactsReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListContactsReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListContactsReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: ListContactsReq.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListContactsReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListContactsReq copyWith(void Function(ListContactsReq) updates) =>
      super.copyWith((message) => updates(message as ListContactsReq))
          as ListContactsReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use ListContactsReq() / ListContactsReq.new instead')
  static ListContactsReq create() => ListContactsReq._();
  static $pb.GeneratedMessage $_createMessage() => ListContactsReq._();
  @$core.override
  ListContactsReq createEmptyInstance() => ListContactsReq._();
  @$core.pragma('dart2js:noInline')
  static ListContactsReq getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ListContactsReq>(
          ListContactsReq.$_createMessage);
  static ListContactsReq? _defaultInstance;
}

class ContactListResp extends $pb.GeneratedMessage {
  factory ContactListResp({
    $core.Iterable<ContactCard>? contacts,
  }) {
    final result = ContactListResp._();
    if (contacts != null) result.contacts.addAll(contacts);
    return result;
  }

  ContactListResp._();

  factory ContactListResp.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ContactListResp()..mergeFromBuffer(data, registry);
  factory ContactListResp.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ContactListResp()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ContactListResp',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: ContactListResp.$_createMessage)
    ..pPM<ContactCard>(1, _omitFieldNames ? '' : 'contacts',
        subBuilder: ContactCard.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ContactListResp clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ContactListResp copyWith(void Function(ContactListResp) updates) =>
      super.copyWith((message) => updates(message as ContactListResp))
          as ContactListResp;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use ContactListResp() / ContactListResp.new instead')
  static ContactListResp create() => ContactListResp._();
  static $pb.GeneratedMessage $_createMessage() => ContactListResp._();
  @$core.override
  ContactListResp createEmptyInstance() => ContactListResp._();
  @$core.pragma('dart2js:noInline')
  static ContactListResp getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ContactListResp>(
          ContactListResp.$_createMessage);
  static ContactListResp? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<ContactCard> get contacts => $_getList(0);
}

/// Добавить контакта по одному лишь ключу его узла: ядро шлёт bundle_req и
/// заводит карту, когда придёт честный ответ. Имя локальное — бандл его не
/// несёт, nickname выбирает тот, кто добавляет, а не тот, кого добавляют.
class FetchContactReq extends $pb.GeneratedMessage {
  factory FetchContactReq({
    $core.List<$core.int>? nodeEd25519Pubkey,
    $core.String? nickname,
  }) {
    final result = FetchContactReq._();
    if (nodeEd25519Pubkey != null) result.nodeEd25519Pubkey = nodeEd25519Pubkey;
    if (nickname != null) result.nickname = nickname;
    return result;
  }

  FetchContactReq._();

  factory FetchContactReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FetchContactReq()..mergeFromBuffer(data, registry);
  factory FetchContactReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FetchContactReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FetchContactReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: FetchContactReq.$_createMessage)
    ..a<$core.List<$core.int>>(
        1, _omitFieldNames ? '' : 'nodeEd25519Pubkey', $pb.PbFieldType.OY)
    ..aOS(2, _omitFieldNames ? '' : 'nickname')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FetchContactReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FetchContactReq copyWith(void Function(FetchContactReq) updates) =>
      super.copyWith((message) => updates(message as FetchContactReq))
          as FetchContactReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use FetchContactReq() / FetchContactReq.new instead')
  static FetchContactReq create() => FetchContactReq._();
  static $pb.GeneratedMessage $_createMessage() => FetchContactReq._();
  @$core.override
  FetchContactReq createEmptyInstance() => FetchContactReq._();
  @$core.pragma('dart2js:noInline')
  static FetchContactReq getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FetchContactReq>(
          FetchContactReq.$_createMessage);
  static FetchContactReq? _defaultInstance;

  /// Ed25519-ключ узла Yggdrasil — он же адрес в оверлее.
  @$pb.TagNumber(1)
  $core.List<$core.int> get nodeEd25519Pubkey => $_getN(0);
  @$pb.TagNumber(1)
  set nodeEd25519Pubkey($core.List<$core.int> value) => $_setBytes(0, value);
  @$pb.TagNumber(1)
  $core.bool hasNodeEd25519Pubkey() => $_has(0);
  @$pb.TagNumber(1)
  void clearNodeEd25519Pubkey() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get nickname => $_getSZ(1);
  @$pb.TagNumber(2)
  set nickname($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasNickname() => $_has(1);
  @$pb.TagNumber(2)
  void clearNickname() => $_clearField(2);
}

class FetchContactResp extends $pb.GeneratedMessage {
  factory FetchContactResp({
    $core.bool? requested,
    $fixnum.Int64? requestId,
    $core.String? nodeIpv6,
  }) {
    final result = FetchContactResp._();
    if (requested != null) result.requested = requested;
    if (requestId != null) result.requestId = requestId;
    if (nodeIpv6 != null) result.nodeIpv6 = nodeIpv6;
    return result;
  }

  FetchContactResp._();

  factory FetchContactResp.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FetchContactResp()..mergeFromBuffer(data, registry);
  factory FetchContactResp.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      FetchContactResp()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FetchContactResp',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: FetchContactResp.$_createMessage)
    ..aOB(1, _omitFieldNames ? '' : 'requested')
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'requestId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(3, _omitFieldNames ? '' : 'nodeIpv6')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FetchContactResp clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FetchContactResp copyWith(void Function(FetchContactResp) updates) =>
      super.copyWith((message) => updates(message as FetchContactResp))
          as FetchContactResp;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use FetchContactResp() / FetchContactResp.new instead')
  static FetchContactResp create() => FetchContactResp._();
  static $pb.GeneratedMessage $_createMessage() => FetchContactResp._();
  @$core.override
  FetchContactResp createEmptyInstance() => FetchContactResp._();
  @$core.pragma('dart2js:noInline')
  static FetchContactResp getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FetchContactResp>(
          FetchContactResp.$_createMessage);
  static FetchContactResp? _defaultInstance;

  /// Запрос ушёл в сеть. Ответ — событие contact_updated, позже и с другой
  /// стороны: ждать его внутри команды нельзя, тот же mutex нужен приёмному
  /// циклу, который этот ответ и разберёт.
  @$pb.TagNumber(1)
  $core.bool get requested => $_getBF(0);
  @$pb.TagNumber(1)
  set requested($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequested() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequested() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get requestId => $_getI64(1);
  @$pb.TagNumber(2)
  set requestId($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRequestId() => $_has(1);
  @$pb.TagNumber(2)
  void clearRequestId() => $_clearField(2);

  /// Адрес узла, выведенный из запрошенного ключа: UI показывает его, пока
  /// ответа нет, и сверяет с тем, что видит в списке пиров.
  @$pb.TagNumber(3)
  $core.String get nodeIpv6 => $_getSZ(2);
  @$pb.TagNumber(3)
  set nodeIpv6($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasNodeIpv6() => $_has(2);
  @$pb.TagNumber(3)
  void clearNodeIpv6() => $_clearField(3);
}

/// Итоговая новость по fetch-запросу: карта появилась либо ответ не сошёлся.
/// contact заполнен только при успехе; error — причина отказа (узел не
/// ответил, бандл не прошёл ворота, роль раздачи выключена).
class ContactResolvedEvent extends $pb.GeneratedMessage {
  factory ContactResolvedEvent({
    $fixnum.Int64? requestId,
    ContactCard? contact,
    $core.String? error,
  }) {
    final result = ContactResolvedEvent._();
    if (requestId != null) result.requestId = requestId;
    if (contact != null) result.contact = contact;
    if (error != null) result.error = error;
    return result;
  }

  ContactResolvedEvent._();

  factory ContactResolvedEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ContactResolvedEvent()..mergeFromBuffer(data, registry);
  factory ContactResolvedEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ContactResolvedEvent()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ContactResolvedEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: ContactResolvedEvent.$_createMessage)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'requestId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<ContactCard>(2, _omitFieldNames ? '' : 'contact',
        subBuilder: ContactCard.$_createMessage)
    ..aOS(3, _omitFieldNames ? '' : 'error')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ContactResolvedEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ContactResolvedEvent copyWith(void Function(ContactResolvedEvent) updates) =>
      super.copyWith((message) => updates(message as ContactResolvedEvent))
          as ContactResolvedEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ContactResolvedEvent() / ContactResolvedEvent.new instead')
  static ContactResolvedEvent create() => ContactResolvedEvent._();
  static $pb.GeneratedMessage $_createMessage() => ContactResolvedEvent._();
  @$core.override
  ContactResolvedEvent createEmptyInstance() => ContactResolvedEvent._();
  @$core.pragma('dart2js:noInline')
  static ContactResolvedEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ContactResolvedEvent>(
          ContactResolvedEvent.$_createMessage);
  static ContactResolvedEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get requestId => $_getI64(0);
  @$pb.TagNumber(1)
  set requestId($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  ContactCard get contact => $_getN(1);
  @$pb.TagNumber(2)
  set contact(ContactCard value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasContact() => $_has(1);
  @$pb.TagNumber(2)
  void clearContact() => $_clearField(2);
  @$pb.TagNumber(2)
  ContactCard ensureContact() => $_ensure(1);

  @$pb.TagNumber(3)
  $core.String get error => $_getSZ(2);
  @$pb.TagNumber(3)
  set error($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasError() => $_has(2);
  @$pb.TagNumber(3)
  void clearError() => $_clearField(3);
}

/// Забрать почту со всех настроенных релеев прямо сейчас, не дожидаясь таймера
/// опроса: UI зовёт это при выходе из офлайна, тест — чтобы не спать секунды.
class SfCollectReq extends $pb.GeneratedMessage {
  factory SfCollectReq() => SfCollectReq._();

  SfCollectReq._();

  factory SfCollectReq.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SfCollectReq()..mergeFromBuffer(data, registry);
  factory SfCollectReq.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SfCollectReq()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SfCollectReq',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: SfCollectReq.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SfCollectReq clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SfCollectReq copyWith(void Function(SfCollectReq) updates) =>
      super.copyWith((message) => updates(message as SfCollectReq))
          as SfCollectReq;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use SfCollectReq() / SfCollectReq.new instead')
  static SfCollectReq create() => SfCollectReq._();
  static $pb.GeneratedMessage $_createMessage() => SfCollectReq._();
  @$core.override
  SfCollectReq createEmptyInstance() => SfCollectReq._();
  @$core.pragma('dart2js:noInline')
  static SfCollectReq getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SfCollectReq>(
          SfCollectReq.$_createMessage);
  static SfCollectReq? _defaultInstance;
}

class SfCollectResp extends $pb.GeneratedMessage {
  factory SfCollectResp({
    $core.int? requested,
    $core.Iterable<$core.String>? relays,
  }) {
    final result = SfCollectResp._();
    if (requested != null) result.requested = requested;
    if (relays != null) result.relays.addAll(relays);
    return result;
  }

  SfCollectResp._();

  factory SfCollectResp.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SfCollectResp()..mergeFromBuffer(data, registry);
  factory SfCollectResp.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SfCollectResp()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SfCollectResp',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: SfCollectResp.$_createMessage)
    ..aI(1, _omitFieldNames ? '' : 'requested', fieldType: $pb.PbFieldType.OU3)
    ..pPS(2, _omitFieldNames ? '' : 'relays')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SfCollectResp clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SfCollectResp copyWith(void Function(SfCollectResp) updates) =>
      super.copyWith((message) => updates(message as SfCollectResp))
          as SfCollectResp;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use SfCollectResp() / SfCollectResp.new instead')
  static SfCollectResp create() => SfCollectResp._();
  static $pb.GeneratedMessage $_createMessage() => SfCollectResp._();
  @$core.override
  SfCollectResp createEmptyInstance() => SfCollectResp._();
  @$core.pragma('dart2js:noInline')
  static SfCollectResp getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SfCollectResp>(
          SfCollectResp.$_createMessage);
  static SfCollectResp? _defaultInstance;

  /// Сколько ящиков запросено у релеев. Ответы приходят асинхронно через
  /// событийный канал: команда не имеет права держать блокировку ядра, дожидаясь
  /// их, иначе приёмный цикл встанет ровно на то время, пока мы ждём.
  @$pb.TagNumber(1)
  $core.int get requested => $_getIZ(0);
  @$pb.TagNumber(1)
  set requested($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequested() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequested() => $_clearField(1);

  /// Hex-ключи релеев, которым ушёл запрос.
  @$pb.TagNumber(2)
  $pb.PbList<$core.String> get relays => $_getList(1);
}

/// Что ядро знает о своём участии в S&F. Отдельно роль хранителя и роль
/// клиента: «я держу чужие конверты» и «я забрал свои» — разные обязательства.
class SfStats extends $pb.GeneratedMessage {
  factory SfStats({
    $core.bool? relayEnabled,
    $core.int? relaysConfigured,
    $fixnum.Int64? pushed,
    $fixnum.Int64? pushFailed,
    $fixnum.Int64? stored,
    $fixnum.Int64? refused,
    $fixnum.Int64? pulled,
    $fixnum.Int64? handedOut,
    $core.String? lastRefusal,
    $fixnum.Int64? expired,
  }) {
    final result = SfStats._();
    if (relayEnabled != null) result.relayEnabled = relayEnabled;
    if (relaysConfigured != null) result.relaysConfigured = relaysConfigured;
    if (pushed != null) result.pushed = pushed;
    if (pushFailed != null) result.pushFailed = pushFailed;
    if (stored != null) result.stored = stored;
    if (refused != null) result.refused = refused;
    if (pulled != null) result.pulled = pulled;
    if (handedOut != null) result.handedOut = handedOut;
    if (lastRefusal != null) result.lastRefusal = lastRefusal;
    if (expired != null) result.expired = expired;
    return result;
  }

  SfStats._();

  factory SfStats.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SfStats()..mergeFromBuffer(data, registry);
  factory SfStats.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SfStats()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SfStats',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: SfStats.$_createMessage)
    ..aOB(1, _omitFieldNames ? '' : 'relayEnabled')
    ..aI(2, _omitFieldNames ? '' : 'relaysConfigured',
        fieldType: $pb.PbFieldType.OU3)
    ..a<$fixnum.Int64>(3, _omitFieldNames ? '' : 'pushed', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'pushFailed', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(5, _omitFieldNames ? '' : 'stored', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(6, _omitFieldNames ? '' : 'refused', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(7, _omitFieldNames ? '' : 'pulled', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        8, _omitFieldNames ? '' : 'handedOut', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(9, _omitFieldNames ? '' : 'lastRefusal')
    ..a<$fixnum.Int64>(
        10, _omitFieldNames ? '' : 'expired', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SfStats clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SfStats copyWith(void Function(SfStats) updates) =>
      super.copyWith((message) => updates(message as SfStats)) as SfStats;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use SfStats() / SfStats.new instead')
  static SfStats create() => SfStats._();
  static $pb.GeneratedMessage $_createMessage() => SfStats._();
  @$core.override
  SfStats createEmptyInstance() => SfStats._();
  @$core.pragma('dart2js:noInline')
  static SfStats getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SfStats>(SfStats.$_createMessage);
  static SfStats? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get relayEnabled => $_getBF(0);
  @$pb.TagNumber(1)
  set relayEnabled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRelayEnabled() => $_has(0);
  @$pb.TagNumber(1)
  void clearRelayEnabled() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get relaysConfigured => $_getIZ(1);
  @$pb.TagNumber(2)
  set relaysConfigured($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRelaysConfigured() => $_has(1);
  @$pb.TagNumber(2)
  void clearRelaysConfigured() => $_clearField(2);

  /// Отправитель: сколько конвертов положено на релеи и сколько не удалось.
  @$pb.TagNumber(3)
  $fixnum.Int64 get pushed => $_getI64(2);
  @$pb.TagNumber(3)
  set pushed($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPushed() => $_has(2);
  @$pb.TagNumber(3)
  void clearPushed() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get pushFailed => $_getI64(3);
  @$pb.TagNumber(4)
  set pushFailed($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPushFailed() => $_has(3);
  @$pb.TagNumber(4)
  void clearPushFailed() => $_clearField(4);

  /// Хранитель: сколько принято и сколько отвергнуто (PoW, квота, подпись,
  /// несвязка ящика с идентичностью).
  @$pb.TagNumber(5)
  $fixnum.Int64 get stored => $_getI64(4);
  @$pb.TagNumber(5)
  set stored($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasStored() => $_has(4);
  @$pb.TagNumber(5)
  void clearStored() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get refused => $_getI64(5);
  @$pb.TagNumber(6)
  set refused($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasRefused() => $_has(5);
  @$pb.TagNumber(6)
  void clearRefused() => $_clearField(6);

  /// Клиент: сколько забрал со стороны и сколько выдал чужим конвертов.
  @$pb.TagNumber(7)
  $fixnum.Int64 get pulled => $_getI64(6);
  @$pb.TagNumber(7)
  set pulled($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasPulled() => $_has(6);
  @$pb.TagNumber(7)
  void clearPulled() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get handedOut => $_getI64(7);
  @$pb.TagNumber(8)
  set handedOut($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasHandedOut() => $_has(7);
  @$pb.TagNumber(8)
  void clearHandedOut() => $_clearField(8);

  /// Последняя причина отказа релея (нам как клиенту). «Ящик пуст» и «тебя не
  /// пустили» для пользователя значит разное, поэтому молча терять вторую
  /// новость нельзя: счётчик refuse её считает, а строка объясняет.
  @$pb.TagNumber(9)
  $core.String get lastRefusal => $_getSZ(8);
  @$pb.TagNumber(9)
  set lastRefusal($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasLastRefusal() => $_has(8);
  @$pb.TagNumber(9)
  void clearLastRefusal() => $_clearField(9);

  /// Как хранитель: сколько чужих конвертов истекло по опубликованному TTL. Это
  /// потерянная почта, и прятать её за «refused» нельзя — срок вышел честно.
  @$pb.TagNumber(10)
  $fixnum.Int64 get expired => $_getI64(9);
  @$pb.TagNumber(10)
  set expired($fixnum.Int64 value) => $_setInt64(9, value);
  @$pb.TagNumber(10)
  $core.bool hasExpired() => $_has(9);
  @$pb.TagNumber(10)
  void clearExpired() => $_clearField(10);
}

enum CoreRequest_Request {
  startNode,
  stopNode,
  sendMessage,
  connectPeer,
  disconnectPeer,
  getPeers,
  getNodeStatus,
  createBackup,
  restoreBackup,
  addContact,
  getPrekeyBundle,
  generateIdentity,
  listConversations,
  getMessages,
  listContacts,
  sfCollect,
  fetchContact,
  notSet
}

class CoreRequest extends $pb.GeneratedMessage {
  factory CoreRequest({
    $fixnum.Int64? requestId,
    StartNodeReq? startNode,
    StopNodeReq? stopNode,
    SendMessageReq? sendMessage,
    ConnectPeerReq? connectPeer,
    DisconnectPeerReq? disconnectPeer,
    GetPeersReq? getPeers,
    GetNodeStatusReq? getNodeStatus,
    CreateBackupReq? createBackup,
    RestoreBackupReq? restoreBackup,
    AddContactReq? addContact,
    GetPreKeyBundleReq? getPrekeyBundle,
    GenerateIdentityReq? generateIdentity,
    ListConversationsReq? listConversations,
    GetMessagesReq? getMessages,
    ListContactsReq? listContacts,
    SfCollectReq? sfCollect,
    FetchContactReq? fetchContact,
  }) {
    final result = CoreRequest._();
    if (requestId != null) result.requestId = requestId;
    if (startNode != null) result.startNode = startNode;
    if (stopNode != null) result.stopNode = stopNode;
    if (sendMessage != null) result.sendMessage = sendMessage;
    if (connectPeer != null) result.connectPeer = connectPeer;
    if (disconnectPeer != null) result.disconnectPeer = disconnectPeer;
    if (getPeers != null) result.getPeers = getPeers;
    if (getNodeStatus != null) result.getNodeStatus = getNodeStatus;
    if (createBackup != null) result.createBackup = createBackup;
    if (restoreBackup != null) result.restoreBackup = restoreBackup;
    if (addContact != null) result.addContact = addContact;
    if (getPrekeyBundle != null) result.getPrekeyBundle = getPrekeyBundle;
    if (generateIdentity != null) result.generateIdentity = generateIdentity;
    if (listConversations != null) result.listConversations = listConversations;
    if (getMessages != null) result.getMessages = getMessages;
    if (listContacts != null) result.listContacts = listContacts;
    if (sfCollect != null) result.sfCollect = sfCollect;
    if (fetchContact != null) result.fetchContact = fetchContact;
    return result;
  }

  CoreRequest._();

  factory CoreRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CoreRequest()..mergeFromBuffer(data, registry);
  factory CoreRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CoreRequest()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, CoreRequest_Request>
      _CoreRequest_RequestByTag = {
    2: CoreRequest_Request.startNode,
    3: CoreRequest_Request.stopNode,
    4: CoreRequest_Request.sendMessage,
    5: CoreRequest_Request.connectPeer,
    6: CoreRequest_Request.disconnectPeer,
    7: CoreRequest_Request.getPeers,
    8: CoreRequest_Request.getNodeStatus,
    9: CoreRequest_Request.createBackup,
    10: CoreRequest_Request.restoreBackup,
    11: CoreRequest_Request.addContact,
    12: CoreRequest_Request.getPrekeyBundle,
    13: CoreRequest_Request.generateIdentity,
    14: CoreRequest_Request.listConversations,
    15: CoreRequest_Request.getMessages,
    16: CoreRequest_Request.listContacts,
    17: CoreRequest_Request.sfCollect,
    18: CoreRequest_Request.fetchContact,
    0: CoreRequest_Request.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CoreRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: CoreRequest.$_createMessage)
    ..oo(0, [2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18])
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'requestId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<StartNodeReq>(2, _omitFieldNames ? '' : 'startNode',
        subBuilder: StartNodeReq.$_createMessage)
    ..aOM<StopNodeReq>(3, _omitFieldNames ? '' : 'stopNode',
        subBuilder: StopNodeReq.$_createMessage)
    ..aOM<SendMessageReq>(4, _omitFieldNames ? '' : 'sendMessage',
        subBuilder: SendMessageReq.$_createMessage)
    ..aOM<ConnectPeerReq>(5, _omitFieldNames ? '' : 'connectPeer',
        subBuilder: ConnectPeerReq.$_createMessage)
    ..aOM<DisconnectPeerReq>(6, _omitFieldNames ? '' : 'disconnectPeer',
        subBuilder: DisconnectPeerReq.$_createMessage)
    ..aOM<GetPeersReq>(7, _omitFieldNames ? '' : 'getPeers',
        subBuilder: GetPeersReq.$_createMessage)
    ..aOM<GetNodeStatusReq>(8, _omitFieldNames ? '' : 'getNodeStatus',
        subBuilder: GetNodeStatusReq.$_createMessage)
    ..aOM<CreateBackupReq>(9, _omitFieldNames ? '' : 'createBackup',
        subBuilder: CreateBackupReq.$_createMessage)
    ..aOM<RestoreBackupReq>(10, _omitFieldNames ? '' : 'restoreBackup',
        subBuilder: RestoreBackupReq.$_createMessage)
    ..aOM<AddContactReq>(11, _omitFieldNames ? '' : 'addContact',
        subBuilder: AddContactReq.$_createMessage)
    ..aOM<GetPreKeyBundleReq>(12, _omitFieldNames ? '' : 'getPrekeyBundle',
        subBuilder: GetPreKeyBundleReq.$_createMessage)
    ..aOM<GenerateIdentityReq>(13, _omitFieldNames ? '' : 'generateIdentity',
        subBuilder: GenerateIdentityReq.$_createMessage)
    ..aOM<ListConversationsReq>(14, _omitFieldNames ? '' : 'listConversations',
        subBuilder: ListConversationsReq.$_createMessage)
    ..aOM<GetMessagesReq>(15, _omitFieldNames ? '' : 'getMessages',
        subBuilder: GetMessagesReq.$_createMessage)
    ..aOM<ListContactsReq>(16, _omitFieldNames ? '' : 'listContacts',
        subBuilder: ListContactsReq.$_createMessage)
    ..aOM<SfCollectReq>(17, _omitFieldNames ? '' : 'sfCollect',
        subBuilder: SfCollectReq.$_createMessage)
    ..aOM<FetchContactReq>(18, _omitFieldNames ? '' : 'fetchContact',
        subBuilder: FetchContactReq.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CoreRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CoreRequest copyWith(void Function(CoreRequest) updates) =>
      super.copyWith((message) => updates(message as CoreRequest))
          as CoreRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use CoreRequest() / CoreRequest.new instead')
  static CoreRequest create() => CoreRequest._();
  static $pb.GeneratedMessage $_createMessage() => CoreRequest._();
  @$core.override
  CoreRequest createEmptyInstance() => CoreRequest._();
  @$core.pragma('dart2js:noInline')
  static CoreRequest getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CoreRequest>(
          CoreRequest.$_createMessage);
  static CoreRequest? _defaultInstance;

  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  @$pb.TagNumber(6)
  @$pb.TagNumber(7)
  @$pb.TagNumber(8)
  @$pb.TagNumber(9)
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  @$pb.TagNumber(15)
  @$pb.TagNumber(16)
  @$pb.TagNumber(17)
  @$pb.TagNumber(18)
  CoreRequest_Request whichRequest() =>
      _CoreRequest_RequestByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  @$pb.TagNumber(6)
  @$pb.TagNumber(7)
  @$pb.TagNumber(8)
  @$pb.TagNumber(9)
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  @$pb.TagNumber(15)
  @$pb.TagNumber(16)
  @$pb.TagNumber(17)
  @$pb.TagNumber(18)
  void clearRequest() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $fixnum.Int64 get requestId => $_getI64(0);
  @$pb.TagNumber(1)
  set requestId($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  StartNodeReq get startNode => $_getN(1);
  @$pb.TagNumber(2)
  set startNode(StartNodeReq value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasStartNode() => $_has(1);
  @$pb.TagNumber(2)
  void clearStartNode() => $_clearField(2);
  @$pb.TagNumber(2)
  StartNodeReq ensureStartNode() => $_ensure(1);

  @$pb.TagNumber(3)
  StopNodeReq get stopNode => $_getN(2);
  @$pb.TagNumber(3)
  set stopNode(StopNodeReq value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasStopNode() => $_has(2);
  @$pb.TagNumber(3)
  void clearStopNode() => $_clearField(3);
  @$pb.TagNumber(3)
  StopNodeReq ensureStopNode() => $_ensure(2);

  @$pb.TagNumber(4)
  SendMessageReq get sendMessage => $_getN(3);
  @$pb.TagNumber(4)
  set sendMessage(SendMessageReq value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasSendMessage() => $_has(3);
  @$pb.TagNumber(4)
  void clearSendMessage() => $_clearField(4);
  @$pb.TagNumber(4)
  SendMessageReq ensureSendMessage() => $_ensure(3);

  @$pb.TagNumber(5)
  ConnectPeerReq get connectPeer => $_getN(4);
  @$pb.TagNumber(5)
  set connectPeer(ConnectPeerReq value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasConnectPeer() => $_has(4);
  @$pb.TagNumber(5)
  void clearConnectPeer() => $_clearField(5);
  @$pb.TagNumber(5)
  ConnectPeerReq ensureConnectPeer() => $_ensure(4);

  @$pb.TagNumber(6)
  DisconnectPeerReq get disconnectPeer => $_getN(5);
  @$pb.TagNumber(6)
  set disconnectPeer(DisconnectPeerReq value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasDisconnectPeer() => $_has(5);
  @$pb.TagNumber(6)
  void clearDisconnectPeer() => $_clearField(6);
  @$pb.TagNumber(6)
  DisconnectPeerReq ensureDisconnectPeer() => $_ensure(5);

  @$pb.TagNumber(7)
  GetPeersReq get getPeers => $_getN(6);
  @$pb.TagNumber(7)
  set getPeers(GetPeersReq value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasGetPeers() => $_has(6);
  @$pb.TagNumber(7)
  void clearGetPeers() => $_clearField(7);
  @$pb.TagNumber(7)
  GetPeersReq ensureGetPeers() => $_ensure(6);

  @$pb.TagNumber(8)
  GetNodeStatusReq get getNodeStatus => $_getN(7);
  @$pb.TagNumber(8)
  set getNodeStatus(GetNodeStatusReq value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasGetNodeStatus() => $_has(7);
  @$pb.TagNumber(8)
  void clearGetNodeStatus() => $_clearField(8);
  @$pb.TagNumber(8)
  GetNodeStatusReq ensureGetNodeStatus() => $_ensure(7);

  @$pb.TagNumber(9)
  CreateBackupReq get createBackup => $_getN(8);
  @$pb.TagNumber(9)
  set createBackup(CreateBackupReq value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasCreateBackup() => $_has(8);
  @$pb.TagNumber(9)
  void clearCreateBackup() => $_clearField(9);
  @$pb.TagNumber(9)
  CreateBackupReq ensureCreateBackup() => $_ensure(8);

  @$pb.TagNumber(10)
  RestoreBackupReq get restoreBackup => $_getN(9);
  @$pb.TagNumber(10)
  set restoreBackup(RestoreBackupReq value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasRestoreBackup() => $_has(9);
  @$pb.TagNumber(10)
  void clearRestoreBackup() => $_clearField(10);
  @$pb.TagNumber(10)
  RestoreBackupReq ensureRestoreBackup() => $_ensure(9);

  @$pb.TagNumber(11)
  AddContactReq get addContact => $_getN(10);
  @$pb.TagNumber(11)
  set addContact(AddContactReq value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasAddContact() => $_has(10);
  @$pb.TagNumber(11)
  void clearAddContact() => $_clearField(11);
  @$pb.TagNumber(11)
  AddContactReq ensureAddContact() => $_ensure(10);

  @$pb.TagNumber(12)
  GetPreKeyBundleReq get getPrekeyBundle => $_getN(11);
  @$pb.TagNumber(12)
  set getPrekeyBundle(GetPreKeyBundleReq value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasGetPrekeyBundle() => $_has(11);
  @$pb.TagNumber(12)
  void clearGetPrekeyBundle() => $_clearField(12);
  @$pb.TagNumber(12)
  GetPreKeyBundleReq ensureGetPrekeyBundle() => $_ensure(11);

  @$pb.TagNumber(13)
  GenerateIdentityReq get generateIdentity => $_getN(12);
  @$pb.TagNumber(13)
  set generateIdentity(GenerateIdentityReq value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasGenerateIdentity() => $_has(12);
  @$pb.TagNumber(13)
  void clearGenerateIdentity() => $_clearField(13);
  @$pb.TagNumber(13)
  GenerateIdentityReq ensureGenerateIdentity() => $_ensure(12);

  @$pb.TagNumber(14)
  ListConversationsReq get listConversations => $_getN(13);
  @$pb.TagNumber(14)
  set listConversations(ListConversationsReq value) => $_setField(14, value);
  @$pb.TagNumber(14)
  $core.bool hasListConversations() => $_has(13);
  @$pb.TagNumber(14)
  void clearListConversations() => $_clearField(14);
  @$pb.TagNumber(14)
  ListConversationsReq ensureListConversations() => $_ensure(13);

  @$pb.TagNumber(15)
  GetMessagesReq get getMessages => $_getN(14);
  @$pb.TagNumber(15)
  set getMessages(GetMessagesReq value) => $_setField(15, value);
  @$pb.TagNumber(15)
  $core.bool hasGetMessages() => $_has(14);
  @$pb.TagNumber(15)
  void clearGetMessages() => $_clearField(15);
  @$pb.TagNumber(15)
  GetMessagesReq ensureGetMessages() => $_ensure(14);

  @$pb.TagNumber(16)
  ListContactsReq get listContacts => $_getN(15);
  @$pb.TagNumber(16)
  set listContacts(ListContactsReq value) => $_setField(16, value);
  @$pb.TagNumber(16)
  $core.bool hasListContacts() => $_has(15);
  @$pb.TagNumber(16)
  void clearListContacts() => $_clearField(16);
  @$pb.TagNumber(16)
  ListContactsReq ensureListContacts() => $_ensure(15);

  @$pb.TagNumber(17)
  SfCollectReq get sfCollect => $_getN(16);
  @$pb.TagNumber(17)
  set sfCollect(SfCollectReq value) => $_setField(17, value);
  @$pb.TagNumber(17)
  $core.bool hasSfCollect() => $_has(16);
  @$pb.TagNumber(17)
  void clearSfCollect() => $_clearField(17);
  @$pb.TagNumber(17)
  SfCollectReq ensureSfCollect() => $_ensure(16);

  @$pb.TagNumber(18)
  FetchContactReq get fetchContact => $_getN(17);
  @$pb.TagNumber(18)
  set fetchContact(FetchContactReq value) => $_setField(18, value);
  @$pb.TagNumber(18)
  $core.bool hasFetchContact() => $_has(17);
  @$pb.TagNumber(18)
  void clearFetchContact() => $_clearField(18);
  @$pb.TagNumber(18)
  FetchContactReq ensureFetchContact() => $_ensure(17);
}

class PeerInfo extends $pb.GeneratedMessage {
  factory PeerInfo({
    $core.String? uri,
    $core.String? remoteAddress,
    $core.String? yggdrasilIpv6,
    PeerTransportType? transport,
    $core.bool? isInbound,
    $fixnum.Int64? bytesSent,
    $fixnum.Int64? bytesReceived,
    $fixnum.Int64? latencyMs,
    $core.bool? isConnected,
    $core.String? sniUsed,
  }) {
    final result = PeerInfo._();
    if (uri != null) result.uri = uri;
    if (remoteAddress != null) result.remoteAddress = remoteAddress;
    if (yggdrasilIpv6 != null) result.yggdrasilIpv6 = yggdrasilIpv6;
    if (transport != null) result.transport = transport;
    if (isInbound != null) result.isInbound = isInbound;
    if (bytesSent != null) result.bytesSent = bytesSent;
    if (bytesReceived != null) result.bytesReceived = bytesReceived;
    if (latencyMs != null) result.latencyMs = latencyMs;
    if (isConnected != null) result.isConnected = isConnected;
    if (sniUsed != null) result.sniUsed = sniUsed;
    return result;
  }

  PeerInfo._();

  factory PeerInfo.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      PeerInfo()..mergeFromBuffer(data, registry);
  factory PeerInfo.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      PeerInfo()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PeerInfo',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: PeerInfo.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'uri')
    ..aOS(2, _omitFieldNames ? '' : 'remoteAddress')
    ..aOS(3, _omitFieldNames ? '' : 'yggdrasilIpv6')
    ..aE<PeerTransportType>(4, _omitFieldNames ? '' : 'transport',
        enumValues: PeerTransportType.values)
    ..aOB(5, _omitFieldNames ? '' : 'isInbound')
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'bytesSent', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        7, _omitFieldNames ? '' : 'bytesReceived', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        8, _omitFieldNames ? '' : 'latencyMs', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(9, _omitFieldNames ? '' : 'isConnected')
    ..aOS(10, _omitFieldNames ? '' : 'sniUsed')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PeerInfo clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PeerInfo copyWith(void Function(PeerInfo) updates) =>
      super.copyWith((message) => updates(message as PeerInfo)) as PeerInfo;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use PeerInfo() / PeerInfo.new instead')
  static PeerInfo create() => PeerInfo._();
  static $pb.GeneratedMessage $_createMessage() => PeerInfo._();
  @$core.override
  PeerInfo createEmptyInstance() => PeerInfo._();
  @$core.pragma('dart2js:noInline')
  static PeerInfo getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PeerInfo>(PeerInfo.$_createMessage);
  static PeerInfo? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get uri => $_getSZ(0);
  @$pb.TagNumber(1)
  set uri($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasUri() => $_has(0);
  @$pb.TagNumber(1)
  void clearUri() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get remoteAddress => $_getSZ(1);
  @$pb.TagNumber(2)
  set remoteAddress($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRemoteAddress() => $_has(1);
  @$pb.TagNumber(2)
  void clearRemoteAddress() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get yggdrasilIpv6 => $_getSZ(2);
  @$pb.TagNumber(3)
  set yggdrasilIpv6($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasYggdrasilIpv6() => $_has(2);
  @$pb.TagNumber(3)
  void clearYggdrasilIpv6() => $_clearField(3);

  @$pb.TagNumber(4)
  PeerTransportType get transport => $_getN(3);
  @$pb.TagNumber(4)
  set transport(PeerTransportType value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasTransport() => $_has(3);
  @$pb.TagNumber(4)
  void clearTransport() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get isInbound => $_getBF(4);
  @$pb.TagNumber(5)
  set isInbound($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasIsInbound() => $_has(4);
  @$pb.TagNumber(5)
  void clearIsInbound() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get bytesSent => $_getI64(5);
  @$pb.TagNumber(6)
  set bytesSent($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasBytesSent() => $_has(5);
  @$pb.TagNumber(6)
  void clearBytesSent() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get bytesReceived => $_getI64(6);
  @$pb.TagNumber(7)
  set bytesReceived($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasBytesReceived() => $_has(6);
  @$pb.TagNumber(7)
  void clearBytesReceived() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get latencyMs => $_getI64(7);
  @$pb.TagNumber(8)
  set latencyMs($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasLatencyMs() => $_has(7);
  @$pb.TagNumber(8)
  void clearLatencyMs() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.bool get isConnected => $_getBF(8);
  @$pb.TagNumber(9)
  set isConnected($core.bool value) => $_setBool(8, value);
  @$pb.TagNumber(9)
  $core.bool hasIsConnected() => $_has(8);
  @$pb.TagNumber(9)
  void clearIsConnected() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.String get sniUsed => $_getSZ(9);
  @$pb.TagNumber(10)
  set sniUsed($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasSniUsed() => $_has(9);
  @$pb.TagNumber(10)
  void clearSniUsed() => $_clearField(10);
}

class NodeStatusResp extends $pb.GeneratedMessage {
  factory NodeStatusResp({
    $core.bool? isRunning,
    $core.String? myIpv6,
    $core.List<$core.int>? myEd25519Pubkey,
    $core.List<$core.int>? myX25519Pubkey,
    $core.int? connectedPeersCount,
    $fixnum.Int64? uptimeSeconds,
    AntiDpiMode? activeAntiDpi,
    $fixnum.Int64? routingTableNodes,
    SfStats? sf,
  }) {
    final result = NodeStatusResp._();
    if (isRunning != null) result.isRunning = isRunning;
    if (myIpv6 != null) result.myIpv6 = myIpv6;
    if (myEd25519Pubkey != null) result.myEd25519Pubkey = myEd25519Pubkey;
    if (myX25519Pubkey != null) result.myX25519Pubkey = myX25519Pubkey;
    if (connectedPeersCount != null)
      result.connectedPeersCount = connectedPeersCount;
    if (uptimeSeconds != null) result.uptimeSeconds = uptimeSeconds;
    if (activeAntiDpi != null) result.activeAntiDpi = activeAntiDpi;
    if (routingTableNodes != null) result.routingTableNodes = routingTableNodes;
    if (sf != null) result.sf = sf;
    return result;
  }

  NodeStatusResp._();

  factory NodeStatusResp.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      NodeStatusResp()..mergeFromBuffer(data, registry);
  factory NodeStatusResp.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      NodeStatusResp()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'NodeStatusResp',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: NodeStatusResp.$_createMessage)
    ..aOB(1, _omitFieldNames ? '' : 'isRunning')
    ..aOS(2, _omitFieldNames ? '' : 'myIpv6')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'myEd25519Pubkey', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'myX25519Pubkey', $pb.PbFieldType.OY)
    ..aI(5, _omitFieldNames ? '' : 'connectedPeersCount',
        fieldType: $pb.PbFieldType.OU3)
    ..a<$fixnum.Int64>(
        6, _omitFieldNames ? '' : 'uptimeSeconds', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aE<AntiDpiMode>(7, _omitFieldNames ? '' : 'activeAntiDpi',
        enumValues: AntiDpiMode.values)
    ..a<$fixnum.Int64>(
        8, _omitFieldNames ? '' : 'routingTableNodes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<SfStats>(9, _omitFieldNames ? '' : 'sf',
        subBuilder: SfStats.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NodeStatusResp clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NodeStatusResp copyWith(void Function(NodeStatusResp) updates) =>
      super.copyWith((message) => updates(message as NodeStatusResp))
          as NodeStatusResp;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use NodeStatusResp() / NodeStatusResp.new instead')
  static NodeStatusResp create() => NodeStatusResp._();
  static $pb.GeneratedMessage $_createMessage() => NodeStatusResp._();
  @$core.override
  NodeStatusResp createEmptyInstance() => NodeStatusResp._();
  @$core.pragma('dart2js:noInline')
  static NodeStatusResp getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<NodeStatusResp>(
          NodeStatusResp.$_createMessage);
  static NodeStatusResp? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get isRunning => $_getBF(0);
  @$pb.TagNumber(1)
  set isRunning($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasIsRunning() => $_has(0);
  @$pb.TagNumber(1)
  void clearIsRunning() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get myIpv6 => $_getSZ(1);
  @$pb.TagNumber(2)
  set myIpv6($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMyIpv6() => $_has(1);
  @$pb.TagNumber(2)
  void clearMyIpv6() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get myEd25519Pubkey => $_getN(2);
  @$pb.TagNumber(3)
  set myEd25519Pubkey($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMyEd25519Pubkey() => $_has(2);
  @$pb.TagNumber(3)
  void clearMyEd25519Pubkey() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get myX25519Pubkey => $_getN(3);
  @$pb.TagNumber(4)
  set myX25519Pubkey($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasMyX25519Pubkey() => $_has(3);
  @$pb.TagNumber(4)
  void clearMyX25519Pubkey() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get connectedPeersCount => $_getIZ(4);
  @$pb.TagNumber(5)
  set connectedPeersCount($core.int value) => $_setUnsignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasConnectedPeersCount() => $_has(4);
  @$pb.TagNumber(5)
  void clearConnectedPeersCount() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get uptimeSeconds => $_getI64(5);
  @$pb.TagNumber(6)
  set uptimeSeconds($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasUptimeSeconds() => $_has(5);
  @$pb.TagNumber(6)
  void clearUptimeSeconds() => $_clearField(6);

  @$pb.TagNumber(7)
  AntiDpiMode get activeAntiDpi => $_getN(6);
  @$pb.TagNumber(7)
  set activeAntiDpi(AntiDpiMode value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasActiveAntiDpi() => $_has(6);
  @$pb.TagNumber(7)
  void clearActiveAntiDpi() => $_clearField(7);

  @$pb.TagNumber(8)
  $fixnum.Int64 get routingTableNodes => $_getI64(7);
  @$pb.TagNumber(8)
  set routingTableNodes($fixnum.Int64 value) => $_setInt64(7, value);
  @$pb.TagNumber(8)
  $core.bool hasRoutingTableNodes() => $_has(7);
  @$pb.TagNumber(8)
  void clearRoutingTableNodes() => $_clearField(8);

  @$pb.TagNumber(9)
  SfStats get sf => $_getN(8);
  @$pb.TagNumber(9)
  set sf(SfStats value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasSf() => $_has(8);
  @$pb.TagNumber(9)
  void clearSf() => $_clearField(9);
  @$pb.TagNumber(9)
  SfStats ensureSf() => $_ensure(8);
}

class PeerListResp extends $pb.GeneratedMessage {
  factory PeerListResp({
    $core.Iterable<PeerInfo>? peers,
  }) {
    final result = PeerListResp._();
    if (peers != null) result.peers.addAll(peers);
    return result;
  }

  PeerListResp._();

  factory PeerListResp.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      PeerListResp()..mergeFromBuffer(data, registry);
  factory PeerListResp.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      PeerListResp()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PeerListResp',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: PeerListResp.$_createMessage)
    ..pPM<PeerInfo>(1, _omitFieldNames ? '' : 'peers',
        subBuilder: PeerInfo.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PeerListResp clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PeerListResp copyWith(void Function(PeerListResp) updates) =>
      super.copyWith((message) => updates(message as PeerListResp))
          as PeerListResp;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use PeerListResp() / PeerListResp.new instead')
  static PeerListResp create() => PeerListResp._();
  static $pb.GeneratedMessage $_createMessage() => PeerListResp._();
  @$core.override
  PeerListResp createEmptyInstance() => PeerListResp._();
  @$core.pragma('dart2js:noInline')
  static PeerListResp getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<PeerListResp>(
          PeerListResp.$_createMessage);
  static PeerListResp? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<PeerInfo> get peers => $_getList(0);
}

class SendMessageResp extends $pb.GeneratedMessage {
  factory SendMessageResp({
    $core.String? messageId,
    $core.bool? writtenToOverlay,
    $core.bool? queuedInOutbox,
    $core.List<$core.int>? sealedEnvelope,
    $core.bool? relaysPushed,
    $core.int? relaysHanded,
  }) {
    final result = SendMessageResp._();
    if (messageId != null) result.messageId = messageId;
    if (writtenToOverlay != null) result.writtenToOverlay = writtenToOverlay;
    if (queuedInOutbox != null) result.queuedInOutbox = queuedInOutbox;
    if (sealedEnvelope != null) result.sealedEnvelope = sealedEnvelope;
    if (relaysPushed != null) result.relaysPushed = relaysPushed;
    if (relaysHanded != null) result.relaysHanded = relaysHanded;
    return result;
  }

  SendMessageResp._();

  factory SendMessageResp.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SendMessageResp()..mergeFromBuffer(data, registry);
  factory SendMessageResp.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SendMessageResp()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SendMessageResp',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: SendMessageResp.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'messageId')
    ..aOB(2, _omitFieldNames ? '' : 'writtenToOverlay')
    ..aOB(3, _omitFieldNames ? '' : 'queuedInOutbox')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'sealedEnvelope', $pb.PbFieldType.OY)
    ..aOB(5, _omitFieldNames ? '' : 'relaysPushed')
    ..aI(6, _omitFieldNames ? '' : 'relaysHanded',
        fieldType: $pb.PbFieldType.OU3)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SendMessageResp clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SendMessageResp copyWith(void Function(SendMessageResp) updates) =>
      super.copyWith((message) => updates(message as SendMessageResp))
          as SendMessageResp;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use SendMessageResp() / SendMessageResp.new instead')
  static SendMessageResp create() => SendMessageResp._();
  static $pb.GeneratedMessage $_createMessage() => SendMessageResp._();
  @$core.override
  SendMessageResp createEmptyInstance() => SendMessageResp._();
  @$core.pragma('dart2js:noInline')
  static SendMessageResp getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SendMessageResp>(
          SendMessageResp.$_createMessage);
  static SendMessageResp? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get messageId => $_getSZ(0);
  @$pb.TagNumber(1)
  set messageId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMessageId() => $_has(0);
  @$pb.TagNumber(1)
  void clearMessageId() => $_clearField(1);

  /// Пакет ушёл в оверлей, когда spanning-дерево знало маршрут до ключа
  /// получателя. Это НЕ доставка: адресат мог быть офлайн, а «доставлено»
  /// подтверждает только DeliveryAckContent в ChatMessage.
  @$pb.TagNumber(2)
  $core.bool get writtenToOverlay => $_getBF(1);
  @$pb.TagNumber(2)
  set writtenToOverlay($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasWrittenToOverlay() => $_has(1);
  @$pb.TagNumber(2)
  void clearWrittenToOverlay() => $_clearField(2);

  /// Маршрута в момент отправки не было, и конверт лёг в зашифрованную
  /// исходящую очередь ядра: сканер перешлёт ровно эти байты, как только
  /// оверлей узнает адресата, в том числе после перезапуска. Параллельно кадр
  /// может уйти слепым релеям (relays_pushed) — локальная очередь от этого не
  /// снимается, потому что релей не ручается за хранение.
  @$pb.TagNumber(3)
  $core.bool get queuedInOutbox => $_getBF(2);
  @$pb.TagNumber(3)
  set queuedInOutbox($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasQueuedInOutbox() => $_has(2);
  @$pb.TagNumber(3)
  void clearQueuedInOutbox() => $_clearField(3);

  /// Те самые байты, что ушли (или должны уйти) в сеть: кадр WireFrame с
  /// конвертом внутри. Без них клиент не сможет переложить сообщение в
  /// S&F-очередь.
  @$pb.TagNumber(4)
  $core.List<$core.int> get sealedEnvelope => $_getN(3);
  @$pb.TagNumber(4)
  set sealedEnvelope($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSealedEnvelope() => $_has(3);
  @$pb.TagNumber(4)
  void clearSealedEnvelope() => $_clearField(4);

  /// Конверт оставлен слепому релею: адресат офлайн, и байты легли в чужое
  /// хранилище до его появления. Локальная очередь при этом остаётся — релеи
  /// не дают гарантии, что почту не потеряют.
  /// Честная формулировка: кадр передан оверлею в адрес релея. Хранение этим не
  /// подтверждается — релей молчит, и единственный настоящий сигнал «письмо
  /// дошло» это ACK адресата.
  @$pb.TagNumber(5)
  $core.bool get relaysPushed => $_getBF(4);
  @$pb.TagNumber(5)
  set relaysPushed($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRelaysPushed() => $_has(4);
  @$pb.TagNumber(5)
  void clearRelaysPushed() => $_clearField(5);

  /// Сколько релеев получили кадр (не «сколько хранят»).
  @$pb.TagNumber(6)
  $core.int get relaysHanded => $_getIZ(5);
  @$pb.TagNumber(6)
  set relaysHanded($core.int value) => $_setUnsignedInt32(5, value);
  @$pb.TagNumber(6)
  $core.bool hasRelaysHanded() => $_has(5);
  @$pb.TagNumber(6)
  void clearRelaysHanded() => $_clearField(6);
}

class BackupResp extends $pb.GeneratedMessage {
  factory BackupResp({
    BackupManifest? manifest,
    $core.String? statusMessage,
  }) {
    final result = BackupResp._();
    if (manifest != null) result.manifest = manifest;
    if (statusMessage != null) result.statusMessage = statusMessage;
    return result;
  }

  BackupResp._();

  factory BackupResp.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      BackupResp()..mergeFromBuffer(data, registry);
  factory BackupResp.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      BackupResp()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'BackupResp',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: BackupResp.$_createMessage)
    ..aOM<BackupManifest>(1, _omitFieldNames ? '' : 'manifest',
        subBuilder: BackupManifest.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'statusMessage')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BackupResp clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BackupResp copyWith(void Function(BackupResp) updates) =>
      super.copyWith((message) => updates(message as BackupResp)) as BackupResp;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use BackupResp() / BackupResp.new instead')
  static BackupResp create() => BackupResp._();
  static $pb.GeneratedMessage $_createMessage() => BackupResp._();
  @$core.override
  BackupResp createEmptyInstance() => BackupResp._();
  @$core.pragma('dart2js:noInline')
  static BackupResp getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<BackupResp>(BackupResp.$_createMessage);
  static BackupResp? _defaultInstance;

  @$pb.TagNumber(1)
  BackupManifest get manifest => $_getN(0);
  @$pb.TagNumber(1)
  set manifest(BackupManifest value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasManifest() => $_has(0);
  @$pb.TagNumber(1)
  void clearManifest() => $_clearField(1);
  @$pb.TagNumber(1)
  BackupManifest ensureManifest() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.String get statusMessage => $_getSZ(1);
  @$pb.TagNumber(2)
  set statusMessage($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasStatusMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearStatusMessage() => $_clearField(2);
}

enum CoreResponse_Response {
  nodeStatus,
  peerList,
  sendMessage,
  backup,
  prekeyBundle,
  identity,
  conversations,
  messages,
  contacts,
  sfCollect,
  fetchContact,
  notSet
}

class CoreResponse extends $pb.GeneratedMessage {
  factory CoreResponse({
    $fixnum.Int64? requestId,
    $core.bool? success,
    $core.String? errorMessage,
    NodeStatusResp? nodeStatus,
    PeerListResp? peerList,
    SendMessageResp? sendMessage,
    BackupResp? backup,
    PreKeyBundle? prekeyBundle,
    IdentityResp? identity,
    ConversationListResp? conversations,
    MessageListResp? messages,
    ContactListResp? contacts,
    SfCollectResp? sfCollect,
    FetchContactResp? fetchContact,
  }) {
    final result = CoreResponse._();
    if (requestId != null) result.requestId = requestId;
    if (success != null) result.success = success;
    if (errorMessage != null) result.errorMessage = errorMessage;
    if (nodeStatus != null) result.nodeStatus = nodeStatus;
    if (peerList != null) result.peerList = peerList;
    if (sendMessage != null) result.sendMessage = sendMessage;
    if (backup != null) result.backup = backup;
    if (prekeyBundle != null) result.prekeyBundle = prekeyBundle;
    if (identity != null) result.identity = identity;
    if (conversations != null) result.conversations = conversations;
    if (messages != null) result.messages = messages;
    if (contacts != null) result.contacts = contacts;
    if (sfCollect != null) result.sfCollect = sfCollect;
    if (fetchContact != null) result.fetchContact = fetchContact;
    return result;
  }

  CoreResponse._();

  factory CoreResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CoreResponse()..mergeFromBuffer(data, registry);
  factory CoreResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CoreResponse()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, CoreResponse_Response>
      _CoreResponse_ResponseByTag = {
    4: CoreResponse_Response.nodeStatus,
    5: CoreResponse_Response.peerList,
    6: CoreResponse_Response.sendMessage,
    7: CoreResponse_Response.backup,
    8: CoreResponse_Response.prekeyBundle,
    9: CoreResponse_Response.identity,
    10: CoreResponse_Response.conversations,
    11: CoreResponse_Response.messages,
    12: CoreResponse_Response.contacts,
    13: CoreResponse_Response.sfCollect,
    14: CoreResponse_Response.fetchContact,
    0: CoreResponse_Response.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CoreResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: CoreResponse.$_createMessage)
    ..oo(0, [4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14])
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'requestId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOB(2, _omitFieldNames ? '' : 'success')
    ..aOS(3, _omitFieldNames ? '' : 'errorMessage')
    ..aOM<NodeStatusResp>(4, _omitFieldNames ? '' : 'nodeStatus',
        subBuilder: NodeStatusResp.$_createMessage)
    ..aOM<PeerListResp>(5, _omitFieldNames ? '' : 'peerList',
        subBuilder: PeerListResp.$_createMessage)
    ..aOM<SendMessageResp>(6, _omitFieldNames ? '' : 'sendMessage',
        subBuilder: SendMessageResp.$_createMessage)
    ..aOM<BackupResp>(7, _omitFieldNames ? '' : 'backup',
        subBuilder: BackupResp.$_createMessage)
    ..aOM<PreKeyBundle>(8, _omitFieldNames ? '' : 'prekeyBundle',
        subBuilder: PreKeyBundle.$_createMessage)
    ..aOM<IdentityResp>(9, _omitFieldNames ? '' : 'identity',
        subBuilder: IdentityResp.$_createMessage)
    ..aOM<ConversationListResp>(10, _omitFieldNames ? '' : 'conversations',
        subBuilder: ConversationListResp.$_createMessage)
    ..aOM<MessageListResp>(11, _omitFieldNames ? '' : 'messages',
        subBuilder: MessageListResp.$_createMessage)
    ..aOM<ContactListResp>(12, _omitFieldNames ? '' : 'contacts',
        subBuilder: ContactListResp.$_createMessage)
    ..aOM<SfCollectResp>(13, _omitFieldNames ? '' : 'sfCollect',
        subBuilder: SfCollectResp.$_createMessage)
    ..aOM<FetchContactResp>(14, _omitFieldNames ? '' : 'fetchContact',
        subBuilder: FetchContactResp.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CoreResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CoreResponse copyWith(void Function(CoreResponse) updates) =>
      super.copyWith((message) => updates(message as CoreResponse))
          as CoreResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use CoreResponse() / CoreResponse.new instead')
  static CoreResponse create() => CoreResponse._();
  static $pb.GeneratedMessage $_createMessage() => CoreResponse._();
  @$core.override
  CoreResponse createEmptyInstance() => CoreResponse._();
  @$core.pragma('dart2js:noInline')
  static CoreResponse getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CoreResponse>(
          CoreResponse.$_createMessage);
  static CoreResponse? _defaultInstance;

  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  @$pb.TagNumber(6)
  @$pb.TagNumber(7)
  @$pb.TagNumber(8)
  @$pb.TagNumber(9)
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  CoreResponse_Response whichResponse() =>
      _CoreResponse_ResponseByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  @$pb.TagNumber(6)
  @$pb.TagNumber(7)
  @$pb.TagNumber(8)
  @$pb.TagNumber(9)
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  void clearResponse() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $fixnum.Int64 get requestId => $_getI64(0);
  @$pb.TagNumber(1)
  set requestId($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get success => $_getBF(1);
  @$pb.TagNumber(2)
  set success($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSuccess() => $_has(1);
  @$pb.TagNumber(2)
  void clearSuccess() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get errorMessage => $_getSZ(2);
  @$pb.TagNumber(3)
  set errorMessage($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasErrorMessage() => $_has(2);
  @$pb.TagNumber(3)
  void clearErrorMessage() => $_clearField(3);

  @$pb.TagNumber(4)
  NodeStatusResp get nodeStatus => $_getN(3);
  @$pb.TagNumber(4)
  set nodeStatus(NodeStatusResp value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasNodeStatus() => $_has(3);
  @$pb.TagNumber(4)
  void clearNodeStatus() => $_clearField(4);
  @$pb.TagNumber(4)
  NodeStatusResp ensureNodeStatus() => $_ensure(3);

  @$pb.TagNumber(5)
  PeerListResp get peerList => $_getN(4);
  @$pb.TagNumber(5)
  set peerList(PeerListResp value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasPeerList() => $_has(4);
  @$pb.TagNumber(5)
  void clearPeerList() => $_clearField(5);
  @$pb.TagNumber(5)
  PeerListResp ensurePeerList() => $_ensure(4);

  @$pb.TagNumber(6)
  SendMessageResp get sendMessage => $_getN(5);
  @$pb.TagNumber(6)
  set sendMessage(SendMessageResp value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasSendMessage() => $_has(5);
  @$pb.TagNumber(6)
  void clearSendMessage() => $_clearField(6);
  @$pb.TagNumber(6)
  SendMessageResp ensureSendMessage() => $_ensure(5);

  @$pb.TagNumber(7)
  BackupResp get backup => $_getN(6);
  @$pb.TagNumber(7)
  set backup(BackupResp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasBackup() => $_has(6);
  @$pb.TagNumber(7)
  void clearBackup() => $_clearField(7);
  @$pb.TagNumber(7)
  BackupResp ensureBackup() => $_ensure(6);

  @$pb.TagNumber(8)
  PreKeyBundle get prekeyBundle => $_getN(7);
  @$pb.TagNumber(8)
  set prekeyBundle(PreKeyBundle value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasPrekeyBundle() => $_has(7);
  @$pb.TagNumber(8)
  void clearPrekeyBundle() => $_clearField(8);
  @$pb.TagNumber(8)
  PreKeyBundle ensurePrekeyBundle() => $_ensure(7);

  @$pb.TagNumber(9)
  IdentityResp get identity => $_getN(8);
  @$pb.TagNumber(9)
  set identity(IdentityResp value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasIdentity() => $_has(8);
  @$pb.TagNumber(9)
  void clearIdentity() => $_clearField(9);
  @$pb.TagNumber(9)
  IdentityResp ensureIdentity() => $_ensure(8);

  @$pb.TagNumber(10)
  ConversationListResp get conversations => $_getN(9);
  @$pb.TagNumber(10)
  set conversations(ConversationListResp value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasConversations() => $_has(9);
  @$pb.TagNumber(10)
  void clearConversations() => $_clearField(10);
  @$pb.TagNumber(10)
  ConversationListResp ensureConversations() => $_ensure(9);

  @$pb.TagNumber(11)
  MessageListResp get messages => $_getN(10);
  @$pb.TagNumber(11)
  set messages(MessageListResp value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasMessages() => $_has(10);
  @$pb.TagNumber(11)
  void clearMessages() => $_clearField(11);
  @$pb.TagNumber(11)
  MessageListResp ensureMessages() => $_ensure(10);

  @$pb.TagNumber(12)
  ContactListResp get contacts => $_getN(11);
  @$pb.TagNumber(12)
  set contacts(ContactListResp value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasContacts() => $_has(11);
  @$pb.TagNumber(12)
  void clearContacts() => $_clearField(12);
  @$pb.TagNumber(12)
  ContactListResp ensureContacts() => $_ensure(11);

  @$pb.TagNumber(13)
  SfCollectResp get sfCollect => $_getN(12);
  @$pb.TagNumber(13)
  set sfCollect(SfCollectResp value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasSfCollect() => $_has(12);
  @$pb.TagNumber(13)
  void clearSfCollect() => $_clearField(13);
  @$pb.TagNumber(13)
  SfCollectResp ensureSfCollect() => $_ensure(12);

  @$pb.TagNumber(14)
  FetchContactResp get fetchContact => $_getN(13);
  @$pb.TagNumber(14)
  set fetchContact(FetchContactResp value) => $_setField(14, value);
  @$pb.TagNumber(14)
  $core.bool hasFetchContact() => $_has(13);
  @$pb.TagNumber(14)
  void clearFetchContact() => $_clearField(14);
  @$pb.TagNumber(14)
  FetchContactResp ensureFetchContact() => $_ensure(13);
}

class NetworkStatusEvent extends $pb.GeneratedMessage {
  factory NetworkStatusEvent({
    $core.bool? online,
    $core.String? currentIpv6,
    $core.int? activePeers,
    $fixnum.Int64? totalRxBytes,
    $fixnum.Int64? totalTxBytes,
    $core.String? statusNote,
  }) {
    final result = NetworkStatusEvent._();
    if (online != null) result.online = online;
    if (currentIpv6 != null) result.currentIpv6 = currentIpv6;
    if (activePeers != null) result.activePeers = activePeers;
    if (totalRxBytes != null) result.totalRxBytes = totalRxBytes;
    if (totalTxBytes != null) result.totalTxBytes = totalTxBytes;
    if (statusNote != null) result.statusNote = statusNote;
    return result;
  }

  NetworkStatusEvent._();

  factory NetworkStatusEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      NetworkStatusEvent()..mergeFromBuffer(data, registry);
  factory NetworkStatusEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      NetworkStatusEvent()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'NetworkStatusEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: NetworkStatusEvent.$_createMessage)
    ..aOB(1, _omitFieldNames ? '' : 'online')
    ..aOS(2, _omitFieldNames ? '' : 'currentIpv6')
    ..aI(3, _omitFieldNames ? '' : 'activePeers',
        fieldType: $pb.PbFieldType.OU3)
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'totalRxBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'totalTxBytes', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(6, _omitFieldNames ? '' : 'statusNote')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NetworkStatusEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NetworkStatusEvent copyWith(void Function(NetworkStatusEvent) updates) =>
      super.copyWith((message) => updates(message as NetworkStatusEvent))
          as NetworkStatusEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use NetworkStatusEvent() / NetworkStatusEvent.new instead')
  static NetworkStatusEvent create() => NetworkStatusEvent._();
  static $pb.GeneratedMessage $_createMessage() => NetworkStatusEvent._();
  @$core.override
  NetworkStatusEvent createEmptyInstance() => NetworkStatusEvent._();
  @$core.pragma('dart2js:noInline')
  static NetworkStatusEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<NetworkStatusEvent>(
          NetworkStatusEvent.$_createMessage);
  static NetworkStatusEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get online => $_getBF(0);
  @$pb.TagNumber(1)
  set online($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasOnline() => $_has(0);
  @$pb.TagNumber(1)
  void clearOnline() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get currentIpv6 => $_getSZ(1);
  @$pb.TagNumber(2)
  set currentIpv6($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCurrentIpv6() => $_has(1);
  @$pb.TagNumber(2)
  void clearCurrentIpv6() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get activePeers => $_getIZ(2);
  @$pb.TagNumber(3)
  set activePeers($core.int value) => $_setUnsignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasActivePeers() => $_has(2);
  @$pb.TagNumber(3)
  void clearActivePeers() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get totalRxBytes => $_getI64(3);
  @$pb.TagNumber(4)
  set totalRxBytes($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasTotalRxBytes() => $_has(3);
  @$pb.TagNumber(4)
  void clearTotalRxBytes() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get totalTxBytes => $_getI64(4);
  @$pb.TagNumber(5)
  set totalTxBytes($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasTotalTxBytes() => $_has(4);
  @$pb.TagNumber(5)
  void clearTotalTxBytes() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get statusNote => $_getSZ(5);
  @$pb.TagNumber(6)
  set statusNote($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasStatusNote() => $_has(5);
  @$pb.TagNumber(6)
  void clearStatusNote() => $_clearField(6);
}

class PeerListUpdatedEvent extends $pb.GeneratedMessage {
  factory PeerListUpdatedEvent({
    $core.Iterable<PeerInfo>? peers,
  }) {
    final result = PeerListUpdatedEvent._();
    if (peers != null) result.peers.addAll(peers);
    return result;
  }

  PeerListUpdatedEvent._();

  factory PeerListUpdatedEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      PeerListUpdatedEvent()..mergeFromBuffer(data, registry);
  factory PeerListUpdatedEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      PeerListUpdatedEvent()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PeerListUpdatedEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: PeerListUpdatedEvent.$_createMessage)
    ..pPM<PeerInfo>(1, _omitFieldNames ? '' : 'peers',
        subBuilder: PeerInfo.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PeerListUpdatedEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PeerListUpdatedEvent copyWith(void Function(PeerListUpdatedEvent) updates) =>
      super.copyWith((message) => updates(message as PeerListUpdatedEvent))
          as PeerListUpdatedEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use PeerListUpdatedEvent() / PeerListUpdatedEvent.new instead')
  static PeerListUpdatedEvent create() => PeerListUpdatedEvent._();
  static $pb.GeneratedMessage $_createMessage() => PeerListUpdatedEvent._();
  @$core.override
  PeerListUpdatedEvent createEmptyInstance() => PeerListUpdatedEvent._();
  @$core.pragma('dart2js:noInline')
  static PeerListUpdatedEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PeerListUpdatedEvent>(
          PeerListUpdatedEvent.$_createMessage);
  static PeerListUpdatedEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<PeerInfo> get peers => $_getList(0);
}

class AntiDpiStatusEvent extends $pb.GeneratedMessage {
  factory AntiDpiStatusEvent({
    AntiDpiMode? mode,
    $core.String? currentSni,
    $core.bool? handshakeSuccess,
    $core.String? transportDetails,
    $core.bool? fallbackTriggered,
  }) {
    final result = AntiDpiStatusEvent._();
    if (mode != null) result.mode = mode;
    if (currentSni != null) result.currentSni = currentSni;
    if (handshakeSuccess != null) result.handshakeSuccess = handshakeSuccess;
    if (transportDetails != null) result.transportDetails = transportDetails;
    if (fallbackTriggered != null) result.fallbackTriggered = fallbackTriggered;
    return result;
  }

  AntiDpiStatusEvent._();

  factory AntiDpiStatusEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      AntiDpiStatusEvent()..mergeFromBuffer(data, registry);
  factory AntiDpiStatusEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      AntiDpiStatusEvent()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AntiDpiStatusEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: AntiDpiStatusEvent.$_createMessage)
    ..aE<AntiDpiMode>(1, _omitFieldNames ? '' : 'mode',
        enumValues: AntiDpiMode.values)
    ..aOS(2, _omitFieldNames ? '' : 'currentSni')
    ..aOB(3, _omitFieldNames ? '' : 'handshakeSuccess')
    ..aOS(4, _omitFieldNames ? '' : 'transportDetails')
    ..aOB(5, _omitFieldNames ? '' : 'fallbackTriggered')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AntiDpiStatusEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AntiDpiStatusEvent copyWith(void Function(AntiDpiStatusEvent) updates) =>
      super.copyWith((message) => updates(message as AntiDpiStatusEvent))
          as AntiDpiStatusEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use AntiDpiStatusEvent() / AntiDpiStatusEvent.new instead')
  static AntiDpiStatusEvent create() => AntiDpiStatusEvent._();
  static $pb.GeneratedMessage $_createMessage() => AntiDpiStatusEvent._();
  @$core.override
  AntiDpiStatusEvent createEmptyInstance() => AntiDpiStatusEvent._();
  @$core.pragma('dart2js:noInline')
  static AntiDpiStatusEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AntiDpiStatusEvent>(
          AntiDpiStatusEvent.$_createMessage);
  static AntiDpiStatusEvent? _defaultInstance;

  @$pb.TagNumber(1)
  AntiDpiMode get mode => $_getN(0);
  @$pb.TagNumber(1)
  set mode(AntiDpiMode value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasMode() => $_has(0);
  @$pb.TagNumber(1)
  void clearMode() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get currentSni => $_getSZ(1);
  @$pb.TagNumber(2)
  set currentSni($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCurrentSni() => $_has(1);
  @$pb.TagNumber(2)
  void clearCurrentSni() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get handshakeSuccess => $_getBF(2);
  @$pb.TagNumber(3)
  set handshakeSuccess($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasHandshakeSuccess() => $_has(2);
  @$pb.TagNumber(3)
  void clearHandshakeSuccess() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get transportDetails => $_getSZ(3);
  @$pb.TagNumber(4)
  set transportDetails($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasTransportDetails() => $_has(3);
  @$pb.TagNumber(4)
  void clearTransportDetails() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get fallbackTriggered => $_getBF(4);
  @$pb.TagNumber(5)
  set fallbackTriggered($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasFallbackTriggered() => $_has(4);
  @$pb.TagNumber(5)
  void clearFallbackTriggered() => $_clearField(5);
}

class IncomingMessageEvent extends $pb.GeneratedMessage {
  factory IncomingMessageEvent({
    ChatMessage? message,
    $core.bool? sealedSenderVerified,
    $fixnum.Int64? hopCount,
  }) {
    final result = IncomingMessageEvent._();
    if (message != null) result.message = message;
    if (sealedSenderVerified != null)
      result.sealedSenderVerified = sealedSenderVerified;
    if (hopCount != null) result.hopCount = hopCount;
    return result;
  }

  IncomingMessageEvent._();

  factory IncomingMessageEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      IncomingMessageEvent()..mergeFromBuffer(data, registry);
  factory IncomingMessageEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      IncomingMessageEvent()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'IncomingMessageEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: IncomingMessageEvent.$_createMessage)
    ..aOM<ChatMessage>(1, _omitFieldNames ? '' : 'message',
        subBuilder: ChatMessage.$_createMessage)
    ..aOB(2, _omitFieldNames ? '' : 'sealedSenderVerified')
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'hopCount', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  IncomingMessageEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  IncomingMessageEvent copyWith(void Function(IncomingMessageEvent) updates) =>
      super.copyWith((message) => updates(message as IncomingMessageEvent))
          as IncomingMessageEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use IncomingMessageEvent() / IncomingMessageEvent.new instead')
  static IncomingMessageEvent create() => IncomingMessageEvent._();
  static $pb.GeneratedMessage $_createMessage() => IncomingMessageEvent._();
  @$core.override
  IncomingMessageEvent createEmptyInstance() => IncomingMessageEvent._();
  @$core.pragma('dart2js:noInline')
  static IncomingMessageEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<IncomingMessageEvent>(
          IncomingMessageEvent.$_createMessage);
  static IncomingMessageEvent? _defaultInstance;

  @$pb.TagNumber(1)
  ChatMessage get message => $_getN(0);
  @$pb.TagNumber(1)
  set message(ChatMessage value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasMessage() => $_has(0);
  @$pb.TagNumber(1)
  void clearMessage() => $_clearField(1);
  @$pb.TagNumber(1)
  ChatMessage ensureMessage() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.bool get sealedSenderVerified => $_getBF(1);
  @$pb.TagNumber(2)
  set sealedSenderVerified($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSealedSenderVerified() => $_has(1);
  @$pb.TagNumber(2)
  void clearSealedSenderVerified() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get hopCount => $_getI64(2);
  @$pb.TagNumber(3)
  set hopCount($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasHopCount() => $_has(2);
  @$pb.TagNumber(3)
  void clearHopCount() => $_clearField(3);
}

/// Судьба исходящего сообщения. Этапы отделены друг от друга, потому что
/// «оверлей принял пакет» и «адресат расшифровал» — разные утверждения:
/// первое ничего о получателе не знает.
class MessageStatusEvent extends $pb.GeneratedMessage {
  factory MessageStatusEvent({
    $core.String? messageId,
    $core.String? conversationId,
    $core.bool? writtenToOverlay,
    $core.bool? delivered,
    $fixnum.Int64? ackTimestampMs,
  }) {
    final result = MessageStatusEvent._();
    if (messageId != null) result.messageId = messageId;
    if (conversationId != null) result.conversationId = conversationId;
    if (writtenToOverlay != null) result.writtenToOverlay = writtenToOverlay;
    if (delivered != null) result.delivered = delivered;
    if (ackTimestampMs != null) result.ackTimestampMs = ackTimestampMs;
    return result;
  }

  MessageStatusEvent._();

  factory MessageStatusEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      MessageStatusEvent()..mergeFromBuffer(data, registry);
  factory MessageStatusEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      MessageStatusEvent()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'MessageStatusEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: MessageStatusEvent.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'messageId')
    ..aOS(2, _omitFieldNames ? '' : 'conversationId')
    ..aOB(3, _omitFieldNames ? '' : 'writtenToOverlay')
    ..aOB(4, _omitFieldNames ? '' : 'delivered')
    ..a<$fixnum.Int64>(
        5, _omitFieldNames ? '' : 'ackTimestampMs', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MessageStatusEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MessageStatusEvent copyWith(void Function(MessageStatusEvent) updates) =>
      super.copyWith((message) => updates(message as MessageStatusEvent))
          as MessageStatusEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use MessageStatusEvent() / MessageStatusEvent.new instead')
  static MessageStatusEvent create() => MessageStatusEvent._();
  static $pb.GeneratedMessage $_createMessage() => MessageStatusEvent._();
  @$core.override
  MessageStatusEvent createEmptyInstance() => MessageStatusEvent._();
  @$core.pragma('dart2js:noInline')
  static MessageStatusEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<MessageStatusEvent>(
          MessageStatusEvent.$_createMessage);
  static MessageStatusEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get messageId => $_getSZ(0);
  @$pb.TagNumber(1)
  set messageId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMessageId() => $_has(0);
  @$pb.TagNumber(1)
  void clearMessageId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get conversationId => $_getSZ(1);
  @$pb.TagNumber(2)
  set conversationId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasConversationId() => $_has(1);
  @$pb.TagNumber(2)
  void clearConversationId() => $_clearField(2);

  /// Пакет ушёл в оверлей, когда spanning-дерево знало маршрут до ключа.
  @$pb.TagNumber(3)
  $core.bool get writtenToOverlay => $_getBF(2);
  @$pb.TagNumber(3)
  set writtenToOverlay($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasWrittenToOverlay() => $_has(2);
  @$pb.TagNumber(3)
  void clearWrittenToOverlay() => $_clearField(3);

  /// Получатель прислал DeliveryAckContent из открытого сеанса.
  @$pb.TagNumber(4)
  $core.bool get delivered => $_getBF(3);
  @$pb.TagNumber(4)
  set delivered($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDelivered() => $_has(3);
  @$pb.TagNumber(4)
  void clearDelivered() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get ackTimestampMs => $_getI64(4);
  @$pb.TagNumber(5)
  set ackTimestampMs($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasAckTimestampMs() => $_has(4);
  @$pb.TagNumber(5)
  void clearAckTimestampMs() => $_clearField(5);
}

class BackupProgressEvent extends $pb.GeneratedMessage {
  factory BackupProgressEvent({
    $fixnum.Int64? backupId,
    $core.int? completedChunks,
    $core.int? totalChunks,
    $core.double? progressPercentage,
    $core.String? currentAction,
    $core.bool? isFinished,
    $core.String? error,
  }) {
    final result = BackupProgressEvent._();
    if (backupId != null) result.backupId = backupId;
    if (completedChunks != null) result.completedChunks = completedChunks;
    if (totalChunks != null) result.totalChunks = totalChunks;
    if (progressPercentage != null)
      result.progressPercentage = progressPercentage;
    if (currentAction != null) result.currentAction = currentAction;
    if (isFinished != null) result.isFinished = isFinished;
    if (error != null) result.error = error;
    return result;
  }

  BackupProgressEvent._();

  factory BackupProgressEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      BackupProgressEvent()..mergeFromBuffer(data, registry);
  factory BackupProgressEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      BackupProgressEvent()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'BackupProgressEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: BackupProgressEvent.$_createMessage)
    ..a<$fixnum.Int64>(
        1, _omitFieldNames ? '' : 'backupId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aI(2, _omitFieldNames ? '' : 'completedChunks',
        fieldType: $pb.PbFieldType.OU3)
    ..aI(3, _omitFieldNames ? '' : 'totalChunks',
        fieldType: $pb.PbFieldType.OU3)
    ..aD(4, _omitFieldNames ? '' : 'progressPercentage',
        fieldType: $pb.PbFieldType.OF)
    ..aOS(5, _omitFieldNames ? '' : 'currentAction')
    ..aOB(6, _omitFieldNames ? '' : 'isFinished')
    ..aOS(7, _omitFieldNames ? '' : 'error')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BackupProgressEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BackupProgressEvent copyWith(void Function(BackupProgressEvent) updates) =>
      super.copyWith((message) => updates(message as BackupProgressEvent))
          as BackupProgressEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core
      .Deprecated('Use BackupProgressEvent() / BackupProgressEvent.new instead')
  static BackupProgressEvent create() => BackupProgressEvent._();
  static $pb.GeneratedMessage $_createMessage() => BackupProgressEvent._();
  @$core.override
  BackupProgressEvent createEmptyInstance() => BackupProgressEvent._();
  @$core.pragma('dart2js:noInline')
  static BackupProgressEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<BackupProgressEvent>(
          BackupProgressEvent.$_createMessage);
  static BackupProgressEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get backupId => $_getI64(0);
  @$pb.TagNumber(1)
  set backupId($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasBackupId() => $_has(0);
  @$pb.TagNumber(1)
  void clearBackupId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get completedChunks => $_getIZ(1);
  @$pb.TagNumber(2)
  set completedChunks($core.int value) => $_setUnsignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCompletedChunks() => $_has(1);
  @$pb.TagNumber(2)
  void clearCompletedChunks() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get totalChunks => $_getIZ(2);
  @$pb.TagNumber(3)
  set totalChunks($core.int value) => $_setUnsignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTotalChunks() => $_has(2);
  @$pb.TagNumber(3)
  void clearTotalChunks() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.double get progressPercentage => $_getN(3);
  @$pb.TagNumber(4)
  set progressPercentage($core.double value) => $_setFloat(3, value);
  @$pb.TagNumber(4)
  $core.bool hasProgressPercentage() => $_has(3);
  @$pb.TagNumber(4)
  void clearProgressPercentage() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get currentAction => $_getSZ(4);
  @$pb.TagNumber(5)
  set currentAction($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasCurrentAction() => $_has(4);
  @$pb.TagNumber(5)
  void clearCurrentAction() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get isFinished => $_getBF(5);
  @$pb.TagNumber(6)
  set isFinished($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasIsFinished() => $_has(5);
  @$pb.TagNumber(6)
  void clearIsFinished() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get error => $_getSZ(6);
  @$pb.TagNumber(7)
  set error($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasError() => $_has(6);
  @$pb.TagNumber(7)
  void clearError() => $_clearField(7);
}

class IncomingCallEvent extends $pb.GeneratedMessage {
  factory IncomingCallEvent({
    $core.String? callId,
    $core.String? callerIpv6,
    ContactCard? callerContact,
    $core.bool? hasVideo,
    CallSignalContent? signal,
  }) {
    final result = IncomingCallEvent._();
    if (callId != null) result.callId = callId;
    if (callerIpv6 != null) result.callerIpv6 = callerIpv6;
    if (callerContact != null) result.callerContact = callerContact;
    if (hasVideo != null) result.hasVideo = hasVideo;
    if (signal != null) result.signal = signal;
    return result;
  }

  IncomingCallEvent._();

  factory IncomingCallEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      IncomingCallEvent()..mergeFromBuffer(data, registry);
  factory IncomingCallEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      IncomingCallEvent()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'IncomingCallEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: IncomingCallEvent.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'callId')
    ..aOS(2, _omitFieldNames ? '' : 'callerIpv6')
    ..aOM<ContactCard>(3, _omitFieldNames ? '' : 'callerContact',
        subBuilder: ContactCard.$_createMessage)
    ..aOB(4, _omitFieldNames ? '' : 'hasVideo')
    ..aOM<CallSignalContent>(5, _omitFieldNames ? '' : 'signal',
        subBuilder: CallSignalContent.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  IncomingCallEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  IncomingCallEvent copyWith(void Function(IncomingCallEvent) updates) =>
      super.copyWith((message) => updates(message as IncomingCallEvent))
          as IncomingCallEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use IncomingCallEvent() / IncomingCallEvent.new instead')
  static IncomingCallEvent create() => IncomingCallEvent._();
  static $pb.GeneratedMessage $_createMessage() => IncomingCallEvent._();
  @$core.override
  IncomingCallEvent createEmptyInstance() => IncomingCallEvent._();
  @$core.pragma('dart2js:noInline')
  static IncomingCallEvent getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<IncomingCallEvent>(
          IncomingCallEvent.$_createMessage);
  static IncomingCallEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get callId => $_getSZ(0);
  @$pb.TagNumber(1)
  set callId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCallId() => $_has(0);
  @$pb.TagNumber(1)
  void clearCallId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get callerIpv6 => $_getSZ(1);
  @$pb.TagNumber(2)
  set callerIpv6($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCallerIpv6() => $_has(1);
  @$pb.TagNumber(2)
  void clearCallerIpv6() => $_clearField(2);

  @$pb.TagNumber(3)
  ContactCard get callerContact => $_getN(2);
  @$pb.TagNumber(3)
  set callerContact(ContactCard value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasCallerContact() => $_has(2);
  @$pb.TagNumber(3)
  void clearCallerContact() => $_clearField(3);
  @$pb.TagNumber(3)
  ContactCard ensureCallerContact() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.bool get hasVideo => $_getBF(3);
  @$pb.TagNumber(4)
  set hasVideo($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasHasVideo() => $_has(3);
  @$pb.TagNumber(4)
  void clearHasVideo() => $_clearField(4);

  @$pb.TagNumber(5)
  CallSignalContent get signal => $_getN(4);
  @$pb.TagNumber(5)
  set signal(CallSignalContent value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasSignal() => $_has(4);
  @$pb.TagNumber(5)
  void clearSignal() => $_clearField(5);
  @$pb.TagNumber(5)
  CallSignalContent ensureSignal() => $_ensure(4);
}

enum CoreEvent_Event {
  networkStatus,
  peersUpdated,
  antiDpiStatus,
  incomingMessage,
  backupProgress,
  incomingCall,
  messageStatus,
  contactUpdated,
  notSet
}

class CoreEvent extends $pb.GeneratedMessage {
  factory CoreEvent({
    $fixnum.Int64? eventId,
    $fixnum.Int64? timestampMs,
    NetworkStatusEvent? networkStatus,
    PeerListUpdatedEvent? peersUpdated,
    AntiDpiStatusEvent? antiDpiStatus,
    IncomingMessageEvent? incomingMessage,
    BackupProgressEvent? backupProgress,
    IncomingCallEvent? incomingCall,
    MessageStatusEvent? messageStatus,
    ContactResolvedEvent? contactUpdated,
  }) {
    final result = CoreEvent._();
    if (eventId != null) result.eventId = eventId;
    if (timestampMs != null) result.timestampMs = timestampMs;
    if (networkStatus != null) result.networkStatus = networkStatus;
    if (peersUpdated != null) result.peersUpdated = peersUpdated;
    if (antiDpiStatus != null) result.antiDpiStatus = antiDpiStatus;
    if (incomingMessage != null) result.incomingMessage = incomingMessage;
    if (backupProgress != null) result.backupProgress = backupProgress;
    if (incomingCall != null) result.incomingCall = incomingCall;
    if (messageStatus != null) result.messageStatus = messageStatus;
    if (contactUpdated != null) result.contactUpdated = contactUpdated;
    return result;
  }

  CoreEvent._();

  factory CoreEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CoreEvent()..mergeFromBuffer(data, registry);
  factory CoreEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CoreEvent()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, CoreEvent_Event> _CoreEvent_EventByTag = {
    3: CoreEvent_Event.networkStatus,
    4: CoreEvent_Event.peersUpdated,
    5: CoreEvent_Event.antiDpiStatus,
    6: CoreEvent_Event.incomingMessage,
    7: CoreEvent_Event.backupProgress,
    8: CoreEvent_Event.incomingCall,
    9: CoreEvent_Event.messageStatus,
    10: CoreEvent_Event.contactUpdated,
    0: CoreEvent_Event.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CoreEvent',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'ygggram'),
      createEmptyInstance: CoreEvent.$_createMessage)
    ..oo(0, [3, 4, 5, 6, 7, 8, 9, 10])
    ..a<$fixnum.Int64>(1, _omitFieldNames ? '' : 'eventId', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..a<$fixnum.Int64>(
        2, _omitFieldNames ? '' : 'timestampMs', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<NetworkStatusEvent>(3, _omitFieldNames ? '' : 'networkStatus',
        subBuilder: NetworkStatusEvent.$_createMessage)
    ..aOM<PeerListUpdatedEvent>(4, _omitFieldNames ? '' : 'peersUpdated',
        subBuilder: PeerListUpdatedEvent.$_createMessage)
    ..aOM<AntiDpiStatusEvent>(5, _omitFieldNames ? '' : 'antiDpiStatus',
        subBuilder: AntiDpiStatusEvent.$_createMessage)
    ..aOM<IncomingMessageEvent>(6, _omitFieldNames ? '' : 'incomingMessage',
        subBuilder: IncomingMessageEvent.$_createMessage)
    ..aOM<BackupProgressEvent>(7, _omitFieldNames ? '' : 'backupProgress',
        subBuilder: BackupProgressEvent.$_createMessage)
    ..aOM<IncomingCallEvent>(8, _omitFieldNames ? '' : 'incomingCall',
        subBuilder: IncomingCallEvent.$_createMessage)
    ..aOM<MessageStatusEvent>(9, _omitFieldNames ? '' : 'messageStatus',
        subBuilder: MessageStatusEvent.$_createMessage)
    ..aOM<ContactResolvedEvent>(10, _omitFieldNames ? '' : 'contactUpdated',
        subBuilder: ContactResolvedEvent.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CoreEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CoreEvent copyWith(void Function(CoreEvent) updates) =>
      super.copyWith((message) => updates(message as CoreEvent)) as CoreEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use CoreEvent() / CoreEvent.new instead')
  static CoreEvent create() => CoreEvent._();
  static $pb.GeneratedMessage $_createMessage() => CoreEvent._();
  @$core.override
  CoreEvent createEmptyInstance() => CoreEvent._();
  @$core.pragma('dart2js:noInline')
  static CoreEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CoreEvent>(CoreEvent.$_createMessage);
  static CoreEvent? _defaultInstance;

  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  @$pb.TagNumber(6)
  @$pb.TagNumber(7)
  @$pb.TagNumber(8)
  @$pb.TagNumber(9)
  @$pb.TagNumber(10)
  CoreEvent_Event whichEvent() => _CoreEvent_EventByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  @$pb.TagNumber(6)
  @$pb.TagNumber(7)
  @$pb.TagNumber(8)
  @$pb.TagNumber(9)
  @$pb.TagNumber(10)
  void clearEvent() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $fixnum.Int64 get eventId => $_getI64(0);
  @$pb.TagNumber(1)
  set eventId($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEventId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEventId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get timestampMs => $_getI64(1);
  @$pb.TagNumber(2)
  set timestampMs($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTimestampMs() => $_has(1);
  @$pb.TagNumber(2)
  void clearTimestampMs() => $_clearField(2);

  @$pb.TagNumber(3)
  NetworkStatusEvent get networkStatus => $_getN(2);
  @$pb.TagNumber(3)
  set networkStatus(NetworkStatusEvent value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasNetworkStatus() => $_has(2);
  @$pb.TagNumber(3)
  void clearNetworkStatus() => $_clearField(3);
  @$pb.TagNumber(3)
  NetworkStatusEvent ensureNetworkStatus() => $_ensure(2);

  @$pb.TagNumber(4)
  PeerListUpdatedEvent get peersUpdated => $_getN(3);
  @$pb.TagNumber(4)
  set peersUpdated(PeerListUpdatedEvent value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasPeersUpdated() => $_has(3);
  @$pb.TagNumber(4)
  void clearPeersUpdated() => $_clearField(4);
  @$pb.TagNumber(4)
  PeerListUpdatedEvent ensurePeersUpdated() => $_ensure(3);

  @$pb.TagNumber(5)
  AntiDpiStatusEvent get antiDpiStatus => $_getN(4);
  @$pb.TagNumber(5)
  set antiDpiStatus(AntiDpiStatusEvent value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasAntiDpiStatus() => $_has(4);
  @$pb.TagNumber(5)
  void clearAntiDpiStatus() => $_clearField(5);
  @$pb.TagNumber(5)
  AntiDpiStatusEvent ensureAntiDpiStatus() => $_ensure(4);

  @$pb.TagNumber(6)
  IncomingMessageEvent get incomingMessage => $_getN(5);
  @$pb.TagNumber(6)
  set incomingMessage(IncomingMessageEvent value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasIncomingMessage() => $_has(5);
  @$pb.TagNumber(6)
  void clearIncomingMessage() => $_clearField(6);
  @$pb.TagNumber(6)
  IncomingMessageEvent ensureIncomingMessage() => $_ensure(5);

  @$pb.TagNumber(7)
  BackupProgressEvent get backupProgress => $_getN(6);
  @$pb.TagNumber(7)
  set backupProgress(BackupProgressEvent value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasBackupProgress() => $_has(6);
  @$pb.TagNumber(7)
  void clearBackupProgress() => $_clearField(7);
  @$pb.TagNumber(7)
  BackupProgressEvent ensureBackupProgress() => $_ensure(6);

  @$pb.TagNumber(8)
  IncomingCallEvent get incomingCall => $_getN(7);
  @$pb.TagNumber(8)
  set incomingCall(IncomingCallEvent value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasIncomingCall() => $_has(7);
  @$pb.TagNumber(8)
  void clearIncomingCall() => $_clearField(8);
  @$pb.TagNumber(8)
  IncomingCallEvent ensureIncomingCall() => $_ensure(7);

  @$pb.TagNumber(9)
  MessageStatusEvent get messageStatus => $_getN(8);
  @$pb.TagNumber(9)
  set messageStatus(MessageStatusEvent value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasMessageStatus() => $_has(8);
  @$pb.TagNumber(9)
  void clearMessageStatus() => $_clearField(9);
  @$pb.TagNumber(9)
  MessageStatusEvent ensureMessageStatus() => $_ensure(8);

  @$pb.TagNumber(10)
  ContactResolvedEvent get contactUpdated => $_getN(9);
  @$pb.TagNumber(10)
  set contactUpdated(ContactResolvedEvent value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasContactUpdated() => $_has(9);
  @$pb.TagNumber(10)
  void clearContactUpdated() => $_clearField(10);
  @$pb.TagNumber(10)
  ContactResolvedEvent ensureContactUpdated() => $_ensure(9);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
