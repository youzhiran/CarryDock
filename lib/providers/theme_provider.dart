import 'package:carrydock/services/settings_service.dart';
import 'package:fluent_ui/fluent_ui.dart';

class ThemeProvider extends ChangeNotifier {
  final SettingsService _settingsService = SettingsService();
  String _fontFamily = 'Microsoft YaHei UI';

  String get fontFamily => _fontFamily;

  ThemeProvider() {
    loadFontFamily();
  }

  Future<void> loadFontFamily() async {
    final savedFont = await _settingsService.getFontFamily();
    if (savedFont != null) {
      _fontFamily = savedFont;
      notifyListeners();
    }
  }

  Future<void> updateFontFamily(String newFontFamily) async {
    _fontFamily = newFontFamily;
    await _settingsService.saveFontFamily(newFontFamily);
    notifyListeners();
  }
}
