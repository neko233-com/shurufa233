package engine

import (
	"sort"
	"strings"
)

const associationWeightBase = 7600

var builtinAssociations = map[string][]Entry{
	"你好": {
		{Reading: "shijie", Text: "世界", Weight: associationWeightBase},
		{Reading: "ya", Text: "呀", Weight: associationWeightBase - 20},
		{Reading: "qingwen", Text: "请问", Weight: associationWeightBase - 40},
	},
	"中国": {
		{Reading: "renmin", Text: "人民", Weight: associationWeightBase},
		{Reading: "wenhua", Text: "文化", Weight: associationWeightBase - 20},
		{Reading: "shurufa", Text: "输入法", Weight: associationWeightBase - 40},
	},
	"微信": {
		{Reading: "shurufa", Text: "输入法", Weight: associationWeightBase},
		{Reading: "xiaoxi", Text: "消息", Weight: associationWeightBase - 20},
		{Reading: "liaotian", Text: "聊天", Weight: associationWeightBase - 40},
	},
	"输入法": {
		{Reading: "houxuan", Text: "候选", Weight: associationWeightBase},
		{Reading: "shezhi", Text: "设置", Weight: associationWeightBase - 20},
		{Reading: "ciku", Text: "词库", Weight: associationWeightBase - 40},
		{Reading: "pifu", Text: "皮肤", Weight: associationWeightBase - 60},
	},
	"收到": {
		{Reading: "le", Text: "了", Weight: associationWeightBase},
		{Reading: "xiexie", Text: "谢谢", Weight: associationWeightBase - 20},
		{Reading: "mashangchuli", Text: "马上处理", Weight: associationWeightBase - 40},
	},
	"谢谢": {
		{Reading: "ni", Text: "你", Weight: associationWeightBase},
		{Reading: "dajia", Text: "大家", Weight: associationWeightBase - 20},
		{Reading: "zhichi", Text: "支持", Weight: associationWeightBase - 40},
	},
	"马上": {
		{Reading: "dao", Text: "到", Weight: associationWeightBase},
		{Reading: "chuli", Text: "处理", Weight: associationWeightBase - 20},
		{Reading: "huifu", Text: "回复", Weight: associationWeightBase - 40},
	},
	"今天": {
		{Reading: "tianqi", Text: "天气", Weight: associationWeightBase},
		{Reading: "xiawu", Text: "下午", Weight: associationWeightBase - 20},
		{Reading: "wanshang", Text: "晚上", Weight: associationWeightBase - 40},
	},
	"明天": {
		{Reading: "shangwu", Text: "上午", Weight: associationWeightBase},
		{Reading: "xiawu", Text: "下午", Weight: associationWeightBase - 20},
		{Reading: "jian", Text: "见", Weight: associationWeightBase - 40},
	},
}

func (e *Engine) associationCandidatesLocked(context string, limit int) []Candidate {
	context = normalizeAssociationContext(context)
	if context == "" || !e.config.Associations {
		return nil
	}
	entries := e.associationEntriesLocked(context)
	if len(entries) == 0 {
		return nil
	}

	candidates := make([]Candidate, 0, len(entries))
	seen := map[string]bool{}
	for _, entry := range entries {
		entry.Reading = normalizeReading(entry.Reading)
		if entry.Text == "" {
			continue
		}
		displayText := convertScriptText(entry.Text, e.config.Script)
		if e.isRejectedLocked(entry.Reading, displayText) {
			continue
		}
		key := displayText + "\x00" + entry.Reading + "\x00" + entry.Kind + "\x00" + entry.Source
		if seen[key] {
			continue
		}
		seen[key] = true
		candidates = append(candidates, Candidate{
			Text:      displayText,
			Reading:   entry.Reading,
			Kind:      entry.Kind,
			Source:    entry.Source,
			Comment:   entry.Comment,
			Weight:    entry.Weight,
			UserScore: e.entryUserScoreLocked(entry),
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left := candidates[i].Weight + candidates[i].UserScore
		right := candidates[j].Weight + candidates[j].UserScore
		if left == right {
			return len([]rune(candidates[i].Text)) > len([]rune(candidates[j].Text))
		}
		return left > right
	})
	if limit <= 0 {
		limit = e.config.MaxCandidates
	}
	if limit <= 0 {
		limit = DefaultConfig().MaxCandidates
	}
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func (e *Engine) associationEntriesLocked(context string) []Entry {
	var out []Entry
	for trigger, entries := range builtinAssociations {
		if !strings.HasSuffix(context, trigger) {
			continue
		}
		for index, entry := range entries {
			if entry.Kind == "" {
				entry.Kind = "association"
			}
			if entry.Source == "" {
				entry.Source = "builtin-association"
			}
			if entry.Comment == "" {
				entry.Comment = "联想"
			}
			if entry.Weight <= 0 {
				entry.Weight = associationWeightBase - index*20
			}
			out = append(out, entry)
		}
	}
	return out
}

func normalizeAssociationContext(context string) string {
	context = strings.TrimSpace(context)
	if context == "" {
		return ""
	}
	runes := []rune(context)
	if len(runes) > 32 {
		context = string(runes[len(runes)-32:])
	}
	return context
}
