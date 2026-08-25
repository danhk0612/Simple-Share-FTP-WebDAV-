package main

var ko = map[string]string{
	"app": "Simple Share (FTP/WebDAV)", "running": "서버 실행 중", "stopped": "서버 중지됨",
	"start": "서버 시작", "stop": "서버 중지", "settings": "설정...", "openRoot": "루트 폴더 열기",
	"firewall": "방화벽 상태 확인", "update": "업데이트 확인", "backup": "설정 백업", "restore": "설정 복원",
	"reset": "설정 초기화", "autostart": "Windows 시작 시 자동 실행", "language": "English로 전환", "exit": "종료",
	"protocol": "프로토콜", "port": "포트", "root": "루트 폴더", "browse": "찾아보기...", "anonymous": "익명 접속 허용",
	"username": "사용자 이름", "password": "암호", "save": "저장", "cancel": "취소", "korean": "한국어", "english": "English",
	"invalidRoot": "공유할 루트 폴더를 선택하세요.", "invalidPort": "포트는 1~65535 범위여야 합니다.",
	"needCredential": "익명 접속을 사용하지 않으면 사용자 이름과 암호가 필요합니다.",
}

var en = map[string]string{
	"app": "Simple Share (FTP/WebDAV)", "running": "Server running", "stopped": "Server stopped",
	"start": "Start server", "stop": "Stop server", "settings": "Settings...", "openRoot": "Open root folder",
	"firewall": "Check firewall", "update": "Check for updates", "backup": "Back up settings", "restore": "Restore settings",
	"reset": "Reset settings", "autostart": "Start with Windows", "language": "한국어로 전환", "exit": "Exit",
	"protocol": "Protocol", "port": "Port", "root": "Root folder", "browse": "Browse...", "anonymous": "Allow anonymous access",
	"username": "Username", "password": "Password", "save": "Save", "cancel": "Cancel", "korean": "한국어", "english": "English",
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
