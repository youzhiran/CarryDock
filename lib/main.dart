import 'dart:async';

import 'package:bitsdojo_window/bitsdojo_window.dart';
import 'package:carrydock/l10n/app_localizations.dart';
import 'package:carrydock/providers/language_provider.dart';
import 'package:carrydock/services/settings_service.dart';
import 'package:carrydock/utils/error_handler.dart';
import 'package:carrydock/utils/logger.dart';
import 'package:fluent_ui/fluent_ui.dart';
import 'package:provider/provider.dart';

import 'providers/developer_options_provider.dart';
import 'providers/theme_provider.dart';
import 'providers/update_provider.dart';
import 'screens/home_screen.dart';
import 'screens/developer_options_screen.dart';
import 'screens/archive_manager_screen.dart';
import 'screens/settings/settings_screen.dart';

void main() {
  // 使用 GlobalKey 来访问 Navigator 状态
  final navigatorKey = GlobalKey<NavigatorState>();
  // 实例化错误处理器
  final errorHandler = ErrorHandler(navigatorKey);

  // 使用 runZonedGuarded 捕获所有未处理的错误
  runZonedGuarded<Future<void>>(
    () async {
      // 初始化日志管理器
      final settingsService = SettingsService();
      final enableFileLogging = await settingsService.getEnableFileLogging();
      final logFilePath = await settingsService.getLogFilePath();
      
      // 初始化日志管理器
      await LogManager().initialize(
        enableFileLogging: enableFileLogging,
        logFilePath: logFilePath,
      );
      
      logger.i('应用启动');
      // 初始化错误处理器
      errorHandler.init();

      // 创建语言 Provider 并初始化
      final languageProvider = LanguageProvider();
      await languageProvider.initialize();

      runApp(
        MultiProvider(
          providers: [
            ChangeNotifierProvider(create: (context) => ThemeProvider()),
            ChangeNotifierProvider(
              create: (context) => DeveloperOptionsProvider(),
            ),
            ChangeNotifierProvider(create: (context) => UpdateProvider()),
            ChangeNotifierProvider.value(value: languageProvider),
            Provider<ErrorHandler>.value(value: errorHandler),
            Provider<SettingsService>.value(value: settingsService),
          ],
          child: MyApp(navigatorKey: navigatorKey),
        ),
      );

      doWhenWindowReady(() {
        final win = appWindow;
        const initialSize = Size(960, 640);
        win.minSize = const Size(640, 480);
        win.size = initialSize;
        win.alignment = Alignment.center;
        win.title = "CarryDock";
        win.show();
      });
    },
    (error, stack) {
      // 所有未捕获的错误最终都会在这里被处理
      errorHandler.handleError(error, stack);
    },
  );
}

class MyApp extends StatelessWidget {
  final GlobalKey<NavigatorState> navigatorKey;

  const MyApp({super.key, required this.navigatorKey});

  @override
  Widget build(BuildContext context) {
    final themeProvider = context.watch<ThemeProvider>();
    final languageProvider = context.watch<LanguageProvider>();

    return FluentApp(
      navigatorKey: navigatorKey,
      onGenerateTitle: (context) => AppLocalizations.of(context)!.appName,
      theme: FluentThemeData(
        fontFamily: themeProvider.fontFamily,
        brightness: Brightness.light,
        accentColor: Colors.blue,
        visualDensity: VisualDensity.standard,
        focusTheme: FocusThemeData(
          glowFactor: is10footScreen(context) ? 2.0 : 0.0,
        ),
      ),
      darkTheme: FluentThemeData(
        fontFamily: themeProvider.fontFamily,
        brightness: Brightness.dark,
        accentColor: Colors.blue,
        visualDensity: VisualDensity.standard,
        focusTheme: FocusThemeData(
          glowFactor: is10footScreen(context) ? 2.0 : 0.0,
        ),
      ),
      locale: languageProvider.currentLocale,
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: const MyHomePage(),
    );
  }
}

class MyHomePage extends StatefulWidget {
  const MyHomePage({super.key});

  @override
  State<MyHomePage> createState() => _MyHomePageState();
}

class _MyHomePageState extends State<MyHomePage> {
  int _currentIndex = 0;

  final ValueNotifier<int> _homeScreenNotifier = ValueNotifier<int>(0);
  bool _isPaneOpen = true;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final l10n = AppLocalizations.of(context)!;
    appWindow.title = l10n.appName;
  }

  @override
  Widget build(BuildContext context) {
    final developerOptionsEnabled = context
        .watch<DeveloperOptionsProvider>()
        .enabled;
    final l10n = AppLocalizations.of(context)!;

    final footerItems = <NavigationPaneItem>[];
    if (developerOptionsEnabled) {
      footerItems.add(
        PaneItem(
          icon: const Icon(FluentIcons.developer_tools),
          title: Text(l10n.navDeveloperOptions),
          body: const DeveloperOptionsScreen(),
        ),
      );
    }
    footerItems.add(
      PaneItem(
        icon: const Icon(FluentIcons.settings),
        title: Text(l10n.navSettings),
        body: const SettingsScreen(),
      ),
    );

    return Column(
      children: [
        // 自定义标题栏
        WindowTitleBarBox(
          child: Container(
            color: FluentTheme.of(context).micaBackgroundColor,
            child: Row(
              children: [
                Expanded(child: MoveWindow()),
                const WindowButtons(),
              ],
            ),
          ),
        ),
        // 应用主体
        Expanded(
          child: NavigationView(
            pane: NavigationPane(
              size: const NavigationPaneSize(openWidth: 200, compactWidth: 56),
              selected: _currentIndex,
              onChanged: (index) => setState(() => _currentIndex = index),
              displayMode: _isPaneOpen
                  ? PaneDisplayMode.open
                  : PaneDisplayMode.compact,
              toggleable: false,
              menuButton: Tooltip(
                message: _isPaneOpen ? l10n.navToggleTooltip : l10n.navExpandTooltip,
                child: IconButton(
                  icon: const Icon(FluentIcons.global_nav_button),
                  onPressed: () => setState(() {
                    _isPaneOpen = !_isPaneOpen;
                  }),
                ),
              ),
              header: _isPaneOpen
                  ? Text(l10n.appName, style: const TextStyle(fontSize: 20))
                  : null,
              items: [
                PaneItem(
                  icon: const Icon(FluentIcons.home),
                  title: Text(l10n.navHome),
                  body: ValueListenableBuilder(
                    valueListenable: _homeScreenNotifier,
                    builder: (context, _, __) => const HomeScreen(),
                  ),
                ),
                PaneItem(
                  icon: const Icon(FluentIcons.archive),
                  title: Text(l10n.navArchive),
                  body: const ArchiveManagerScreen(),
                ),
              ],
              footerItems: footerItems,
            ),
          ),
        ),
      ],
    );
  }
}

// 窗口按钮 (最小化, 最大化, 关闭)
class WindowButtons extends StatelessWidget {
  const WindowButtons({super.key});

  @override
  Widget build(BuildContext context) {
    final theme = FluentTheme.of(context);
    final buttonColors = WindowButtonColors(
      iconNormal: theme.inactiveColor,
      mouseOver: theme.accentColor.lighter,
      mouseDown: theme.accentColor.darker,
      iconMouseOver: theme.inactiveColor,
      iconMouseDown: theme.inactiveBackgroundColor,
    );
    final closeButtonColors = WindowButtonColors(
      mouseOver: Colors.red,
      mouseDown: Colors.red.dark,
      iconNormal: theme.inactiveColor,
      iconMouseOver: Colors.white,
    );
    return Row(
      children: [
        MinimizeWindowButton(colors: buttonColors),
        MaximizeWindowButton(colors: buttonColors),
        CloseWindowButton(colors: closeButtonColors),
      ],
    );
  }
}
