package dice

import (
	"fmt"
	"regexp"
	"strings"
)

// ---------- 死亡豁免辅助函数 ----------
func pmdndDeathSavingStable(ctx *MsgContext) {
	VarDelValue(ctx, "DSS")
	VarDelValue(ctx, "DSF")
	if ctx.Player.AutoSetNameTemplate != "" {
		_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
	}
}

func pmdndDeathSaving(ctx *MsgContext, successPlus int64, failurePlus int64) (int64, int64) {
	readAndAssign := func(name string) int64 {
		var val int64
		v, exists := _VarGetValueV1(ctx, name)
		if !exists {
			VarSetValueInt64(ctx, name, int64(0))
		} else {
			val, _ = v.ReadInt64()
		}
		return val
	}

	val1 := readAndAssign("DSS")
	val2 := readAndAssign("DSF")

	if successPlus != 0 {
		val1 += successPlus
		VarSetValueInt64(ctx, "DSS", val1)
	}
	if failurePlus != 0 {
		val2 += failurePlus
		VarSetValueInt64(ctx, "DSF", val2)
	}

	if ctx.Player.AutoSetNameTemplate != "" {
		_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
	}
	return val1, val2
}

// 检查是否达到3成功或3失败，并清除状态
func pmdndDeathSavingResultCheck(ctx *MsgContext, a int64, b int64) string {
	if a >= 3 {
		pmdndDeathSavingStable(ctx)
		return "✨ 伤势稳定，脱离危险！"
	}
	if b >= 3 {
		pmdndDeathSavingStable(ctx)
		return "💀 宝可梦失去了战斗能力……"
	}
	return ""
}

// ---------- 帮助文本 ----------
func getDsHelp() string {
	return "PMDnD 濒死豁免:\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		".ds                     进行濒死豁免检定\n" +
		".ds 成功±N              调整成功次数（例: .ds 成功+1）\n" +
		".ds 失败±N              调整失败次数（例: .ds 失败-1）\n" +
		".ds stat                查看当前豁免状态\n" +
		".ds help                显示本帮助\n" +
		"\n" +
		"📋 规则说明:\n" +
		"  3次成功 → 伤势稳定，脱离危险\n" +
		"  3次失败 → 角色永久失去战斗能力\n" +
		"  d20=1   → 计为2次失败\n" +
		"  d20=20  → 立即恢复1点HP，脱离濒死\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

// ---------- .ds 命令 ----------
var cmdDs = &CmdItemInfo{
	Name:          "ds",
	ShortHelp:     ".ds // 进行濒死豁免\n.ds stat // 查看状态\n.ds 成功±1 // 调整成功次数",
	Help:          getDsHelp(),
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		mctx := GetCtxProxyFirst(ctx, cmdArgs)
		tmpl := mctx.Group.GetCharTemplate(mctx.Dice)
		if tmpl != nil {
			mctx.SystemTemplate = tmpl
		}

		// 检查是否请求帮助
		if cmdArgs.IsArgEqual(1, "help") {
			ReplyToSender(mctx, msg, getDsHelp())
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		restText := cmdArgs.CleanArgs

		// 处理成功/失败调整：.ds 成功+1 或 .ds s+1
		re := regexp.MustCompile(`^(s|S|成功|f|F|失败)([+-＋－])`)
		m := re.FindStringSubmatch(restText)
		if len(m) > 0 {
			restText = strings.TrimSpace(restText[len(m[0]):])
			isNeg := m[2] == "-" || m[2] == "－"
			r := ctx.Eval(restText, nil)
			if r.vm.Error != nil {
				ReplyToSender(mctx, msg, "错误: 无法解析表达式: "+restText)
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			_v, _ := r.ReadInt()
			v := int64(_v)
			if isNeg {
				v = -v
			}

			var a, b int64
			label := ""
			switch m[1] {
			case "s", "S", "成功":
				a, b = pmdndDeathSaving(mctx, v, 0)
				label = "成功"
			case "f", "F", "失败":
				a, b = pmdndDeathSaving(mctx, 0, v)
				label = "失败"
			}
			text := fmt.Sprintf("📊 %s 的濒死豁免状态已更新：\n  %s: %d  失败: %d",
				getPlayerNameTempFunc(mctx), label, a, b)
			exText := pmdndDeathSavingResultCheck(mctx, a, b)
			if exText != "" {
				text += "\n" + exText
			}
			ReplyToSender(mctx, msg, text)
			return CmdExecuteResult{Matched: true, Solved: true}
		}

		val := cmdArgs.GetArgN(1)
		switch val {
		case "stat":
			a, b := pmdndDeathSaving(mctx, 0, 0)
			text := fmt.Sprintf("📊 %s 的濒死豁免状态：\n  成功: %d  失败: %d",
				getPlayerNameTempFunc(mctx), a, b)
			exText := pmdndDeathSavingResultCheck(mctx, a, b)
			if exText != "" {
				text += "\n" + exText
			}
			ReplyToSender(mctx, msg, text)

		case "":
			fallthrough
		default:
			// 检查是否设置了 HP
			hp, exists := VarGetValueInt64(mctx, "hp")
			if !exists {
				ReplyToSender(mctx, msg, fmt.Sprintf("❌ %s 未设置生命值，无法进行濒死豁免检定。", getPlayerNameTempFunc(mctx)))
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			if hp > 0 {
				ReplyToSender(mctx, msg, fmt.Sprintf("💚 %s 生命值大于0 (当前 %d)，无需进行濒死豁免。", getPlayerNameTempFunc(mctx), hp))
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			// 解析优势/劣势
			restText := cmdArgs.CleanArgs
			re := regexp.MustCompile(`^(优势|劣势|優勢|劣勢)`)
			m2 := re.FindString(restText)
			if m2 != "" {
				restText = strings.TrimSpace(restText[len(m2):])
			}
			expr := fmt.Sprintf("d20%s%s", m2, restText)

			mctx.CreateVmIfNotExists()
			mctx.setDndReadForVM(true)
			r := mctx.Eval(expr, nil)
			if r.vm.Error != nil {
				ReplyToSender(mctx, msg, "无法解析表达式: "+restText)
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			d20, ok := r.ReadInt()
			if !ok {
				ReplyToSender(mctx, msg, "并非数值类型: "+r.vm.Matched)
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			detail := r.vm.GetDetailText()
			exprToShow := fmt.Sprintf("[%s]", expr)
			if detail != r.ToString() {
				s := r.ToString()
				exprToShow, _ = strings.CutPrefix(detail, s)
			}

			playerName := getPlayerNameTempFunc(mctx)
			var text string

			// D20=20: 恢复1HP
			if d20 == 20 {
				pmdndDeathSavingStable(mctx)
				VarSetValueInt64(mctx, "hp", 1)
				text = fmt.Sprintf("💔 %s 失去战斗能力！\n正在进行 濒死判定...\n%s = 20 ✨ 大成功！\n奇迹发生！恢复 1 HP！\n%s 重新站了起来！",
					playerName, exprToShow, playerName)
				ReplyToSender(mctx, msg, text)
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			// D20=1: 计为2次失败
			if d20 == 1 {
				a, b := pmdndDeathSaving(mctx, 0, 2)
				text = fmt.Sprintf("💔 %s 失去战斗能力！\n正在进行 濒死判定...\n%s = 1 💀 大失败！\n（计为2次失败）\n当前状态: 成功 %d  失败 %d",
					playerName, exprToShow, a, b)
				exText := pmdndDeathSavingResultCheck(mctx, a, b)
				if exText != "" {
					text += "\n" + exText
				}
				ReplyToSender(mctx, msg, text)
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			// D20 >= 10: 成功
			if d20 >= 10 {
				a, b := pmdndDeathSaving(mctx, 1, 0)
				text = fmt.Sprintf("💔 %s 失去战斗能力！\n正在进行 濒死判定...\n%s = %d 🌟 成功！\n当前状态: 成功 %d  失败 %d",
					playerName, exprToShow, d20, a, b)
				exText := pmdndDeathSavingResultCheck(mctx, a, b)
				if exText != "" {
					text += "\n" + exText
				}
				ReplyToSender(mctx, msg, text)
				return CmdExecuteResult{Matched: true, Solved: true}
			}

			// D20 < 10: 失败
			a, b := pmdndDeathSaving(mctx, 0, 1)
			text = fmt.Sprintf("💔 %s 失去战斗能力！\n正在进行 濒死判定...\n%s = %d 💨 失败……\n当前状态: 成功 %d  失败 %d",
				playerName, exprToShow, d20, a, b)
			exText := pmdndDeathSavingResultCheck(mctx, a, b)
			if exText != "" {
				text += "\n" + exText
			}
			ReplyToSender(mctx, msg, text)
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
