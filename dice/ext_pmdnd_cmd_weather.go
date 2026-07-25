package dice

import (
	"fmt"
)

// ----- 天气命令 -----
func getWeatherHelp() string {
	return "PMDnD 天气管理:\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		".weather                查看当前天气\n" +
		".weather 大晴天          设置天气为大晴天（火系×1.5，水系×0.5）\n" +
		".weather 下雨            设置为下雨（水系×1.5，火系×0.5）\n" +
		".weather 沙暴            设置为沙暴（岩石系×1.5）\n" +
		".weather 冰雹            设置为冰雹（冰系×1.5）\n" +
		".weather 雪景            设置为雪景（冰系×1.5）\n" +
		".weather clear          清除天气\n" +
		"\n" +
		"💡 天气会持续影响战斗，直到被清除或切换\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

var cmdWeather = &CmdItemInfo{
	Name:      "weather",
	ShortHelp: ".weather                查看当前天气\n.weather 大晴天          设置天气",
	Help:      getWeatherHelp(),
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		sub := cmdArgs.GetArgN(1)
		state := loadBattleState(ctx)

		switch sub {
		case "help":
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}

		case "", "show":
			if state.Weather == "" {
				ReplyToSender(ctx, msg, "当前没有特殊天气")
			} else {
				ReplyToSender(ctx, msg, fmt.Sprintf("☀️ 当前天气: %s", state.Weather))
			}

		case "clear":
			state.Weather = ""
			saveBattleState(ctx, state)
			ReplyToSender(ctx, msg, "天气已清除")

		default:
			// 检查是否是有效的天气名称
			validWeathers := map[string]bool{
				"大晴天": true, "sunny": true,
				"下雨": true, "rain": true,
				"沙暴": true, "sand": true,
				"冰雹": true, "hail": true,
				"雪景": true, "snow": true,
			}
			if !validWeathers[sub] {
				ReplyToSender(ctx, msg, "无效天气，可选: 大晴天, 下雨, 沙暴, 冰雹, 雪景")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			// 统一使用中文名存储
			weatherMap := map[string]string{
				"sunny": "大晴天", "rain": "下雨", "sand": "沙暴",
				"hail": "冰雹", "snow": "雪景",
			}
			if val, ok := weatherMap[sub]; ok {
				sub = val
			}
			state.Weather = sub
			saveBattleState(ctx, state)
			ReplyToSender(ctx, msg, fmt.Sprintf("☀️ 天气已设置为: %s", sub))
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}

// ----- 场地命令 -----
func getTerrainHelp() string {
	return "PMDnD 场地管理:\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		".terrain                查看当前场地\n" +
		".terrain 电气场地        设置为电气场地（电系×1.3）\n" +
		".terrain 青草场地        设置为青草场地（草系×1.3，治疗×1.3）\n" +
		".terrain 精神场地        设置为精神场地（超能力系×1.3）\n" +
		".terrain 薄雾场地        设置为薄雾场地（妖精系×1.3）\n" +
		".terrain 龙之场地        设置为龙之场地（龙系×1.3）\n" +
		".terrain 失序场地        设置为失序场地（多数属性×1.2）\n" +
		".terrain clear          清除场地\n" +
		"\n" +
		"💡 场地会持续影响战斗，直到被清除或切换\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

var cmdTerrain = &CmdItemInfo{
	Name:      "terrain",
	ShortHelp: ".terrain                查看当前场地\n.terrain 电气场地          设置场地",
	Help:      getTerrainHelp(),
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		sub := cmdArgs.GetArgN(1)
		state := loadBattleState(ctx)

		switch sub {
		case "help":
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}

		case "", "show":
			if state.Terrain == "" {
				ReplyToSender(ctx, msg, "当前没有特殊场地")
			} else {
				ReplyToSender(ctx, msg, fmt.Sprintf("🌿 当前场地: %s", state.Terrain))
			}

		case "clear":
			state.Terrain = ""
			saveBattleState(ctx, state)
			ReplyToSender(ctx, msg, "场地已清除")

		default:
			validTerrains := map[string]bool{
				"电气场地": true, "electric": true,
				"青草场地": true, "grassy": true,
				"精神场地": true, "psychic": true,
				"薄雾场地": true, "misty": true,
				"龙之场地": true, "dragon": true,
				"失序场地": true, "chaos": true,
			}
			if !validTerrains[sub] {
				ReplyToSender(ctx, msg, "无效场地，可选: 电气场地, 青草场地, 精神场地, 薄雾场地, 龙之场地, 失序场地")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			terrainMap := map[string]string{
				"electric": "电气场地", "grassy": "青草场地",
				"psychic": "精神场地", "misty": "薄雾场地",
				"dragon": "龙之场地", "chaos": "失序场地",
			}
			if val, ok := terrainMap[sub]; ok {
				sub = val
			}
			state.Terrain = sub
			saveBattleState(ctx, state)
			ReplyToSender(ctx, msg, fmt.Sprintf("🌿 场地已设置为: %s", sub))
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
