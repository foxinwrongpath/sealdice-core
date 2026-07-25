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

// updateNPCHP 更新 NPC 的 HP（扣除伤害），返回新 HP、最大值和是否成功
func updateNPCHP(ctx *MsgContext, name string, damage int64) (newHp int64, maxHp int64, ok bool) {
	data := loadNPCData(ctx)
	props, exists := data[name]
	if !exists {
		return 0, 0, false
	}

	var curHp int64
	if v, ok := props["hp"]; ok {
		switch val := v.(type) {
		case int:
			curHp = int64(val)
		case float64:
			curHp = int64(val)
		default:
			return 0, 0, false
		}
	} else {
		return 0, 0, false
	}

	if v, ok := props["hpmax"]; ok {
		switch val := v.(type) {
		case int:
			maxHp = int64(val)
		case float64:
			maxHp = int64(val)
		default:
			maxHp = curHp
		}
	} else {
		maxHp = curHp
	}

	newHp = curHp - damage
	if newHp < 0 {
		newHp = 0
	}
	props["hp"] = int(newHp)
	saveNPCData(ctx, data)
	return newHp, maxHp, true
}

// getNPCHP 获取 NPC 的 HP 信息
func getNPCHP(ctx *MsgContext, name string) (curHp int64, maxHp int64, ok bool) {
	data := loadNPCData(ctx)
	props, exists := data[name]
	if !exists {
		return 0, 0, false
	}
	if v, ok := props["hp"]; ok {
		switch val := v.(type) {
		case int:
			curHp = int64(val)
		case float64:
			curHp = int64(val)
		}
	} else {
		return 0, 0, false
	}
	if v, ok := props["hpmax"]; ok {
		switch val := v.(type) {
		case int:
			maxHp = int64(val)
		case float64:
			maxHp = int64(val)
		}
	} else {
		maxHp = curHp
	}
	return curHp, maxHp, true
}

// cmdNPC .npc 命令
var cmdNPC = &CmdItemInfo{
	Name:      "npc",
	ShortHelp: ".npc set <名称> <属性1>:<值1> [<属性2>:<值2> ...]\n.npc show <名称>\n.npc list\n.npc del <名称>\n.npc clear",
	Help: "PMDnD NPC 管理:\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		"📌 为战斗中的宝可梦（敌人/盟友）设置数据，让 @名称 可被识别。\n" +
		"\n" +
		"📖 子命令:\n" +
		"  set <名称> <属性1>:<值1> ...   设置或更新角色数据\n" +
		"  show <名称>                    查看角色数据\n" +
		"  list                           列出所有已设置的角色\n" +
		"  del <名称>                     删除指定角色\n" +
		"  clear                          清除所有角色数据\n" +
		"\n" +
		"📝 可用属性:\n" +
		"  物攻 patk:数字   物防 pdef:数字   速度 spd:数字\n" +
		"  特攻 satk:数字   特防 sdef:数字\n" +
		"  生命 hp:数字     生命上限 hpmax:数字\n" +
		"  战斗等级 cr:数字   (默认30，影响伤害计算)\n" +
		"  属性类型 type_火:1  type_水:1  type_格斗:1  ……\n" +
		"\n" +
		"📝 常用示例:\n" +
		"  .npc set 圈圈熊 patk:30 pdef:20 hpmax:100 cr:30 type_格斗:1\n" +
		"  .npc set 大嘴蝠 patk:15 pdef:10 satk:25 sdef:12 hpmax:80\n" +
		"  .npc list\n" +
		"  .npc show 圈圈熊\n" +
		"\n" +
		"⚔️ 战斗中使用 (配合 .dmg):\n" +
		"  .dmg 格斗 80 物 @圈圈熊          # 攻击圈圈熊，读取其 pdef\n" +
		"  .dmg 格斗 80 物 @圈圈熊 @伊布    # 圈圈熊攻击伊布，读取其 patk 和 cr\n" +
		"  .move attack 撞击 @大嘴蝠        # 攻击大嘴蝠，读取其 pdef\n" +
		"\n" +
		"💡 设置 hpmax 后，攻击会自动扣血并显示血条:\n" +
		"  📊 HP: ████████░░ 80/100\n" +
		"\n" +
		"🗑️ 删除:\n" +
		"  .npc del 圈圈熊\n" +
		"  .npc clear                         # 清除所有\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━",
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		sub := cmdArgs.GetArgN(1)
		switch sub {
		case "set":
			parts := strings.Fields(cmdArgs.CleanArgs)
			if len(parts) < 3 {
				ReplyToSender(ctx, msg, "格式错误: .npc set <名称> <属性1>:<值1> ...")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			name := parts[1]
			attrParts := parts[2:]

			if len(attrParts) == 0 {
				ReplyToSender(ctx, msg, "请指定至少一个属性: .npc set <名称> <属性1>:<值1> ...")
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			data := loadNPCData(ctx)
			if _, ok := data[name]; !ok {
				data[name] = make(map[string]interface{})
			}
			props := data[name]

			var setItems []string
			for _, arg := range attrParts {
				kv := strings.SplitN(arg, ":", 2)
				if len(kv) != 2 {
					ReplyToSender(ctx, msg, fmt.Sprintf("属性格式错误: %s，应为 属性:值", arg))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				key := strings.TrimSpace(kv[0])
				valStr := strings.TrimSpace(kv[1])
				if i, err := strconv.ParseInt(valStr, 10, 64); err == nil {
					props[key] = int(i)
				} else if f, err := strconv.ParseFloat(valStr, 64); err == nil {
					props[key] = f
				} else {
					props[key] = valStr
				}
				setItems = append(setItems, fmt.Sprintf("%s:%s", key, valStr))
			}

			// 如果设置了 hpmax 但没有设置 hp，自动将 hp 设为 hpmax
			if _, hasHp := props["hp"]; !hasHp {
				if hpmax, ok := props["hpmax"]; ok {
					if v, ok := hpmax.(int); ok {
						props["hp"] = v
					}
				}
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
				// 友好显示 hp/hpmax
				if k == "hp" {
					if hpmax, ok := props["hpmax"]; ok {
						lines = append(lines, fmt.Sprintf("  HP: %d/%d", v, hpmax))
						continue
					}
				}
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
					if k == "hp" {
						if hpmax, ok := props["hpmax"]; ok {
							lines = append(lines, fmt.Sprintf("  HP: %d/%d", v, hpmax))
							continue
						}
					}
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
