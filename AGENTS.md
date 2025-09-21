# Repository Guidelines

## 项目结构与模块组织

- `lib/` 按职能拆分：`screens/` 承载导航与可视层，建议按业务模块继续分子目录，保持 UI 与逻辑解耦。
- `widgets/` 存放复用组件，编写时同步补充文档注释，示例放在临时 `example` 分支或
  `test/widget_test.dart` 中。
- 状态与数据交互集中在 `providers/`、`services/`，模型定义保存在 `lib/models/`，工具函数放 `lib/utils/`
  ，这些层尽量保持可测试与平台无关。
- 测试位于 `test/`，与 `lib/` 保持镜像结构，命名遵循 `{feature}_test.dart`，便于快速定位覆盖空白。
- 静态资产可放置在 `assets/`（如需新增），记得同步更新 `pubspec.yaml` 的 `flutter.assets` 配置。
- 平台工程分别在 `linux/`、`macos/`、`windows/`，涉及原生插件或渠道配置时务必在对应目录记录改动说明。
- `build/` 与生成产物不应提交，确保 `.gitignore` 规则与 CI 环境一致，减少冲突。

## 构建、测试与开发命令

- 由于正在使用wsl环境，暂时不要运行dart/flutter相关命令。

## 编码风格与命名约定

- 遵循 Flutter 默认 2 空格缩进，行宽控制在 100 列内，超出需拆分表达式。
- 命名遵循驼峰：类使用帕斯卡命名，方法与变量使用小驼峰，常量前缀 `k` 并以功能补充后缀。
- UI 组件文件命名推荐 `feature_purpose.dart`，状态管理文件命名 `feature_provider.dart`，服务端交互命名
  `feature_service.dart`。
- `analysis_options.yaml` 已启用官方 lint，切勿屏蔽规则；如需例外，请在 PR 中写明理由。
- 重要逻辑需补充中文文档注释，说明参数含义、前置条件与返回值，提升协作效率。
- 避免在 widget 中直接维护全局状态，改由 Provider 或 Riverpod 管理，确保可测性与可扩展性。

## 提交与合并规范

- 保持 Git 历史的简洁中文短语风格（示例：`修复拖拽逻辑`、`补全状态同步`），首字使用动词，突出动作与对象。
- 每个 PR 包含：变更说明（含核心改动与风险）、测试结果、关联 Issue 链接；若涉及 UI，附上前后对比截图或录屏。
- 遵循小步提交策略：配置、逻辑、资源分别提交，减少回溯成本；必要时在描述中标注依赖顺序。
- 请求评审前需通过 `flutter analyze` 与 `flutter test`，并在 PR 描述中确认已覆盖的验证步骤。
- 被指派评审者响应前可先自检，使用 checklist（分析、测试、文档）确保内容齐备。
- 分支命名推荐 `feature/<topic>`、`fix/<issue>`、`chore/<task>`，方便 CI 与发布流水线自动识别。

## 安全与配置提示

- 不在仓库中保存敏感凭据，将密钥与 API Token 置于本地 `.env` 或系统变量，通过 `--dart-define` 注入运行时。
- 更新 `pubspec.yaml` 中第三方库时评估 `changelog` 与兼容性，必要时记录迁移步骤和回滚策略。
- 保持日志聚焦调试所需信息，发布版本关闭 `debugPrint`，避免泄露用户或业务数据。
- 若引入后端接口，使用 `services/` 层统一封装请求，并记录超时与重试策略，便于性能追踪。
- 配置 `devtools_options.yaml` 优化调试体验，可在 PR 中共享推荐设置，促进团队一致性。
