import 'package:carrydock/services/settings_service.dart';
import 'package:flutter/foundation.dart';

class DeveloperOptionsProvider extends ChangeNotifier {
  final SettingsService _settingsService;
  bool _enabled = false;
  bool _initialized = false;

  bool get enabled => _enabled;

  bool get initialized => _initialized;

  DeveloperOptionsProvider({SettingsService? settingsService})
    : _settingsService = settingsService ?? SettingsService() {
    _load();
  }

  Future<void> _load() async {
    final stored = await _settingsService.getDeveloperOptionsEnabled();
    _enabled = stored;
    _initialized = true;
    notifyListeners();
  }

  Future<void> setEnabled(bool value) async {
    if (_enabled == value) {
      return;
    }
    _enabled = value;
    notifyListeners();
    await _settingsService.saveDeveloperOptionsEnabled(value);
  }

  Future<void> refresh() async {
    await _load();
  }
}
