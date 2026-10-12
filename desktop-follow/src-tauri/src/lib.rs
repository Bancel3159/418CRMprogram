use tauri::Manager;

#[tauri::command]
fn greet(name: &str) -> String {
    format!("Hello, {}! 欢迎使用 CRM 维护客户端", name)
}

// ============ P12 系统通知:任务栏闪 + toast ============

/// 闪烁/取消闪烁主窗口任务栏图标
///
/// 收到 realtime 事件时由前端调一次 `flash_window(true)`(开始闪,直到窗口获得焦点自动停);
/// 想强制停可以再调 `flash_window(false)`。
///
/// 实现:Win32 `FlashWindowEx` + FLASHW_TRAY | FLASHW_TIMERNOFG
///   - FLASHW_TRAY(2) = 闪任务栏图标
///   - FLASHW_TIMERNOFG(12) = 即使窗口已经获得焦点也继续闪(uCount 决定次数,0 = 直到前台)
#[tauri::command]
fn flash_window(window: tauri::WebviewWindow, on: bool) -> Result<(), String> {
    #[cfg(target_os = "windows")]
    {
        use windows_sys::Win32::UI::WindowsAndMessaging::{
            FlashWindowEx, FLASHWINFO, FLASHW_TIMERNOFG, FLASHW_TRAY,
        };
        let hwnd_raw = window
            .hwnd()
            .map_err(|e| format!("获取 hwnd 失败: {}", e))?
            .0 as isize as *mut std::ffi::c_void;

        let mut flash_info = FLASHWINFO {
            cbSize: std::mem::size_of::<FLASHWINFO>() as u32,
            hwnd: hwnd_raw,
            dwFlags: if on {
                FLASHW_TRAY | FLASHW_TIMERNOFG
            } else {
                0 // FLASHW_STOP = 0
            },
            uCount: if on { 0 } else { 0 }, // 0 = 直到窗口前台
            dwTimeout: 0,
        };

        unsafe {
            let _ = FlashWindowEx(&mut flash_info);
        }
        log::info!("[window] flash_window on={}", on);
        Ok(())
    }

    #[cfg(not(target_os = "windows"))]
    {
        let _ = window;
        let _ = on;
        log::warn!("[window] flash_window 仅 Windows 实现");
        Ok(())
    }
}

/// 通过 tauri-plugin-notification 触发系统 toast(Win10/11 走 Action Center 通知)
///
/// 用法:前端直接 invoke('plugin:notification|notify', { ... }) 也行,
/// 这里留个简化包装便于测试。
#[tauri::command]
fn show_toast(app: tauri::AppHandle, title: String, body: String) -> Result<(), String> {
    use tauri_plugin_notification::NotificationExt;
    app.notification()
        .builder()
        .title(title)
        .body(body)
        .show()
        .map_err(|e| format!("系统通知失败: {}", e))?;
    Ok(())
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_notification::init()) // P12 系统通知
        .setup(|app| {
            let _ = app.get_webview_window("main");
            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            greet,
            flash_window,
            show_toast
        ])
        .run(tauri::generate_context!())
        .expect("启动 Tauri 应用失败");
}