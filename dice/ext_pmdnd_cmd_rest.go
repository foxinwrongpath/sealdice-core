package dice

import (
	"fmt"
)

var cmdRest = &CmdItemInfo{
	Name:      "rest",
	ShortHelp: ".rest {long|short}",
	Help: "PMDnD 休息:\n" +
		".rest long    长休（恢复全部HP和PP）\n" +
		".rest short   短休（恢复一半HP和PP）\n" +
		"快捷别名：.长休  .短休",
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
			var newHp int64
			if isShort {
				// 短休：固定恢复一半
				newHp = hpMax / 2
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
			var newPp int64
			if isShort {
				// 短休：固定恢复一半
				newPp = ppMax / 2
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
