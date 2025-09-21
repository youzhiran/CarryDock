import 'dart:ffi';
import 'dart:typed_data';

import 'package:ffi/ffi.dart';
import 'package:image/image.dart' as img;
import 'package:win32/win32.dart';

const int _diNormal = 0x0003;

void _free<T extends NativeType>(Pointer<T>? pointer) {
  if (pointer != null && pointer.address != 0) {
    calloc.free(pointer);
  }
}

final _shell32 = DynamicLibrary.open('shell32.dll');
final _extractIconEx = _shell32
    .lookupFunction<
      Uint32 Function(
        Pointer<Utf16> lpszFile,
        Int32 nIconIndex,
        Pointer<IntPtr> phiconLarge,
        Pointer<IntPtr> phiconSmall,
        Uint32 nIcons,
      ),
      int Function(
        Pointer<Utf16> lpszFile,
        int nIconIndex,
        Pointer<IntPtr> phiconLarge,
        Pointer<IntPtr> phiconSmall,
        int nIcons,
      )
    >('ExtractIconExW');

final _user32 = DynamicLibrary.open('user32.dll');
final _drawIconEx = _user32
    .lookupFunction<
      Int32 Function(
        IntPtr hdc,
        Int32 xLeft,
        Int32 yTop,
        IntPtr hIcon,
        Int32 cxWidth,
        Int32 cyHeight,
        Uint32 istepIfAniCur,
        IntPtr hbrFlickerFreeDraw,
        Uint32 diFlags,
      ),
      int Function(
        int hdc,
        int xLeft,
        int yTop,
        int hIcon,
        int cxWidth,
        int cyHeight,
        int istepIfAniCur,
        int hbrFlickerFreeDraw,
        int diFlags,
      )
    >('DrawIconEx');

/// 一个用于提取 Windows 可执行文件 (.exe) 信息的服务。
class ExecutableInfoService {
  /// 从给定的 .exe 文件路径中提取文件说明。
  ///
  /// 返回文件说明字符串，如果找不到则返回 null。
  Future<String?> getFileDescription(String exePath) async {
    Pointer<Utf16>? exePathPtr;
    Pointer<DWORD>? handle;
    Pointer<BYTE>? info;
    Pointer<Pointer<UINT>>? langCodepagePtr;
    Pointer<UINT>? langCodepageSize;
    Pointer<Utf16>? translationKey;
    Pointer<Utf16>? query;
    Pointer<Pointer<Utf16>>? descriptionPtr;
    Pointer<UINT>? descriptionSize;

    try {
      exePathPtr = exePath.toNativeUtf16();
      handle = calloc<DWORD>();

      final infoSize = GetFileVersionInfoSize(exePathPtr, handle);
      if (infoSize == 0) return null;

      info = calloc<BYTE>(infoSize);
      if (GetFileVersionInfo(exePathPtr, 0, infoSize, info) == 0) {
        return null;
      }

      langCodepagePtr = calloc<Pointer<UINT>>();
      langCodepageSize = calloc<UINT>();
      translationKey = r'\\VarFileInfo\\Translation'.toNativeUtf16();

      if (VerQueryValue(
            info,
            translationKey,
            langCodepagePtr,
            langCodepageSize,
          ) ==
          0) {
        return null;
      }

      final translationLength = langCodepageSize.value;
      if (translationLength < sizeOf<WORD>() * 2) {
        return null;
      }

      final translation = langCodepagePtr.value.cast<WORD>();
      if (translation.address == 0) {
        return null;
      }

      final lang = translation[0].toRadixString(16).padLeft(4, '0');
      final codepage = translation[1].toRadixString(16).padLeft(4, '0');

      query = '\\StringFileInfo\\$lang$codepage\\FileDescription'
          .toNativeUtf16();
      descriptionPtr = calloc<Pointer<Utf16>>();
      descriptionSize = calloc<UINT>();

      if (VerQueryValue(info, query, descriptionPtr, descriptionSize) == 0) {
        return null;
      }

      if (descriptionPtr.value.address == 0 || descriptionSize.value == 0) {
        return null;
      }

      final description = descriptionPtr.value.toDartString().trim();
      return description.isNotEmpty ? description : null;
    } catch (e) {
      return null;
    } finally {
      _free(descriptionSize);
      _free(descriptionPtr);
      _free(query);
      _free(translationKey);
      _free(langCodepageSize);
      _free(langCodepagePtr);
      _free(info);
      _free(handle);
      _free(exePathPtr);
    }
  }

  /// 从给定的 .exe 文件路径中提取主图标。
  ///
  /// 使用 DrawIconEx 将图标绘制到 32 位 DIBSection 上，以正确处理所有图标格式（包括带 alpha 通道的现代图标和带掩码的旧式图标）的透明度。
  /// 返回图标的 PNG 编码字节数据 (Uint8List)，如果找不到则返回 null。
  Future<Uint8List?> getIcon(String exePath) async {
    final exePathPtr = exePath.toNativeUtf16();
    final largeIcon = calloc<IntPtr>();
    final smallIcon = calloc<IntPtr>();

    // GDI 资源需要清理。
    final iconInfo = calloc<ICONINFO>();
    int hdcScreen = 0;
    int hdcMem = 0;
    int hDib = 0;
    int hOldBitmap = 0;

    // ffi 分配的内存需要清理。
    final bmp = calloc<BITMAP>();
    final bmi = calloc<BITMAPINFO>();
    final bits = calloc<Pointer<Void>>();

    try {
      final count = _extractIconEx(exePathPtr, 0, largeIcon, smallIcon, 1);
      // 根据文档，如果 count 不为 0，我们必须销毁提取的图标。
      // 这将在 finally 块中处理。
      if (count == 0) return null;

      final hIcon = largeIcon.value != 0 ? largeIcon.value : smallIcon.value;
      if (hIcon == 0) return null;

      if (GetIconInfo(hIcon, iconInfo) == 0) return null;

      final isMonochrome = iconInfo.ref.hbmColor == 0;
      final hBitmap = isMonochrome
          ? iconInfo.ref.hbmMask
          : iconInfo.ref.hbmColor;
      if (hBitmap == 0) return null;

      if (GetObject(hBitmap, sizeOf<BITMAP>(), bmp) == 0) return null;

      final width = bmp.ref.bmWidth;
      var height = bmp.ref.bmHeight;
      if (isMonochrome) {
        height ~/= 2; // mask bitmap stores both AND/OR masks stacked vertically
      }

      if (width <= 0 || height <= 0) return null;

      hdcScreen = GetDC(NULL);
      hdcMem = CreateCompatibleDC(hdcScreen);
      if (hdcMem == 0) return null;

      final bmih = bmi.ref.bmiHeader;
      bmih.biSize = sizeOf<BITMAPINFOHEADER>();
      bmih.biWidth = width;
      bmih.biHeight = -height; // top-down DIB, for easier buffer reading
      bmih.biPlanes = 1;
      bmih.biBitCount = 32;
      bmih.biCompression = BI_RGB;

      hDib = CreateDIBSection(hdcMem, bmi, DIB_RGB_COLORS, bits, NULL, 0);
      if (hDib == 0 || bits.value.address == 0) return null;

      hOldBitmap = SelectObject(hdcMem, hDib);
      if (hOldBitmap == 0) return null;

      // 将图标绘制到 DIBSection。这能正确处理透明度。
      if (_drawIconEx(hdcMem, 0, 0, hIcon, width, height, 0, NULL, _diNormal) ==
          0) {
        return null;
      }

      final bufferSize = width * height * 4;
      final buffer = bits.value.cast<BYTE>().asTypedList(bufferSize);

      final image = img.Image.fromBytes(
        width: width,
        height: height,
        bytes: buffer.buffer,
        order: img.ChannelOrder.bgra,
      );

      final pngData = img.encodePng(image);
      return Uint8List.fromList(pngData);
    } catch (e) {
      // 在实际应用中最好记录错误。暂时遵循原始行为。
      return null;
    } finally {
      // 恢复 DC
      if (hOldBitmap != 0) SelectObject(hdcMem, hOldBitmap);

      // 释放 GDI 对象
      if (hDib != 0) DeleteObject(hDib);
      if (hdcMem != 0) DeleteDC(hdcMem);
      if (hdcScreen != 0) ReleaseDC(NULL, hdcScreen);

      // GetIconInfo 创建了位图的副本，我们必须删除它们。
      if (iconInfo.ref.hbmColor != 0) DeleteObject(iconInfo.ref.hbmColor);
      if (iconInfo.ref.hbmMask != 0) DeleteObject(iconInfo.ref.hbmMask);

      // 销毁从可执行文件中提取的图标。
      if (largeIcon.value != 0) DestroyIcon(largeIcon.value);
      if (smallIcon.value != 0) DestroyIcon(smallIcon.value);

      // 释放 ffi 分配的内存
      _free(exePathPtr);
      _free(largeIcon);
      _free(smallIcon);
      _free(iconInfo);
      _free(bmp);
      _free(bmi);
      _free(bits);
    }
  }
}
