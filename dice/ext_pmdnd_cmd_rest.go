package dice

import (
	"fmt"
)

var cmdRest = &CmdItemInfo{
	Name:      "rest",
	ShortHelp: ".rest {long|short}",
	Help: "PMDnD 休息:\n" +
		".rest long    长休（恢复全部HP和环位）\n" +
		".rest short   短休（恢复一半HP和环位）\n" +
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

		recoveryRate := int64(1)
		if isShort {
			recoveryRate = 2
		}

		playerName := getPlayerNameTempFunc(mctx)
		hpText := "没有设置hpmax，无法回复hp"
		hpMax, exists := VarGetValueInt64(mctx, "hpmax")
		var newHp int64
		if exists {
			curHp, _ := VarGetValueInt64(mctx, "hp")
			newHp = hpMax / recoveryRate
			if isShort && curHp > newHp && curHp <= hpMax {
				newHp = curHp
			}
			if !isShort {
				newHp = hpMax
			}
			VarSetValueInt64(mctx, "hp", newHp)
			hpText = fmt.Sprintf("%s %d/%d", "❤️", newHp, hpMax)
		}

		n := spellRingsRenew(mctx, msg)
		ringText := ""
		if n > 0 {
			if isShort {
				ringText = "\n⚡ 招式能量部分恢复！"
			} else {
				ringText = "\n⚡ 招式能量已全部恢复！"
			}
		}
		if ctx.Player.AutoSetNameTemplate != "" {
			_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
		}

		var fullText string
		if isShort {
			fullText = fmt.Sprintf("🌿 %s 进行了短暂休整！\n%s%s", playerName, hpText, ringText)
		} else {
			fullText = fmt.Sprintf("🏕️ %s 完成了充分的长休！\n%s%s\n✨ 精力充沛！", playerName, hpText, ringText)
		}
		ReplyToSender(mctx, msg, fullText)
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
