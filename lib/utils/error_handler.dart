import 'package:fluent_ui/fluent_ui.dart';
import 'package:flutter/foundation.dart';

import 'logger.dart';

class ErrorHandler {
  final GlobalKey<NavigatorState> navigatorKey;
  bool _isShowingDialog = false;

  ErrorHandler(this.navigatorKey);

  /// 初始化错误处理器
  void init() {
    // 捕获 Flutter 框架内的错误
    FlutterError.onError = (FlutterErrorDetails details) {
      if (_isLayoutOverflowError(details.exception)) {
        logger.w(
          '捕获到 Flutter 布局溢出警告',
          error: details.exception,
          stackTrace: details.stack,
        );
        return;
      }

      logger.e(
        '捕获到 Flutter 框架错误',
        error: details.exception,
        stackTrace: details.stack,
      );
      _showErrorDialog('Flutter 错误', details.exception, details.stack);
    };

    // 捕获 Dart Isolate 中的错误 (例如 async/await)
    PlatformDispatcher.instance.onError = (error, stack) {
      logger.e('捕获到 Dart Isolate 错误', error: error, stackTrace: stack);
      _showErrorDialog('应用程序错误', error, stack);
      return true; // 表示错误已处理
    };
  }

  /// 手动处理错误，用于 try-catch 块中
  void handleError(Object error, StackTrace? stackTrace) {
    logger.e('手动处理错误', error: error, stackTrace: stackTrace);
    _showErrorDialog('操作失败', error, stackTrace);
  }

  /// 用于展示指导用户操作的提示对话框。
  void showHint(String title, String message) {
    if (_isShowingDialog) {
      return;
    }
    _isShowingDialog = true;

    Future<void>.delayed(Duration.zero, () async {
      try {
        final navigator = navigatorKey.currentState;
        if (navigator == null || !navigator.mounted) {
          logger.w('无法显示提示对话框，因为 navigator 未初始化或已卸载');
          return;
        }
        await showDialog(
          context: navigator.context,
          builder: (context) => ContentDialog(
            title: Text(title),
            content: SingleChildScrollView(child: SelectableText(message)),
            actions: [
              FilledButton(
                child: const Text('好的'),
                onPressed: () => Navigator.of(context).pop(),
              ),
            ],
          ),
        );
      } finally {
        _isShowingDialog = false;
      }
    });
  }

  /// 显示一个统一的错误对话框
  void _showErrorDialog(String title, Object error, [StackTrace? stackTrace]) {
    // if (_isLayoutOverflowError(error)) {
    //   logger.w('布局溢出警告', error: error);
    //   return;
    // }
    logger.e(error, stackTrace: stackTrace);
    if (_isShowingDialog) {
      return;
    }
    _isShowingDialog = true;

    Future<void>.delayed(Duration.zero, () async {
      try {
        final navigator = navigatorKey.currentState;
        if (navigator == null || !navigator.mounted) {
          logger.w('无法显示错误对话框，因为 navigator 未初始化或已卸载');
          return;
        }
        await showDialog(
          context: navigator.context,
          builder: (context) => ContentDialog(
            title: Text(title),
            content: SingleChildScrollView(
              child: SelectableText(
                '发生了一个未处理的错误:\n\n$error\n\n堆栈跟踪:\n$stackTrace',
              ),
            ),
            actions: [
              FilledButton(
                child: const Text('好的'),
                onPressed: () => Navigator.of(context).pop(),
              ),
            ],
          ),
        );
      } finally {
        _isShowingDialog = false;
      }
    });
  }

  bool _isLayoutOverflowError(Object error) {
    final message = error.toString();
    return message.contains('RenderFlex overflowed');
  }
}
