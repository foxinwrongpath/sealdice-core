package dice

import (
	"fmt"
	"strings"
)

var pmdndTypeChart = map[string]map[string]float64{
	"一般":  {},
	"力场":  {"钢": -0.5},
	"格斗":  {"一般": 1, "钢": 1, "岩石": 1, "恶": 2, "飞行": -1, "虫": -1, "妖精": -1},
	"飞行":  {"格斗": 1, "草": 1, "虫": 1, "钢": -1, "电": -1, "岩石": -1},
	"毒":   {"草": 1, "妖精": 1, "地面": -1, "岩石": -1, "幽灵": -1, "钢": -2},
	"酸":   {"钢": 2, "草": 1, "冰": 1, "岩石": -1, "水": -1},
	"地面":  {"毒": 1, "钢": 1, "火": 1, "岩石": 1, "电": 2, "草": -1, "虫": -1, "飞行": -2},
	"岩石":  {"飞行": 1, "火": 1, "虫": 1, "冰": 1, "格斗": -1, "地面": -1, "钢": -1},
	"虫":   {"草": 1, "恶": 1, "超能力": 1, "格斗": -1, "火": -1, "飞行": -1, "妖精": -1, "幽灵": -1, "钢": -1},
	"幽灵":  {"超能力": 1, "幽灵": 1, "一般": -2, "恶": -1},
	"钢":   {"岩石": 1, "冰": 1, "妖精": 1, "钢": -1, "火": -1, "水": -1, "电": -1},
	"火":   {"草": 1, "钢": 1, "冰": 1, "虫": 1, "火": -1, "水": -1, "岩石": -1, "地面": -1},
	"水":   {"火": 1, "地面": 1, "岩石": 1, "水": -1, "草": -1, "电": -2},
	"草":   {"水": 1, "地面": 1, "岩石": 1, "火": -1, "草": -1, "虫": -1, "飞行": -1, "毒": -1, "钢": -1, "冰": -1},
	"电":   {"飞行": 1, "水": 1, "草": -1, "地面": -2, "电": -1, "龙": -1},
	"超能力": {"格斗": 1, "毒": 1, "钢": -1, "超能力": -1, "恶": -2},
	"冰":   {"草": 1, "地面": 1, "飞行": 1, "龙": 1, "火": -1, "水": -1, "冰": -1, "钢": -1},
	"龙":   {"龙": 1, "钢": -1, "妖精": -2},
	"恶":   {"幽灵": 1, "超能力": 1, "格斗": -1, "恶": -1, "妖精": -1},
	"妖精":  {"格斗": 1, "龙": 1, "恶": 1, "火": -1, "毒": -1, "钢": -1},
	"光耀":  {"恶": 1, "幽灵": 1, "黯蚀": 2, "光耀": -1, "草": -1},
	"黯蚀":  {"超能力": 1, "妖精": 1, "光耀": 2, "格斗": -1, "恶": -1, "黯蚀": -1},
}

var pmdndTypeNames = []string{
	"一般", "力场", "格斗", "飞行", "毒", "酸", "地面", "岩石", "虫", "幽灵", "钢",
	"火", "水", "草", "电", "超能力", "冰", "龙", "恶", "妖精", "光耀", "黯蚀",
}

var pmdndTypeAliases = map[string]string{
	"normal": "一般", "force": "力场", "fighting": "格斗", "flying": "飞行",
	"poison": "毒", "acid": "酸", "ground": "地面", "rock": "岩石",
	"bug": "虫", "ghost": "幽灵", "steel": "钢", "fire": "火",
	"water": "水", "grass": "草", "electric": "电", "psychic": "超能力",
	"ice": "冰", "dragon": "龙", "dark": "恶", "fairy": "妖精",
	"radiant": "光耀", "necrotic": "黯蚀",
}

func getPMDnDType(name string) string {
	if alias, ok := pmdndTypeAliases[strings.ToLower(name)]; ok {
		return alias
	}
	return name
}

// getTypeEffectivenessText 返回属性克制的描述文本
// 在 PMDnD 中，克制是修正值（正负整数），不是直接的倍率
func getTypeEffectivenessText(mod float64) string {
	// 先判断修正值类型
	switch {
	case mod >= 2:
		return fmt.Sprintf("修正 +%.0f (双倍易伤, %.1fx)", mod, (2.0+mod)/2.0)
	case mod >= 1:
		return fmt.Sprintf("修正 +%.0f (易伤, %.1fx)", mod, (2.0+mod)/2.0)
	case mod > 0:
		return fmt.Sprintf("修正 +%.1f (易伤, %.1fx)", mod, (2.0+mod)/2.0)
	case mod == 0:
		return "修正 0 (正常, 1.0x)"
	case mod > -1:
		return fmt.Sprintf("修正 %.1f (抗性, %.2fx)", mod, 2.0/(2.0-mod))
	case mod > -2:
		return fmt.Sprintf("修正 %.0f (抗性, %.2fx)", mod, 2.0/(2.0-mod))
	case mod >= -3:
		return fmt.Sprintf("修正 %.0f (强抗性, %.2fx)", mod, 2.0/(2.0-mod))
	default:
		return fmt.Sprintf("修正 %.0f (免疫级抗性, %.2fx)", mod, 2.0/(2.0-mod))
	}
}

// getTypeModifierText 简化版，只返回修正值说明
func getTypeModifierText(mod float64) string {
	switch {
	case mod >= 1:
		return fmt.Sprintf("伤害修正 +%.0f", mod)
	case mod <= -1:
		return fmt.Sprintf("伤害修正 %.0f", mod)
	default:
		return "伤害修正 0"
	}
}
