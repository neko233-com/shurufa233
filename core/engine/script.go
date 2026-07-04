package engine

import "strings"

var simplifiedToTraditionalRunes = map[rune]rune{
	'们': '們',
	'个': '個',
	'国': '國',
	'华': '華',
	'汉': '漢',
	'语': '語',
	'输': '輸',
	'简': '簡',
	'体': '體',
	'爱': '愛',
	'错': '錯',
	'词': '詞',
	'测': '測',
	'试': '試',
	'时': '時',
	'间': '間',
	'发': '發',
	'现': '現',
	'开': '開',
	'关': '關',
	'欢': '歡',
	'马': '馬',
	'吗': '嗎',
	'这': '這',
	'请': '請',
	'问': '問',
	'润': '潤',
	'选': '選',
	'学': '學',
	'习': '習',
	'库': '庫',
	'热': '熱',
	'导': '導',
	'复': '復',
	'删': '刪',
	'云': '雲',
	'后': '後',
	'台': '臺',
	'设': '設',
	'启': '啟',
	'动': '動',
	'双': '雙',
	'颜': '顏',
	'号': '號',
	'态': '態',
	'谢': '謝',
	'气': '氣',
	'电': '電',
	'脑': '腦',
	'网': '網',
	'页': '頁',
	'见': '見',
	'观': '觀',
	'买': '買',
	'卖': '賣',
	'键': '鍵',
}

func convertScriptText(text string, script string) string {
	if normalizeScript(script) != "traditional" || text == "" {
		return text
	}

	var builder strings.Builder
	builder.Grow(len(text))
	changed := false
	for _, char := range text {
		if mapped, ok := simplifiedToTraditionalRunes[char]; ok {
			builder.WriteRune(mapped)
			changed = true
			continue
		}
		builder.WriteRune(char)
	}
	if !changed {
		return text
	}
	return builder.String()
}
