package dice

import (
	"fmt"
	"strconv"
	"strings"

	ds "github.com/sealdice/dicescript"
)

var cmdMove = &CmdItemInfo{
	Name:      "move",
	ShortHelp: ".move // 查看招式列表\n.move add <名称> <威力> <类型> <环位> <物/特> // 添加招式\n.move add 火花 40 火 1 特\n.move del <名称> // 删除招式\n.move use <名称> // 仅消耗PP和环位，不计算伤害\n.move attack <名称> [@目标] [优势/劣势] [暴击阈值] // 攻击！自动计算伤害\n💡 NPC使用: .npc set <名称> <属性>:<值> 设置NPC属性后，@NPC自动读取\n.move pp <名称> +/-N // 修改招式PP\n.move clr // 清除所有招式",
	Help: "PMDnD 招式管理(.move):\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		".move                       查看招式列表\n" +
		".move add <名称> <威力> <类型> <环位> <物/特>  添加招式\n" +
		"  例: .move add 火花 40 火 1 特\n" +
		".move del <名称>            删除招式\n" +
		".move use <名称>            消耗PP和环位（不计算伤害）\n" +
		".move attack <名称> [@目标] [优势/劣势] [暴击阈值]  攻击！自动计算伤害\n" +
		"  例: .move attack 撞击 @圈圈熊 优势 19\n" +
		".move pp <名称> +/-N        修改招式PP\n" +
		".move clr                   清除所有招式\n" +
		"\n" +
		"👾 NPC使用: 先用 .npc set <名称> <属性>:<值> 设置NPC属性，\n" +
		"   然后 .move attack 招式名 @NPC 即可自动读取NPC的防御属性。\n" +
		"   例: .npc set 圈圈熊 pdef:20 sdef:15\n" +
		"       .move attack 撞击 @圈圈熊\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━",
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		cmdArgs.ChopPrefixToArgsWith("add", "del", "use", "attack", "atk", "pp", "rm")
		val := cmdArgs.GetArgN(1)
		mctx := GetCtxProxyFirst(ctx, cmdArgs)
		attrs, err := mctx.Dice.AttrsManager.LoadByCtx(mctx)
		if err != nil {
			ReplyToSender(mctx, msg, "加载属性失败: "+err.Error())
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		switch val {
		case "add":
			name := cmdArgs.GetArgN(2)
			powerStr := cmdArgs.GetArgN(3)
			elemType := cmdArgs.GetArgN(4)
			ringStr := cmdArgs.GetArgN(5)
			category := cmdArgs.GetArgN(6)

			if name == "" || powerStr == "" || elemType == "" || ringStr == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			power, err := strconv.ParseInt(powerStr, 10, 64)
			if err != nil {
				ReplyToSender(mctx, msg, "威力必须是数字")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			ring, err := strconv.ParseInt(ringStr, 10, 64)
			if err != nil {
				ReplyToSender(mctx, msg, "环位必须是数字")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if category == "" {
				category = "特"
			}

			key := "$move_" + name
			m := ds.ValueMap{}
			m.Store("name", ds.NewStrVal(name))
			m.Store("power", ds.NewIntVal(ds.IntType(power)))
			m.Store("type", ds.NewStrVal(elemType))
			m.Store("ring", ds.NewIntVal(ds.IntType(ring)))
			m.Store("category", ds.NewStrVal(category))
			m.Store("pp", ds.NewIntVal(5))
			m.Store("ppmax", ds.NewIntVal(5))
			attrs.Store(key, ds.NewDictVal(&m).V())
			ReplyToSender(mctx, msg, fmt.Sprintf("添加招式: %s 威力%d 类型%s %d环 %s PP:5/5",
				name, power, elemType, ring, category))

		case "del", "rm":
			name := cmdArgs.GetArgN(2)
			if name == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			key := "$move_" + name
			if _, exists := attrs.LoadX(key); exists {
				attrs.Delete(key)
				ReplyToSender(mctx, msg, fmt.Sprintf("已删除招式: %s", name))
			} else {
				ReplyToSender(mctx, msg, fmt.Sprintf("未找到招式: %s", name))
			}

		case "use":
			name := cmdArgs.GetArgN(2)
			if name == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			key := "$move_" + name
			val, exists := attrs.LoadX(key)
			if !exists || val.TypeId != ds.VMTypeDict {
				ReplyToSender(mctx, msg, fmt.Sprintf("未找到招式: %s", name))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			dd := val.MustReadDictData()
			ppV, _ := dd.Dict.Load("pp")
			ringV, _ := dd.Dict.Load("ring")
			pp, _ := ppV.ReadInt()
			ring, _ := ringV.ReadInt()

			if pp <= 0 {
				ReplyToSender(mctx, msg, fmt.Sprintf("招式%s PP不足，当前%d", name, pp))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			dd.Dict.Store("pp", ds.NewIntVal(pp-1))
			text2, ok := spellRingsGet(mctx, int64(ring), -1)
			if !ok {
				ReplyToSender(mctx, msg, fmt.Sprintf("环位不足: %s", text2))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			ppmaxV, _ := dd.Dict.Load("ppmax")
			ppmax, _ := ppmaxV.ReadInt()
			ReplyToSender(mctx, msg, fmt.Sprintf("%s使用了%s！PP: %d/%d (%s)",
				getPlayerNameTempFunc(mctx), name, pp-1, ppmax, text2))

		case "attack", "atk":
			name := cmdArgs.GetArgN(2)
			if name == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			key := "$move_" + name
			val, exists := attrs.LoadX(key)
			if !exists || val.TypeId != ds.VMTypeDict {
				ReplyToSender(mctx, msg, fmt.Sprintf("未找到招式: %s", name))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			dd := val.MustReadDictData()

			powerV, _ := dd.Dict.Load("power")
			power, _ := powerV.ReadInt()
			typeV, _ := dd.Dict.Load("type")
			elemType := typeV.ToString()
			catV, _ := dd.Dict.Load("category")
			category := catV.ToString()
			ppV, _ := dd.Dict.Load("pp")
			pp, _ := ppV.ReadInt()
			ppmaxV, _ := dd.Dict.Load("ppmax")
			ppmax, _ := ppmaxV.ReadInt()
			ringV, _ := dd.Dict.Load("ring")
			ring, _ := ringV.ReadInt()

			if pp <= 0 {
				ReplyToSender(mctx, msg, fmt.Sprintf("招式%s PP不足，当前%d", name, pp))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			isSpecial := category == "特" || category == "特殊"
			advantage := ""
			ctLimit := int64(20)
			attacker := mctx.Player.Name
			defender := ""

			rest := cmdArgs.CleanArgs
			parts := strings.Fields(rest)
			if len(parts) >= 2 {
				params := parts[2:]
				for _, p := range params {
					if strings.HasPrefix(p, "@") {
						if defender == "" {
							defender = strings.TrimPrefix(p, "@")
						}
					} else if p == "优势" || p == "優勢" {
						advantage = "优势"
					} else if p == "劣势" || p == "劣勢" {
						advantage = "劣势"
					} else if n, e := strconv.ParseInt(p, 10, 64); e == nil && n >= 2 && n <= 20 {
						ctLimit = n
					}
				}
			}
			if defender == "" {
				riList := (RIList{}).LoadByCurGroup(ctx)
				for _, item := range riList {
					if item.name != attacker {
						defender = item.name
						break
					}
				}
			}
			if defender == "" {
				defender = "目标"
			}

			// ----- 消耗资源（先消耗环位，再消耗 PP） -----
			ringText, ok := spellRingsGet(mctx, int64(ring), -1)
			if !ok {
				ReplyToSender(mctx, msg, fmt.Sprintf("环位不足: %s", ringText))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			dd.Dict.Store("pp", ds.NewIntVal(pp-1))

			// ----- 计算伤害 -----
			result, errMsg := calculateDamage(mctx, int64(power), elemType, isSpecial, advantage, ctLimit, attacker, defender)
			if errMsg != "" {
				ReplyToSender(mctx, msg, errMsg)
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			// ========== 输出格式（宝可梦化） ==========

			// 第一部分：详细计算过程（括号包裹）
			var calcText strings.Builder
			fmt.Fprintf(&calcText, "( %s -> %s | %s | ", attacker, defender, name)
			if advantage != "" {
				fmt.Fprintf(&calcText, "d20%s=%d", advantage, result.D20)
			} else {
				fmt.Fprintf(&calcText, "d20=%d", result.D20)
			}
			if result.CritText != "" {
				fmt.Fprintf(&calcText, " %s", result.CritText)
			}

			if !result.Hit {
				fmt.Fprintf(&calcText, " )")
				ReplyToSender(mctx, msg, calcText.String()+"\n"+
					fmt.Sprintf("\xf0\x9f\x92\xa8 %s 对 %s 使用了 %s！\n但 是 没 有 命 中……",
						getPlayerNameTempFunc(mctx), defender, name))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			fmt.Fprintf(&calcText, " | 基础: %d*%d*%d*%d%%/(100*%d)=%d",
				power, result.BattleLv, result.AtkVal, result.RollPct, result.DefVal, result.BaseDmg)

			if result.StabMul != 1.0 || result.TypeMod != 0 {
				factor := (2.0 + result.TypeMod) / 2.0
				if factor < 0.25 {
					factor = 0.25
				}
				fmt.Fprintf(&calcText, " | STAB x%.2f, 克制 x%.2f", result.StabMul, factor)
				if result.FinalDmg != result.BaseDmg {
					fmt.Fprintf(&calcText, " => %d", result.FinalDmg)
				}
			}
			fmt.Fprintf(&calcText, " | PP %d/%d", pp-1, ppmax)
			if ringText != "" {
				fmt.Fprintf(&calcText, " | %s", ringText)
			}
			fmt.Fprintf(&calcText, " )")

			// 战斗演说
			var flavorText strings.Builder
			fmt.Fprintf(&flavorText, "\xe2\x9a\x94\xef\xb8\x8f %s 对 %s 使用了 %s！\n",
				getPlayerNameTempFunc(mctx), defender, name)

			if result.Crit {
				fmt.Fprintf(&flavorText, "\xf0\x9f\x92\xa5 命中要害！\n")
			}
			if result.EffectText != "" {
				fmt.Fprintf(&flavorText, "%s\n", result.EffectText)
			}
			if result.FinalDmg == 0 {
				fmt.Fprintf(&flavorText, "对 %s 没有造成伤害……", defender)
			} else {
				fmt.Fprintf(&flavorText, "%s 受到了 %d 点伤害！", defender, result.FinalDmg)
				// HP条
				// 优先检查防御者是否为 NPC 且已有 HP
				if newHp, maxHp, ok := updateNPCHP(ctx, defender, result.FinalDmg); ok && maxHp > 0 {
					pct := newHp * 10 / maxHp
					if pct > 10 {
						pct = 10
					}
					if pct < 0 {
						pct = 0
					}
					bar := strings.Repeat("█", int(pct)) + strings.Repeat("░", 10-int(pct))
					fmt.Fprintf(&flavorText, "\n  📊 HP: %s %d/%d", bar, newHp, maxHp)
				} else if hpMax, exists := VarGetValueInt64(ctx, "hpmax"); exists && hpMax > 0 {
					// 防御者是玩家
					curHp, _ := VarGetValueInt64(ctx, "hp")
					pct := curHp * 10 / hpMax
					if pct > 10 {
						pct = 10
					}
					if pct < 0 {
						pct = 0
					}
					bar := strings.Repeat("█", int(pct)) + strings.Repeat("░", 10-int(pct))
					fmt.Fprintf(&flavorText, "\n  📊 HP: %s %d/%d", bar, curHp, hpMax)
				}
			}

			fullText := calcText.String() + "\n" + flavorText.String()
			ReplyToSender(mctx, msg, fullText)

		case "pp":
			name := cmdArgs.GetArgN(2)
			deltaStr := cmdArgs.GetArgN(3)
			if name == "" || deltaStr == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			key := "$move_" + name
			val, exists := attrs.LoadX(key)
			if !exists || val.TypeId != ds.VMTypeDict {
				ReplyToSender(mctx, msg, fmt.Sprintf("未找到招式: %s", name))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			dd := val.MustReadDictData()
			ppV, _ := dd.Dict.Load("pp")
			maxV, _ := dd.Dict.Load("ppmax")
			curPP, _ := ppV.ReadInt()
			maxPP, _ := maxV.ReadInt()

			isNeg := strings.HasPrefix(deltaStr, "-")
			deltaStr = strings.TrimLeft(deltaStr, "+-")
			delta, _ := strconv.ParseInt(deltaStr, 10, 64)
			if isNeg {
				delta = -delta
			}

			newPP := int64(curPP) + delta
			if newPP < 0 {
				newPP = 0
			}
			if newPP > int64(maxPP) {
				newPP = int64(maxPP)
			}
			dd.Dict.Store("pp", ds.NewIntVal(ds.IntType(newPP)))
			ReplyToSender(mctx, msg, fmt.Sprintf("%s PP: %d/%d", name, newPP, maxPP))

		case "clr", "clear":
			count := 0
			attrs.Range(func(key string, _ *ds.VMValue) bool {
				if strings.HasPrefix(key, "$move_") {
					attrs.Delete(key)
					count++
				}
				return true
			})
			ReplyToSender(mctx, msg, fmt.Sprintf("已清除%d个招式", count))

		case "", "list", "show":
			var items []string
			attrs.Range(func(key string, value *ds.VMValue) bool {
				if strings.HasPrefix(key, "$move_") && value.TypeId == ds.VMTypeDict {
					dd := value.MustReadDictData()
					readStr := func(k string) string {
						v, ok := dd.Dict.Load(k)
						if !ok {
							return ""
						}
						return v.ToString()
					}
					readInt := func(k string) ds.IntType {
						v, ok := dd.Dict.Load(k)
						if !ok {
							return 0
						}
						ret, _ := v.ReadInt()
						return ret
					}
					name := readStr("name")
					power := readStr("power")
					elem := readStr("type")
					ring := readStr("ring")
					cat := readStr("category")
					pp := readInt("pp")
					ppmax := readInt("ppmax")
					items = append(items, fmt.Sprintf("%s 威力%s %s %s环 %s PP:%d/%d",
						name, power, elem, ring, cat, pp, ppmax))
				}
				return true
			})
			if len(items) == 0 {
				ReplyToSender(mctx, msg, "没有已记录的招式")
			} else {
				ReplyToSender(mctx, msg, fmt.Sprintf("%s的招式:\n%s", getPlayerNameTempFunc(mctx), strings.Join(items, "\n")))
			}

		case "help":
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		default:
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
