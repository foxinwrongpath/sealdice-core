package dice

import (
	"fmt"
	"strconv"
	"strings"
)

// ---------- .dmg 命令 ----------
var cmdDmg = &CmdItemInfo{
	Name:          "dmg",
	ShortHelp:     ".dmg <类型> <威力> [物/特] [优势/劣势] [暴击阈值] [+N/-N] [@攻击者] [@防御者] [detail] [debug]\n示例: .dmg 火 80 物 优势 19 +2 @伊布 @圈圈熊",
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

		// 新格式: <类型> <威力> [物/特] [优势/劣势] [暴击阈值] [+N/-N] [@攻击者] [@防御者] [detail] [debug]
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
		detailMode := false
		debugMode := false
		attackBonus := int64(0)

		// ----- 先收集所有 @ 目标 -----
		var atTargets []string
		var otherArgs []string

		for i := 2; i < len(parts); i++ {
			p := parts[i]
			if strings.HasPrefix(p, "@") {
				atTargets = append(atTargets, strings.TrimPrefix(p, "@"))
			} else if p == "detail" || p == "-d" {
				detailMode = true
			} else if p == "debug" || p == "-D" {
				debugMode = true
			} else if strings.HasPrefix(p, "+") {
				if n, e := strconv.ParseInt(p[1:], 10, 64); e == nil {
					attackBonus = n
				}
			} else if strings.HasPrefix(p, "-") {
				if n, e := strconv.ParseInt(p[1:], 10, 64); e == nil {
					attackBonus = -n
				}
			} else {
				otherArgs = append(otherArgs, p)
			}
		}

		// ----- 根据 @ 数量分配攻击者和防御者 -----
		switch len(atTargets) {
		case 0:
			attacker = ctx.Player.Name
			defender = ""
		case 1:
			attacker = ctx.Player.Name
			defender = atTargets[0]
		default:
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

		// 调用 calculateDamage 时传入 attackBonus
		result, errMsg := calculateDamage(mctx, power, atkType, isSpecial, advantage, ctLimit, attacker, defender, attackBonus)
		if errMsg != "" {
			ReplyToSender(ctx, msg, errMsg)
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		// 合并全局输出模式
		detailMode = getPmdndDetailMode(ctx, detailMode)
		debugMode = getPmdndDebugMode(ctx, debugMode)

		// ========== 输出格式（三段式，与 .move 统一） ==========
		var lines []string
		pctDisplay := fmt.Sprintf("%.0f%%", result.RollPct*100)

		// 1. 骰子行（显示攻击掷骰过程）
		diceLine := fmt.Sprintf("🎲 d20=%d", result.D20)
		if attackBonus != 0 {
			if attackBonus > 0 {
				diceLine += fmt.Sprintf(" + %d", attackBonus)
			} else {
				diceLine += fmt.Sprintf(" - %d", -attackBonus)
			}
		}
		diceLine += fmt.Sprintf(" = %d", result.AttackRoll)
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
			lines = append(lines, fmt.Sprintf("  攻击者: %s  |  防御者: %s", attacker, defender))
			lines = append(lines, fmt.Sprintf("  属性: %s  |  威力: %d", atkType, power))
			lines = append(lines, fmt.Sprintf("  战斗等级: %d  |  攻击值: %d  |  防御值: %d", result.BattleLv, result.AtkVal, result.DefVal))
			if result.Hit {
				lines = append(lines, fmt.Sprintf("  基础: %d × %d × %d × %s ÷ (100 × %d) = %d",
					power, result.BattleLv, result.AtkVal, pctDisplay, result.DefVal, result.BaseDmg))
				if result.StabMul != 1.0 || result.TypeMod != 0 {
					factor := damageModifierFactor(result.TypeMod)
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
				if result.Crit {
					lines = append(lines, "  暴击加成: x1.5")
				}
				lines = append(lines, fmt.Sprintf("  最终伤害: %d", result.FinalDmg))
			} else {
				lines = append(lines, "  结果: 未命中")
			}
		} else if detailMode && result.Hit {
			calcLine := fmt.Sprintf("📐 %d × %d级 × %d攻 × %s ÷ %d防",
				power, result.BattleLv, result.AtkVal, pctDisplay, result.DefVal)
			if result.StabMul != 1.0 || result.TypeMod != 0 {
				factor := damageModifierFactor(result.TypeMod)
				if factor < 0.25 {
					factor = 0.25
				}
				calcLine += fmt.Sprintf(" × %.2f修正", factor*result.StabMul)
			}
			if result.Crit {
				calcLine += " × 1.5暴击"
			}
			calcLine += fmt.Sprintf(" = %d", result.FinalDmg)
			lines = append(lines, calcLine)
		}

		// 3. 战斗演说
		flavorLines := []string{}
		if attacker == defender {
			flavorLines = append(flavorLines, fmt.Sprintf("⚔️ %s 使用了 %s 攻击自己！", attacker, atkType))
		} else {
			flavorLines = append(flavorLines, fmt.Sprintf("⚔️ %s 对 %s 使用了 %s 攻击！", attacker, defender, atkType))
		}
		if result.Hit && result.FinalDmg > 0 {
			flavorLines = append(flavorLines, fmt.Sprintf("  %s", randomBattleFlavor(result, attacker, defender)))
		}
		if result.EnvText != "" {
			flavorLines = append(flavorLines, fmt.Sprintf("  %s", result.EnvText))
		}
		if result.StateText != "" {
			flavorLines = append(flavorLines, fmt.Sprintf("  %s", result.StateText))
		}

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
				flavorLines = append(flavorLines, fmt.Sprintf("  对 %s 没有造成伤害……", defender))
			} else {
				flavorLines = append(flavorLines, fmt.Sprintf("  %s 受到了 %d 点伤害！", defender, result.FinalDmg))

				// HP 条
				if newHp, maxHp, ok := updateNPCHP(ctx, defender, result.FinalDmg); ok && maxHp > 0 {
					pct := newHp * 10 / maxHp
					if pct > 10 {
						pct = 10
					}
					if pct < 0 {
						pct = 0
					}
					bar := strings.Repeat("█", int(pct)) + strings.Repeat("░", 10-int(pct))
					flavorLines = append(flavorLines, fmt.Sprintf("  📊 HP: %s %d/%d", bar, newHp, maxHp))
				} else if defender == ctx.Player.Name {
					if hpMax, exists := VarGetValueInt64(ctx, "hpmax"); exists && hpMax > 0 {
						curHp, _ := VarGetValueInt64(ctx, "hp")
						newHp := curHp - result.FinalDmg
						if newHp < 0 {
							newHp = 0
						}
						VarSetValueInt64(ctx, "hp", newHp)
						if newHp == 0 && curHp > 0 {
							flavorLines = append(flavorLines, fmt.Sprintf("💔 %s 失去了战斗能力！\n请使用 .ds 进行濒死豁免", getPlayerNameTempFunc(ctx)))
						}
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
	},
}

// ---------- .type 命令 ----------
var cmdType = &CmdItemInfo{
	Name:      "type",
	ShortHelp: ".type <技能类型> @<目标类型> // 查单一克制关系\n.type <技能类型> // 查该类型对所有类型的克制关系\n.type list // 列出所有类型",
	Help:      getTypeHelp(),
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
				// 计算倍率
				var ratio float64
				if mod >= 0 {
					ratio = (2.0 + mod) / 2.0
				} else {
					ratio = 2.0 / (2.0 - mod)
				}
				// 生成描述文本
				var desc string
				switch {
				case mod >= 2:
					desc = fmt.Sprintf("修正 +%.0f (双倍易伤, %.2fx)", mod, ratio)
				case mod >= 1:
					desc = fmt.Sprintf("修正 +%.0f (易伤, %.2fx)", mod, ratio)
				case mod > 0:
					desc = fmt.Sprintf("修正 +%.1f (易伤, %.2fx)", mod, ratio)
				case mod == 0:
					desc = "修正 0 (正常, 1.00x)"
				case mod > -1:
					desc = fmt.Sprintf("修正 %.1f (抗性, %.2fx)", mod, ratio)
				case mod > -2:
					desc = fmt.Sprintf("修正 %.0f (抗性, %.2fx)", mod, ratio)
				case mod >= -3:
					desc = fmt.Sprintf("修正 %.0f (强抗性, %.2fx)", mod, ratio)
				default:
					desc = fmt.Sprintf("修正 %.0f (免疫级抗性, %.2fx)", mod, ratio)
				}
				ReplyToSender(ctx, msg, fmt.Sprintf("%s -> %s: %s", atkType, defType, desc))
			} else {
				var lines []string
				for _, defType := range pmdndTypeNames {
					mod := checkChart[defType]
					if mod != 0 {
						// 计算倍率
						var ratio float64
						if mod >= 0 {
							ratio = (2.0 + mod) / 2.0
						} else {
							ratio = 2.0 / (2.0 - mod)
						}
						var desc string
						switch {
						case mod >= 2:
							desc = fmt.Sprintf("修正 +%.0f (双倍易伤, %.2fx)", mod, ratio)
						case mod >= 1:
							desc = fmt.Sprintf("修正 +%.0f (易伤, %.2fx)", mod, ratio)
						case mod > 0:
							desc = fmt.Sprintf("修正 +%.1f (易伤, %.2fx)", mod, ratio)
						case mod > -1:
							desc = fmt.Sprintf("修正 %.1f (抗性, %.2fx)", mod, ratio)
						case mod > -2:
							desc = fmt.Sprintf("修正 %.0f (抗性, %.2fx)", mod, ratio)
						case mod >= -3:
							desc = fmt.Sprintf("修正 %.0f (强抗性, %.2fx)", mod, ratio)
						default:
							desc = fmt.Sprintf("修正 %.0f (免疫级抗性, %.2fx)", mod, ratio)
						}
						lines = append(lines, fmt.Sprintf("%s: %s", defType, desc))
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

		for i := 0; i < len(cmdArgs.Args); i++ {
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
