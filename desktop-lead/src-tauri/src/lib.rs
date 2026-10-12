use std::sync::Mutex;
use tauri::{Manager, WindowEvent};
use tauri_plugin_global_shortcut::{GlobalShortcutExt, Shortcut, ShortcutState};

/// Windows:调用 DWM API 把窗口设为系统级圆角
///
/// 背景:
/// - tauri.conf.json 配了 `decorations:false + transparent:true + shadow:false`,
///   OS 窗口本身仍是矩形,前端 CSS `border-radius` 只让 webview 内容看着圆,
///   残留在 OS 层的就是矩形 drop-shadow 和边缘 —— 用户反馈"四角还在"。
/// - Win11 提供 `DWMWA_WINDOW_CORNER_PREFERENCE = DWMWCP_ROUND`,
///   可让 DWM 在系统层把窗口绘制成圆角矩形(同时阴影也跟着圆),
///   这样无论 webview 内容怎么排,OS 窗口本身就没矩形角。
#[cfg(target_os = "windows")]
fn apply_windows_rounded_corners(hwnd: *mut std::ffi::c_void) {
    use windows_sys::Win32::Graphics::Dwm::DwmSetWindowAttribute;
    unsafe {
        // DWMWA_WINDOW_CORNER_PREFERENCE = 33,DWMWCP_ROUND = 2
        // windows-sys 0.59 把这俩常量声明成 i32,但 DwmSetWindowAttribute 收 u32,需要手动 cast
        let attribute: u32 = 33;
        let preference: u32 = 2;
        // 返回值忽略:DWM 在不支持的 Win10/旧环境会失败,但窗口仍能用
        let _ = DwmSetWindowAttribute(
            hwnd,
            attribute,
            &preference as *const _ as *const _,
            std::mem::size_of::<u32>() as u32,
        );
    }
}

/// 当前注册的全局快捷键(由前端通过 invoke('set_window_shortcut', ...) 设置)
/// 启动时为 None,前端在 App.vue 挂载后调用 set_window_shortcut 设置默认或用户自定义。
static CURRENT_SHORTCUT: once_cell::sync::Lazy<Mutex<Option<Shortcut>>> =
    once_cell::sync::Lazy::new(|| Mutex::new(None));

/// 解析快捷键组合字符串 → Shortcut
///
/// 接受的格式(由 global-hotkey crate 定义):
///   - 修饰符在前,主键在后,用 `+` 连接
///   - 修饰符:Ctrl / Shift / Alt / Super(CmdOrCtrl 等同 Ctrl)
///   - 主键:字母(A-Z)、数字(0-9)、功能键(F1-F12)、方向键、Backquote、BracketLeft/Right、
///     Backslash、Slash、Comma、Period、Semicolon、Quote、Minus、Equal、Space、Tab、
///     Enter、Escape、Backspace、Delete、Home、End、PageUp、PageDown、Insert 等
///
/// 示例:
///   "Ctrl+\`"        → Ctrl + Backquote
///   "Ctrl+Shift+K"   → Ctrl + Shift + K
///   "Alt+Q"          → Alt + Q
///   "CmdOrCtrl+L"    → Ctrl + L(在 macOS 上等同 Cmd+L)
fn parse_shortcut(combo: &str) -> Result<Shortcut, String> {
    let trimmed = combo.trim();
    if trimmed.is_empty() {
        return Err("快捷键不能为空".into());
    }

    // 用 global-hotkey 自带的 from_str 解析(支持上述所有格式)
    let shortcut: Shortcut = trimmed.parse().map_err(|e: global_hotkey::hotkey::HotKeyParseError| {
        format!("无效的快捷键组合 '{}': {}", trimmed, e)
    })?;

    // 必须包含至少一个修饰键 + 一个非修饰键主键
    // global-hotkey 接受纯 key 单键(无修饰符),但我们要求强制组合(更安全)
    if shortcut.mods.is_empty() {
        return Err("至少需要一个修饰键(Ctrl / Alt / Shift / Super)".into());
    }

    Ok(shortcut)
}

/// 全局快捷键事件 handler:任何已注册的快捷键按下都触发窗口显隐切换
fn handle_shortcut_press(app: &tauri::AppHandle) {
    toggle_main_window(app);
}

/// 切换主窗口显示/隐藏
fn toggle_main_window(app: &tauri::AppHandle) {
    if let Some(window) = app.get_webview_window("main") {
        match window.is_visible() {
            Ok(true) => {
                let _ = window.hide();
                log::info!("[window] hidden via shortcut");
            }
            _ => {
                let _ = window.show();
                let _ = window.set_focus();
                log::info!("[window] shown via shortcut");
            }
        }
    }
}

/// 供前端调用的命令:切换全局快捷键
///
/// 参数:
///   - combo:   "Ctrl+\`" 这类组合字符串;空字符串 "disabled" / "" / "none" → 注销
///
/// 返回:
///   - 成功注册时返回组合字符串;禁用时返回 "disabled"
#[tauri::command]
fn set_window_shortcut(app: tauri::AppHandle, combo: String) -> Result<String, String> {
    // disabled 标记
    let normalized = combo.trim();
    let is_disabled = normalized.is_empty() || normalized.eq_ignore_ascii_case("disabled") || normalized.eq_ignore_ascii_case("none");

    let new_shortcut: Option<Shortcut> = if is_disabled {
        None
    } else {
        Some(parse_shortcut(normalized)?)
    };

    let mut current = CURRENT_SHORTCUT.lock().unwrap();

    // 1. 先把旧快捷键注销掉(如果存在)
    if let Some(old) = current.take() {
        if let Err(e) = app.global_shortcut().unregister(old) {
            log::warn!("[shortcut] unregister old failed: {:?}", e);
        } else {
            log::info!("[shortcut] unregistered old shortcut");
        }
    }

    // 2. 注册新的(如果不是 disabled)
    if let Some(sc) = new_shortcut {
        app.global_shortcut()
            .register(sc)
            .map_err(|e| {
                let msg = format!("{:?}", e);
                log::error!("[shortcut] register '{}' failed: {}", normalized, msg);
                msg
            })?;
        *current = Some(sc);
        log::info!("[shortcut] registered: {}", normalized);
        Ok(normalized.to_string())
    } else {
        log::info!("[shortcut] cleared");
        Ok("disabled".into())
    }
}

#[tauri::command]
fn greet(name: &str) -> String {
    format!("Hello, {}! 欢迎使用 418前端", name)
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let mut builder = tauri::Builder::default()
        .plugin(
            tauri_plugin_global_shortcut::Builder::new()
                .with_handler(|app, _shortcut, event| {
                    if event.state == ShortcutState::Pressed {
                        handle_shortcut_press(app);
                    }
                })
                .build(),
        );

    builder = builder
        .setup(|app| {
            // 启动时默认显示
            if let Some(window) = app.get_webview_window("main") {
                let _ = window.show();

                // Windows:把 OS 窗口本身设成圆角(DWM API)
                #[cfg(target_os = "windows")]
                {
                    if let Ok(hwnd) = window.hwnd() {
                        apply_windows_rounded_corners(hwnd.0 as *mut std::ffi::c_void);
                        log::info!("[window] applied DWMWCP_ROUND to main window");
                    }
                }
            }

            // 全局快捷键默认 **不** 在 Rust 侧启动注册 —— 由前端 App.vue 挂载后读
            // localStorage 决定要 enable 哪个 preset,保证前端可以无副作用地切换。

            // 监听窗口关闭事件:关闭即隐藏(不退程序,支持快捷键恢复)
            let app_handle_for_event = app.handle().clone();
            if let Some(window) = app.get_webview_window("main") {
                window.on_window_event(move |event| {
                    if let WindowEvent::CloseRequested { api, .. } = event {
                        if let Some(w) = app_handle_for_event.get_webview_window("main") {
                            // 阻止默认关闭,改为隐藏
                            api.prevent_close();
                            let _ = w.hide();
                            log::info!("[window event] CloseRequested → hide");
                        }
                    }
                });
            }

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            greet,
            set_window_shortcut
        ]);

    builder
        .run(tauri::generate_context!())
        .expect("启动 Tauri 应用失败");
}
