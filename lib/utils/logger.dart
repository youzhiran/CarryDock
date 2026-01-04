import 'dart:convert';
import 'dart:io';
import 'package:logger/logger.dart';

/// 一个自定义的日志打印机，提供简洁的单行输出，但在错误发生时打印完整的堆栈跟踪。
class MinimalLogPrinter extends LogPrinter {
  static final levelColors = {
    Level.trace: AnsiColor.fg(AnsiColor.grey(0.5)),
    Level.debug: AnsiColor.fg(6), // Cyan
    Level.info: AnsiColor.fg(2), // Green
    Level.warning: AnsiColor.fg(3), // Yellow
    Level.error: AnsiColor.fg(1), // Red
    Level.fatal: AnsiColor.fg(1), // Red
  };

  @override
  List<String> log(LogEvent event) {
    final color = levelColors[event.level]!;
    // 将时间格式化为 HH:mm:ss
    final time = DateTime.now().toIso8601String().substring(11, 19);
    final levelPrefix = '[${event.level.name.toUpperCase()}]'.padRight(8);

    final output = <String>[];
    // 打印主要日志信息
    output.add(color('$time $levelPrefix ${event.message}'));

    // 如果有错误对象，打印它
    if (event.error != null) {
      output.add(color('  Error: ${event.error}'));
    }

    // 如果有堆栈跟踪，完整地打印它
    if (event.stackTrace != null) {
      output.add(color(event.stackTrace.toString()));
    }

    return output;
  }
}

/// 用于文件输出的日志打印机，不包含颜色代码
class FileLogPrinter extends LogPrinter {
  @override
  List<String> log(LogEvent event) {
    // 将时间格式化为 YYYY-MM-DD HH:mm:ss
    final time = DateTime.now().toIso8601String().substring(0, 19);
    final levelPrefix = '[${event.level.name.toUpperCase()}]'.padRight(8);

    final output = <String>[];
    // 打印主要日志信息
    output.add('$time $levelPrefix ${event.message}');

    // 如果有错误对象，打印它
    if (event.error != null) {
      output.add('  Error: ${event.error}');
    }

    // 如果有堆栈跟踪，完整地打印它
    if (event.stackTrace != null) {
      output.add(event.stackTrace.toString());
    }

    return output;
  }
}

/// 日志管理器，负责配置和管理日志输出
class LogManager {
  static final LogManager _instance = LogManager._internal();
  factory LogManager() => _instance;

  LogManager._internal() {
    // 初始化默认的日志记录器，仅输出到控制台
    _initializeDefaultLogger();
  }

  late Logger _logger;
  bool _isFileLoggingEnabled = false;
  String _logFilePath = '';
  bool _isInitialized = false;

  /// 初始化默认的日志记录器，仅输出到控制台
  void _initializeDefaultLogger() {
    _logger = Logger(
      level: Level.trace,
      printer: MinimalLogPrinter(),
      output: ConsoleOutput(),
    );
  }

  /// 初始化日志管理器
  Future<void> initialize({bool enableFileLogging = false, String logFilePath = ''}) async {
    _isFileLoggingEnabled = enableFileLogging;
    _logFilePath = logFilePath;

    final outputs = <LogOutput>[];
    // 始终输出到控制台
    outputs.add(ConsoleOutput());

    // 如果启用了文件日志，添加文件输出
    if (enableFileLogging && logFilePath.isNotEmpty) {
      try {
        // 确保日志目录存在
        final logDir = Directory(Directory(logFilePath).parent.path);
        if (!logDir.existsSync()) {
          logDir.createSync(recursive: true);
        }
        outputs.add(FileOutput(
          file: File(logFilePath),
          overrideExisting: false,
          encoding: utf8,
        ));
      } catch (e) {
        print('Failed to initialize file logging: $e');
      }
    }

    _logger = Logger(
      level: Level.trace,
      printer: MinimalLogPrinter(),
      output: MultiOutput(outputs),
    );
    
    _isInitialized = true;
  }

  /// 获取日志实例
  Logger get logger {
    // 如果尚未初始化，返回默认的控制台日志记录器
    // 这确保了在 LogManager 初始化之前，logger 也能正常工作
    return _logger;
  }

  /// 更新日志配置
  Future<void> updateConfiguration({bool? enableFileLogging, String? logFilePath}) async {
    await initialize(
      enableFileLogging: enableFileLogging ?? _isFileLoggingEnabled,
      logFilePath: logFilePath ?? _logFilePath,
    );
  }

  /// 设置默认日志文件路径（与配置文件同一目录）
  String getDefaultLogFilePath(String configFilePath) {
    final configDir = Directory(configFilePath).parent.path;
    return '$configDir/carrydock.log';
  }
}

/// 全局日志实例
final logger = LogManager().logger;
