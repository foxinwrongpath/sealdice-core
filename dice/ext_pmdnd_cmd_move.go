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

	// 处理范围 2-5
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

	// 尝试作为 dicescript 表达式解析
	if strings.ContainsAny(hitsStr, "dD+-*/") {
		ctx.CreateVmIfNotExists()
		r := ctx.Eval(hitsStr, nil)
		if r.vm.Error == nil {
			if val, ok := r.ReadInt(); ok && val > 0 {
				return int(val)
			}
		}
	}

	// 尝试直接解析为数字
	if val, err := strconv.Atoi(hitsStr); err == nil && val > 0 {
		return val
	}

	return 1
}

// ---------- parseMoveTargets ----------
func parseMoveTargets(ctx *MsgContext, mctx *MsgContext, cmdArgs *CmdArgs, attacker string) ([]string, string, int64, bool) {
	advantage := ""
	ctLimit := int64(20)
	var specifiedTargets []string
	var groupType string
	isGroupMode := false

	rest := cmdArgs.CleanArgs
	parts := strings.Fields(rest)
	if len(parts) >= 2 {
		params := parts[1:]
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
			} else if n, e := strconv.ParseInt(p, 10, 64); e == nil && n >= 2 && n <= 20 {
				ctLimit = n
			}
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

	return finalTargets, advantage, ctLimit, isGroupMode
}

// ---------- validateMoveTargets ----------
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
func executeHealMove(ctx *MsgContext, mctx *MsgContext, msg *Message, name string, power int64, elemType string, advantage string, ctLimit int64, attacker string, defender string, pp int64, ppmax int64, ringText string, ring int64) CmdExecuteResult {
	healResult, errMsg := calculateHeal(mctx, power, elemType, advantage, ctLimit, attacker, defender)
	if errMsg != "" {
		ReplyToSender(mctx, msg, errMsg)
		return CmdExecuteResult{Matched: true, Solved: true}
	}

	var calcText strings.Builder
	fmt.Fprintf(&calcText, "( %s -> %s | %s", attacker, defender, name)
	if advantage != "" {
		fmt.Fprintf(&calcText, " | d20%s=%d", advantage, healResult.D20)
	} else {
		fmt.Fprintf(&calcText, " | d20=%d", healResult.D20)
	}
	if healResult.CritText != "" {
		fmt.Fprintf(&calcText, " %s", healResult.CritText)
	}
	fmt.Fprintf(&calcText, " | 基础: %d*100*%d*%.0f%%/(100*200)=%d",
		power, healResult.HealAtkVal, healResult.RollPct*100, healResult.BaseHeal)
	if healResult.StabMul != 1.0 {
		fmt.Fprintf(&calcText, " | STAB x%.2f => %d", healResult.StabMul, healResult.FinalHeal)
	}
	fmt.Fprintf(&calcText, " | PP %d/%d", pp-1, ppmax)
	if ringText != "" {
		fmt.Fprintf(&calcText, " | %s", ringText)
	}
	fmt.Fprintf(&calcText, " )")

	var flavorText strings.Builder
	fmt.Fprintf(&flavorText, "\xf0\x9f\x92\x9a %s 对 %s 使用了 %s！\n",
		getPlayerNameTempFunc(mctx), defender, name)
	if healResult.Crit {
		fmt.Fprintf(&flavorText, "命中要害！\n")
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
			fmt.Fprintf(&flavorText, "%s 恢复了 %d 点 HP！\n", defender, actualHeal)
			pct := newHp * 10 / hpMax
			if pct > 10 {
				pct = 10
			}
			bar := strings.Repeat("█", int(pct)) + strings.Repeat("░", 10-int(pct))
			fmt.Fprintf(&flavorText, "  📊 HP: %s %d/%d", bar, newHp, hpMax)
		} else {
			fmt.Fprintf(&flavorText, "%s 恢复了 %d 点 HP！", defender, healResult.FinalHeal)
		}
	} else if newHp, maxHp, ok := updateNPCHP(ctx, defender, -healResult.FinalHeal); ok && maxHp > 0 {
		actualHeal := healResult.FinalHeal
		fmt.Fprintf(&flavorText, "%s 恢复了 %d 点 HP！\n", defender, actualHeal)
		pct := newHp * 10 / maxHp
		if pct > 10 {
			pct = 10
		}
		bar := strings.Repeat("█", int(pct)) + strings.Repeat("░", 10-int(pct))
		fmt.Fprintf(&flavorText, "  📊 HP: %s %d/%d", bar, newHp, maxHp)
	} else {
		fmt.Fprintf(&flavorText, "%s 恢复了 %d 点 HP！", defender, healResult.FinalHeal)
	}

	ReplyToSender(mctx, msg, calcText.String()+"\n"+flavorText.String())
	return CmdExecuteResult{Matched: true, Solved: true}
}

// ---------- executeBuffMove ----------
func executeBuffMove(ctx *MsgContext, mctx *MsgContext, msg *Message, name string, effectsRaw string, pp int64, ppmax int64, ringText string) CmdExecuteResult {
	if effectsRaw != "" {
		state := loadBattleState(ctx)
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
		saveBattleState(ctx, state)
	} else {
		state := loadBattleState(ctx)
		state.AttackLevel = clampLevel(state.AttackLevel + 1)
		saveBattleState(ctx, state)
	}

	state := loadBattleState(ctx)
	ReplyToSender(mctx, msg, fmt.Sprintf(
		"\xf0\x9f\x92\xaa %s 使用了 %s！\n状态已提升！\n📊 %s\nPP: %d/%d  %s",
		getPlayerNameTempFunc(mctx), name, stateToString(state), pp-1, ppmax, ringText))
	return CmdExecuteResult{Matched: true, Solved: true}
}

// ---------- executeDamageMove ----------
func executeDamageMove(ctx *MsgContext, mctx *MsgContext, msg *Message, name string, power int64, elemType string, category string, advantage string, ctLimit int64, attacker string, targets []string, pp int64, ppmax int64, ringText string, hitsStr string) CmdExecuteResult {
	// 如果有多段攻击，走多段攻击逻辑
	if hitsStr != "" && hitsStr != "1" {
		return executeMultiHitMove(ctx, mctx, msg, name, power, elemType, category, advantage, ctLimit, attacker, targets, pp, ppmax, ringText, hitsStr)
	}

	isSpecial := category == "特" || category == "特殊"

	type targetResult struct {
		name     string
		result   DamageResult
		errMsg   string
		calcText string
		flavor   string
	}
	var results []targetResult

	for _, defender := range targets {
		result, errMsg := calculateDamage(mctx, power, elemType, isSpecial, advantage, ctLimit, attacker, defender)
		if errMsg != "" {
			results = append(results, targetResult{name: defender, errMsg: errMsg})
			continue
		}

		var calcText strings.Builder
		fmt.Fprintf(&calcText, "%s -> %s | %s", attacker, defender, name)
		if advantage != "" {
			fmt.Fprintf(&calcText, " | d20%s=%d", advantage, result.D20)
		} else {
			fmt.Fprintf(&calcText, " | d20=%d", result.D20)
		}
		if result.CritText != "" {
			fmt.Fprintf(&calcText, " %s", result.CritText)
		}
		if !result.Hit {
			calcText.WriteString(" | 未命中")
		} else {
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
		}
		fmt.Fprintf(&calcText, " | PP %d/%d", pp-1, ppmax)
		if ringText != "" {
			fmt.Fprintf(&calcText, " | %s", ringText)
		}

		var flavorText strings.Builder
		if !result.Hit {
			fmt.Fprintf(&flavorText, "  → %s：未命中", defender)
		} else if result.FinalDmg == 0 {
			fmt.Fprintf(&flavorText, "  → %s：没有效果", defender)
		} else {
			fmt.Fprintf(&flavorText, "  → %s 受到了 %d 点伤害", defender, result.FinalDmg)
			if newHp, maxHp, ok := updateNPCHP(ctx, defender, result.FinalDmg); ok && maxHp > 0 {
				pct := newHp * 10 / maxHp
				if pct > 10 {
					pct = 10
				}
				if pct < 0 {
					pct = 0
				}
				bar := strings.Repeat("█", int(pct)) + strings.Repeat("░", 10-int(pct))
				fmt.Fprintf(&flavorText, " (%s %d/%d)", bar, newHp, maxHp)
			} else if defender == ctx.Player.Name {
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
					fmt.Fprintf(&flavorText, " (%s %d/%d)", bar, newHp, hpMax)
				}
			}
		}
		results = append(results, targetResult{
			name:     defender,
			result:   result,
			errMsg:   errMsg,
			calcText: calcText.String(),
			flavor:   flavorText.String(),
		})
	}

	var calcLines []string
	var flavorLines []string

	for _, r := range results {
		if r.errMsg != "" {
			calcLines = append(calcLines, fmt.Sprintf("  %s: %s", r.name, r.errMsg))
			continue
		}
		calcLines = append(calcLines, fmt.Sprintf("  (%s)", r.calcText))
		flavorLines = append(flavorLines, r.flavor)
	}

	attackerName := getPlayerNameTempFunc(mctx)
	targetListStr := strings.Join(targets, ", ")
	var header string
	if attacker == ctx.Player.Name && len(targets) == 1 && targets[0] == attacker {
		header = fmt.Sprintf("\xe2\x9a\x94\xef\xb8\x8f %s 使用了 %s 攻击自己！", attackerName, name)
	} else {
		header = fmt.Sprintf("\xe2\x9a\x94\xef\xb8\x8f %s 对 %s 使用了 %s！", attackerName, targetListStr, name)
	}

	if len(results) > 0 && results[0].errMsg == "" && results[0].result.Hit {
		if results[0].result.Crit {
			header += "\n💥 命中要害！"
		}
		if results[0].result.EffectText != "" && len(targets) == 1 {
			header += "\n" + results[0].result.EffectText
		}
	}

	fullText := strings.Join(calcLines, "\n") + "\n" + header + "\n" + strings.Join(flavorLines, "\n")
	ReplyToSender(mctx, msg, fullText)
	return CmdExecuteResult{Matched: true, Solved: true}
}

// ---------- executeMultiHitMove ----------
// executeMultiHitMove 执行多段攻击（每次攻击独立判定）
// executeMultiHitMove 执行多段攻击（每次攻击独立判定）
func executeMultiHitMove(ctx *MsgContext, mctx *MsgContext, msg *Message, name string, power int64, elemType string, category string, advantage string, ctLimit int64, attacker string, targets []string, pp int64, ppmax int64, ringText string, hitsStr string) CmdExecuteResult {
	// 多段攻击暂不支持群体，只取第一个目标
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

	// ---- 执行多次攻击（每次独立判定） ----
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

		// 构建每次攻击的详情（包含计算过程）
		detail := fmt.Sprintf("  #%d: d20=%d", i+1, result.D20)
		if result.Hit && singleDmg > 0 {
			// 显示完整计算过程：基础公式 + 最终伤害
			detail += fmt.Sprintf(" | 基础: %d*%d*%d*%.0f%%/(100*%d)=%d | 伤害 %d",
				power, result.BattleLv, result.AtkVal, result.RollPct*100, result.DefVal, result.BaseDmg, singleDmg)
		} else {
			detail += " | 未命中"
		}
		if result.Crit {
			detail += " 💥暴击"
		}
		hitDetails = append(hitDetails, detail)
	}

	// ---- 输出 ----
	var calcText strings.Builder
	fmt.Fprintf(&calcText, "( %s -> %s | %s | 总攻击次数: %d )",
		attacker, defender, name, hitCount)
	for _, detail := range hitDetails {
		fmt.Fprintf(&calcText, "\n%s", detail)
	}
	fmt.Fprintf(&calcText, "\n  命中 %d 次，总伤害: %d 点！", hitCountActual, totalDmg)
	fmt.Fprintf(&calcText, " | PP %d/%d", pp-1, ppmax)
	if ringText != "" {
		fmt.Fprintf(&calcText, " | %s", ringText)
	}

	var flavorText strings.Builder
	fmt.Fprintf(&flavorText, "\xe2\x9a\x94\xef\xb8\x8f %s 对 %s 使用了 %s！\n",
		getPlayerNameTempFunc(mctx), defender, name)
	fmt.Fprintf(&flavorText, "  攻击 %d 次，命中 %d 次！", hitCount, hitCountActual)
	if critCount > 0 {
		fmt.Fprintf(&flavorText, " 其中 %d 次暴击！", critCount)
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
		fmt.Fprintf(&flavorText, "\n  📊 HP: %s %d/%d", bar, newHp, maxHp)
	} else if defender == ctx.Player.Name {
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
			fmt.Fprintf(&flavorText, "\n  📊 HP: %s %d/%d", bar, newHp, hpMax)
		}
	}

	ReplyToSender(mctx, msg, calcText.String()+"\n"+flavorText.String())
	return CmdExecuteResult{Matched: true, Solved: true}
}

// ---------- cmdMove ----------
var cmdMove = &CmdItemInfo{
	Name:      "move",
	ShortHelp: ".move // 查看招式列表\n.move add <名称> <威力> <类型> <环位> <类别> [效果1] [效果2] ...\n.move del <名称>\n.move pp <名称> +/-N\n.move use <名称>\n.move clr\n.move <招式名> [@目标] [优势/劣势] [暴击阈值]",
	Help: "PMDnD 招式管理(.move):\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		".move                           查看招式列表\n" +
		".move add <名称> <威力> <类型> <环位> <类别> [效果1] [效果2] ...\n" +
		"  类别: 物/特/治疗/强化\n" +
		"  强化效果格式: <属性>:<变化值>  例: 物攻:+2  速度:+1\n" +
		"  多段攻击格式: hits:2-5  例: .move add 种子机关枪 25 草 1 物 hits:2-5\n" +
		"  例: .move add 喷射火焰 90 火 1 特\n" +
		"  例: .move add 剑舞 0 一般 1 强化 物攻:+2\n" +
		"  例: .move add 龙之舞 0 龙 1 强化 物攻:+1,速度:+1\n" +
		".move del <名称>                 删除招式\n" +
		".move use <名称>                 非战斗使用（仅消耗PP和环位）\n" +
		".move pp <名称> +/-N             修改招式PP\n" +
		".move clr                        清除所有招式\n" +
		"\n" +
		"⚔️ 使用招式:\n" +
		"  .move <招式名> [@目标] [优势/劣势] [暴击阈值]\n" +
		"\n" +
		"🎯 目标选取规则:\n" +
		"  · 伤害招式: 不指定目标时自动从先攻列表选取第一个非己单位\n" +
		"  · 治疗/强化: 不指定目标时默认对自己使用\n" +
		"  · 可使用 @目标 指定任意目标\n" +
		"  · 使用 @enemies / @allies / @others / @all 指定群体\n" +
		"\n" +
		"📝 示例:\n" +
		"  .move 喷射火焰 @圈圈熊 优势 19    # 攻击圈圈熊，优势，暴击阈值19\n" +
		"  .move 治愈波动                    # 默认治疗自己\n" +
		"  .move 剑舞 @自己                  # 物攻+2强化自己\n" +
		"  .move 热风 @enemies              # 攻击所有敌人\n" +
		"  .move 种子机关枪 @圈圈熊          # 多段攻击（2-5次）\n" +
		"\n" +
		"📋 类别说明:\n" +
		"  物: 物理攻击  特: 特殊攻击  治疗: 恢复HP  强化: 提升能力\n" +
		"\n" +
		"💡 使用 .buff stat 查看当前战斗状态（能力等级、天气、场地等）\n" +
		"💡 不指定 优势/劣势 时为普通掷骰\n" +
		"💡 不指定 暴击阈值 时默认为20\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━",
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		cmdArgs.ChopPrefixToArgsWith("add", "del", "pp", "rm", "clr", "clear", "use")
		val := cmdArgs.GetArgN(1)
		mctx := GetCtxProxyFirst(ctx, cmdArgs)
		attrs, err := mctx.Dice.AttrsManager.LoadByCtx(mctx)
		if err != nil {
			ReplyToSender(mctx, msg, "加载属性失败: "+err.Error())
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		switch val {
		// ========================================
		// .move add <名称> <威力> <类型> <环位> <类别> [效果1] [效果2] ...
		// ========================================
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

			// 解析效果参数和 hits
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

			// 构建招式数据
			key := "$move_" + name
			m := ds.ValueMap{}
			m.Store("name", ds.NewStrVal(name))
			m.Store("power", ds.NewIntVal(ds.IntType(power)))
			m.Store("type", ds.NewStrVal(elemType))
			m.Store("ring", ds.NewIntVal(ds.IntType(ring)))
			m.Store("category", ds.NewStrVal(category))
			m.Store("pp", ds.NewIntVal(5))
			m.Store("ppmax", ds.NewIntVal(5))
			if len(effects) > 0 {
				m.Store("effects", ds.NewStrVal(strings.Join(effects, ",")))
			}
			if hitsStr != "1" {
				m.Store("hits", ds.NewStrVal(hitsStr))
			}

			attrs.Store(key, ds.NewDictVal(&m).V())

			replyMsg := fmt.Sprintf("添加招式: %s 威力%d 类型%s %d环 %s PP:5/5",
				name, power, elemType, ring, category)
			if len(effects) > 0 {
				replyMsg += fmt.Sprintf(" 效果: %s", strings.Join(effects, ", "))
			}
			if hitsStr != "1" {
				replyMsg += fmt.Sprintf(" 多段攻击: %s", hitsStr)
			}
			ReplyToSender(mctx, msg, replyMsg)

		// ========================================
		// .move del <名称>
		// ========================================
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

		// ========================================
		// .move use <名称>  非战斗使用
		// ========================================
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
			ppmaxV, _ := dd.Dict.Load("ppmax")
			ppmax, _ := ppmaxV.ReadInt()

			if pp <= 0 {
				ReplyToSender(mctx, msg, fmt.Sprintf("招式%s PP不足，当前%d", name, pp))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			dd.Dict.Store("pp", ds.NewIntVal(pp-1))
			ringText, ok := spellRingsGet(mctx, int64(ring), -1)
			if !ok {
				dd.Dict.Store("pp", ds.NewIntVal(pp))
				ReplyToSender(mctx, msg, fmt.Sprintf("环位不足: %s", ringText))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			ReplyToSender(mctx, msg, fmt.Sprintf("%s使用了%s！(非战斗使用) PP: %d/%d (%s)",
				getPlayerNameTempFunc(mctx), name, pp-1, ppmax, ringText))

		// ========================================
		// .move pp <名称> +/-N
		// ========================================
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

		// ========================================
		// .move clr / clear
		// ========================================
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

		// ========================================
		// .move  查看列表
		// ========================================
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
					effects := readStr("effects")
					hits := readStr("hits")

					line := fmt.Sprintf("%s 威力%s %s %s环 %s PP:%d/%d",
						name, power, elem, ring, cat, pp, ppmax)
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

		// ========================================
		// .move <招式名>  使用招式（自动识别类型）
		// ========================================
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

			// 读取招式数据
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
			effectsStr, _ := dd.Dict.Load("effects")
			effectsRaw := ""
			if effectsStr != nil {
				effectsRaw = effectsStr.ToString()
			}
			hitsStr := "1"
			if hitsV, _ := dd.Dict.Load("hits"); hitsV != nil {
				hitsStr = hitsV.ToString()
			}

			if pp <= 0 {
				ReplyToSender(mctx, msg, fmt.Sprintf("招式 %s PP不足，当前%d", name, pp))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			// ---- 解析目标和参数 ----
			attacker := mctx.Player.Name
			targets, advantage, ctLimit, isGroupMode := parseMoveTargets(ctx, mctx, cmdArgs, attacker)
			categoryLower := strings.ToLower(category)

			// ---- 验证目标 ----
			if ok, errMsg := validateMoveTargets(categoryLower, targets, attacker, isGroupMode); !ok {
				ReplyToSender(mctx, msg, errMsg)
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			// 如果目标为空，设置默认
			if len(targets) == 0 {
				if categoryLower == "治疗" || categoryLower == "heal" || categoryLower == "强化" || categoryLower == "buff" {
					targets = []string{attacker}
				} else {
					targets = []string{"目标"}
				}
			}

			// ---- 消耗资源 ----
			dd.Dict.Store("pp", ds.NewIntVal(pp-1))
			ringText, ok := spellRingsGet(mctx, int64(ring), -1)
			if !ok {
				dd.Dict.Store("pp", ds.NewIntVal(pp))
				ReplyToSender(mctx, msg, fmt.Sprintf("环位不足: %s", ringText))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			// ---- 根据类别分发 ----
			if categoryLower == "治疗" || categoryLower == "heal" {
				if len(targets) > 1 {
					dd.Dict.Store("pp", ds.NewIntVal(pp))
					spellRingsGet(mctx, int64(ring), 1)
					ReplyToSender(mctx, msg, "治疗只能指定一个目标，暂不支持群体")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				return executeHealMove(ctx, mctx, msg, name, int64(power), elemType, advantage, ctLimit, attacker, targets[0], int64(pp), int64(ppmax), ringText, int64(ring))
			}

			if categoryLower == "强化" || categoryLower == "buff" {
				if len(targets) != 1 || targets[0] != attacker {
					dd.Dict.Store("pp", ds.NewIntVal(pp))
					spellRingsGet(mctx, int64(ring), 1)
					ReplyToSender(mctx, msg, "强化招式只能对自己使用")
					return CmdExecuteResult{Matched: true, Solved: true}
				}
				return executeBuffMove(ctx, mctx, msg, name, effectsRaw, int64(pp), int64(ppmax), ringText)
			}

			// 默认：伤害
			return executeDamageMove(ctx, mctx, msg, name, int64(power), elemType, category, advantage, ctLimit, attacker, targets, int64(pp), int64(ppmax), ringText, hitsStr)
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
