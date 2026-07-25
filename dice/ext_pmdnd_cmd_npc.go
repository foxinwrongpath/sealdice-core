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

// ensureNPC 确保 NPC 存在，不存在则创建空数据
func ensureNPC(ctx *MsgContext, name string) {
	data := loadNPCData(ctx)
	if _, ok := data[name]; !ok {
		data[name] = make(map[string]interface{})
		saveNPCData(ctx, data)
	}
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

// ---------- NPC 招式管理 ----------

// loadNPCMoves 加载 NPC 招式数据
func loadNPCMoves(ctx *MsgContext, npcName string) map[string]map[string]interface{} {
	attrs, _ := ctx.Dice.AttrsManager.LoadById(ctx.Group.GroupID)
	key := "$npc_moves_" + npcName
	val := attrs.Load(key)
	if val == nil || val.TypeId != ds.VMTypeDict {
		return make(map[string]map[string]interface{})
	}
	dd := val.MustReadDictData()
	result := make(map[string]map[string]interface{})
	dd.Dict.Range(func(moveName string, moveVal *ds.VMValue) bool {
		if moveVal.TypeId == ds.VMTypeDict {
			moveDD := moveVal.MustReadDictData()
			moveData := make(map[string]interface{})
			moveDD.Dict.Range(func(k string, v *ds.VMValue) bool {
				switch v.TypeId {
				case ds.VMTypeInt:
					iv, _ := v.ReadInt()
					moveData[k] = int(iv)
				case ds.VMTypeFloat:
					fv, _ := v.ReadFloat()
					moveData[k] = fv
				default:
					moveData[k] = v.ToString()
				}
				return true
			})
			result[moveName] = moveData
		}
		return true
	})
	return result
}

// saveNPCMoves 保存 NPC 招式数据
func saveNPCMoves(ctx *MsgContext, npcName string, moves map[string]map[string]interface{}) {
	attrs, _ := ctx.Dice.AttrsManager.LoadById(ctx.Group.GroupID)
	key := "$npc_moves_" + npcName
	dict := ds.NewDictValWithArrayMust()
	for moveName, moveData := range moves {
		subDict := ds.NewDictValWithArrayMust()
		for k, v := range moveData {
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
		(*ds.VMValue)(dict).MustReadDictData().Dict.Store(moveName, (*ds.VMValue)(subDict))
	}
	attrs.Store(key, (*ds.VMValue)(dict))
}

// getNPCMove 获取 NPC 招式数据
func getNPCMove(ctx *MsgContext, npcName string, moveName string) (map[string]interface{}, bool) {
	moves := loadNPCMoves(ctx, npcName)
	move, ok := moves[moveName]
	return move, ok
}

// ---------- NPC 攻击 ----------
func executeNPCAttack(ctx *MsgContext, msg *Message, npcName string, moveName string, target string, advantage string, ctLimit int64, detailMode bool, debugMode bool) CmdExecuteResult {
	// 获取招式数据
	move, ok := getNPCMove(ctx, npcName, moveName)
	if !ok {
		ReplyToSender(ctx, msg, fmt.Sprintf("NPC %s 没有招式 %s", npcName, moveName))
		return CmdExecuteResult{Matched: true, Solved: true}
	}

	// 读取招式属性
	power := int64(10)
	if v, ok := move["power"]; ok {
		switch val := v.(type) {
		case int:
			power = int64(val)
		case float64:
			power = int64(val)
		}
	}
	elemType := "一般"
	if v, ok := move["type"]; ok {
		elemType = fmt.Sprintf("%v", v)
	}
	category := "物"
	if v, ok := move["category"]; ok {
		category = fmt.Sprintf("%v", v)
	}
	hitsStr := "1"
	if v, ok := move["hits"]; ok {
		hitsStr = fmt.Sprintf("%v", v)
	}
	ring := int64(1)
	if v, ok := move["ring"]; ok {
		switch val := v.(type) {
		case int:
			ring = int64(val)
		case float64:
			ring = int64(val)
		}
	}

	// 检查 NPC 是否存在
	data := loadNPCData(ctx)
	if _, exists := data[npcName]; !exists {
		ReplyToSender(ctx, msg, fmt.Sprintf("NPC %s 不存在，请先用 .npc new %s 创建", npcName, npcName))
		return CmdExecuteResult{Matched: true, Solved: true}
	}

	// 目标处理：如果未指定目标，使用先攻列表第一个非NPC单位
	if target == "" {
		riList := (RIList{}).LoadByCurGroup(ctx)
		for _, item := range riList {
			if item.name != npcName {
				target = item.name
				break
			}
		}
		if target == "" {
			target = ctx.Player.Name
		}
	}

	// 设置攻击者上下文（使用 NPC 数据）
	attacker := npcName

	// 判断是否为特殊攻击
	isSpecial := category == "特" || category == "特殊"

	// 处理多段攻击
	if hitsStr != "" && hitsStr != "1" {
		hitCount := parseHitCount(ctx, hitsStr)
		if hitCount <= 0 {
			hitCount = 1
		}
		if hitCount > 10 {
			hitCount = 10
		}

		var hitDetails []string
		var totalDmg int64
		var critCount int
		var hitCountActual int

		for i := 0; i < hitCount; i++ {
			result, errMsg := calculateDamage(ctx, power, elemType, isSpecial, advantage, ctLimit, attacker, target)
			if errMsg != "" {
				ReplyToSender(ctx, msg, errMsg)
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if result.Hit && result.FinalDmg > 0 {
				hitCountActual++
				totalDmg += result.FinalDmg
			}
			if result.Crit {
				critCount++
			}

			pctDisplay := fmt.Sprintf("%.0f%%", result.RollPct*100)
			detail := fmt.Sprintf("  #%d: d20=%d", i+1, result.D20)
			if result.Hit && result.FinalDmg > 0 {
				if debugMode {
					detail += fmt.Sprintf(" → %s → 伤害 %d", pctDisplay, result.FinalDmg)
				} else if detailMode {
					detail += fmt.Sprintf(" → %s → 伤害 %d", pctDisplay, result.FinalDmg)
				} else {
					detail += fmt.Sprintf(" → 伤害 %d", result.FinalDmg)
				}
			} else {
				detail += " → 未命中"
			}
			if result.Crit {
				detail += " 💥暴击"
			}
			hitDetails = append(hitDetails, detail)
		}

		var lines []string
		lines = append(lines, fmt.Sprintf("🎲 %s 使用 %s 攻击 %d 次！", npcName, moveName, hitCount))
		for _, detail := range hitDetails {
			lines = append(lines, detail)
		}
		lines = append(lines, fmt.Sprintf("  总伤害: %d 点！", totalDmg))

		// 战斗演说
		flavorLines := []string{}
		flavorLines = append(flavorLines, fmt.Sprintf("⚔️ %s 对 %s 使用了 %s！", npcName, target, moveName))
		flavorLines = append(flavorLines, fmt.Sprintf("  攻击 %d 次，命中 %d 次！", hitCount, hitCountActual))
		if critCount > 0 {
			flavorLines = append(flavorLines, fmt.Sprintf("  其中 %d 次暴击！", critCount))
		}

		if newHp, maxHp, ok := updateNPCHP(ctx, target, totalDmg); ok && maxHp > 0 {
			pct := newHp * 10 / maxHp
			if pct > 10 {
				pct = 10
			}
			if pct < 0 {
				pct = 0
			}
			bar := strings.Repeat("█", int(pct)) + strings.Repeat("░", 10-int(pct))
			flavorLines = append(flavorLines, fmt.Sprintf("  📊 HP: %s %d/%d", bar, newHp, maxHp))
		} else if target == ctx.Player.Name {
			if hpMax, exists := VarGetValueInt64(ctx, "hpmax"); exists && hpMax > 0 {
				curHp, _ := VarGetValueInt64(ctx, "hp")
				newHp := curHp - totalDmg
				if newHp < 0 {
					newHp = 0
				}
				VarSetValueInt64(ctx, "hp", newHp)
				pct := newHp * 10 / hpMax
				if pct > 10 {
					pct = 10
				}
				if pct < 0 {
					pct = 0
				}
				bar := strings.Repeat("█", int(pct)) + strings.Repeat("░", 10-int(pct))
				flavorLines = append(flavorLines, fmt.Sprintf("  📊 HP: %s %d/%d", bar, newHp, hpMax))
			}
		}

		fullText := strings.Join(lines, "\n") + "\n" + strings.Join(flavorLines, "\n")
		ReplyToSender(ctx, msg, fullText)
		return CmdExecuteResult{Matched: true, Solved: true}
	}

	// 单次攻击
	result, errMsg := calculateDamage(ctx, power, elemType, isSpecial, advantage, ctLimit, attacker, target)
	if errMsg != "" {
		ReplyToSender(ctx, msg, errMsg)
		return CmdExecuteResult{Matched: true, Solved: true}
	}

	var lines []string
	pctDisplay := fmt.Sprintf("%.0f%%", result.RollPct*100)

	// 1. 骰子行
	diceLine := fmt.Sprintf("🎲 d20=%d", result.D20)
	if result.Hit && result.RollPct > 0 {
		diceLine += fmt.Sprintf(" → %s", pctDisplay)
	}
	if result.Crit {
		diceLine += " 💥暴击"
	}
	if result.CritText == "【大失败】" {
		diceLine += " 💀大失败"
	}
	lines = append(lines, diceLine)

	// 2. 详细计算
	if debugMode {
		lines = append(lines, fmt.Sprintf("📐 [计算详情]"))
		lines = append(lines, fmt.Sprintf("  攻击者: %s  |  防御者: %s", npcName, target))
		lines = append(lines, fmt.Sprintf("  招式: %s  |  环位: %d", moveName, ring))
		lines = append(lines, fmt.Sprintf("  威力: %d  |  战斗等级: %d  |  攻击值: %d  |  防御值: %d", power, result.BattleLv, result.AtkVal, result.DefVal))
		if result.Hit {
			lines = append(lines, fmt.Sprintf("  基础: %d × %d × %d × %s ÷ (100 × %d) = %d", power, result.BattleLv, result.AtkVal, pctDisplay, result.DefVal, result.BaseDmg))
			if result.StabMul != 1.0 || result.TypeMod != 0 {
				factor := (2.0 + result.TypeMod) / 2.0
				if factor < 0.25 {
					factor = 0.25
				}
				if result.StabMul != 1.0 && result.TypeMod != 0 {
					lines = append(lines, fmt.Sprintf("  STAB: x%.2f  |  克制: x%.2f", result.StabMul, factor))
				} else if result.StabMul != 1.0 {
					lines = append(lines, fmt.Sprintf("  STAB: x%.2f", result.StabMul))
				} else if result.TypeMod != 0 {
					lines = append(lines, fmt.Sprintf("  克制: x%.2f", factor))
				}
			}
			lines = append(lines, fmt.Sprintf("  最终伤害: %d", result.FinalDmg))
		} else {
			lines = append(lines, "  结果: 未命中")
		}
	} else if detailMode && result.Hit {
		calcLine := fmt.Sprintf("📐 %d × %d级 × %d攻 × %s ÷ %d防", power, result.BattleLv, result.AtkVal, pctDisplay, result.DefVal)
		if result.StabMul != 1.0 || result.TypeMod != 0 {
			factor := (2.0 + result.TypeMod) / 2.0
			if factor < 0.25 {
				factor = 0.25
			}
			calcLine += fmt.Sprintf(" × %.2f修正", factor*result.StabMul)
		}
		calcLine += fmt.Sprintf(" = %d", result.FinalDmg)
		lines = append(lines, calcLine)
	}

	// 3. 战斗演说
	flavorLines := []string{}
	flavorLines = append(flavorLines, fmt.Sprintf("⚔️ %s 对 %s 使用了 %s！", npcName, target, moveName))

	if !result.Hit {
		flavorLines = append(flavorLines, "  但 是 没 有 命 中……")
	} else {
		if result.Crit {
			flavorLines = append(flavorLines, "  命中要害！")
		}
		if result.EffectText != "" {
			flavorLines = append(flavorLines, fmt.Sprintf("  %s", result.EffectText))
		}
		if result.FinalDmg == 0 {
			flavorLines = append(flavorLines, fmt.Sprintf("  对 %s 没有造成伤害……", target))
		} else {
			flavorLines = append(flavorLines, fmt.Sprintf("  %s 受到了 %d 点伤害！", target, result.FinalDmg))

			if newHp, maxHp, ok := updateNPCHP(ctx, target, result.FinalDmg); ok && maxHp > 0 {
				pct := newHp * 10 / maxHp
				if pct > 10 {
					pct = 10
				}
				if pct < 0 {
					pct = 0
				}
				bar := strings.Repeat("█", int(pct)) + strings.Repeat("░", 10-int(pct))
				flavorLines = append(flavorLines, fmt.Sprintf("  📊 HP: %s %d/%d", bar, newHp, maxHp))
			} else if target == ctx.Player.Name {
				if hpMax, exists := VarGetValueInt64(ctx, "hpmax"); exists && hpMax > 0 {
					curHp, _ := VarGetValueInt64(ctx, "hp")
					newHp := curHp - result.FinalDmg
					if newHp < 0 {
						newHp = 0
					}
					VarSetValueInt64(ctx, "hp", newHp)
					pct := newHp * 10 / hpMax
					if pct > 10 {
						pct = 10
					}
					if pct < 0 {
						pct = 0
					}
					bar := strings.Repeat("█", int(pct)) + strings.Repeat("░", 10-int(pct))
					flavorLines = append(flavorLines, fmt.Sprintf("  📊 HP: %s %d/%d", bar, newHp, hpMax))
				}
			}
		}
	}

	fullText := strings.Join(lines, "\n") + "\n" + strings.Join(flavorLines, "\n")
	ReplyToSender(ctx, msg, fullText)
	return CmdExecuteResult{Matched: true, Solved: true}
}

// ---------- cmdNPC ----------
func getNpcHelp() string {
	return "PMDnD NPC 管理:\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		"📌 为战斗中的宝可梦（敌人/盟友）设置数据。\n" +
		"\n" +
		"📖 基础操作:\n" +
		"  .npc new <名称>                      创建新NPC\n" +
		"  .npc <名称> st <属性1>:<值1> ...     设置属性（类似 .st）\n" +
		"  .npc <名称> show                     查看NPC属性\n" +
		"  .npc list                            列出所有NPC\n" +
		"  .npc del <名称>                      删除NPC\n" +
		"  .npc clear                           清除所有NPC\n" +
		"\n" +
		"📋 招式管理:\n" +
		"  .npc <名称> move add <招式名> <威力> <类型> <环位> <类别> [hits:2-5]\n" +
		"  .npc <名称> move list                列出NPC招式\n" +
		"  .npc <名称> move del <招式名>        删除NPC招式\n" +
		"  .npc <名称> move clear               清除NPC所有招式\n" +
		"\n" +
		"⚔️ 战斗:\n" +
		"  .npc <名称> move <招式名> [@目标] [优势/劣势] [暴击阈值] [detail] [debug]\n" +
		"  例: .npc 圈圈熊 move 臂锤 @伊布\n" +
		"\n" +
		"📝 可用属性:\n" +
		"  物攻 patk:数字   物防 pdef:数字   速度 spd:数字\n" +
		"  特攻 satk:数字   特防 sdef:数字\n" +
		"  生命 hp:数字     生命上限 hpmax:数字\n" +
		"  战斗等级 cr:数字   (默认30，影响伤害计算)\n" +
		"  属性类型 type_火:1  type_水:1  type_格斗:1  ……\n" +
		"\n" +
		"📝 示例:\n" +
		"  .npc new 圈圈熊\n" +
		"  .npc 圈圈熊 st patk:30 pdef:20 hpmax:100 type_格斗:1\n" +
		"  .npc 圈圈熊 move add 臂锤 80 格斗 2 物\n" +
		"  .npc 圈圈熊 move 臂锤 @伊布\n" +
		"  .npc 圈圈熊 show\n" +
		"\n" +
		"💡 设置 hpmax 后，攻击会自动扣血并显示血条\n" +
		"💡 NPC 使用招式不消耗 PP，无需设置 pp\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

var cmdNPC = &CmdItemInfo{
	Name:          "npc",
	ShortHelp:     ".npc new <名称>\n.npc <名称> st <属性>:<值>\n.npc <名称> show\n.npc <名称> move <招式名> [@目标]",
	Help:          getNpcHelp(),
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		// cmdArgs.CleanArgs 不包含命令名，直接分割参数
		parts := strings.Fields(cmdArgs.CleanArgs)
		if len(parts) == 0 {
			ReplyToSender(ctx, msg, getNpcHelp())
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		arg1 := parts[0]

		// ============================================
		// 顶层命令: new, list, del, clear, help
		// ============================================
		switch arg1 {
		case "help":
			ReplyToSender(ctx, msg, getNpcHelp())
			return CmdExecuteResult{Matched: true, Solved: true}

		case "new":
			if len(parts) < 2 {
				ReplyToSender(ctx, msg, "用法: .npc new <名称>")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			name := parts[1]
			ensureNPC(ctx, name)
			ReplyToSender(ctx, msg, fmt.Sprintf("已创建NPC: %s", name))
			return CmdExecuteResult{Matched: true, Solved: true}

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
			return CmdExecuteResult{Matched: true, Solved: true}

		case "del":
			if len(parts) < 2 {
				ReplyToSender(ctx, msg, "用法: .npc del <名称>")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			name := parts[1]
			data := loadNPCData(ctx)
			if _, ok := data[name]; !ok {
				ReplyToSender(ctx, msg, fmt.Sprintf("未找到NPC: %s", name))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			delete(data, name)
			saveNPCData(ctx, data)
			// 同时删除招式数据
			attrs, _ := ctx.Dice.AttrsManager.LoadById(ctx.Group.GroupID)
			attrs.Delete("$npc_moves_" + name)
			ReplyToSender(ctx, msg, fmt.Sprintf("NPC %s 已删除", name))
			return CmdExecuteResult{Matched: true, Solved: true}

		case "clear":
			data := loadNPCData(ctx)
			if len(data) == 0 {
				ReplyToSender(ctx, msg, "当前没有定义任何NPC")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			saveNPCData(ctx, make(NPCData))
			// 清除所有NPC的招式数据
			attrs, _ := ctx.Dice.AttrsManager.LoadById(ctx.Group.GroupID)
			attrs.Range(func(key string, _ *ds.VMValue) bool {
				if strings.HasPrefix(key, "$npc_moves_") {
					attrs.Delete(key)
				}
				return true
			})
			ReplyToSender(ctx, msg, "已清除所有NPC数据")
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		// ============================================
		// 有名称的操作: .npc <名称> <子命令> ...
		// ============================================
		if len(parts) < 2 {
			ReplyToSender(ctx, msg, fmt.Sprintf("缺少子命令，可用: st, show, move"))
			return CmdExecuteResult{Matched: true, Solved: true}
		}
		npcName := parts[0]
		sub := parts[1]

		// 检查 NPC 是否存在
		data := loadNPCData(ctx)
		if _, ok := data[npcName]; !ok {
			ReplyToSender(ctx, msg, fmt.Sprintf("NPC %s 不存在，请先用 .npc new %s 创建", npcName, npcName))
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		// ============================================
		// .npc <名称> st <属性1>:<值1> ...
		// ============================================
		if sub == "st" || sub == "属性" {
			if len(parts) < 3 {
				ReplyToSender(ctx, msg, "请指定属性: .npc <名称> st <属性1>:<值1> ...")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			props := data[npcName]
			args := parts[2:] // 属性对列表

			var setItems []string
			for _, arg := range args {
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

			// 如果设置了 hpmax 但没有 hp，自动同步
			if _, hasHp := props["hp"]; !hasHp {
				if hpmax, ok := props["hpmax"]; ok {
					if v, ok := hpmax.(int); ok {
						props["hp"] = v
					}
				}
			}

			saveNPCData(ctx, data)
			ReplyToSender(ctx, msg, fmt.Sprintf("NPC %s 属性已更新: %s", npcName, strings.Join(setItems, " ")))
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		// ============================================
		// .npc <名称> show
		// ============================================
		if sub == "show" {
			props := data[npcName]
			var lines []string
			lines = append(lines, fmt.Sprintf("NPC %s 属性:", npcName))
			// 按固定顺序显示
			order := []string{"patk", "pdef", "satk", "sdef", "spd", "hp", "hpmax", "cr"}
			for _, key := range order {
				if v, ok := props[key]; ok {
					if key == "hp" {
						if hpmax, ok := props["hpmax"]; ok {
							lines = append(lines, fmt.Sprintf("  HP: %v/%v", v, hpmax))
							continue
						}
					}
					lines = append(lines, fmt.Sprintf("  %s: %v", key, v))
				}
			}
			// 显示其他属性
			for k, v := range props {
				skip := false
				for _, key := range order {
					if k == key {
						skip = true
						break
					}
				}
				if k == "hp" {
					skip = true
				}
				if !skip {
					lines = append(lines, fmt.Sprintf("  %s: %v", k, v))
				}
			}
			ReplyToSender(ctx, msg, strings.Join(lines, "\n"))
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		// ============================================
		// .npc <名称> move ...
		// ============================================
		if sub == "move" {
			if len(parts) < 3 {
				ReplyToSender(ctx, msg, "缺少 move 子命令: add/list/del/clear/招式名")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			moveSub := parts[2]

			switch moveSub {
			case "add":
				if len(parts) < 8 { // 需要：名称 move add 招式名 威力 类型 环位 类别 (7个)
					ReplyToSender(ctx, msg, "用法: .npc <名称> move add <招式名> <威力> <类型> <环位> <类别> [hits:2-5]")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				moveName := parts[3]
				power, err := strconv.ParseInt(parts[4], 10, 64)
				if err != nil {
					ReplyToSender(ctx, msg, "威力必须是数字")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				elemType := parts[5]
				ring, err := strconv.ParseInt(parts[6], 10, 64)
				if err != nil {
					ReplyToSender(ctx, msg, "环位必须是数字")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				category := parts[7]
				if category == "" {
					category = "物"
				}
				switch category {
				case "物理", "p", "physical":
					category = "物"
				case "特殊", "s", "special":
					category = "特"
				case "heal", "治疗":
					category = "治疗"
				case "buff", "强化":
					category = "强化"
				}

				// 解析 hits（从剩余参数中查找）
				hitsStr := "1"
				for i := 8; i < len(parts); i++ {
					if strings.HasPrefix(parts[i], "hits:") {
						hitsStr = strings.TrimPrefix(parts[i], "hits:")
						break
					}
				}

				moves := loadNPCMoves(ctx, npcName)
				moveData := map[string]interface{}{
					"power":    int(power),
					"type":     elemType,
					"ring":     int(ring),
					"category": category,
					"hits":     hitsStr,
				}
				moves[moveName] = moveData
				saveNPCMoves(ctx, npcName, moves)

				reply := fmt.Sprintf("NPC %s 添加招式: %s 威力%d 类型%s %d环 %s", npcName, moveName, power, elemType, ring, category)
				if hitsStr != "1" {
					reply += fmt.Sprintf(" 多段攻击: %s", hitsStr)
				}
				ReplyToSender(ctx, msg, reply)

			case "list":
				moves := loadNPCMoves(ctx, npcName)
				if len(moves) == 0 {
					ReplyToSender(ctx, msg, fmt.Sprintf("NPC %s 没有招式", npcName))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				var items []string
				for moveName, moveData := range moves {
					power := moveData["power"]
					elemType := moveData["type"]
					ring := moveData["ring"]
					category := moveData["category"]
					hits := "1"
					if v, ok := moveData["hits"]; ok {
						hits = fmt.Sprintf("%v", v)
					}
					line := fmt.Sprintf("%s 威力%v %v %v环 %v", moveName, power, elemType, ring, category)
					if hits != "1" {
						line += fmt.Sprintf(" {多段攻击:%s}", hits)
					}
					items = append(items, line)
				}
				ReplyToSender(ctx, msg, fmt.Sprintf("NPC %s 的招式:\n%s", npcName, strings.Join(items, "\n")))

			case "del":
				if len(parts) < 4 {
					ReplyToSender(ctx, msg, "用法: .npc <名称> move del <招式名>")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				moveName := parts[3]
				moves := loadNPCMoves(ctx, npcName)
				if _, ok := moves[moveName]; !ok {
					ReplyToSender(ctx, msg, fmt.Sprintf("NPC %s 没有招式 %s", npcName, moveName))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				delete(moves, moveName)
				saveNPCMoves(ctx, npcName, moves)
				ReplyToSender(ctx, msg, fmt.Sprintf("NPC %s 已删除招式: %s", npcName, moveName))

			case "clear":
				moves := loadNPCMoves(ctx, npcName)
				if len(moves) == 0 {
					ReplyToSender(ctx, msg, fmt.Sprintf("NPC %s 没有招式", npcName))
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				saveNPCMoves(ctx, npcName, make(map[string]map[string]interface{}))
				ReplyToSender(ctx, msg, fmt.Sprintf("NPC %s 已清除所有招式", npcName))

			case "help", "":
				ReplyToSender(ctx, msg, "用法: .npc <名称> move add/list/del/clear <参数>\n  例: .npc 圈圈熊 move add 臂锤 80 格斗 2 物")

			default:
				// ============================================
				// .npc <名称> move <招式名> [@目标] [优势/劣势] ...
				// ============================================
				moveName := moveSub
				if moveName == "" {
					ReplyToSender(ctx, msg, "请指定招式名: .npc <名称> move <招式名> [@目标]")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				// 解析参数
				target := ""
				advantage := ""
				ctLimit := int64(20)
				detailMode := false
				debugMode := false

				for i := 3; i < len(parts); i++ {
					p := parts[i]
					if strings.HasPrefix(p, "@") {
						target = strings.TrimPrefix(p, "@")
					} else if p == "优势" || p == "優勢" {
						advantage = "优势"
					} else if p == "劣势" || p == "劣勢" {
						advantage = "劣势"
					} else if p == "detail" || p == "-d" {
						detailMode = true
					} else if p == "debug" || p == "-D" {
						debugMode = true
					} else if n, e := strconv.ParseInt(p, 10, 64); e == nil && n >= 2 && n <= 20 {
						ctLimit = n
					}
				}

				return executeNPCAttack(ctx, msg, npcName, moveName, target, advantage, ctLimit, detailMode, debugMode)
			}
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		// ============================================
		// 未知子命令
		// ============================================
		ReplyToSender(ctx, msg, fmt.Sprintf("未知子命令: %s\n可用: st, show, move", sub))
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
