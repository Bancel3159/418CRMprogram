// 始终不在 Windows 上打开额外控制台窗口(debug 和 release 都生效)
// 非 Windows 平台忽略此属性
#![cfg_attr(windows, windows_subsystem = "windows")]

fn main() {
    crm_desktop_lead_lib::run()
}