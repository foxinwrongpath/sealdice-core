package dice

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"

	ds "github.com/sealdice/dicescript"
)

// ---------- 辅助函数 ----------

// randomBattleFlavor 根据战斗结果返回随机风味文本
func randomBattleFlavor(result DamageResult, atkName string, defName string) string {
	critPools := []string{
		fmt.Sprintf("💥 %s 的会心一击！", atkName),
		fmt.Sprintf("🔥 命中要害！%s 发出了全力一击！", atkName),
		fmt.Sprintf("⚡ 好机会！%s 击中了 %s 的弱点！", atkName, defName),
	}
	hitPools := []string{
		fmt.Sprintf("👊 %s 的攻击命中了 %s！", atkName, defName),
		fmt.Sprintf("✨ %s 的招式漂亮地击中了！", atkName),
		fmt.Sprintf("🎯 精准的一击！%s 没有给 %s 喘息的机会！", atkName, defName),
		fmt.Sprintf("💪 %s 的攻势凌厉！", atkName),
	}
	missPools := []string{
		fmt.Sprintf("💨 %s 的攻击被 %s 躲开了……", atkName, defName),
		fmt.Sprintf("😰 %s 的招式没有命中……", atkName),
		fmt.Sprintf("🌀 %s 的攻击落空了！", atkName),
	}
	killPools := []string{
		fmt.Sprintf("💀 %s 被彻底击倒了！", defName),
		fmt.Sprintf("⚰️ %s 失去了战斗能力……", defName),
		fmt.Sprintf("✴️ 决定性的 一 击！%s 倒下了！", defName),
	}
	superEffectivePools := []string{
		fmt.Sprintf("💢 效果拔群！%s 发出了痛苦的声音！", defName),
		"💥 这一击效果绝佳！",
	}

	if result.Crit && result.FinalDmg > 0 {
		r := rand.Intn(len(critPools))
		return critPools[r]
	}
	if result.FinalDmg > 0 && result.EffectText != "" {
		r := rand.Intn(len(superEffectivePools))
		return superEffectivePools[r]
	}
	if result.FinalDmg > 0 {
		r := rand.Intn(len(hitPools))
		return hitPools[r]
	}
	if result.FinalDmg == 0 && result.Hit && result.RollPct > 0 {
		r := rand.Intn(len(killPools))
		return killPools[r]
	}
	if !result.Hit {
		r := rand.Intn(len(missPools))
		return missPools[r]
	}
	return ""
}

// getPmdndDetailMode 合并局部 detail 标志和群组全局设置
func getPmdndDetailMode(ctx *MsgContext, localDetail bool) bool {
	if localDetail {
		return true
	}
	if ctx.Group != nil {
		if v, _ := VarGetValueInt64(ctx, "$g_pmdnd_detail"); v != 0 {
			return true
		}
	}
	return false
}

// getPmdndDebugMode 合并局部 debug 标志和群组全局设置
func getPmdndDebugMode(ctx *MsgContext, localDebug bool) bool {
	if localDebug {
		return true
	}
	if ctx.Group != nil {
		if v, _ := VarGetValueInt64(ctx, "$g_pmdnd_debug"); v != 0 {
			return true
		}
	}
	return false
}

// ---------- .pmode 命令 ----------
var cmdPmode = &CmdItemInfo{
	Name:      "pmode",
	ShortHelp: ".pmode // 查看当前输出模式\n.pmode detail // 切换详细计算\n.pmode debug // 切换调试详情\n.pmode off // 关闭所有",
	Help:      getPmodeHelp(),
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		if ctx.Group == nil {
			ReplyToSender(ctx, msg, "此命令只能在群组中使用")
			return CmdExecuteResult{Matched: true, Solved: true}
		}
		val := cmdArgs.GetArgN(1)
		detailOn := false
		debugOn := false

		switch val {
		case "", "show", "list":
			if v, _ := VarGetValueInt64(ctx, "$g_pmdnd_detail"); v != 0 {
				detailOn = true
			}
			if v, _ := VarGetValueInt64(ctx, "$g_pmdnd_debug"); v != 0 {
				debugOn = true
			}
			var status string
			if detailOn && debugOn {
				status = "⚙️ 当前输出模式: detail + debug（显示完整计算过程）"
			} else if debugOn {
				status = "⚙️ 当前输出模式: debug（显示完整计算过程）"
			} else if detailOn {
				status = "⚙️ 当前输出模式: detail（显示简要计算公式）"
			} else {
				status = "⚙️ 当前输出模式: 默认（仅显示伤害结果和血条）"
			}
			status += "\n可用: .pmode detail / .pmode debug / .pmode off"
			ReplyToSender(ctx, msg, status)

		case "detail", "-d":
			v, _ := VarGetValueInt64(ctx, "$g_pmdnd_detail")
			if v != 0 {
				VarSetValueInt64(ctx, "$g_pmdnd_detail", 0)
				ReplyToSender(ctx, msg, "⚙️ detail 模式已关闭")
			} else {
				VarSetValueInt64(ctx, "$g_pmdnd_detail", 1)
				ReplyToSender(ctx, msg, "⚙️ detail 模式已开启（显示简要计算公式）")
			}

		case "debug", "-D":
			v, _ := VarGetValueInt64(ctx, "$g_pmdnd_debug")
			if v != 0 {
				VarSetValueInt64(ctx, "$g_pmdnd_debug", 0)
				ReplyToSender(ctx, msg, "⚙️ debug 模式已关闭")
			} else {
				VarSetValueInt64(ctx, "$g_pmdnd_debug", 1)
				ReplyToSender(ctx, msg, "⚙️ debug 模式已开启（显示完整计算过程，自动包含 detail）")
			}

		case "off", "clear", "clr":
			VarSetValueInt64(ctx, "$g_pmdnd_detail", 0)
			VarSetValueInt64(ctx, "$g_pmdnd_debug", 0)
			ReplyToSender(ctx, msg, "⚙️ 所有全局模式已关闭")

		case "help":
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}

		default:
			ReplyToSender(ctx, msg, "未知参数，可用: detail / debug / off / help")
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}

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
// 返回: 目标列表, 优势/劣势, 暴击阈值, 是否群体模式, detail模式, debug模式, 攻击加值
func parseMoveTargets(ctx *MsgContext, mctx *MsgContext, cmdArgs *CmdArgs, attacker string) ([]string, string, int64, bool, bool, bool, int64) {
	advantage := ""
	ctLimit := int64(20)
	var specifiedTargets []string
	var groupType string
	isGroupMode := false
	detailMode := false
	debugMode := false
	attackBonus := int64(0)

	if len(cmdArgs.Args) < 2 {
		return nil, advantage, ctLimit, isGroupMode, detailMode, debugMode, attackBonus
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
		} else if strings.HasPrefix(p, "+") {
			if n, e := strconv.ParseInt(p[1:], 10, 64); e == nil {
				attackBonus = n
			}
		} else if strings.HasPrefix(p, "-") {
			if n, e := strconv.ParseInt(p[1:], 10, 64); e == nil {
				attackBonus = -n
			}
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

	return finalTargets, advantage, ctLimit, isGroupMode, getPmdndDetailMode(ctx, detailMode), getPmdndDebugMode(ctx, debugMode), attackBonus
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
func executeHealMove(ctx *MsgContext, mctx *MsgContext, msg *Message, name string, power int64, elemType string, advantage string, ctLimit int64, attacker string, defender string, remainingPP int64, attackBonus int64, detailMode bool, debugMode bool) CmdExecuteResult {
	healResult, errMsg := calculateHeal(mctx, power, elemType, advantage, ctLimit, attacker, defender, attackBonus)
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
// 状态/强化招式：仅显示招式效果提醒，不做自动写入。
// 状态的实际命中、层数、是否免疫等由 DM 裁决后通过 .buff add / .buff set 手工应用。
// 机器职责：数值计算（伤害公式含状态修正自动生效）。DM 职责：裁决 and 叙事。
func executeBuffMove(ctx *MsgContext, mctx *MsgContext, msg *Message, name string, effectsRaw string, remainingPP int64, detailMode bool, debugMode bool) CmdExecuteResult {
	playerName := getPlayerNameTempFunc(mctx)

	var lines []string
	lines = append(lines, fmt.Sprintf("💪 %s 使用了 %s！", playerName, name))

	if effectsRaw != "" {
		effectList := strings.Split(effectsRaw, ",")
		var cleanEffects []string
		for _, eff := range effectList {
			eff = strings.TrimSpace(eff)
			if eff != "" {
				cleanEffects = append(cleanEffects, eff)
			}
		}
		if len(cleanEffects) > 0 {
			lines = append(lines, fmt.Sprintf("  📋 招式效果: %s", strings.Join(cleanEffects, " / ")))
			lines = append(lines, fmt.Sprintf("  💡 请 DM 根据命中情况使用以下命令手工应用:"))
			for _, eff := range cleanEffects {
				if strings.Contains(eff, ":") {
					parts := strings.SplitN(eff, ":", 2)
					attr := strings.TrimSpace(parts[0])
					val := strings.TrimSpace(parts[1])
					lines = append(lines, fmt.Sprintf("     .buff set %s %s    (能力等级)", attr, val))
				} else {
					lines = append(lines, fmt.Sprintf("     .buff add <目标> %s <层数>    (状态)", eff))
				}
			}
		}
	} else {
		lines = append(lines, "  💡 该招式未存储效果描述，请 DM 根据规则手工裁决")
		lines = append(lines, "     状态应用: .buff add <目标> <状态名> <层数>")
		lines = append(lines, "     能力等级: .buff set <名称> <值>")
	}

	if debugMode || detailMode {
		lines = append(lines, fmt.Sprintf("  资源: PP %d", remainingPP))
	}

	ReplyToSender(mctx, msg, strings.Join(lines, "\n"))
	return CmdExecuteResult{Matched: true, Solved: true}
}

// ---------- executeDamageMove ----------
func executeDamageMove(ctx *MsgContext, mctx *MsgContext, msg *Message, name string, power int64, elemType string, category string, advantage string, ctLimit int64, attacker string, targets []string, remainingPP int64, hitsStr string, attackBonus int64, detailMode bool, debugMode bool) CmdExecuteResult {
	if hitsStr != "" && hitsStr != "1" {
		return executeMultiHitMove(ctx, mctx, msg, name, power, elemType, category, advantage, ctLimit, attacker, targets, remainingPP, hitsStr, attackBonus, detailMode, debugMode)
	}

	isSpecial := category == "特" || category == "特殊"
	defender := targets[0]

	result, errMsg := calculateDamage(mctx, power, elemType, isSpecial, advantage, ctLimit, attacker, defender, attackBonus)
	if errMsg != "" {
		ReplyToSender(mctx, msg, errMsg)
		return CmdExecuteResult{Matched: true, Solved: true}
	}

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

	// 2. 详细计算（debug模式）
	if debugMode {
		lines = append(lines, fmt.Sprintf("📐 [计算详情]"))
		lines = append(lines, fmt.Sprintf("  攻击者: %s  |  防御者: %s", attacker, defender))
		lines = append(lines, fmt.Sprintf("  招式: %s", name))
		lines = append(lines, fmt.Sprintf("  威力: %d  |  战斗等级: %d  |  攻击值: %d  |  防御值: %d", power, result.BattleLv, result.AtkVal, result.DefVal))
		if result.Hit {
			lines = append(lines, fmt.Sprintf("  基础: %d × %d × %d × %s ÷ (100 × %d) = %d", power, result.BattleLv, result.AtkVal, pctDisplay, result.DefVal, result.BaseDmg))
			if result.StabMul != 1.0 || result.TypeMod != 0 {
				factor := damageModifierFactor(result.TypeMod)
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
		if detailMode {
			lines = append(lines, fmt.Sprintf("  资源: PP %d", remainingPP))
		}
	} else if detailMode && result.Hit {
		calcLine := fmt.Sprintf("📐 %d × %d级 × %d攻 × %s ÷ %d防", power, result.BattleLv, result.AtkVal, pctDisplay, result.DefVal)
		if result.StabMul != 1.0 || result.TypeMod != 0 {
			factor := damageModifierFactor(result.TypeMod)
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
	attackerName := getPlayerNameTempFunc(mctx)
	if attacker == ctx.Player.Name && defender == attacker {
		flavorLines = append(flavorLines, fmt.Sprintf("⚔️ %s 使用了 %s 攻击自己！", attackerName, name))
	} else {
		flavorLines = append(flavorLines, fmt.Sprintf("⚔️ %s 对 %s 使用了 %s！", attackerName, defender, name))
	}
	flavorLines = append(flavorLines, fmt.Sprintf("  %s", randomBattleFlavor(result, attackerName, defender)))
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
					overflow := result.FinalDmg - curHp
					if overflow < 0 {
						overflow = 0
					}

					// 即时死亡：溢出伤害 ≥ HP上限
					if overflow >= hpMax && overflow > 0 {
						flavorLines = append(flavorLines, fmt.Sprintf("💀 溢出伤害 %d ≥ HP上限 %d，%s 被一击致命！", overflow, hpMax, getPlayerNameTempFunc(ctx)))
						newHp = 0
						VarSetValueInt64(ctx, "hp", 0)
						pmdndDeathSavingStable(ctx)
					} else if curHp <= 0 && result.FinalDmg > 0 {
						// HP 已为 0 时受伤 → 死亡豁免失败
						VarSetValueInt64(ctx, "hp", 0)
						failureCount := int64(1)
						if result.Crit {
							failureCount = 2
						}
						a, b := pmdndDeathSaving(ctx, 0, failureCount)
						flavorLines = append(flavorLines, fmt.Sprintf("💔 %s 在濒死状态下受到 %d 伤害！死亡豁免+%d失败 (当前: 成功%d 失败%d)", getPlayerNameTempFunc(ctx), result.FinalDmg, failureCount, a, b))
						exText := pmdndDeathSavingResultCheck(ctx, a, b)
						if exText != "" {
							flavorLines = append(flavorLines, exText)
						}
					} else {
						VarSetValueInt64(ctx, "hp", newHp)
						if newHp == 0 && curHp > 0 {
							flavorLines = append(flavorLines, fmt.Sprintf("💔 %s 失去了战斗能力！\n请使用 .ds 进行濒死豁免", getPlayerNameTempFunc(ctx)))
						}
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
func executeMultiHitMove(ctx *MsgContext, mctx *MsgContext, msg *Message, name string, power int64, elemType string, category string, advantage string, ctLimit int64, attacker string, targets []string, remainingPP int64, hitsStr string, attackBonus int64, detailMode bool, debugMode bool) CmdExecuteResult {
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
		result, errMsg := calculateDamage(mctx, power, elemType, isSpecial, advantage, ctLimit, attacker, defender, attackBonus)
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
		if attackBonus != 0 {
			if attackBonus > 0 {
				detail += fmt.Sprintf(" + %d", attackBonus)
			} else {
				detail += fmt.Sprintf(" - %d", -attackBonus)
			}
		}
		detail += fmt.Sprintf(" = %d", result.AttackRoll)
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
	if hitCountActual > 0 {
		flavorLines = append(flavorLines, fmt.Sprintf("  %s", randomBattleFlavor(DamageResult{
			Crit: critCount > 0, Hit: hitCountActual > 0, FinalDmg: totalDmg,
			RollPct: 1.0, EffectText: "",
		}, getPlayerNameTempFunc(mctx), defender)))
	}
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
			overflow := totalDmg - curHp
			if overflow < 0 {
				overflow = 0
			}
			if overflow >= hpMax && overflow > 0 {
				flavorLines = append(flavorLines, fmt.Sprintf("💀 溢出伤害 %d ≥ HP上限 %d，%s 被一击致命！", overflow, hpMax, getPlayerNameTempFunc(ctx)))
				pmdndDeathSavingStable(ctx)
			} else if curHp <= 0 && totalDmg > 0 {
				failureCount := int64(1)
				if critCount > 0 {
					failureCount = 2
				}
				a, b := pmdndDeathSaving(ctx, 0, failureCount)
				flavorLines = append(flavorLines, fmt.Sprintf("💔 %s 在濒死状态下受到 %d 伤害！死亡豁免+%d失败 (当前: 成功%d 失败%d)", getPlayerNameTempFunc(ctx), totalDmg, failureCount, a, b))
				exText := pmdndDeathSavingResultCheck(ctx, a, b)
				if exText != "" {
					flavorLines = append(flavorLines, exText)
				}
			} else {
				VarSetValueInt64(ctx, "hp", newHp)
				if newHp == 0 && curHp > 0 {
					flavorLines = append(flavorLines, fmt.Sprintf("💔 %s 失去了战斗能力！\n请使用 .ds 进行濒死豁免", getPlayerNameTempFunc(ctx)))
				}
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
	ShortHelp: ".move // 查看招式列表\n.move add ...\n.move <招式名>[+N/-N] [@目标] [优势/劣势] [暴击阈值] [detail] [debug]",
	Help:          getMoveHelp(),
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
			moveVal, exists := attrs.LoadX(key)
			suffixBonus := int64(0)

			// 支持 "撞击+2" 这种将加值拼在招式名后面的写法
			if !exists || moveVal.TypeId != ds.VMTypeDict {
				re := regexp.MustCompile(`^(.+?)([+-]\d+)$`)
				if m := re.FindStringSubmatch(name); m != nil {
					strippedName := m[1]
					if n, err := strconv.ParseInt(m[2], 10, 64); err == nil {
						key = "$move_" + strippedName
						moveVal, exists = attrs.LoadX(key)
						if exists && moveVal.TypeId == ds.VMTypeDict {
							suffixBonus = n
							name = strippedName
						}
					}
				}
			}

			if !exists || moveVal.TypeId != ds.VMTypeDict {
				ReplyToSender(mctx, msg, fmt.Sprintf("未找到招式: %s，请使用 .move add 添加", name))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			dd := moveVal.MustReadDictData()

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
			targets, advantage, ctLimit, isGroupMode, detailMode, debugMode, attackBonus := parseMoveTargets(ctx, mctx, cmdArgs, attacker)
			attackBonus += suffixBonus
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
				return executeHealMove(ctx, mctx, msg, name, int64(power), elemType, advantage, ctLimit, attacker, targets[0], newPP, attackBonus, detailMode, debugMode)
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
			return executeDamageMove(ctx, mctx, msg, name, int64(power), elemType, category, advantage, ctLimit, attacker, targets, newPP, hitsStr, attackBonus, detailMode, debugMode)
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
