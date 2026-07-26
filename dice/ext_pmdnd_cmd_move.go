package dice

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	ds "github.com/sealdice/dicescript"
)

// ---------- 辅助函数 ----------

// parseHitCount 解析攻击次数
func parseHitCount(ctx *MsgContext, hitsStr string) int {
	hitsStr = strings.TrimSpace(hitsStr)
	if hitsStr == "" || hitsStr == "1" {
		return 1
	}

	if strings.Contains(hitsStr, "-") {
		parts := strings.SplitN(hitsStr, "-", 2)
		if len(parts) == 2 {
			min, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
			max, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err1 == nil && err2 == nil && min > 0 && max >= min {
				return min + rand.Intn(max-min+1)
			}
		}
	}

	if strings.ContainsAny(hitsStr, "dD+-*/") {
		ctx.CreateVmIfNotExists()
		r := ctx.Eval(hitsStr, nil)
		if r.vm.Error == nil {
			if val, ok := r.ReadInt(); ok && val > 0 {
				return int(val)
			}
		}
	}

	if val, err := strconv.Atoi(hitsStr); err == nil && val > 0 {
		return val
	}

	return 1
}

// parseMoveTargets 解析 .move 的目标和参数
func parseMoveTargets(ctx *MsgContext, mctx *MsgContext, cmdArgs *CmdArgs, attacker string) ([]string, string, int64, bool, bool, bool) {
	advantage := ""
	ctLimit := int64(20)
	var specifiedTargets []string
	var groupType string
	isGroupMode := false
	detailMode := false
	debugMode := false

	if len(cmdArgs.Args) < 2 {
		return nil, advantage, ctLimit, isGroupMode, detailMode, debugMode
	}
	params := cmdArgs.Args[1:]

	for _, p := range params {
		if strings.HasPrefix(p, "@") {
			target := strings.TrimPrefix(p, "@")
			switch strings.ToLower(target) {
			case "enemies", "敌人", "敌方":
				isGroupMode = true
				groupType = "enemies"
			case "allies", "友方", "队友":
				isGroupMode = true
				groupType = "allies"
			case "others", "其他", "除己":
				isGroupMode = true
				groupType = "others"
			case "all", "全部", "全体":
				isGroupMode = true
				groupType = "all"
			default:
				specifiedTargets = append(specifiedTargets, target)
			}
		} else if p == "优势" || p == "優勢" || p == "adv" || p == "advantage" {
			advantage = "优势"
		} else if p == "劣势" || p == "劣勢" || p == "dis" || p == "disadvantage" {
			advantage = "劣势"
		} else if p == "detail" || p == "-d" {
			detailMode = true
		} else if p == "debug" || p == "-D" {
			debugMode = true
		} else if n, e := strconv.ParseInt(p, 10, 64); e == nil && n >= 2 && n <= 20 {
			ctLimit = n
		}
	}

	finalTargets := []string{}
	seen := make(map[string]bool)

	if isGroupMode {
		riList := (RIList{}).LoadByCurGroup(ctx)
		for _, item := range riList {
			switch groupType {
			case "enemies":
				if item.name != attacker {
					if !seen[item.name] {
						seen[item.name] = true
						finalTargets = append(finalTargets, item.name)
					}
				}
			case "allies":
				if item.name == attacker {
					if !seen[item.name] {
						seen[item.name] = true
						finalTargets = append(finalTargets, item.name)
					}
				}
			case "others":
				if item.name != attacker {
					if !seen[item.name] {
						seen[item.name] = true
						finalTargets = append(finalTargets, item.name)
					}
				}
			case "all":
				if !seen[item.name] {
					seen[item.name] = true
					finalTargets = append(finalTargets, item.name)
				}
			}
		}
	}

	for _, t := range specifiedTargets {
		if !seen[t] {
			seen[t] = true
			finalTargets = append(finalTargets, t)
		}
	}

	if len(finalTargets) == 0 {
		riList := (RIList{}).LoadByCurGroup(ctx)
		for _, item := range riList {
			if item.name != attacker {
				finalTargets = append(finalTargets, item.name)
				break
			}
		}
	}

	return finalTargets, advantage, ctLimit, isGroupMode, detailMode, debugMode
}

// validateMoveTargets 验证目标是否合法
func validateMoveTargets(categoryLower string, targets []string, attacker string, isGroupMode bool) (bool, string) {
	if len(targets) == 0 {
		return false, "没有指定目标"
	}

	if categoryLower == "强化" || categoryLower == "buff" {
		if len(targets) != 1 || targets[0] != attacker {
			return false, "强化招式只能对自己使用"
		}
	}

	if categoryLower == "治疗" || categoryLower == "heal" {
		if isGroupMode || len(targets) > 1 {
			return false, "治疗只能指定一个目标，暂不支持群体"
		}
	}

	return true, ""
}

// ---------- executeHealMove ----------
func executeHealMove(ctx *MsgContext, mctx *MsgContext, msg *Message, name string, power int64, elemType string, advantage string, ctLimit int64, attacker string, defender string, remainingPP int64, detailMode bool, debugMode bool) CmdExecuteResult {
	healResult, errMsg := calculateHeal(mctx, power, elemType, advantage, ctLimit, attacker, defender)
	if errMsg != "" {
		ReplyToSender(mctx, msg, errMsg)
		return CmdExecuteResult{Matched: true, Solved: true}
	}

	pctDisplay := fmt.Sprintf("%.0f%%", healResult.RollPct*100)

	var lines []string

	// 1. 骰子行
	diceLine := fmt.Sprintf("🎲 d20=%d", healResult.D20)
	if healResult.RollPct > 0 {
		diceLine += fmt.Sprintf(" → %s", pctDisplay)
	}
	if healResult.Crit {
		diceLine += " 💥暴击"
	}
	if healResult.CritText == "【大失败】" {
		diceLine += " 💀大失败"
	}
	lines = append(lines, diceLine)

	// 2. 详细计算（debug模式）
	if debugMode {
		lines = append(lines, fmt.Sprintf("📐 [计算详情]"))
		lines = append(lines, fmt.Sprintf("  攻击者: %s  |  防御者: %s", attacker, defender))
		lines = append(lines, fmt.Sprintf("  招式: %s", name))
		lines = append(lines, fmt.Sprintf("  威力: %d  |  战斗等级: 100  |  治疗值: %d  |  防御固定: 200", power, healResult.HealAtkVal))
		lines = append(lines, fmt.Sprintf("  基础: %d × 100 × %d × %s ÷ (100 × 200) = %d", power, healResult.HealAtkVal, pctDisplay, healResult.BaseHeal))
		if healResult.StabMul != 1.0 {
			lines = append(lines, fmt.Sprintf("  STAB: x%.2f", healResult.StabMul))
		}
		lines = append(lines, fmt.Sprintf("  最终: %d", healResult.FinalHeal))
		if detailMode {
			lines = append(lines, fmt.Sprintf("  资源: PP %d", remainingPP))
		}
	} else if detailMode {
		lines = append(lines, fmt.Sprintf("📐 威力 %d × 100级 × %d治疗 × %s ÷ 200 × %.2f修正 = %d",
			power, healResult.HealAtkVal, pctDisplay, healResult.StabMul, healResult.FinalHeal))
	}

	// 3. 战斗演说
	flavorLines := []string{}
	flavorLines = append(flavorLines, fmt.Sprintf("💚 %s 对 %s 使用了 %s！", getPlayerNameTempFunc(mctx), defender, name))
	if healResult.Crit {
		flavorLines = append(flavorLines, "  命中要害！")
	}

	if defender == ctx.Player.Name {
		if hpMax, exists := VarGetValueInt64(ctx, "hpmax"); exists && hpMax > 0 {
			curHp, _ := VarGetValueInt64(ctx, "hp")
			newHp := curHp + healResult.FinalHeal
			if newHp > hpMax {
				newHp = hpMax
			}
			actualHeal := newHp - curHp
			VarSetValueInt64(ctx, "hp", newHp)
			flavorLines = append(flavorLines, fmt.Sprintf("  %s 恢复了 %d 点 HP！", defender, actualHeal))
			pct := newHp * 10 / hpMax
			if pct > 10 {
				pct = 10
			}
			if pct < 0 {
				pct = 0
			}
			bar := strings.Repeat("█", int(pct)) + strings.Repeat("░", 10-int(pct))
			flavorLines = append(flavorLines, fmt.Sprintf("  📊 HP: %s %d/%d", bar, newHp, hpMax))
		} else {
			flavorLines = append(flavorLines, fmt.Sprintf("  %s 恢复了 %d 点 HP！", defender, healResult.FinalHeal))
		}
	} else if newHp, maxHp, ok := updateNPCHP(ctx, defender, -healResult.FinalHeal); ok && maxHp > 0 {
		actualHeal := healResult.FinalHeal
		flavorLines = append(flavorLines, fmt.Sprintf("  %s 恢复了 %d 点 HP！", defender, actualHeal))
		pct := newHp * 10 / maxHp
		if pct > 10 {
			pct = 10
		}
		if pct < 0 {
			pct = 0
		}
		bar := strings.Repeat("█", int(pct)) + strings.Repeat("░", 10-int(pct))
		flavorLines = append(flavorLines, fmt.Sprintf("  📊 HP: %s %d/%d", bar, newHp, maxHp))
	} else {
		flavorLines = append(flavorLines, fmt.Sprintf("  %s 恢复了 %d 点 HP！", defender, healResult.FinalHeal))
	}

	fullText := strings.Join(lines, "\n") + "\n" + strings.Join(flavorLines, "\n")
	ReplyToSender(mctx, msg, fullText)
	return CmdExecuteResult{Matched: true, Solved: true}
}

// ---------- executeBuffMove ----------
func executeBuffMove(ctx *MsgContext, mctx *MsgContext, msg *Message, name string, effectsRaw string, remainingPP int64, detailMode bool, debugMode bool) CmdExecuteResult {
	if effectsRaw != "" {
		state := loadBattleStateFor(ctx, mctx.Player.Name)
		effectList := strings.Split(effectsRaw, ",")
		applied := false
		for _, eff := range effectList {
			eff = strings.TrimSpace(eff)
			if eff == "" {
				continue
			}
			parts := strings.SplitN(eff, ":", 2)
			if len(parts) != 2 {
				continue
			}
			attr := strings.TrimSpace(parts[0])
			valStr := strings.TrimSpace(parts[1])
			val, err := strconv.Atoi(valStr)
			if err != nil {
				continue
			}
			switch attr {
			case "物攻", "attack":
				state.AttackLevel = clampLevel(state.AttackLevel + val)
				applied = true
			case "物防", "defense":
				state.DefenseLevel = clampLevel(state.DefenseLevel + val)
				applied = true
			case "特攻", "spattack":
				state.SpAttackLevel = clampLevel(state.SpAttackLevel + val)
				applied = true
			case "特防", "spdefense":
				state.SpDefenseLevel = clampLevel(state.SpDefenseLevel + val)
				applied = true
			case "速度", "speed":
				state.SpeedLevel = clampLevel(state.SpeedLevel + val)
				applied = true
			case "反射壁", "reflect":
				state.ReflectWall = val
				if state.ReflectWall < 0 {
					state.ReflectWall = 0
				}
				applied = true
			case "光墙", "lightscreen":
				state.LightScreen = val
				if state.LightScreen < 0 {
					state.LightScreen = 0
				}
				applied = true
			case "保护", "protect":
				state.Protect = val
				if state.Protect < 0 {
					state.Protect = 0
				}
				applied = true
			case "替身", "substitute":
				state.Substitute = val
				if state.Substitute < 0 {
					state.Substitute = 0
				}
				applied = true
			}
		}
		if !applied {
			state.AttackLevel = clampLevel(state.AttackLevel + 1)
		}
		saveBattleStateFor(ctx, mctx.Player.Name, state)
	} else {
		state := loadBattleStateFor(ctx, mctx.Player.Name)
		state.AttackLevel = clampLevel(state.AttackLevel + 1)
		saveBattleStateFor(ctx, mctx.Player.Name, state)
	}

	state := loadBattleStateFor(ctx, mctx.Player.Name)
	var lines []string
	lines = append(lines, fmt.Sprintf("💪 %s 使用了 %s！", getPlayerNameTempFunc(mctx), name))
	lines = append(lines, fmt.Sprintf("  %s", stateToString(state)))
	if debugMode || detailMode {
		lines = append(lines, fmt.Sprintf("  资源: PP %d", remainingPP))
	}

	ReplyToSender(mctx, msg, strings.Join(lines, "\n"))
	return CmdExecuteResult{Matched: true, Solved: true}
}

// ---------- executeDamageMove ----------
func executeDamageMove(ctx *MsgContext, mctx *MsgContext, msg *Message, name string, power int64, elemType string, category string, advantage string, ctLimit int64, attacker string, targets []string, remainingPP int64, hitsStr string, detailMode bool, debugMode bool) CmdExecuteResult {
	if hitsStr != "" && hitsStr != "1" {
		return executeMultiHitMove(ctx, mctx, msg, name, power, elemType, category, advantage, ctLimit, attacker, targets, remainingPP, hitsStr, detailMode, debugMode)
	}

	isSpecial := category == "特" || category == "特殊"
	defender := targets[0]

	result, errMsg := calculateDamage(mctx, power, elemType, isSpecial, advantage, ctLimit, attacker, defender)
	if errMsg != "" {
		ReplyToSender(mctx, msg, errMsg)
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

	// 2. 详细计算（debug模式）
	if debugMode {
		lines = append(lines, fmt.Sprintf("📐 [计算详情]"))
		lines = append(lines, fmt.Sprintf("  攻击者: %s  |  防御者: %s", attacker, defender))
		lines = append(lines, fmt.Sprintf("  招式: %s", name))
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
		if detailMode {
			lines = append(lines, fmt.Sprintf("  资源: PP %d", remainingPP))
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
	attackerName := getPlayerNameTempFunc(mctx)
	if attacker == ctx.Player.Name && defender == attacker {
		flavorLines = append(flavorLines, fmt.Sprintf("⚔️ %s 使用了 %s 攻击自己！", attackerName, name))
	} else {
		flavorLines = append(flavorLines, fmt.Sprintf("⚔️ %s 对 %s 使用了 %s！", attackerName, defender, name))
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
					// 如果 HP 归零，触发死亡豁免提示
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
	ReplyToSender(mctx, msg, fullText)
	return CmdExecuteResult{Matched: true, Solved: true}
}

// ---------- executeMultiHitMove ----------
func executeMultiHitMove(ctx *MsgContext, mctx *MsgContext, msg *Message, name string, power int64, elemType string, category string, advantage string, ctLimit int64, attacker string, targets []string, remainingPP int64, hitsStr string, detailMode bool, debugMode bool) CmdExecuteResult {
	if len(targets) > 1 {
		ReplyToSender(mctx, msg, "多段攻击暂不支持群体，请指定一个目标")
		return CmdExecuteResult{Matched: true, Solved: true}
	}
	defender := targets[0]

	hitCount := parseHitCount(ctx, hitsStr)
	if hitCount <= 0 {
		hitCount = 1
	}
	if hitCount > 10 {
		hitCount = 10
	}

	isSpecial := category == "特" || category == "特殊"

	var hitDetails []string
	var totalDmg int64
	var critCount int
	var hitCountActual int

	for i := 0; i < hitCount; i++ {
		result, errMsg := calculateDamage(mctx, power, elemType, isSpecial, advantage, ctLimit, attacker, defender)
		if errMsg != "" {
			ReplyToSender(mctx, msg, errMsg)
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		singleDmg := result.FinalDmg
		totalDmg += singleDmg

		if result.Hit && singleDmg > 0 {
			hitCountActual++
		}
		if result.Crit {
			critCount++
		}

		pctDisplay := fmt.Sprintf("%.0f%%", result.RollPct*100)
		detail := fmt.Sprintf("  #%d: d20=%d", i+1, result.D20)
		if result.Hit && singleDmg > 0 {
			if debugMode {
				detail += fmt.Sprintf(" → %s → %d × %.2f修正 = %d", pctDisplay, result.BaseDmg, (2.0+result.TypeMod)/2.0*result.StabMul, singleDmg)
			} else if detailMode {
				detail += fmt.Sprintf(" → %s → 伤害 %d", pctDisplay, singleDmg)
			} else {
				detail += fmt.Sprintf(" → 伤害 %d", singleDmg)
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
	lines = append(lines, fmt.Sprintf("🎲 %s 攻击 %d 次！", name, hitCount))
	for _, detail := range hitDetails {
		lines = append(lines, detail)
	}
	lines = append(lines, fmt.Sprintf("  总伤害: %d 点！", totalDmg))

	if debugMode || detailMode {
		lines = append(lines, fmt.Sprintf("  资源: PP %d", remainingPP))
	}

	flavorLines := []string{}
	flavorLines = append(flavorLines, fmt.Sprintf("⚔️ %s 对 %s 使用了 %s！",
		getPlayerNameTempFunc(mctx), defender, name))
	flavorLines = append(flavorLines, fmt.Sprintf("  攻击 %d 次，命中 %d 次！", hitCount, hitCountActual))
	if critCount > 0 {
		flavorLines = append(flavorLines, fmt.Sprintf("  其中 %d 次暴击！", critCount))
	}

	if newHp, maxHp, ok := updateNPCHP(ctx, defender, totalDmg); ok && maxHp > 0 {
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
			newHp := curHp - totalDmg
			if newHp < 0 {
				newHp = 0
			}
			VarSetValueInt64(ctx, "hp", newHp)
			// 如果 HP 归零，触发死亡豁免提示
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

	fullText := strings.Join(lines, "\n") + "\n" + strings.Join(flavorLines, "\n")
	ReplyToSender(mctx, msg, fullText)
	return CmdExecuteResult{Matched: true, Solved: true}
}

// ---------- cmdMove ----------
var cmdMove = &CmdItemInfo{
	Name:      "move",
	ShortHelp: ".move // 查看招式列表\n.move add ...\n.move <招式名> [@目标] [优势/劣势] [暴击阈值] [detail] [debug]",
	Help: "PMDnD 招式管理(.move):\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		".move                           查看招式列表\n" +
		".move add <名称> <威力> <类型> <环位> <类别> [效果1] [效果2] ...\n" +
		"  类别: 物/特/治疗/强化\n" +
		"  强化效果格式: <属性>:<变化值>  例: 物攻:+2\n" +
		"  多段攻击格式: hits:2-5\n" +
		"  例: .move add 种子机关枪 25 草 1 物 hits:2-5\n" +
		".move del <名称>                 删除招式\n" +
		".move use <名称>                 非战斗使用（仅消耗PP）\n" +
		".move clr                        清除所有招式\n" +
		"\n" +
		"⚔️ 使用招式:\n" +
		"  .move <招式名> [@目标] [优势/劣势] [暴击阈值] [detail] [debug]\n" +
		"\n" +
		"🎯 目标选取规则:\n" +
		"  · 伤害招式: 不指定目标时自动从先攻列表选取第一个非己单位\n" +
		"  · 治疗/强化: 不指定目标时默认对自己使用\n" +
		"  · 可使用 @目标 指定任意目标\n" +
		"  · 使用 @enemies / @allies / @others / @all 指定群体\n" +
		"\n" +
		"📝 示例:\n" +
		"  .move 喷射火焰 @圈圈熊 优势 19          # 常规攻击\n" +
		"  .move 喷射火焰 @圈圈熊 detail            # 显示简要计算\n" +
		"  .move 喷射火焰 @圈圈熊 debug             # 显示完整计算\n" +
		"  .move 治愈波动                          # 默认治疗自己\n" +
		"  .move 种子机关枪 @圈圈熊                # 多段攻击\n" +
		"\n" +
		"📋 类别说明:\n" +
		"  物: 物理攻击  特: 特殊攻击  治疗: 恢复HP  强化: 提升能力\n" +
		"\n" +
		"💡 使用 .buff stat 查看当前战斗状态\n" +
		"💡 PP消耗 = 环位 × 30，从角色总PP中扣除\n" +
		"💡 当 NPC HP 归零时自动从先攻列表移除\n" +
		"💡 当玩家 HP 归零时自动提示使用 .ds 进行濒死豁免\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━",
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		cmdArgs.ChopPrefixToArgsWith("add", "del", "clr", "clear", "use")
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
				category = "物"
			}
			if category == "物理" || category == "p" || category == "physical" {
				category = "物"
			}
			if category == "特殊" || category == "s" || category == "special" {
				category = "特"
			}
			if category == "heal" || category == "治疗" {
				category = "治疗"
			}
			if category == "buff" || category == "强化" {
				category = "强化"
			}

			var effects []string
			hitsStr := "1"
			for i := 6; i < len(cmdArgs.Args); i++ {
				arg := cmdArgs.Args[i]
				if strings.HasPrefix(arg, "hits:") {
					hitsStr = strings.TrimPrefix(arg, "hits:")
				} else if strings.Contains(arg, ",") {
					parts := strings.Split(arg, ",")
					for _, p := range parts {
						p = strings.TrimSpace(p)
						if strings.HasPrefix(p, "hits:") {
							hitsStr = strings.TrimPrefix(p, "hits:")
						} else if p != "" {
							effects = append(effects, p)
						}
					}
				} else {
					effects = append(effects, arg)
				}
			}

			key := "$move_" + name
			m := ds.ValueMap{}
			m.Store("name", ds.NewStrVal(name))
			m.Store("power", ds.NewIntVal(ds.IntType(power)))
			m.Store("type", ds.NewStrVal(elemType))
			m.Store("ring", ds.NewIntVal(ds.IntType(ring)))
			m.Store("category", ds.NewStrVal(category))
			if len(effects) > 0 {
				m.Store("effects", ds.NewStrVal(strings.Join(effects, ",")))
			}
			if hitsStr != "1" {
				m.Store("hits", ds.NewStrVal(hitsStr))
			}
			attrs.Store(key, ds.NewDictVal(&m).V())

			replyMsg := fmt.Sprintf("添加招式: %s 威力%d 类型%s %d环 %s",
				name, power, elemType, ring, category)
			if len(effects) > 0 {
				replyMsg += fmt.Sprintf(" 效果: %s", strings.Join(effects, ", "))
			}
			if hitsStr != "1" {
				replyMsg += fmt.Sprintf(" 多段攻击: %s", hitsStr)
			}
			ReplyToSender(mctx, msg, replyMsg)

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

			ringV, _ := dd.Dict.Load("ring")
			ring, _ := ringV.ReadInt()

			// 读取全局PP
			pp, _ := VarGetValueInt64(mctx, "pp")
			ppConsume := int64(ring) * 30
			if pp < ppConsume {
				ReplyToSender(mctx, msg, fmt.Sprintf("PP不足！需要 %d，当前 %d", ppConsume, pp))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			newPP := pp - ppConsume
			VarSetValueInt64(mctx, "pp", newPP)

			ReplyToSender(mctx, msg, fmt.Sprintf("%s使用了%s！(非战斗使用) PP: %d", getPlayerNameTempFunc(mctx), name, newPP))

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
					name := readStr("name")
					power := readStr("power")
					elem := readStr("type")
					ring := readStr("ring")
					cat := readStr("category")
					effects := readStr("effects")
					hits := readStr("hits")

					line := fmt.Sprintf("%s 威力%s %s %s环 %s",
						name, power, elem, ring, cat)
					if effects != "" {
						line += fmt.Sprintf(" [%s]", effects)
					}
					if hits != "" && hits != "1" {
						line += fmt.Sprintf(" {多段攻击:%s}", hits)
					}
					items = append(items, line)
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
			name := val
			if name == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}

			key := "$move_" + name
			val, exists := attrs.LoadX(key)
			if !exists || val.TypeId != ds.VMTypeDict {
				ReplyToSender(mctx, msg, fmt.Sprintf("未找到招式: %s，请使用 .move add 添加", name))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			dd := val.MustReadDictData()

			powerV, _ := dd.Dict.Load("power")
			power, _ := powerV.ReadInt()
			typeV, _ := dd.Dict.Load("type")
			elemType := typeV.ToString()
			catV, _ := dd.Dict.Load("category")
			category := catV.ToString()
			ringV, _ := dd.Dict.Load("ring")
			ring, _ := ringV.ReadInt()
			effectsStr, _ := dd.Dict.Load("effects")
			effectsRaw := ""
			if effectsStr != nil {
				effectsRaw = effectsStr.ToString()
			}
			hitsStr := "1"
			if hitsV, _ := dd.Dict.Load("hits"); hitsV != nil {
				hitsStr = hitsV.ToString()
			}

			// ---- 读取全局PP ----
			pp, _ := VarGetValueInt64(mctx, "pp")
			ppConsume := int64(ring) * 30
			if pp < ppConsume {
				ReplyToSender(mctx, msg, fmt.Sprintf("PP不足！需要 %d，当前 %d", ppConsume, pp))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			newPP := pp - ppConsume
			VarSetValueInt64(mctx, "pp", newPP)

			// ---- 解析目标和参数 ----
			attacker := mctx.Player.Name
			targets, advantage, ctLimit, isGroupMode, detailMode, debugMode := parseMoveTargets(ctx, mctx, cmdArgs, attacker)
			categoryLower := strings.ToLower(category)

			if ok, errMsg := validateMoveTargets(categoryLower, targets, attacker, isGroupMode); !ok {
				// 回滚PP
				VarSetValueInt64(mctx, "pp", pp)
				ReplyToSender(mctx, msg, errMsg)
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			if len(targets) == 0 {
				if categoryLower == "治疗" || categoryLower == "heal" || categoryLower == "强化" || categoryLower == "buff" {
					targets = []string{attacker}
				} else {
					targets = []string{"目标"}
				}
			}

			// 根据类别分发
			if categoryLower == "治疗" || categoryLower == "heal" {
				if len(targets) > 1 {
					VarSetValueInt64(mctx, "pp", pp)
					ReplyToSender(mctx, msg, "治疗只能指定一个目标，暂不支持群体")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				return executeHealMove(ctx, mctx, msg, name, int64(power), elemType, advantage, ctLimit, attacker, targets[0], newPP, detailMode, debugMode)
			}

			if categoryLower == "强化" || categoryLower == "buff" {
				if len(targets) != 1 || targets[0] != attacker {
					VarSetValueInt64(mctx, "pp", pp)
					ReplyToSender(mctx, msg, "强化招式只能对自己使用")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				return executeBuffMove(ctx, mctx, msg, name, effectsRaw, newPP, detailMode, debugMode)
			}

			// 默认：伤害
			return executeDamageMove(ctx, mctx, msg, name, int64(power), elemType, category, advantage, ctLimit, attacker, targets, newPP, hitsStr, detailMode, debugMode)
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
