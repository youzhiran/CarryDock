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

final logger = Logger(
  // 确保所有级别的日志都被处理
  level: Level.trace,
  printer: MinimalLogPrinter(),
);
