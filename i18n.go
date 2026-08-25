package main

var ko = map[string]string{
	"app": "Simple Share (FTP/WebDAV)", "running": "서버 실행 중", "stopped": "서버 중지됨",
	"start": "서버 시작", "stop": "서버 중지", "settings": "설정...", "settingsManage": "설정 관리", "openRoot": "루트 폴더 열기",
	"firewall": "방화벽 상태 확인", "firewallChecking": "Windows 방화벽 상태를 확인하고 있습니다...", "firewallAdding": "관리자 권한 확인 후 방화벽 규칙을 추가합니다...",
	"update": "업데이트 확인", "backup": "설정 백업", "restore": "설정 복원", "reset": "설정 초기화",
	"autostart": "Windows 시작 시 자동 실행", "language": "언어 선택", "exit": "종료",
	"protocol": "프로토콜", "port": "포트", "root": "루트 폴더", "browse": "찾아보기...", "anonymous": "익명 접속 허용",
	"username": "사용자 이름", "password": "암호", "save": "저장", "cancel": "취소", "korean": "한국어", "english": "영어",
	"invalidRoot": "공유할 루트 폴더를 선택하세요.", "invalidPort": "포트는 1~65535 범위여야 합니다.",
	"needCredential": "익명 접속을 사용하지 않으면 사용자 이름과 암호가 필요합니다.",
}

var en = map[string]string{
	"app": "Simple Share (FTP/WebDAV)", "running": "Server running", "stopped": "Server stopped",
	"start": "Start server", "stop": "Stop server", "settings": "Settings...", "settingsManage": "Settings management", "openRoot": "Open root folder",
	"firewall": "Check firewall", "firewallChecking": "Checking Windows Firewall status...", "firewallAdding": "Waiting for administrator approval to add the firewall rule...",
	"update": "Check for updates", "backup": "Back up settings", "restore": "Restore settings", "reset": "Reset settings",
	"autostart": "Start with Windows", "language": "Language", "exit": "Exit",
	"protocol": "Protocol", "port": "Port", "root": "Root folder", "browse": "Browse...", "anonymous": "Allow anonymous access",
	"username": "Username", "password": "Password", "save": "Save", "cancel": "Cancel", "korean": "Korean", "english": "English",
	"invalidRoot": "Select a root folder to share.", "invalidPort": "Port must be between 1 and 65535.",
	"needCredential": "Username and password are required when anonymous access is disabled.",
}

func tr(lang, key string) string {
	if lang == "en" {
		if s, ok := en[key]; ok { return s }
	}
	if s, ok := ko[key]; ok { return s }
	return key
}
