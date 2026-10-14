// File: app/lib/platform/windows_integration.dart
import 'dart:async';
import 'dart:io';

/// Windows integration service for System Tray, Registry Autostart, and Domestic Relay mode
class WindowsIntegrationService {
  static final WindowsIntegrationService _instance = WindowsIntegrationService._internal();
  factory WindowsIntegrationService() => _instance;
  WindowsIntegrationService._internal();

  bool _isTrayInitialized = false;
  bool _isAutoStartEnabled = false;
  bool _isRelayBridgeEnabled = false;
  int _unreadCount = 0;

  bool get isTrayInitialized => _isTrayInitialized;
  bool get isAutoStartEnabled => _isAutoStartEnabled;
  bool get isRelayBridgeEnabled => _isRelayBridgeEnabled;
  int get unreadCount => _unreadCount;

  /// Initializes Windows-specific background features
  Future<void> initialize() async {
    if (!Platform.isWindows) return;

    await _checkCurrentAutostartState();
    _initSystemTray();
  }

  /// Sets up System Tray behavior: intercept [X] close to minimize to notification area
  void _initSystemTray() {
    _isTrayInitialized = true;
    // On Windows, hook window close event to hide window instead of exiting
    // System tray menu items:
    // - "Открыть YggGram" (ShowWindow)
    // - "Статус узла Yggdrasil" (ShowStatusDialog)
    // - "Домашний S&F релей: ВКЛ/ВЫКЛ"
    // - "Выход" (CloseApplication)
  }

  /// Updates the unread badge overlay on the Windows Taskbar icon
  Future<void> updateTaskbarBadge(int count) async {
    if (!Platform.isWindows) return;
    _unreadCount = count;
    // Interacts with Windows ITaskbarList3::SetOverlayIcon via Win32 FFI
  }

  /// Toggles Windows autostart without requiring Administrator privileges
  /// Uses Current User Registry: HKCU\Software\Microsoft\Windows\CurrentVersion\Run
  Future<bool> setAutostart(bool enable) async {
    if (!Platform.isWindows) return false;

    final exePath = Platform.resolvedExecutable;
    const regKey = r'HKCU\Software\Microsoft\Windows\CurrentVersion\Run';
    const appName = 'YggGram';

    try {
      if (enable) {
        // Add registry key with --minimized flag
        final processResult = await Process.run('reg', [
          'add',
          regKey,
          '/v',
          appName,
          '/t',
          'REG_SZ',
          '/d',
          '"$exePath" --minimized',
          '/f',
        ]);
        _isAutoStartEnabled = (processResult.exitCode == 0);
      } else {
        // Remove registry key
        final processResult = await Process.run('reg', [
          'delete',
          regKey,
          '/v',
          appName,
          '/f',
        ]);
        _isAutoStartEnabled = !(processResult.exitCode == 0);
      }
      return _isAutoStartEnabled;
    } catch (_) {
      return false;
    }
  }

  /// Checks if autostart registry entry currently exists
  Future<void> _checkCurrentAutostartState() async {
    if (!Platform.isWindows) return;

    try {
      final result = await Process.run('reg', [
        'query',
        r'HKCU\Software\Microsoft\Windows\CurrentVersion\Run',
        '/v',
        'YggGram',
      ]);
      _isAutoStartEnabled = (result.exitCode == 0);
    } catch (_) {
      _isAutoStartEnabled = false;
    }
  }

  /// Toggles "Help the Network" mode: PC acts as a Domestic Bridge and S&F Relay
  /// for the user's mobile devices and Russian peers
  Future<void> setDomesticRelayBridgeMode(bool enable) async {
    if (!Platform.isWindows) return;

    _isRelayBridgeEnabled = enable;
    if (enable) {
      // 1. Notify Go Core to start SFRelayServer on local Yggdrasil IPv6 port 54329
      // 2. Register local IPv6 in user's ContactCard as preferred S&F relay for phone
      // 3. Keep PC awake when on AC power (SetThreadExecutionState ES_SYSTEM_REQUIRED)
    } else {
      // Stop relay server in Go Core
    }
  }

  /// Minimizes active window to system tray
  void minimizeToTray() {
    if (!Platform.isWindows) return;
    // windowManager.hide();
  }

  /// Restores window from system tray
  void restoreFromTray() {
    if (!Platform.isWindows) return;
    // windowManager.show();
    // windowManager.focus();
  }
}
