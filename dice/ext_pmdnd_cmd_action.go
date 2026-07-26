package dice

import (
	"fmt"
	"strconv"
)

var cmdAction = &CmdItemInfo{
	Name:      "action",
	ShortHelp: ".action {use|bonus|reaction|rest|set} ...",
	Help:          getActionHelp(),
	AllowDelegate: true,
	Solve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) CmdExecuteResult {
		cmdArgs.ChopPrefixToArgsWith("use", "bonus", "reaction", "rest", "set", "show")
		sub := cmdArgs.GetArgN(1)
		mctx := GetCtxProxyFirst(ctx, cmdArgs)

		switch sub {
		case "show", "":
			action, _ := VarGetValueInt64(mctx, "行动力")
			if action == 0 {
				action = 1
			}
			bonus, _ := VarGetValueInt64(mctx, "附加行动")
			if bonus == 0 {
				bonus = 1
			}
			reac, _ := VarGetValueInt64(mctx, "反应")
			if reac == 0 {
				reac = 1
			}
			actionCur, _ := VarGetValueInt64(mctx, "$行动力_cur")
			bonusCur, _ := VarGetValueInt64(mctx, "$附加行动_cur")
			reacCur, _ := VarGetValueInt64(mctx, "$反应_cur")
			ReplyToSender(mctx, msg, fmt.Sprintf("%s的行动资源: 行动力 %d/%d  附加行动 %d/%d  反应 %d/%d",
				getPlayerNameTempFunc(mctx), actionCur, action, bonusCur, bonus, reacCur, reac))

		case "use":
			actionCur, _ := VarGetValueInt64(mctx, "$行动力_cur")
			if actionCur <= 0 {
				if action, _ := VarGetValueInt64(mctx, "行动力"); action != 0 {
					actionCur = action
				} else {
					actionCur = 1
				}
			}
			if actionCur <= 0 {
				ReplyToSender(mctx, msg, fmt.Sprintf("%s没有可用的行动力", getPlayerNameTempFunc(mctx)))
			} else {
				VarSetValueInt64(mctx, "$行动力_cur", actionCur-1)
				ReplyToSender(mctx, msg, fmt.Sprintf("%s使用了行动力(%d→%d)", getPlayerNameTempFunc(mctx), actionCur, actionCur-1))
			}

		case "bonus":
			bonusCur, _ := VarGetValueInt64(mctx, "$附加行动_cur")
			if bonusCur <= 0 {
				if bonus, _ := VarGetValueInt64(mctx, "附加行动"); bonus != 0 {
					bonusCur = bonus
				} else {
					bonusCur = 1
				}
			}
			if bonusCur <= 0 {
				ReplyToSender(mctx, msg, fmt.Sprintf("%s没有可用的附加行动", getPlayerNameTempFunc(mctx)))
			} else {
				VarSetValueInt64(mctx, "$附加行动_cur", bonusCur-1)
				ReplyToSender(mctx, msg, fmt.Sprintf("%s使用了附加行动(%d→%d)", getPlayerNameTempFunc(mctx), bonusCur, bonusCur-1))
			}

		case "reaction":
			reacCur, _ := VarGetValueInt64(mctx, "$反应_cur")
			if reacCur <= 0 {
				if reac, _ := VarGetValueInt64(mctx, "反应"); reac != 0 {
					reacCur = reac
				} else {
					reacCur = 1
				}
			}
			if reacCur <= 0 {
				ReplyToSender(mctx, msg, fmt.Sprintf("%s没有可用的反应", getPlayerNameTempFunc(mctx)))
			} else {
				VarSetValueInt64(mctx, "$反应_cur", reacCur-1)
				ReplyToSender(mctx, msg, fmt.Sprintf("%s使用了反应(%d→%d)", getPlayerNameTempFunc(mctx), reacCur, reacCur-1))
			}

		case "rest":
			action, _ := VarGetValueInt64(mctx, "行动力")
			if action == 0 {
				action = 1
			}
			bonus, _ := VarGetValueInt64(mctx, "附加行动")
			if bonus == 0 {
				bonus = 1
			}
			reac, _ := VarGetValueInt64(mctx, "反应")
			if reac == 0 {
				reac = 1
			}
			VarSetValueInt64(mctx, "$行动力_cur", action)
			VarSetValueInt64(mctx, "$附加行动_cur", bonus)
			VarSetValueInt64(mctx, "$反应_cur", reac)
			ReplyToSender(mctx, msg, fmt.Sprintf("%s恢复了全部行动资源", getPlayerNameTempFunc(mctx)))

		case "set":
			target := cmdArgs.GetArgN(2)
			valStr := cmdArgs.GetArgN(3)
			if target == "" || valStr == "" {
				return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
			}
			n, err := strconv.ParseInt(valStr, 10, 64)
			if err != nil {
				ReplyToSender(mctx, msg, "数值必须是数字")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			switch target {
			case "行动力", "action":
				VarSetValueInt64(mctx, "行动力", n)
			case "附加行动", "bonusAction", "ba":
				VarSetValueInt64(mctx, "附加行动", n)
			case "反应", "reaction", "re":
				VarSetValueInt64(mctx, "反应", n)
			default:
				ReplyToSender(mctx, msg, "请指定: 行动力/附加行动/反应")
				return CmdExecuteResult{Matched: true, Solved: true}
			}
			ReplyToSender(mctx, msg, fmt.Sprintf("已设置%s为%d", target, n))

		default:
			return CmdExecuteResult{Matched: true, Solved: true, ShowHelp: true}
		}
		return CmdExecuteResult{Matched: true, Solved: true}
	},
}
