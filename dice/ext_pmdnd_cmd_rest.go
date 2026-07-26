package dice

import (
	"fmt"
)

var cmdRest = &CmdItemInfo{
	Name:      "rest",
	ShortHelp: ".rest {long|short}",
	Help:          getRestHelp(),
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		sub := cmdArgs.GetArgN(1)
		mctx := GetCtxProxyFirst(ctx, cmdArgs)
		tmpl := mctx.Group.GetCharTemplate(mctx.Dice)
		if tmpl != nil {
			mctx.SystemTemplate = tmpl
		}

		isShort := false
		if sub == "short" || sub == "短休" {
			isShort = true
		} else if sub == "long" || sub == "长休" || sub == "" {
			isShort = false
		} else {
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		}

		playerName := getPlayerNameTempFunc(mctx)
		var hpText string
		var ppText string

		// ---- HP 恢复 ----
		hpMax, hpExists := VarGetValueInt64(mctx, "hpmax")
		if hpExists {
			curHp, _ := VarGetValueInt64(mctx, "hp")
			var newHp int64
			if isShort {
				// 短休：当前值 + 最大值的一半
				recoverAmount := hpMax / 2
				newHp = curHp + recoverAmount
				if newHp > hpMax {
					newHp = hpMax
				}
			} else {
				// 长休：恢复全部
				newHp = hpMax
			}
			if newHp < 1 {
				newHp = 1
			}
			VarSetValueInt64(mctx, "hp", newHp)
			hpText = fmt.Sprintf("❤️ HP: %d/%d", newHp, hpMax)
		} else {
			hpText = "❤️ 未设置 hpmax"
		}

		// ---- PP 恢复（法力值系统） ----
		ppMax, ppExists := VarGetValueInt64(mctx, "ppmax")
		if ppExists {
			curPp, _ := VarGetValueInt64(mctx, "pp")
			var newPp int64
			if isShort {
				// 短休：当前值 + 最大值的一半
				recoverAmount := ppMax / 2
				newPp = curPp + recoverAmount
				if newPp > ppMax {
					newPp = ppMax
				}
			} else {
				// 长休：恢复全部
				newPp = ppMax
			}
			if newPp < 1 {
				newPp = 1
			}
			VarSetValueInt64(mctx, "pp", newPp)
			ppText = fmt.Sprintf("💎 PP: %d/%d", newPp, ppMax)
		} else {
			ppText = "💎 未设置 ppmax（可使用 .st ppmax:XXX 设置）"
		}

		if ctx.Player.AutoSetNameTemplate != "" {
			_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
		}

		// ---- 输出 ----
		var fullText string
		if isShort {
			fullText = fmt.Sprintf("🌿 %s 进行了短暂休整！\n%s\n%s", playerName, hpText, ppText)
		} else {
			fullText = fmt.Sprintf("🏕️ %s 完成了充分的长休！\n%s\n%s\n✨ 精力充沛！", playerName, hpText, ppText)
		}
		ReplyToSender(mctx, msg, fullText)
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
