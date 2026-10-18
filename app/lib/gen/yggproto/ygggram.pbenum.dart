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

import 'package:protobuf/protobuf.dart' as $pb;

class AntiDpiMode extends $pb.ProtobufEnum {
  static const AntiDpiMode ANTI_DPI_DISABLED =
      AntiDpiMode._(0, _omitEnumNames ? '' : 'ANTI_DPI_DISABLED');
  static const AntiDpiMode ANTI_DPI_UTLS_CHROME =
      AntiDpiMode._(1, _omitEnumNames ? '' : 'ANTI_DPI_UTLS_CHROME');
  static const AntiDpiMode ANTI_DPI_UTLS_IOS =
      AntiDpiMode._(2, _omitEnumNames ? '' : 'ANTI_DPI_UTLS_IOS');
  static const AntiDpiMode ANTI_DPI_SHADOW_TLS_V3 =
      AntiDpiMode._(3, _omitEnumNames ? '' : 'ANTI_DPI_SHADOW_TLS_V3');
  static const AntiDpiMode ANTI_DPI_WEBSOCKET_CDN =
      AntiDpiMode._(4, _omitEnumNames ? '' : 'ANTI_DPI_WEBSOCKET_CDN');
  static const AntiDpiMode ANTI_DPI_UDP_QUIC_OBFS =
      AntiDpiMode._(5, _omitEnumNames ? '' : 'ANTI_DPI_UDP_QUIC_OBFS');

  static const $core.List<AntiDpiMode> values = <AntiDpiMode>[
    ANTI_DPI_DISABLED,
    ANTI_DPI_UTLS_CHROME,
    ANTI_DPI_UTLS_IOS,
    ANTI_DPI_SHADOW_TLS_V3,
    ANTI_DPI_WEBSOCKET_CDN,
    ANTI_DPI_UDP_QUIC_OBFS,
  ];

  static final $core.List<AntiDpiMode?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 5);
  static AntiDpiMode? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const AntiDpiMode._(super.value, super.name);
}

class PeerTransportType extends $pb.ProtobufEnum {
  static const PeerTransportType TRANSPORT_TCP =
      PeerTransportType._(0, _omitEnumNames ? '' : 'TRANSPORT_TCP');
  static const PeerTransportType TRANSPORT_TLS_UTLS =
      PeerTransportType._(1, _omitEnumNames ? '' : 'TRANSPORT_TLS_UTLS');
  static const PeerTransportType TRANSPORT_WSS =
      PeerTransportType._(2, _omitEnumNames ? '' : 'TRANSPORT_WSS');
  static const PeerTransportType TRANSPORT_SHADOWTLS =
      PeerTransportType._(3, _omitEnumNames ? '' : 'TRANSPORT_SHADOWTLS');
  static const PeerTransportType TRANSPORT_QUIC_OBFS =
      PeerTransportType._(4, _omitEnumNames ? '' : 'TRANSPORT_QUIC_OBFS');

  static const $core.List<PeerTransportType> values = <PeerTransportType>[
    TRANSPORT_TCP,
    TRANSPORT_TLS_UTLS,
    TRANSPORT_WSS,
    TRANSPORT_SHADOWTLS,
    TRANSPORT_QUIC_OBFS,
  ];

  static final $core.List<PeerTransportType?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 4);
  static PeerTransportType? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const PeerTransportType._(super.value, super.name);
}

class CallSignalType extends $pb.ProtobufEnum {
  static const CallSignalType CALL_SIGNAL_OFFER =
      CallSignalType._(0, _omitEnumNames ? '' : 'CALL_SIGNAL_OFFER');
  static const CallSignalType CALL_SIGNAL_ANSWER =
      CallSignalType._(1, _omitEnumNames ? '' : 'CALL_SIGNAL_ANSWER');
  static const CallSignalType CALL_SIGNAL_ICE_CANDIDATE =
      CallSignalType._(2, _omitEnumNames ? '' : 'CALL_SIGNAL_ICE_CANDIDATE');
  static const CallSignalType CALL_SIGNAL_HANGUP =
      CallSignalType._(3, _omitEnumNames ? '' : 'CALL_SIGNAL_HANGUP');
  static const CallSignalType CALL_SIGNAL_REJECT =
      CallSignalType._(4, _omitEnumNames ? '' : 'CALL_SIGNAL_REJECT');

  static const $core.List<CallSignalType> values = <CallSignalType>[
    CALL_SIGNAL_OFFER,
    CALL_SIGNAL_ANSWER,
    CALL_SIGNAL_ICE_CANDIDATE,
    CALL_SIGNAL_HANGUP,
    CALL_SIGNAL_REJECT,
  ];

  static final $core.List<CallSignalType?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 4);
  static CallSignalType? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const CallSignalType._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
