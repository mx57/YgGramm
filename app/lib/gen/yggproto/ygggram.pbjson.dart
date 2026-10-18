// This is a generated file - do not edit.
//
// Generated from ygggram.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports
// ignore_for_file: unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use antiDpiModeDescriptor instead')
const AntiDpiMode$json = {
  '1': 'AntiDpiMode',
  '2': [
    {'1': 'ANTI_DPI_DISABLED', '2': 0},
    {'1': 'ANTI_DPI_UTLS_CHROME', '2': 1},
    {'1': 'ANTI_DPI_UTLS_IOS', '2': 2},
    {'1': 'ANTI_DPI_SHADOW_TLS_V3', '2': 3},
    {'1': 'ANTI_DPI_WEBSOCKET_CDN', '2': 4},
    {'1': 'ANTI_DPI_UDP_QUIC_OBFS', '2': 5},
  ],
};

/// Descriptor for `AntiDpiMode`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List antiDpiModeDescriptor = $convert.base64Decode(
    'CgtBbnRpRHBpTW9kZRIVChFBTlRJX0RQSV9ESVNBQkxFRBAAEhgKFEFOVElfRFBJX1VUTFNfQ0'
    'hST01FEAESFQoRQU5USV9EUElfVVRMU19JT1MQAhIaChZBTlRJX0RQSV9TSEFET1dfVExTX1Yz'
    'EAMSGgoWQU5USV9EUElfV0VCU09DS0VUX0NEThAEEhoKFkFOVElfRFBJX1VEUF9RVUlDX09CRl'
    'MQBQ==');

@$core.Deprecated('Use peerTransportTypeDescriptor instead')
const PeerTransportType$json = {
  '1': 'PeerTransportType',
  '2': [
    {'1': 'TRANSPORT_TCP', '2': 0},
    {'1': 'TRANSPORT_TLS_UTLS', '2': 1},
    {'1': 'TRANSPORT_WSS', '2': 2},
    {'1': 'TRANSPORT_SHADOWTLS', '2': 3},
    {'1': 'TRANSPORT_QUIC_OBFS', '2': 4},
  ],
};

/// Descriptor for `PeerTransportType`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List peerTransportTypeDescriptor = $convert.base64Decode(
    'ChFQZWVyVHJhbnNwb3J0VHlwZRIRCg1UUkFOU1BPUlRfVENQEAASFgoSVFJBTlNQT1JUX1RMU1'
    '9VVExTEAESEQoNVFJBTlNQT1JUX1dTUxACEhcKE1RSQU5TUE9SVF9TSEFET1dUTFMQAxIXChNU'
    'UkFOU1BPUlRfUVVJQ19PQkZTEAQ=');

@$core.Deprecated('Use callSignalTypeDescriptor instead')
const CallSignalType$json = {
  '1': 'CallSignalType',
  '2': [
    {'1': 'CALL_SIGNAL_OFFER', '2': 0},
    {'1': 'CALL_SIGNAL_ANSWER', '2': 1},
    {'1': 'CALL_SIGNAL_ICE_CANDIDATE', '2': 2},
    {'1': 'CALL_SIGNAL_HANGUP', '2': 3},
    {'1': 'CALL_SIGNAL_REJECT', '2': 4},
  ],
};

/// Descriptor for `CallSignalType`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List callSignalTypeDescriptor = $convert.base64Decode(
    'Cg5DYWxsU2lnbmFsVHlwZRIVChFDQUxMX1NJR05BTF9PRkZFUhAAEhYKEkNBTExfU0lHTkFMX0'
    'FOU1dFUhABEh0KGUNBTExfU0lHTkFMX0lDRV9DQU5ESURBVEUQAhIWChJDQUxMX1NJR05BTF9I'
    'QU5HVVAQAxIWChJDQUxMX1NJR05BTF9SRUpFQ1QQBA==');

@$core.Deprecated('Use nodeConfigDescriptor instead')
const NodeConfig$json = {
  '1': 'NodeConfig',
  '2': [
    {'1': 'data_dir', '3': 1, '4': 1, '5': 9, '10': 'dataDir'},
    {'1': 'cache_dir', '3': 2, '4': 1, '5': 9, '10': 'cacheDir'},
    {'1': 'mnemonic_seed', '3': 3, '4': 1, '5': 9, '10': 'mnemonicSeed'},
    {
      '1': 'ed25519_node_privkey',
      '3': 4,
      '4': 1,
      '5': 12,
      '10': 'ed25519NodePrivkey'
    },
    {
      '1': 'x25519_chat_privkey',
      '3': 5,
      '4': 1,
      '5': 12,
      '10': 'x25519ChatPrivkey'
    },
    {'1': 'master_key_256', '3': 6, '4': 1, '5': 12, '10': 'masterKey256'},
    {
      '1': 'anti_dpi_mode',
      '3': 7,
      '4': 1,
      '5': 14,
      '6': '.ygggram.AntiDpiMode',
      '10': 'antiDpiMode'
    },
    {'1': 'anti_dpi_sni', '3': 8, '4': 1, '5': 9, '10': 'antiDpiSni'},
    {'1': 'anti_dpi_password', '3': 9, '4': 1, '5': 9, '10': 'antiDpiPassword'},
    {'1': 'bootstrap_peers', '3': 10, '4': 3, '5': 9, '10': 'bootstrapPeers'},
    {'1': 'doh_servers', '3': 11, '4': 3, '5': 9, '10': 'dohServers'},
    {'1': 'enable_dht_bep44', '3': 12, '4': 1, '5': 8, '10': 'enableDhtBep44'},
    {'1': 'listen_address', '3': 13, '4': 1, '5': 9, '10': 'listenAddress'},
    {'1': 'listen_port', '3': 14, '4': 1, '5': 5, '10': 'listenPort'},
    {'1': 'userspace_only', '3': 15, '4': 1, '5': 8, '10': 'userspaceOnly'},
    {
      '1': 'connection_timeout_seconds',
      '3': 16,
      '4': 1,
      '5': 13,
      '10': 'connectionTimeoutSeconds'
    },
    {
      '1': 'keepalive_interval_seconds',
      '3': 17,
      '4': 1,
      '5': 13,
      '10': 'keepaliveIntervalSeconds'
    },
    {
      '1': 'enable_lan_multicast',
      '3': 18,
      '4': 1,
      '5': 8,
      '10': 'enableLanMulticast'
    },
    {'1': 'sf_relay_enabled', '3': 19, '4': 1, '5': 8, '10': 'sfRelayEnabled'},
    {
      '1': 'sf_relay_node_keys',
      '3': 20,
      '4': 3,
      '5': 12,
      '10': 'sfRelayNodeKeys'
    },
    {
      '1': 'serve_prekey_bundles',
      '3': 21,
      '4': 1,
      '5': 8,
      '9': 0,
      '10': 'servePrekeyBundles',
      '17': true
    },
  ],
  '8': [
    {'1': '_serve_prekey_bundles'},
  ],
};

/// Descriptor for `NodeConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List nodeConfigDescriptor = $convert.base64Decode(
    'CgpOb2RlQ29uZmlnEhkKCGRhdGFfZGlyGAEgASgJUgdkYXRhRGlyEhsKCWNhY2hlX2RpchgCIA'
    'EoCVIIY2FjaGVEaXISIwoNbW5lbW9uaWNfc2VlZBgDIAEoCVIMbW5lbW9uaWNTZWVkEjAKFGVk'
    'MjU1MTlfbm9kZV9wcml2a2V5GAQgASgMUhJlZDI1NTE5Tm9kZVByaXZrZXkSLgoTeDI1NTE5X2'
    'NoYXRfcHJpdmtleRgFIAEoDFIReDI1NTE5Q2hhdFByaXZrZXkSJAoObWFzdGVyX2tleV8yNTYY'
    'BiABKAxSDG1hc3RlcktleTI1NhI4Cg1hbnRpX2RwaV9tb2RlGAcgASgOMhQueWdnZ3JhbS5Bbn'
    'RpRHBpTW9kZVILYW50aURwaU1vZGUSIAoMYW50aV9kcGlfc25pGAggASgJUgphbnRpRHBpU25p'
    'EioKEWFudGlfZHBpX3Bhc3N3b3JkGAkgASgJUg9hbnRpRHBpUGFzc3dvcmQSJwoPYm9vdHN0cm'
    'FwX3BlZXJzGAogAygJUg5ib290c3RyYXBQZWVycxIfCgtkb2hfc2VydmVycxgLIAMoCVIKZG9o'
    'U2VydmVycxIoChBlbmFibGVfZGh0X2JlcDQ0GAwgASgIUg5lbmFibGVEaHRCZXA0NBIlCg5saX'
    'N0ZW5fYWRkcmVzcxgNIAEoCVINbGlzdGVuQWRkcmVzcxIfCgtsaXN0ZW5fcG9ydBgOIAEoBVIK'
    'bGlzdGVuUG9ydBIlCg51c2Vyc3BhY2Vfb25seRgPIAEoCFINdXNlcnNwYWNlT25seRI8Chpjb2'
    '5uZWN0aW9uX3RpbWVvdXRfc2Vjb25kcxgQIAEoDVIYY29ubmVjdGlvblRpbWVvdXRTZWNvbmRz'
    'EjwKGmtlZXBhbGl2ZV9pbnRlcnZhbF9zZWNvbmRzGBEgASgNUhhrZWVwYWxpdmVJbnRlcnZhbF'
    'NlY29uZHMSMAoUZW5hYmxlX2xhbl9tdWx0aWNhc3QYEiABKAhSEmVuYWJsZUxhbk11bHRpY2Fz'
    'dBIoChBzZl9yZWxheV9lbmFibGVkGBMgASgIUg5zZlJlbGF5RW5hYmxlZBIrChJzZl9yZWxheV'
    '9ub2RlX2tleXMYFCADKAxSD3NmUmVsYXlOb2RlS2V5cxI1ChRzZXJ2ZV9wcmVrZXlfYnVuZGxl'
    'cxgVIAEoCEgAUhJzZXJ2ZVByZWtleUJ1bmRsZXOIAQFCFwoVX3NlcnZlX3ByZWtleV9idW5kbG'
    'Vz');

@$core.Deprecated('Use sealedEnvelopeDescriptor instead')
const SealedEnvelope$json = {
  '1': 'SealedEnvelope',
  '2': [
    {
      '1': 'recipient_x25519_pubkey',
      '3': 1,
      '4': 1,
      '5': 12,
      '10': 'recipientX25519Pubkey'
    },
    {
      '1': 'ephemeral_x25519_pubkey',
      '3': 2,
      '4': 1,
      '5': 12,
      '10': 'ephemeralX25519Pubkey'
    },
    {
      '1': 'encrypted_payload',
      '3': 3,
      '4': 1,
      '5': 12,
      '10': 'encryptedPayload'
    },
    {'1': 'nonce', '3': 4, '4': 1, '5': 12, '10': 'nonce'},
    {'1': 'timestamp', '3': 5, '4': 1, '5': 4, '10': 'timestamp'},
    {'1': 'pow_nonce', '3': 6, '4': 1, '5': 4, '10': 'powNonce'},
    {'1': 'blind_token', '3': 7, '4': 1, '5': 12, '10': 'blindToken'},
    {'1': 'routing_hops', '3': 8, '4': 1, '5': 13, '10': 'routingHops'},
  ],
};

/// Descriptor for `SealedEnvelope`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sealedEnvelopeDescriptor = $convert.base64Decode(
    'Cg5TZWFsZWRFbnZlbG9wZRI2ChdyZWNpcGllbnRfeDI1NTE5X3B1YmtleRgBIAEoDFIVcmVjaX'
    'BpZW50WDI1NTE5UHVia2V5EjYKF2VwaGVtZXJhbF94MjU1MTlfcHVia2V5GAIgASgMUhVlcGhl'
    'bWVyYWxYMjU1MTlQdWJrZXkSKwoRZW5jcnlwdGVkX3BheWxvYWQYAyABKAxSEGVuY3J5cHRlZF'
    'BheWxvYWQSFAoFbm9uY2UYBCABKAxSBW5vbmNlEhwKCXRpbWVzdGFtcBgFIAEoBFIJdGltZXN0'
    'YW1wEhsKCXBvd19ub25jZRgGIAEoBFIIcG93Tm9uY2USHwoLYmxpbmRfdG9rZW4YByABKAxSCm'
    'JsaW5kVG9rZW4SIQoMcm91dGluZ19ob3BzGAggASgNUgtyb3V0aW5nSG9wcw==');

@$core.Deprecated('Use wireFrameDescriptor instead')
const WireFrame$json = {
  '1': 'WireFrame',
  '2': [
    {
      '1': 'chat',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.ygggram.SealedEnvelope',
      '9': 0,
      '10': 'chat'
    },
    {
      '1': 'sf_put',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.ygggram.SfPut',
      '9': 0,
      '10': 'sfPut'
    },
    {
      '1': 'sf_pull',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.ygggram.SfPull',
      '9': 0,
      '10': 'sfPull'
    },
    {
      '1': 'sf_reply',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.ygggram.SfReply',
      '9': 0,
      '10': 'sfReply'
    },
    {
      '1': 'bundle_req',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.ygggram.BundleReq',
      '9': 0,
      '10': 'bundleReq'
    },
    {
      '1': 'bundle_resp',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.ygggram.BundleResp',
      '9': 0,
      '10': 'bundleResp'
    },
  ],
  '8': [
    {'1': 'body'},
  ],
};

/// Descriptor for `WireFrame`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List wireFrameDescriptor = $convert.base64Decode(
    'CglXaXJlRnJhbWUSLQoEY2hhdBgBIAEoCzIXLnlnZ2dyYW0uU2VhbGVkRW52ZWxvcGVIAFIEY2'
    'hhdBInCgZzZl9wdXQYAiABKAsyDi55Z2dncmFtLlNmUHV0SABSBXNmUHV0EioKB3NmX3B1bGwY'
    'AyABKAsyDy55Z2dncmFtLlNmUHVsbEgAUgZzZlB1bGwSLQoIc2ZfcmVwbHkYBCABKAsyEC55Z2'
    'dncmFtLlNmUmVwbHlIAFIHc2ZSZXBseRIzCgpidW5kbGVfcmVxGAUgASgLMhIueWdnZ3JhbS5C'
    'dW5kbGVSZXFIAFIJYnVuZGxlUmVxEjYKC2J1bmRsZV9yZXNwGAYgASgLMhMueWdnZ3JhbS5CdW'
    '5kbGVSZXNwSABSCmJ1bmRsZVJlc3BCBgoEYm9keQ==');

@$core.Deprecated('Use wireIdentityProofDescriptor instead')
const WireIdentityProof$json = {
  '1': 'WireIdentityProof',
  '2': [
    {'1': 'master_ed25519', '3': 1, '4': 1, '5': 12, '10': 'masterEd25519'},
    {'1': 'master_x25519', '3': 2, '4': 1, '5': 12, '10': 'masterX25519'},
    {'1': 'binding', '3': 3, '4': 1, '5': 12, '10': 'binding'},
    {'1': 'device_id', '3': 4, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'device_name', '3': 5, '4': 1, '5': 9, '10': 'deviceName'},
    {'1': 'device_ed25519', '3': 6, '4': 1, '5': 12, '10': 'deviceEd25519'},
    {'1': 'device_x25519', '3': 7, '4': 1, '5': 12, '10': 'deviceX25519'},
    {'1': 'device_created_at', '3': 8, '4': 1, '5': 3, '10': 'deviceCreatedAt'},
    {'1': 'device_cert_sig', '3': 9, '4': 1, '5': 12, '10': 'deviceCertSig'},
  ],
};

/// Descriptor for `WireIdentityProof`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List wireIdentityProofDescriptor = $convert.base64Decode(
    'ChFXaXJlSWRlbnRpdHlQcm9vZhIlCg5tYXN0ZXJfZWQyNTUxORgBIAEoDFINbWFzdGVyRWQyNT'
    'UxORIjCg1tYXN0ZXJfeDI1NTE5GAIgASgMUgxtYXN0ZXJYMjU1MTkSGAoHYmluZGluZxgDIAEo'
    'DFIHYmluZGluZxIbCglkZXZpY2VfaWQYBCABKAlSCGRldmljZUlkEh8KC2RldmljZV9uYW1lGA'
    'UgASgJUgpkZXZpY2VOYW1lEiUKDmRldmljZV9lZDI1NTE5GAYgASgMUg1kZXZpY2VFZDI1NTE5'
    'EiMKDWRldmljZV94MjU1MTkYByABKAxSDGRldmljZVgyNTUxORIqChFkZXZpY2VfY3JlYXRlZF'
    '9hdBgIIAEoA1IPZGV2aWNlQ3JlYXRlZEF0EiYKD2RldmljZV9jZXJ0X3NpZxgJIAEoDFINZGV2'
    'aWNlQ2VydFNpZw==');

@$core.Deprecated('Use sfPutDescriptor instead')
const SfPut$json = {
  '1': 'SfPut',
  '2': [
    {'1': 'mailbox_id', '3': 1, '4': 1, '5': 12, '10': 'mailboxId'},
    {
      '1': 'envelope',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.ygggram.SealedEnvelope',
      '10': 'envelope'
    },
  ],
};

/// Descriptor for `SfPut`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sfPutDescriptor = $convert.base64Decode(
    'CgVTZlB1dBIdCgptYWlsYm94X2lkGAEgASgMUgltYWlsYm94SWQSMwoIZW52ZWxvcGUYAiABKA'
    'syFy55Z2dncmFtLlNlYWxlZEVudmVsb3BlUghlbnZlbG9wZQ==');

@$core.Deprecated('Use sfPullDescriptor instead')
const SfPull$json = {
  '1': 'SfPull',
  '2': [
    {'1': 'mailbox_id', '3': 1, '4': 1, '5': 12, '10': 'mailboxId'},
    {'1': 'epoch_day', '3': 2, '4': 1, '5': 3, '10': 'epochDay'},
    {'1': 'timestamp_s', '3': 3, '4': 1, '5': 3, '10': 'timestampS'},
    {'1': 'signature', '3': 4, '4': 1, '5': 12, '10': 'signature'},
    {
      '1': 'proof',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.ygggram.WireIdentityProof',
      '10': 'proof'
    },
  ],
};

/// Descriptor for `SfPull`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sfPullDescriptor = $convert.base64Decode(
    'CgZTZlB1bGwSHQoKbWFpbGJveF9pZBgBIAEoDFIJbWFpbGJveElkEhsKCWVwb2NoX2RheRgCIA'
    'EoA1IIZXBvY2hEYXkSHwoLdGltZXN0YW1wX3MYAyABKANSCnRpbWVzdGFtcFMSHAoJc2lnbmF0'
    'dXJlGAQgASgMUglzaWduYXR1cmUSMAoFcHJvb2YYBSABKAsyGi55Z2dncmFtLldpcmVJZGVudG'
    'l0eVByb29mUgVwcm9vZg==');

@$core.Deprecated('Use sfReplyDescriptor instead')
const SfReply$json = {
  '1': 'SfReply',
  '2': [
    {
      '1': 'envelopes',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.ygggram.SealedEnvelope',
      '10': 'envelopes'
    },
    {'1': 'error', '3': 2, '4': 1, '5': 9, '10': 'error'},
  ],
};

/// Descriptor for `SfReply`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sfReplyDescriptor = $convert.base64Decode(
    'CgdTZlJlcGx5EjUKCWVudmVsb3BlcxgBIAMoCzIXLnlnZ2dyYW0uU2VhbGVkRW52ZWxvcGVSCW'
    'VudmVsb3BlcxIUCgVlcnJvchgCIAEoCVIFZXJyb3I=');

@$core.Deprecated('Use preKeyBundleDescriptor instead')
const PreKeyBundle$json = {
  '1': 'PreKeyBundle',
  '2': [
    {
      '1': 'identity_ed25519_pubkey',
      '3': 1,
      '4': 1,
      '5': 12,
      '10': 'identityEd25519Pubkey'
    },
    {
      '1': 'identity_x25519_pubkey',
      '3': 2,
      '4': 1,
      '5': 12,
      '10': 'identityX25519Pubkey'
    },
    {'1': 'signed_prekey', '3': 3, '4': 1, '5': 12, '10': 'signedPrekey'},
    {
      '1': 'signed_prekey_signature',
      '3': 4,
      '4': 1,
      '5': 12,
      '10': 'signedPrekeySignature'
    },
    {'1': 'one_time_prekeys', '3': 5, '4': 3, '5': 12, '10': 'oneTimePrekeys'},
    {'1': 'prekey_timestamp', '3': 6, '4': 1, '5': 4, '10': 'prekeyTimestamp'},
    {'1': 'key_id', '3': 7, '4': 1, '5': 13, '10': 'keyId'},
    {
      '1': 'node_ed25519_pubkey',
      '3': 8,
      '4': 1,
      '5': 12,
      '10': 'nodeEd25519Pubkey'
    },
    {'1': 'node_key_proof', '3': 9, '4': 1, '5': 12, '10': 'nodeKeyProof'},
  ],
};

/// Descriptor for `PreKeyBundle`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List preKeyBundleDescriptor = $convert.base64Decode(
    'CgxQcmVLZXlCdW5kbGUSNgoXaWRlbnRpdHlfZWQyNTUxOV9wdWJrZXkYASABKAxSFWlkZW50aX'
    'R5RWQyNTUxOVB1YmtleRI0ChZpZGVudGl0eV94MjU1MTlfcHVia2V5GAIgASgMUhRpZGVudGl0'
    'eVgyNTUxOVB1YmtleRIjCg1zaWduZWRfcHJla2V5GAMgASgMUgxzaWduZWRQcmVrZXkSNgoXc2'
    'lnbmVkX3ByZWtleV9zaWduYXR1cmUYBCABKAxSFXNpZ25lZFByZWtleVNpZ25hdHVyZRIoChBv'
    'bmVfdGltZV9wcmVrZXlzGAUgAygMUg5vbmVUaW1lUHJla2V5cxIpChBwcmVrZXlfdGltZXN0YW'
    '1wGAYgASgEUg9wcmVrZXlUaW1lc3RhbXASFQoGa2V5X2lkGAcgASgNUgVrZXlJZBIuChNub2Rl'
    'X2VkMjU1MTlfcHVia2V5GAggASgMUhFub2RlRWQyNTUxOVB1YmtleRIkCg5ub2RlX2tleV9wcm'
    '9vZhgJIAEoDFIMbm9kZUtleVByb29m');

@$core.Deprecated('Use bundleReqDescriptor instead')
const BundleReq$json = {
  '1': 'BundleReq',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 4, '10': 'requestId'},
  ],
};

/// Descriptor for `BundleReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List bundleReqDescriptor = $convert
    .base64Decode('CglCdW5kbGVSZXESHQoKcmVxdWVzdF9pZBgBIAEoBFIJcmVxdWVzdElk');

@$core.Deprecated('Use bundleRespDescriptor instead')
const BundleResp$json = {
  '1': 'BundleResp',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 4, '10': 'requestId'},
    {
      '1': 'bundle',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.ygggram.PreKeyBundle',
      '10': 'bundle'
    },
    {'1': 'error', '3': 3, '4': 1, '5': 9, '10': 'error'},
  ],
};

/// Descriptor for `BundleResp`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List bundleRespDescriptor = $convert.base64Decode(
    'CgpCdW5kbGVSZXNwEh0KCnJlcXVlc3RfaWQYASABKARSCXJlcXVlc3RJZBItCgZidW5kbGUYAi'
    'ABKAsyFS55Z2dncmFtLlByZUtleUJ1bmRsZVIGYnVuZGxlEhQKBWVycm9yGAMgASgJUgVlcnJv'
    'cg==');

@$core.Deprecated('Use textContentDescriptor instead')
const TextContent$json = {
  '1': 'TextContent',
  '2': [
    {'1': 'body', '3': 1, '4': 1, '5': 9, '10': 'body'},
    {'1': 'mentions', '3': 2, '4': 3, '5': 9, '10': 'mentions'},
  ],
};

/// Descriptor for `TextContent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List textContentDescriptor = $convert.base64Decode(
    'CgtUZXh0Q29udGVudBISCgRib2R5GAEgASgJUgRib2R5EhoKCG1lbnRpb25zGAIgAygJUghtZW'
    '50aW9ucw==');

@$core.Deprecated('Use replyContentDescriptor instead')
const ReplyContent$json = {
  '1': 'ReplyContent',
  '2': [
    {
      '1': 'reply_to_message_id',
      '3': 1,
      '4': 1,
      '5': 9,
      '10': 'replyToMessageId'
    },
    {'1': 'body', '3': 2, '4': 1, '5': 9, '10': 'body'},
    {'1': 'quote_snippet', '3': 3, '4': 1, '5': 9, '10': 'quoteSnippet'},
  ],
};

/// Descriptor for `ReplyContent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List replyContentDescriptor = $convert.base64Decode(
    'CgxSZXBseUNvbnRlbnQSLQoTcmVwbHlfdG9fbWVzc2FnZV9pZBgBIAEoCVIQcmVwbHlUb01lc3'
    'NhZ2VJZBISCgRib2R5GAIgASgJUgRib2R5EiMKDXF1b3RlX3NuaXBwZXQYAyABKAlSDHF1b3Rl'
    'U25pcHBldA==');

@$core.Deprecated('Use reactionContentDescriptor instead')
const ReactionContent$json = {
  '1': 'ReactionContent',
  '2': [
    {'1': 'target_message_id', '3': 1, '4': 1, '5': 9, '10': 'targetMessageId'},
    {'1': 'emoji', '3': 2, '4': 1, '5': 9, '10': 'emoji'},
    {'1': 'is_removed', '3': 3, '4': 1, '5': 8, '10': 'isRemoved'},
  ],
};

/// Descriptor for `ReactionContent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List reactionContentDescriptor = $convert.base64Decode(
    'Cg9SZWFjdGlvbkNvbnRlbnQSKgoRdGFyZ2V0X21lc3NhZ2VfaWQYASABKAlSD3RhcmdldE1lc3'
    'NhZ2VJZBIUCgVlbW9qaRgCIAEoCVIFZW1vamkSHQoKaXNfcmVtb3ZlZBgDIAEoCFIJaXNSZW1v'
    'dmVk');

@$core.Deprecated('Use mediaManifestDescriptor instead')
const MediaManifest$json = {
  '1': 'MediaManifest',
  '2': [
    {'1': 'file_name', '3': 1, '4': 1, '5': 9, '10': 'fileName'},
    {'1': 'mime_type', '3': 2, '4': 1, '5': 9, '10': 'mimeType'},
    {'1': 'file_size', '3': 3, '4': 1, '5': 4, '10': 'fileSize'},
    {'1': 'sha256_hash', '3': 4, '4': 1, '5': 12, '10': 'sha256Hash'},
    {'1': 'xchacha_key', '3': 5, '4': 1, '5': 12, '10': 'xchachaKey'},
    {'1': 'xchacha_nonce', '3': 6, '4': 1, '5': 12, '10': 'xchachaNonce'},
    {'1': 'chunk_hashes', '3': 7, '4': 3, '5': 12, '10': 'chunkHashes'},
    {'1': 'thumbnail_bytes', '3': 8, '4': 1, '5': 12, '10': 'thumbnailBytes'},
    {'1': 'width', '3': 9, '4': 1, '5': 13, '10': 'width'},
    {'1': 'height', '3': 10, '4': 1, '5': 13, '10': 'height'},
    {
      '1': 'duration_seconds',
      '3': 11,
      '4': 1,
      '5': 13,
      '10': 'durationSeconds'
    },
  ],
};

/// Descriptor for `MediaManifest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List mediaManifestDescriptor = $convert.base64Decode(
    'Cg1NZWRpYU1hbmlmZXN0EhsKCWZpbGVfbmFtZRgBIAEoCVIIZmlsZU5hbWUSGwoJbWltZV90eX'
    'BlGAIgASgJUghtaW1lVHlwZRIbCglmaWxlX3NpemUYAyABKARSCGZpbGVTaXplEh8KC3NoYTI1'
    'Nl9oYXNoGAQgASgMUgpzaGEyNTZIYXNoEh8KC3hjaGFjaGFfa2V5GAUgASgMUgp4Y2hhY2hhS2'
    'V5EiMKDXhjaGFjaGFfbm9uY2UYBiABKAxSDHhjaGFjaGFOb25jZRIhCgxjaHVua19oYXNoZXMY'
    'ByADKAxSC2NodW5rSGFzaGVzEicKD3RodW1ibmFpbF9ieXRlcxgIIAEoDFIOdGh1bWJuYWlsQn'
    'l0ZXMSFAoFd2lkdGgYCSABKA1SBXdpZHRoEhYKBmhlaWdodBgKIAEoDVIGaGVpZ2h0EikKEGR1'
    'cmF0aW9uX3NlY29uZHMYCyABKA1SD2R1cmF0aW9uU2Vjb25kcw==');

@$core.Deprecated('Use callSignalContentDescriptor instead')
const CallSignalContent$json = {
  '1': 'CallSignalContent',
  '2': [
    {'1': 'call_id', '3': 1, '4': 1, '5': 9, '10': 'callId'},
    {
      '1': 'signal_type',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.ygggram.CallSignalType',
      '10': 'signalType'
    },
    {'1': 'sdp_payload', '3': 3, '4': 1, '5': 9, '10': 'sdpPayload'},
    {'1': 'ice_candidate', '3': 4, '4': 1, '5': 9, '10': 'iceCandidate'},
    {'1': 'is_video', '3': 5, '4': 1, '5': 8, '10': 'isVideo'},
  ],
};

/// Descriptor for `CallSignalContent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List callSignalContentDescriptor = $convert.base64Decode(
    'ChFDYWxsU2lnbmFsQ29udGVudBIXCgdjYWxsX2lkGAEgASgJUgZjYWxsSWQSOAoLc2lnbmFsX3'
    'R5cGUYAiABKA4yFy55Z2dncmFtLkNhbGxTaWduYWxUeXBlUgpzaWduYWxUeXBlEh8KC3NkcF9w'
    'YXlsb2FkGAMgASgJUgpzZHBQYXlsb2FkEiMKDWljZV9jYW5kaWRhdGUYBCABKAlSDGljZUNhbm'
    'RpZGF0ZRIZCghpc192aWRlbxgFIAEoCFIHaXNWaWRlbw==');

@$core.Deprecated('Use deliveryAckContentDescriptor instead')
const DeliveryAckContent$json = {
  '1': 'DeliveryAckContent',
  '2': [
    {
      '1': 'acknowledges_message_id',
      '3': 1,
      '4': 1,
      '5': 9,
      '10': 'acknowledgesMessageId'
    },
  ],
};

/// Descriptor for `DeliveryAckContent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deliveryAckContentDescriptor = $convert.base64Decode(
    'ChJEZWxpdmVyeUFja0NvbnRlbnQSNgoXYWNrbm93bGVkZ2VzX21lc3NhZ2VfaWQYASABKAlSFW'
    'Fja25vd2xlZGdlc01lc3NhZ2VJZA==');

@$core.Deprecated('Use chatMessageDescriptor instead')
const ChatMessage$json = {
  '1': 'ChatMessage',
  '2': [
    {'1': 'message_id', '3': 1, '4': 1, '5': 9, '10': 'messageId'},
    {'1': 'conversation_id', '3': 2, '4': 1, '5': 9, '10': 'conversationId'},
    {'1': 'sender_node_ipv6', '3': 3, '4': 1, '5': 9, '10': 'senderNodeIpv6'},
    {
      '1': 'sender_x25519_pubkey',
      '3': 4,
      '4': 1,
      '5': 12,
      '10': 'senderX25519Pubkey'
    },
    {'1': 'timestamp_ms', '3': 5, '4': 1, '5': 4, '10': 'timestampMs'},
    {'1': 'is_outbound', '3': 6, '4': 1, '5': 8, '10': 'isOutbound'},
    {'1': 'delivered', '3': 12, '4': 1, '5': 8, '10': 'delivered'},
    {
      '1': 'text',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.ygggram.TextContent',
      '9': 0,
      '10': 'text'
    },
    {
      '1': 'reply',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.ygggram.ReplyContent',
      '9': 0,
      '10': 'reply'
    },
    {
      '1': 'reaction',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.ygggram.ReactionContent',
      '9': 0,
      '10': 'reaction'
    },
    {
      '1': 'media',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.ygggram.MediaManifest',
      '9': 0,
      '10': 'media'
    },
    {
      '1': 'call_signal',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.ygggram.CallSignalContent',
      '9': 0,
      '10': 'callSignal'
    },
    {
      '1': 'delivery_ack',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.ygggram.DeliveryAckContent',
      '9': 0,
      '10': 'deliveryAck'
    },
  ],
  '8': [
    {'1': 'content'},
  ],
};

/// Descriptor for `ChatMessage`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List chatMessageDescriptor = $convert.base64Decode(
    'CgtDaGF0TWVzc2FnZRIdCgptZXNzYWdlX2lkGAEgASgJUgltZXNzYWdlSWQSJwoPY29udmVyc2'
    'F0aW9uX2lkGAIgASgJUg5jb252ZXJzYXRpb25JZBIoChBzZW5kZXJfbm9kZV9pcHY2GAMgASgJ'
    'Ug5zZW5kZXJOb2RlSXB2NhIwChRzZW5kZXJfeDI1NTE5X3B1YmtleRgEIAEoDFISc2VuZGVyWD'
    'I1NTE5UHVia2V5EiEKDHRpbWVzdGFtcF9tcxgFIAEoBFILdGltZXN0YW1wTXMSHwoLaXNfb3V0'
    'Ym91bmQYBiABKAhSCmlzT3V0Ym91bmQSHAoJZGVsaXZlcmVkGAwgASgIUglkZWxpdmVyZWQSKg'
    'oEdGV4dBgHIAEoCzIULnlnZ2dyYW0uVGV4dENvbnRlbnRIAFIEdGV4dBItCgVyZXBseRgIIAEo'
    'CzIVLnlnZ2dyYW0uUmVwbHlDb250ZW50SABSBXJlcGx5EjYKCHJlYWN0aW9uGAkgASgLMhgueW'
    'dnZ3JhbS5SZWFjdGlvbkNvbnRlbnRIAFIIcmVhY3Rpb24SLgoFbWVkaWEYCiABKAsyFi55Z2dn'
    'cmFtLk1lZGlhTWFuaWZlc3RIAFIFbWVkaWESPQoLY2FsbF9zaWduYWwYCyABKAsyGi55Z2dncm'
    'FtLkNhbGxTaWduYWxDb250ZW50SABSCmNhbGxTaWduYWwSQAoMZGVsaXZlcnlfYWNrGA0gASgL'
    'MhsueWdnZ3JhbS5EZWxpdmVyeUFja0NvbnRlbnRIAFILZGVsaXZlcnlBY2tCCQoHY29udGVudA'
    '==');

@$core.Deprecated('Use contactCardDescriptor instead')
const ContactCard$json = {
  '1': 'ContactCard',
  '2': [
    {'1': 'node_ipv6', '3': 1, '4': 1, '5': 9, '10': 'nodeIpv6'},
    {'1': 'ed25519_pubkey', '3': 2, '4': 1, '5': 12, '10': 'ed25519Pubkey'},
    {'1': 'x25519_pubkey', '3': 3, '4': 1, '5': 12, '10': 'x25519Pubkey'},
    {'1': 'nickname', '3': 4, '4': 1, '5': 9, '10': 'nickname'},
    {'1': 'bio', '3': 5, '4': 1, '5': 9, '10': 'bio'},
    {'1': 'avatar_hash', '3': 6, '4': 1, '5': 12, '10': 'avatarHash'},
    {'1': 'updated_at', '3': 7, '4': 1, '5': 4, '10': 'updatedAt'},
    {'1': 'is_verified', '3': 8, '4': 1, '5': 8, '10': 'isVerified'},
    {
      '1': 'bundle',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.ygggram.PreKeyBundle',
      '10': 'bundle'
    },
  ],
};

/// Descriptor for `ContactCard`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List contactCardDescriptor = $convert.base64Decode(
    'CgtDb250YWN0Q2FyZBIbCglub2RlX2lwdjYYASABKAlSCG5vZGVJcHY2EiUKDmVkMjU1MTlfcH'
    'Via2V5GAIgASgMUg1lZDI1NTE5UHVia2V5EiMKDXgyNTUxOV9wdWJrZXkYAyABKAxSDHgyNTUx'
    'OVB1YmtleRIaCghuaWNrbmFtZRgEIAEoCVIIbmlja25hbWUSEAoDYmlvGAUgASgJUgNiaW8SHw'
    'oLYXZhdGFyX2hhc2gYBiABKAxSCmF2YXRhckhhc2gSHQoKdXBkYXRlZF9hdBgHIAEoBFIJdXBk'
    'YXRlZEF0Eh8KC2lzX3ZlcmlmaWVkGAggASgIUgppc1ZlcmlmaWVkEi0KBmJ1bmRsZRgJIAEoCz'
    'IVLnlnZ2dyYW0uUHJlS2V5QnVuZGxlUgZidW5kbGU=');

@$core.Deprecated('Use getPreKeyBundleReqDescriptor instead')
const GetPreKeyBundleReq$json = {
  '1': 'GetPreKeyBundleReq',
};

/// Descriptor for `GetPreKeyBundleReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getPreKeyBundleReqDescriptor =
    $convert.base64Decode('ChJHZXRQcmVLZXlCdW5kbGVSZXE=');

@$core.Deprecated('Use backupChunkDescriptor instead')
const BackupChunk$json = {
  '1': 'BackupChunk',
  '2': [
    {'1': 'chunk_id', '3': 1, '4': 1, '5': 9, '10': 'chunkId'},
    {'1': 'offset', '3': 2, '4': 1, '5': 4, '10': 'offset'},
    {'1': 'size', '3': 3, '4': 1, '5': 13, '10': 'size'},
    {'1': 'sha256_hash', '3': 4, '4': 1, '5': 12, '10': 'sha256Hash'},
    {'1': 'storage_provider', '3': 5, '4': 1, '5': 9, '10': 'storageProvider'},
  ],
};

/// Descriptor for `BackupChunk`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List backupChunkDescriptor = $convert.base64Decode(
    'CgtCYWNrdXBDaHVuaxIZCghjaHVua19pZBgBIAEoCVIHY2h1bmtJZBIWCgZvZmZzZXQYAiABKA'
    'RSBm9mZnNldBISCgRzaXplGAMgASgNUgRzaXplEh8KC3NoYTI1Nl9oYXNoGAQgASgMUgpzaGEy'
    'NTZIYXNoEikKEHN0b3JhZ2VfcHJvdmlkZXIYBSABKAlSD3N0b3JhZ2VQcm92aWRlcg==');

@$core.Deprecated('Use backupManifestDescriptor instead')
const BackupManifest$json = {
  '1': 'BackupManifest',
  '2': [
    {'1': 'backup_id', '3': 1, '4': 1, '5': 4, '10': 'backupId'},
    {'1': 'created_at', '3': 2, '4': 1, '5': 4, '10': 'createdAt'},
    {'1': 'total_bytes', '3': 3, '4': 1, '5': 4, '10': 'totalBytes'},
    {'1': 'chunk_count', '3': 4, '4': 1, '5': 13, '10': 'chunkCount'},
    {'1': 'root_merkle_hash', '3': 5, '4': 1, '5': 12, '10': 'rootMerkleHash'},
    {'1': 'argon2id_salt', '3': 6, '4': 1, '5': 12, '10': 'argon2idSalt'},
    {'1': 'compression_algo', '3': 7, '4': 1, '5': 9, '10': 'compressionAlgo'},
    {'1': 'encryption_algo', '3': 8, '4': 1, '5': 9, '10': 'encryptionAlgo'},
    {
      '1': 'chunks',
      '3': 9,
      '4': 3,
      '5': 11,
      '6': '.ygggram.BackupChunk',
      '10': 'chunks'
    },
  ],
};

/// Descriptor for `BackupManifest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List backupManifestDescriptor = $convert.base64Decode(
    'Cg5CYWNrdXBNYW5pZmVzdBIbCgliYWNrdXBfaWQYASABKARSCGJhY2t1cElkEh0KCmNyZWF0ZW'
    'RfYXQYAiABKARSCWNyZWF0ZWRBdBIfCgt0b3RhbF9ieXRlcxgDIAEoBFIKdG90YWxCeXRlcxIf'
    'CgtjaHVua19jb3VudBgEIAEoDVIKY2h1bmtDb3VudBIoChByb290X21lcmtsZV9oYXNoGAUgAS'
    'gMUg5yb290TWVya2xlSGFzaBIjCg1hcmdvbjJpZF9zYWx0GAYgASgMUgxhcmdvbjJpZFNhbHQS'
    'KQoQY29tcHJlc3Npb25fYWxnbxgHIAEoCVIPY29tcHJlc3Npb25BbGdvEicKD2VuY3J5cHRpb2'
    '5fYWxnbxgIIAEoCVIOZW5jcnlwdGlvbkFsZ28SLAoGY2h1bmtzGAkgAygLMhQueWdnZ3JhbS5C'
    'YWNrdXBDaHVua1IGY2h1bmtz');

@$core.Deprecated('Use startNodeReqDescriptor instead')
const StartNodeReq$json = {
  '1': 'StartNodeReq',
  '2': [
    {
      '1': 'config',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.ygggram.NodeConfig',
      '10': 'config'
    },
  ],
};

/// Descriptor for `StartNodeReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List startNodeReqDescriptor = $convert.base64Decode(
    'CgxTdGFydE5vZGVSZXESKwoGY29uZmlnGAEgASgLMhMueWdnZ3JhbS5Ob2RlQ29uZmlnUgZjb2'
    '5maWc=');

@$core.Deprecated('Use stopNodeReqDescriptor instead')
const StopNodeReq$json = {
  '1': 'StopNodeReq',
  '2': [
    {'1': 'force', '3': 1, '4': 1, '5': 8, '10': 'force'},
  ],
};

/// Descriptor for `StopNodeReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List stopNodeReqDescriptor =
    $convert.base64Decode('CgtTdG9wTm9kZVJlcRIUCgVmb3JjZRgBIAEoCFIFZm9yY2U=');

@$core.Deprecated('Use sendMessageReqDescriptor instead')
const SendMessageReq$json = {
  '1': 'SendMessageReq',
  '2': [
    {'1': 'destination_ipv6', '3': 1, '4': 1, '5': 9, '10': 'destinationIpv6'},
    {
      '1': 'destination_x25519_pubkey',
      '3': 2,
      '4': 1,
      '5': 12,
      '10': 'destinationX25519Pubkey'
    },
    {
      '1': 'message',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.ygggram.ChatMessage',
      '10': 'message'
    },
  ],
};

/// Descriptor for `SendMessageReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sendMessageReqDescriptor = $convert.base64Decode(
    'Cg5TZW5kTWVzc2FnZVJlcRIpChBkZXN0aW5hdGlvbl9pcHY2GAEgASgJUg9kZXN0aW5hdGlvbk'
    'lwdjYSOgoZZGVzdGluYXRpb25feDI1NTE5X3B1YmtleRgCIAEoDFIXZGVzdGluYXRpb25YMjU1'
    'MTlQdWJrZXkSLgoHbWVzc2FnZRgDIAEoCzIULnlnZ2dyYW0uQ2hhdE1lc3NhZ2VSB21lc3NhZ2'
    'U=');

@$core.Deprecated('Use connectPeerReqDescriptor instead')
const ConnectPeerReq$json = {
  '1': 'ConnectPeerReq',
  '2': [
    {'1': 'peer_uri', '3': 1, '4': 1, '5': 9, '10': 'peerUri'},
    {
      '1': 'transport',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.ygggram.PeerTransportType',
      '10': 'transport'
    },
    {'1': 'custom_sni', '3': 3, '4': 1, '5': 9, '10': 'customSni'},
  ],
};

/// Descriptor for `ConnectPeerReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectPeerReqDescriptor = $convert.base64Decode(
    'Cg5Db25uZWN0UGVlclJlcRIZCghwZWVyX3VyaRgBIAEoCVIHcGVlclVyaRI4Cgl0cmFuc3Bvcn'
    'QYAiABKA4yGi55Z2dncmFtLlBlZXJUcmFuc3BvcnRUeXBlUgl0cmFuc3BvcnQSHQoKY3VzdG9t'
    'X3NuaRgDIAEoCVIJY3VzdG9tU25p');

@$core.Deprecated('Use disconnectPeerReqDescriptor instead')
const DisconnectPeerReq$json = {
  '1': 'DisconnectPeerReq',
  '2': [
    {'1': 'peer_uri', '3': 1, '4': 1, '5': 9, '10': 'peerUri'},
  ],
};

/// Descriptor for `DisconnectPeerReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List disconnectPeerReqDescriptor = $convert.base64Decode(
    'ChFEaXNjb25uZWN0UGVlclJlcRIZCghwZWVyX3VyaRgBIAEoCVIHcGVlclVyaQ==');

@$core.Deprecated('Use getPeersReqDescriptor instead')
const GetPeersReq$json = {
  '1': 'GetPeersReq',
};

/// Descriptor for `GetPeersReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getPeersReqDescriptor =
    $convert.base64Decode('CgtHZXRQZWVyc1JlcQ==');

@$core.Deprecated('Use getNodeStatusReqDescriptor instead')
const GetNodeStatusReq$json = {
  '1': 'GetNodeStatusReq',
};

/// Descriptor for `GetNodeStatusReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getNodeStatusReqDescriptor =
    $convert.base64Decode('ChBHZXROb2RlU3RhdHVzUmVx');

@$core.Deprecated('Use createBackupReqDescriptor instead')
const CreateBackupReq$json = {
  '1': 'CreateBackupReq',
  '2': [
    {
      '1': 'destination_target',
      '3': 1,
      '4': 1,
      '5': 9,
      '10': 'destinationTarget'
    },
    {
      '1': 'encryption_passphrase',
      '3': 2,
      '4': 1,
      '5': 9,
      '10': 'encryptionPassphrase'
    },
  ],
};

/// Descriptor for `CreateBackupReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createBackupReqDescriptor = $convert.base64Decode(
    'Cg9DcmVhdGVCYWNrdXBSZXESLQoSZGVzdGluYXRpb25fdGFyZ2V0GAEgASgJUhFkZXN0aW5hdG'
    'lvblRhcmdldBIzChVlbmNyeXB0aW9uX3Bhc3NwaHJhc2UYAiABKAlSFGVuY3J5cHRpb25QYXNz'
    'cGhyYXNl');

@$core.Deprecated('Use restoreBackupReqDescriptor instead')
const RestoreBackupReq$json = {
  '1': 'RestoreBackupReq',
  '2': [
    {'1': 'source_target', '3': 1, '4': 1, '5': 9, '10': 'sourceTarget'},
    {
      '1': 'encryption_passphrase',
      '3': 2,
      '4': 1,
      '5': 9,
      '10': 'encryptionPassphrase'
    },
  ],
};

/// Descriptor for `RestoreBackupReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List restoreBackupReqDescriptor = $convert.base64Decode(
    'ChBSZXN0b3JlQmFja3VwUmVxEiMKDXNvdXJjZV90YXJnZXQYASABKAlSDHNvdXJjZVRhcmdldB'
    'IzChVlbmNyeXB0aW9uX3Bhc3NwaHJhc2UYAiABKAlSFGVuY3J5cHRpb25QYXNzcGhyYXNl');

@$core.Deprecated('Use addContactReqDescriptor instead')
const AddContactReq$json = {
  '1': 'AddContactReq',
  '2': [
    {
      '1': 'contact',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.ygggram.ContactCard',
      '10': 'contact'
    },
  ],
};

/// Descriptor for `AddContactReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List addContactReqDescriptor = $convert.base64Decode(
    'Cg1BZGRDb250YWN0UmVxEi4KB2NvbnRhY3QYASABKAsyFC55Z2dncmFtLkNvbnRhY3RDYXJkUg'
    'djb250YWN0');

@$core.Deprecated('Use generateIdentityReqDescriptor instead')
const GenerateIdentityReq$json = {
  '1': 'GenerateIdentityReq',
};

/// Descriptor for `GenerateIdentityReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List generateIdentityReqDescriptor =
    $convert.base64Decode('ChNHZW5lcmF0ZUlkZW50aXR5UmVx');

@$core.Deprecated('Use identityRespDescriptor instead')
const IdentityResp$json = {
  '1': 'IdentityResp',
  '2': [
    {'1': 'mnemonic', '3': 1, '4': 1, '5': 9, '10': 'mnemonic'},
  ],
};

/// Descriptor for `IdentityResp`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List identityRespDescriptor = $convert
    .base64Decode('CgxJZGVudGl0eVJlc3ASGgoIbW5lbW9uaWMYASABKAlSCG1uZW1vbmlj');

@$core.Deprecated('Use conversationInfoDescriptor instead')
const ConversationInfo$json = {
  '1': 'ConversationInfo',
  '2': [
    {'1': 'conversation_id', '3': 1, '4': 1, '5': 9, '10': 'conversationId'},
    {'1': 'last_message', '3': 2, '4': 1, '5': 9, '10': 'lastMessage'},
    {'1': 'last_timestamp_ms', '3': 3, '4': 1, '5': 3, '10': 'lastTimestampMs'},
    {'1': 'message_count', '3': 4, '4': 1, '5': 13, '10': 'messageCount'},
    {'1': 'unread_count', '3': 5, '4': 1, '5': 13, '10': 'unreadCount'},
  ],
};

/// Descriptor for `ConversationInfo`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List conversationInfoDescriptor = $convert.base64Decode(
    'ChBDb252ZXJzYXRpb25JbmZvEicKD2NvbnZlcnNhdGlvbl9pZBgBIAEoCVIOY29udmVyc2F0aW'
    '9uSWQSIQoMbGFzdF9tZXNzYWdlGAIgASgJUgtsYXN0TWVzc2FnZRIqChFsYXN0X3RpbWVzdGFt'
    'cF9tcxgDIAEoA1IPbGFzdFRpbWVzdGFtcE1zEiMKDW1lc3NhZ2VfY291bnQYBCABKA1SDG1lc3'
    'NhZ2VDb3VudBIhCgx1bnJlYWRfY291bnQYBSABKA1SC3VucmVhZENvdW50');

@$core.Deprecated('Use listConversationsReqDescriptor instead')
const ListConversationsReq$json = {
  '1': 'ListConversationsReq',
  '2': [
    {'1': 'limit', '3': 1, '4': 1, '5': 13, '10': 'limit'},
  ],
};

/// Descriptor for `ListConversationsReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listConversationsReqDescriptor =
    $convert.base64Decode(
        'ChRMaXN0Q29udmVyc2F0aW9uc1JlcRIUCgVsaW1pdBgBIAEoDVIFbGltaXQ=');

@$core.Deprecated('Use conversationListRespDescriptor instead')
const ConversationListResp$json = {
  '1': 'ConversationListResp',
  '2': [
    {
      '1': 'conversations',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.ygggram.ConversationInfo',
      '10': 'conversations'
    },
  ],
};

/// Descriptor for `ConversationListResp`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List conversationListRespDescriptor = $convert.base64Decode(
    'ChRDb252ZXJzYXRpb25MaXN0UmVzcBI/Cg1jb252ZXJzYXRpb25zGAEgAygLMhkueWdnZ3JhbS'
    '5Db252ZXJzYXRpb25JbmZvUg1jb252ZXJzYXRpb25z');

@$core.Deprecated('Use getMessagesReqDescriptor instead')
const GetMessagesReq$json = {
  '1': 'GetMessagesReq',
  '2': [
    {'1': 'conversation_id', '3': 1, '4': 1, '5': 9, '10': 'conversationId'},
    {'1': 'limit', '3': 2, '4': 1, '5': 13, '10': 'limit'},
  ],
};

/// Descriptor for `GetMessagesReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getMessagesReqDescriptor = $convert.base64Decode(
    'Cg5HZXRNZXNzYWdlc1JlcRInCg9jb252ZXJzYXRpb25faWQYASABKAlSDmNvbnZlcnNhdGlvbk'
    'lkEhQKBWxpbWl0GAIgASgNUgVsaW1pdA==');

@$core.Deprecated('Use messageListRespDescriptor instead')
const MessageListResp$json = {
  '1': 'MessageListResp',
  '2': [
    {
      '1': 'messages',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.ygggram.ChatMessage',
      '10': 'messages'
    },
  ],
};

/// Descriptor for `MessageListResp`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List messageListRespDescriptor = $convert.base64Decode(
    'Cg9NZXNzYWdlTGlzdFJlc3ASMAoIbWVzc2FnZXMYASADKAsyFC55Z2dncmFtLkNoYXRNZXNzYW'
    'dlUghtZXNzYWdlcw==');

@$core.Deprecated('Use listContactsReqDescriptor instead')
const ListContactsReq$json = {
  '1': 'ListContactsReq',
};

/// Descriptor for `ListContactsReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listContactsReqDescriptor =
    $convert.base64Decode('Cg9MaXN0Q29udGFjdHNSZXE=');

@$core.Deprecated('Use contactListRespDescriptor instead')
const ContactListResp$json = {
  '1': 'ContactListResp',
  '2': [
    {
      '1': 'contacts',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.ygggram.ContactCard',
      '10': 'contacts'
    },
  ],
};

/// Descriptor for `ContactListResp`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List contactListRespDescriptor = $convert.base64Decode(
    'Cg9Db250YWN0TGlzdFJlc3ASMAoIY29udGFjdHMYASADKAsyFC55Z2dncmFtLkNvbnRhY3RDYX'
    'JkUghjb250YWN0cw==');

@$core.Deprecated('Use fetchContactReqDescriptor instead')
const FetchContactReq$json = {
  '1': 'FetchContactReq',
  '2': [
    {
      '1': 'node_ed25519_pubkey',
      '3': 1,
      '4': 1,
      '5': 12,
      '10': 'nodeEd25519Pubkey'
    },
    {'1': 'nickname', '3': 2, '4': 1, '5': 9, '10': 'nickname'},
  ],
};

/// Descriptor for `FetchContactReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fetchContactReqDescriptor = $convert.base64Decode(
    'Cg9GZXRjaENvbnRhY3RSZXESLgoTbm9kZV9lZDI1NTE5X3B1YmtleRgBIAEoDFIRbm9kZUVkMj'
    'U1MTlQdWJrZXkSGgoIbmlja25hbWUYAiABKAlSCG5pY2tuYW1l');

@$core.Deprecated('Use fetchContactRespDescriptor instead')
const FetchContactResp$json = {
  '1': 'FetchContactResp',
  '2': [
    {'1': 'requested', '3': 1, '4': 1, '5': 8, '10': 'requested'},
    {'1': 'request_id', '3': 2, '4': 1, '5': 4, '10': 'requestId'},
    {'1': 'node_ipv6', '3': 3, '4': 1, '5': 9, '10': 'nodeIpv6'},
  ],
};

/// Descriptor for `FetchContactResp`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fetchContactRespDescriptor = $convert.base64Decode(
    'ChBGZXRjaENvbnRhY3RSZXNwEhwKCXJlcXVlc3RlZBgBIAEoCFIJcmVxdWVzdGVkEh0KCnJlcX'
    'Vlc3RfaWQYAiABKARSCXJlcXVlc3RJZBIbCglub2RlX2lwdjYYAyABKAlSCG5vZGVJcHY2');

@$core.Deprecated('Use contactResolvedEventDescriptor instead')
const ContactResolvedEvent$json = {
  '1': 'ContactResolvedEvent',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 4, '10': 'requestId'},
    {
      '1': 'contact',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.ygggram.ContactCard',
      '10': 'contact'
    },
    {'1': 'error', '3': 3, '4': 1, '5': 9, '10': 'error'},
  ],
};

/// Descriptor for `ContactResolvedEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List contactResolvedEventDescriptor = $convert.base64Decode(
    'ChRDb250YWN0UmVzb2x2ZWRFdmVudBIdCgpyZXF1ZXN0X2lkGAEgASgEUglyZXF1ZXN0SWQSLg'
    'oHY29udGFjdBgCIAEoCzIULnlnZ2dyYW0uQ29udGFjdENhcmRSB2NvbnRhY3QSFAoFZXJyb3IY'
    'AyABKAlSBWVycm9y');

@$core.Deprecated('Use sfCollectReqDescriptor instead')
const SfCollectReq$json = {
  '1': 'SfCollectReq',
};

/// Descriptor for `SfCollectReq`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sfCollectReqDescriptor =
    $convert.base64Decode('CgxTZkNvbGxlY3RSZXE=');

@$core.Deprecated('Use sfCollectRespDescriptor instead')
const SfCollectResp$json = {
  '1': 'SfCollectResp',
  '2': [
    {'1': 'requested', '3': 1, '4': 1, '5': 13, '10': 'requested'},
    {'1': 'relays', '3': 2, '4': 3, '5': 9, '10': 'relays'},
  ],
};

/// Descriptor for `SfCollectResp`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sfCollectRespDescriptor = $convert.base64Decode(
    'Cg1TZkNvbGxlY3RSZXNwEhwKCXJlcXVlc3RlZBgBIAEoDVIJcmVxdWVzdGVkEhYKBnJlbGF5cx'
    'gCIAMoCVIGcmVsYXlz');

@$core.Deprecated('Use sfStatsDescriptor instead')
const SfStats$json = {
  '1': 'SfStats',
  '2': [
    {'1': 'relay_enabled', '3': 1, '4': 1, '5': 8, '10': 'relayEnabled'},
    {
      '1': 'relays_configured',
      '3': 2,
      '4': 1,
      '5': 13,
      '10': 'relaysConfigured'
    },
    {'1': 'pushed', '3': 3, '4': 1, '5': 4, '10': 'pushed'},
    {'1': 'push_failed', '3': 4, '4': 1, '5': 4, '10': 'pushFailed'},
    {'1': 'stored', '3': 5, '4': 1, '5': 4, '10': 'stored'},
    {'1': 'refused', '3': 6, '4': 1, '5': 4, '10': 'refused'},
    {'1': 'pulled', '3': 7, '4': 1, '5': 4, '10': 'pulled'},
    {'1': 'handed_out', '3': 8, '4': 1, '5': 4, '10': 'handedOut'},
    {'1': 'last_refusal', '3': 9, '4': 1, '5': 9, '10': 'lastRefusal'},
    {'1': 'expired', '3': 10, '4': 1, '5': 4, '10': 'expired'},
  ],
};

/// Descriptor for `SfStats`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sfStatsDescriptor = $convert.base64Decode(
    'CgdTZlN0YXRzEiMKDXJlbGF5X2VuYWJsZWQYASABKAhSDHJlbGF5RW5hYmxlZBIrChFyZWxheX'
    'NfY29uZmlndXJlZBgCIAEoDVIQcmVsYXlzQ29uZmlndXJlZBIWCgZwdXNoZWQYAyABKARSBnB1'
    'c2hlZBIfCgtwdXNoX2ZhaWxlZBgEIAEoBFIKcHVzaEZhaWxlZBIWCgZzdG9yZWQYBSABKARSBn'
    'N0b3JlZBIYCgdyZWZ1c2VkGAYgASgEUgdyZWZ1c2VkEhYKBnB1bGxlZBgHIAEoBFIGcHVsbGVk'
    'Eh0KCmhhbmRlZF9vdXQYCCABKARSCWhhbmRlZE91dBIhCgxsYXN0X3JlZnVzYWwYCSABKAlSC2'
    'xhc3RSZWZ1c2FsEhgKB2V4cGlyZWQYCiABKARSB2V4cGlyZWQ=');

@$core.Deprecated('Use coreRequestDescriptor instead')
const CoreRequest$json = {
  '1': 'CoreRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 4, '10': 'requestId'},
    {
      '1': 'start_node',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.ygggram.StartNodeReq',
      '9': 0,
      '10': 'startNode'
    },
    {
      '1': 'stop_node',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.ygggram.StopNodeReq',
      '9': 0,
      '10': 'stopNode'
    },
    {
      '1': 'send_message',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.ygggram.SendMessageReq',
      '9': 0,
      '10': 'sendMessage'
    },
    {
      '1': 'connect_peer',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.ygggram.ConnectPeerReq',
      '9': 0,
      '10': 'connectPeer'
    },
    {
      '1': 'disconnect_peer',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.ygggram.DisconnectPeerReq',
      '9': 0,
      '10': 'disconnectPeer'
    },
    {
      '1': 'get_peers',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.ygggram.GetPeersReq',
      '9': 0,
      '10': 'getPeers'
    },
    {
      '1': 'get_node_status',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.ygggram.GetNodeStatusReq',
      '9': 0,
      '10': 'getNodeStatus'
    },
    {
      '1': 'create_backup',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.ygggram.CreateBackupReq',
      '9': 0,
      '10': 'createBackup'
    },
    {
      '1': 'restore_backup',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.ygggram.RestoreBackupReq',
      '9': 0,
      '10': 'restoreBackup'
    },
    {
      '1': 'add_contact',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.ygggram.AddContactReq',
      '9': 0,
      '10': 'addContact'
    },
    {
      '1': 'get_prekey_bundle',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.ygggram.GetPreKeyBundleReq',
      '9': 0,
      '10': 'getPrekeyBundle'
    },
    {
      '1': 'generate_identity',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.ygggram.GenerateIdentityReq',
      '9': 0,
      '10': 'generateIdentity'
    },
    {
      '1': 'list_conversations',
      '3': 14,
      '4': 1,
      '5': 11,
      '6': '.ygggram.ListConversationsReq',
      '9': 0,
      '10': 'listConversations'
    },
    {
      '1': 'get_messages',
      '3': 15,
      '4': 1,
      '5': 11,
      '6': '.ygggram.GetMessagesReq',
      '9': 0,
      '10': 'getMessages'
    },
    {
      '1': 'list_contacts',
      '3': 16,
      '4': 1,
      '5': 11,
      '6': '.ygggram.ListContactsReq',
      '9': 0,
      '10': 'listContacts'
    },
    {
      '1': 'sf_collect',
      '3': 17,
      '4': 1,
      '5': 11,
      '6': '.ygggram.SfCollectReq',
      '9': 0,
      '10': 'sfCollect'
    },
    {
      '1': 'fetch_contact',
      '3': 18,
      '4': 1,
      '5': 11,
      '6': '.ygggram.FetchContactReq',
      '9': 0,
      '10': 'fetchContact'
    },
  ],
  '8': [
    {'1': 'request'},
  ],
};

/// Descriptor for `CoreRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List coreRequestDescriptor = $convert.base64Decode(
    'CgtDb3JlUmVxdWVzdBIdCgpyZXF1ZXN0X2lkGAEgASgEUglyZXF1ZXN0SWQSNgoKc3RhcnRfbm'
    '9kZRgCIAEoCzIVLnlnZ2dyYW0uU3RhcnROb2RlUmVxSABSCXN0YXJ0Tm9kZRIzCglzdG9wX25v'
    'ZGUYAyABKAsyFC55Z2dncmFtLlN0b3BOb2RlUmVxSABSCHN0b3BOb2RlEjwKDHNlbmRfbWVzc2'
    'FnZRgEIAEoCzIXLnlnZ2dyYW0uU2VuZE1lc3NhZ2VSZXFIAFILc2VuZE1lc3NhZ2USPAoMY29u'
    'bmVjdF9wZWVyGAUgASgLMhcueWdnZ3JhbS5Db25uZWN0UGVlclJlcUgAUgtjb25uZWN0UGVlch'
    'JFCg9kaXNjb25uZWN0X3BlZXIYBiABKAsyGi55Z2dncmFtLkRpc2Nvbm5lY3RQZWVyUmVxSABS'
    'DmRpc2Nvbm5lY3RQZWVyEjMKCWdldF9wZWVycxgHIAEoCzIULnlnZ2dyYW0uR2V0UGVlcnNSZX'
    'FIAFIIZ2V0UGVlcnMSQwoPZ2V0X25vZGVfc3RhdHVzGAggASgLMhkueWdnZ3JhbS5HZXROb2Rl'
    'U3RhdHVzUmVxSABSDWdldE5vZGVTdGF0dXMSPwoNY3JlYXRlX2JhY2t1cBgJIAEoCzIYLnlnZ2'
    'dyYW0uQ3JlYXRlQmFja3VwUmVxSABSDGNyZWF0ZUJhY2t1cBJCCg5yZXN0b3JlX2JhY2t1cBgK'
    'IAEoCzIZLnlnZ2dyYW0uUmVzdG9yZUJhY2t1cFJlcUgAUg1yZXN0b3JlQmFja3VwEjkKC2FkZF'
    '9jb250YWN0GAsgASgLMhYueWdnZ3JhbS5BZGRDb250YWN0UmVxSABSCmFkZENvbnRhY3QSSQoR'
    'Z2V0X3ByZWtleV9idW5kbGUYDCABKAsyGy55Z2dncmFtLkdldFByZUtleUJ1bmRsZVJlcUgAUg'
    '9nZXRQcmVrZXlCdW5kbGUSSwoRZ2VuZXJhdGVfaWRlbnRpdHkYDSABKAsyHC55Z2dncmFtLkdl'
    'bmVyYXRlSWRlbnRpdHlSZXFIAFIQZ2VuZXJhdGVJZGVudGl0eRJOChJsaXN0X2NvbnZlcnNhdG'
    'lvbnMYDiABKAsyHS55Z2dncmFtLkxpc3RDb252ZXJzYXRpb25zUmVxSABSEWxpc3RDb252ZXJz'
    'YXRpb25zEjwKDGdldF9tZXNzYWdlcxgPIAEoCzIXLnlnZ2dyYW0uR2V0TWVzc2FnZXNSZXFIAF'
    'ILZ2V0TWVzc2FnZXMSPwoNbGlzdF9jb250YWN0cxgQIAEoCzIYLnlnZ2dyYW0uTGlzdENvbnRh'
    'Y3RzUmVxSABSDGxpc3RDb250YWN0cxI2CgpzZl9jb2xsZWN0GBEgASgLMhUueWdnZ3JhbS5TZk'
    'NvbGxlY3RSZXFIAFIJc2ZDb2xsZWN0Ej8KDWZldGNoX2NvbnRhY3QYEiABKAsyGC55Z2dncmFt'
    'LkZldGNoQ29udGFjdFJlcUgAUgxmZXRjaENvbnRhY3RCCQoHcmVxdWVzdA==');

@$core.Deprecated('Use peerInfoDescriptor instead')
const PeerInfo$json = {
  '1': 'PeerInfo',
  '2': [
    {'1': 'uri', '3': 1, '4': 1, '5': 9, '10': 'uri'},
    {'1': 'remote_address', '3': 2, '4': 1, '5': 9, '10': 'remoteAddress'},
    {'1': 'yggdrasil_ipv6', '3': 3, '4': 1, '5': 9, '10': 'yggdrasilIpv6'},
    {
      '1': 'transport',
      '3': 4,
      '4': 1,
      '5': 14,
      '6': '.ygggram.PeerTransportType',
      '10': 'transport'
    },
    {'1': 'is_inbound', '3': 5, '4': 1, '5': 8, '10': 'isInbound'},
    {'1': 'bytes_sent', '3': 6, '4': 1, '5': 4, '10': 'bytesSent'},
    {'1': 'bytes_received', '3': 7, '4': 1, '5': 4, '10': 'bytesReceived'},
    {'1': 'latency_ms', '3': 8, '4': 1, '5': 4, '10': 'latencyMs'},
    {'1': 'is_connected', '3': 9, '4': 1, '5': 8, '10': 'isConnected'},
    {'1': 'sni_used', '3': 10, '4': 1, '5': 9, '10': 'sniUsed'},
  ],
};

/// Descriptor for `PeerInfo`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List peerInfoDescriptor = $convert.base64Decode(
    'CghQZWVySW5mbxIQCgN1cmkYASABKAlSA3VyaRIlCg5yZW1vdGVfYWRkcmVzcxgCIAEoCVINcm'
    'Vtb3RlQWRkcmVzcxIlCg55Z2dkcmFzaWxfaXB2NhgDIAEoCVINeWdnZHJhc2lsSXB2NhI4Cgl0'
    'cmFuc3BvcnQYBCABKA4yGi55Z2dncmFtLlBlZXJUcmFuc3BvcnRUeXBlUgl0cmFuc3BvcnQSHQ'
    'oKaXNfaW5ib3VuZBgFIAEoCFIJaXNJbmJvdW5kEh0KCmJ5dGVzX3NlbnQYBiABKARSCWJ5dGVz'
    'U2VudBIlCg5ieXRlc19yZWNlaXZlZBgHIAEoBFINYnl0ZXNSZWNlaXZlZBIdCgpsYXRlbmN5X2'
    '1zGAggASgEUglsYXRlbmN5TXMSIQoMaXNfY29ubmVjdGVkGAkgASgIUgtpc0Nvbm5lY3RlZBIZ'
    'CghzbmlfdXNlZBgKIAEoCVIHc25pVXNlZA==');

@$core.Deprecated('Use nodeStatusRespDescriptor instead')
const NodeStatusResp$json = {
  '1': 'NodeStatusResp',
  '2': [
    {'1': 'is_running', '3': 1, '4': 1, '5': 8, '10': 'isRunning'},
    {'1': 'my_ipv6', '3': 2, '4': 1, '5': 9, '10': 'myIpv6'},
    {
      '1': 'my_ed25519_pubkey',
      '3': 3,
      '4': 1,
      '5': 12,
      '10': 'myEd25519Pubkey'
    },
    {'1': 'my_x25519_pubkey', '3': 4, '4': 1, '5': 12, '10': 'myX25519Pubkey'},
    {
      '1': 'connected_peers_count',
      '3': 5,
      '4': 1,
      '5': 13,
      '10': 'connectedPeersCount'
    },
    {'1': 'uptime_seconds', '3': 6, '4': 1, '5': 4, '10': 'uptimeSeconds'},
    {
      '1': 'active_anti_dpi',
      '3': 7,
      '4': 1,
      '5': 14,
      '6': '.ygggram.AntiDpiMode',
      '10': 'activeAntiDpi'
    },
    {
      '1': 'routing_table_nodes',
      '3': 8,
      '4': 1,
      '5': 4,
      '10': 'routingTableNodes'
    },
    {'1': 'sf', '3': 9, '4': 1, '5': 11, '6': '.ygggram.SfStats', '10': 'sf'},
  ],
};

/// Descriptor for `NodeStatusResp`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List nodeStatusRespDescriptor = $convert.base64Decode(
    'Cg5Ob2RlU3RhdHVzUmVzcBIdCgppc19ydW5uaW5nGAEgASgIUglpc1J1bm5pbmcSFwoHbXlfaX'
    'B2NhgCIAEoCVIGbXlJcHY2EioKEW15X2VkMjU1MTlfcHVia2V5GAMgASgMUg9teUVkMjU1MTlQ'
    'dWJrZXkSKAoQbXlfeDI1NTE5X3B1YmtleRgEIAEoDFIObXlYMjU1MTlQdWJrZXkSMgoVY29ubm'
    'VjdGVkX3BlZXJzX2NvdW50GAUgASgNUhNjb25uZWN0ZWRQZWVyc0NvdW50EiUKDnVwdGltZV9z'
    'ZWNvbmRzGAYgASgEUg11cHRpbWVTZWNvbmRzEjwKD2FjdGl2ZV9hbnRpX2RwaRgHIAEoDjIULn'
    'lnZ2dyYW0uQW50aURwaU1vZGVSDWFjdGl2ZUFudGlEcGkSLgoTcm91dGluZ190YWJsZV9ub2Rl'
    'cxgIIAEoBFIRcm91dGluZ1RhYmxlTm9kZXMSIAoCc2YYCSABKAsyEC55Z2dncmFtLlNmU3RhdH'
    'NSAnNm');

@$core.Deprecated('Use peerListRespDescriptor instead')
const PeerListResp$json = {
  '1': 'PeerListResp',
  '2': [
    {
      '1': 'peers',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.ygggram.PeerInfo',
      '10': 'peers'
    },
  ],
};

/// Descriptor for `PeerListResp`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List peerListRespDescriptor = $convert.base64Decode(
    'CgxQZWVyTGlzdFJlc3ASJwoFcGVlcnMYASADKAsyES55Z2dncmFtLlBlZXJJbmZvUgVwZWVycw'
    '==');

@$core.Deprecated('Use sendMessageRespDescriptor instead')
const SendMessageResp$json = {
  '1': 'SendMessageResp',
  '2': [
    {'1': 'message_id', '3': 1, '4': 1, '5': 9, '10': 'messageId'},
    {
      '1': 'written_to_overlay',
      '3': 2,
      '4': 1,
      '5': 8,
      '10': 'writtenToOverlay'
    },
    {'1': 'queued_in_outbox', '3': 3, '4': 1, '5': 8, '10': 'queuedInOutbox'},
    {'1': 'sealed_envelope', '3': 4, '4': 1, '5': 12, '10': 'sealedEnvelope'},
    {'1': 'relays_pushed', '3': 5, '4': 1, '5': 8, '10': 'relaysPushed'},
    {'1': 'relays_handed', '3': 6, '4': 1, '5': 13, '10': 'relaysHanded'},
  ],
};

/// Descriptor for `SendMessageResp`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sendMessageRespDescriptor = $convert.base64Decode(
    'Cg9TZW5kTWVzc2FnZVJlc3ASHQoKbWVzc2FnZV9pZBgBIAEoCVIJbWVzc2FnZUlkEiwKEndyaX'
    'R0ZW5fdG9fb3ZlcmxheRgCIAEoCFIQd3JpdHRlblRvT3ZlcmxheRIoChBxdWV1ZWRfaW5fb3V0'
    'Ym94GAMgASgIUg5xdWV1ZWRJbk91dGJveBInCg9zZWFsZWRfZW52ZWxvcGUYBCABKAxSDnNlYW'
    'xlZEVudmVsb3BlEiMKDXJlbGF5c19wdXNoZWQYBSABKAhSDHJlbGF5c1B1c2hlZBIjCg1yZWxh'
    'eXNfaGFuZGVkGAYgASgNUgxyZWxheXNIYW5kZWQ=');

@$core.Deprecated('Use backupRespDescriptor instead')
const BackupResp$json = {
  '1': 'BackupResp',
  '2': [
    {
      '1': 'manifest',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.ygggram.BackupManifest',
      '10': 'manifest'
    },
    {'1': 'status_message', '3': 2, '4': 1, '5': 9, '10': 'statusMessage'},
  ],
};

/// Descriptor for `BackupResp`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List backupRespDescriptor = $convert.base64Decode(
    'CgpCYWNrdXBSZXNwEjMKCG1hbmlmZXN0GAEgASgLMhcueWdnZ3JhbS5CYWNrdXBNYW5pZmVzdF'
    'IIbWFuaWZlc3QSJQoOc3RhdHVzX21lc3NhZ2UYAiABKAlSDXN0YXR1c01lc3NhZ2U=');

@$core.Deprecated('Use coreResponseDescriptor instead')
const CoreResponse$json = {
  '1': 'CoreResponse',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 4, '10': 'requestId'},
    {'1': 'success', '3': 2, '4': 1, '5': 8, '10': 'success'},
    {'1': 'error_message', '3': 3, '4': 1, '5': 9, '10': 'errorMessage'},
    {
      '1': 'node_status',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.ygggram.NodeStatusResp',
      '9': 0,
      '10': 'nodeStatus'
    },
    {
      '1': 'peer_list',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.ygggram.PeerListResp',
      '9': 0,
      '10': 'peerList'
    },
    {
      '1': 'send_message',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.ygggram.SendMessageResp',
      '9': 0,
      '10': 'sendMessage'
    },
    {
      '1': 'backup',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.ygggram.BackupResp',
      '9': 0,
      '10': 'backup'
    },
    {
      '1': 'prekey_bundle',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.ygggram.PreKeyBundle',
      '9': 0,
      '10': 'prekeyBundle'
    },
    {
      '1': 'identity',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.ygggram.IdentityResp',
      '9': 0,
      '10': 'identity'
    },
    {
      '1': 'conversations',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.ygggram.ConversationListResp',
      '9': 0,
      '10': 'conversations'
    },
    {
      '1': 'messages',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.ygggram.MessageListResp',
      '9': 0,
      '10': 'messages'
    },
    {
      '1': 'contacts',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.ygggram.ContactListResp',
      '9': 0,
      '10': 'contacts'
    },
    {
      '1': 'sf_collect',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.ygggram.SfCollectResp',
      '9': 0,
      '10': 'sfCollect'
    },
    {
      '1': 'fetch_contact',
      '3': 14,
      '4': 1,
      '5': 11,
      '6': '.ygggram.FetchContactResp',
      '9': 0,
      '10': 'fetchContact'
    },
  ],
  '8': [
    {'1': 'response'},
  ],
};

/// Descriptor for `CoreResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List coreResponseDescriptor = $convert.base64Decode(
    'CgxDb3JlUmVzcG9uc2USHQoKcmVxdWVzdF9pZBgBIAEoBFIJcmVxdWVzdElkEhgKB3N1Y2Nlc3'
    'MYAiABKAhSB3N1Y2Nlc3MSIwoNZXJyb3JfbWVzc2FnZRgDIAEoCVIMZXJyb3JNZXNzYWdlEjoK'
    'C25vZGVfc3RhdHVzGAQgASgLMhcueWdnZ3JhbS5Ob2RlU3RhdHVzUmVzcEgAUgpub2RlU3RhdH'
    'VzEjQKCXBlZXJfbGlzdBgFIAEoCzIVLnlnZ2dyYW0uUGVlckxpc3RSZXNwSABSCHBlZXJMaXN0'
    'Ej0KDHNlbmRfbWVzc2FnZRgGIAEoCzIYLnlnZ2dyYW0uU2VuZE1lc3NhZ2VSZXNwSABSC3Nlbm'
    'RNZXNzYWdlEi0KBmJhY2t1cBgHIAEoCzITLnlnZ2dyYW0uQmFja3VwUmVzcEgAUgZiYWNrdXAS'
    'PAoNcHJla2V5X2J1bmRsZRgIIAEoCzIVLnlnZ2dyYW0uUHJlS2V5QnVuZGxlSABSDHByZWtleU'
    'J1bmRsZRIzCghpZGVudGl0eRgJIAEoCzIVLnlnZ2dyYW0uSWRlbnRpdHlSZXNwSABSCGlkZW50'
    'aXR5EkUKDWNvbnZlcnNhdGlvbnMYCiABKAsyHS55Z2dncmFtLkNvbnZlcnNhdGlvbkxpc3RSZX'
    'NwSABSDWNvbnZlcnNhdGlvbnMSNgoIbWVzc2FnZXMYCyABKAsyGC55Z2dncmFtLk1lc3NhZ2VM'
    'aXN0UmVzcEgAUghtZXNzYWdlcxI2Cghjb250YWN0cxgMIAEoCzIYLnlnZ2dyYW0uQ29udGFjdE'
    'xpc3RSZXNwSABSCGNvbnRhY3RzEjcKCnNmX2NvbGxlY3QYDSABKAsyFi55Z2dncmFtLlNmQ29s'
    'bGVjdFJlc3BIAFIJc2ZDb2xsZWN0EkAKDWZldGNoX2NvbnRhY3QYDiABKAsyGS55Z2dncmFtLk'
    'ZldGNoQ29udGFjdFJlc3BIAFIMZmV0Y2hDb250YWN0QgoKCHJlc3BvbnNl');

@$core.Deprecated('Use networkStatusEventDescriptor instead')
const NetworkStatusEvent$json = {
  '1': 'NetworkStatusEvent',
  '2': [
    {'1': 'online', '3': 1, '4': 1, '5': 8, '10': 'online'},
    {'1': 'current_ipv6', '3': 2, '4': 1, '5': 9, '10': 'currentIpv6'},
    {'1': 'active_peers', '3': 3, '4': 1, '5': 13, '10': 'activePeers'},
    {'1': 'total_rx_bytes', '3': 4, '4': 1, '5': 4, '10': 'totalRxBytes'},
    {'1': 'total_tx_bytes', '3': 5, '4': 1, '5': 4, '10': 'totalTxBytes'},
    {'1': 'status_note', '3': 6, '4': 1, '5': 9, '10': 'statusNote'},
  ],
};

/// Descriptor for `NetworkStatusEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List networkStatusEventDescriptor = $convert.base64Decode(
    'ChJOZXR3b3JrU3RhdHVzRXZlbnQSFgoGb25saW5lGAEgASgIUgZvbmxpbmUSIQoMY3VycmVudF'
    '9pcHY2GAIgASgJUgtjdXJyZW50SXB2NhIhCgxhY3RpdmVfcGVlcnMYAyABKA1SC2FjdGl2ZVBl'
    'ZXJzEiQKDnRvdGFsX3J4X2J5dGVzGAQgASgEUgx0b3RhbFJ4Qnl0ZXMSJAoOdG90YWxfdHhfYn'
    'l0ZXMYBSABKARSDHRvdGFsVHhCeXRlcxIfCgtzdGF0dXNfbm90ZRgGIAEoCVIKc3RhdHVzTm90'
    'ZQ==');

@$core.Deprecated('Use peerListUpdatedEventDescriptor instead')
const PeerListUpdatedEvent$json = {
  '1': 'PeerListUpdatedEvent',
  '2': [
    {
      '1': 'peers',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.ygggram.PeerInfo',
      '10': 'peers'
    },
  ],
};

/// Descriptor for `PeerListUpdatedEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List peerListUpdatedEventDescriptor = $convert.base64Decode(
    'ChRQZWVyTGlzdFVwZGF0ZWRFdmVudBInCgVwZWVycxgBIAMoCzIRLnlnZ2dyYW0uUGVlckluZm'
    '9SBXBlZXJz');

@$core.Deprecated('Use antiDpiStatusEventDescriptor instead')
const AntiDpiStatusEvent$json = {
  '1': 'AntiDpiStatusEvent',
  '2': [
    {
      '1': 'mode',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.ygggram.AntiDpiMode',
      '10': 'mode'
    },
    {'1': 'current_sni', '3': 2, '4': 1, '5': 9, '10': 'currentSni'},
    {
      '1': 'handshake_success',
      '3': 3,
      '4': 1,
      '5': 8,
      '10': 'handshakeSuccess'
    },
    {
      '1': 'transport_details',
      '3': 4,
      '4': 1,
      '5': 9,
      '10': 'transportDetails'
    },
    {
      '1': 'fallback_triggered',
      '3': 5,
      '4': 1,
      '5': 8,
      '10': 'fallbackTriggered'
    },
  ],
};

/// Descriptor for `AntiDpiStatusEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List antiDpiStatusEventDescriptor = $convert.base64Decode(
    'ChJBbnRpRHBpU3RhdHVzRXZlbnQSKAoEbW9kZRgBIAEoDjIULnlnZ2dyYW0uQW50aURwaU1vZG'
    'VSBG1vZGUSHwoLY3VycmVudF9zbmkYAiABKAlSCmN1cnJlbnRTbmkSKwoRaGFuZHNoYWtlX3N1'
    'Y2Nlc3MYAyABKAhSEGhhbmRzaGFrZVN1Y2Nlc3MSKwoRdHJhbnNwb3J0X2RldGFpbHMYBCABKA'
    'lSEHRyYW5zcG9ydERldGFpbHMSLQoSZmFsbGJhY2tfdHJpZ2dlcmVkGAUgASgIUhFmYWxsYmFj'
    'a1RyaWdnZXJlZA==');

@$core.Deprecated('Use incomingMessageEventDescriptor instead')
const IncomingMessageEvent$json = {
  '1': 'IncomingMessageEvent',
  '2': [
    {
      '1': 'message',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.ygggram.ChatMessage',
      '10': 'message'
    },
    {
      '1': 'sealed_sender_verified',
      '3': 2,
      '4': 1,
      '5': 8,
      '10': 'sealedSenderVerified'
    },
    {'1': 'hop_count', '3': 3, '4': 1, '5': 4, '10': 'hopCount'},
  ],
};

/// Descriptor for `IncomingMessageEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List incomingMessageEventDescriptor = $convert.base64Decode(
    'ChRJbmNvbWluZ01lc3NhZ2VFdmVudBIuCgdtZXNzYWdlGAEgASgLMhQueWdnZ3JhbS5DaGF0TW'
    'Vzc2FnZVIHbWVzc2FnZRI0ChZzZWFsZWRfc2VuZGVyX3ZlcmlmaWVkGAIgASgIUhRzZWFsZWRT'
    'ZW5kZXJWZXJpZmllZBIbCglob3BfY291bnQYAyABKARSCGhvcENvdW50');

@$core.Deprecated('Use messageStatusEventDescriptor instead')
const MessageStatusEvent$json = {
  '1': 'MessageStatusEvent',
  '2': [
    {'1': 'message_id', '3': 1, '4': 1, '5': 9, '10': 'messageId'},
    {'1': 'conversation_id', '3': 2, '4': 1, '5': 9, '10': 'conversationId'},
    {
      '1': 'written_to_overlay',
      '3': 3,
      '4': 1,
      '5': 8,
      '10': 'writtenToOverlay'
    },
    {'1': 'delivered', '3': 4, '4': 1, '5': 8, '10': 'delivered'},
    {'1': 'ack_timestamp_ms', '3': 5, '4': 1, '5': 4, '10': 'ackTimestampMs'},
  ],
};

/// Descriptor for `MessageStatusEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List messageStatusEventDescriptor = $convert.base64Decode(
    'ChJNZXNzYWdlU3RhdHVzRXZlbnQSHQoKbWVzc2FnZV9pZBgBIAEoCVIJbWVzc2FnZUlkEicKD2'
    'NvbnZlcnNhdGlvbl9pZBgCIAEoCVIOY29udmVyc2F0aW9uSWQSLAoSd3JpdHRlbl90b19vdmVy'
    'bGF5GAMgASgIUhB3cml0dGVuVG9PdmVybGF5EhwKCWRlbGl2ZXJlZBgEIAEoCFIJZGVsaXZlcm'
    'VkEigKEGFja190aW1lc3RhbXBfbXMYBSABKARSDmFja1RpbWVzdGFtcE1z');

@$core.Deprecated('Use backupProgressEventDescriptor instead')
const BackupProgressEvent$json = {
  '1': 'BackupProgressEvent',
  '2': [
    {'1': 'backup_id', '3': 1, '4': 1, '5': 4, '10': 'backupId'},
    {'1': 'completed_chunks', '3': 2, '4': 1, '5': 13, '10': 'completedChunks'},
    {'1': 'total_chunks', '3': 3, '4': 1, '5': 13, '10': 'totalChunks'},
    {
      '1': 'progress_percentage',
      '3': 4,
      '4': 1,
      '5': 2,
      '10': 'progressPercentage'
    },
    {'1': 'current_action', '3': 5, '4': 1, '5': 9, '10': 'currentAction'},
    {'1': 'is_finished', '3': 6, '4': 1, '5': 8, '10': 'isFinished'},
    {'1': 'error', '3': 7, '4': 1, '5': 9, '10': 'error'},
  ],
};

/// Descriptor for `BackupProgressEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List backupProgressEventDescriptor = $convert.base64Decode(
    'ChNCYWNrdXBQcm9ncmVzc0V2ZW50EhsKCWJhY2t1cF9pZBgBIAEoBFIIYmFja3VwSWQSKQoQY2'
    '9tcGxldGVkX2NodW5rcxgCIAEoDVIPY29tcGxldGVkQ2h1bmtzEiEKDHRvdGFsX2NodW5rcxgD'
    'IAEoDVILdG90YWxDaHVua3MSLwoTcHJvZ3Jlc3NfcGVyY2VudGFnZRgEIAEoAlIScHJvZ3Jlc3'
    'NQZXJjZW50YWdlEiUKDmN1cnJlbnRfYWN0aW9uGAUgASgJUg1jdXJyZW50QWN0aW9uEh8KC2lz'
    'X2ZpbmlzaGVkGAYgASgIUgppc0ZpbmlzaGVkEhQKBWVycm9yGAcgASgJUgVlcnJvcg==');

@$core.Deprecated('Use incomingCallEventDescriptor instead')
const IncomingCallEvent$json = {
  '1': 'IncomingCallEvent',
  '2': [
    {'1': 'call_id', '3': 1, '4': 1, '5': 9, '10': 'callId'},
    {'1': 'caller_ipv6', '3': 2, '4': 1, '5': 9, '10': 'callerIpv6'},
    {
      '1': 'caller_contact',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.ygggram.ContactCard',
      '10': 'callerContact'
    },
    {'1': 'has_video', '3': 4, '4': 1, '5': 8, '10': 'hasVideo'},
    {
      '1': 'signal',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.ygggram.CallSignalContent',
      '10': 'signal'
    },
  ],
};

/// Descriptor for `IncomingCallEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List incomingCallEventDescriptor = $convert.base64Decode(
    'ChFJbmNvbWluZ0NhbGxFdmVudBIXCgdjYWxsX2lkGAEgASgJUgZjYWxsSWQSHwoLY2FsbGVyX2'
    'lwdjYYAiABKAlSCmNhbGxlcklwdjYSOwoOY2FsbGVyX2NvbnRhY3QYAyABKAsyFC55Z2dncmFt'
    'LkNvbnRhY3RDYXJkUg1jYWxsZXJDb250YWN0EhsKCWhhc192aWRlbxgEIAEoCFIIaGFzVmlkZW'
    '8SMgoGc2lnbmFsGAUgASgLMhoueWdnZ3JhbS5DYWxsU2lnbmFsQ29udGVudFIGc2lnbmFs');

@$core.Deprecated('Use coreEventDescriptor instead')
const CoreEvent$json = {
  '1': 'CoreEvent',
  '2': [
    {'1': 'event_id', '3': 1, '4': 1, '5': 4, '10': 'eventId'},
    {'1': 'timestamp_ms', '3': 2, '4': 1, '5': 4, '10': 'timestampMs'},
    {
      '1': 'network_status',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.ygggram.NetworkStatusEvent',
      '9': 0,
      '10': 'networkStatus'
    },
    {
      '1': 'peers_updated',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.ygggram.PeerListUpdatedEvent',
      '9': 0,
      '10': 'peersUpdated'
    },
    {
      '1': 'anti_dpi_status',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.ygggram.AntiDpiStatusEvent',
      '9': 0,
      '10': 'antiDpiStatus'
    },
    {
      '1': 'incoming_message',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.ygggram.IncomingMessageEvent',
      '9': 0,
      '10': 'incomingMessage'
    },
    {
      '1': 'backup_progress',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.ygggram.BackupProgressEvent',
      '9': 0,
      '10': 'backupProgress'
    },
    {
      '1': 'incoming_call',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.ygggram.IncomingCallEvent',
      '9': 0,
      '10': 'incomingCall'
    },
    {
      '1': 'message_status',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.ygggram.MessageStatusEvent',
      '9': 0,
      '10': 'messageStatus'
    },
    {
      '1': 'contact_updated',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.ygggram.ContactResolvedEvent',
      '9': 0,
      '10': 'contactUpdated'
    },
  ],
  '8': [
    {'1': 'event'},
  ],
};

/// Descriptor for `CoreEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List coreEventDescriptor = $convert.base64Decode(
    'CglDb3JlRXZlbnQSGQoIZXZlbnRfaWQYASABKARSB2V2ZW50SWQSIQoMdGltZXN0YW1wX21zGA'
    'IgASgEUgt0aW1lc3RhbXBNcxJECg5uZXR3b3JrX3N0YXR1cxgDIAEoCzIbLnlnZ2dyYW0uTmV0'
    'd29ya1N0YXR1c0V2ZW50SABSDW5ldHdvcmtTdGF0dXMSRAoNcGVlcnNfdXBkYXRlZBgEIAEoCz'
    'IdLnlnZ2dyYW0uUGVlckxpc3RVcGRhdGVkRXZlbnRIAFIMcGVlcnNVcGRhdGVkEkUKD2FudGlf'
    'ZHBpX3N0YXR1cxgFIAEoCzIbLnlnZ2dyYW0uQW50aURwaVN0YXR1c0V2ZW50SABSDWFudGlEcG'
    'lTdGF0dXMSSgoQaW5jb21pbmdfbWVzc2FnZRgGIAEoCzIdLnlnZ2dyYW0uSW5jb21pbmdNZXNz'
    'YWdlRXZlbnRIAFIPaW5jb21pbmdNZXNzYWdlEkcKD2JhY2t1cF9wcm9ncmVzcxgHIAEoCzIcLn'
    'lnZ2dyYW0uQmFja3VwUHJvZ3Jlc3NFdmVudEgAUg5iYWNrdXBQcm9ncmVzcxJBCg1pbmNvbWlu'
    'Z19jYWxsGAggASgLMhoueWdnZ3JhbS5JbmNvbWluZ0NhbGxFdmVudEgAUgxpbmNvbWluZ0NhbG'
    'wSRAoObWVzc2FnZV9zdGF0dXMYCSABKAsyGy55Z2dncmFtLk1lc3NhZ2VTdGF0dXNFdmVudEgA'
    'Ug1tZXNzYWdlU3RhdHVzEkgKD2NvbnRhY3RfdXBkYXRlZBgKIAEoCzIdLnlnZ2dyYW0uQ29udG'
    'FjdFJlc29sdmVkRXZlbnRIAFIOY29udGFjdFVwZGF0ZWRCBwoFZXZlbnQ=');
