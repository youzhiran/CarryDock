import 'package:flutter/services.dart';

/// 设置屏幕相关常量和辅助方法

/// 保存全部动画持续时间
const Duration saveAllAnimationDuration = Duration(milliseconds: 280);

/// 最大搜索深度选项列表
final List<int> maxSearchDepthOptions = List<int>.generate(
  20,
  (index) => index + 1,
);

/// 常用可执行文件扩展名列表
const List<String> commonExecutableExtensions = [
  'exe',
  'bat',
  'cmd',
  'com',
  'lnk',
  'msi',
  'ps1',
  'vbs',
];

/// 可用字体列表
const List<String> availableFonts = [
  'PingFang SC',
  'HarmonyOS Sans SC',
  'Maple Mono Normal NF CN',
  'Microsoft YaHei UI',
  'SimSun',
  'Segoe UI',
  'Arial',
  'Verdana',
];

/// 规范化扩展名列表
List<String> normalizeExtensionsList(Iterable<String> extensions) {
  final normalized = <String>{};
  for (final ext in extensions) {
    final trimmed = ext.trim().toLowerCase();
    if (trimmed.isEmpty) continue;
    final cleaned = trimmed.startsWith('.') ? trimmed.substring(1) : trimmed;
    if (cleaned.isEmpty) continue;
    normalized.add(cleaned);
  }
  final sorted = normalized.toList()..sort();
  return sorted;
}

/// 规范化搜索深度
int normalizeDepth(int depth) {
  if (depth < maxSearchDepthOptions.first) {
    return maxSearchDepthOptions.first;
  }
  if (depth > maxSearchDepthOptions.last) {
    return maxSearchDepthOptions.last;
  }
  return depth;
}

/// 将数字格式化为两位数字符串
String twoDigits(int n) {
  return n.toString().padLeft(2, '0');
}

/// 格式化构建时间
String formatBuildTime(String rawTime) {
  try {
    // 尝试解析时间字符串
    final dateTime = DateTime.parse(rawTime.trim());
    // 格式化只显示到秒
    return '${dateTime.year}-${twoDigits(dateTime.month)}-${twoDigits(dateTime.day)} ${twoDigits(dateTime.hour)}:${twoDigits(dateTime.minute)}:${twoDigits(dateTime.second)}';
  } catch (e) {
    // 如果解析失败，尝试直接截取到秒
    final trimmed = rawTime.trim();
    if (trimmed.contains('.')) {
      return trimmed.split('.')[0];
    }
    return trimmed;
  }
}

/// 读取构建信息
Future<String> readBuildInfo() async {
  try {
    final buildTime = await rootBundle.loadString('build_info.txt');
    return formatBuildTime(buildTime);
  } catch (e) {
    return '';
  }
}
