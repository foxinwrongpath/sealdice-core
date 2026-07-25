package dice

import (
	"fmt"
	"strconv"
	"strings"

	ds "github.com/sealdice/dicescript"
)

// NPCData 存储 NPC 属性，键为 NPC 名称，值为属性 map
type NPCData map[string]map[string]interface{}

// loadNPCData 从群组变量中加载 NPC 数据
func loadNPCData(ctx *MsgContext) NPCData {
	attrs, _ := ctx.Dice.AttrsManager.LoadById(ctx.Group.GroupID)
	val := attrs.Load("$g_npc_data")
	if val == nil || val.TypeId != ds.VMTypeDict {
		return make(NPCData)
	}
	dd := val.MustReadDictData()
	result := make(NPCData)
	dd.Dict.Range(func(key string, value *ds.VMValue) bool {
		if value.TypeId == ds.VMTypeDict {
			subDD := value.MustReadDictData()
			subMap := make(map[string]interface{})
			subDD.Dict.Range(func(subKey string, subVal *ds.VMValue) bool {
				switch subVal.TypeId {
				case ds.VMTypeInt:
					iv, _ := subVal.ReadInt()
					subMap[subKey] = int(iv)
				case ds.VMTypeFloat:
					fv, _ := subVal.ReadFloat()
					subMap[subKey] = fv
				default:
					subMap[subKey] = subVal.ToString()
				}
				return true
			})
			result[key] = subMap
		}
		return true
	})
	return result
}

// saveNPCData 保存 NPC 数据到群组变量
func saveNPCData(ctx *MsgContext, data NPCData) {
	attrs, _ := ctx.Dice.AttrsManager.LoadById(ctx.Group.GroupID)
	dict := ds.NewDictValWithArrayMust()
	for name, props := range data {
		subDict := ds.NewDictValWithArrayMust()
		for k, v := range props {
			var val *ds.VMValue
			switch vv := v.(type) {
			case int:
				val = ds.NewIntVal(ds.IntType(vv))
			case float64:
				val = ds.NewFloatVal(vv)
			default:
				val = ds.NewStrVal(fmt.Sprintf("%v", vv))
			}
			(*ds.VMValue)(subDict).MustReadDictData().Dict.Store(k, val)
		}
		// 关键修复：dict 也是 *VMDictValue，需要转换为 *VMValue 再调用
		(*ds.VMValue)(dict).MustReadDictData().Dict.Store(name, (*ds.VMValue)(subDict))
	}
	attrs.Store("$g_npc_data", (*ds.VMValue)(dict))
}

// getNPCAttr 获取 NPC 的指定属性值（返回 int64，若不存在返回 0）
func getNPCAttr(ctx *MsgContext, name string, attr string) int64 {
	data := loadNPCData(ctx)
	if props, ok := data[name]; ok {
		if v, ok := props[attr]; ok {
			switch val := v.(type) {
			case int:
				return int64(val)
			case float64:
				return int64(val)
			default:
				if s, ok := v.(string); ok {
					if i, err := strconv.ParseInt(s, 10, 64); err == nil {
						return i
					}
				}
			}
		}
	}
	return 0
}

// getNPCStringAttr 获取 NPC 的指定属性值（返回字符串）
func getNPCStringAttr(ctx *MsgContext, name string, attr string) string {
	data := loadNPCData(ctx)
	if props, ok := data[name]; ok {
		if v, ok := props[attr]; ok {
			if s, ok := v.(string); ok {
				return s
			}
			if i, ok := v.(int); ok {
				return strconv.Itoa(i)
			}
			if f, ok := v.(float64); ok {
				return strconv.FormatFloat(f, 'f', -1, 64)
			}
		}
	}
	return ""
}

// cmdNPC .npc 命令
var cmdNPC = &CmdItemInfo{
	Name:      "npc",
	ShortHelp: ".npc set <名称> <属性1>:<值1> [<属性2>:<值2> ...]\n.npc show <名称>\n.npc list\n.npc del <名称>\n.npc clear",
	Help: "PMDnD NPC 管理:\n" +
		".npc set <名称> <属性1>:<值1> [<属性2>:<值2> ...] // 创建/更新NPC属性，例: .npc set 圈圈熊 patk:30 pdef:20 type_格斗:1\n" +
		".npc show <名称> // 查看NPC属性\n" +
		".npc list // 列出所有NPC名称\n" +
		".npc del <名称> // 删除某个NPC\n" +
		".npc clear // 清除所有NPC\n" +
		"支持属性: patk, pdef, satk, sdef, spd, type_*, stab_* 等",
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		sub := cmdArgs.GetArgN(1)
		switch sub {
		case "set":
			name := cmdArgs.GetArgN(2)
			if name == "" {
				ReplyToSender(ctx, msg, "请指定NPC名称: .npc set <名称> <属性1>:<值1> ...")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			data := loadNPCData(ctx)
			if _, ok := data[name]; !ok {
				data[name] = make(map[string]interface{})
			}
			props := data[name]
			args := cmdArgs.Args[3:]
			if len(args) == 0 {
				ReplyToSender(ctx, msg, "请指定至少一个属性: .npc set <名称> <属性1>:<值1> ...")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			var setItems []string
			for _, arg := range args {
				parts := strings.SplitN(arg, ":", 2)
				if len(parts) != 2 {
					ReplyToSender(ctx, msg, fmt.Sprintf("属性格式错误: %s，应为 属性:值", arg))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				key := strings.TrimSpace(parts[0])
				valStr := strings.TrimSpace(parts[1])
				if i, err := strconv.ParseInt(valStr, 10, 64); err == nil {
					props[key] = int(i)
				} else if f, err := strconv.ParseFloat(valStr, 64); err == nil {
					props[key] = f
				} else {
					props[key] = valStr
				}
				setItems = append(setItems, fmt.Sprintf("%s:%s", key, valStr))
			}
			saveNPCData(ctx, data)
			ReplyToSender(ctx, msg, fmt.Sprintf("NPC %s 属性已设置: %s", name, strings.Join(setItems, " ")))

		case "show":
			name := cmdArgs.GetArgN(2)
			if name == "" {
				ReplyToSender(ctx, msg, "请指定NPC名称: .npc show <名称>")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			data := loadNPCData(ctx)
			props, ok := data[name]
			if !ok {
				ReplyToSender(ctx, msg, fmt.Sprintf("未找到NPC: %s", name))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			var lines []string
			lines = append(lines, fmt.Sprintf("NPC %s 属性:", name))
			for k, v := range props {
				lines = append(lines, fmt.Sprintf("  %s: %v", k, v))
			}
			ReplyToSender(ctx, msg, strings.Join(lines, "\n"))

		case "list":
			data := loadNPCData(ctx)
			if len(data) == 0 {
				ReplyToSender(ctx, msg, "当前没有定义任何NPC")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			var names []string
			for name := range data {
				names = append(names, name)
			}
			ReplyToSender(ctx, msg, fmt.Sprintf("已定义的NPC: %s", strings.Join(names, ", ")))

		case "del":
			name := cmdArgs.GetArgN(2)
			if name == "" {
				ReplyToSender(ctx, msg, "请指定NPC名称: .npc del <名称>")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			data := loadNPCData(ctx)
			if _, ok := data[name]; !ok {
				ReplyToSender(ctx, msg, fmt.Sprintf("未找到NPC: %s", name))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			delete(data, name)
			saveNPCData(ctx, data)
			ReplyToSender(ctx, msg, fmt.Sprintf("NPC %s 已删除", name))

		case "clear":
			data := loadNPCData(ctx)
			if len(data) == 0 {
				ReplyToSender(ctx, msg, "当前没有定义任何NPC")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			saveNPCData(ctx, make(NPCData))
			ReplyToSender(ctx, msg, "所有NPC数据已清除")

		case "help", "":
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}

		default:
			name := sub
			if name != "" {
				data := loadNPCData(ctx)
				props, ok := data[name]
				if !ok {
					ReplyToSender(ctx, msg, fmt.Sprintf("未找到NPC: %s，使用 .npc set 创建", name))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				var lines []string
				lines = append(lines, fmt.Sprintf("NPC %s 属性:", name))
				for k, v := range props {
					lines = append(lines, fmt.Sprintf("  %s: %v", k, v))
				}
				ReplyToSender(ctx, msg, strings.Join(lines, "\n"))
			} else {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
