enum SoftwareStatus { managed, unknownInstall, unknownArchive }

class Software {
  final String id;
  final String name;
  final String installPath;
  String executablePath;
  final String archivePath;
  final String? iconPath;
  bool archiveExists;
  // 运行时状态：安装目录是否存在（不进行持久化，仅用于 UI 呈现与交互判断）。
  bool installExists;
  SoftwareStatus status;
  int sortOrder;

  Software({
    required this.id,
    required this.name,
    this.installPath = '',
    this.executablePath = '',
    this.archivePath = '',
    this.iconPath,
    this.archiveExists = false,
    this.installExists = true,
    this.status = SoftwareStatus.managed,
    this.sortOrder = 0,
  });

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'installPath': installPath,
      'executablePath': executablePath,
      'archivePath': archivePath,
      'iconPath': iconPath,
      'sortOrder': sortOrder,
    };
  }

  factory Software.fromJson(Map<String, dynamic> json) {
    return Software(
      id: json['id'],
      name: json['name'],
      installPath: json['installPath'],
      executablePath: json['executablePath'],
      archivePath: json['archivePath'],
      iconPath: json['iconPath'],
      sortOrder: json['sortOrder'] is int ? json['sortOrder'] as int : 0,
    );
  }
}
