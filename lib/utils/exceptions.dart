/// 当在单个软件目录中找到多个可执行文件时抛出，
/// 需要用户进行选择。
class MultipleExecutablesFoundException implements Exception {
  /// 可执行文件的路径列表。
  final List<String> executablePaths;

  /// 软件解压后的安装路径。
  final String installPath;

  /// 软件的归档文件路径。
  final String archivePath;

  MultipleExecutablesFoundException(
    this.executablePaths,
    this.installPath,
    this.archivePath,
  );

  @override
  String toString() {
    return 'Multiple executables found in $installPath: $executablePaths';
  }
}
