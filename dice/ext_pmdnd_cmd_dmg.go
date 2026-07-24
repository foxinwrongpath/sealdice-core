package dice

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var cmdDmg = &CmdItemInfo{
	Name:      "dmg",
	ShortHelp: ".dmg <威力> <类型> [物/特] [优势/劣势] [暴击阈值] [@攻击者] [@防御者]\n示例: .dmg 80 火 物 优势 19 @伊布 @圈圈熊\n💡 NPC处理技巧: @NPC 会读取你的属性（因NPC无独立存储）。\n   - 模拟NPC攻击：先用 .st patk:X satk:Y 修改自身为NPC攻击值，再用 @自己 @目标\n   - 模拟PC打NPC：先用 .st pdef:X sdef:Y 修改自身为NPC防御值，再用 @攻击者 @自己\n   - 嫌麻烦也可以直接手算，或由DM口述结果。",
	Help: "PMDnD 伤害计算(.dmg):\n" +
		".dmg <威力> <类型> [物/特] [优势/劣势] [暴击阈值] [@攻击者] [@防御者]\n示例: .dmg 80 火 物 优势 19 @伊布 @圈圈熊\n💡 NPC处理技巧: @NPC 会读取你的属性（因NPC无独立存储）。\n   - 模拟NPC攻击：先用 .st patk:X satk:Y 修改自身为NPC攻击值，再用 @自己 @目标\n   - 模拟PC打NPC：先用 .st pdef:X sdef:Y 修改自身为NPC防御值，再用 @攻击者 @自己\n   - 嫌麻烦也可以直接手算，或由DM口述结果。",
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		clean := cmdArgs.CleanArgs
		re := regexp.MustCompile(`(\d+)\s+([^\s@]+)\s*(物|特|物理|特殊)?\s*(优势|劣势)?\s*(\d{1,2})?\s*(@\S+)?\s*(@\S+)?`)
		matches := re.FindStringSubmatch(clean)
		if len(matches) < 3 {
			ReplyToSender(ctx, msg, "格式错误，正确格式: .dmg <威力> <类型> [物/特] [优势/劣势] [暴击阈值] [@攻击者] [@防御者]")
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		power, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			ReplyToSender(ctx, msg, "威力必须是数字: "+matches[1])
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		atkType := getPMDnDType(matches[2])
		if _, ok := pmdndTypeChart[atkType]; !ok {
			ReplyToSender(ctx, msg, fmt.Sprintf("未知类型: %s，可用类型: %s", matches[2], strings.Join(pmdndTypeNames, " ")))
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		isSpecial := false
		if len(matches) > 3 && matches[3] != "" {
			isSpecial = matches[3] == "特" || matches[3] == "特殊"
		}

		advantage := ""
		if len(matches) > 4 && matches[4] != "" {
			advantage = matches[4]
		}

		ctLimit := int64(20)
		if len(matches) > 5 && matches[5] != "" {
			if n, e := strconv.ParseInt(matches[5], 10, 64); e == nil && n >= 2 && n <= 20 {
				ctLimit = n
			}
		}

		attacker := ctx.Player.Name
		defender := ""
		if len(matches) > 6 && matches[6] != "" {
			attacker = strings.TrimPrefix(matches[6], "@")
		}
		if len(matches) > 7 && matches[7] != "" {
			defender = strings.TrimPrefix(matches[7], "@")
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

		mctx := ctx
		if attacker != ctx.Player.Name {
			mctx = GetCtxProxyFirst(ctx, cmdArgs)
		}

		result, errMsg := calculateDamage(mctx, power, atkType, isSpecial, advantage, ctLimit, attacker, defender)
		if errMsg != "" {
			ReplyToSender(ctx, msg, errMsg)
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		atkLabel := "物攻"
		defLabel := "物防"
		if isSpecial {
			atkLabel = "特攻"
			defLabel = "特防"
		}

		atkVal := int64(10)
		if isSpecial {
			atkVal, _ = VarGetValueInt64(mctx, "satk")
		} else {
			atkVal, _ = VarGetValueInt64(mctx, "patk")
		}
		if atkVal == 0 {
			atkVal = 10
		}
		defVal := int64(10)
		defCtx := ctx
		if defender != "" && defender != attacker {
			defCtx = ctx
		}
		if isSpecial {
			defVal, _ = VarGetValueInt64(defCtx, "sdef")
		} else {
			defVal, _ = VarGetValueInt64(defCtx, "pdef")
		}
		if defVal == 0 {
			defVal = 10
		}
		battleLv := int64(1)
		if v, _ := VarGetValueInt64(mctx, "战斗等级"); v != 0 {
			battleLv = v
		}

		text := fmt.Sprintf("%s的%s(威力%d) d20 %d%s\n",
			attacker, atkType, power, result.D20, result.CritText)
		text += fmt.Sprintf("基础伤害 = %d * %d(战斗等级) * %d(%s) * %d%% / (100 * %d(%s)) = %d",
			power, battleLv, atkVal, atkLabel, result.RollPct, defVal, defLabel, result.BaseDmg)
		if result.TypeMod != 0 || result.StabMul != 1.0 {
			text += fmt.Sprintf("\n修正: STAB x%.2f, 克制 x%.2f => 最终伤害 %d",
				result.StabMul, (2.0+result.TypeMod)/2.0, result.FinalDmg)
		}
		if atkType != "一般" && defender != "" {
			if mod, ok := pmdndTypeChart[atkType][atkType]; ok && mod != 0 {
				text += fmt.Sprintf("\n提示: %s系对%s系 %s", atkType, atkType, getTypeEffectivenessText(mod))
			}
		}

		ReplyToSender(ctx, msg, text)
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}

var cmdStab = &CmdItemInfo{
	Name:          "stab",
	ShortHelp:     ".stab <招式类型> // 查询自身STAB\n.stab <招式类型> @某人 // 查询他人STAB\n.stab list // 列出自身属性类型",
	Help:          "PMDnD 本系加成(.stab):\n.stab <招式类型> // 查询自身STAB\n.stab <招式类型> @某人 // 查询他人STAB\n.stab list // 列出自身属性类型",
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		mctx := GetCtxProxyFirst(ctx, cmdArgs)
		val := cmdArgs.GetArgN(1)
		switch val {
		case "", "help":
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		case "list":
			var typeList []string
			for _, t := range pmdndTypeNames {
				if v, _ := VarGetValueInt64(mctx, "$type_"+t); v != 0 {
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
				if v, _ := VarGetValueInt64(mctx, "$stab_"+t); v != 0 {
					stabTotal += v
					typeNames = append(typeNames, t)
				}
			}
			if len(typeNames) == 0 {
				for _, t := range pmdndTypeNames {
					if v, _ := VarGetValueInt64(mctx, "$type_"+t); v != 0 {
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

var cmdType = &CmdItemInfo{
	Name:      "type",
	ShortHelp: ".type <技能类型> @<目标类型> // 查单一克制关系\n.type <技能类型> // 查该类型对所有类型的克制关系\n.type list // 列出所有类型",
	Help:      "PMDnD 类型克制(.type):\n.type <技能类型> @<目标类型> // 查单一克制关系\n.type <技能类型> // 查该类型对所有类型的克制关系\n.type list // 列出所有类型",
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
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

var cmdCrit = &CmdItemInfo{
	Name:      "暴击",
	ShortHelp: ".暴击 [阈值] // 判定暴击，默认阈值20\n.crit [阈值] [优势/劣势] // d20暴击判定\n.crit 19 // 暴击阈值19\n.crit 20 优势 // d20优势，暴击阈值20",
	Help: "PMDnD 暴击判定(.暴击 .crit):\n" +
		".暴击 [阈值] // 判定暴击，默认阈值20\n.crit [阈值] [优势/劣势] // d20暴击判定\n.crit 19 // 暴击阈值19\n.crit 20 优势 // d20优势，暴击阈值20",
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
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
