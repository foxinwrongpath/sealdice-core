package dice

import (
	"fmt"
	"strconv"
	"strings"
)

// ---------- .dmg 帮助函数 ----------
func getDmgHelp() string {
	return "PMDnD 伤害计算(.dmg):\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		".dmg <类型> <威力> [物/特] [优势/劣势] [暴击阈值] [@攻击者] [@防御者]\n" +
		"  例: .dmg 火 80 物 优势 19 @伊布 @圈圈熊\n" +
		"  例: .dmg 格斗 80 物 @圈圈熊           # 自己攻击圈圈熊\n" +
		"  例: .dmg 格斗 80 物 @圈圈熊 @伊布     # 圈圈熊攻击伊布\n" +
		"  例: .dmg 水 60 特 优势\n" +
		"\n" +
		"📋 @ 参数规则:\n" +
		"  一个 @ → 防御者（被攻击的目标），攻击者默认为自己\n" +
		"  两个 @ → 第一个是攻击者，第二个是防御者\n" +
		"\n" +
		"👾 NPC使用: 先用 .npc set <名称> <属性>:<值> 设置NPC属性\n" +
		"   然后 @NPC 即可自动读取攻击/防御数值\n" +
		"   例: .npc set 圈圈熊 patk:30 pdef:20\n" +
		"       .dmg 格斗 80 物 @圈圈熊\n" +
		"\n" +
		"📋 参数说明:\n" +
		"  类型: 攻击的属性（火/水/草/格斗/一般等）\n" +
		"  威力: 招式的威力值（数字）\n" +
		"  物/特: 物理攻击或特殊攻击（默认物理）\n" +
		"  优势/劣势: d20掷骰的优势或劣势\n" +
		"  暴击阈值: 触发暴击的d20出目（默认20）\n" +
		"  @攻击者: 攻击者名称（默认自己）\n" +
		"  @防御者: 防御者名称\n" +
		"\n" +
		"💡 不指定 @防御者 时自动从先攻列表选取\n" +
		"💡 不指定 @攻击者 时默认为自己\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

func getDmgShortHelp() string {
	return ".dmg <类型> <威力> [物/特] [优势/劣势] [暴击阈值] [@攻击者] [@防御者]\n" +
		"示例: .dmg 火 80 物 优势 19 @伊布 @圈圈熊\n" +
		"一个 @ → 防御者，攻击者是自己\n" +
		"两个 @ → 第一个攻击者，第二个防御者"
}

// ---------- .dmg 命令 ----------
var cmdDmg = &CmdItemInfo{
	Name:          "dmg",
	ShortHelp:     ".dmg <类型> <威力> [物/特] [优势/劣势] [暴击阈值] [@攻击者] [@防御者]\n示例: .dmg 火 80 物 优势 19 @伊布 @圈圈熊",
	Help:          getDmgHelp(),
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		// 检查是否请求帮助
		if cmdArgs.IsArgEqual(1, "help") {
			ReplyToSender(ctx, msg, getDmgHelp())
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		parts := strings.Fields(cmdArgs.CleanArgs)
		if len(parts) < 2 {
			ReplyToSender(ctx, msg, getDmgHelp())
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		// 新格式: <类型> <威力> [物/特] [优势/劣势] [暴击阈值] [@攻击者] [@防御者]
		atkType := getPMDnDType(parts[0])
		if _, ok := pmdndTypeChart[atkType]; !ok {
			ReplyToSender(ctx, msg, fmt.Sprintf("未知类型: %s，可用类型: %s", parts[0], strings.Join(pmdndTypeNames, " ")))
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		power, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			ReplyToSender(ctx, msg, "威力必须是数字: "+parts[1])
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		// 解析剩余参数
		isSpecial := false
		advantage := ""
		ctLimit := int64(20)
		attacker := ctx.Player.Name
		defender := ""

		// ----- 先收集所有 @ 目标 -----
		var atTargets []string
		var otherArgs []string

		for i := 2; i < len(parts); i++ {
			p := parts[i]
			if strings.HasPrefix(p, "@") {
				atTargets = append(atTargets, strings.TrimPrefix(p, "@"))
			} else {
				otherArgs = append(otherArgs, p)
			}
		}

		// ----- 根据 @ 数量分配攻击者和防御者 -----
		switch len(atTargets) {
		case 0:
			// 没有 @，防御者从先攻列表取
			attacker = ctx.Player.Name
			defender = ""
		case 1:
			// 一个 @ → 防御者
			attacker = ctx.Player.Name
			defender = atTargets[0]
		default:
			// 两个及以上 @ → 第一个攻击者，第二个防御者
			attacker = atTargets[0]
			defender = atTargets[1]
		}

		// ----- 解析其他参数（物/特、优势/劣势、暴击阈值） -----
		for _, p := range otherArgs {
			switch {
			case p == "物" || p == "物理" || p == "p" || p == "physical":
				isSpecial = false
			case p == "特" || p == "特殊" || p == "s" || p == "special":
				isSpecial = true
			case p == "优势" || p == "優勢" || p == "adv" || p == "advantage":
				advantage = "优势"
			case p == "劣势" || p == "劣勢" || p == "dis" || p == "disadvantage":
				advantage = "劣势"
			default:
				if n, e := strconv.ParseInt(p, 10, 64); e == nil && n >= 2 && n <= 20 {
					ctLimit = n
				}
			}
		}

		// 如果还没有防御者，从先攻列表取
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

		mctx := ctx
		if attacker != ctx.Player.Name {
			mctx = GetCtxProxyFirst(ctx, cmdArgs)
		}

		result, errMsg := calculateDamage(mctx, power, atkType, isSpecial, advantage, ctLimit, attacker, defender)
		if errMsg != "" {
			ReplyToSender(ctx, msg, errMsg)
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		// ========== 输出格式（宝可梦化） ==========
		var calcText strings.Builder
		fmt.Fprintf(&calcText, "( %s -> %s | %s", attacker, defender, atkType)

		if advantage != "" {
			fmt.Fprintf(&calcText, " | d20%s=%d", advantage, result.D20)
		} else {
			fmt.Fprintf(&calcText, " | d20=%d", result.D20)
		}
		if result.CritText != "" {
			fmt.Fprintf(&calcText, " %s", result.CritText)
		}

		if !result.Hit {
			fmt.Fprintf(&calcText, " )")
			ReplyToSender(ctx, msg, calcText.String()+"\n"+
				fmt.Sprintf("\xf0\x9f\x92\xa8 %s 对 %s 使用了 %s 攻击！\n但 是 没 有 命 中……",
					attacker, defender, atkType))
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		fmt.Fprintf(&calcText, " | 基础: %d*%d*%d*%.0f%%/(100*%d)=%d",
			power, result.BattleLv, result.AtkVal, result.RollPct*100, result.DefVal, result.BaseDmg)

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
		fmt.Fprintf(&calcText, " )")

		// 战斗演说
		var flavorText strings.Builder
		if attacker == defender {
			fmt.Fprintf(&flavorText, "\xe2\x9a\x94\xef\xb8\x8f %s 使用了 %s 攻击自己！\n", attacker, atkType)
		} else {
			fmt.Fprintf(&flavorText, "\xe2\x9a\x94\xef\xb8\x8f %s 对 %s 使用了 %s 攻击！\n", attacker, defender, atkType)
		}

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

			// ---- HP 条显示 ----
			// 1. 先检查防御者是否是 NPC（有 HP 数据）
			if newHp, maxHp, ok := updateNPCHP(ctx, defender, result.FinalDmg); ok && maxHp > 0 {
				// NPC HP 已在 updateNPCHP 中扣除，直接显示
				pct := newHp * 10 / maxHp
				if pct > 10 {
					pct = 10
				}
				if pct < 0 {
					pct = 0
				}
				bar := strings.Repeat("█", int(pct)) + strings.Repeat("░", 10-int(pct))
				fmt.Fprintf(&flavorText, "\n  📊 HP: %s %d/%d", bar, newHp, maxHp)
			} else {
				// 2. 否则检查是否是当前玩家（因为只有当前玩家的上下文可用）
				if defender == ctx.Player.Name {
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
						fmt.Fprintf(&flavorText, "\n  📊 HP: %s %d/%d", bar, newHp, hpMax)
					}
				}
				// 如果防御者是其他玩家（非当前用户），暂时无法读取其 HP，不显示条
			}
		}

		fullText := calcText.String() + "\n" + flavorText.String()
		ReplyToSender(ctx, msg, fullText)
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}

// ---------- .stab 命令 ----------
var cmdStab = &CmdItemInfo{
	Name:          "stab",
	ShortHelp:     ".stab <招式类型> // 查询自身STAB\n.stab <招式类型> @某人 // 查询他人STAB\n.stab list // 列出自身属性类型",
	Help:          "PMDnD 本系加成(.stab):\n.stab <招式类型> // 查询自身STAB\n.stab <招式类型> @某人 // 查询他人STAB\n.stab list // 列出自身属性类型",
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		// 检查是否请求帮助
		if cmdArgs.IsArgEqual(1, "help") {
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		}

		mctx := GetCtxProxyFirst(ctx, cmdArgs)
		val := cmdArgs.GetArgN(1)
		switch val {
		case "", "help":
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		case "list":
			var typeList []string
			for _, t := range pmdndTypeNames {
				if v, _ := VarGetValueInt64(mctx, "type_"+t); v != 0 {
					typeList = append(typeList, fmt.Sprintf("%s:%d", t, v))
				}
			}
			if len(typeList) == 0 {
				ReplyToSender(mctx, msg, fmt.Sprintf("%s未设置属性类型，可用.st type_火:1 设置", getPlayerNameTempFunc(mctx)))
			} else {
				ReplyToSender(mctx, msg, fmt.Sprintf("%s的属性类型: %s", getPlayerNameTempFunc(mctx), strings.Join(typeList, " ")))
			}
		default:
			moveType := getPMDnDType(val)
			var stabTotal int64
			var typeNames []string
			for _, t := range pmdndTypeNames {
				if v, _ := VarGetValueInt64(mctx, "stab_"+t); v != 0 {
					stabTotal += v
					typeNames = append(typeNames, t)
				}
			}
			if len(typeNames) == 0 {
				for _, t := range pmdndTypeNames {
					if v, _ := VarGetValueInt64(mctx, "type_"+t); v != 0 {
						stabTotal += v
						typeNames = append(typeNames, t)
					}
				}
			}
			isStab := false
			for _, t := range typeNames {
				if t == moveType {
					isStab = true
					break
				}
			}
			if isStab {
				var stabCoef int64 = 100
				if stabTotal != 0 {
					stabCoef = 100 + stabTotal
				}
				ReplyToSender(mctx, msg, fmt.Sprintf("%s(%s) 为本系招式！STAB=%.2f 伤害×%.2f",
					moveType, strings.Join(typeNames, "/"), float64(stabCoef)/100, float64(stabCoef)/100))
			} else {
				ReplyToSender(mctx, msg, fmt.Sprintf("%s(%s) 非本系招式，无STAB加成。当前属性类型: %s",
					moveType, strings.Join(typeNames, "/"), strings.Join(typeNames, " ")))
			}
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}

// ---------- .type 命令 ----------
var cmdType = &CmdItemInfo{
	Name:      "type",
	ShortHelp: ".type <技能类型> @<目标类型> // 查单一克制关系\n.type <技能类型> // 查该类型对所有类型的克制关系\n.type list // 列出所有类型",
	Help:      "PMDnD 类型克制(.type):\n.type <技能类型> @<目标类型> // 查单一克制关系\n.type <技能类型> // 查该类型对所有类型的克制关系\n.type list // 列出所有类型",
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		// 检查是否请求帮助
		if cmdArgs.IsArgEqual(1, "help") {
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		}

		val := cmdArgs.GetArgN(1)
		switch val {
		case "", "help":
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		case "list":
			ReplyToSender(ctx, msg, fmt.Sprintf("PMDnD伤害类型: %s", strings.Join(pmdndTypeNames, " ")))
		default:
			atkType := getPMDnDType(val)
			checkChart, ok := pmdndTypeChart[atkType]
			if !ok {
				ReplyToSender(ctx, msg, fmt.Sprintf("未知类型: %s。可用类型: %s", val, strings.Join(pmdndTypeNames, " ")))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			arg2 := cmdArgs.GetArgN(2)
			if arg2 != "" {
				defType := strings.TrimPrefix(arg2, "@")
				defType = getPMDnDType(defType)
				mod := checkChart[defType]
				VarSetValueStr(ctx, "$t攻击类型", atkType)
				VarSetValueStr(ctx, "$t防御类型", defType)
				ReplyToSender(ctx, msg, fmt.Sprintf("%s -> %s: %s (%.0fx)", atkType, defType, getTypeEffectivenessText(mod), mod*2+2))
			} else {
				var lines []string
				for _, defType := range pmdndTypeNames {
					mod := checkChart[defType]
					if mod != 0 {
						lines = append(lines, fmt.Sprintf("%s: %s", defType, getTypeEffectivenessText(mod)))
					}
				}
				if len(lines) == 0 {
					ReplyToSender(ctx, msg, fmt.Sprintf("%s类型对所有其他类型均为正常(1x)", atkType))
				} else {
					ReplyToSender(ctx, msg, fmt.Sprintf("%s类型克制关系:\n%s", atkType, strings.Join(lines, "\n")))
				}
			}
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}

// ---------- .暴击 / .crit 命令 ----------
func getCritHelp() string {
	return "PMDnD 暴击判定(.暴击 .crit):\n" +
		".暴击 [阈值] [优势/劣势]     判定暴击，默认阈值20\n" +
		".crit [阈值] [优势/劣势]     同.暴击\n" +
		"  例: .crit 19               暴击阈值19\n" +
		"  例: .crit 20 优势          d20优势，暴击阈值20"
}

var cmdCrit = &CmdItemInfo{
	Name:      "暴击",
	ShortHelp: ".暴击 [阈值] [优势/劣势] // 暴击判定\n.crit [阈值] [优势/劣势] // 同.暴击",
	Help:      getCritHelp(),
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		// 检查是否请求帮助
		if cmdArgs.IsArgEqual(1, "help") {
			ReplyToSender(ctx, msg, getCritHelp())
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		ctLimit := int64(20)
		advantage := ""

		for i := 1; i < len(cmdArgs.Args); i++ {
			a := cmdArgs.Args[i]
			if n, err := strconv.ParseInt(a, 10, 64); err == nil && n >= 2 && n <= 20 {
				ctLimit = n
			} else if a == "优势" || a == "優勢" {
				advantage = "优势"
			} else if a == "劣势" || a == "劣勢" {
				advantage = "劣势"
			}
		}

		d20Expr := "d20"
		if advantage != "" {
			d20Expr = "d20" + advantage
		}

		ctx.CreateVmIfNotExists()
		r := ctx.Eval(d20Expr, nil)
		if r.vm.Error != nil {
			ReplyToSender(ctx, msg, "骰点失败")
			return CmdExecuteResult{Matched: true, Solved: true}
		}
		d20, _ := r.ReadInt()
		detail := r.vm.GetDetailText()
		if detail == "" {
			detail = fmt.Sprintf("d20=%d", d20)
		}

		if int64(d20) >= ctLimit {
			ReplyToSender(ctx, msg, fmt.Sprintf("%s 暴击！阈值%d %s",
				detail, ctLimit, "(伤害比例+50%%)"))
		} else if int64(d20) == 1 {
			ReplyToSender(ctx, msg, fmt.Sprintf("%s 大失败！", detail))
		} else {
			ReplyToSender(ctx, msg, fmt.Sprintf("%s 未暴击(阈值%d)", detail, ctLimit))
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
