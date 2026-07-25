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
		restType := "长休"
		if isShort {
			recoveryRate = 2
			restType = "短休"
		}

		hpText := "没有设置hpmax，无法回复hp"
		hpMax, exists := VarGetValueInt64(mctx, "hpmax")
		if exists {
			curHp, _ := VarGetValueInt64(mctx, "hp")
			recoveredHP := hpMax / recoveryRate
			if isShort && curHp > recoveredHP && curHp <= hpMax {
				recoveredHP = curHp
			}
			if recoveryRate == 1 {
				recoveredHP = hpMax
			}
			VarSetValueInt64(mctx, "hp", recoveredHP)
			hpText = fmt.Sprintf("hp恢复至%d", recoveredHP)
		}

		n := spellRingsRenew(mctx, msg)
		ringText := ""
		if n > 0 {
			ringText = "，环位得到了恢复"
		}
		if ctx.Player.AutoSetNameTemplate != "" {
			_, _ = SetPlayerGroupCardByTemplate(ctx, ctx.Player.AutoSetNameTemplate)
		}
		ReplyToSender(mctx, msg, fmt.Sprintf("%s的%s: %s%s", getPlayerNameTempFunc(mctx), restType, hpText, ringText))
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
