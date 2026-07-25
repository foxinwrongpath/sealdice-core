package dice

import (
	"fmt"
	"strconv"
	"strings"

	ds "github.com/sealdice/dicescript"
)

var cmdMove = &CmdItemInfo{
	Name: "move",
	ShortHelp: ".move // 查看招式列表\n" +
		".move add <名称> <威力> <类型> <环位> <类别> // 添加招式\n" +
		".move del <名称> // 删除招式\n" +
		".move use <名称> // 非战斗使用（仅消耗PP和环位）\n" +
		".move pp <名称> +/-N // 修改PP\n" +
		".move clr // 清除所有招式\n" +
		".move <招式名> [@目标...] [@群体] [优势/劣势] [暴击阈值] // 战斗使用",
	Help: "PMDnD 招式管理(.move):\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		".move                           查看招式列表\n" +
		".move add <名称> <威力> <类型> <环位> <类别>  添加招式\n" +
		"  类别: 物/特/治疗  例: .move add 喷射火焰 90 火 1 特\n" +
		".move del <名称>                 删除招式\n" +
		".move use <名称>                 非战斗使用（仅消耗PP和环位）\n" +
		".move pp <名称> +/-N             修改招式PP\n" +
		".move clr                        清除所有招式\n" +
		".move <招式名> [@目标...] [@群体] [优势/劣势] [暴击阈值]  战斗使用\n" +
		"\n" +
		"📋 目标指定方式:\n" +
		"  @目标1 @目标2 ...   指定多个目标（单体/多目标）\n" +
		"  @enemies / @敌人     所有敌人（先攻列表中除自己外）\n" +
		"  @allies / @友方      所有友方（目前只有自己）\n" +
		"  @others / @其他      除自己外所有目标\n" +
		"  @all / @全部         先攻列表中所有目标\n" +
		"  不指定目标           自动从先攻列表选第一个非己单位\n" +
		"\n" +
		"📝 示例:\n" +
		"  .move 喷射火焰 @圈圈熊 优势 19      # 单体攻击\n" +
		"  .move 热风 @圈圈熊 @大嘴蝠          # 多目标攻击\n" +
		"  .move 热风 @enemies                 # 攻击所有敌人\n" +
		"  .move 生命水滴 @伊布                # 单体治疗\n" +
		"  .move 撞击                          # 自动选目标\n" +
		"  .move use 喷射火焰                  # 非战斗使用\n" +
		"\n" +
		"📋 类别说明:\n" +
		"  物: 物理攻击  特: 特殊攻击  治疗: 恢复HP\n" +
		"\n" +
		"💡 不指定 @目标 时自动从先攻列表选取\n" +
		"💡 不指定 优势/劣势 时为普通掷骰\n" +
		"💡 不指定 暴击阈值 时默认为20\n" +
		"💡 治疗类招式暂不支持群体目标\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━",
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		cmdArgs.ChopPrefixToArgsWith("add", "del", "use", "pp", "rm", "clr", "clear")
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
			// 兼容旧输入
			if category == "物理" || category == "p" || category == "physical" {
				category = "物"
			}
			if category == "特殊" || category == "s" || category == "special" {
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
			// ============================================
			// .move <招式名> 自动识别类型 + 群体支持
			// ============================================
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

			if pp <= 0 {
				ReplyToSender(mctx, msg, fmt.Sprintf("招式 %s PP不足，当前%d", name, pp))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			// ==========================================
			// 解析目标和参数
			// ==========================================
			advantage := ""
			ctLimit := int64(20)
			attacker := mctx.Player.Name

			// 收集目标
			var specifiedTargets []string
			var groupType string // "enemies", "allies", "others", "all", ""
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

			// ==========================================
			// 生成最终目标列表
			// ==========================================
			var finalTargets []string
			seen := make(map[string]bool)

			// 如果有群体模式，从先攻列表生成目标
			if isGroupMode {
				riList := (RIList{}).LoadByCurGroup(ctx)
				for _, item := range riList {
					// 根据群体类型筛选
					switch groupType {
					case "enemies":
						// 敌人：先攻列表中除自己外的所有单位
						if item.name != attacker {
							if !seen[item.name] {
								seen[item.name] = true
								finalTargets = append(finalTargets, item.name)
							}
						}
					case "allies":
						// 友方：只有自己
						if item.name == attacker {
							if !seen[item.name] {
								seen[item.name] = true
								finalTargets = append(finalTargets, item.name)
							}
						}
					case "others":
						// 除自己外所有
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

			// 合并指定的目标（去重）
			for _, t := range specifiedTargets {
				if !seen[t] {
					seen[t] = true
					finalTargets = append(finalTargets, t)
				}
			}

			// 如果没有任何目标，从先攻列表取第一个非己单位
			if len(finalTargets) == 0 {
				riList := (RIList{}).LoadByCurGroup(ctx)
				for _, item := range riList {
					if item.name != attacker {
						finalTargets = append(finalTargets, item.name)
						break
					}
				}
			}
			if len(finalTargets) == 0 {
				finalTargets = []string{"目标"}
			}

			// ==========================================
			// 消耗资源（先扣PP和环位，如果后续失败再回滚）
			// ==========================================
			dd.Dict.Store("pp", ds.NewIntVal(pp-1))
			ringText, ok := spellRingsGet(mctx, int64(ring), -1)
			if !ok {
				dd.Dict.Store("pp", ds.NewIntVal(pp))
				ReplyToSender(mctx, msg, fmt.Sprintf("环位不足: %s", ringText))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			categoryLower := strings.ToLower(category)

			// ==========================================
			// 处理治疗（目前只支持单体，群体提示）
			// ==========================================
			if categoryLower == "治疗" || categoryLower == "heal" {
				if len(finalTargets) > 1 {
					dd.Dict.Store("pp", ds.NewIntVal(pp))
					spellRingsGet(mctx, int64(ring), 1) // 回滚环位
					ReplyToSender(mctx, msg, "暂不支持群体治疗，请指定一个目标")
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				defender := finalTargets[0]
				healResult, errMsg := calculateHeal(mctx, int64(power), elemType, advantage, ctLimit, attacker, defender)
				if errMsg != "" {
					dd.Dict.Store("pp", ds.NewIntVal(pp))
					spellRingsGet(mctx, int64(ring), 1)
					ReplyToSender(mctx, msg, errMsg)
					return CmdExecuteResult{Matched: true, Solved: true}
				}

				// 输出治疗结果
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
				fmt.Fprintf(&calcText, " | 基础: %d*100*%d*%d%%/(100*200)=%d",
					power, healResult.HealAtkVal, healResult.RollPct, healResult.BaseHeal)
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

				// 应用治疗
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

			// ==========================================
			// 伤害分支（支持多目标）
			// ==========================================
			isSpecial := category == "特" || category == "特殊"

			// 对每个目标计算伤害
			type targetResult struct {
				name     string
				result   DamageResult
				errMsg   string
				calcText string
				flavor   string
			}
			var results []targetResult

			for _, defender := range finalTargets {
				// 计算伤害
				result, errMsg := calculateDamage(mctx, int64(power), elemType, isSpecial, advantage, ctLimit, attacker, defender)
				if errMsg != "" {
					// 如果出错，跳过这个目标
					results = append(results, targetResult{name: defender, errMsg: errMsg})
					continue
				}

				// 生成这个目标的详细计算文本
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
				}
				fmt.Fprintf(&calcText, " | PP %d/%d", pp-1, ppmax)
				if ringText != "" {
					fmt.Fprintf(&calcText, " | %s", ringText)
				}

				// 生成战斗演说文本
				var flavorText strings.Builder
				if !result.Hit {
					fmt.Fprintf(&flavorText, "  → %s：未命中", defender)
				} else if result.FinalDmg == 0 {
					fmt.Fprintf(&flavorText, "  → %s：没有效果", defender)
				} else {
					fmt.Fprintf(&flavorText, "  → %s 受到了 %d 点伤害", defender, result.FinalDmg)
					// 更新HP
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
				// 效果文本（只在单体时有）
				if result.EffectText != "" && !result.Hit {
					// 不显示
				}

				results = append(results, targetResult{
					name:     defender,
					result:   result,
					errMsg:   errMsg,
					calcText: calcText.String(),
					flavor:   flavorText.String(),
				})
			}

			// ==========================================
			// 输出
			// ==========================================
			var calcLines []string
			var flavorLines []string

			// 处理有错误的目标
			for _, r := range results {
				if r.errMsg != "" {
					calcLines = append(calcLines, fmt.Sprintf("  %s: %s", r.name, r.errMsg))
					continue
				}
				calcLines = append(calcLines, fmt.Sprintf("  (%s)", r.calcText))
				flavorLines = append(flavorLines, r.flavor)
			}

			// 主宣言
			attackerName := getPlayerNameTempFunc(mctx)
			targetListStr := strings.Join(finalTargets, ", ")
			var header string
			if attacker == ctx.Player.Name && len(finalTargets) == 1 && finalTargets[0] == attacker {
				header = fmt.Sprintf("⚔️ %s 使用了 %s 攻击自己！", attackerName, name)
			} else {
				header = fmt.Sprintf("⚔️ %s 对 %s 使用了 %s！", attackerName, targetListStr, name)
			}

			// 暴击和效果（取第一个目标的信息，多目标时暴击/效果可能不同，简化只对第一个显示）
			if len(results) > 0 && results[0].errMsg == "" && results[0].result.Hit {
				if results[0].result.Crit {
					header += "\n💥 命中要害！"
				}
				if results[0].result.EffectText != "" && len(finalTargets) == 1 {
					header += "\n" + results[0].result.EffectText
				}
			}

			// 合并输出
			fullText := strings.Join(calcLines, "\n") + "\n" + header + "\n" + strings.Join(flavorLines, "\n")
			ReplyToSender(mctx, msg, fullText)
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
